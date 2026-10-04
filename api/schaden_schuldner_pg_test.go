package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/sse"
)

// Der Schuldner einer Forderung ist, wer das Buch geliehen hat. Der Dialog „Verlust/Schaden
// melden" schickt eine schueler_id mit; die Tür nimmt das Feld an und liest es nicht. Eine
// vertippte oder verfälschte Kennung hängt die Forderung deshalb keinem Unbeteiligten an.
func TestSchadenMelden_SchuldnerKommtAusDerAusleihe(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := t.Context()

	authenticator, err := auth.NewAuthenticator(
		"schaden-schuldner-testgeheimnis-32-bytes!!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	var kontoID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Sina', 'Schaden', 'schaden-schuldner@example.org', 'admin', true)
		RETURNING id`).Scan(&kontoID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	sitzung, err := authenticator.GenerateToken(kontoID, "SCHULD-1", auth.RoleAdmin, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	router := NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false).Routes()

	ausleiher := seedSchueler(t, pool, "S-SCHULD-A", "Anna", "07A")
	unbeteiligt := seedSchueler(t, pool, "S-SCHULD-B", "Ben", "07B")
	ex := exemplar(t, pool, titelMitMeldebestand(t, pool, "Schuldnertitel", 1), "EX-SCHULD-1", true, "")
	var ausleihe string
	if err := pool.QueryRow(ctx, `INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		VALUES ($1, $2, now() + interval '14 days') RETURNING id`, ex, ausleiher).Scan(&ausleihe); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}

	req := jsonPost("/api/damage/report", `{"loan_id":"`+ausleihe+`","schueler_id":"`+unbeteiligt+
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
