package theke

import (
	"testing"
	"time"

	"bibliothek/pkg/schulzeit"
)

// Eine Frist von Hand weit hinter der längsten, die das Programm vergibt, ist ein Tippfehler. Die
// Frist eines Bandes über mehrere Jahrgänge geht durch.
func TestLeseFaelligAm_GrenzeInDerZukunft(t *testing.T) {
	for _, eingabe := range []string{"2206-11-10", "9999-12-31", "2206-11-10T10:00:00Z"} {
		if frist, err := leseFaelligAm(eingabe); err == nil {
			t.Errorf("leseFaelligAm(%q) = %s, erwartet eine Ablehnung", eingabe, frist.Format(time.DateOnly))
		}
	}
	mehrjahresband := schulzeit.Jetzt().AddDate(9, 0, 0).Format(time.DateOnly)
	if _, err := leseFaelligAm(mehrjahresband); err != nil {
		t.Errorf("leseFaelligAm(%q) = %v, erwartet die Frist", mehrjahresband, err)
	}
}
