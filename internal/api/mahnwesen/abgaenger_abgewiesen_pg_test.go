package mahnwesen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/auth"
	"bibliothek/internal/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/smtptest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
)

// Die Tür mit Datenbank und echtem Versand: Weist der Mailserver die Mail an die Klassenleitung
// der zuerst gewählten Klasse ab, bekommt die zweite ihre trotzdem. Die Antwort nennt die Klasse,
// an der es lag; die Zahl allein sagte nicht, wessen Adresse zu prüfen ist.
func TestAbgaengerVersand_AbgewieseneKlasseStehtInDerAntwort(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)

	post, sitzungen := tuertest.FangeMailAbMit(t, smtptest.KenntEmpfaengerNicht)
	srv := &Tueren{Basis: &tuer.Basis{Mail: post, DB: &db.Verbindung{Pool: pool}, Uhr: testdaten.SaisonUhr}}

	// Die Liste der Abgänger schreibt die Klasse mit der Null davor, wie die Auswahl sie schickt.
	abgaengerMitBuch(t, pool, "S-ABW-1", "Anna", "9H", "Annas Buch")
	abgaengerMitBuch(t, pool, "S-ABW-2", "Cem", "10R1", "Cems Buch")
	testdaten.SetzeKlassenleitung(t, pool, "9H", smtptest.UnbekannterEmpfaenger)
	testdaten.SetzeKlassenleitung(t, pool, "10R1", "leitung10r1@schule.invalid")

	req := httptest.NewRequest(http.MethodPost, "/api/abgaenger/mail", strings.NewReader(`{"klassen":["09H","10R1"]}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(t.Context(), auth.ClaimsSchluessel,
		&auth.Claims{KontoID: testdaten.LegeBearbeiterAn(t, pool), Rolle: auth.RolleAdmin}))
	rec := httptest.NewRecorder()
	srv.AbgaengerKontoauszuegeVersenden()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/abgaenger/mail = %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	antwort := rec.Body.String()
	for _, soll := range []string{`"sent_count":1`, `"failed_count":1`, `"abgebrochen":false`, "Für 1 Klasse nicht zugestellt: 09H"} {
		if !strings.Contains(antwort, soll) {
			t.Errorf("POST /api/abgaenger/mail = %s, erwartet darin %s", antwort, soll)
		}
	}

	// Zwei Sitzungen am Server: die abgewiesene ohne Nachricht, danach die der 10R1.
	var zugestellt []string
	for range 2 {
		select {
		case sitzung := <-sitzungen:
			if sitzung.Nachricht != "" {
				zugestellt = append(zugestellt, sitzung.Empfaenger...)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("der Server sah keine zweite Sitzung: nach der abgewiesenen Klasse kam keine Mail mehr")
		}
	}
	if len(zugestellt) != 1 || zugestellt[0] != "leitung10r1@schule.invalid" {
		t.Errorf("zugestellte Mails = %v, erwartet genau die an leitung10r1@schule.invalid", zugestellt)
	}
}
