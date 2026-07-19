package service

import (
	"testing"
	"time"
)

// TestTagesEndeInSchulzeitzone deckt die EINE zentrale Frist-Normalisierung ab, über die
// alle Fristen (Bücher, Medien, Geräte, Handapparat) laufen: Ende des Berliner Kalendertags
// (23:59:59), korrekt über die UTC-/Berlin-Tagesgrenze hinweg.
func TestTagesEndeInSchulzeitzone(t *testing.T) {
	loc := schoolLocation()
	// 2026-06-10 22:30 UTC entspricht 2026-06-11 00:30 MESZ → Berliner Kalendertag ist der 11.
	in := time.Date(2026, time.June, 10, 22, 30, 0, 0, time.UTC)
	got := tagesEndeInSchulzeitzone(in)
	want := time.Date(2026, time.June, 11, 23, 59, 59, 0, loc)
	if !got.Equal(want) {
		t.Errorf("got %s, want %s", got.In(loc), want)
	}
}

// TestGeraeteRueckgabeFrist_TagesendeSchulzeitzone sichert #4 ab: Die Geräte-Leihfrist muss —
// wie die Buch-Fristen — auf das Tagesende in der Schul-Zeitzone (Europe/Berlin) fallen, nicht
// auf die sekundengenaue Server-Zeit (UTC). Ein um 10:00 MESZ geliehenes Gerät war sonst
// 08:00 UTC "am 14. Tag" fällig — verwirrend für Mahnläufe und die Fälligkeitsanzeige.
func TestGeraeteRueckgabeFrist_TagesendeSchulzeitzone(t *testing.T) {
	loc := schoolLocation()

	// Ausleihe am 10.06.2026 um 10:00 Ortszeit (Sommerzeit, MESZ = UTC+2).
	ausgeliehen := time.Date(2026, time.June, 10, 10, 0, 0, 0, loc)

	frist := geraeteRueckgabeFrist(ausgeliehen)

	// 14 Tage später, auf 23:59:59 Ortszeit normalisiert.
	if got, want := frist.In(loc), time.Date(2026, time.June, 24, 23, 59, 59, 0, loc); !got.Equal(want) {
		t.Errorf("Frist = %s, erwartet %s", got, want)
	}

	// Uhrzeit-Komponente muss in der Schul-Zeitzone das Tagesende sein.
	h, m, sec := frist.In(loc).Clock()
	if h != 23 || m != 59 || sec != 59 {
		t.Errorf("Uhrzeit der Frist = %02d:%02d:%02d, erwartet 23:59:59 (Ortszeit)", h, m, sec)
	}
}
