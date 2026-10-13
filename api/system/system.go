// Package system trägt die Türen der Systempflege: die Einstellungen und die Mail-Einrichtung
// mit ihren Vorlagen und den Adressen der Klassenleitungen, die Selbstprüfung der
// Betriebsbereitschaft mit ihrer täglichen Warnmail, der Stand der Sicherung, das Protokoll,
// Statistik und Übersicht und die zwei Importe, über die ein Bestand ins Programm kommt.
package system

import "bibliothek/api/tuer"

// Tueren sind die Türen dieses Themas. Sie stehen auf der Basis des Servers und halten sie als
// Zeiger: Datenbank, Anmeldung, Live-Leitung und Uhr gibt es so einmal, für jedes Thema dieselben.
type Tueren struct {
	*tuer.Basis
}
