package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/repository"
)

// Die Tür der Leserliste prüft die Sortierung, statt sie stillschweigend zu ersetzen.
//
// Der stille Ersatzwert ist in diesem Projekt zweimal teuer geworden. Hier wäre er
// besonders heimtückisch: Die Liste käme sortiert nach irgendetwas zurück, die
// Oberfläche zeigte ihren Pfeil an der geklickten Spalte, und niemand würde je
// erfahren, dass beides nicht zusammenpasst.
func TestLeserListe_UnbekannteSortierungIstEin400(t *testing.T) {
	// Kein Repository nötig: Die Prüfung sitzt VOR jedem Datenbankzugriff. Käme sie
	// später, stürzte dieser Test am nil-Repository ab — das ist der Nachweis.
	handler := (&Server{}).ListStudentsHandler(nil, nil)

	faelle := []struct {
		name      string
		anfrage   string
		imKoerper string
	}{
		{"unbekannte Spalte", "?sortierung=nachname", "erlaubt sind"},
		// SQL im Spaltennamen. KEIN Semikolon im Beispiel: Go verwirft seit 1.17 einen
		// ganzen Query-Parameter, der ein Semikolon enthält (URL.Query ignoriert ihn
		// still) — die Anfrage käme also als „keine Sortierung" an und wäre erlaubt.
		// Das ist zufälliger Schutz; geprüft wird der, der hier gebaut ist.
		{"SQL im Spaltennamen", "?sortierung=nachname+DESC+--", "Unbekannte Sortierung"},
		{"Klammerausdruck", "?sortierung=%28SELECT+1%29", "Unbekannte Sortierung"},
		{"unbekannte Richtung", "?sortierung=name&richtung=aufwaerts", "Unbekannte Richtung"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/schueler"+f.anfrage, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), f.imKoerper) {
				t.Errorf("Antwort %q, erwartet darin %q", rec.Body.String(), f.imKoerper)
			}
		})
	}
}

// Die erlaubten Werte kommen aus DERSELBEN Liste, die das Repository kennt — sonst
// erlaubt die Tür etwas, das die Abfrage nicht sortieren kann, oder umgekehrt.
func TestLeserListe_ErlaubteSortierungen(t *testing.T) {
	for _, spalte := range repository.SchuelerSortierspalten {
		for _, richtung := range []string{"", "auf", "ab"} {
			req := httptest.NewRequest(http.MethodGet,
				"/api/schueler?sortierung="+string(spalte)+"&richtung="+richtung, nil)
			sortierung, apiErr := leserSortierung(req)
			if apiErr != nil {
				t.Errorf("Spalte %q, Richtung %q: %v", spalte, richtung, apiErr)
				continue
			}
			if sortierung.Spalte != spalte {
				t.Errorf("Spalte %q kam als %q an", spalte, sortierung.Spalte)
			}
			if sortierung.Absteigend != (richtung == "ab") {
				t.Errorf("Richtung %q ergab Absteigend=%v", richtung, sortierung.Absteigend)
			}
		}
	}
}
