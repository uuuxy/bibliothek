package api

import (
	"fmt"

	"bibliothek/repository"
)

// Der Vermerk, der auf jeder Bestellung steht: aus welchem Topf sie bezahlt wird.
//
// Die Schule vermerkt schon auf der BESTELLUNG, ob die Bücher Lernmittel sind oder für die
// Schülerbücherei angeschafft werden: Der Händler richtet danach seinen Nachlass, und die
// Rechnungen der beiden Töpfe werden getrennt geführt. Bis zum 10.09.2026 behauptete jedes
// Anschreiben „für unsere Schulbibliothek", auch für Lernmittel-Klassensätze — der Vermerk
// stand also da, nur falsch herum.
//
// EINE Quelle für Anschreiben (order_pdf.go) und Mail (bestellmail_text.go): Zwei
// Formulierungen desselben Vermerks liefen früher oder später auseinander, und der
// Händler hielte dann zwei Dokumente in der Hand, die sich widersprechen.
//
// Der Text nennt bewusst nur die Unterscheidung — keine Behörde, keinen Träger, keine
// Eigentums- oder Rechtsangabe. Der Aufdruck auf den Büchern ist eine eigene Einstellung
// (etikett_eigentumsvermerk) und gehört nicht in dieses Anschreiben.
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
		Vermerk: "Die Bücher werden im Rahmen der Lernmittelfreiheit beschafft — Sammelbestellung der Schule.",
	},
	repository.MittelSchultraeger: {
		Kurz:    "Schülerbücherei",
		Betreff: "Bestellung für die Schülerbücherei",
		Vermerk: "Die Bücher sind eine Anschaffung für die Schülerbücherei aus Mitteln des Schulträgers — keine Beschaffung im Rahmen der Lernmittelfreiheit.",
		// „Schultr" ist das Unterscheidungswort im PDF-Gate; es steht nur in diesem Vermerk.
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
