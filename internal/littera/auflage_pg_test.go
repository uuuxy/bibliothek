package littera

import (
	"context"
	"strings"
	"testing"

	"bibliothek/repository"
)

// TestAuflageKommtMit geht den Weg von der Zeile des Exports bis in die Tabelle und liest mit
// dem Code der Anwendung zurück. Ohne Angabe bleibt die Spalte NULL; eine Angabe über der
// Spaltenbreite kostet nicht den Titel samt Exemplaren.
func TestAuflageKommtMit(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, protokoll := testSchreiber(t, pool, nil)
	ctx := context.Background()

	gelesen, err := LeseTitel(strings.NewReader(
		"Buchungsnummer,Haupttitel,Auflage,Verlag,Medienart\n" +
			`1,"Faust","13. völlig überarb.  Aufl. ",1,1` + "\n" +
			`2,"Die Welle","",1,1` + "\n" +
			`3,"Gesetze","3., vollst. überarb. und erg. Aufl. / hrsg. und bearb. von der Redaktion",1,1` + "\n"))
	if err != nil {
		t.Fatalf("Titel lesen: %v", err)
	}

	bericht, err := s.SchreibeBestand(ctx, bestand(gelesen...))
	if err != nil {
		t.Fatalf("SchreibeBestand: %v", err)
	}
	if bericht.Titel != 3 || bericht.Exemplare != 3 || bericht.Uebersprungen != 0 {
		t.Fatalf("geschrieben %d Titel und %d Exemplare, übersprungen %d — erwartet 3, 3 und 0",
			bericht.Titel, bericht.Exemplare, bericht.Uebersprungen)
	}

	lies := func(litteraID string) *string {
		t.Helper()
		var auflage *string
		if err := pool.QueryRow(ctx, `
			SELECT auflage FROM buecher_titel WHERE erweiterte_eigenschaften->>'littera_id' = $1`, litteraID).
			Scan(&auflage); err != nil {
			t.Fatalf("Titel %s lesen: %v", litteraID, err)
		}
		return auflage
	}
	if got := lies("1"); got == nil || *got != "13. völlig überarb. Aufl." {
		t.Errorf("Titel 1: auflage = %v", zeige(got))
	}
	if got := lies("2"); got != nil {
		t.Errorf("Titel 2: auflage = %q, erwartet NULL", *got)
	}
	if got := lies("3"); got == nil || *got != "3., vollst. überarb. und erg. Aufl. / hrsg. und be" {
		t.Errorf("Titel 3: auflage = %v", zeige(got))
	}
	if log := protokoll(); !strings.Contains(log, "auflage") || !strings.Contains(log, "gekürzt") {
		t.Errorf("die Kürzung fehlt im Protokoll:\n%s", log)
	}

	treffer, err := repository.NewBookRepository(pool).SearchTitles(ctx, "Faust")
	if err != nil {
		t.Fatalf("Suche der Anwendung: %v", err)
	}
	if len(treffer) != 1 || treffer[0].Auflage != "13. völlig überarb. Aufl." {
		t.Errorf("die Suche der Anwendung liefert die Auflage nicht: %+v", treffer)
	}
}
