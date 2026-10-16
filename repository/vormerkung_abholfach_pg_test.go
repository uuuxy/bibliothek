package repository

import (
	"context"
	"testing"
)

// Eine abholbereite Vormerkung hält ein bestimmtes Exemplar im Abholfach. Verschwindet dieses
// Exemplar aus dem Umlauf, geht die Vormerkung zurück auf „wartend"; sonst
//
//   - hielte der Verfall-Lauf sie nach drei Tagen für „nicht abgeholt" und löschte sie, und das
//     Kind verlöre seinen Platz in der Warteschlange, obwohl es nichts versäumt hat;
//   - lehnte bedieneNaechstenWartenden das ausgesonderte Exemplar ab, und der Nächste würde nie
//     bedient.
//
// Die Regel liegt in der Datenbank und nicht an jeder der sieben Türen, über die ein Exemplar
// aus dem Umlauf geht (Schaden melden, Abschluss der Inventur, Status ändern, Aussondern,
// Ausbuchen, Bestandskorrektur, Löschen von Fehlbeständen). Der Test prüft sie deshalb an den
// drei Übergängen der Zeile selbst und zusätzlich an einem der Wege.
func TestAbholbereit_FaelltZurueckWennDasExemplarVerschwindet(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := legeSignaturMitExemplarenAn(t, pool, "Abholfach", 4)
	schueler := legeSchuelerAn(t, pool, "AF-1", "Mia", "7a")
	bearbeiter := legeBearbeiterAn(t, pool)
	books := NewKatalogAbfragen(pool)

	tueren := []struct {
		name string
		ex   string
		tun  func(ex string) error
	}{
		{"ausgesondert (roh)", ex[0], func(ex string) error {
			_, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET ist_ausgesondert = true, ist_ausleihbar = false,
				aussonderung_grund = 'VERLUST' WHERE id = $1`, ex)
			return err
		}},
		{"unausleihbar (roh, Defekt)", ex[1], func(ex string) error {
			_, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET ist_ausleihbar = false WHERE id = $1`, ex)
			return err
		}},
		{"gelöscht (roh)", ex[2], func(ex string) error {
			_, err := pool.Exec(ctx, `DELETE FROM buecher_exemplare WHERE id = $1`, ex)
			return err
		}},
		{"DecommissionCopy", ex[3], func(ex string) error { return books.AendereExemplarStatus(ctx, ex, false, true, "", nil, bearbeiter) }},
	}
	for _, tuer := range tueren {
		var vID string
		if err := pool.QueryRow(ctx, `INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis)
			VALUES ($1, $2, 'abholbereit', $3, now() + interval '3 days') RETURNING id`,
			titelIDVonExemplar(t, pool, tuer.ex), schueler, tuer.ex).Scan(&vID); err != nil {
			t.Fatalf("%s: Vormerkung anlegen: %v", tuer.name, err)
		}
		if err := tuer.tun(tuer.ex); err != nil {
			t.Fatalf("%s: %v", tuer.name, err)
		}
		var status string
		var bereit *string
		if err := pool.QueryRow(ctx, `SELECT status, bereitgestellt_exemplar_id::text FROM vormerkungen WHERE id = $1`, vID).
			Scan(&status, &bereit); err != nil {
			t.Fatalf("%s: %v", tuer.name, err)
		}
		if status != "wartend" || bereit != nil {
			t.Errorf("%s: Vormerkung steht auf %q mit Exemplar %v — erwartet wartend ohne Exemplar", tuer.name, status, bereit)
		}
	}
}

// Wird eine abholbereite Vormerkung gelöscht, rückt der Nächste nach, wie beim Verfall-Lauf und
// bei der Tilgung der Spuren (vormerkung_nachruecken.go). Sonst bliebe er für immer „wartend",
// und das Exemplar gälte als frei: Jeder Dritte könnte es ausleihen.
func TestVormerkungLoeschen_AbholbereitRuecktNach(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := legeSignaturMitExemplarenAn(t, pool, "Nachruecken", 1)[0]
	titel := titelIDVonExemplar(t, pool, ex)
	a := legeSchuelerAn(t, pool, "NR-A", "Anna", "7a")
	b := legeSchuelerAn(t, pool, "NR-B", "Ben", "7a")

	var va string
	if err := pool.QueryRow(ctx, `INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis, erstellt_am)
		VALUES ($1, $2, 'abholbereit', $3, now() + interval '3 days', now() - interval '2 days') RETURNING id`, titel, a, ex).Scan(&va); err != nil {
		t.Fatal(err)
	}
	var vb string
	if err := pool.QueryRow(ctx, `INSERT INTO vormerkungen (titel_id, schueler_id, status, erstellt_am)
		VALUES ($1, $2, 'wartend', now() - interval '1 day') RETURNING id`, titel, b).Scan(&vb); err != nil {
		t.Fatal(err)
	}

	if err := NewVormerkungsAbfragen(pool).Loesche(ctx, va); err != nil {
		t.Fatalf("Loesche: %v", err)
	}
	var status string
	var bereit *string
	if err := pool.QueryRow(ctx, `SELECT status, bereitgestellt_exemplar_id::text FROM vormerkungen WHERE id = $1`, vb).
		Scan(&status, &bereit); err != nil {
		t.Fatal(err)
	}
	if status != "abholbereit" || bereit == nil || *bereit != ex {
		t.Errorf("Nächster nicht bedient: Status %q, Exemplar %v, erwartet abholbereit auf %s", status, bereit, ex)
	}
}
