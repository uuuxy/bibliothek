package repository

import (
	"fmt"

	"bibliothek/pkg/mitteltopf"
)

// Der Topf-Filter für Listen und Berichte (#596, Bauplan 7.3 Schritt 4).
//
// Drei Werte, und der dritte ist der Grund für diese Datei: „ohne" sind die
// Alt-Bestellungen, deren Topf der Backfill (Migration 109) nicht eindeutig bestimmen
// konnte. In der SPALTE sind sie NULL, im Filter brauchen sie einen Namen — ein leerer
// Parameter heißt „alle", nicht „die ohne Zuordnung". Ohne diese Unterscheidung suchte
// man die Bestellungen, die man korrigieren will, und bekäme die ganze Liste.
//
// Beide Türen (Historie und Bericht) benutzen dieselbe Übersetzung. Zwei Auslegungen
// desselben Parameters wären genau die Doppelung, bei der die Liste eine Bestellung zeigt,
// die im Bericht daneben fehlt.

// MittelOhneFilter ist der Filterwert für „ohne Zuordnung" (Spalte IS NULL).
const MittelOhneFilter = "ohne"

// MittelFilterGueltig prüft den Parameter, bevor er in eine Abfrage geht. Leer = alle.
func MittelFilterGueltig(filter string) bool {
	return filter == "" || filter == MittelOhneFilter || mitteltopf.Gueltig(filter)
}

// MittelFilterFehler ist die Meldung an der Tür — sie nennt die erlaubten Werte, damit
// niemand raten muss.
func MittelFilterFehler(filter string) error {
	return fmt.Errorf("unbekannter Topf %q — erlaubt sind %q, %q und %q",
		filter, mitteltopf.Land, mitteltopf.Schultraeger, MittelOhneFilter)
}

// MittelBedingung liefert die SQL-Bedingung zum Filter und den Parameterwert dazu.
// Die Bedingung ist leer, wenn nicht gefiltert wird; arg ist nil, wenn die Bedingung
// ohne Parameter auskommt (IS NULL).
func MittelBedingung(filter, spalte string, index int) (bedingung string, arg any) {
	switch filter {
	case "":
		return "", nil
	case MittelOhneFilter:
		return fmt.Sprintf(" AND %s IS NULL", spalte), nil
	default:
		return fmt.Sprintf(" AND %s = $%d", spalte, index), filter
	}
}

// MittelDatenwert übersetzt den Filterwert in den Wert, der in der Spalte steht:
// „ohne" ist dort NULL und wird als leerer String gelesen (coalesce).
func MittelDatenwert(filter string) string {
	if filter == MittelOhneFilter {
		return ""
	}
	return filter
}
