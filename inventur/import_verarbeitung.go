package inventur

import (
	"regexp"
	"strconv"
	"strings"

	"bibliothek/pkg/lmf"
)

var gradeFromTitlePattern = regexp.MustCompile(`(?i)(?:^|[^0-9])(1[0-3]|[1-9])(?:[^0-9]|$)`)

func inferGradeLevelFromTitle(title string) int {
	match := gradeFromTitlePattern.FindStringSubmatch(title)
	if len(match) < 2 {
		return 0
	}
	grade, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return grade
}

// inferSubjectFromTitle rät das Fach aus dem Titel — über dieselbe Liste wie der
// ISBN-Lookup (pkg/lmf), damit beide Wege dasselbe Fach registrieren.
func inferSubjectFromTitle(title string) string {
	return lmf.FachAusText(strings.TrimSpace(title))
}

func mapHeaderToField(header string) string {
	raw := strings.ToLower(strings.TrimSpace(header))

	// ⚡ Bolt: Single pass string cleaning using strings.Builder
	// Replaces multiple strings.ReplaceAll calls to prevent unnecessary allocations
	// when cleaning headers from spaces, underscores, dashes, and slashes.
	var b strings.Builder
	b.Grow(len(raw))
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c != ' ' && c != '_' && c != '-' && c != '/' {
			b.WriteByte(c)
		}
	}
	name := b.String()

	if strings.Contains(name, "isbn") {
		return "isbn"
	}
	if strings.Contains(name, "titel") || strings.Contains(name, "title") || strings.Contains(name, "band") || strings.Contains(name, "ausgabe") {
		return "titel"
	}
	if strings.Contains(name, "autor") || strings.Contains(name, "author") {
		return "autor"
	}
	if strings.Contains(name, "fach") || strings.Contains(name, "subject") {
		return "fach"
	}
	if strings.Contains(name, "klasse") || strings.Contains(name, "stufe") || strings.Contains(name, "grade") {
		return "klasse"
	}
	if strings.Contains(name, "bestand") || strings.Contains(name, "anzahl") || strings.Contains(name, "stock") || strings.Contains(name, "menge") || strings.Contains(name, "stueck") || strings.Contains(name, "stück") {
		return "bestand"
	}

	return ""
}
