package inventur

import (
	"regexp"
	"strconv"
	"strings"
)

var gradeFromTitlePattern = regexp.MustCompile(`(?i)(?:^|[^0-9])(1[0-3]|[1-9])(?:[^0-9]|$)`)

type subjectRule struct {
	keyword string
	subject string
}

var defaultSubjectRules = []subjectRule{
	{keyword: "deutsch", subject: "Deutsch"},
	{keyword: "englisch", subject: "Englisch"},
	{keyword: "mathe", subject: "Mathematik"},
	{keyword: "mathematik", subject: "Mathematik"},
	{keyword: "physik", subject: "Physik"},
	{keyword: "chemie", subject: "Chemie"},
	{keyword: "biologie", subject: "Biologie"},
	{keyword: "geschichte", subject: "Geschichte"},
	{keyword: "geographie", subject: "Geographie"},
	{keyword: "politik", subject: "Politik"},
	{keyword: "informatik", subject: "Informatik"},
	{keyword: "kunst", subject: "Kunst"},
	{keyword: "musik", subject: "Musik"},
	{keyword: "religion", subject: "Religion"},
	{keyword: "latein", subject: "Latein"},
	{keyword: "französisch", subject: "Französisch"},
	{keyword: "franzoesisch", subject: "Französisch"},
	{keyword: "natur und technik", subject: "Naturwissenschaften"},
}

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

func inferSubjectFromTitle(title string) string {
	lower := strings.ToLower(strings.TrimSpace(title))

	for _, rule := range defaultSubjectRules {
		if strings.Contains(lower, rule.keyword) {
			return rule.subject
		}
	}

	return ""
}

func mapHeaderToField(header string) string {
	name := strings.ToLower(strings.TrimSpace(header))

	// Fast string cleaning instead of multiple ReplaceAll calls
	var b strings.Builder
	b.Grow(len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c != ' ' && c != '_' && c != '-' && c != '/' {
			b.WriteByte(c)
		}
	}
	name = b.String()

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
