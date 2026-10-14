package bereitschaft

import (
	"fmt"
)

// ferientabelleVorlaufJahre sagt, wie viele Jahre vor dem Ende der Tabelle die Warnung
// kommt. Zwei, weil die KMK den nächsten Block Jahre im Voraus beschließt und der Plan für
// den Sommer im Frühjahr entsteht.
const ferientabelleVorlaufJahre = 2

// pruefeFerientabelle fragt, ob die hinterlegten Sommerferien noch weit genug reichen.
func pruefeFerientabelle(l Lage) Befund {
	b := Befund{Bereich: "Ferientabelle"}
	if l.Jetzt.Year()+ferientabelleVorlaufJahre > l.FerientabelleBis {
		b.Stufe = StufeWarnung
		b.Befund = fmt.Sprintf("Die Sommerferien Hessen sind nur bis %d hinterlegt.", l.FerientabelleBis)
		b.Folge = fmt.Sprintf("Ab dem Schuljahr %d/%d belegt der LMF-Planer den letzten Tag des Büchertauschs "+
			"und den ersten Tag der Bücherausgabe nicht mehr vor — die Termine müssen dann von Hand gesetzt werden.",
			l.FerientabelleBis, (l.FerientabelleBis+1)%100)
		b.Abhilfe = "Den nächsten Beschluss der Kultusministerkonferenz (kmk.org/service/ferienregelung) " +
			"unter Einstellungen → LUSD & Versetzung → Sommerferien eintragen, je Jahr Beginn und Ende."
		return b
	}
	b.Stufe = StufeOK
	b.Befund = fmt.Sprintf("Sommerferien Hessen bis %d hinterlegt.", l.FerientabelleBis)
	return b
}

// pruefeUebrigeFerien fragt, ob die Leihfristen die Herbst-, Weihnachts- und Osterferien
// noch weit genug kennen, mit demselben Vorlauf wie bei den Sommerferien.
func pruefeUebrigeFerien(l Lage) Befund {
	b := Befund{Bereich: "Ferien für Leihfristen"}
	if l.Jetzt.Year()+ferientabelleVorlaufJahre > l.UebrigeFerienBis {
		b.Stufe = StufeWarnung
		b.Befund = fmt.Sprintf("Herbst-, Weihnachts- und Osterferien Hessen sind nur bis %d hinterlegt.", l.UebrigeFerienBis)
		b.Folge = "Danach rückt eine Leihfrist, die in diese Ferien fällt, nicht mehr auf den nächsten Schultag. " +
			"Wer in den Ferien nicht zurückgeben kann, steht nach den Ferien in der Mahnliste."
		b.Abhilfe = "Die Termine stehen im Programm (Quelle: kultus.hessen.de/schulsystem/ferien/ferientermine). " +
			"Auf eine Programmversion aktualisieren, die die folgenden Schuljahre enthält."
		return b
	}
	b.Stufe = StufeOK
	b.Befund = fmt.Sprintf("Herbst-, Weihnachts- und Osterferien Hessen bis %d hinterlegt.", l.UebrigeFerienBis)
	return b
}
