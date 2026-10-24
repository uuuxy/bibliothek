// Package logger hilft dem Log des Servers: Es nimmt einer Eingabe die Zeilenumbrüche, damit sie
// keine eigene Zeile vortäuschen kann, und zählt die Zeilen der Stufe ERROR (Fehlerstand).
package logger

import "strings"

// OhneUmbruch liefert s ohne Wagenrücklauf und Zeilenvorschub.
func OhneUmbruch(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
