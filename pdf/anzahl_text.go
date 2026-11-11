package pdf

import "fmt"

// anzahlText nennt eine Zahl mit ihrem Hauptwort in der passenden Form: „1 Exemplar",
// „2 Exemplare". Die Summenzeilen der Bestandsbücher und des Bestellberichts schrieben die
// Mehrzahl fest; auf einem Blatt mit einem einzigen Buch stand „1 Exemplare".
func anzahlText(n int, einzahl, mehrzahl string) string {
	if n == 1 {
		return "1 " + einzahl
	}
	return fmt.Sprintf("%d %s", n, mehrzahl)
}

// exemplarText ist anzahlText für das Wort, das auf jedem dieser Blätter steht.
func exemplarText(n int) string {
	return anzahlText(n, "Exemplar", "Exemplare")
}
