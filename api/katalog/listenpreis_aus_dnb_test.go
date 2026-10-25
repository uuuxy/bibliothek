package katalog

import (
	"testing"

	"bibliothek/pkg/betrag"
)

// Der Ladenpreis der DNB füllt den Listenpreis.
//
// Ohne diese Verbindung wäre `listenpreis` eine leere Spalte, die jemand für rund 4.000 Titel von
// Hand füllen müsste, und bis dahin rechnete die Staffel des Erlasses ab dem zweiten Verleihjahr
// ersatzweise mit dem Kaufpreis.
//
// Die DNB liefert den Ladenpreis aus MARC21 020 $c in jeder Antwort mit, sorgfältig gefiltert
// (kein DM, keine Umrechnung aus der Zeit der Umstellung; metadaten_preis.go, an echten Antworten
// belegt).
//
// Geprüft wird die Regel der Entscheidung, nicht der Weg über HTTP: Dass die Antwort der DNB
// richtig gelesen wird, steht in metadaten_preis_test.go; hier geht es darum, wann ihr Preis
// übernommen werden darf.
func TestListenpreisAusNachschlagen(t *testing.T) {
	preis := func(p betrag.Cent) *betrag.Cent { return &p }

	faelle := []struct {
		name      string
		vorhanden *betrag.Cent
		gefunden  betrag.Cent
		want      *betrag.Cent
		warum     string
	}{
		{
			name: "leeres Feld wird gefüllt", vorhanden: nil, gefunden: 2700, want: preis(2700),
			warum: "sonst bliebe das Feld für den ganzen Bestand leer",
		},
		{
			name: "ein getippter Preis bleibt stehen", vorhanden: preis(1990), gefunden: 2700, want: preis(1990),
			warum: "was der Mensch eingetragen hat, gewinnt gegen jedes Nachschlagen",
		},
		{
			name: "auch eine getippte 0 bleibt stehen", vorhanden: preis(0), gefunden: 2700, want: preis(0),
			warum: "eine getippte 0 ist eine Aussage („verschenkt bekommen“), keine Lücke",
		},
		{
			name: "kein ermittelbarer Preis lässt das Feld leer", vorhanden: nil, gefunden: 0, want: nil,
			warum: "„nicht ermittelbar“ ist etwas anderes als „kostet nichts“ — eine 0 ergäbe " +
				"einen Ersatzbetrag von 0,00 € im Bescheid",
		},
		{
			name: "ein negativer Fund wird nicht übernommen", vorhanden: nil, gefunden: -500, want: nil,
			warum: "die Spalte verbietet es ohnehin (chk_listenpreis_nonneg) — hier scheitert " +
				"es schon vorher, statt als 500 aus der Datenbank zu kommen",
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got := listenpreisAusNachschlagen(f.vorhanden, f.gefunden)

			switch {
			case f.want == nil && got != nil:
				t.Errorf("Listenpreis = %s, erwartet nil — %s", got.Text(), f.warum)
			case f.want != nil && got == nil:
				t.Errorf("Listenpreis = nil, erwartet %s — %s", f.want.Text(), f.warum)
			case f.want != nil && got != nil && *got != *f.want:
				t.Errorf("Listenpreis = %s, erwartet %s — %s", got.Text(), f.want.Text(), f.warum)
			}
		})
	}
}
