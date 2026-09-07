package api

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/repository"
)

// Paar-Gate: Die Go-Rechnung des Abgangsjahres (abschlussJahrgang) und das SQL-Prädikat
// „ist Abschlussklasse" (repository.AbschlussklasseSQL) sind Zwillinge derselben Regel.
// Verglichen wird nicht eine Stichprobe, sondern der Formenraum: Jahrgang 1–13 × Zweig
// (keiner, F, G, H, R, T, plus Kleinschreibung und Leerzeichen) × Zug. Zwilling mit
// Vollprobe (sweeps.md) — der Suchnorm-Zwilling lag bei einer Stichprobe 110 Zeichen
// daneben, ohne dass es jemand sah.
func TestAbgaengerJahr_GoUndSQLSindEineRegel(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	var formen []string
	for jg := 1; jg <= 13; jg++ {
		for _, zweig := range []string{"", "F", "G", "H", "R", "T", "f", "h", " H"} {
			for _, zug := range []string{"", "1", "2"} {
				formen = append(formen, fmt.Sprintf("%02d%s%s", jg, zweig, zug), fmt.Sprintf("%d%s%s", jg, zweig, zug))
			}
		}
	}
	formen = append(formen, "ET", "E1", "12T", "13T")

	gesehen := 0
	for _, klasse := range formen {
		jahrgang, abschluss, ok := abschlussJahrgang(klasse)
		if !ok {
			t.Errorf("%q: Go liest keinen Jahrgang", klasse)
			continue
		}
		var sqlAbschluss bool
		if err := pool.QueryRow(ctx, "SELECT "+repository.AbschlussklasseSQL("$1::text"), klasse).Scan(&sqlAbschluss); err != nil {
			t.Fatalf("%q: SQL: %v", klasse, err)
		}
		goAbschluss := jahrgang >= abschluss
		// Die SQL-Regel kennt Klassen ohne führende Ziffer nicht als Abschlussklasse —
		// „ET“ ist Jahrgang 11 und in beiden Welten keine.
		if goAbschluss != sqlAbschluss {
			t.Errorf("%q: Go sagt Abschlussklasse=%v (Jahrgang %d, Ende %d), SQL sagt %v", klasse, goAbschluss, jahrgang, abschluss, sqlAbschluss)
		}
		gesehen++
	}
	if gesehen < 200 {
		t.Fatalf("nur %d Formen verglichen — die Probe läuft ins Leere", gesehen)
	}
}

// Das Anzeigejahr selbst, an festen Daten: 10G geht nicht dieses Jahr ab.
func TestAbgaengerJahrAm(t *testing.T) {
	sept := time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC) // Schuljahr 2026/27
	mai := time.Date(2027, time.May, 3, 12, 0, 0, 0, time.UTC)        // dasselbe Schuljahr
	faelle := []struct {
		klasse string
		jetzt  time.Time
		want   int
	}{
		{"10G1", sept, 2030}, // G läuft bis 13: 2027 + 3
		{"10G1", mai, 2030},
		{"10R1", sept, 2027},
		{"09H1", sept, 2027},
		{"10H1", sept, 2027}, // freiwilliges 10. Hauptschuljahr: Ende erreicht
		{"05F1", sept, 2035},
		{"ET", sept, 2029},
		{"13T", sept, 2027},
		{"ABG", sept, 2031}, // kein Jahrgang lesbar: Rückfall Jahr+5
	}
	for _, f := range faelle {
		if got := abgaengerJahrAm(f.klasse, f.jetzt); got != f.want {
			t.Errorf("%s am %s = %d, want %d", f.klasse, f.jetzt.Format("2006-01"), got, f.want)
		}
	}
}
