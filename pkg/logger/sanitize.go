// Package logger nimmt einer Eingabe die Zeilenumbrüche, bevor sie ins Protokoll geht: Eine
// Eingabe kann so keine eigene Protokollzeile vortäuschen.
package logger

import "strings"

// SanitizeLog filters out carriage return and newline characters from user input
// to prevent Log Injection (CWE-117) vulnerabilities.
func SanitizeLog(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
