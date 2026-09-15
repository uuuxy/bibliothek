package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Stufe 2 des Mahnverfahrens: Der Bescheid entsteht direkt aus den überfälligen Büchern.
// Die Zusicherung ist „Papier == Datenbank": Mit dem Brief endet die Ausleihe, das
// Exemplar gilt als Verlust, die Forderung trägt Nummer und Betrag — oder nichts davon.

// bescheidAusleihe legt eine Ausleihe an, deren Frist vor `fristTage` Tagen war
// (negativ = noch nicht fällig).
func bescheidAusleihe(t *testing.T, pool *pgxpool.Pool, exemplarID, schuelerID string, ueberfaelligSeitTagen int) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
		VALUES ($1, $2, CURRENT_DATE - ($3::int + 30), CURRENT_DATE - $3::int) RETURNING id`,
		exemplarID, schuelerID, ueberfaelligSeitTagen).Scan(&id); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
	return id
}

// bescheidRumpfMitAusleihen baut den Anfragerumpf für Bücher ohne Forderung.
func bescheidRumpfMitAusleihen(frist string, betraege map[string]float64) string {
	teile := make([]string, 0, len(betraege))
	for id, betrag := range betraege {
		teile = append(teile, fmt.Sprintf(`{"ausleihe_id":%q,"betrag":%.2f}`, id, betrag))
	}
	return fmt.Sprintf(`{"mittel":"land","frist_bis":%q,"positionen":[],"ausleihen":[%s]}`, frist, strings.Join(teile, ","))
}

// bescheidVorschlagUeberHandler ruft GET /api/schueler/{id}/bescheid-vorschlag.
func bescheidVorschlagUeberHandler(t *testing.T, srv *Server, pool *pgxpool.Pool, schuelerID string) BescheidVorschlag {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/schueler/"+schuelerID+"/bescheid-vorschlag", nil)
	req.SetPathValue("id", schuelerID)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: adminFuerAudit(t, pool), Rolle: auth.RoleAdmin}))
	rec := httptest.NewRecorder()
	srv.BescheidVorschlagHandler(repository.NewBescheidRepository(pool))(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Vorschlag: Status %d: %s", rec.Code, rec.Body.String())
	}
	var v BescheidVorschlag
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

// nummernkreisStand liest die letzte vergebene Nummer (0 = noch keine).
func nummernkreisStand(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT coalesce(max(letzte_nr), 0) FROM schadensersatz_nummern`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Der Vorschlag nennt die überfälligen Bücher ohne Forderung — und nur die: Ein noch
// nicht fälliges Buch gehört nicht auf den Brief, ein Buch mit Forderung steht schon bei
// den Positionen (sonst stünde es zweimal im Dialog).
func TestBescheidVorschlag_NenntUeberfaelligeBuecherOhneForderung(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	bescheidReset(t, pool)
	bescheidAngabenSetzen(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	sid := seedSchueler(t, pool, "S-VORSCHLAG-1", "Vorschlagkind", "08G2")
	titelID := bescheidLernmittel(t, pool, "Physik 8")
	ueberfaellig := exemplar(t, pool, titelID, "VOR-EX-1", true, "")
	nochNichtFaellig := exemplar(t, pool, titelID, "VOR-EX-2", true, "")
	mitForderung := exemplar(t, pool, titelID, "VOR-EX-3", true, "")
	aUeberfaellig := bescheidAusleihe(t, pool, ueberfaellig, sid, 30)
	bescheidAusleihe(t, pool, nochNichtFaellig, sid, -10)
	aMitForderung := bescheidAusleihe(t, pool, mitForderung, sid, 40)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO schadensfaelle (exemplar_id, schueler_id, ausleihe_id, beschreibung, betrag, art)
		VALUES ($1, $2, $3, 'schon gemeldet', 0, 'nicht_zurueckgegeben')`, mitForderung, sid, aMitForderung); err != nil {
		t.Fatal(err)
	}

	v := bescheidVorschlagUeberHandler(t, srv, pool, sid)
	if len(v.Ausleihen) != 1 || v.Ausleihen[0].AusleiheID != aUeberfaellig {
		t.Fatalf("Ausleihen = %+v, erwartet genau die überfällige ohne Forderung", v.Ausleihen)
	}
	if !v.Ausleihen[0].IstLernmittel || v.Ausleihen[0].Titel != "Physik 8" || v.Ausleihen[0].FaelligSeit == "" {
		t.Errorf("Ausleihe unvollständig: %+v", v.Ausleihen[0])
	}
	if len(v.Positionen) != 1 {
		t.Errorf("Positionen = %+v, erwartet die eine bestehende Forderung", v.Positionen)
	}
}

// Der Kern von Stufe 2: Ein Brief aus zwei überfälligen Büchern, in einer Transaktion.
func TestBescheidErstellen_AusUeberfaelligenBuechern(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	bescheidReset(t, pool)
	ctx := context.Background()
	bescheidAngabenSetzen(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	sid := seedSchueler(t, pool, "S-VERLUST-1", "Verlustkind", "08G2")
	titelID := bescheidLernmittel(t, pool, "Chemie 8")
	ex1 := exemplar(t, pool, titelID, "VERL-EX-1", true, "")
	ex2 := exemplar(t, pool, titelID, "VERL-EX-2", true, "")
	a1 := bescheidAusleihe(t, pool, ex1, sid, 30)
	a2 := bescheidAusleihe(t, pool, ex2, sid, 45)

	rec := bescheidErstellenUeberHandler(t, srv, pool, sid,
		bescheidRumpfMitAusleihen(in28Tagen(), map[string]float64{a1: 24.90, a2: 12.00}))
	if rec.Code != http.StatusCreated {
		t.Fatalf("Status %d, want 201: %s", rec.Code, rec.Body.String())
	}
	var b repository.Bescheid
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if b.Gesamtbetrag != 36.90 || b.AnzahlPositionen != 2 {
		t.Errorf("Brief: Summe %.2f, Positionen %d — want 36.90 und 2", b.Gesamtbetrag, b.AnzahlPositionen)
	}

	for ausleihe, want := range map[string]float64{a1: 24.90, a2: 12.00} {
		// 1. Die Ausleihe ist beendet.
		var beendet bool
		if err := pool.QueryRow(ctx, `SELECT rueckgabe_am IS NOT NULL FROM ausleihen WHERE id = $1`, ausleihe).Scan(&beendet); err != nil {
			t.Fatal(err)
		}
		if !beendet {
			t.Errorf("Ausleihe %s läuft nach dem Brief weiter", ausleihe)
		}
		// 2. Die Forderung hängt am Brief, trägt den Betrag, die Fallgruppe und die Nummer.
		var art, beschreibung, bescheidID string
		var betrag float64
		if err := pool.QueryRow(ctx, `
			SELECT art, beschreibung, coalesce(bescheid_id::text, ''), betrag::float8
			FROM schadensfaelle WHERE ausleihe_id = $1`, ausleihe).Scan(&art, &beschreibung, &bescheidID, &betrag); err != nil {
			t.Fatalf("Forderung zu Ausleihe %s: %v", ausleihe, err)
		}
		if art != "nicht_zurueckgegeben" || bescheidID != b.ID || betrag != want {
			t.Errorf("Forderung zu %s: art=%s bescheid=%s betrag=%.2f", ausleihe, art, bescheidID, betrag)
		}
		if !strings.Contains(beschreibung, b.Referenznummer) {
			t.Errorf("Forderung ohne Referenznummer: %q", beschreibung)
		}
	}
	// 3. Die Exemplare sind als VERLUST ausgesondert.
	var verluste int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM buecher_exemplare
		WHERE id IN ($1, $2) AND ist_ausgesondert AND aussonderung_grund = 'VERLUST'`, ex1, ex2).Scan(&verluste); err != nil {
		t.Fatal(err)
	}
	if verluste != 2 {
		t.Errorf("%d von 2 Exemplaren als VERLUST ausgesondert", verluste)
	}
	// 4. Der Vorschlag nennt die Bücher danach nicht mehr — sie stehen auf dem Brief.
	if v := bescheidVorschlagUeberHandler(t, srv, pool, sid); len(v.Ausleihen) != 0 || len(v.Positionen) != 0 {
		t.Errorf("Vorschlag nach dem Brief: Ausleihen %d, Positionen %d — want 0/0", len(v.Ausleihen), len(v.Positionen))
	}
}

// Die Lage ändert sich, während der Dialog offen steht. Drei Fälle, jeder ein 409 ohne
// Brief, ohne verbrauchte Nummer, ohne beendete Ausleihe: das Buch kam zurück, jemand
// hat den Verlust in der Akte gemeldet, das Buch gehört einem anderen Kind. Rot am alten
// Code gesehen (Prüfung ausgehängt): Brief über ein Buch, das im Regal steht.
func TestBescheidErstellen_VeralteterDialogWirdAbgewiesen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	bescheidReset(t, pool)
	ctx := context.Background()
	bescheidAngabenSetzen(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	sid := seedSchueler(t, pool, "S-VERALTET-1", "Dialogkind", "08G2")
	anderes := seedSchueler(t, pool, "S-VERALTET-2", "Anderes", "08G2")
	titelID := bescheidLernmittel(t, pool, "Biologie 8")

	zurueckEx := exemplar(t, pool, titelID, "ALT-EX-1", true, "")
	zurueck := bescheidAusleihe(t, pool, zurueckEx, sid, 30)
	if _, err := pool.Exec(ctx, `UPDATE ausleihen SET rueckgabe_am = now() WHERE id = $1`, zurueck); err != nil {
		t.Fatal(err)
	}
	gemeldetEx := exemplar(t, pool, titelID, "ALT-EX-2", true, "")
	gemeldet := bescheidAusleihe(t, pool, gemeldetEx, sid, 30)
	if _, err := pool.Exec(ctx, `
		INSERT INTO schadensfaelle (exemplar_id, schueler_id, ausleihe_id, beschreibung, betrag, art)
		VALUES ($1, $2, $3, 'in der Akte gemeldet', 0, 'nicht_zurueckgegeben')`, gemeldetEx, sid, gemeldet); err != nil {
		t.Fatal(err)
	}
	fremdEx := exemplar(t, pool, titelID, "ALT-EX-3", true, "")
	fremd := bescheidAusleihe(t, pool, fremdEx, anderes, 30)

	for name, ausleihe := range map[string]string{"zurückgegeben": zurueck, "schon gemeldet": gemeldet, "fremdes Kind": fremd} {
		rec := bescheidErstellenUeberHandler(t, srv, pool, sid,
			bescheidRumpfMitAusleihen(in28Tagen(), map[string]float64{ausleihe: 10}))
		if rec.Code != http.StatusConflict {
			t.Errorf("%s: Status %d, want 409: %s", name, rec.Code, rec.Body.String())
		}
	}
	var briefe int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schadensersatz_bescheide`).Scan(&briefe); err != nil {
		t.Fatal(err)
	}
	if briefe != 0 || nummernkreisStand(t, pool) != 0 {
		t.Errorf("%d Briefe, Nummernkreis bei %d — want 0 und 0", briefe, nummernkreisStand(t, pool))
	}
	var fremdeBeendet bool
	if err := pool.QueryRow(ctx, `SELECT rueckgabe_am IS NOT NULL FROM ausleihen WHERE id = $1`, fremd).Scan(&fremdeBeendet); err != nil {
		t.Fatal(err)
	}
	if fremdeBeendet {
		t.Error("die Ausleihe des anderen Kindes wurde beendet")
	}
}

// Ein Bücherei-Buch (kein Lernmittel) gehört nicht auf den Brief des Landes — auch nicht
// über den Umweg der Ausleihe. Die Abweisung nimmt den Verlust mit zurück: Das Buch
// bleibt ausgeliehen, das Exemplar im Umlauf.
func TestBescheidErstellen_BuechereiBuchNichtAlsVerlust(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	bescheidReset(t, pool)
	ctx := context.Background()
	bescheidAngabenSetzen(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	sid := seedSchueler(t, pool, "S-BUECHEREI-1", "Leseratte", "08G2")
	titelID := seedMonitorTitel(t, pool, "Roman", "Autor", true, 0)
	ex := exemplar(t, pool, titelID, "BUE-EX-1", true, "")
	a := bescheidAusleihe(t, pool, ex, sid, 30)

	rec := bescheidErstellenUeberHandler(t, srv, pool, sid,
		bescheidRumpfMitAusleihen(in28Tagen(), map[string]float64{a: 9.90}))
	if rec.Code != http.StatusConflict {
		t.Fatalf("Status %d, want 409: %s", rec.Code, rec.Body.String())
	}
	var laeuft, imUmlauf bool
	if err := pool.QueryRow(ctx, `
		SELECT a.rueckgabe_am IS NULL, NOT e.ist_ausgesondert
		FROM ausleihen a JOIN buecher_exemplare e ON e.id = a.exemplar_id WHERE a.id = $1`, a).Scan(&laeuft, &imUmlauf); err != nil {
		t.Fatal(err)
	}
	if !laeuft || !imUmlauf {
		t.Errorf("nach der Abweisung: Ausleihe läuft=%v, Exemplar im Umlauf=%v — want true/true", laeuft, imUmlauf)
	}
	if nummernkreisStand(t, pool) != 0 {
		t.Error("die Abweisung hat eine Nummer verbraucht")
	}
}
