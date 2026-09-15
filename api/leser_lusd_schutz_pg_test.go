package api

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"bibliothek/repository"
)

// Der LUSD-Abgleich darf einen Kollegen nicht anfassen.
//
// Durch die LUSD kommen ausschließlich Schüler (Peter, 16.09.2026: „durch die LUSD kommen
// keine Lehrer!!!!"). Genau deshalb ist ein Kollege in derselben Tabelle in Gefahr: Der
// Abgleich lädt den Bestand und behandelt JEDE Zeile, die der Export nicht kennt, als
// Abgänger — sperren, Grund setzen, nach der Karenz Name, Adresse und Geburtsdatum
// anonymisieren. Ein Kollegium in dieser Tabelle wäre beim ersten Import Freiwild, und
// gemerkt hätte es niemand, bis jemand einen Namen sucht.
//
// Zwei Schranken, zwei Tests. Die erste ist die Abfrage (art = 'schueler'), die zweite der
// CHECK in der Datenbank. Beide, weil eine Einschränkung in EINER Abfrage nur für diese
// gilt: Der nächste Schreibweg, der den Bestand anders liest, kennt das Versprechen nicht.

// leserZeile legt eine Leserzeile ohne Konto an und gibt ihre ID zurück.
func leserZeile(t *testing.T, art, vorname, nachname string) string {
	t.Helper()
	pool := pgTestPool(t)
	var id string
	// Klasse, Abgängerjahr und Ausweis bleiben leer — für einen Kollegen ist das der
	// Normalfall, und chk_leser_schueler_pflichtfelder erlaubt es nur für ihn.
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO schueler (vorname, nachname, art) VALUES ($1, $2, $3) RETURNING id`,
		vorname, nachname, art).Scan(&id); err != nil {
		t.Fatalf("Leserzeile (%s) anlegen: %v", art, err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM schueler WHERE id = $1`, id); err != nil {
			t.Logf("Aufräumen der Leserzeile %s: %v", id, err)
		}
	})
	return id
}

// leserZeileSchueler legt einen Schüler an — mit allem, was für ihn Pflicht ist.
func leserZeileSchueler(t *testing.T, barcode, vorname, nachname, klasse string) string {
	t.Helper()
	pool := pgTestPool(t)
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, art)
		 VALUES ($1, $2, $3, $4, 2030, 'schueler') RETURNING id`,
		barcode, vorname, nachname, klasse).Scan(&id); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM schueler WHERE id = $1`, id); err != nil {
			t.Logf("Aufräumen des Schülers %s: %v", id, err)
		}
	})
	return id
}

// TestLusdBestandKenntNurSchueler: Die Abfrage, aus der die Abgänger-Erkennung ihren
// Bestand zieht, liefert Kollegen nicht mit. Was sie nicht sieht, kann sie nicht sperren.
func TestLusdBestandKenntNurSchueler(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	schuelerID := leserZeileSchueler(t, "S-LUSD-1", "Mia", "Muster", "7a")
	kollegeID := leserZeile(t, "lehrkraft", "Petra", "Kollegin")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	bestand, err := ladeLusdBestand(ctx, tx)
	if err != nil {
		t.Fatalf("ladeLusdBestand: %v", err)
	}

	var sahSchueler, sahKollegen bool
	for _, b := range bestand {
		switch b.ID {
		case schuelerID:
			sahSchueler = true
		case kollegeID:
			sahKollegen = true
		}
	}
	if !sahSchueler {
		t.Error("der Schüler fehlt im Bestand — dann erkennt der Abgleich Klassenwechsel " +
			"und Abgänger nicht mehr; die Einschränkung greift zu weit")
	}
	if sahKollegen {
		t.Error("die Lehrkraft steht im LUSD-Bestand — der nächste Import, der sie nicht " +
			"kennt, würde sie zum Abgänger machen und später ihren Namen anonymisieren")
	}
}

// TestKollegeKannKeinAbgaengerWerden: die zweite Schranke, in der Datenbank. Selbst wenn
// ein Schreibweg an der Abfrage vorbeigeht und die Lehrkraft direkt sperrt, bricht die
// Transaktion ab. Ein fehlgeschlagener Import ist ein Ärgernis; anonymisierte
// Kollegennamen sind ein stiller Datenverlust.
func TestKollegeKannKeinAbgaengerWerden(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	kollegeID := leserZeile(t, "lehrkraft", "Ulf", "Unberuehrt")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Der ECHTE Schreibweg des Abgleichs, nicht ein nachgebautes UPDATE.
	err = sperreAbgaenger(ctx, tx, kollegeID, "Automatisierte Abgänger-Sperre")
	if err == nil {
		t.Fatal("eine Lehrkraft ließ sich als Abgänger sperren — der CHECK fehlt")
	}
	if !strings.Contains(err.Error(), "chk_leser_nur_schueler_werden_abgaenger") {
		t.Errorf("Abbruch kam, aber von anderer Stelle: %v", err)
	}
}

// TestKollegeKannNichtAnonymisiertWerden: derselbe Schutz für den Schritt, der wirklich
// Daten vernichtet. Er läuft in der Karenz nach dem Abgang und leert Name, Adresse,
// Geburtsdatum und Ausweis.
func TestKollegeKannNichtAnonymisiertWerden(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	kollegeID := leserZeile(t, "liv", "Lea", "Liv")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := anonymisiereAbgaenger(ctx, tx, kollegeID); err == nil {
		t.Fatal("eine LiV ließ sich anonymisieren — Name und Geburtsdatum wären weg")
	}
}

// TestNachtJobFasstKollegenNichtAn: Die nächtliche Anonymisierung hat zwei Zweige. Der
// eine hängt an „ist Abgänger" — den kann ein Kollege nach dem CHECK nie erreichen. Der
// andere hängt allein am PAPIERKORB (deleted_at) und fragt nicht nach der Art.
//
// Das ist die gefährlichere Hälfte, und zwar doppelt: Der Job würde den Kollegen
// anonymisieren wollen, der CHECK verbietet es, das UPDATE bricht ab — und damit fällt
// der GANZE Lauf aus. Eine einzige gelöschte Kollegenzeile hätte die Anonymisierung für
// alle echten Abgänger stillgelegt, Nacht für Nacht, mit einer Zeile im Protokoll.
//
// Geprüft wird an der EINEN Quelle der Frist (repository.PredikatAnonymisierung) — der
// Bedingung, die auch der Wächter der Selbstprüfung stellt.
func TestNachtJobFasstKollegenNichtAn(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	kollegeID := leserZeile(t, "lehrkraft", "Paul", "Papierkorb")
	schuelerID := leserZeileSchueler(t, "S-PK-1", "Sina", "Schluss", "10a")

	// Beide liegen lange genug im Papierkorb, dass die Frist abgelaufen ist.
	for _, id := range []string{kollegeID, schuelerID} {
		if _, err := pool.Exec(ctx,
			`UPDATE schueler SET deleted_at = NOW() - interval '400 days' WHERE id = $1`, id); err != nil {
			t.Fatalf("in den Papierkorb legen: %v", err)
		}
	}

	bedingung := repository.PredikatAnonymisierung(90, 0)
	trifft := func(id string) bool {
		args := append(append([]any{}, bedingung.Args...), id)
		var anzahl int
		abfrage := `SELECT count(*) FROM schueler WHERE id = $` +
			strconv.Itoa(len(args)) + ` AND (` + bedingung.Where + `)`
		if err := pool.QueryRow(ctx, abfrage, args...).Scan(&anzahl); err != nil {
			t.Fatalf("Bedingung prüfen: %v", err)
		}
		return anzahl > 0
	}

	// Gegenprobe zuerst: Trifft die Bedingung überhaupt etwas? Ohne sie könnte dieser
	// Test auch dann grün sein, wenn er gar nichts misst.
	if !trifft(schuelerID) {
		t.Fatal("die Bedingung trifft nicht einmal den gelöschten Schüler — der Test " +
			"misst nichts, statt etwas zu beweisen")
	}
	if trifft(kollegeID) {
		t.Error("die nächtliche Anonymisierung würde die Lehrkraft erfassen — das UPDATE " +
			"scheitert am CHECK, und damit fällt der ganze Lauf aus")
	}
}
