package uebernahme

import "fmt"

// Die Spaltenbreiten aus schema.sql. Ist ein Freitextfeld länger, lehnt Postgres den ganzen
// Datensatz mit SQLSTATE 22001 ab und nennt die Spalte nicht, weil der Fehler bei der
// Umwandlung des Typs entsteht. Im Altbestand aus Littera sind überlange Titel und Angaben zum
// Verlag häufig.
const (
	MaxFreitext  = 255 // titel, untertitel, autor, verlag, signatur, ort
	MaxMedientyp = 100 // medientyp, vorname, nachname
	MaxKlasse    = 20  // schueler.klasse
	MaxBarcode   = 100 // barcode_id
	MaxAuflage   = 50  // auflage
)

// Kuerzung nennt, was Kuerze braucht: das Feld mit seinem Wert, die Breite der Spalte und
// womit die Zeile im Protokoll steht.
type Kuerzung struct {
	Protokoll *Protokoll
	QuellID   string
	Kennung   string
	Feld      string
	Wert      string
	Max       int
}

// Kuerze bringt ein Freitextfeld auf die Spaltenbreite und meldet jede Kürzung.
//
// Gekürzt wird und nicht abgelehnt: Der Datensatz kommt an, und die Kürzung steht mit dem
// ganzen Wert als Warnung im Protokoll und lässt sich nacharbeiten. Ein Buch wegen zwölf
// Zeichen zu viel im Untertitel nicht zu übernehmen, wäre der schlechtere Tausch.
//
// Gezählt wird in Zeichen und nicht in Bytes: Postgres zählt bei VARCHAR Zeichen, und ein
// Schnitt nach Bytes zerlegte einen Umlaut in ungültiges UTF-8.
func Kuerze(k Kuerzung) string {
	runen := []rune(k.Wert)
	if len(runen) <= k.Max {
		return k.Wert
	}
	k.Protokoll.Warnung(k.QuellID, k.Kennung, fmt.Sprintf(
		"%s war %d Zeichen lang (Spaltenbreite %d) und wurde gekürzt; Original: %q",
		k.Feld, len(runen), k.Max, k.Wert))
	return string(runen[:k.Max])
}

// KuerzeNullbar arbeitet wie Kuerze und liefert nil statt einer leeren Zeichenkette, wie pgx
// es für eine Spalte braucht, die NULL sein darf.
func KuerzeNullbar(k Kuerzung) *string {
	if k.Wert == "" {
		return nil
	}
	gekuerzt := Kuerze(k)
	return &gekuerzt
}

// Nullbar liefert nil für eine leere Zeichenkette, sonst einen Zeiger auf sie.
func Nullbar(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
