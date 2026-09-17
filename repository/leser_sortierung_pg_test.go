package repository

import (
	"context"
	"testing"
)

// Die Sortierung der Leserliste am echten Postgres (Protokoll des Medienzentrums vom
// 16.09.2026, Punkt 5).
//
// Zwei Dinge prüft der Test, und das zweite ist das wichtigere:
//
//  1. Jede erlaubte Spalte ändert die Reihenfolge wirklich, in beide Richtungen.
//  2. Das Kollegium bleibt in der Leserdatei OBEN, auch wenn nach Namen sortiert wird.
//     Das ist keine Höflichkeit, sondern der Schutz gegen die Kappung bei 500 Zeilen:
//     Ohne diesen Schlüssel stünde ein Kollege bei 875 Schülern unter „Z" und wäre
//     lautlos nicht mehr in der Liste. Genau dafür gab es die Reihenfolge ursprünglich,
//     und eine Sortierfunktion, die sie aufhebt, nimmt den Schutz mit.
func TestLeserSortierung(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	// Die Namen sind so gewählt, dass Klasse und Name GEGENLÄUFIG sortieren: Wer nach
	// Klasse sortiert, bekommt eine andere Reihenfolge als wer nach Namen sortiert.
	// Sonst wäre der Test blind gegen eine Sortierung, die immer dasselbe tut.
	seedSuchSchueler(t, pool, "SORT-1", "Anna", "Zeidler") // Klasse 05F1
	seedSuchSchueler(t, pool, "SORT-2", "Bodo", "Ahrens")  // Klasse 05F1
	seedSuchSchueler(t, pool, "SORT-3", "Clara", "Maier")  // Klasse 05F1
	seedSuchLeser(t, pool, "SORT-4", "Detlef", "Bauer", "lehrkraft")

	if _, err := pool.Exec(ctx, `
		UPDATE leser SET klasse = CASE barcode_id
			WHEN 'SORT-1' THEN '05F1' WHEN 'SORT-2' THEN '07B' WHEN 'SORT-3' THEN '09G' END
		WHERE barcode_id IN ('SORT-1','SORT-2','SORT-3')`); err != nil {
		t.Fatalf("Klassen setzen: %v", err)
	}

	repo := NewStudentRepository(pool)
	namen := func(t *testing.T, sortierung SchuelerSortierung) []string {
		t.Helper()
		zeilen, err := repo.ListLeserMitStats(ctx, nil, "SORT", sortierung)
		if err != nil {
			t.Fatalf("ListLeserMitStats: %v", err)
		}
		var out []string
		for _, z := range zeilen {
			out = append(out, z.Nachname)
		}
		return out
	}
	gleich := func(t *testing.T, was string, ist, soll []string) {
		t.Helper()
		if len(ist) != len(soll) {
			t.Fatalf("%s: %v, erwartet %v", was, ist, soll)
		}
		for i := range soll {
			if ist[i] != soll[i] {
				t.Errorf("%s: %v, erwartet %v", was, ist, soll)
				return
			}
		}
	}

	// Kollegium (Bauer) steht in JEDEM Fall vorn — der Schutz gegen die Kappung, und
	// eine Regel, die der Benutzer sieht. Für die Vorgabe wird NUR das geprüft: Welche
	// Reihenfolge der Suchrang innerhalb der Gruppen wählt, ist nicht Sache dieses
	// Tests (die Zeilen hier sind über ihren Barcode gefunden, nicht über den Namen).
	for _, sortierung := range []SchuelerSortierung{
		{},
		{Spalte: SortiereName},
		{Spalte: SortiereName, Absteigend: true},
		{Spalte: SortiereKlasse},
		{Spalte: SortiereKlasse, Absteigend: true},
		{Spalte: SortiereAusgeliehen},
	} {
		if erste := namen(t, sortierung)[0]; erste != "Bauer" {
			t.Errorf("Sortierung %+v: erste Zeile %q, erwartet die Lehrkraft Bauer — "+
				"das Kollegium muss oben bleiben, sonst fällt es bei 875 Schülern aus "+
				"der gekappten Liste", sortierung, erste)
		}
	}

	gleich(t, "nach Name aufsteigend",
		namen(t, SchuelerSortierung{Spalte: SortiereName}),
		[]string{"Bauer", "Ahrens", "Maier", "Zeidler"})

	gleich(t, "nach Name absteigend",
		namen(t, SchuelerSortierung{Spalte: SortiereName, Absteigend: true}),
		[]string{"Bauer", "Zeidler", "Maier", "Ahrens"})

	// Nach Klasse: 05F1 (Zeidler), 07B (Ahrens), 09G (Maier) — gegenläufig zum Namen.
	gleich(t, "nach Klasse aufsteigend",
		namen(t, SchuelerSortierung{Spalte: SortiereKlasse}),
		[]string{"Bauer", "Zeidler", "Ahrens", "Maier"})

	gleich(t, "nach Klasse absteigend",
		namen(t, SchuelerSortierung{Spalte: SortiereKlasse, Absteigend: true}),
		[]string{"Bauer", "Maier", "Ahrens", "Zeidler"})

	// Die Spalte „Geliehene Bücher" — ohne Ausleihen sortiert sie nach dem zweiten
	// Schlüssel (Name), und genau das ist die Zusicherung: eine stabile Reihenfolge
	// statt der, die Postgres gerade liefert.
	gleich(t, "nach Ausleihzahl (alle 0 → Name)",
		namen(t, SchuelerSortierung{Spalte: SortiereAusgeliehen}),
		[]string{"Bauer", "Ahrens", "Maier", "Zeidler"})
}

// Eine unbekannte Spalte ist nicht erlaubt — die Prüfung sitzt im Typ, nicht erst im SQL.
func TestSchuelerSortierungErlaubt(t *testing.T) {
	for _, f := range []struct {
		spalte SchuelerSortierspalte
		soll   bool
	}{
		{SortiereKartei, true},
		{SortiereName, true},
		{SortiereKlasse, true},
		{SortiereAusgeliehen, true},
		{"nachname", false},
		{"s.nachname; DROP TABLE leser", false},
		{"ausgeliehen_anzahl", false},
	} {
		if ist := (SchuelerSortierung{Spalte: f.spalte}).Erlaubt(); ist != f.soll {
			t.Errorf("Erlaubt(%q) = %v, erwartet %v", f.spalte, ist, f.soll)
		}
	}
}
