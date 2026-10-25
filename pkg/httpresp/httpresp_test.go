package httpresp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

type eintrag struct {
	ID string `json:"id"`
}

// TestEncodeLeereListeAlsArray sichert den Vertrag ab, an dem die Schuelerdatei beim
// Erst-Deployment gescheitert ist: Eine leere Liste muss als [] herausgehen, nicht als
// null. Ein Client, der darauf .length aufruft, bricht bei null ab.
func TestSchreibeJSON_LeereListeAlsArray(t *testing.T) {
	// Exakt das Muster aus den Repositories: "var xs []T" bleibt ohne Treffer nil.
	var leer []eintrag

	var buf bytes.Buffer
	SchreibeJSON(&buf, leer)

	if got := strings.TrimSpace(buf.String()); got != "[]" {
		t.Errorf("nil-Slice muss als [] kodiert werden, war: %s", got)
	}
}

// TestEncodeBefuellteListe stellt sicher, dass die Normalisierung echte Daten nicht
// antastet.
func TestSchreibeJSON_BefuellteListe(t *testing.T) {
	var buf bytes.Buffer
	SchreibeJSON(&buf, []eintrag{{ID: "a"}})

	if got := strings.TrimSpace(buf.String()); got != `[{"id":"a"}]` {
		t.Errorf("befuellte Liste wurde veraendert: %s", got)
	}
}

// Eine Antwort trägt Listen auch als Werte einer Map ({"faecher": …, "titel": …},
// api/portal/schulbuecher.go). Auch dort kommt eine leere Liste als [] an, nicht als null: Das
// null säße eine Ebene tiefer, als die Normalisierung der obersten Ebene reicht.
func TestSchreibeJSON_ListeInMapMitLeererListe(t *testing.T) {
	var leer []eintrag

	var buf bytes.Buffer
	SchreibeJSON(&buf, map[string]any{"titel": leer})

	if got := strings.TrimSpace(buf.String()); got != `{"titel":[]}` {
		t.Errorf(`Liste in einer Map: erwartet {"titel":[]}, war: %s`, got)
	}
}

// TestEncodeNichtListenUnveraendert prueft die Faelle, die NICHT angefasst werden
// duerfen. Vor allem null selbst: Wo eine Antwort bewusst null ist (kein Datensatz),
// darf daraus kein [] werden.
func TestSchreibeJSON_NichtListenUnveraendert(t *testing.T) {
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
		// Ein untypisiertes nil in einer Map ist keine Liste und bleibt null: ein bewusstes
		// „kein Datensatz", etwa {"vorhanden": null} der Dublettenkontrolle vorab.
		{"Map mit nil-Objekt", map[string]any{"vorhanden": nil}, `{"vorhanden":null}`},
		{"Map mit Struct", map[string]any{"titel": eintrag{ID: "y"}}, `{"titel":{"id":"y"}}`},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			var buf bytes.Buffer
			SchreibeJSON(&buf, f.payload)

			if got := strings.TrimSpace(buf.String()); got != f.want {
				t.Errorf("%s = %s, erwartet %s", f.name, got, f.want)
			}
		})
	}
}

// Die Antwort bleibt Byte für Byte, was json.Encoder schrieb, auch mit Zeichen, die er umschreibt.
func TestSchreibeJSON_WieDerEncoder(t *testing.T) {
	antwort := map[string]any{"titel": "<Faust> & Co", "liste": []eintrag{{ID: "a"}}}
	var ist, encoder bytes.Buffer
	SchreibeJSON(&ist, antwort)
	if err := json.NewEncoder(&encoder).Encode(antwort); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if ist.String() != encoder.String() {
		t.Errorf("SchreibeJSON = %q, erwartet %q", ist.String(), encoder.String())
	}
}
