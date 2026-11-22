package katalog

import (
	"strings"
	"testing"
)

// Die Zeilengrenze greift beim Lesen der CSV, nicht erst danach: 100 MB aus kurzen Zeilen stünden
// sonst als Millionen Zeilen im Speicher, bevor jemand zählt. Eine Zeile über der Grenze ist
// erlaubt, die Kopfzeile.
func TestLeseCSVZeilen_BrichtUeberDerGrenzeAb(t *testing.T) {
	if _, err := leseCSVZeilen(strings.NewReader(strings.Repeat("9783060626755;Titel\n", maxImportZeilen+1))); err != nil {
		t.Errorf("Kopfzeile und %d Zeilen: %v, erwartet kein Fehler", maxImportZeilen, err)
	}
	_, err := leseCSVZeilen(strings.NewReader(strings.Repeat("9783060626755;Titel\n", maxImportZeilen+2)))
	if err == nil || !strings.Contains(err.Error(), "höchstens") {
		t.Errorf("zwei Zeilen über der Grenze: %v, erwartet die Ablehnung mit der Grenze", err)
	}
}
