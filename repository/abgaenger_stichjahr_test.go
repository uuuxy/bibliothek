package repository

import (
	"testing"
	"time"
)

// Das Stichjahr der Abgänger-Löschung ist eine Aussage über den SCHULKALENDER, nicht
// über die Uhr des Containers. Dieselbe Sekunde muss deshalb dasselbe Jahr ergeben,
// gleich in welcher Zeitzone das Programm läuft.
//
// Vorher tat sie das nicht: Die Funktion nahm das Jahr aus der lokalen Zeit und verglich
// es mit einem fest in UTC gebauten 30. Januar. Solange der Container auf UTC lief,
// stimmten beide zufällig überein — genau die Bauform, gegen die dieses Raster antritt.
//
// Der gewählte Zeitpunkt ist die Nahtstelle: 29.01. 23:30 UTC ist in Berlin bereits der
// 30.01. um 00:30. Für die Schule ist der Stichtag also erreicht.
func TestAbgaengerStichjahr_HaengtNichtAnDerContainerZeitzone(t *testing.T) {
	// Ein einziger Zeitpunkt, verschieden ausgedrückt.
	nahtstelle := time.Date(2027, time.January, 29, 23, 30, 0, 0, time.UTC)

	zonen := []string{"UTC", "Europe/Berlin", "Pacific/Midway", "Pacific/Kiritimati"}
	var erstes int
	for i, name := range zonen {
		loc, err := time.LoadLocation(name)
		if err != nil {
			t.Skipf("Zeitzone %s nicht verfügbar (tzdata fehlt): %v", name, err)
		}
		got := AbgaengerStichjahr(nahtstelle.In(loc))
		if i == 0 {
			erstes = got
			continue
		}
		if got != erstes {
			t.Errorf("Stichjahr in %s = %d, in %s = %d — dasselbe Datum, zwei Antworten",
				name, got, zonen[0], erstes)
		}
	}

	// Und die Antwort muss die des Schulkalenders sein: In Berlin ist es der 30.01.,
	// der Stichtag ist erreicht, also zählt das laufende Jahr.
	if erstes != 2027 {
		t.Errorf("Stichjahr %d, erwartet 2027 — am 30.01. (Schulzeit) ist die Karenz vorbei", erstes)
	}
}

// Und der Tag davor muss noch in der Karenz liegen, sonst prüfte der Test oben nur,
// dass irgendein festes Jahr herauskommt.
func TestAbgaengerStichjahr_VorDemStichtagGiltDasVorjahr(t *testing.T) {
	vorher := time.Date(2027, time.January, 15, 12, 0, 0, 0, time.UTC)
	if got := AbgaengerStichjahr(vorher); got != 2026 {
		t.Errorf("Stichjahr am 15.01.2027 = %d, erwartet 2026 (Karenzzeit laeuft noch)", got)
	}
	nachher := time.Date(2027, time.March, 1, 12, 0, 0, 0, time.UTC)
	if got := AbgaengerStichjahr(nachher); got != 2027 {
		t.Errorf("Stichjahr am 01.03.2027 = %d, erwartet 2027", got)
	}
}
