package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/sse"
)

// Eine Klasse, die mit einer langen Ziffernfolge beginnt (eine Buchnummer im Feld Klasse),
// passt als Zahl in keine ganze Zahl der Datenbank. Die Türen, die den Jahrgang aus der
// Klasse lesen, müssen trotzdem für alle anderen Klassen antworten: Jahrgangs-Auswahl und
// Jahrgangsfilter der Leserdatei, der LMF-Planer und die Abgängerliste.
func TestKlasseAusLangerZiffernfolge_LesendeTuerenAntwortenWeiter(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := t.Context()

	authenticator, err := auth.NewAuthenticator(
		"klasse-ziffernfolge-testgeheimnis-32-bytes!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	var kontoID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Zita', 'Ziffer', 'klasse-ziffernfolge@example.org', 'admin', true)
		RETURNING id`).Scan(&kontoID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	sitzung, err := authenticator.GenerateToken(kontoID, "KLZ-1", auth.RoleAdmin, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	srv := NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false)
	// Die Abgängerliste liest ihre Abfrage nur in der Saison.
	srv.Uhr = saisonUhr
	router := srv.Routes()

	fuenfte := schuelerAnlegen(t, pool, "Fuenfte", "05F1", "KLZ-S-1")
	neunte := schuelerAnlegen(t, pool, "Neunte", "09H1", "KLZ-S-2")
	// Die Abgängerliste nennt nur, wer noch ein Buch hat.
	titelID := titelMitMeldebestand(t, pool, "Ziffernfolge Testband", 0)
	exemplarID := exemplar(t, pool, titelID, "KLZ-B-1", false, "")
	if _, err := pool.Exec(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, bearbeiter_id)
		VALUES ($1, $2, now() - interval '30 days', now() + interval '30 days', $3)`,
		exemplarID, neunte, kontoID); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}

	// tueren nennt je Route, was in ihrer Antwort stehen muss.
	tueren := []struct {
		pfad    string
		enthalt []string
	}{
		{"/api/jahrgaenge", []string{"[5,9]"}},
		{"/api/schueler?jahrgang=5", []string{fuenfte}},
		{"/api/lmf-plan/rueckgabe", []string{`"05F1"`, `"09H1"`}},
		{"/api/abgaenger", []string{neunte}},
	}
	pruefe := func(t *testing.T) {
		t.Helper()
		for _, tuer := range tueren {
			req := httptest.NewRequest(http.MethodGet, tuer.pfad, nil)
			req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("GET %s: Status %d, erwartet 200 — %s", tuer.pfad, rec.Code, rec.Body.String())
				continue
			}
			for _, wert := range tuer.enthalt {
				if !strings.Contains(rec.Body.String(), wert) {
					t.Errorf("GET %s: %q fehlt in der Antwort — %s", tuer.pfad, wert, rec.Body.String())
				}
			}
		}
	}

	t.Run("ohne die Klasse aus Ziffern", pruefe)

	// Die zweite Klasse beginnt mit Nullen: Ihre ersten neun Ziffern ergeben 0, die ganze
	// Folge ist trotzdem zu lang.
	schuelerAnlegen(t, pool, "Buchnummer", "9783123456789", "KLZ-S-3")
	schuelerAnlegen(t, pool, "Nullen", "0000000009999999999", "KLZ-S-4")

	t.Run("mit Klassen aus langen Ziffernfolgen", pruefe)
}
