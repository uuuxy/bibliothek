package inventur

import (
	"context"
	"strings"
	"testing"
)

// „Titel löschen" gegen ein echtes Postgres — an der einen Zusage, die dabei zählt:
// Ein Titel, von dem noch ein Exemplar bei jemandem zu Hause liegt, wird NICHT gelöscht.
//
// Warum das hier nochmal steht, obwohl TestPruefeKeineAktivenAusleihen existiert: Jener
// Test ruft den Wächter DIREKT auf und beantwortet ihm die Frage per Mock. Er kann
// deshalb zwei Dinge nicht sehen — ob DeleteBooks den Wächter überhaupt aufruft, und ob
// die Abfrage am echten Schema das trifft, was sie treffen soll. Genau diese Lücke
// (isoliert grün, im Live-Pfad unerreichbar) hat dieses Projekt schon einmal getäuscht.
//
// Der Schaden wäre still und gross: Der Titel-Delete räumt per CASCADE alle Exemplare
// und löscht vorher Schadensfälle und vergangene Ausleihen. Griffe er auch bei einem
// verliehenen Exemplar durch, verschwände das Buch aus dem Bestand, WÄHREND es unterwegs
// ist — niemand könnte es mehr zurückbuchen, und keine Mahnliste wüsste davon.
func TestDeleteBooks_VerliehenesExemplarBlockiert(t *testing.T) {
	pool := ladeSchemaPool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)

	eins := func(was, sql string, args ...any) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %v", was, err)
		}
		return id
	}

	titelID := eins("Titel", `INSERT INTO buecher_titel (titel) VALUES ('Der Zauberberg') RETURNING id`)
	// Zwei Exemplare: eines liegt im Regal, eines ist draussen. Es genügt EINES,
	// um den ganzen Titel zu halten — sonst risse das Löschen dem Kind das Buch
	// unter den Händen weg.
	imRegal := eins("Exemplar im Regal",
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'ZB-REGAL') RETURNING id`, titelID)
	verliehen := eins("Exemplar verliehen",
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'ZB-DRAUSSEN') RETURNING id`, titelID)
	schuelerID := eins("Schüler", `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('ZB-S1', 'Hans', 'Castorp', '9a', 2030) RETURNING id`)

	// rueckgabe_am IS NULL — das Buch ist bei ihm zu Hause.
	if _, err := pool.Exec(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
		VALUES ($1, $2, now() - interval '5 days', now() + interval '9 days')`,
		verliehen, schuelerID); err != nil {
		t.Fatalf("Ausleihe: %v", err)
	}

	err := repo.DeleteBooks(ctx, []string{titelID})
	if err == nil {
		t.Fatal("Titel mit verliehenem Exemplar wurde gelöscht — das Buch wäre spurlos aus dem Bestand")
	}
	// Die Meldung muss den Grund nennen. Ein blosser FK-Fehler käme als 500 zurück und
	// würde zu „interner Datenbankfehler" sanitisiert.
	if !strings.Contains(err.Error(), "verliehen") {
		t.Errorf("Meldung nennt den Grund nicht: %v", err)
	}

	// Nichts darf angefasst worden sein — auch nicht das Exemplar im Regal, und
	// vor allem nicht die Ausleihe selbst.
	for _, fall := range []struct{ sql, name string }{
		{`SELECT EXISTS(SELECT 1 FROM buecher_titel WHERE id = $1)`, "Titel"},
		{`SELECT EXISTS(SELECT 1 FROM buecher_exemplare WHERE id = $1)`, "Exemplar"},
	} {
		for _, id := range []string{titelID, imRegal, verliehen} {
			if fall.name == "Titel" && id != titelID {
				continue
			}
			if fall.name == "Exemplar" && id == titelID {
				continue
			}
			var da bool
			if e := pool.QueryRow(ctx, fall.sql, id).Scan(&da); e != nil {
				t.Fatal(e)
			}
			if !da {
				t.Errorf("%s %s wurde trotz Blockade entfernt", fall.name, id)
			}
		}
	}
	var offen bool
	if e := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL)`,
		verliehen).Scan(&offen); e != nil {
		t.Fatal(e)
	}
	if !offen {
		t.Error("die offene Ausleihe wurde entfernt — das Buch wäre nicht mehr zurückbuchbar")
	}

	// Gegenprobe am selben Titel: Nach der Rückgabe geht es. Ohne sie bewiese der Test
	// nur, dass DeleteBooks irgendetwas ablehnt — nicht, dass es die RICHTIGE Frage stellt.
	if _, err := pool.Exec(ctx,
		`UPDATE ausleihen SET rueckgabe_am = now() WHERE exemplar_id = $1`, verliehen); err != nil {
		t.Fatalf("Rückgabe: %v", err)
	}
	if err := repo.DeleteBooks(ctx, []string{titelID}); err != nil {
		t.Fatalf("nach der Rückgabe muss der Titel löschbar sein: %v", err)
	}
	var rest int
	if e := pool.QueryRow(ctx,
		`SELECT count(*) FROM buecher_exemplare WHERE titel_id = $1`, titelID).Scan(&rest); e != nil {
		t.Fatal(e)
	}
	if rest != 0 {
		t.Errorf("%d Exemplare überlebten den Titel-Delete (CASCADE griff nicht)", rest)
	}
}
