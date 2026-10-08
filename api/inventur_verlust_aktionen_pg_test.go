package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Die zwei Knöpfe des Fehlbestandsberichts über ihre Routen: „gefunden" und „endgültig
// löschen". Die Regeln prüft repository/inventur_verlust_*_pg_test.go an der Funktion;
// hier steht, was erst die Route dazutut: Die Transaktion wird festgeschrieben, eine
// gebundene Lage kommt als 409 mit Klartext an, und die Antwort nennt, was geschah.

// alsBenutzer hängt die Sitzung des Benutzers an die Anfrage; ohne Benutzer bleibt sie
// unangemeldet.
func alsBenutzer(req *http.Request, benutzerID string) *http.Request {
	if benutzerID == "" {
		return req
	}
	return req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: benutzerID, Rolle: auth.RoleAdmin}))
}

// inventurReset leert Bestand und Inventuren. Eine offene Inventur aus einem früheren Test
// sperrt sonst die neue: Je Bereich darf nur ein Durchgang offen sein.
func inventurReset(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `TRUNCATE inventur_sessions RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("Inventuren leeren: %v", err)
	}
	resetBestandsdaten(t, pool)
}

// verlustLage legt die Exemplare an und bucht sie über einen Inventurabschluss als
// Verlust: der Zustand, den der Fehlbestandsbericht vorfindet. Liefert je Barcode die
// Kennung des Exemplars.
func verlustLage(t *testing.T, pool *pgxpool.Pool, barcodes ...string) map[string]string {
	t.Helper()
	ctx := context.Background()
	titelID := titelMitMeldebestand(t, pool, "Verlust-Titel", 0)
	lage := make(map[string]string, len(barcodes))
	for _, b := range barcodes {
		lage[b] = exemplar(t, pool, titelID, b, true, "")
	}
	var sessionID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO inventur_sessions (scope_type) VALUES ('global') RETURNING id`).Scan(&sessionID); err != nil {
		t.Fatalf("Inventur anlegen: %v", err)
	}
	gebucht, err := repository.NewInventoryRepository(pool).FinishInventurSession(ctx, sessionID, repository.InventurScope{}, adminFuerAudit(t, pool))
	if err != nil {
		t.Fatalf("Inventur abschließen: %v", err)
	}
	if gebucht != len(barcodes) {
		t.Fatalf("als Verlust gebucht: %d, erwartet %d", gebucht, len(barcodes))
	}
	return lage
}

// verlustExemplar liest, ob das Exemplar noch in der Datenbank steht und ob es als
// Verlust gilt. Gelesen wird über den Pool, also nur, was festgeschrieben ist.
func verlustExemplar(t *testing.T, pool *pgxpool.Pool, id string) (da, verlust bool) {
	t.Helper()
	err := pool.QueryRow(context.Background(), `
		SELECT true, ist_ausgesondert AND coalesce(aussonderung_grund, '') = 'VERLUST'
		FROM buecher_exemplare WHERE id = $1`, id).Scan(&da, &verlust)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false
	}
	if err != nil {
		t.Fatalf("Exemplar lesen: %v", err)
	}
	return da, verlust
}

// verlustForderung legt eine offene Forderung am Exemplar an; bescheidID bindet sie an
// einen Bescheid.
func verlustForderung(t *testing.T, pool *pgxpool.Pool, schuelerID, exemplarID, art string, betrag float64, bescheidID *string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, art, bescheid_id)
		VALUES ($1, $2, 'Forderung im Test', $3, $4, $5) RETURNING id`,
		exemplarID, schuelerID, betrag, art, bescheidID).Scan(&id); err != nil {
		t.Fatalf("Forderung anlegen: %v", err)
	}
	return id
}

func verlustRumpf(t *testing.T, exemplarIDs ...string) string {
	t.Helper()
	rumpf, err := json.Marshal(map[string][]string{"exemplar_ids": exemplarIDs})
	if err != nil {
		t.Fatal(err)
	}
	return string(rumpf)
}

// verlusteLoeschen ruft POST /api/buecher/exemplare/verlust-endgueltig-loeschen.
func verlusteLoeschen(t *testing.T, srv *Server, benutzerID, rumpf string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/buecher/exemplare/verlust-endgueltig-loeschen", strings.NewReader(rumpf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.InventurVerlusteLoeschenHandler()(rec, alsBenutzer(req, benutzerID))
	return rec
}

// verlustGefunden ruft POST /api/buecher/exemplare/{id}/gefunden.
func verlustGefunden(t *testing.T, srv *Server, benutzerID, exemplarID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/buecher/exemplare/"+exemplarID+"/gefunden", nil)
	req.SetPathValue("id", exemplarID)
	rec := httptest.NewRecorder()
	srv.InventurVerlustGefundenHandler()(rec, alsBenutzer(req, benutzerID))
	return rec
}

// Gelöscht wird, was als Verlust gebucht ist, und die Antwort nennt genau diese
// Exemplare. Die Oberfläche nimmt nur die genannten Zeilen aus ihrer Liste.
func TestVerlusteLoeschen_LoeschtNurVerlusteUndNenntSie(t *testing.T) {
	pool := pgTestPool(t)
	inventurReset(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	admin := adminFuerAudit(t, pool)

	lage := verlustLage(t, pool, "VLH-1", "VLH-2")
	imRegal := exemplar(t, pool, titelMitMeldebestand(t, pool, "Im Regal", 0), "VLH-REGAL", true, "")

	rec := verlusteLoeschen(t, srv, admin, verlustRumpf(t, lage["VLH-1"], imRegal, lage["VLH-2"]))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		Geloescht     int      `json:"geloescht"`
		GeloeschteIDs []string `json:"geloeschte_ids"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort nicht lesbar: %v (%s)", err, rec.Body.String())
	}
	erwartet := []string{lage["VLH-1"], lage["VLH-2"]}
	slices.Sort(erwartet)
	slices.Sort(antwort.GeloeschteIDs)
	if antwort.Geloescht != 2 || !slices.Equal(antwort.GeloeschteIDs, erwartet) {
		t.Errorf("Antwort nennt %d gelöschte (%v), erwartet die zwei Verluste %v", antwort.Geloescht, antwort.GeloeschteIDs, erwartet)
	}

	for _, barcode := range []string{"VLH-1", "VLH-2"} {
		if da, _ := verlustExemplar(t, pool, lage[barcode]); da {
			t.Errorf("%s ist als gelöscht gemeldet und steht noch in der Datenbank", barcode)
		}
		if n := zaehleZeilen(t, pool, `
			SELECT count(*) FROM audit_log
			WHERE tabelle = 'buecher_exemplare' AND aktion = 'DELETE' AND datensatz_id = $1
			  AND bearbeiter_id = $2 AND details->>'action' = $3`,
			lage[barcode], admin, repository.AuditAktionVerlustEndgueltigGeloescht); n != 1 {
			t.Errorf("%s: %d Protokolleinträge mit dem angemeldeten Benutzer, erwartet 1", barcode, n)
		}
	}
	if da, _ := verlustExemplar(t, pool, imRegal); !da {
		t.Error("ein Exemplar, das nicht als Verlust gebucht ist, wurde gelöscht")
	}
}

// Ist keines der Exemplare als Verlust gebucht, bleibt alles stehen. Die Liste der
// gelöschten ist dann leer und nicht null, weil die Oberfläche ihre Zeilen darüber filtert.
func TestVerlusteLoeschen_OhneVerlustBleibtAllesStehen(t *testing.T) {
	pool := pgTestPool(t)
	inventurReset(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	admin := adminFuerAudit(t, pool)
	imRegal := exemplar(t, pool, titelMitMeldebestand(t, pool, "Im Regal", 0), "VLH-REGAL-2", true, "")

	rec := verlusteLoeschen(t, srv, admin, verlustRumpf(t, imRegal))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	for _, teil := range []string{`"geloescht":0`, `"geloeschte_ids":[]`} {
		if !strings.Contains(rec.Body.String(), teil) {
			t.Errorf("Antwort %s enthält %s nicht", strings.TrimSpace(rec.Body.String()), teil)
		}
	}
	if da, _ := verlustExemplar(t, pool, imRegal); !da {
		t.Error("das Exemplar im Regal wurde gelöscht")
	}
}

// Hängt an einem Exemplar eine offene Forderung, wird der ganze Stapel abgewiesen. Die
// Antwort ist eine 409 mit dem Barcode: Bei einer 500 ersetzt der Server den Text durch
// eine neutrale Meldung, und niemand sieht, welches Exemplar im Weg steht.
func TestVerlusteLoeschen_OffeneForderungIst409MitBarcode(t *testing.T) {
	pool := pgTestPool(t)
	inventurReset(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	admin := adminFuerAudit(t, pool)

	lage := verlustLage(t, pool, "VLH-FREI", "VLH-GEB")
	schuelerID := seedSchueler(t, pool, "VLH-S1", "Forderungskind", "07A")
	forderung := verlustForderung(t, pool, schuelerID, lage["VLH-GEB"], "beschaedigt", 12.50, nil)

	rec := verlusteLoeschen(t, srv, admin, verlustRumpf(t, lage["VLH-FREI"], lage["VLH-GEB"]))
	if rec.Code != http.StatusConflict {
		t.Fatalf("Status %d, erwartet 409: %s", rec.Code, rec.Body.String())
	}
	meldung := fehlermeldung(t, rec)
	if !strings.Contains(meldung, "VLH-GEB") || strings.Contains(meldung, "VLH-FREI") {
		t.Errorf("Meldung %q nennt nicht genau das gebundene Exemplar VLH-GEB", meldung)
	}

	for _, barcode := range []string{"VLH-FREI", "VLH-GEB"} {
		if da, verlust := verlustExemplar(t, pool, lage[barcode]); !da || !verlust {
			t.Errorf("%s nach der Abweisung: vorhanden=%v, Verlust=%v, erwartet unverändert", barcode, da, verlust)
		}
	}
	if n := zaehleZeilen(t, pool, `SELECT count(*) FROM schadensfaelle WHERE id = $1 AND ist_bezahlt = false`, forderung); n != 1 {
		t.Error("die offene Forderung ist nach der Abweisung nicht mehr offen")
	}
}

// Eine Anfrage ohne Exemplare, ohne lesbaren Rumpf oder ohne Anmeldung löscht nichts.
func TestVerlusteLoeschen_Abweisungen(t *testing.T) {
	pool := pgTestPool(t)
	inventurReset(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	admin := adminFuerAudit(t, pool)
	lage := verlustLage(t, pool, "VLH-BLEIBT")

	faelle := []struct {
		name, benutzer, rumpf string
		status                int
	}{
		{"leere Liste", admin, `{"exemplar_ids":[]}`, http.StatusBadRequest},
		{"Liste fehlt", admin, `{}`, http.StatusBadRequest},
		{"kein JSON", admin, `exemplar_ids`, http.StatusBadRequest},
		{"ohne Anmeldung", "", verlustRumpf(t, lage["VLH-BLEIBT"]), http.StatusUnauthorized},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			rec := verlusteLoeschen(t, srv, f.benutzer, f.rumpf)
			if rec.Code != f.status {
				t.Errorf("Status %d, erwartet %d: %s", rec.Code, f.status, rec.Body.String())
			}
		})
	}
	if da, verlust := verlustExemplar(t, pool, lage["VLH-BLEIBT"]); !da || !verlust {
		t.Errorf("nach den Abweisungen: vorhanden=%v, Verlust=%v, erwartet unverändert", da, verlust)
	}
}

// Der Fund bringt das Exemplar zurück in den Umlauf, steht im Bericht und beendet die
// Forderung, die das Buch abgerechnet hat. Die Antwort nennt Zahl und Betrag; daraus baut
// die Oberfläche ihre Meldung.
func TestVerlustGefunden_ZurueckInUmlaufUndForderungStorniert(t *testing.T) {
	pool := pgTestPool(t)
	inventurReset(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	admin := adminFuerAudit(t, pool)

	exemplarID := verlustLage(t, pool, "VGH-1")["VGH-1"]
	schuelerID := seedSchueler(t, pool, "VGH-S1", "Fundkind", "07A")
	forderung := verlustForderung(t, pool, schuelerID, exemplarID, "nicht_zurueckgegeben", 24.90, nil)

	rec := verlustGefunden(t, srv, admin, exemplarID)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	var antwort VerlustGefundenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort nicht lesbar: %v (%s)", err, rec.Body.String())
	}
	if antwort.Status != "ok" || antwort.StornierteForderungen != 1 || antwort.StornierterBetrag != 24.90 || antwort.Hinweis != "" {
		t.Errorf("Antwort %+v, erwartet eine stornierte Forderung über 24,90 ohne Hinweis", antwort)
	}

	var ausleihbar bool
	if err := pool.QueryRow(context.Background(),
		`SELECT ist_ausleihbar FROM buecher_exemplare WHERE id = $1`, exemplarID).Scan(&ausleihbar); err != nil {
		t.Fatalf("Exemplar lesen: %v", err)
	}
	if da, verlust := verlustExemplar(t, pool, exemplarID); !da || verlust || !ausleihbar {
		t.Errorf("Exemplar nach dem Fund: vorhanden=%v, Verlust=%v, ausleihbar=%v", da, verlust, ausleihbar)
	}
	if n := zaehleZeilen(t, pool,
		`SELECT count(*) FROM inventur_verluste WHERE exemplar_id = $1 AND gefunden_am IS NOT NULL`, exemplarID); n != 1 {
		t.Errorf("der Bericht vermerkt den Fund nicht (%d Zeilen mit gefunden_am)", n)
	}
	if n := zaehleZeilen(t, pool,
		`SELECT count(*) FROM schadensfaelle WHERE id = $1 AND storniert_am IS NOT NULL AND storniert_von = $2`, forderung, admin); n != 1 {
		t.Error("die Forderung ist nicht vom angemeldeten Benutzer storniert")
	}
	if n := zaehleZeilen(t, pool, `
		SELECT count(*) FROM audit_log
		WHERE tabelle = 'buecher_exemplare' AND aktion = 'UPDATE' AND datensatz_id = $1
		  AND bearbeiter_id = $2 AND details->>'action' = 'verlust_gefunden'`, exemplarID, admin); n != 1 {
		t.Errorf("%d Protokolleinträge zum Fund mit dem angemeldeten Benutzer, erwartet 1", n)
	}
}

// Liegt der Bescheid zur Forderung schon bei der Schulaufsicht, storniert die Schule
// nichts. Die Antwort trägt dann den Hinweis mit der Nummer des Bescheids.
func TestVerlustGefunden_UebergebenerBescheidKommtAlsHinweis(t *testing.T) {
	pool := pgTestPool(t)
	inventurReset(t, pool)
	bescheidReset(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	admin := adminFuerAudit(t, pool)
	ctx := context.Background()

	exemplarID := verlustLage(t, pool, "VGH-B")["VGH-B"]
	schuelerID := seedSchueler(t, pool, "VGH-S2", "Bescheidkind", "08A")
	var bescheidID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schadensersatz_bescheide (schueler_id, mittel, kassenjahr, laufende_nr, referenznummer, frist_bis, status, uebergeben_am)
		VALUES ($1, 'land', 2026, 1, 'VGH-REF-1', CURRENT_DATE + 28, 'uebergeben', NOW()) RETURNING id`,
		schuelerID).Scan(&bescheidID); err != nil {
		t.Fatalf("Bescheid anlegen: %v", err)
	}
	forderung := verlustForderung(t, pool, schuelerID, exemplarID, "nicht_zurueckgegeben", 31.00, &bescheidID)

	rec := verlustGefunden(t, srv, admin, exemplarID)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	var antwort VerlustGefundenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort nicht lesbar: %v (%s)", err, rec.Body.String())
	}
	if antwort.StornierteForderungen != 0 || antwort.StornierterBetrag != 0 || !strings.Contains(antwort.Hinweis, "VGH-REF-1") {
		t.Errorf("Antwort %+v, erwartet keine Stornierung und den Hinweis mit VGH-REF-1", antwort)
	}
	if n := zaehleZeilen(t, pool,
		`SELECT count(*) FROM schadensfaelle WHERE id = $1 AND ist_bezahlt = false AND storniert_am IS NULL`, forderung); n != 1 {
		t.Error("die Forderung auf dem übergebenen Bescheid ist nicht mehr offen")
	}
	if n := zaehleZeilen(t, pool,
		`SELECT count(*) FROM schadensersatz_bescheide WHERE id = $1 AND rueckgabe_nach_uebergabe`, bescheidID); n != 1 {
		t.Error("am Bescheid fehlt der Vermerk zur Rückgabe nach der Übergabe")
	}
	if da, verlust := verlustExemplar(t, pool, exemplarID); !da || verlust {
		t.Errorf("Exemplar nach dem Fund: vorhanden=%v, Verlust=%v", da, verlust)
	}
}

// Gibt es unter der Kennung keinen offenen Verlust, antwortet die Route mit 404 und
// bucht nichts: bei einer unbekannten Kennung, bei einem Exemplar im Regal und beim
// zweiten Klick auf denselben Fund.
func TestVerlustGefunden_OhneOffenenVerlustIst404(t *testing.T) {
	pool := pgTestPool(t)
	inventurReset(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	admin := adminFuerAudit(t, pool)

	exemplarID := verlustLage(t, pool, "VGH-2")["VGH-2"]
	imRegal := exemplar(t, pool, titelMitMeldebestand(t, pool, "Im Regal", 0), "VGH-REGAL", true, "")

	if rec := verlustGefunden(t, srv, admin, "00000000-0000-0000-0000-000000000000"); rec.Code != http.StatusNotFound {
		t.Errorf("unbekannte Kennung: Status %d, erwartet 404: %s", rec.Code, rec.Body.String())
	}
	if rec := verlustGefunden(t, srv, admin, imRegal); rec.Code != http.StatusNotFound {
		t.Errorf("Exemplar im Regal: Status %d, erwartet 404: %s", rec.Code, rec.Body.String())
	}
	if n := zaehleZeilen(t, pool, `SELECT count(*) FROM audit_log WHERE datensatz_id = $1`, imRegal); n != 0 {
		t.Errorf("zum Exemplar im Regal stehen %d Protokolleinträge, erwartet keiner", n)
	}

	if rec := verlustGefunden(t, srv, admin, exemplarID); rec.Code != http.StatusOK {
		t.Fatalf("erster Fund: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	if rec := verlustGefunden(t, srv, admin, exemplarID); rec.Code != http.StatusNotFound {
		t.Errorf("zweiter Fund: Status %d, erwartet 404: %s", rec.Code, rec.Body.String())
	}
	if n := zaehleZeilen(t, pool,
		`SELECT count(*) FROM audit_log WHERE datensatz_id = $1 AND details->>'action' = 'verlust_gefunden'`, exemplarID); n != 1 {
		t.Errorf("%d Protokolleinträge zum Fund, erwartet genau einer", n)
	}
}

// Ohne Anmeldung und ohne Kennung bleibt der Verlust gebucht.
func TestVerlustGefunden_Abweisungen(t *testing.T) {
	pool := pgTestPool(t)
	inventurReset(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	admin := adminFuerAudit(t, pool)
	exemplarID := verlustLage(t, pool, "VGH-3")["VGH-3"]

	if rec := verlustGefunden(t, srv, "", exemplarID); rec.Code != http.StatusUnauthorized {
		t.Errorf("ohne Anmeldung: Status %d, erwartet 401: %s", rec.Code, rec.Body.String())
	}
	if rec := verlustGefunden(t, srv, admin, ""); rec.Code != http.StatusBadRequest {
		t.Errorf("ohne Kennung: Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
	}
	if da, verlust := verlustExemplar(t, pool, exemplarID); !da || !verlust {
		t.Errorf("nach den Abweisungen: vorhanden=%v, Verlust=%v, erwartet unverändert", da, verlust)
	}
}
