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

// TestEncodeUmschlagMitLeererListe sichert die Umschlag-Konvention {"data": ...} ab,
// die u. a. das inventur-Paket durchgehend nutzt: Auch dort muss eine leere Liste als
// [] ankommen, nicht als null — das null saesse sonst eine Ebene tiefer als die
// Top-Level-Normalisierung reicht.
func TestEncodeUmschlagMitLeererListe(t *testing.T) {
	var leer []eintrag

	var buf bytes.Buffer
	Encode(&buf, map[string]any{"data": leer})

	if got := strings.TrimSpace(buf.String()); got != `{"data":[]}` {
		t.Errorf(`Umschlag: erwartet {"data":[]}, war: %s`, got)
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
		// Ein untypisiertes nil im Umschlag ist KEINE Liste — es muss null bleiben
		// (bewusstes "kein Datensatz", z. B. {"data": null} bei Einzelobjekten).
		{"Umschlag mit nil-Objekt", map[string]any{"data": nil}, `{"data":null}`},
		{"Umschlag mit Struct", map[string]any{"data": eintrag{ID: "y"}}, `{"data":{"id":"y"}}`},
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
