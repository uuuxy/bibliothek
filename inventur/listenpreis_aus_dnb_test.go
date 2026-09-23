package inventur

import "testing"

// Der DNB-Ladenpreis füllt den Listenpreis (Migration 127, OFFEN.md 9.8).
//
// Ohne diese Verbindung wäre `listenpreis` eine leere Spalte, die jemand für rund 4.000
// Titel von Hand füllen müsste — und bis dahin rechnete die Staffel des Erlasses ab dem
// zweiten Verleihjahr weiter ersatzweise mit dem Kaufpreis.
//
// Die Quelle gab es schon: Die DNB liefert den Ladenpreis aus MARC21 020 $c in jeder
// Antwort mit, sorgfältig gefiltert (kein DM, keine Umrechnung aus der Umstellungszeit —
// metadaten_preis.go, mit echten Antworten belegt). Er stand bis zum 17.09.2026 ungenutzt
// darin. Gebaut wurde also keine neue Preisquelle, sondern die fehlende Leitung zur
// vorhandenen.
//
// Geprüft wird die Entscheidungsregel, nicht der HTTP-Weg: Dass die DNB-Antwort richtig
// gelesen wird, steht in metadaten_preis_test.go; hier geht es darum, WANN ihr Preis
// übernommen werden darf.
func TestListenpreisAusNachschlagen(t *testing.T) {
	preis := func(p float64) *float64 { return &p }

	faelle := []struct {
		name      string
		vorhanden *float64
		gefunden  float64
		want      *float64
		warum     string
	}{
		{
			name: "leeres Feld wird gefüllt", vorhanden: nil, gefunden: 27.00, want: preis(27.00),
			warum: "sonst bliebe das Feld für den ganzen Bestand leer",
		},
		{
			name: "ein getippter Preis bleibt stehen", vorhanden: preis(19.90), gefunden: 27.00, want: preis(19.90),
			warum: "was der Mensch eingetragen hat, gewinnt gegen jedes Nachschlagen",
		},
		{
			name: "auch eine getippte 0 bleibt stehen", vorhanden: preis(0), gefunden: 27.00, want: preis(0),
			warum: "eine getippte 0 ist eine Aussage („verschenkt bekommen“), keine Lücke",
		},
		{
			name: "kein ermittelbarer Preis lässt das Feld leer", vorhanden: nil, gefunden: 0, want: nil,
			warum: "„nicht ermittelbar“ ist etwas anderes als „kostet nichts“ — eine 0 ergäbe " +
				"einen Ersatzbetrag von 0,00 € im Bescheid",
		},
		{
			name: "ein negativer Fund wird nicht übernommen", vorhanden: nil, gefunden: -5, want: nil,
			warum: "die Spalte verbietet es ohnehin (chk_listenpreis_nonneg) — hier scheitert " +
				"es schon vorher, statt als 500 aus der Datenbank zu kommen",
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got := ListenpreisAusNachschlagen(f.vorhanden, f.gefunden)

			switch {
			case f.want == nil && got != nil:
				t.Errorf("Listenpreis = %.2f, want nil — %s", *got, f.warum)
			case f.want != nil && got == nil:
				t.Errorf("Listenpreis = nil, want %.2f — %s", *f.want, f.warum)
			case f.want != nil && got != nil && *got != *f.want:
				t.Errorf("Listenpreis = %.2f, want %.2f — %s", *got, *f.want, f.warum)
			}
		})
	}
}

// Die Übernahme beim Anlegen (ergaenzeAusNachschlagen): Was fehlt, kommt aus dem
// Nachschlagen, was eingetragen ist, bleibt — auch beim Untertitel, der seit dem 23.09.2026
// mitkommt (OFFEN.md 5.5; der Bestellweg nimmt ihn ebenfalls).
func TestErgaenzeAusNachschlagen_Untertitel(t *testing.T) {
	gefunden := &MetadatenErgebnis{Titel: "Wolkenkind", Untertitel: " Ein Roman ", Preis: 12.00}

	leer := Book{}
	ergaenzeAusNachschlagen(&leer, gefunden)
	if leer.Untertitel != "Ein Roman" {
		t.Errorf("leerer Untertitel: %q, want „Ein Roman“ (getrimmt)", leer.Untertitel)
	}
	if leer.Listenpreis == nil || *leer.Listenpreis != 12.00 {
		t.Errorf("Listenpreis: %v, want 12.00", leer.Listenpreis)
	}

	gepflegt := Book{Untertitel: "Gepflegt"}
	ergaenzeAusNachschlagen(&gepflegt, gefunden)
	if gepflegt.Untertitel != "Gepflegt" {
		t.Errorf("eingetragener Untertitel überschrieben: %q", gepflegt.Untertitel)
	}

	ohne := Book{Title: "Bleibt"}
	ergaenzeAusNachschlagen(&ohne, nil)
	if ohne.Title != "Bleibt" || ohne.Untertitel != "" {
		t.Errorf("ohne Nachschlagen verändert: %+v", ohne)
	}
}
