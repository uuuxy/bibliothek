package katalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/repository"
)

// Die Status-Tür nennt nach dem Speichern, ob das Exemplar zum Bestand zählt. Die Karte der
// Buchakte benennt den Zustand danach („Bestellt", „Ausgesondert" oder ein Wort des
// Bestands) und rechnet die Regel von AendereExemplarStatus nicht selbst nach.
func TestExemplarStatus_AntwortNenntDenBestand(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	bookRepo := repository.NewKatalogAbfragen(pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}
	handler := srv.ExemplarStatusAendern(bookRepo, repository.NewBescheidAbfragen(pool))
	titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Status-Titel", "Sta 1", 0)

	ausgesondert := func(barcode string) string {
		t.Helper()
		id := testdaten.LegeExemplarAn(t, pool, titelID, barcode, true, "")
		if err := bookRepo.AendereExemplarStatus(ctx, id, false, true, "", nil, testdaten.LegeBearbeiterAn(t, pool)); err != nil {
			t.Fatalf("aussondern %s: %v", barcode, err)
		}
		return id
	}

	const gesperrt = `{"ist_ausleihbar":false,"ist_ausgesondert":false,"zustand_notiz":"zurückgelegt"}`
	const frei = `{"ist_ausleihbar":true,"ist_ausgesondert":false,"zustand_notiz":""}`
	const verloren = `{"ist_ausleihbar":false,"ist_ausgesondert":true,"zustand_notiz":""}`

	faelle := []struct {
		name      string
		id        string
		koerper   string
		imBestand bool
	}{
		{"bestellt, als gesperrt gespeichert: bleibt bestellt", testdaten.LegeExemplarImZulaufAn(t, pool, titelID, "STA-ZULAUF-SPERRE"), gesperrt, false},
		{"bestellt, freigegeben: im Bestand", testdaten.LegeExemplarImZulaufAn(t, pool, titelID, "STA-ZULAUF-FREI"), frei, true},
		{"bestellt, ausgesondert: nicht im Bestand", testdaten.LegeExemplarImZulaufAn(t, pool, titelID, "STA-ZULAUF-WEG"), verloren, false},
		{"ausgesondert, als gesperrt gespeichert: wieder im Bestand", ausgesondert("STA-WEG-SPERRE"), gesperrt, true},
		{"im Bestand, gesperrt: bleibt im Bestand", testdaten.LegeExemplarAn(t, pool, titelID, "STA-BESTAND", true, ""), gesperrt, true},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/buecher/exemplare/"+f.id+"/status", strings.NewReader(f.koerper))
			req.SetPathValue("id", f.id)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, tuertest.AlsKonto(req, testdaten.LegeBearbeiterAn(t, pool)))
			if rec.Code != http.StatusOK {
				t.Fatalf("Status speichern: HTTP %d: %s", rec.Code, rec.Body.String())
			}
			var antwort struct {
				ImBestand *bool `json:"im_bestand"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
				t.Fatalf("Antwort unlesbar: %v / %s", err, rec.Body.String())
			}
			if antwort.ImBestand == nil {
				t.Fatalf("die Antwort nennt im_bestand nicht: %s", rec.Body.String())
			}
			if *antwort.ImBestand != f.imBestand {
				t.Errorf("im_bestand = %v, erwartet %v", *antwort.ImBestand, f.imBestand)
			}
		})
	}
}
