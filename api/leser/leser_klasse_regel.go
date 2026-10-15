package leser

import (
	"strings"

	"bibliothek/apierrors"
)

// errKlasseLehrerVerboten lehnt den historischen Spezialwert ab, mit Wegweiser.
var errKlasseLehrerVerboten = apierrors.BadRequest("Lehrkräfte werden nicht als Schüler angelegt. Bitte unter System → Benutzer & Rechte ein Konto mit der Rolle Kollegium anlegen; an der Theke findet man die Lehrkraft dann über ihren Ausweis", nil)

// pruefeKlassenname verteidigt die Trennung aus Migration 072: 'lehrer' als Klassenname
// machte eine Schülerzeile heimlich zur Lehrkraft, mit eigener Mahnschiene, still
// übersprungenem Versetzungslauf und dem LUSD-Abgleich als tickender Löschfalle. Der Wert
// ist gesperrt.
func pruefeKlassenname(klasse string) error {
	if strings.EqualFold(strings.TrimSpace(klasse), "lehrer") {
		return errKlasseLehrerVerboten
	}
	return nil
}
