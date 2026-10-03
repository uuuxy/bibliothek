package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Inventurstart über seine Route. Bereichsbedingung und Überschneidung prüft
// repository/inventur_*_test.go; hier steht, was die Route dazutut: Sie leitet aus der
// Anfrage Bereich und Bezeichnung ab, nennt die erwartete Zahl und beantwortet einen
// belegten Bereich mit 409. Die Oberfläche zeigt Bezeichnung und Zahl aus dieser Antwort.

// inventurStarten ruft POST /api/inventur/start.
func inventurStarten(t *testing.T, srv *Server, benutzerID, rumpf string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/inventur/start", strings.NewReader(rumpf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.InventurStartHandler()(rec, alsBenutzer(req, benutzerID))
	return rec
}

// inventurGestartet verlangt eine 200 und liefert die Antwort.
func inventurGestartet(t *testing.T, rec *httptest.ResponseRecorder) InventurStartResponse {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	var antwort InventurStartResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort nicht lesbar: %v (%s)", err, rec.Body.String())
	}
	if antwort.SessionID == "" {
		t.Fatalf("Antwort ohne Kennung der Inventur: %s", rec.Body.String())
	}
	return antwort
}

// inventurFach trägt das Fach in die Systematik ein, auf die das Fach eines Titels
// verweist, und nimmt es am Ende wieder heraus.
func inventurFach(t *testing.T, pool *pgxpool.Pool, fach string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO systematik_kategorien (kuerzel, bezeichnung) VALUES ($1, $1)`, fach); err != nil {
		t.Fatalf("Fach %q eintragen: %v", fach, err)
	}
	t.Cleanup(func() {
		aufraeumen(t, pool, `UPDATE buecher_titel SET subject = NULL WHERE subject = $1`, fach)
		aufraeumen(t, pool, `DELETE FROM systematik_kategorien WHERE bezeichnung = $1`, fach)
	})
}

// inventurTitel legt einen Titel mit Signatur, Fach und Jahrgangsbereich samt Exemplaren
// an; ein leeres Fach bleibt ohne Eintrag.
func inventurTitel(t *testing.T, pool *pgxpool.Pool, signatur, fach string, von, bis int, barcodes ...string) {
	t.Helper()
	var titelID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO buecher_titel (titel, signatur, subject, jahrgang_von, jahrgang_bis)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5) RETURNING id`,
		"Titel "+signatur, signatur, fach, von, bis).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	for _, b := range barcodes {
		exemplar(t, pool, titelID, b, true, "")
	}
}

func offeneInventuren(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	return zaehleZeilen(t, pool, `SELECT count(*) FROM inventur_sessions WHERE abgeschlossen_am IS NULL`)
}

// Erwartet wird, was im Regal stehen kann: nicht das verliehene, das ausgesonderte und
// das gesperrte Exemplar. Die Inventur trägt den Benutzer, der sie gestartet hat.
func TestInventurStart_GesamtbestandZaehltWasImRegalStehenKann(t *testing.T) {
	pool := pgTestPool(t)
	inventurReset(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	admin := adminFuerAudit(t, pool)
	ctx := context.Background()

	titelID := titelMitSignatur(t, pool, "Lesebuch 5", "BIB Deu 5 KRÜ", 0)
	exemplar(t, pool, titelID, "IS-1", true, "")
	exemplar(t, pool, titelID, "IS-2", true, "")
	exemplar(t, pool, titelID, "IS-GESPERRT", false, "")
	ausgesondert := exemplar(t, pool, titelID, "IS-WEG", true, "")
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET ist_ausgesondert = true, aussonderung_grund = 'AUSSORTIERT' WHERE id = $1`, ausgesondert); err != nil {
		t.Fatalf("Exemplar aussondern: %v", err)
	}
	seedOffeneAusleihe(t, pool, seedSchueler(t, pool, "IS-S1", "Leihkind", "07A"), "IS-VERLIEHEN")

	antwort := inventurGestartet(t, inventurStarten(t, srv, admin, `{"type":"global"}`))
	if antwort.Scope != "global" || antwort.Label != "Gesamtbestand" || antwort.Erwartet != 2 {
		t.Errorf("Antwort %+v, erwartet global, „Gesamtbestand“ und 2 Exemplare", antwort)
	}
	if n := zaehleZeilen(t, pool, `
		SELECT count(*) FROM inventur_sessions
		WHERE id = $1 AND scope_type = 'global' AND gestartet_von = $2 AND abgeschlossen_am IS NULL`,
		antwort.SessionID, admin); n != 1 {
		t.Error("die Inventur steht nicht offen und mit dem angemeldeten Benutzer in der Datenbank")
	}
}

// Die Signatur gilt ohne Leerzeichen am Rand, in der Bezeichnung wie im Bereich: Die
// Bezeichnung darf keinen anderen Bereich nennen, als die Inventur erfasst.
func TestInventurStart_SignaturOhneLeerzeichenAmRand(t *testing.T) {
	pool := pgTestPool(t)
	inventurReset(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	admin := adminFuerAudit(t, pool)

	inventurTitel(t, pool, "BIB Deu 5 KRÜ", "", 5, 6, "IS-D1", "IS-D2")
	inventurTitel(t, pool, "BIB Mat 5", "", 5, 6, "IS-M1")

	antwort := inventurGestartet(t, inventurStarten(t, srv, admin, `{"type":"signature","signatur":"  BIB Deu "}`))
	if antwort.Scope != "signature" || antwort.Label != "Signatur BIB Deu" || antwort.Erwartet != 2 {
		t.Errorf("Antwort %+v, erwartet signature, „Signatur BIB Deu“ und 2 Exemplare", antwort)
	}
	if n := zaehleZeilen(t, pool,
		`SELECT count(*) FROM inventur_sessions WHERE id = $1 AND scope_signatur = 'BIB Deu' AND scope_label = 'Signatur BIB Deu'`,
		antwort.SessionID); n != 1 {
		t.Error("Bereich und Bezeichnung stehen nicht ohne die Leerzeichen in der Datenbank")
	}
}

// Fach und Klasse grenzen den Bereich ein und stehen beide in der Bezeichnung; ein Fach
// allein gilt für alle Klassen.
func TestInventurStart_FilterNachFachUndKlasse(t *testing.T) {
	pool := pgTestPool(t)
	inventurReset(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	admin := adminFuerAudit(t, pool)

	inventurFach(t, pool, "Inventurfach Mathe")
	inventurFach(t, pool, "Inventurfach Deutsch")
	inventurTitel(t, pool, "LMF Mat 5", "Inventurfach Mathe", 5, 6, "IS-F1", "IS-F2")
	inventurTitel(t, pool, "LMF Mat 7", "Inventurfach Mathe", 7, 8, "IS-F3")
	inventurTitel(t, pool, "LMF Deu 5", "Inventurfach Deutsch", 5, 6, "IS-F4")

	antwort := inventurGestartet(t, inventurStarten(t, srv, admin, `{"type":"filter","subject":"Inventurfach Mathe","grade":5}`))
	if antwort.Scope != "filter" || antwort.Label != "Inventurfach Mathe · Kl. 5" || antwort.Erwartet != 2 {
		t.Errorf("Antwort %+v, erwartet filter, „Inventurfach Mathe · Kl. 5“ und 2 Exemplare", antwort)
	}

	antwort = inventurGestartet(t, inventurStarten(t, srv, admin, `{"type":"filter","subject":"Inventurfach Deutsch"}`))
	if antwort.Label != "Inventurfach Deutsch" || antwort.Erwartet != 1 {
		t.Errorf("Antwort %+v, erwartet „Inventurfach Deutsch“ und 1 Exemplar", antwort)
	}
	if n := offeneInventuren(t, pool); n != 2 {
		t.Errorf("%d offene Inventuren, erwartet 2 nebeneinander", n)
	}
}

// Ein belegter Bereich ist eine Lage und keine Störung: 409, mit dem Satz, der sagt,
// welche Inventur im Weg steht. Eine zweite Inventur entsteht dabei nicht.
func TestInventurStart_BelegterBereichIst409(t *testing.T) {
	pool := pgTestPool(t)
	inventurReset(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	admin := adminFuerAudit(t, pool)
	inventurTitel(t, pool, "BIB Deu 5 KRÜ", "", 5, 6, "IS-B1")

	inventurGestartet(t, inventurStarten(t, srv, admin, `{"type":"global"}`))

	faelle := []struct{ name, rumpf, meldung string }{
		{"derselbe Bereich", `{"type":"global"}`, "läuft bereits eine Inventur"},
		{"Regal im laufenden Gesamtbestand", `{"type":"signature","signatur":"BIB Deu"}`, `"Gesamtbestand" läuft bereits`},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			rec := inventurStarten(t, srv, admin, f.rumpf)
			if rec.Code != http.StatusConflict {
				t.Errorf("Status %d, erwartet 409: %s", rec.Code, rec.Body.String())
			}
			if meldung := fehlermeldung(t, rec); !strings.Contains(meldung, f.meldung) {
				t.Errorf("Meldung %q, erwartet %q darin", meldung, f.meldung)
			}
		})
	}
	if n := offeneInventuren(t, pool); n != 1 {
		t.Errorf("%d offene Inventuren nach den Abweisungen, erwartet 1", n)
	}
}
