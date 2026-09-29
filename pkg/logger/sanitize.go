package logger

import "strings"

// SanitizeLog filters out carriage return and newline characters from user input
// to prevent Log Injection (CWE-117) vulnerabilities.
func SanitizeLog(s string) string {
	// ⚡ Bolt: Fast-path to avoid any allocation or processing if the string is clean.
	if !strings.ContainsAny(s, "\n\r") {
		return s
	}

	// ⚡ Bolt: Single-pass, single-allocation byte filtering. Because \n and \r are standard
	// ASCII characters (bytes 10 and 13), it's safe to process byte-by-byte without breaking UTF-8.
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if c := s[i]; c != '\n' && c != '\r' {
			b = append(b, c)
		}
	}
	return string(b)
}
