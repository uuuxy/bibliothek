package leser

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Eine Klasse, die mit keinem Jahrgang von 1 bis 13 beginnt („14A", eine Buchnummer im Feld
// Klasse), versetzt der Schuljahreswechsel nicht und macht ihre Schüler nicht zu Abgängern; die
// Vorschau nennt sie. Die übrigen Klassen versetzt er wie immer. Eine Zahl über 2.147.483.647
// brach den ganzen Lauf mit 500 ab.
func TestVersetzung_KlassenAusserhalbDesJahrgangsbereichs(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	testdaten.LeereBestandsdaten(t, pool)
	raeume := func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM klassen_lehrer_mapping`)
		pgtest.RaeumeAuf(t, pool, `DELETE FROM audit_logs WHERE aktion = 'SCHULJAHRESWECHSEL'`)
	}
	raeume()
	t.Cleanup(raeume)

	klassen := []string{"05F1", "13", "14A", "4711000123"}
	schueler := map[string]string{}
	for i, klasse := range klassen {
		schueler[klasse] = testdaten.LegeSchuelerAn(t, pool, fmt.Sprintf("JGB-%d", i), "Kind"+klasse, klasse)
		if _, err := pool.Exec(ctx, `INSERT INTO klassen_lehrer_mapping (klasse, lehrer_email) VALUES ($1, $2)`,
			klasse, fmt.Sprintf("leitung%d@schule.example", i)); err != nil {
			t.Fatalf("Klassenleitung %s: %v", klasse, err)
		}
	}

	ausserhalb := []string{"14A", "4711000123"}
	for _, lauf := range []struct {
		name  string
		probe bool
	}{
		{name: "Vorschau", probe: true},
		{name: "Lauf", probe: false},
	} {
		antwort := versetzungAusfuehren(t, pool, lauf.probe)
		if !slices.Equal(antwort.KlassenAusserhalb, ausserhalb) {
			t.Errorf("%s nennt %v, erwartet %v", lauf.name, antwort.KlassenAusserhalb, ausserhalb)
		}
		if antwort.Versetzt != 1 || antwort.Archiviert != 1 {
			t.Errorf("%s: %d versetzt, %d Abgänger, erwartet je 1", lauf.name, antwort.Versetzt, antwort.Archiviert)
		}
	}

	soll := map[string]struct {
		klasse    string
		abgaenger bool
	}{
		"05F1":       {"06F1", false},
		"13":         {"ABG", true},
		"14A":        {"14A", false},
		"4711000123": {"4711000123", false},
	}
	for alt, s := range soll {
		var klasse string
		var abgaenger bool
		if err := pool.QueryRow(ctx, `SELECT klasse, ist_abgaenger FROM schueler WHERE id = $1`,
			schueler[alt]).Scan(&klasse, &abgaenger); err != nil {
			t.Fatalf("Schüler aus %s lesen: %v", alt, err)
		}
		if klasse != s.klasse || abgaenger != s.abgaenger {
			t.Errorf("aus %s: Klasse %q, Abgänger %v; erwartet %q, %v", alt, klasse, abgaenger, s.klasse, s.abgaenger)
		}
	}
	for _, klasse := range ausserhalb {
		var vorhanden bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM klassen_lehrer_mapping WHERE klasse = $1)`,
			klasse).Scan(&vorhanden); err != nil {
			t.Fatalf("Klassenleitung %s lesen: %v", klasse, err)
		}
		if !vorhanden {
			t.Errorf("Die Klassenleitung von %s ist weg, obwohl die Klasse nicht versetzt wurde", klasse)
		}
	}
}
