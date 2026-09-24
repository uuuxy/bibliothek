package api

import (
	"context"
	"testing"

	"bibliothek/repository"
)

// Eine Ausweisnummer, die in der Leserdatei stand, gibt der Generator nie ein zweites Mal
// aus — auch nicht, nachdem sie dort verschwunden ist (docs/OFFEN.md 5.23).
//
// Bis Migration 146 rechnete ausweis_nummer_start() „höchste A-Nummer in leser + 1".
// Verschwand die höchste, bekam die nächste Person dieselbe Nummer, und eine noch
// vorhandene Karte buchte an der Theke auf sie. Littera kennt dasselbe als Einstellung
// („freie Nummern wieder vergeben", Handbuch „Nummernvergabe/Nummernkreis"), und die Schule
// hat es dort genutzt: 2010 bekamen 349 neue Leser Nummern zwischen 2 und 1743, während
// Nummern bis 3531 vergeben waren. Bei uns ist die Nummer selbst der Scanwert der Karte.
//
// Je Weg, auf dem eine Nummer verschwindet, einmal: Die höchste Nummer (A-10050) geht, die
// nächste gezogene ist A-10051.
func TestAusweisnummer_KommtNieWieder(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	neu := func(t *testing.T, art, nummer string) string {
		t.Helper()
		sql := `INSERT INTO leser (vorname, nachname, art, klasse, abgaenger_jahr, barcode_id)
		        VALUES ('Nie', 'Wieder', 'schueler', '07A', 2031, $1) RETURNING id::text`
		if art != "schueler" {
			sql = `INSERT INTO leser (vorname, nachname, art, barcode_id)
			       VALUES ('Nie', 'Wieder', 'lehrkraft', $1) RETURNING id::text`
		}
		var id string
		if err := pool.QueryRow(ctx, sql, nummer).Scan(&id); err != nil {
			t.Fatalf("Leser %s anlegen: %v", nummer, err)
		}
		return id
	}
	ausfuehren := func(t *testing.T, sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}

	for _, fall := range []struct {
		name string
		art  string
		weg  func(t *testing.T, id string)
	}{
		{"endgültig gelöscht (Papierkorb, PurgeStudent)", "schueler", func(t *testing.T, id string) {
			ausfuehren(t, `UPDATE leser SET deleted_at = now() WHERE id = $1`, id)
			if err := repository.NewAuditRepository(pool).PurgeStudent(ctx, id, ""); err != nil {
				t.Fatalf("PurgeStudent: %v", err)
			}
		}},
		// Dieselbe Zuweisung über dieselbe Sicht wie die DSGVO-Anonymisierung
		// (jobs/cron_dsgvo.go) und der LUSD-Pfad: Die Nummer weicht einem Platzhalter.
		{"anonymisiert (über die Sicht schueler)", "schueler", func(t *testing.T, id string) {
			ausfuehren(t, `UPDATE schueler SET barcode_id = 'ANON-' || id::text, anonymized_at = now() WHERE id = $1`, id)
		}},
		{"von Hand umgeschrieben", "schueler", func(t *testing.T, id string) {
			ausfuehren(t, `UPDATE leser SET barcode_id = 'A-TIPPFEHLER' WHERE id = $1`, id)
		}},
		{"geleert (Kollege ohne Konto)", "lehrkraft", func(t *testing.T, id string) {
			ausfuehren(t, `UPDATE leser SET barcode_id = NULL WHERE id = $1`, id)
		}},
		{"zusammengeführt (die Quelle fällt)", "schueler", func(t *testing.T, id string) {
			ziel := neu(t, "schueler", "A-10048")
			if _, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(ziel, id)); err != nil {
				t.Fatalf("Zusammenführen: %v", err)
			}
		}},
	} {
		t.Run(fall.name, func(t *testing.T) {
			resetBestandsdaten(t, pool)
			schueler(t, pool, "A-10049")
			id := neu(t, fall.art, "A-10050")
			fall.weg(t, id)
			if n := zfZaehle(t, pool, `SELECT count(*) FROM leser WHERE barcode_id = 'A-10050'`); n != 0 {
				t.Fatalf("der Weg hat die Nummer nicht entfernt (%d Zeilen tragen sie)", n)
			}
			if got := naechsteAusweisnummer(t, pool); got != 10051 {
				t.Errorf("nächste Nummer %d, erwartet 10051 — A-10050 gehörte schon einmal einer Person", got)
			}
		})
	}

	// Weich gelöscht bleibt die Nummer in der Zeile stehen; auch sie kommt nicht wieder.
	t.Run("in den Papierkorb gelegt", func(t *testing.T) {
		resetBestandsdaten(t, pool)
		id := neu(t, "schueler", "A-10050")
		ausfuehren(t, `UPDATE leser SET deleted_at = now() WHERE id = $1`, id)
		if got := naechsteAusweisnummer(t, pool); got != 10051 {
			t.Errorf("nächste Nummer %d, erwartet 10051", got)
		}
	})

	// Eine von Hand eingetragene „A-0" ist eine Zahl, aber keine Nummer, die der Generator je
	// ausgäbe. Der Trigger lässt sie aus, statt an chk_ausweisnummer_ausgeschieden_positiv zu
	// scheitern — sonst ließe sich der Leser nicht mehr löschen.
	t.Run("A-0 lässt sich löschen", func(t *testing.T) {
		resetBestandsdaten(t, pool)
		id := neu(t, "schueler", "A-00000")
		ausfuehren(t, `DELETE FROM leser WHERE id = $1`, id)
		if got := naechsteAusweisnummer(t, pool); got != 10001 {
			t.Errorf("nächste Nummer %d, erwartet 10001", got)
		}
	})

	// Die Regel gilt dem Generator. Von Hand bleibt eine frühere Nummer eintragbar — wie die
	// eines Schülers im Papierkorb (TestAusweisnummer_UeberSchuelerUndKollegium): etwa die
	// alte Karte, die ein zurückgekehrter Schüler noch in der Hand hat.
	t.Run("von Hand bleibt sie eintragbar", func(t *testing.T) {
		resetBestandsdaten(t, pool)
		id := neu(t, "schueler", "A-10050")
		ausfuehren(t, `DELETE FROM leser WHERE id = $1`, id)
		neu(t, "schueler", "A-10050")
		if got := naechsteAusweisnummer(t, pool); got != 10051 {
			t.Errorf("nächste Nummer %d, erwartet 10051", got)
		}
	})
}
