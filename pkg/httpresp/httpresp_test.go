package httpresp

import (
	"bytes"
	"strings"
	"testing"
)

type eintrag struct {
	ID string `json:"id"`
}

// TestEncodeLeereListeAlsArray sichert den Vertrag ab, an dem die Schuelerdatei beim
// Erst-Deployment gescheitert ist: Eine leere Liste muss als [] herausgehen, nicht als
// null. Ein Client, der darauf .length aufruft, bricht bei null ab.
func TestEncodeLeereListeAlsArray(t *testing.T) {
	// Exakt das Muster aus den Repositories: "var xs []T" bleibt ohne Treffer nil.
	var leer []eintrag

	var buf bytes.Buffer
	Encode(&buf, leer)

	if got := strings.TrimSpace(buf.String()); got != "[]" {
		t.Errorf("nil-Slice muss als [] kodiert werden, war: %s", got)
	}
}

// TestEncodeBefuellteListe stellt sicher, dass die Normalisierung echte Daten nicht
// antastet.
func TestEncodeBefuellteListe(t *testing.T) {
	var buf bytes.Buffer
	Encode(&buf, []eintrag{{ID: "a"}})

	if got := strings.TrimSpace(buf.String()); got != `[{"id":"a"}]` {
		t.Errorf("befuellte Liste wurde veraendert: %s", got)
	}
}

// TestEncodeNichtListenUnveraendert prueft die Faelle, die NICHT angefasst werden
// duerfen. Vor allem null selbst: Wo eine Antwort bewusst null ist (kein Datensatz),
// darf daraus kein [] werden.
func TestEncodeNichtListenUnveraendert(t *testing.T) {
	faelle := []struct {
		name    string
		payload any
		want    string
	}{
		{"Objekt", map[string]string{"status": "success"}, `{"status":"success"}`},
		{"Struct", eintrag{ID: "x"}, `{"id":"x"}`},
		{"nil-Payload bleibt null", nil, "null"},
		{"leere Map bleibt {}", map[string]string{}, "{}"},
		{"Zahl", 42, "42"},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			var buf bytes.Buffer
			Encode(&buf, f.payload)

			if got := strings.TrimSpace(buf.String()); got != f.want {
				t.Errorf("%s: erwartet %s, war %s", f.name, f.want, got)
			}
		})
	}
}
