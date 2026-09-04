package uebernahme

import "fmt"

// Spaltenbreiten aus schema.sql. Läuft ein Freitextfeld darüber, lehnt Postgres den
// ganzen Datensatz mit SQLSTATE 22001 ab — und nennt dabei nicht einmal die Spalte, weil
// der Fehler bei der Typumwandlung entsteht. Bei einer Altbestandsübernahme aus Littera
// sind überlange Titel- und Verlagsangaben Alltag.
const (
	MaxFreitext  = 255 // titel, untertitel, autor, verlag, signatur, ort
	MaxMedientyp = 100 // medientyp, vorname, nachname
	MaxKlasse    = 20  // schueler.klasse
	MaxBarcode   = 100 // barcode_id
)

// Kuerzung enthält die Parameter für die Kuerze-Funktionen.
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
// Gekürzt statt abgelehnt, im selben Geist wie die ISBN-Behandlung: der Datensatz kommt
// an, die Kürzung steht mit vollem Originalwert als WARNUNG im Protokoll und lässt sich
// gezielt nacharbeiten. Ein Buch wegen zwölf überzähliger Zeichen im Untertitel gar
// nicht zu übernehmen wäre der schlechtere Tausch.
//
// Gekürzt wird auf max ZEICHEN, nicht auf max Bytes: Postgres zählt bei VARCHAR Zeichen,
// und ein byteweiser Schnitt zerlegte deutsche Umlaute in ungültiges UTF-8.
func Kuerze(k Kuerzung) string {
	r := []rune(k.Wert)
	if len(r) <= k.Max {
		return k.Wert
	}
	k.Protokoll.Warnung(k.QuellID, k.Kennung, fmt.Sprintf(
		"%s war %d Zeichen lang (Spaltenbreite %d) und wurde gekürzt; Original: %q",
		k.Feld, len(r), k.Max, k.Wert))
	return string(r[:k.Max])
}

// KuerzeNullbar arbeitet wie Kuerze, liefert aber nil statt einer leeren Zeichenkette —
// die Form, die pgx für eine nullbare Spalte braucht.
func KuerzeNullbar(k Kuerzung) *string {
	if k.Wert == "" {
		return nil
	}
	gekuerzt := Kuerze(k)
	return &gekuerzt
}

// Nullbar wandelt eine leere Zeichenkette in ein typisiertes nil.
func Nullbar(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
