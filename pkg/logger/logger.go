// Package logger nimmt einer Eingabe die Zeilenumbrüche, bevor sie ins Log geht: Eine Eingabe
// kann so keine eigene Zeile vortäuschen.
package logger

import "strings"

// OhneUmbruch liefert s ohne Wagenrücklauf und Zeilenvorschub.
func OhneUmbruch(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
