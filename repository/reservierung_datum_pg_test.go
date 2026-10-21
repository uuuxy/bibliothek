package repository

import (
	"context"
	"testing"

	"bibliothek/internal/testdaten"
	"bibliothek/pkg/schulzeit"
)

// Ein Zeitpunkt aus der Datenbank wird erst in der Zone der Schule zum Datum. pgx liest ihn in
// time.Local; in einem Prozess in UTC (Server, CI) stünde eine Reservierung von 0:30 Uhr
// Berliner Zeit sonst mit dem Vortag in allen drei Listen.
func TestReservierungen_DatumIstDerTagDerSchule(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewReservierungsAbfragen(pool)

	titel := testdaten.LegeTitelMitSignaturAn(t, pool, "KSZ-Klassensatz", "KSZ", 5)
	bearbeiter := legeEigenenBearbeiterAn(t, pool, "KSZ-B")
	id, _, err := repo.LegeAn(ctx, titel, "7c", 1, nil, bearbeiter, nil)
	if err != nil {
		t.Fatalf("Reservierung: %v", err)
	}
	// Gestern um 0:30 Uhr in der Zone der Schule: in UTC noch der Tag davor, und innerhalb der 60
	// Tage, die LadeEigene erledigte Reservierungen zeigt.
	var tagDerSchule string
	if err := pool.QueryRow(ctx, `
		UPDATE klassensatz_reservierungen
		SET erstellt_am = ((now() AT TIME ZONE $2)::date - 1 + time '00:30') AT TIME ZONE $2
		WHERE id = $1
		RETURNING to_char(erstellt_am AT TIME ZONE $2, 'DD.MM.YYYY')`, id, schulzeit.ZonenName).Scan(&tagDerSchule); err != nil {
		t.Fatal(err)
	}

	offene, err := repo.LadeOffene(ctx)
	if err != nil {
		t.Fatalf("LadeOffene: %v", err)
	}
	gefunden := false
	for _, o := range offene {
		if o.TitelID == titel {
			gefunden = true
			if o.ErstelltAm != tagDerSchule {
				t.Errorf("LadeOffene: erstellt am %q, erwartet %q", o.ErstelltAm, tagDerSchule)
			}
		}
	}
	if !gefunden {
		t.Fatal("LadeOffene: die Reservierung fehlt")
	}

	// Erledigt eine Viertelstunde später, am selben Tag der Schule.
	if _, err := pool.Exec(ctx, `UPDATE klassensatz_reservierungen
		SET erledigt = true, erledigt_am = erstellt_am + interval '15 minutes' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	alle, err := repo.LadeAlle(ctx)
	if err != nil {
		t.Fatalf("LadeAlle: %v", err)
	}
	meine := nurTitel(alle, titel)
	if len(meine) != 1 || meine[0].ErstelltAm != tagDerSchule || meine[0].ErledigtAm != tagDerSchule {
		t.Errorf("LadeAlle = %+v, erwartet erstellt und erledigt am %s", meine, tagDerSchule)
	}
	eigene, err := repo.LadeEigene(ctx, bearbeiter)
	if err != nil {
		t.Fatalf("LadeEigene: %v", err)
	}
	gefunden = false
	for _, m := range eigene {
		if m.TitelID != titel {
			continue
		}
		gefunden = true
		if m.ErstelltAm != tagDerSchule || m.ErledigtAm != tagDerSchule {
			t.Errorf("LadeEigene: erstellt %q, erledigt %q, erwartet beide %s", m.ErstelltAm, m.ErledigtAm, tagDerSchule)
		}
	}
	if !gefunden {
		t.Errorf("LadeEigene: die Reservierung fehlt (%d eigene)", len(eigene))
	}
}
