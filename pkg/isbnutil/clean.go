package isbnutil

import "strings"

// CleanISBN removes hyphens and spaces from an ISBN string.
// ⚡ Bolt: High-performance string cleaning using a single pass to avoid multiple strings.ReplaceAll allocations.
func CleanISBN(isbn string) string {
	if !strings.ContainsAny(isbn, "- ") {
		return isbn
	}

	b := make([]byte, 0, len(isbn))
	for i := 0; i < len(isbn); i++ {
		if isbn[i] != '-' && isbn[i] != ' ' {
			b = append(b, isbn[i])
		}
	}
	return string(b)
}
