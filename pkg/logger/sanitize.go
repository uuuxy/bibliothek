package logger

import "strings"

// SanitizeLog filters out carriage return and newline characters from user input
// to prevent Log Injection (CWE-117) vulnerabilities.
func SanitizeLog(s string) string {
	// ⚡ Bolt: Fast-path check to avoid allocations when the string is already clean.
	if strings.IndexAny(s, "\n\r") < 0 {
		return s
	}

	// ⚡ Bolt: Single-pass byte iteration to avoid multiple string allocations.
	buf := make([]byte, 0, len(s))
	for j := 0; j < len(s); j++ {
		c := s[j]
		if c != '\n' && c != '\r' {
			buf = append(buf, c)
		}
	}
	return string(buf)
}
