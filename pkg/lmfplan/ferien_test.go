package lmfplan

import (
	"testing"
	"time"
)

// Die Termine, gegen die Peters Plan 2026 aufgeht: Ferienbeginn Montag 29.06.2026, der
// Donnerstag davor ist der 25.06. — die letzte Zeile seines Excels (5G5, 4. Stunde).
// Dazu die anderen Jahre aus dem KMK-Beschluss, damit ein Zahlendreher auffällt.
func TestSommerferienHessen_DonnerstagVor(t *testing.T) {
	for jahr, soll := range map[int][2]string{
		2026: {"2026-06-29", "2026-06-25"},
		2027: {"2027-06-28", "2027-06-24"},
		2028: {"2028-07-03", "2028-06-29"},
		2029: {"2029-07-16", "2029-07-12"},
		2030: {"2030-07-22", "2030-07-18"},
	} {
		z, ok := Hessen().Sommerferien(jahr)
		if !ok {
			t.Fatalf("%d fehlt", jahr)
		}
		if von := z.Von.Format("2006-01-02"); von != soll[0] {
			t.Errorf("%d: Ferienbeginn %s, erwartet %s", jahr, von, soll[0])
		}
		if do := DonnerstagVor(z.Von).Format("2006-01-02"); do != soll[1] {
			t.Errorf("%d: Donnerstag vor den Ferien %s, erwartet %s", jahr, do, soll[1])
		}
		if DonnerstagVor(z.Von).Weekday() != time.Thursday {
			t.Errorf("%d: kein Donnerstag", jahr)
		}
	}
	if _, ok := Hessen().Sommerferien(1999); ok {
		t.Error("1999 darf nicht hinterlegt sein")
	}
	// Ein Donnerstag als Ferienbeginn: der Donnerstag DAVOR, nicht er selbst.
	if do := DonnerstagVor(tag("2026-06-25")); !do.Equal(tag("2026-06-18")) {
		t.Errorf("Donnerstag vor einem Donnerstag: %s", do.Format("2006-01-02"))
	}
}

// Ferienende Freitag 06.08.2027 → erster Schultag Montag 09.08.2027 (der Tag, den die
// E2E-Probe des Planers seit dem 05.09.2026 von Hand tippte).
func TestErsterSchultagNach(t *testing.T) {
	z, _ := Hessen().Sommerferien(2027)
	if e := ErsterSchultagNach(z); !e.Equal(tag("2027-08-09")) {
		t.Errorf("erster Schultag nach den Ferien 2027: %s", e.Format("2006-01-02"))
	}
}

// Von heute aus gesehen: Am 06.09.2026 liegen die nächsten Ferien 2027 — für beide
// Pläne. Am 01.07.2026 (in den Ferien) ist der Büchertausch schon vorbei (2027), die
// Ausgabe kommt noch (2026). Am 07.08.2026 (letzter Ferientag) gilt die Ausgabe noch
// für 2026. Nach dem Ende der Tabelle kommt das Jahr ohne ok — der Planer nennt es.
func TestNaechsteSommerferien(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatal(err)
	}
	faelle := []struct {
		heute        string
		bevorstehend bool
		jahr         int
		ok           bool
	}{
		{"2026-09-06", true, 2027, true},
		{"2026-09-06", false, 2027, true},
		{"2026-07-01", true, 2027, true},
		{"2026-07-01", false, 2026, true},
		{"2026-06-28", true, 2026, true}, // Sonntag vor dem Ferienbeginn: noch bevorstehend
		{"2026-06-29", true, 2027, true}, // Ferienbeginn: nicht mehr
		{"2026-08-07", false, 2026, true},
		{"2026-08-08", false, 2027, true},
		{"2030-08-31", true, 2031, false},
		{"2031-03-01", true, 2031, false},
	}
	for _, f := range faelle {
		heute, err := time.ParseInLocation("2006-01-02", f.heute, berlin)
		if err != nil {
			t.Fatal(err)
		}
		heute = heute.Add(23 * time.Hour) // spät am Abend in Berlin: der Kalendertag bleibt
		z, jahr, ok := Hessen().Naechste(heute, f.bevorstehend)
		if jahr != f.jahr || ok != f.ok {
			t.Errorf("heute %s bevorstehend=%v: Jahr %d ok=%v, erwartet %d/%v", f.heute, f.bevorstehend, jahr, ok, f.jahr, f.ok)
		}
		if ok && z.Von.Year() != f.jahr {
			t.Errorf("heute %s: Zeitraum %s gehört nicht zu %d", f.heute, z.Von.Format("2006-01-02"), f.jahr)
		}
	}
}

// Die Erinnerung: Die KMK beschließt die Sommerferien in Blöcken; dieser Test wird rot,
// sobald das laufende Jahr näher als zwei Jahre an das Ende der Tabelle rückt (bei
// Tabelle bis 2030 also ab Januar 2029). Dann: Beschluss unter
// kmk.org/service/ferienregelung nachlesen, sommerferienHessen ergänzen, Zahlen oben
// prüfen. Bewusst zeitabhängig — ein Wächter, der nie schlägt, wäre keiner.
func TestSommerferienHessen_Horizont(t *testing.T) {
	if letztes := Hessen().LueckenlosBis(2025); time.Now().Year()+2 > letztes {
		t.Errorf("Sommerferien Hessen sind nur bis %d hinterlegt — KMK-Beschluss nachtragen (pkg/lmfplan/ferien.go)", letztes)
	}
}
