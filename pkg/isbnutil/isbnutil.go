package isbnutil

// Clean removes hyphens and spaces from an ISBN string.
// It iterates through the string in a single pass to avoid unnecessary allocations,
// which is a significant performance improvement when processing many ISBNs (e.g., bulk import).
func Clean(s string) string {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '-' && s[i] != ' ' {
			n++
		}
	}
	if n == len(s) {
		return s // no allocations if already clean
	}

	b := make([]byte, n)
	j := 0
	for i := 0; i < len(s); i++ {
		if c := s[i]; c != '-' && c != ' ' {
			b[j] = c
			j++
		}
	}
	return string(b)
}
