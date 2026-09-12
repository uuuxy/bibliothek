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
	// Traeger: wer den Topf trägt — „Land" oder „Schulträger". Steht in den Berichten
	// hinter der Kurzform („Lernmittelfreiheit (Land)"), damit dort ohne Vorwissen
	// lesbar ist, wessen Geld gemeint ist. Im Anschreiben an den Händler steht er
	// bewusst NICHT (kein Rechts- oder Regionalbezug, Peters Vorgabe).
	Traeger string
}

var mittelTexte = map[string]mittelText{
	repository.MittelLand: {
		Kurz:    "Lernmittelfreiheit",
		Traeger: "Land",
		Betreff: "Bestellung im Rahmen der Lernmittelfreiheit",
		Vermerk: "Die Bücher werden im Rahmen der Lernmittelfreiheit beschafft — Sammelbestellung der Schule.",
	},
	repository.MittelSchultraeger: {
		Kurz:    "Schülerbücherei",
		Traeger: "Schulträger",
		Betreff: "Bestellung für die Schülerbücherei",
		Vermerk: "Die Bücher sind eine Anschaffung für die Schülerbücherei aus Mitteln des Schulträgers — keine Beschaffung im Rahmen der Lernmittelfreiheit.",
		// „Schultr" ist das Unterscheidungswort im PDF-Gate; es steht nur in diesem Vermerk.
	},
}

// mittelOhneZuordnung ist die Beschriftung der Alt-Bestellungen, deren Topf sich beim
// Backfill (Migration 109) nicht eindeutig ergab. Sie werden ausgewiesen, nie geraten.
const mittelOhneZuordnung = "ohne Zuordnung"

// mittelReihenfolge ist die Reihenfolge in Berichten und Listen: Lernmittel zuerst (der
// Regelfall), dann die Schülerbücherei, zuletzt die Alt-Bestellungen ohne Zuordnung.
// Dieselbe Reihenfolge wie MITTEL_REIHENFOLGE im Warenkorb.
var mittelReihenfolge = []string{repository.MittelLand, repository.MittelSchultraeger, ""}

// mittelBeschriftung ist die Überschrift eines Topfs, wo ein Mensch sie liest:
// „Lernmittelfreiheit (Land)". Der leere Wert ist die Alt-Bestellung ohne Zuordnung; ein
// unbekannter Wert ergibt einen leeren String, damit der Aufrufer entscheiden kann.
func mittelBeschriftung(mittel string) string {
	if mittel == "" {
		return mittelOhneZuordnung
	}
	t, err := mittelTexteFuer(mittel)
	if err != nil {
		return ""
	}
	return t.Kurz + " (" + t.Traeger + ")"
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
