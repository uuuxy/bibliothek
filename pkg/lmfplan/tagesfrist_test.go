package lmfplan

import (
	"testing"
	"time"

	"bibliothek/pkg/schulzeit"
)

// Gezählt wird ab dem Berliner Kalendertag: Freitag, 11.09.2026, 22:30 UTC ist in Berlin schon
// Samstag, der 12.09.; plus 21 Tage ist Samstag, der 03.10., und der nächste Schultag nach dem
// Wochenende und den Herbstferien (05.10. bis 17.10.) der 19.10. In UTC gezählt käme Freitag,
// der 02.10., heraus, ein Schultag. Der Zeitpunkt ist so gewählt, dass beide Zählungen sich
// unterscheiden: Ein Tag, der in UTC wie in Berlin in die Herbstferien fiele, hielte den Test
// auch bei einer Rechnung in UTC grün.
func TestTagesfrist_ZaehltAbDemBerlinerKalendertag(t *testing.T) {
	ab := time.Date(2026, time.September, 11, 22, 30, 0, 0, time.UTC)
	got := Hessen().Tagesfrist(ab, 21)
	if want := time.Date(2026, time.October, 19, 23, 59, 59, 0, schulzeit.Zone()); !got.Equal(want) {
		t.Errorf("Frist %s, erwartet %s", got.In(schulzeit.Zone()), want)
	}
}
