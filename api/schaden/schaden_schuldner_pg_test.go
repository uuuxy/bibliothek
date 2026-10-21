package schaden_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/api"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/sse"
)

// Der Schuldner einer Forderung ist, wer das Buch geliehen hat. Der Dialog „Verlust/Schaden
// melden" schickt eine schueler_id mit; die Tür nimmt das Feld an und liest es nicht. Eine
// vertippte oder verfälschte Kennung hängt die Forderung deshalb keinem Unbeteiligten an.
func TestSchadenMelden_SchuldnerKommtAusDerAusleihe(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := t.Context()

	authenticator, err := auth.NewAnmeldung(
		"schaden-schuldner-testgeheimnis-32-bytes!!", pool, time.Hour)
	if err != nil {
		t.Fatalf("NewAnmeldung: %v", err)
	}
	kontoID := testdaten.LegeKontoAn(t, pool, "Sina", "Schaden", "schaden-schuldner@example.org", "admin", true)
	sitzung, err := authenticator.ErzeugeToken(kontoID, auth.RolleAdmin, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	router := api.NewServer(&db.Verbindung{Pool: pool}, authenticator, sse.NewBroker(), api.ServerKonfig{}).Routes()

	ausleiher := testdaten.LegeSchuelerAn(t, pool, "S-SCHULD-A", "Anna", "07A")
	unbeteiligt := testdaten.LegeSchuelerAn(t, pool, "S-SCHULD-B", "Ben", "07B")
	ex := testdaten.LegeExemplarAn(t, pool, testdaten.LegeTitelAn(t, pool, "Schuldnertitel", 1), "EX-SCHULD-1", true, "")
	var ausleihe string
	if err := pool.QueryRow(ctx, `INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		VALUES ($1, $2, now() + interval '14 days') RETURNING id`, ex, ausleiher).Scan(&ausleihe); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}

	req := tuertest.JSONPost("/api/damage/report", `{"loan_id":"`+ausleihe+`","schueler_id":"`+unbeteiligt+
		`","copy_id":"`+ex+`","beschreibung":"Riss","art":"beschaedigt","betrag":4}`)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Schaden melden: Status %d — %s", rec.Code, rec.Body.String())
	}

	var gebucht string
	if err := pool.QueryRow(ctx,
		`SELECT schueler_id::text FROM schadensfaelle WHERE ausleihe_id = $1`, ausleihe).Scan(&gebucht); err != nil {
		t.Fatalf("Forderung lesen: %v", err)
	}
	if gebucht != ausleiher {
		t.Errorf("die Forderung steht bei %s; sie gehört dem Ausleiher %s, nicht der mitgeschickten Kennung %s",
			gebucht, ausleiher, unbeteiligt)
	}
}

// „Nicht zurückgegeben" für ein Buch, das an einem anderen Platz schon zurückkam: Die Akte
// zeigt die Ausleihe noch, die Tür lehnt mit einem Satz ab (409), sondert das Buch nicht aus
// und legt keine Forderung an.
func TestSchadenMelden_VerlustFuerZurueckgegebenesBuch(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := t.Context()

	authenticator, err := auth.NewAnmeldung(
		"schaden-zurueck-testgeheimnis-32-bytes!!!!", pool, time.Hour)
	if err != nil {
		t.Fatalf("NewAnmeldung: %v", err)
	}
	kontoID := testdaten.LegeKontoAn(t, pool, "Zora", "Zurueck", "schaden-zurueck@example.org", "admin", true)
	sitzung, err := authenticator.ErzeugeToken(kontoID, auth.RolleAdmin, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	router := api.NewServer(&db.Verbindung{Pool: pool}, authenticator, sse.NewBroker(), api.ServerKonfig{}).Routes()

	kind := testdaten.LegeSchuelerAn(t, pool, "S-ZURUECK-A", "Zoe", "07A")
	ex := testdaten.LegeExemplarAn(t, pool, testdaten.LegeTitelAn(t, pool, "Zurücktitel", 1), "EX-ZURUECK-1", true, "")
	var ausleihe string
	if err := pool.QueryRow(ctx, `INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist, rueckgabe_am)
		VALUES ($1, $2, now() + interval '14 days', now()) RETURNING id`, ex, kind).Scan(&ausleihe); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}

	req := tuertest.JSONPost("/api/damage/report", `{"loan_id":"`+ausleihe+`","copy_id":"`+ex+
		`","beschreibung":"Verloren","art":"nicht_zurueckgegeben","betrag":12}`)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("Status %d, erwartet 409 — %s", rec.Code, rec.Body.String())
	}
	const erwartet = "Ein gewähltes Buch ist inzwischen zurückgegeben — bitte den Dialog neu öffnen."
	if got := tuertest.Fehlermeldung(t, rec); got != erwartet {
		t.Errorf("SchadenMelden: Meldung %q, erwartet %q", got, erwartet)
	}

	var ausgesondert bool
	var forderungen int
	if err := pool.QueryRow(ctx, `
		SELECT (SELECT ist_ausgesondert FROM buecher_exemplare WHERE id = $1),
		       (SELECT count(*) FROM schadensfaelle WHERE ausleihe_id = $2)`, ex, ausleihe).
		Scan(&ausgesondert, &forderungen); err != nil {
		t.Fatalf("Stand lesen: %v", err)
	}
	if ausgesondert || forderungen != 0 {
		t.Errorf("ausgesondert=%v, Forderungen=%d: Das Buch steht im Regal, gebucht werden darf nichts", ausgesondert, forderungen)
	}
}
