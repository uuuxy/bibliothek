package api

import (
	"fmt"

	"bibliothek/repository"
)

// Der Vermerk, der auf jeder Bestellung steht: aus welchem Topf sie bezahlt wird.
//
// Der Leitfaden zur Lernmittelfreiheit verlangt, dass die Schule schon auf der BESTELLUNG
// vermerkt, ob die Bücher im Rahmen der Lernmittelfreiheit beschafft werden oder für die
// Schülerbücherei — der Händler richtet danach den Nachlass, und die Rechnung geht je
// nach Topf einen anderen Weg (Land: nach Prüfung ans Staatliche Schulamt; Schulträger:
// bleibt bei der Schule). Bis zum 10.09.2026 behauptete jedes Anschreiben „für unsere
// Schulbibliothek", auch für Lernmittel-Klassensätze — der Vermerk stand also da, nur
// falsch herum.
//
// EINE Quelle für Anschreiben (order_pdf.go) und Mail (bestellmail_text.go): Zwei
// Formulierungen desselben Vermerks liefen früher oder später auseinander, und der
// Händler hielte dann zwei Dokumente in der Hand, die sich widersprechen.
//
// Bewusst ohne Namen des Schulträgers: Die Anwendung kennt ihn nicht, und der Leitfaden
// verlangt nur die Unterscheidung — nicht die Behörde.
type mittelText struct {
	// Kurz: das eine Wort für Betreffzeile und Platzhalter {{.Mittel}}.
	Kurz string
	// Betreff: die Betreffzeile des Anschreibens.
	Betreff string
	// Vermerk: der ganze Satz im Anschreiben (und in der Mail, wenn die Vorlage den
	// Platzhalter nicht trägt).
	Vermerk string
}

var mittelTexte = map[string]mittelText{
	repository.MittelLand: {
		Kurz:    "Lernmittelfreiheit",
		Betreff: "Bestellung im Rahmen der Lernmittelfreiheit",
		Vermerk: "Die Bücher werden im Rahmen der Lernmittelfreiheit beschafft (Sammelbestellung, Eigentum des Landes Hessen).",
	},
	repository.MittelSchultraeger: {
		Kurz:    "Schülerbücherei",
		Betreff: "Bestellung für die Schülerbücherei",
		Vermerk: "Die Bücher sind eine Anschaffung für die Schülerbücherei aus Mitteln des Schulträgers — keine Beschaffung im Rahmen der Lernmittelfreiheit.",
	},
}

// mittelTexteFuer liefert die Texte zum Topf. Ein unbekannter Wert ist ein Fehler und
// kein leerer Vermerk: Ein Anschreiben ohne Vermerk wäre genau das Dokument, das diese
// Datei abschaffen soll.
func mittelTexteFuer(mittel string) (mittelText, error) {
	t, ok := mittelTexte[mittel]
	if !ok {
		return mittelText{}, fmt.Errorf("unbekannter Mittel-Topf %q", mittel)
	}
	return t, nil
}
