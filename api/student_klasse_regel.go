package api

import (
	"errors"
	"strings"
)

// errKlasseLehrerVerboten lehnt den historischen Spezialwert ab, mit Wegweiser.
//
//nolint:staticcheck // ST1005: Endnutzer-Meldung, bewusst großgeschrieben
var errKlasseLehrerVerboten = errors.New("Lehrkräfte werden nicht als Schüler angelegt. Bitte unter System → Benutzerverwaltung ein Konto (Rolle Kollegium) anlegen — die Ausleihe läuft dann über den Handapparat-Weg")

// pruefeKlassenname verteidigt die Trennung aus Migration 072 (Befund F4):
// 'lehrer' als Klassenname machte eine Schülerzeile heimlich zur Lehrkraft —
// mit eigener Mahnschiene, still übersprungenem Versetzungslauf und dem
// LUSD-Abgleich als tickender Löschfalle. Der Wert ist seither gesperrt.
func pruefeKlassenname(klasse string) error {
	if strings.EqualFold(strings.TrimSpace(klasse), "lehrer") {
		return errKlasseLehrerVerboten
	}
	return nil
}
