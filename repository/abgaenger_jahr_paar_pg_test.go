package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Paar-Gate: Die Go-Rechnung des Abgangsjahres (repository.AbschlussJahrgang) und das SQL-Prädikat
// „ist Abschlussklasse" (repository.AbschlussklasseSQL) sind Zwillinge derselben Regel.
// Verglichen wird nicht eine Stichprobe, sondern der Formenraum: Jahrgang 1–13 × Zweig
// (keiner, F, G, H, R, T, plus Kleinschreibung, Leerzeichen und Unicode-Leerraum wie das
// geschützte Leerzeichen) × Zug. Zwilling mit
// Vollprobe (sweeps.md) — der suchnorm-Zwilling lag bei einer Stichprobe 110 Zeichen
// daneben, ohne dass es jemand sah.
func TestAbgaengerJahr_GoUndSQLSindEineRegel(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	gesehen := 0
	for _, klasse := range klassenFormen() {
		jahrgang, abschluss, ok := repository.AbschlussJahrgang(klasse)
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

	// Was mit keinem Jahrgang von 1 bis 13 beginnt, ist in beiden Welten keine Abschlussklasse,
	// auch eine Zahl, die in keine ganze Zahl der Datenbank passt.
	for _, klasse := range []string{"0", "00a", "14", "14a", "20H1", "4711000123", "99999999999999999999"} {
		if _, _, ok := repository.AbschlussJahrgang(klasse); ok {
			t.Errorf("%q: Go liest einen Jahrgang", klasse)
		}
		var sqlAbschluss bool
		if err := pool.QueryRow(ctx, "SELECT "+repository.AbschlussklasseSQL("$1::text"), klasse).Scan(&sqlAbschluss); err != nil {
			t.Fatalf("%q: SQL: %v", klasse, err)
		}
		if sqlAbschluss {
			t.Errorf("%q: SQL sagt Abschlussklasse", klasse)
		}
	}
}

// klassenFormen ist der Formenraum der Paar-Gates: Jahrgang 1–13 × Zweig × Zug, dazu die
// Oberstufe in der Schreibweise der Schule.
func klassenFormen() []string {
	var formen []string
	for jg := 1; jg <= 13; jg++ {
		for _, zweig := range []string{"", "F", "G", "H", "R", "T", "f", "h", " H", "\u00a0H", "\u202fR", "\u3000h"} {
			for _, zug := range []string{"", "1", "2"} {
				// Ohne Zweig wüchse der Zug in die Zahl hinein: „051" ist Jahrgang 51, keine 5.
				if zweig == "" && zug != "" {
					continue
				}
				formen = append(formen, fmt.Sprintf("%02d%s%s", jg, zweig, zug), fmt.Sprintf("%d%s%s", jg, zweig, zug))
			}
		}
	}
	return append(formen, "ET", "E1", "12T", "13T")
}

// Paar-Gate zu repository.JahrgangSQL: Der Jahrgang aus SQL ist der aus AbschlussJahrgang, über
// den Formenraum, die Einführungsphase in mehreren Schreibweisen und Klassen ohne Jahrgang, die
// in SQL NULL ergeben.
func TestJahrgang_GoUndSQLSindEineRegel(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	formen := append(klassenFormen(), "ET1", "ET10", "et2", "E2",
		"", "ABG", "AUS", "Q1", "Q4", "0", "00a", "14", "14a", "20H1", "4711000123", "99999999999999999999")
	if len(formen) < 200 {
		t.Fatalf("nur %d Formen — die Probe läuft ins Leere", len(formen))
	}
	for _, klasse := range formen {
		jahrgang, _, ok := repository.AbschlussJahrgang(klasse)
		var sqlJahrgang *int
		if err := pool.QueryRow(ctx, "SELECT "+repository.JahrgangSQL("$1::text"), klasse).Scan(&sqlJahrgang); err != nil {
			t.Fatalf("%q: SQL: %v", klasse, err)
		}
		switch {
		case !ok && sqlJahrgang != nil:
			t.Errorf("%q: Go liest keinen Jahrgang, SQL liest %d", klasse, *sqlJahrgang)
		case ok && sqlJahrgang == nil:
			t.Errorf("%q: Go liest Jahrgang %d, SQL keinen", klasse, jahrgang)
		case ok && *sqlJahrgang != jahrgang:
			t.Errorf("%q: Go liest Jahrgang %d, SQL %d", klasse, jahrgang, *sqlJahrgang)
		}
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
		if got := repository.AbgaengerJahrAm(f.klasse, f.jetzt); got != f.want {
			t.Errorf("%s am %s = %d, erwartet %d", f.klasse, f.jetzt.Format("2006-01"), got, f.want)
		}
	}
}
