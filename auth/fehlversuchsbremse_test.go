package auth

import (
	"fmt"
	"testing"
	"time"
)

// Abgelaufene Fehlversuche fallen heraus, sobald die Liste groß wird. Sonst wüchse sie mit jeder
// durchprobierten Adresse ohne Grenze.
func TestFehlversuchsBremse_RaeumtAbgelaufenes(t *testing.T) {
	b := newFehlversuchsBremse(3, time.Minute)
	vorbei := time.Now().Add(-time.Hour)
	for i := 0; i <= 2000; i++ {
		b.eintraege[fmt.Sprintf("alt-%d", i)] = &fehlversuche{anzahl: 1, fensterEnde: vorbei}
	}

	// Der erste Fehlversuch beginnt ein Fenster, erst der zweite zählt und räumt.
	b.merkeFehlversuch("frisch")
	b.merkeFehlversuch("frisch")

	if len(b.eintraege) != 1 {
		t.Errorf("%d Einträge, erwartet nur den frischen", len(b.eintraege))
	}
}
