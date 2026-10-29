package bestellung

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pdftest"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"

	"github.com/google/go-cmp/cmp"
)

// Die Bestellliste geht ungelesen an den Händler: Als Soll steht die eingestellte Schwelle, nicht der
// Meldebestand am Titel, den niemand mehr pflegt, und nachbestellt wird Soll minus Gesamtbestand,
// weil verliehene Exemplare zurückkommen. Ein Titel, der die Schwelle erreicht, steht nicht darauf.
func TestBestellbedarfPDF_ZeileRechnetMitDerSchwelle(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	for schluessel, wert := range map[string]string{"bestellbedarf_warnung_aktiv": "true", "bestellbedarf_schwelle": "30"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)
			ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, schluessel, wert); err != nil {
			t.Fatalf("Einstellung %s setzen: %v", schluessel, err)
		}
	}
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM system_einstellungen WHERE schluessel IN ('bestellbedarf_warnung_aktiv', 'bestellbedarf_schwelle')`)
	})

	// exemplare legt anzahl Exemplare an, davon ausleihbar viele ausleihbar.
	exemplare := func(titelID, praefix string, anzahl, ausleihbar int) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			SELECT $1, $2 || g, g <= $3 FROM generate_series(1, $4) g`, titelID, praefix, ausleihbar, anzahl); err != nil {
			t.Fatalf("Exemplare anlegen: %v", err)
		}
	}
	knapp := testdaten.LegeTitelAn(t, pool, "LMF-Mathe 7", 5)
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET autor = 'Ada Autorin', isbn = '9783060000017' WHERE id = $1`, knapp); err != nil {
		t.Fatalf("Titelangaben setzen: %v", err)
	}
	exemplare(knapp, "BBL-M-", 10, 7)
	exemplare(testdaten.LegeTitelAn(t, pool, "LMF-Deutsch 7", 5), "BBL-D-", 30, 30)

	rec := httptest.NewRecorder()
	srv.BestellbedarfPDF().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/bestellungen/pdf", nil))
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != tuer.ContentTypePDF {
		t.Fatalf("Status %d, Content-Type %q; erwartet 200 und ein PDF: %.200s", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	// Die Seite lädt die Liste über einen schlichten Link; den Namen der Datei setzt diese Zeile.
	if art := rec.Header().Get("Content-Disposition"); art != "attachment; filename=bestellliste.pdf" {
		t.Errorf("Content-Disposition %q, erwartet den Download als bestellliste.pdf", art)
	}

	texte := pdftest.TexteInReihenfolge(t, rec.Body.Bytes())
	i := slices.Index(texte, "LMF-Mathe 7")
	if i < 0 || i+7 > len(texte) {
		t.Fatalf("die Zeile des knappen Titels fehlt auf der Liste: %q", texte)
	}
	// Titel, Autor, ISBN, Soll, verfügbar, gesamt, nachzubestellen.
	if diff := cmp.Diff([]string{"LMF-Mathe 7", "Ada Autorin", "9783060000017", "30", "7", "10", "20"}, texte[i:i+7]); diff != "" {
		t.Errorf("Zeile des knappen Titels (-erwartet +gedruckt):\n%s", diff)
	}
	if slices.Contains(texte, "LMF-Deutsch 7") {
		t.Errorf("ein Titel mit 30 Exemplaren steht bei der Schwelle 30 auf der Bestellliste")
	}
}
