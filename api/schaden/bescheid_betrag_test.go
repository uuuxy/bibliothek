package schaden

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"bibliothek/apierrors"
	"bibliothek/pkg/schulzeit"
)

// Ein Bescheid fordert für jedes Buch einen Betrag ab einem Cent. Vorher nahm der Server 0 €
// an (Referenznummer verbraucht, Buch als Verlust gebucht, nichts gefordert), und ein
// negativer Betrag endete an der Regel der Datenbank als 500 ohne Grund.
func TestPruefeBescheidAnfrage_Betraege(t *testing.T) {
	frist := schulzeit.Jetzt().AddDate(0, 0, 28).Format("2006-01-02")
	faelle := []struct {
		name       string
		positionen string
		ausleihen  string
		gueltig    bool
	}{
		{"Buch mit 0 €", `[]`, `[{"ausleihe_id":"a1","betrag":0}]`, false},
		{"Forderung mit 0 €", `[{"schadensfall_id":"s1","betrag":0}]`, `[]`, false},
		{"negativer Betrag", `[{"schadensfall_id":"s1","betrag":-25}]`, `[]`, false},
		{"rundet auf 0 Cent", `[]`, `[{"ausleihe_id":"a1","betrag":0.004}]`, false},
		{"ein gültiges, ein leeres Buch", `[{"schadensfall_id":"s1","betrag":12.5}]`, `[{"ausleihe_id":"a1","betrag":0}]`, false},
		{"ein Cent", `[]`, `[{"ausleihe_id":"a1","betrag":0.01}]`, true},
		{"beide Quellen gültig", `[{"schadensfall_id":"s1","betrag":12.5}]`, `[{"ausleihe_id":"a1","betrag":41.5}]`, true},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			var req bescheidErstellenAnfrage
			rumpf := `{"mittel":"land","frist_bis":"` + frist + `","positionen":` + f.positionen + `,"ausleihen":` + f.ausleihen + `}`
			if err := json.Unmarshal([]byte(rumpf), &req); err != nil {
				t.Fatalf("Rumpf %s: %v", rumpf, err)
			}
			_, err := pruefeBescheidAnfrage(req)
			if f.gueltig {
				if err != nil {
					t.Fatalf("pruefeBescheidAnfrage: %v, erwartet gültig", err)
				}
				return
			}
			var apiErr *apierrors.Fehler
			if !errors.As(err, &apiErr) || apiErr.Status != http.StatusBadRequest {
				t.Fatalf("pruefeBescheidAnfrage: %v, erwartet 400", err)
			}
			if apiErr.Meldung != "Bitte für jedes Buch einen Betrag über 0 € eintragen" {
				t.Errorf("Meldung %q", apiErr.Meldung)
			}
		})
	}
}
