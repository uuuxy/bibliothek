package repository

import (
	"os"
	"regexp"
	"strconv"
	"testing"
)

// Die Grenzen der Schlagworte stehen zweimal: im Schreibpfad des Servers
// (SchlagworteJeTitelMax, SchlagwortMaxZeichen) und als Vorgaben des Eingabefelds im Browser
// (frontend/src/lib/components/ui/ChipFeld.svelte, max und maxZeichen). Das Feld sperrt,
// bevor der Server gefragt wird. Nimmt der Server 300 Wörter je Titel
// (Littera führt bis zu 282) und sperrt das Feld bei 30, bekäme ein Titel aus
// Littera im Buchformular kein Wort mehr dazu.
//
// Gelesen wird die Svelte-Datei als Text, wie in api/lernmittel/lmf_texte_paritaet_test.go. Findet der
// Test die Stelle nicht mehr, ist er rot und nicht still grün.
//
// Blind für einen Aufrufer, der dem Feld eigene Grenzen mitgibt: Er sähe nur die Vorgaben.
// Heute gibt keiner welche mit (BuchEingabefelder.svelte, TrefferPruefungSchlagworte.svelte).
func TestSchlagwortGrenzen_FeldUndServerGleich(t *testing.T) {
	quelle, err := os.ReadFile("../frontend/src/lib/components/ui/ChipFeld.svelte")
	if err != nil {
		t.Fatalf("ChipFeld.svelte lesen: %v", err)
	}
	for _, g := range []struct {
		name   string
		server int
	}{
		{"max", SchlagworteJeTitelMax},
		{"maxZeichen", SchlagwortMaxZeichen},
	} {
		treffer := regexp.MustCompile(`(?m)^\s+` + g.name + ` = (\d+),$`).FindSubmatch(quelle)
		if treffer == nil {
			t.Fatalf("Vorgabe %s in ChipFeld.svelte nicht gefunden — Form geändert? Dann diesen Test "+
				"nachziehen, sonst prüft er nichts.", g.name)
		}
		feld, err := strconv.Atoi(string(treffer[1]))
		if err != nil {
			t.Fatalf("Vorgabe %s: %v", g.name, err)
		}
		if feld != g.server {
			t.Errorf("%s: Feld %d, Server %d — das Feld sperrt, wo der Server annimmt, oder umgekehrt",
				g.name, feld, g.server)
		}
	}
}
