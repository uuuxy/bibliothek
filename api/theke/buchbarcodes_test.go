package theke

import (
	"testing"
	"time"

	"bibliothek/repository"
)

// Der Stand folgt der jüngsten Änderung: Bleibt die Zahl gleich und ein Barcode wechselt, muss der
// Theken-Rechner die Liste neu holen, sonst kennt er den neuen Aufkleber im Ausfall nicht.
func TestBuchbarcodesStand_FolgtDerJuengstenAenderung(t *testing.T) {
	frueh := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	spaet := frueh.Add(time.Second)
	ohne := buchbarcodesStand(repository.BuchbarcodeStand{Anzahl: 3})
	vorher := buchbarcodesStand(repository.BuchbarcodeStand{Anzahl: 3, Juengste: &frueh})
	nachher := buchbarcodesStand(repository.BuchbarcodeStand{Anzahl: 3, Juengste: &spaet})
	if vorher == nachher {
		t.Error("zwei verschiedene Änderungszeiten ergeben denselben Stand")
	}
	if ohne == vorher {
		t.Error("mit und ohne Änderungszeit ergeben denselben Stand")
	}
}
