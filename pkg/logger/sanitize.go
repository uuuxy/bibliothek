package logger

import "strings"

// SanitizeLog filters out carriage return and newline characters from user input
// to prevent Log Injection (CWE-117) vulnerabilities.
// ⚡ Bolt: Removed unnecessary strings.ReplaceAll allocations for performance.
func SanitizeLog(s string) string {
	if !strings.ContainsAny(s, "\n\r") {
		return s
	}

	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\n' && s[i] != '\r' {
			b = append(b, s[i])
		}
	}
	return string(b)
}
