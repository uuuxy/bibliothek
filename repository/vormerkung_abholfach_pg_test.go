package repository

import (
	"context"
	"testing"
)

// Eine abholbereite Vormerkung hält ein bestimmtes Exemplar im Abholfach. Verschwindet
// dieses Exemplar aus dem Umlauf, muss die Vormerkung zurück auf „wartend" — sonst:
//   - hält der Verfall-Lauf sie nach drei Tagen für „nicht abgeholt" und löscht sie; das
//     Kind verliert seinen Platz in der Warteschlange, obwohl es nichts versäumt hat;
//   - lehnt bedieneNaechstenWartenden das ausgesonderte Exemplar ab, der Nächste wird nie
//     bedient.
//
// Bis zum 10.09.2026 tat das nur EINE von sieben Türen (ReportDamage). Inventur-Abschluss
// (das Abholfach steht nicht im Regal und wird nicht gescannt → VERLUST), Status-Editor,
// Aussondern, Ausbuchen, Bestandskorrektur, Defekt-Markierung und das Löschen von
// Fehlbeständen ließen sie stehen (Bestands-Durchgang, Bugklasse „Ausgang ohne
// Folgeschritt"). Die Regel liegt seitdem in der Datenbank (Migration 112): Dieser Test
// prüft sie deshalb an den drei ROHEN Übergängen, nicht an jeder Tür — und zusätzlich an
// einem echten Pfad.
func TestAbholbereit_FaelltZurueckWennDasExemplarVerschwindet(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := seedSignaturMitExemplaren(t, pool, "Abholfach", 4)
	schueler := seedSchueler(t, pool, "AF-1", "Mia", "7a")
	books := NewBookRepository(pool)

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
		{"DecommissionCopy", ex[3], func(ex string) error { return books.DecommissionCopy(ctx, ex) }},
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
			t.Errorf("%s: Vormerkung steht auf %q mit Exemplar %v — want wartend ohne Exemplar", tuer.name, status, bereit)
		}
	}
}

// Wird eine abholbereite Vormerkung gelöscht, rückt der Nächste nach — dieselbe
// Invariante wie beim Verfall-Lauf und bei der DSGVO-Tilgung (vormerkung_nachruecken.go).
// Bis zum 10.09.2026 ließ das manuelle Löschen (Knopf auch bei abholbereiten Einträgen)
// den Nächsten für immer „wartend", und das Exemplar galt als frei: Jeder Dritte konnte es
// ausleihen.
func TestVormerkungLoeschen_AbholbereitRuecktNach(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := seedSignaturMitExemplaren(t, pool, "Nachruecken", 1)[0]
	titel := titelIDVonExemplar(t, pool, ex)
	a := seedSchueler(t, pool, "NR-A", "Anna", "7a")
	b := seedSchueler(t, pool, "NR-B", "Ben", "7a")

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

	if err := NewVormerkungRepository(pool).Delete(ctx, va); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var status string
	var bereit *string
	if err := pool.QueryRow(ctx, `SELECT status, bereitgestellt_exemplar_id::text FROM vormerkungen WHERE id = $1`, vb).
		Scan(&status, &bereit); err != nil {
		t.Fatal(err)
	}
	if status != "abholbereit" || bereit == nil || *bereit != ex {
		t.Errorf("Nächster nicht bedient: status %q, Exemplar %v — want abholbereit auf %s", status, bereit, ex)
	}
}
