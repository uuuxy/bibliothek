package repository

import (
	"context"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
)

// Im Abholfach liegt für einen Leser, was abholbereit ist: nicht die wartende Vormerkung, nicht
// das Buch eines anderen Lesers. Genannt werden höchstens fünf, das mit der kürzesten Abholfrist
// zuerst und eines ohne Frist zuletzt; Titel und Frist kommen je in ihrem Feld an.
func TestAbholbereiteBuecher_HoechstensFuenfNachAbholfrist(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	_, leserID, _ := geraeteAufbau(t, tx, "abholfach")
	_, andererLeser, _ := geraeteAufbau(t, tx, "abholfach-anderer")
	tag := time.Date(2026, 5, 4, 21, 59, 59, 0, time.UTC)
	merke := func(titel string, leser string, status string, frist *time.Time) {
		t.Helper()
		if _, err := tx.Exec(ctx, `WITH t AS (INSERT INTO buecher_titel (titel) VALUES ($1) RETURNING id)
			INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_bis) SELECT id, $2, $3, $4 FROM t`,
			titel, leser, status, frist); err != nil {
			t.Fatalf("Vormerkung %q anlegen: %v", titel, err)
		}
	}
	in := func(tage int) *time.Time {
		f := tag.AddDate(0, 0, tage)
		return &f
	}
	merke("Abholprobe drei Tage", leserID, "abholbereit", in(3))
	merke("Abholprobe ein Tag", leserID, "abholbereit", in(1))
	merke("Abholprobe ohne Frist", leserID, "abholbereit", nil)
	merke("Abholprobe zwei Tage", leserID, "abholbereit", in(2))
	merke("Abholprobe fünf Tage", leserID, "abholbereit", in(5))
	merke("Abholprobe vier Tage", leserID, "abholbereit", in(4))
	merke("Abholprobe wartend", leserID, "wartend", nil)
	merke("Abholprobe anderer Leser", andererLeser, "abholbereit", in(1))

	buecher, err := AbholbereiteBuecher(ctx, tx, leserID)
	if err != nil {
		t.Fatalf("AbholbereiteBuecher: %v", err)
	}
	soll := []string{"Abholprobe ein Tag", "Abholprobe zwei Tage", "Abholprobe drei Tage", "Abholprobe vier Tage", "Abholprobe fünf Tage"}
	if len(buecher) != len(soll) {
		t.Fatalf("%d Bücher genannt, erwartet %d: %+v", len(buecher), len(soll), buecher)
	}
	for i, titel := range soll {
		b := buecher[i]
		if b.Titel != titel || b.BereitgestelltBis == nil || !b.BereitgestelltBis.Equal(tag.AddDate(0, 0, i+1)) {
			t.Errorf("Platz %d: %q bis %v, erwartet %q bis %s", i+1, b.Titel, b.BereitgestelltBis, titel, tag.AddDate(0, 0, i+1))
		}
	}

	// Mit weniger als fünf Büchern steht das ohne Frist am Ende, nicht am Anfang.
	nurAnderer, err := AbholbereiteBuecher(ctx, tx, andererLeser)
	if err != nil || len(nurAnderer) != 1 || nurAnderer[0].Titel != "Abholprobe anderer Leser" {
		t.Errorf("anderer Leser: %+v, Fehler %v", nurAnderer, err)
	}
	merke("Abholprobe anderer ohne Frist", andererLeser, "abholbereit", nil)
	zwei, err := AbholbereiteBuecher(ctx, tx, andererLeser)
	if err != nil || len(zwei) != 2 || zwei[0].Titel != "Abholprobe anderer Leser" || zwei[1].BereitgestelltBis != nil {
		t.Errorf("Buch ohne Frist: %+v, Fehler %v; erwartet es an zweiter Stelle", zwei, err)
	}
}
