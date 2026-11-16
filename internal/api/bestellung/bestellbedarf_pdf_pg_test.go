package bestellung

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/db"
	"bibliothek/internal/pdftest"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/repository"
	"bibliothek/internal/testdaten"

	"github.com/google/go-cmp/cmp"
)

// Die Bestellliste geht ungelesen an den Händler: Je Buch stehen Bedarf, Bestand und als Menge,
// was fehlt, dieselbe Zahl, die der Warenkorb übernimmt. Ein Buch, dessen Bestand reicht, steht
// nicht darauf.
func TestBestellbedarfPDF_ZeileNenntBedarfBestandUndMenge(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	if _, err := pool.Exec(ctx, `
		INSERT INTO system_einstellungen (schluessel, wert) VALUES ('bestellbedarf_warnung_aktiv', 'true')
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`); err != nil {
		t.Fatalf("Warnung einschalten: %v", err)
	}
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM system_einstellungen WHERE schluessel = 'bestellbedarf_warnung_aktiv'`)
	})

	// Drei Kinder der 7 haben ihr Schulbuch für dieses Schuljahr, fünf rücken aus der 6 nach.
	frist := repository.SchuljahrBeginn(time.Now()).AddDate(1, 0, 0).Add(-time.Hour)
	for i := range 5 {
		testdaten.LegeSchuelerAn(t, pool, fmt.Sprintf("BBL-6-%d", i), "Kind", "06a")
	}
	siebte := make([]string, 3)
	for i := range siebte {
		siebte[i] = testdaten.LegeSchuelerAn(t, pool, fmt.Sprintf("BBL-7-%d", i), "Kind", "07a")
	}
	// leiheAus legt anzahl Exemplare an und gibt der Reihe nach jedem Kind der 7 eines.
	leiheAus := func(titelID, praefix string, anzahl int) {
		t.Helper()
		for i := range anzahl {
			ex := testdaten.LegeExemplarAn(t, pool, titelID, fmt.Sprintf("%s%d", praefix, i), true, "")
			if i >= len(siebte) {
				continue
			}
			if _, err := pool.Exec(ctx, `INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
				VALUES ($1, $2, $3)`, ex, siebte[i], frist); err != nil {
				t.Fatalf("Ausleihe anlegen: %v", err)
			}
		}
	}
	knapp := testdaten.LegeTitelAn(t, pool, "LMF-Mathe 7")
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET autor = 'Ada Autorin', isbn = '9783060000017' WHERE id = $1`, knapp); err != nil {
		t.Fatalf("Titelangaben setzen: %v", err)
	}
	leiheAus(knapp, "BBL-M-", 3)
	leiheAus(testdaten.LegeTitelAn(t, pool, "LMF-Deutsch 7"), "BBL-D-", 5)

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
	if i < 0 || i+6 > len(texte) {
		t.Fatalf("die Zeile des knappen Titels fehlt auf der Liste: %q", texte)
	}
	// Titel, Autor, ISBN, Bedarf, Bestand, Menge.
	if diff := cmp.Diff([]string{"LMF-Mathe 7", "Ada Autorin", "9783060000017", "5", "3", "2"}, texte[i:i+6]); diff != "" {
		t.Errorf("Zeile des knappen Titels (-erwartet +gedruckt):\n%s", diff)
	}
	if slices.Contains(texte, "LMF-Deutsch 7") {
		t.Errorf("ein Buch mit 5 Exemplaren für 5 nachrückende Kinder steht auf der Bestellliste")
	}
}
