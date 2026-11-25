package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
)

// Senkt die Maske den Bestand unter die Zahl der freien Exemplare, sonderte der Abgleich auch
// verliehene aus: Ihre Ausleihe blieb offen, mahnte weiter, und das Buch ließ sich nicht mehr
// zurückgeben. Jetzt lehnt er ab und nennt, wie viele verliehen sind; nichts ist gespeichert.
func TestBestand_SenktNichtUnterDieVerliehenen(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	abfragen := NewTitelAbfragen(pool)
	marke := fmt.Sprintf("VERL%d", time.Now().UnixNano())

	var titelID, leserID string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, autor, medientyp) VALUES ($1, $2, 'Buch') RETURNING id`,
		"Verliehen "+marke, marke).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO leser (barcode_id, vorname, nachname, art, klasse, abgaenger_jahr)
		VALUES ($1, 'Lena', 'Leserin', 'schueler', '07A', 2031) RETURNING id`, "SCH-"+marke).Scan(&leserID); err != nil {
		t.Fatalf("Leser anlegen: %v", err)
	}
	var bearbeiterID string
	if err := pool.QueryRow(ctx, `INSERT INTO benutzer (email, vorname, nachname, rolle)
		VALUES ($1, 'Be', 'Arbeiter', 'mitarbeiter') RETURNING id`, marke+"@schule.invalid").Scan(&bearbeiterID); err != nil {
		t.Fatalf("Bearbeiter anlegen: %v", err)
	}
	t.Cleanup(func() {
		auf := context.Background()
		for _, sql := range []string{
			`DELETE FROM ausleihen WHERE schueler_id = $1`,
			`DELETE FROM leser WHERE id = $1`,
		} {
			if _, err := pool.Exec(auf, sql, leserID); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		}
		for _, sql := range []string{
			`DELETE FROM buecher_exemplare WHERE titel_id = $1`,
			`DELETE FROM buecher_titel WHERE id = $1`,
		} {
			if _, err := pool.Exec(auf, sql, titelID); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		}
	})
	for nr := 1; nr <= 3; nr++ {
		var exemplarID string
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar) VALUES ($1, $2, true) RETURNING id`,
			titelID, fmt.Sprintf("B-VL-%d-%s", nr, marke)).Scan(&exemplarID); err != nil {
			t.Fatalf("Exemplar %d anlegen: %v", nr, err)
		}
		if nr == 3 {
			continue
		}
		if _, err := pool.Exec(ctx, `INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
			VALUES ($1, $2, NOW(), NOW() + interval '14 days')`, exemplarID, leserID); err != nil {
			t.Fatalf("Ausleihe %d anlegen: %v", nr, err)
		}
	}

	buecher, err := abfragen.LadeTitelNachIDs(ctx, []string{titelID})
	if err != nil || len(buecher) != 1 {
		t.Fatalf("Titel lesen: %v, %d Titel", err, len(buecher))
	}
	gesehen := 3
	err = abfragen.AendereTitel(ctx, titelID, buecher[0], alleTitelFelder(), &Bestandsangabe{Soll: 0, Gesehen: &gesehen, BearbeiterID: bearbeiterID})
	var verliehen *BestandVerliehen
	if !errors.As(err, &verliehen) {
		t.Fatalf("Bestand 3 → 0 bei zwei verliehenen: Fehler %v, erwartet BestandVerliehen", err)
	}
	if verliehen.Verliehen != 2 {
		t.Errorf("BestandVerliehen.Verliehen = %d, erwartet 2", verliehen.Verliehen)
	}
	var ausgesondert int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM buecher_exemplare WHERE titel_id = $1 AND ist_ausgesondert`, titelID).Scan(&ausgesondert); err != nil {
		t.Fatal(err)
	}
	if ausgesondert != 0 {
		t.Errorf("%d Exemplare ausgesondert, erwartet keins: Die Ablehnung speichert nichts", ausgesondert)
	}

	// Das freie Exemplar darf weiter gehen.
	if err := abfragen.AendereTitel(ctx, titelID, buecher[0], alleTitelFelder(), &Bestandsangabe{Soll: 2, Gesehen: &gesehen, BearbeiterID: bearbeiterID}); err != nil {
		t.Fatalf("Bestand 3 → 2: %v", err)
	}
}

// Senkt die Maske den Bestand, sondert der Abgleich zuerst ein Exemplar aus dem Regal aus: Eins im
// Abholfach liegt für einen Leser bereit, und ausgesondert scheiterte seine Abholung an der Theke.
func TestBestand_SondertZuerstAusDemRegalAus(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	abfragen := NewTitelAbfragen(pool)
	marke := fmt.Sprintf("ABH%d", time.Now().UnixNano())

	var titelID, leserID, bearbeiterID string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, autor, medientyp) VALUES ($1, $2, 'Buch') RETURNING id`,
		"Abholfach "+marke, marke).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO leser (barcode_id, vorname, nachname, art, klasse, abgaenger_jahr)
		VALUES ($1, 'Lena', 'Leserin', 'schueler', '07A', 2031) RETURNING id`, "SCH-"+marke).Scan(&leserID); err != nil {
		t.Fatalf("Leser anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO benutzer (email, vorname, nachname, rolle)
		VALUES ($1, 'Be', 'Arbeiter', 'mitarbeiter') RETURNING id`, marke+"@schule.invalid").Scan(&bearbeiterID); err != nil {
		t.Fatalf("Bearbeiter anlegen: %v", err)
	}
	t.Cleanup(func() {
		auf := context.Background()
		for _, sql := range []string{
			`DELETE FROM vormerkungen WHERE titel_id = $1`,
			`DELETE FROM buecher_exemplare WHERE titel_id = $1`,
			`DELETE FROM buecher_titel WHERE id = $1`,
		} {
			if _, err := pool.Exec(auf, sql, titelID); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		}
		if _, err := pool.Exec(auf, `DELETE FROM leser WHERE id = $1`, leserID); err != nil {
			t.Errorf("Aufräumen: %v", err)
		}
	})
	// Das Exemplar im Abholfach zuerst angelegt: Ohne Reihenfolge träfe LIMIT es zuerst.
	ids := make([]string, 2)
	for nr := range ids {
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar) VALUES ($1, $2, true) RETURNING id`,
			titelID, fmt.Sprintf("B-AB-%d-%s", nr, marke)).Scan(&ids[nr]); err != nil {
			t.Fatalf("Exemplar %d anlegen: %v", nr, err)
		}
	}
	abholfach, regal := ids[0], ids[1]
	if _, err := pool.Exec(ctx, `INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis)
		VALUES ($1, $2, 'abholbereit', $3, now() + interval '3 days')`, titelID, leserID, abholfach); err != nil {
		t.Fatalf("Vormerkung anlegen: %v", err)
	}

	buecher, err := abfragen.LadeTitelNachIDs(ctx, []string{titelID})
	if err != nil || len(buecher) != 1 {
		t.Fatalf("Titel lesen: %v, %d Titel", err, len(buecher))
	}
	gesehen := 2
	if err := abfragen.AendereTitel(ctx, titelID, buecher[0], alleTitelFelder(), &Bestandsangabe{Soll: 1, Gesehen: &gesehen, BearbeiterID: bearbeiterID}); err != nil {
		t.Fatalf("Bestand 2 → 1: %v", err)
	}
	var abholfachAus, regalAus bool
	if err := pool.QueryRow(ctx, `SELECT
			(SELECT ist_ausgesondert FROM buecher_exemplare WHERE id = $1),
			(SELECT ist_ausgesondert FROM buecher_exemplare WHERE id = $2)`, abholfach, regal).Scan(&abholfachAus, &regalAus); err != nil {
		t.Fatal(err)
	}
	if abholfachAus || !regalAus {
		t.Errorf("ausgesondert: Abholfach %v, Regal %v; erwartet das Exemplar aus dem Regal", abholfachAus, regalAus)
	}
}
