package littera

import (
	"context"
	"strings"
	"testing"
)

// Littera führt Titel mit zwei Leerzeichen in Folge und mit geschütztem Leerzeichen (in der
// Sicherung von 2010: 103 und 11 von 10.732 Titeln). Die Übernahme liest den Wortlaut, wie
// er steht; die Datenbank speichert ihn mit einem Leerzeichen zwischen den Wörtern
// (Migration 160), an dieser Tür wie an jeder anderen. Der Weg hier: von der Zeile des
// Exports bis in die Tabelle.
func TestTitelKommtOhneLeerraumInFolge(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, _ := testSchreiber(t, pool, nil)
	ctx := context.Background()
	geschuetzt := string(rune(0x00A0))

	gelesen, err := LeseTitel(strings.NewReader(
		"Buchungsnummer,Haupttitel,Untertitel,Verfasserangabe,Verlag,Medienart\n" +
			`1,"¬La¬  Peste","Roman  in fünf Teilen","Camus,  Albert",1,1` + "\n" +
			`2,"Mumienherz. ¬Die¬ Rückkehr des Seth /` + geschuetzt + `1","","",1,1` + "\n"))
	if err != nil {
		t.Fatalf("Titel lesen: %v", err)
	}
	if len(gelesen) != 2 || gelesen[0].Haupttitel != "La  Peste" {
		t.Fatalf("gelesen: %+v — die Übernahme liest den Wortlaut, wie er in Littera steht", gelesen)
	}

	ab := bestand(gelesen...)
	ab.Verlage["1"] = "Rowohlt  Taschenbuch"
	if _, err := s.SchreibeBestand(ctx, ab); err != nil {
		t.Fatalf("SchreibeBestand: %v", err)
	}

	type zeile struct{ titel, untertitel, autor, verlag string }
	lies := func(litteraID string) zeile {
		t.Helper()
		var z zeile
		if err := pool.QueryRow(ctx, `
			SELECT titel, coalesce(untertitel, ''), coalesce(autor, ''), coalesce(verlag, '')
			FROM buecher_titel WHERE erweiterte_eigenschaften->>'littera_id' = $1`, litteraID).
			Scan(&z.titel, &z.untertitel, &z.autor, &z.verlag); err != nil {
			t.Fatalf("Titel %s lesen: %v", litteraID, err)
		}
		return z
	}
	if got, want := lies("1"), (zeile{"La Peste", "Roman in fünf Teilen", "Camus, Albert", "Rowohlt Taschenbuch"}); got != want {
		t.Errorf("Titel 1:\n got %+q\nwant %+q", got, want)
	}
	if got := lies("2").titel; got != "Mumienherz. Die Rückkehr des Seth / 1" {
		t.Errorf("Titel 2: %+q", got)
	}
}
