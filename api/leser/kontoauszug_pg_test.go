package leser

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pdftest"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/testweg"
	"bibliothek/internal/tuertest"
)

// Der Kontoauszug aus der Leserakte trägt den Kopf der Schule aus den Einstellungen und
// kennzeichnet eine abgelaufene Frist. Sind die Einstellungen nicht lesbar, entsteht kein Blatt,
// wie bei Rechnung und Auskunft: Es nennte sonst keine Schule.
func TestKontoauszugPDF_TraegtSchuleUndUeberfaelliges(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	testdaten.SetzeBescheidAngaben(t, pool) // setzt auch Name und Anschrift der Schule
	admin := testdaten.LegeBearbeiterAn(t, pool)
	leserID := testdaten.LegeSchuelerAn(t, pool, "S-KONTO-KOPF", "Lea", "07G1")
	testdaten.LegeAusleiheMitFristAn(t, pool, leserID, "Abgelaufenes Buch", time.Now().AddDate(0, 0, -3))
	testdaten.LegeAusleiheMitFristAn(t, pool, leserID, "Laufendes Buch", time.Now().AddDate(0, 1, 0))

	rufe := func(datenbank db.Pool) *httptest.ResponseRecorder {
		srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: datenbank}}}
		return tuertest.RufeAlsAdmin(t, admin, srv.KontoauszugPDF(), http.MethodGet,
			"/api/print/kontoauszug/"+leserID, "", map[string]string{"schueler_id": leserID})
	}

	rec := rufe(pool)
	if rec.Code != http.StatusOK {
		t.Fatalf("KontoauszugPDF: Status %d, erwartet 200 — %s", rec.Code, pdftest.ErsteBytes(rec.Body.Bytes(), 160))
	}
	texte := pdftest.Texte(t, rec.Body.Bytes())
	for _, soll := range []string{
		"Testschule",
		"Testschule · Schulweg 1 · 12345 Musterstadt",
		"Ein Buch ist überfällig. Bitte gib es so bald wie möglich zurück.",
	} {
		if !slices.Contains(texte, soll) {
			t.Errorf("KontoauszugPDF druckt %q nicht, gedruckt ist %q", soll, texte)
		}
	}

	rec = rufe(testweg.PoolMitLesefehler{Pool: pool, Muster: testweg.SQLEinstellungen})
	if rec.Code != http.StatusInternalServerError || bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
		t.Errorf("KontoauszugPDF bei unlesbaren Einstellungen: Status %d, erwartet 500 ohne Blatt", rec.Code)
	}
}
