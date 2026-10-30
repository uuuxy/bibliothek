package druck

import (
	"encoding/json"
	"testing"

	"bibliothek/repository"
)

func TestDesignMitSchule(t *testing.T) {
	const kopf = `{"schule":{"name":"Musterschule","adresse":"Schulweg 1, 12345 Neustadt"}`
	schule := &repository.SystemEinstellungen{
		SchuleName: "Musterschule", SchuleStrasse: "Schulweg 1", SchulePLZ: "12345", SchuleOrt: "Neustadt",
	}
	faelle := []struct {
		name, design, soll string
	}{
		{"noch kein Design", `{}`, kopf + `}`},
		{"leeres Objekt mit Leerzeichen", ` { } `, kopf + `}`},
		{"das Design bleibt Byte für Byte", `{"b": 1,  "a":[ 2 ]}`, kopf + `,"b": 1,  "a":[ 2 ]}`},
		{"kein Objekt gilt als nicht gespeichert", `[1]`, kopf + `}`},
	}
	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			ist, err := designMitSchule(fall.design, schule)
			if err != nil {
				t.Fatal(err)
			}
			if string(ist) != fall.soll {
				t.Errorf("ergibt %s, erwartet %s", ist, fall.soll)
			}
			if !json.Valid(ist) {
				t.Errorf("kein gültiges JSON: %s", ist)
			}
		})
	}
}

func TestDesignMitSchule_Anschrift(t *testing.T) {
	faelle := []struct {
		name string
		e    repository.SystemEinstellungen
		soll string
	}{
		{"alles da", repository.SystemEinstellungen{SchuleStrasse: "Schulweg 1", SchulePLZ: "12345", SchuleOrt: "Neustadt"}, "Schulweg 1, 12345 Neustadt"},
		{"ohne Straße", repository.SystemEinstellungen{SchulePLZ: "12345", SchuleOrt: "Neustadt"}, "12345 Neustadt"},
		{"ohne Ort", repository.SystemEinstellungen{SchuleStrasse: "Schulweg 1", SchulePLZ: "12345"}, "Schulweg 1, 12345"},
		{"nur Ort", repository.SystemEinstellungen{SchuleOrt: "Neustadt"}, "Neustadt"},
		{"nichts", repository.SystemEinstellungen{}, ""},
	}
	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			ist, err := designMitSchule("{}", &fall.e)
			if err != nil {
				t.Fatal(err)
			}
			var antwort struct {
				Schule ausweisSchule `json:"schule"`
			}
			if err := json.Unmarshal(ist, &antwort); err != nil {
				t.Fatal(err)
			}
			if antwort.Schule.Adresse != fall.soll {
				t.Errorf("Anschrift %q, erwartet %q", antwort.Schule.Adresse, fall.soll)
			}
		})
	}
}
