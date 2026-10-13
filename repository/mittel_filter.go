package repository

import (
	"fmt"

	"bibliothek/pkg/mitteltopf"
)

// Der Filter nach Topf für Listen und Berichte. Er kennt drei Werte, und der dritte ist der
// Grund für diese Datei: „ohne" sind die alten Bestellungen ohne eindeutigen Topf. In der
// Spalte sind sie NULL, im Filter brauchen sie einen Namen, weil ein leerer Parameter „alle"
// heißt und nicht „die ohne Zuordnung"; sonst suchte man die Bestellungen, die man berichtigen
// will, und bekäme die ganze Liste.
//
// Historie und Bericht nehmen dieselbe Übersetzung: Mit zwei Auslegungen desselben Parameters
// zeigte die Liste eine Bestellung, die im Bericht daneben fehlt.

// mittelOhneFilter ist der Filterwert für „ohne Zuordnung" (Spalte IS NULL).
const mittelOhneFilter = "ohne"

// IstMittelFilterGueltig prüft den Parameter, bevor er in eine Abfrage geht. Leer heißt alle.
func IstMittelFilterGueltig(filter string) bool {
	return filter == "" || filter == mittelOhneFilter || mitteltopf.IstGueltig(filter)
}

// MittelFilterFehler ist die Meldung an der Tür. Sie nennt die erlaubten Werte, damit niemand
// raten muss.
func MittelFilterFehler(filter string) error {
	return fmt.Errorf("unbekannter Topf %q — erlaubt sind %q, %q und %q",
		filter, mitteltopf.Land, mitteltopf.Schultraeger, mittelOhneFilter)
}

// mittelBedingung liefert die SQL-Bedingung zum Filter und den Wert des Parameters dazu. Die
// Bedingung ist leer, wenn nicht gefiltert wird; arg ist nil, wenn sie ohne Parameter auskommt
// (IS NULL).
func mittelBedingung(filter, spalte string, index int) (bedingung string, arg any) {
	switch filter {
	case "":
		return "", nil
	case mittelOhneFilter:
		return fmt.Sprintf(" AND %s IS NULL", spalte), nil
	default:
		return fmt.Sprintf(" AND %s = $%d", spalte, index), filter
	}
}

// MittelDatenwert übersetzt den Filterwert in den Wert, der in der Spalte steht:
// „ohne" ist dort NULL und wird als leerer String gelesen (coalesce).
func MittelDatenwert(filter string) string {
	if filter == mittelOhneFilter {
		return ""
	}
	return filter
}
