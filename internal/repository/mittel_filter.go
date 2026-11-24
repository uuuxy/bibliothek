package repository

import (
	"fmt"

	"bibliothek/internal/pkg/mitteltopf"
)

// Der Filter nach Topf für Listen und Berichte: leer heißt alle, sonst einer der Töpfe. Einen
// Wert für Bestellungen ohne Topf gibt es nicht: Jede Bestellung wird mit Topf angelegt.
//
// Historie und Bericht nehmen dieselbe Übersetzung: Mit zwei Auslegungen desselben Parameters
// zeigte die Liste eine Bestellung, die im Bericht daneben fehlt.

// IstMittelFilterGueltig prüft den Parameter, bevor er in eine Abfrage geht. Leer heißt alle.
func IstMittelFilterGueltig(filter string) bool {
	return filter == "" || mitteltopf.IstGueltig(filter)
}

// mittelBedingung liefert die SQL-Bedingung zum Filter und den Wert des Parameters dazu. Die
// Bedingung ist leer, wenn nicht gefiltert wird.
func mittelBedingung(filter, spalte string, index int) (bedingung string, arg any) {
	if filter == "" {
		return "", nil
	}
	return fmt.Sprintf(" AND %s = $%d", spalte, index), filter
}
