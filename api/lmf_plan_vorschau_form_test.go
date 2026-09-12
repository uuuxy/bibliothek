package api

import (
	"encoding/json"
	"strings"
	"testing"

	"bibliothek/repository"
)

// Vorschau und gespeicherter Plan müssen dieselbe FORM haben (#593, Paket 5).
//
// Der Planer rechnet die Vorschau mit demselben Aufruf, mit dem er speichert („vorschau":
// true) — genau damit es keine zweite Wahrheit gibt. Die Zeilen kamen trotzdem
// verschieden zurück: Eine Zeile ohne Klassen („Nachzügler", „Aufräumen" — es gibt sie
// im echten Plan) lieferte in der Vorschau `klassen: null`, nach dem Speichern `[]`. Die
// Oberfläche muss dann beide Formen kennen, und wer das vergisst, bekommt beim ersten
// Vermerk ohne Klasse einen Fehler an einer Stelle, die mit der Vorschau nichts zu tun
// hat.
func TestVorschauZeileOhneKlassenLiefertLeereListe(t *testing.T) {
	req := lmfPlanRequest{
		ErsterTag:    "2027-08-16",
		Startstunde:  2,
		StundenJeTag: 6,
	}
	req.Zeilen = append(req.Zeilen, struct {
		Klassen []string `json:"klassen"`
		Vermerk string   `json:"vermerk"`
		// Fest: Datum und Stunde dieser Zeile von Hand — null, wenn sie fließt.
		Fest *struct {
			Datum  string `json:"datum"`
			Stunde int    `json:"stunde"`
		} `json:"fest"`
	}{Klassen: nil, Vermerk: "Nachzügler"})

	e, err := pruefeLmfPlan(repository.LmfTerminAusgabe, req)
	if err != nil {
		t.Fatalf("Plan prüfen: %v", err)
	}
	if len(e.Zeilen) != 1 {
		t.Fatalf("%d Zeilen, erwartet 1", len(e.Zeilen))
	}

	roh, err := json.Marshal(e.Zeilen[0])
	if err != nil {
		t.Fatalf("Zeile in JSON: %v", err)
	}
	if strings.Contains(string(roh), `"klassen":null`) {
		t.Errorf("die Vorschau liefert `klassen: null`, der gespeicherte Plan `[]` — "+
			"zwei Formen derselben Zeile: %s", roh)
	}
	if !strings.Contains(string(roh), `"klassen":[]`) {
		t.Errorf("erwartet `klassen: []` wie nach dem Speichern (schreibeKlassen), war: %s", roh)
	}
}
