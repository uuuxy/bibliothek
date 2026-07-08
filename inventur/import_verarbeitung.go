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

type acNode struct {
	children [256]*acNode
	fail     *acNode
	ruleIdx  int // 1-based index (0 means no match)
}

func buildAC(rules []subjectRule) *acNode {
	root := &acNode{}
	for idx, r := range rules {
		curr := root
		for i := 0; i < len(r.keyword); i++ {
			c := r.keyword[i]
			if curr.children[c] == nil {
				curr.children[c] = &acNode{}
			}
			curr = curr.children[c]
		}
		// Set ruleIdx to the highest precedence (lowest index)
		if curr.ruleIdx == 0 || idx+1 < curr.ruleIdx {
			curr.ruleIdx = idx + 1
		}
	}

	queue := make([]*acNode, 0, 256)
	for i := 0; i < 256; i++ {
		if root.children[i] != nil {
			root.children[i].fail = root
			queue = append(queue, root.children[i])
		} else {
			root.children[i] = root
		}
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for i := 0; i < 256; i++ {
			if curr.children[i] != nil {
				queue = append(queue, curr.children[i])
				failNode := curr.fail
				curr.children[i].fail = failNode.children[i]

				// Inherit rule from failNode if it has higher precedence
				failIdx := curr.children[i].fail.ruleIdx
				if failIdx > 0 {
					if curr.children[i].ruleIdx == 0 || failIdx < curr.children[i].ruleIdx {
						curr.children[i].ruleIdx = failIdx
					}
				}
			} else {
				curr.children[i] = curr.fail.children[i]
			}
		}
	}
	return root
}

var acMachine = buildAC(defaultSubjectRules)

func inferSubjectFromTitle(title string) string {
	lower := strings.ToLower(strings.TrimSpace(title))
	curr := acMachine
	bestRule := -1

	for i := 0; i < len(lower); i++ {
		curr = curr.children[lower[i]]
		if curr.ruleIdx > 0 {
			if bestRule == -1 || curr.ruleIdx-1 < bestRule {
				bestRule = curr.ruleIdx - 1
			}
		}
	}

	if bestRule != -1 {
		return defaultSubjectRules[bestRule].subject
	}
	return ""
}

func mapHeaderToField(header string) string {
	name := strings.ToLower(strings.TrimSpace(header))
	name = strings.ReplaceAll(name, " ", "")
	name = strings.ReplaceAll(name, "_", "")
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "/", "")

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
