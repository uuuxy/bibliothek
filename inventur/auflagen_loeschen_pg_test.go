package inventur

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Die Massenaktion der Bestandstabelle hält dieselbe Regel wie das Löschen eines einzelnen
// Titels (repository.WerkeDerTitel, RaeumeWerkeAuf): Werden beide Auflagen eines Buchs
// gelöscht, bleibt kein Werk ohne Titel zurück; bleibt eine von zweien, ist sie wieder ihr
// eigenes Buch (Rasterdurchgang 25.09.2026, docs/OFFEN.md 4.18).
func TestDeleteBooks_HaeltDieRegelDerAuflagen(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)
	for _, sql := range []string{`DELETE FROM ausleihen`, `DELETE FROM buecher_exemplare`, `DELETE FROM buecher_titel`, `DELETE FROM werke`} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, sql := range []string{`DELETE FROM buecher_titel WHERE titel = 'Löschprobe'`,
			`DELETE FROM werke w WHERE NOT EXISTS (SELECT 1 FROM buecher_titel b WHERE b.werk_id = w.id)`} {
			if _, err := pool.Exec(context.Background(), sql); err != nil {
				t.Logf("Aufräumen: %v", err)
			}
		}
	})
	buch := func() (werk string, titel []string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `INSERT INTO werke DEFAULT VALUES RETURNING id::text`).Scan(&werk); err != nil {
			t.Fatal(err)
		}
		for _, jahr := range []int{2019, 2023} {
			var id string
			if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, erscheinungsjahr, ist_lernmittel, werk_id)
				VALUES ('Löschprobe', $1, true, $2) RETURNING id::text`, jahr, werk).Scan(&id); err != nil {
				t.Fatal(err)
			}
			titel = append(titel, id)
		}
		return werk, titel
	}
	werkBeide, beide := buch()
	werkEiner, einer := buch()

	if err := repo.DeleteBooks(ctx, append(beide, einer[0])); err != nil {
		t.Fatalf("DeleteBooks: %v", err)
	}
	var rest []string
	if err := pool.QueryRow(ctx, `SELECT coalesce(array_agg(id::text), '{}') FROM werke`).Scan(&rest); err != nil {
		t.Fatal(err)
	}
	if len(rest) != 0 {
		t.Errorf("übrige Werke %v — erwartet keins (%s ohne Titel, %s mit nur einem)", rest, werkBeide, werkEiner)
	}
	var werkID *string
	if err := pool.QueryRow(ctx, `SELECT werk_id::text FROM buecher_titel WHERE id = $1`, einer[1]).Scan(&werkID); err != nil {
		t.Fatal(err)
	}
	if werkID != nil {
		t.Errorf("die übrige Auflage hängt noch an %s — erwartet ihr eigenes Buch", *werkID)
	}
}
