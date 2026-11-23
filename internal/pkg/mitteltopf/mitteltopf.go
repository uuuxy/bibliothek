// Package mitteltopf bündelt das Wissen über die Töpfe, aus denen die Schule Bücher bezahlt:
// welche es gibt, wie sie heißen, wer sie trägt und was auf einer Bestellung dazu steht.
//
// Lernmittel beschafft die Schule aus Landesmitteln im Rahmen der Lernmittelfreiheit, den
// Bestand der Schülerbücherei aus Mitteln des Schulträgers; eine Spende, etwa des
// Fördervereins, kommt aus keinem der beiden Töpfe. Der Händler gewährt darauf
// verschiedene Nachlässe, und die Rechnungen gehen getrennte Wege. Der Topf ist deshalb eine
// Eigenschaft der Bestellung und nicht des Titels: Der Titel (ist_lernmittel) schlägt ihn nur
// vor, und diesen Vorschlag rechnet allein der Warenkorb. Ein Vorschlag des Servers ordnete
// eine Bestellung still dem falschen Topf zu.
//
// Die Werte sind dieselben wie im CHECK bestellungen_verlauf_mittel_check und in
// frontend/src/lib/components/bestellungen/mittel.js; paritaet_test.go hält Werte und Wörter
// gegen die Oberfläche. Das Paket steht unter internal/pkg/, weil Türen, Abfragen, der Export des
// Bestands und die Littera-Übernahme dieselbe Antwort brauchen und einander nicht einbinden
// können.
package mitteltopf

import (
	"errors"
	"fmt"
)

const (
	// Land ist die Lernmittelfreiheit: Sammelbestellung, Eigentum des Landes.
	Land = "land"
	// Schultraeger ist die Schülerbücherei: Anschaffung aus Mitteln des Schulträgers.
	Schultraeger = "schultraeger"
	// Spende ist die Bestellung ohne Topf, etwa mit Geld des Fördervereins. Beim Händler, auf
	// dem Etikett und beim Ersatz gilt sie wie die Schülerbücherei (WieSchuelerbuecherei); eigen
	// sind ihr Vermerk und ihre Liste im Zugangs- und im Abgangsbuch.
	Spende = "spende"
)

// IstGueltig sagt, ob der Wert zum Vokabular gehört. Die Tür fragt vor dem Schreiben, damit ein
// unbekannter Wert als Eingabefehler endet und nicht am CHECK der Datenbank.
func IstGueltig(mittel string) bool {
	return mittel == Land || mittel == Schultraeger || mittel == Spende
}

// WieSchuelerbuecherei sagt, ob ein Topf beim Händler, auf dem Etikett und beim Ersatz wie die
// Schülerbücherei behandelt wird: dieselbe Kundennummer, die Rechnung an die Schule, derselbe
// Eigentumsvermerk und kein großes Lernmittel-Etikett. Das gilt für die Schülerbücherei selbst
// und für die Spende.
func WieSchuelerbuecherei(mittel string) bool {
	return mittel == Schultraeger || mittel == Spende
}

// ErrUngueltig meldet, dass eine Eingabe keinen Topf des Vokabulars nennt. Einen Vorgabewert
// gibt es nicht: Mit ihm landete eine Bestellung still im falschen Topf. Die Türen lehnen die
// Eingabe damit ab.
var ErrUngueltig = errors.New("mittel muss 'land' (Lernmittelfreiheit), 'schultraeger' (Schülerbücherei) oder 'spende' (Spende) sein")

// MeldungUngueltig ist der Satz der Türen zu ErrUngueltig. Er nennt jeden Wert mit seinem Wort,
// damit, wer ihn liest, sieht, was erlaubt ist.
const MeldungUngueltig = "Unbekannter Topf. Erlaubt sind „land“ für die Lernmittelfreiheit, „schultraeger“ für die Schülerbücherei und „spende“ für die Spende."

// Texte sind die Wörter eines Topfs. Anschreiben, Mail und Berichte nehmen sie von hier: Mit
// zwei Formulierungen desselben Vermerks hielte der Händler zwei Dokumente in der Hand, die
// sich widersprechen.
type Texte struct {
	// Kurz ist das eine Wort für Betreffzeile und Platzhalter {{.Mittel}}.
	Kurz string
	// Traeger nennt, wessen Geld der Topf ist. Berichte setzen ihn hinter die Kurzform, damit
	// die Zahl ohne Vorwissen lesbar ist; im Anschreiben an den Händler steht er nicht.
	Traeger string
	// Betreff ist die Betreffzeile des Anschreibens.
	Betreff string
	// Vermerk ist der ganze Satz im Anschreiben, und in der Mail, wenn ihre Vorlage den
	// Platzhalter nicht trägt. Er nennt nur die Unterscheidung, keine Behörde und kein
	// Eigentum: Der Aufdruck auf den Büchern ist eine eigene Einstellung.
	Vermerk string
}

// Auf dem Anschreiben steht das Wort „Schulträger" nur im Vermerk dieses einen Topfs; daran
// unterscheidet ein Test die zwei fertigen Briefe (internal/api/bestellung/bestellanschreiben_test.go).
var texte = map[string]Texte{
	Land: {
		Kurz:    "Lernmittelfreiheit",
		Traeger: "Land",
		Betreff: "Bestellung im Rahmen der Lernmittelfreiheit",
		Vermerk: "Die Bücher werden im Rahmen der Lernmittelfreiheit beschafft — Sammelbestellung der Schule.",
	},
	Schultraeger: {
		Kurz:    "Schülerbücherei",
		Traeger: "Schulträger",
		Betreff: "Bestellung für die Schülerbücherei",
		Vermerk: "Die Bücher sind eine Anschaffung für die Schülerbücherei aus Mitteln des Schulträgers — keine Beschaffung im Rahmen der Lernmittelfreiheit.",
	},
	// „ohne Topf" sagt der Bibliothek, was die Spende ist; der Händler liest „Spende".
	Spende: {
		Kurz:    "Spende",
		Traeger: "ohne Topf",
		Betreff: "Bestellung aus Spendenmitteln",
		Vermerk: "Die Bücher sind eine Spende — keine Beschaffung aus Mitteln des Landes oder des Schulträgers.",
	},
}

// TexteFuer liefert die Texte zum Topf. Ein unbekannter Wert ist ein Fehler und kein leerer
// Vermerk: Ein Anschreiben ohne Vermerk sagte dem Händler nicht, welchen Nachlass er gewährt.
func TexteFuer(mittel string) (Texte, error) {
	t, ok := texte[mittel]
	if !ok {
		return Texte{}, fmt.Errorf("unbekannter Mittel-Topf %q", mittel)
	}
	return t, nil
}

// Traeger nennt, wessen Geld ein Topf ist, wo ein Mensch es liest: „Land" oder „Schulträger".
// Ein leerer oder unbekannter Wert ergibt "".
func Traeger(mittel string) string {
	return texte[mittel].Traeger
}

// OhneZuordnung beschriftet Bestellungen und Exemplare, deren Topf sich nicht belegen lässt:
// Bestellungen aus der Zeit vor der Zuordnung und Bücher ohne Beleg. Sie werden ausgewiesen
// und keinem Topf zugeschlagen.
const OhneZuordnung = "ohne Zuordnung"

// Reihenfolge nennt die Töpfe, wie Berichte und Listen sie ordnen: Lernmittel zuerst (der
// Regelfall), dann die Schülerbücherei und die Spende, zuletzt der leere Wert für „ohne
// Zuordnung". Dieselbe Reihenfolge wie MITTEL_REIHENFOLGE im Warenkorb.
func Reihenfolge() []string {
	return []string{Land, Schultraeger, Spende, ""}
}

// Beschriftung ist die Überschrift eines Topfs, wo ein Mensch sie liest: „Lernmittelfreiheit ·
// Land", ohne Klammer nach M3 (Grammar and punctuation, Parentheses). Der leere Wert heißt „ohne
// Zuordnung"; ein unbekannter Wert ergibt "", damit der Aufrufer entscheidet.
func Beschriftung(mittel string) string {
	if mittel == "" {
		return OhneZuordnung
	}
	t, err := TexteFuer(mittel)
	if err != nil {
		return ""
	}
	return t.Kurz + " · " + t.Traeger
}

// HatGrossesLernmittelEtikett sagt, ob zu einer Bestellung dieses Topfs das große
// Lernmittel-Etikett („Eigentum des Landes") gehört. Die Bestätigungsseite des Lieferanten,
// die Tür dahinter und der Mailanhang fragen dieselbe Regel: Ein ausgeblendeter Knopf allein
// ließe die Adresse offen. Ein Buch der Schülerbücherei oder eine Spende ist kein Lernmittel;
// eine Bestellung ohne Zuordnung behält beide Größen, weil ihr Topf nicht geraten wird.
func HatGrossesLernmittelEtikett(mittel string) bool {
	return !WieSchuelerbuecherei(mittel)
}
