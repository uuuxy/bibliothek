package repository

// Die vier Gründe einer Aussonderung (chk_aussonderung_grund) und ihr Text. Abgangsbuch, Ausdruck
// und Löschdialog nennen dieselben vier Wörter; die Oberfläche führt sie in
// frontend/src/lib/utils/aussonderungsgruende.js, und beide Seiten prüfen gegen
// aussonderungsgruende.faelle.json (aussonderungsgrund_zwilling_pg_test.go).

const (
	grundVerlust           = "VERLUST"
	grundBeschaedigung     = "BESCHAEDIGUNG"
	grundAussortiert       = "AUSSORTIERT"
	grundBestandskorrektur = "BESTANDSKORREKTUR"
)

// IstAussonderungsgrund sagt, ob der Text einer der vier Gründe ist. Die Türen prüfen damit, was
// der Löschdialog schickt, bevor die Datenbank es an der CHECK-Bedingung abweist.
func IstAussonderungsgrund(grund string) bool {
	switch grund {
	case grundVerlust, grundBeschaedigung, grundAussortiert, grundBestandskorrektur:
		return true
	}
	return false
}

// abgangsgrundText übersetzt den gespeicherten Grund in die Wörter der Schule, wie sie in Littera
// am häufigsten standen.
func abgangsgrundText(grund string) string {
	switch grund {
	case grundAussortiert:
		return "Alter Bestand"
	case grundBeschaedigung:
		return "Beschädigt"
	case grundVerlust:
		return "Verlust"
	case grundBestandskorrektur:
		return "Falsch angelegt"
	case "":
		// Kommt nur bei Altdaten vor: chk_aussonderung_grund verlangt einen Grund. Ein leeres
		// Feld im Nachweis sähe nach Druckfehler aus.
		return "ohne Angabe"
	default:
		return grund
	}
}
