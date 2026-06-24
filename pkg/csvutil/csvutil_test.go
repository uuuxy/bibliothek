package csvutil

import "testing"

func TestSanitizeCell(t *testing.T) {
	cases := map[string]string{
		"":                  "",
		"Harry Potter":      "Harry Potter",
		"=HYPERLINK(\"x\")": "'=HYPERLINK(\"x\")",
		"=cmd|'/c calc'!A1": "'=cmd|'/c calc'!A1",
		"+1+1":              "'+1+1",
		"-2+3":              "'-2+3",
		"@SUM(A1)":          "'@SUM(A1)",
		"\tTab":             "'\tTab",
		"\rCarriage":        "'\rCarriage",
		"978-3-16-148410-0": "978-3-16-148410-0", // beginnt mit Ziffer → kein Trigger, unverändert
		"Normaler; Titel":   "Normaler; Titel",
	}
	for in, want := range cases {
		if got := SanitizeCell(in); got != want {
			t.Errorf("SanitizeCell(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSanitizeRow(t *testing.T) {
	got := SanitizeRow([]string{"=evil", "ok", "@bad"})
	want := []string{"'=evil", "ok", "'@bad"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q want %q", i, got[i], want[i])
		}
	}
}
