package theke

import (
	"encoding/json"
	"strings"
	"testing"

	"bibliothek/repository"
)

// vollerTestLeser trägt in jedem sensiblen Feld einen Markerwert. Taucht einer
// der Marker in einer Theken-Antwort auf, ist die Datenminimierung gerissen.
func vollerTestLeser() repository.Leser {
	geb := "2012-05-14"
	return repository.Leser{
		ID: "s-1", BarcodeID: "20240001737", Vorname: "Mia", Nachname: "Muster",
		Klasse: "7a", IstGesperrt: true, IstVonHandGesperrt: true,
		Strasse: "MARKER-STRASSE", Hausnummer: "MARKER-NR", Plz: "MARKER-PLZ",
		Ort: "MARKER-ORT", ElternEmail: "MARKER-ELTERN@example.org",
		Geburtsdatum: &geb,
	}
}

// verbotenInThekenAntwort sind die Feldinhalte und Schlüssel, die eine Antwort
// auf einem perform_actions-Endpunkt (Theke/Helfer) nie tragen darf —
// Sicherheitsbefund bewertung/sicherheitsbefund-kiosk-suche.md.
var verbotenInThekenAntwort = []string{
	"MARKER-STRASSE", "MARKER-ELTERN", "MARKER-PLZ", "MARKER-ORT", "MARKER-NR",
	"2012-05-14", "strasse", "eltern_email", "geburtsdatum", "plz", "block_reason",
}

func pruefeThekenLeserJSON(t *testing.T, wert any, kontext string) {
	t.Helper()
	raw, err := json.Marshal(wert)
	if err != nil {
		t.Fatalf("%s: marshal: %v", kontext, err)
	}
	for _, verboten := range verbotenInThekenAntwort {
		if strings.Contains(string(raw), verboten) {
			t.Errorf("%s enthält %q — Schüler-PII auf einem Theken-Endpunkt:\n%s", kontext, verboten, raw)
		}
	}
}

// TestThekenAntwortenOhneSchuelerPII beweist die Zusage an beiden Ausgängen:
// die Scan-Antwort (/api/action, inkl. Vorbesitzer bei Fremdrückgabe) und die
// Suchtrefferliste (/api/search) reduzieren auf die Theken-Sicht.
func TestThekenAntwortenOhneSchuelerPII(t *testing.T) {
	s := vollerTestLeser()

	resp := eingabeAntwort{
		Art:         "ausleihe",
		Leser:       zumThekenLeser(&s),
		Vorbesitzer: zumThekenLeser(&s),
	}
	pruefeThekenLeserJSON(t, resp, "eingabeAntwort")

	suche := suchErgebnis{Leser: zuThekenLesern([]repository.Leser{s})}
	pruefeThekenLeserJSON(t, suche, "suchErgebnis")

	// Was die Theke braucht, muss drinbleiben — sonst wäre der leichteste Weg,
	// diesen Test grün zu bekommen, eine leere Antwort.
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, muss := range []string{"Mia", "Muster", "7a", "20240001737", "ist_gesperrt", "is_manually_blocked"} {
		if !strings.Contains(string(raw), muss) {
			t.Errorf("Theken-Sicht verlor ein benötigtes Feld: %q fehlt in %s", muss, raw)
		}
	}

	// Leere Trefferliste bleibt ein leeres Array, kein null (Frontend iteriert).
	leer, err := json.Marshal(suchErgebnis{Leser: zuThekenLesern(nil)})
	if err != nil {
		t.Fatalf("marshal leer: %v", err)
	}
	if !strings.Contains(string(leer), `"students":[]`) {
		t.Errorf("leere Trefferliste muss als [] serialisieren: %s", leer)
	}
}
