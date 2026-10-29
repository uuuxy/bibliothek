package mahnwesen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pdftest"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/smtptest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
)

// Druck und Versand der Kontoauszüge laden die Schule je für sich aus den Einstellungen. Am
// fertigen PDF beider Wege geprüft: Kopf der Schule und „überfällig" an der abgelaufenen Frist.
func TestAbgaengerKontoauszug_TraegtDieSchuleAusDenEinstellungen(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	testdaten.SetzeBescheidAngaben(t, pool) // setzt auch Name und Anschrift der Schule
	post, sitzungen := tuertest.FangeMailAb(t)
	srv := &Tueren{Basis: &tuer.Basis{Mail: post, DB: &db.Verbindung{Pool: pool}, Uhr: testdaten.SaisonUhr}}
	abgaengerMitBuch(t, pool, "S-KOPF-1", "Anna", "9H", "Annas Buch")
	testdaten.SetzeKlassenleitung(t, pool, "9H", "leitung9h@schule.invalid")

	rec, gedruckt := kontoauszugPDF(t, srv, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("AbgaengerKontoauszuegePDF: Status %d, erwartet 200 — %s", rec.Code, rec.Body.String())
	}

	// Die Klasse in der Schreibweise, in der die Datenbank sie führt, wie im Kettentest.
	req := httptest.NewRequest(http.MethodPost, "/api/abgaenger/mail", strings.NewReader(`{"klassen":["09H"]}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
		&auth.Claims{KontoID: testdaten.LegeBearbeiterAn(t, pool), Rolle: auth.RolleAdmin}))
	rec = httptest.NewRecorder()
	srv.AbgaengerKontoauszuegeVersenden()(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("AbgaengerKontoauszuegeVersenden: Status %d, erwartet 200 — %s", rec.Code, rec.Body.String())
	}
	nachricht := smtptest.WarteAufMail(t, sitzungen)
	gemailt := strings.Join(pdftest.Texte(t, []byte(smtptest.DateiAusMail(t, nachricht, "Anna"))), " ")

	for weg, text := range map[string]string{"Druck": gedruckt, "Versand": gemailt} {
		for _, soll := range []string{"Testschule · Schulweg 1 · 12345 Musterstadt", "überfällig"} {
			if !strings.Contains(text, soll) {
				t.Errorf("%s: Kontoauszug ohne %q:\n%s", weg, soll, text)
			}
		}
	}
}
