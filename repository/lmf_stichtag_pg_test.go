package repository

import (
	"context"
	"fmt"
	"testing"
)

// Zwilling mit Vollprobe: lmfMittelstufe entscheidet beim Ausleihen und auf dem Rückweg des Plans
// über den Stichtag, lmfMittelstufeSQL beim Büchertausch, den die Datenbank schreibt. Verglichen
// werden die Formen der Klassen dieser Schule mit führender Null, Zweig und Zug, die der
// Oberstufe und solche ohne Jahrgang, je für bis zu drei Schuljahre voraus.
func TestLMFStichtag_MittelstufeInGoUndSQLGleich(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	formen := []string{"", "ABG", "DaZ1", "ET", "E1", "e2", "Q3", "13", "14a", "4711000123"}
	for jahrgang := 1; jahrgang <= 13; jahrgang++ {
		for _, rest := range []string{"", "G1", "H2", "R1", "F3", "T", " H1", " R2"} {
			formen = append(formen, fmt.Sprintf("%d%s", jahrgang, rest), fmt.Sprintf("%02d%s", jahrgang, rest))
		}
	}
	gesehen := 0
	for _, klasse := range formen {
		for jahre := 0; jahre <= 3; jahre++ {
			var inSQL bool
			if err := pool.QueryRow(ctx, "SELECT "+lmfMittelstufeSQL("$1::text", jahre), klasse).Scan(&inSQL); err != nil {
				t.Fatalf("%q (+%d): SQL: %v", klasse, jahre, err)
			}
			if inGo := lmfMittelstufe(klasse, jahre); inGo != inSQL {
				t.Errorf("%q (+%d Schuljahre): Go sagt Mittelstufe=%v, SQL sagt %v", klasse, jahre, inGo, inSQL)
			}
			gesehen++
		}
	}
	if gesehen < 800 {
		t.Fatalf("nur %d Formen verglichen — die Probe läuft ins Leere", gesehen)
	}
}
