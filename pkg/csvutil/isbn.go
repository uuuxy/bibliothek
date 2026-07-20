package csvutil

// CleanISBN is a highly optimized helper that removes hyphens and spaces
// from an ISBN string in a single pass without intermediate allocations.
func CleanISBN(s string) string {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '-' && s[i] != ' ' {
			n++
		}
	}
	if n == len(s) {
		return s
	}
	b := make([]byte, n)
	j := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '-' && s[i] != ' ' {
			b[j] = s[i]
			j++
		}
	}
	return string(b)
}
