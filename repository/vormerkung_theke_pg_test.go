package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/internal/pgtest"

	"github.com/jackc/pgx/v5"
)

// warteschlange legt über q einen Titel mit einem freien Exemplar und je Namen einen Schüler
// mit wartender Vormerkung an, den zuerst genannten mit der ältesten. Die Namen halten Nummern
// und Namen auseinander.
func warteschlange(t *testing.T, q DBQueryer, kuerzel string, namen ...string) (titelID, exemplarID string, leser map[string]string) {
	t.Helper()
	ctx := context.Background()
	if err := q.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ($1) RETURNING id::text`, "Warteschlange "+kuerzel).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if err := q.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id::text`,
		titelID, "WS-"+kuerzel).Scan(&exemplarID); err != nil {
		t.Fatalf("Exemplar anlegen: %v", err)
	}
	leser = map[string]string{}
	for i, name := range namen {
		var id string
		if err := q.QueryRow(ctx, `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			VALUES ($1, $2, $3, '07B', 2031) RETURNING id::text`, "A-WS-"+kuerzel+"-"+name, "Vorname "+name, "Nachname "+name).Scan(&id); err != nil {
			t.Fatalf("Schüler %s anlegen: %v", name, err)
		}
		if _, err := q.Exec(ctx, `INSERT INTO vormerkungen (titel_id, schueler_id, status, erstellt_am)
			VALUES ($1, $2, 'wartend', now() - make_interval(days => $3))`, titelID, id, len(namen)-i); err != nil {
			t.Fatalf("Vormerkung für %s anlegen: %v", name, err)
		}
		leser[name] = id
	}
	return titelID, exemplarID, leser
}

// nimmtAbholrecht macht aus drei Schülern der Warteschlange solche, die nicht abholen dürfen:
// gelöscht, gesperrt, von Hand gesperrt.
func nimmtAbholrecht(t *testing.T, q DBQueryer, geloescht, gesperrt, vonHand string) {
	t.Helper()
	ctx := context.Background()
	for _, a := range []struct{ sql, id string }{
		{`UPDATE leser SET deleted_at = now() WHERE id = $1`, geloescht},
		{`UPDATE leser SET ist_gesperrt = true, block_reason = 'Probe' WHERE id = $1`, gesperrt},
		{`UPDATE leser SET is_manually_blocked = true, block_reason = 'Probe' WHERE id = $1`, vonHand},
	} {
		if _, err := q.Exec(ctx, a.sql, a.id); err != nil {
			t.Fatalf("%s: %v", a.sql, err)
		}
	}
}

// Mit der Ausleihe ist die Vormerkung des Lesers für diesen Titel erfüllt und fällt weg; die
// eines anderen Lesers und die für einen anderen Titel bleiben. Zurück kommt das Exemplar, das
// für ihn bereitlag; ohne Vormerkung und ohne bereitgelegtes Exemplar kommt nichts zurück.
func TestLoescheErfuellteVormerkung_NurDieDesLesersFuerDenTitel(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	titel, exemplar, leser := warteschlange(t, tx, "erfuellt", "Ausleiher", "Anderer")
	andererTitel, _, _ := warteschlange(t, tx, "erfuellt-anderer")
	if _, err := tx.Exec(ctx, `UPDATE vormerkungen SET status = 'abholbereit', bereitgestellt_exemplar_id = $1,
		bereitgestellt_bis = now() + interval '2 days' WHERE titel_id = $2 AND schueler_id = $3`, exemplar, titel, leser["Ausleiher"]); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO vormerkungen (titel_id, schueler_id) VALUES ($1, $2)`, andererTitel, leser["Ausleiher"]); err != nil {
		t.Fatal(err)
	}
	uebrig := func() int {
		t.Helper()
		var n int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM vormerkungen WHERE titel_id = ANY($1::uuid[])`, []string{titel, andererTitel}).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	bereit, err := LoescheErfuellteVormerkung(ctx, tx, titel, leser["Ausleiher"])
	if err != nil || bereit == nil || *bereit != exemplar {
		t.Errorf("erfüllte Vormerkung: bereitgelegt %v, Fehler %v; erwartet das Exemplar %s", bereit, err, exemplar)
	}
	if n := uebrig(); n != 2 {
		t.Errorf("%d Vormerkungen übrig, erwartet die des anderen Lesers und die für den anderen Titel", n)
	}
	if bereit, err := LoescheErfuellteVormerkung(ctx, tx, titel, leser["Ausleiher"]); err != nil || bereit != nil {
		t.Errorf("ohne Vormerkung: bereitgelegt %v, Fehler %v; erwartet nichts und keinen Fehler", bereit, err)
	}
	if bereit, err := LoescheErfuellteVormerkung(ctx, tx, andererTitel, leser["Ausleiher"]); err != nil || bereit != nil {
		t.Errorf("Vormerkung ohne bereitgelegtes Exemplar: %v, Fehler %v; erwartet nichts", bereit, err)
	}
	if n := uebrig(); n != 1 {
		t.Errorf("%d Vormerkungen übrig, erwartet nur die des anderen Lesers", n)
	}
}

// Die Nummer eines Exemplars ist die auf seinem Etikett, nicht seine Kennung.
func TestExemplarNummer_IstDieNummerDesEtiketts(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()
	_, exemplar, _ := warteschlange(t, tx, "nummer")

	if nummer, err := ExemplarNummer(ctx, tx, exemplar); err != nil || nummer != "WS-nummer" {
		t.Errorf("Nummer %q, Fehler %v; erwartet WS-nummer", nummer, err)
	}
	if _, err := ExemplarNummer(ctx, tx, "3f2504e0-4f89-11d3-9a0c-0305e82c3301"); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("unbekanntes Exemplar: Fehler %v, erwartet pgx.ErrNoRows", err)
	}
}

// Bedient wird die älteste wartende Vormerkung eines Schülers, der abholen darf: nicht die des
// gelöschten, des gesperrten oder des von Hand gesperrten, nicht eine, die schon bereitliegt,
// und nicht die des Lesers, der ausgenommen ist. Der Name des Wartenden kommt mit.
func TestSperreNaechsteWartendeVormerkung_AeltesteDieAbholenDarf(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	titel, exemplar, leser := warteschlange(t, tx, "naechste", "Geloescht", "Gesperrt", "VonHand", "Bereit", "Erste", "Zweite")
	nimmtAbholrecht(t, tx, leser["Geloescht"], leser["Gesperrt"], leser["VonHand"])
	if _, err := tx.Exec(ctx, `UPDATE vormerkungen SET status = 'abholbereit', bereitgestellt_exemplar_id = $1,
		bereitgestellt_bis = now() + interval '2 days' WHERE titel_id = $2 AND schueler_id = $3`, exemplar, titel, leser["Bereit"]); err != nil {
		t.Fatal(err)
	}
	kennung := func(name string) string {
		t.Helper()
		var id string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM vormerkungen WHERE titel_id = $1 AND schueler_id = $2`, titel, leser[name]).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}

	ist, da, err := SperreNaechsteWartendeVormerkung(ctx, tx, titel, nil)
	if err != nil || !da {
		t.Fatalf("nächste Wartende: da=%v, Fehler %v", da, err)
	}
	if soll := (WartendeVormerkung{ID: kennung("Erste"), Vorname: "Vorname Erste", Nachname: "Nachname Erste", Klasse: "07B"}); ist != soll {
		t.Errorf("nächste Wartende %+v, erwartet %+v", ist, soll)
	}
	erste := leser["Erste"]
	if ist, da, err := SperreNaechsteWartendeVormerkung(ctx, tx, titel, &erste); err != nil || !da || ist.ID != kennung("Zweite") {
		t.Errorf("ohne die Erste: %+v, da=%v, Fehler %v; erwartet die Zweite", ist, da, err)
	}
	leererTitel, _, _ := warteschlange(t, tx, "naechste-leer")
	if _, da, err := SperreNaechsteWartendeVormerkung(ctx, tx, leererTitel, nil); err != nil || da {
		t.Errorf("Titel ohne Wartende: da=%v, Fehler %v; erwartet niemanden", da, err)
	}
}

// Gesperrt wird die Vormerkung und nur sie. Hält eine Ausleihe gerade die Zeile des ältesten
// Wartenden, bekommt er das Buch trotzdem. Eine zweite Rückgabe desselben Titels wartet nicht
// auf die gesperrte Vormerkung und nimmt sie nicht noch einmal, sie greift die nächste.
func TestSperreNaechsteWartendeVormerkung_SperrtNurDieVormerkung(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	titel, _, leser := warteschlange(t, pool, "sperre", "Aelteste", "Naechste")

	ausleihe := beginne(t, pool)
	defer db.SafeRollback(ctx, ausleihe)
	if err := SperreLeserzeile(ctx, ausleihe, leser["Aelteste"]); err != nil {
		t.Fatalf("Leserzeile sperren: %v", err)
	}

	rueckgabe := beginne(t, pool)
	defer db.SafeRollback(ctx, rueckgabe)
	ist, da, err := SperreNaechsteWartendeVormerkung(ctx, rueckgabe, titel, nil)
	if err != nil || !da || ist.Nachname != "Nachname Aelteste" {
		t.Fatalf("erste Rückgabe: %+v, da=%v, Fehler %v; erwartet die Älteste trotz gesperrter Leserzeile", ist, da, err)
	}

	type antwort struct {
		wartende WartendeVormerkung
		da       bool
		err      error
	}
	zweite := make(chan antwort, 1)
	fertig := make(chan error, 1)
	go func() {
		tx, err := pool.Begin(ctx)
		if err != nil {
			zweite <- antwort{err: err}
			fertig <- err
			return
		}
		defer db.SafeRollback(ctx, tx)
		w, da, err := SperreNaechsteWartendeVormerkung(ctx, tx, titel, nil)
		zweite <- antwort{w, da, err}
		fertig <- err
	}()
	ueberschneidung(t, pool, rueckgabe, fertig)
	if len(fertig) == 0 {
		t.Fatal("die zweite Rückgabe wartet auf die gesperrte Vormerkung, statt die nächste zu greifen")
	}
	if a := <-zweite; a.err != nil || !a.da || a.wartende.Nachname != "Nachname Naechste" {
		t.Errorf("zweite Rückgabe: %+v, da=%v, Fehler %v; erwartet die Nächste", a.wartende, a.da, a.err)
	}
}

// Bereitstellen trägt an der Vormerkung Status, Exemplar und Abholfrist ein und lässt die
// Vormerkung daneben, wie sie ist. Eine Vormerkung, die es nicht gibt, ist
// ErrVormerkungNichtGefunden.
func TestStelleVormerkungBereit_SchreibtJedeAngabe(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	titel, exemplar, leser := warteschlange(t, tx, "bereit", "Erste", "Zweite")
	var vormerkung string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM vormerkungen WHERE titel_id = $1 AND schueler_id = $2`, titel, leser["Erste"]).Scan(&vormerkung); err != nil {
		t.Fatal(err)
	}
	frist := time.Date(2026, 6, 12, 21, 59, 59, 0, time.UTC)
	if err := StelleVormerkungBereit(ctx, tx, vormerkung, exemplar, frist); err != nil {
		t.Fatalf("StelleVormerkungBereit: %v", err)
	}

	var status, istExemplar string
	var istFrist time.Time
	if err := tx.QueryRow(ctx, `SELECT status, bereitgestellt_exemplar_id::text, bereitgestellt_bis FROM vormerkungen WHERE id = $1`, vormerkung).
		Scan(&status, &istExemplar, &istFrist); err != nil {
		t.Fatal(err)
	}
	if status != "abholbereit" || istExemplar != exemplar || !istFrist.Equal(frist) {
		t.Errorf("Vormerkung: Status %q, Exemplar %s, Frist %s", status, istExemplar, istFrist)
	}
	var andere int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM vormerkungen WHERE titel_id = $1 AND schueler_id = $2
		AND status = 'wartend' AND bereitgestellt_exemplar_id IS NULL AND bereitgestellt_bis IS NULL`, titel, leser["Zweite"]).Scan(&andere); err != nil {
		t.Fatal(err)
	}
	if andere != 1 {
		t.Error("die Vormerkung daneben ist nicht mehr unverändert wartend")
	}

	err := StelleVormerkungBereit(ctx, tx, "3f2504e0-4f89-11d3-9a0c-0305e82c3301", exemplar, frist)
	if !errors.Is(err, ErrVormerkungNichtGefunden) {
		t.Errorf("unbekannte Vormerkung: Fehler %v, erwartet ErrVormerkungNichtGefunden", err)
	}
}

// Auch beim Nachrücken bekommt das freigewordene Buch nur, wer abholen darf: Die drei Ältesten
// der Warteschlange sind gelöscht, gesperrt und von Hand gesperrt, bedient wird der Vierte.
func TestNachruecken_UebergehtWerNichtAbholenDarf(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	titel, exemplar, leser := warteschlange(t, tx, "nachruecken", "Geloescht", "Gesperrt", "VonHand", "Darf")
	nimmtAbholrecht(t, tx, leser["Geloescht"], leser["Gesperrt"], leser["VonHand"])

	bedient, err := bedieneNaechstenWartenden(ctx, tx, exemplar, titel, time.Now())
	if err != nil || !bedient {
		t.Fatalf("Nachrücken: bedient=%v, Fehler %v", bedient, err)
	}
	var bereitFuer string
	if err := tx.QueryRow(ctx, `SELECT schueler_id::text FROM vormerkungen WHERE titel_id = $1 AND status = 'abholbereit'`, titel).Scan(&bereitFuer); err != nil {
		t.Fatalf("bereitgestellte Vormerkung lesen: %v", err)
	}
	if bereitFuer != leser["Darf"] {
		t.Errorf("das Buch liegt für %s bereit, erwartet für den Vierten (%s)", bereitFuer, leser["Darf"])
	}
}
