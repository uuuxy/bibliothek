package bereitschaft

import (
	"strings"
)

// pruefeBescheidAngaben fragt, ob sich ein Schadensersatz-Bescheid erstellen lässt. Eine
// Warnung, weil nichts ausfällt, solange niemand einen Bescheid braucht; die Schule soll
// die Lücke vorher kennen und nicht erst am Tag des Falls.
func pruefeBescheidAngaben(l Lage) Befund {
	b := Befund{Bereich: "Schadensersatz-Bescheid"}
	if l.BescheidFehlend == nil {
		b.Stufe = StufeWarnung
		b.Befund = "Die Pflichtangaben wurden nicht erhoben — die Einstellungen waren nicht lesbar."
		b.Folge = "Unklar, ob sich ein Bescheid erstellen lässt."
		b.Abhilfe = abhilfeDBNeuLaden
		return b
	}
	if len(l.BescheidFehlend) > 0 {
		b.Stufe = StufeWarnung
		b.Befund = "Es fehlen: " + strings.Join(l.BescheidFehlend, ", ") + "."
		b.Folge = "Ein Bescheid lässt sich erst erstellen, wenn diese Angaben stehen — das Erstellen weist bis dahin ab."
		b.Abhilfe = "Einstellungen → Schadensersatz: die fehlenden Angaben eintragen; die Anschrift der Schule steht unter Einstellungen → Schule."
		return b
	}
	b.Stufe = StufeOK
	b.Befund = "Alle Pflichtangaben für den Bescheid stehen."
	return b
}
