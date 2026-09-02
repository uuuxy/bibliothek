package api

import (
	"context"
	"fmt"
	"testing"

	"bibliothek/db"
)

// TestLusdImport_KlassenwechselMehrererSchueler sichert den Zweck des LUSD-Imports ab:
// Er entscheidet, in welcher Klasse ein Schüler steht. Zum Schuljahreswechsel wandern
// alle Jahrgänge gleichzeitig — das ist keine Randbedingung, sondern der Normalfall.
//
// Warum als eigener Test und warum mit MEHREREN Schülern: Seit dem Umbau auf
// aktualisiereBestandsschuelerBatch (#441) laufen alle Aktualisierungen als EIN
// pgx.Batch. Jede Anweisung trägt ihre eigene Ziel-ID im Argument ($9), die Zuordnung
// entsteht also beim Einreihen. Genau dort kann sie auch kaputtgehen: Verrutschte der
// Index zwischen Datensatz und ID um eins, bekäme JEDER Schüler die Klasse seines
// Nachbarn. Alle Klassen hätten sich geändert, kein Fehler wäre geflogen, und ein Test,
// der nur „Klasse ist nicht mehr die alte" prüft, bliebe grün.
//
// Deshalb prüft dieser Test die PAARUNG: Jeder Schüler wird über seine lusd_id gelesen
// und gegen genau die Klasse verglichen, die für ihn im Export stand. Die Schüler
// bekommen absichtlich unterscheidbare Klassen — bei gleichen Werten wäre eine
// Verwechslung unsichtbar.
func TestLusdImport_KlassenwechselMehrererSchueler(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// Bestand: sechs aktive Schüler, jeder in einer anderen Klasse.
	type schueler struct{ lusdID, vorname, nachname, alteKlasse, neueKlasse string }
	faelle := []schueler{
		{"L-K1", "Anna", "Albers", "5a", "6a"},
		{"L-K2", "Ben", "Bauer", "5b", "6b"},
		{"L-K3", "Cem", "Celik", "5c", "6c"},
		{"L-K4", "Dana", "Dorn", "9a", "10a"},
		{"L-K5", "Emil", "Ernst", "Q2", "Q3"},
		{"L-K6", "Fine", "Frisch", "E1", "E2"},
	}

	for i, f := range faelle {
		if _, err := pool.Exec(ctx,
			`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id)
			 VALUES ($1, $2, $3, $4, 2030, $5)`,
			fmt.Sprintf("KW-%d", i), f.vorname, f.nachname, f.alteKlasse, f.lusdID,
		); err != nil {
			t.Fatalf("Schüler %s anlegen: %v", f.lusdID, err)
		}
	}

	// Der Export nennt für jeden dieselbe Person, aber die neue Klasse.
	records := make([]parsedStudentRow, 0, len(faelle))
	for i, f := range faelle {
		records = append(records, parsedStudentRow{
			LusdID: f.lusdID, Vorname: f.vorname, Nachname: f.nachname,
			Klasse: f.neueKlasse, LineNum: i + 1,
		})
	}

	s := &Server{DB: &db.Database{Pool: pool}}
	if _, err := s.computeLusdChanges(ctx, records, true, true); err != nil {
		t.Fatalf("Import mit Klassenwechseln scheiterte: %v", err)
	}

	for _, f := range faelle {
		var klasse, vorname string
		if err := pool.QueryRow(ctx,
			`SELECT klasse, vorname FROM schueler WHERE lusd_id = $1 AND deleted_at IS NULL`,
			f.lusdID,
		).Scan(&klasse, &vorname); err != nil {
			t.Fatalf("%s (%s) nach dem Import nicht auffindbar: %v", f.lusdID, f.nachname, err)
		}
		if vorname != f.vorname {
			t.Errorf("%s: Vorname %q, erwartet %q — die Zeilen sind vertauscht", f.lusdID, vorname, f.vorname)
		}
		if !klassenGleich(klasse, f.neueKlasse) {
			t.Errorf("%s (%s %s): Klasse %q, erwartet %q — die Zuordnung im Batch stimmt nicht",
				f.lusdID, f.vorname, f.nachname, klasse, f.neueKlasse)
		}
	}

	// Kein Schüler darf beim Klassenwechsel doppelt entstehen: Ein Bestandsschüler wird
	// AKTUALISIERT, nicht neu angelegt. Ginge das schief, stünde derselbe Mensch zweimal
	// im Bestand und würde beim nächsten Lauf am Unique-Index hängenbleiben.
	var anzahl int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM schueler WHERE lusd_id LIKE 'L-K%' AND deleted_at IS NULL`).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	if anzahl != len(faelle) {
		t.Errorf("%d aktive Zeilen, erwartet %d — der Import hat Schüler dupliziert", anzahl, len(faelle))
	}
}

// TestLusdImport_LeereKlasseUeberschreibtNicht haelt die COALESCE(NULLIF(...))-Regel fest:
// Steht im Export fuer einen Schueler KEINE Klasse, behaelt er seine bisherige. Ohne diese
// Regel wuerde eine luckenhafte Exportspalte die Klassenzuordnung der ganzen Schule leeren —
// und die Klasse ist die Angabe, an der Ausleihlimits, Mahnwege und Abgaengerlisten haengen.
func TestLusdImport_LeereKlasseUeberschreibtNicht(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	if _, err := pool.Exec(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id)
		 VALUES ('KW-LEER', 'Greta', 'Gruen', '7b', 2030, 'L-LEER')`); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}

	s := &Server{DB: &db.Database{Pool: pool}}
	if _, err := s.computeLusdChanges(ctx, []parsedStudentRow{
		{LusdID: "L-LEER", Vorname: "Greta", Nachname: "Gruen", Klasse: "", LineNum: 1},
	}, true, true); err != nil {
		t.Fatalf("Import scheiterte: %v", err)
	}

	var klasse string
	if err := pool.QueryRow(ctx,
		`SELECT klasse FROM schueler WHERE lusd_id = 'L-LEER' AND deleted_at IS NULL`).Scan(&klasse); err != nil {
		t.Fatal(err)
	}
	if klasse != "07B" {
		t.Errorf("Klasse = %q, erwartet 07B — eine leere Exportspalte darf die Zuordnung nicht loeschen", klasse)
	}
}

// TestLusdImport_MehrereAbgaengerGleichzeitig sichert die zweite heute umgebaute Stelle ab:
// behandleAbgaenger holt seit #440 die offenen Ausleihen und Schaeden fuer ALLE Abgaenger
// mit je einer GROUP-BY-Abfrage und loescht die Vormerkungen per ANY($1) — statt drei
// Abfragen je Schueler. Die Zuordnung entsteht dabei ueber eine Map (schueler_id -> Anzahl).
//
// Der Fehlerfall waere still: Greift die Map daneben, wird ein Schueler MIT offener Ausleihe
// anonymisiert (Name und Rechnungsadresse weg, die Schule bleibt auf dem Buch sitzen) oder
// einer OHNE Vorgaenge bleibt unnoetig gesperrt. Beides faellt erst Wochen spaeter auf.
//
// Zum Schuljahresende gehen ganze Jahrgaenge gleichzeitig — mehrere Abgaenger in einem Lauf
// sind der Normalfall, nicht die Ausnahme.
func TestLusdImport_MehrereAbgaengerGleichzeitig(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// Drei Abgaenger: einer sauber, einer mit offener Ausleihe, einer mit offenem Schaden.
	ids := map[string]string{}
	for i, name := range []string{"sauber", "ausleihe", "schaden"} {
		var id string
		if err := pool.QueryRow(ctx,
			`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id)
			 VALUES ($1, $2, 'Weg', 'Q4', 2030, $3) RETURNING id`,
			fmt.Sprintf("ABG-%d", i), name, "L-ABG-"+name,
		).Scan(&id); err != nil {
			t.Fatalf("Abgänger %s anlegen: %v", name, err)
		}
		ids[name] = id
	}
	// Die vorhandenen Helfer benutzen, nicht das SQL nachbauen: schadensfaelle traegt
	// check_damage_item (entweder exemplar_id ODER geraet_id), ein direkter INSERT ohne
	// Exemplar scheitert daran.
	seedOffeneAusleihe(t, pool, ids["ausleihe"], "ABGA")
	seedOffenerSchaden(t, pool, ids["schaden"], "ABGS")

	// Der neue Export nennt KEINEN der drei mehr — alle sind Abgänger.
	s := &Server{DB: &db.Database{Pool: pool}}
	if _, err := s.computeLusdChanges(ctx, []parsedStudentRow{
		{LusdID: "L-BLEIBT", Vorname: "Ida", Nachname: "Immernoch", Klasse: "8a", LineNum: 1},
	}, true, true); err != nil {
		t.Fatalf("Abgängerlauf scheiterte: %v", err)
	}

	// Seit dem 02.09.2026 gilt die Karenzzeit (Vorgabe 90 Tage): Auch „sauber" wird NICHT
	// sofort anonymisiert, sondern nur gesperrt — mit dem Karenz-Grund und gestempeltem
	// abgaenger_seit, damit der nächtliche Job die Frist rechnen kann. Die beiden anderen
	// tragen offene Vorgaenge und den Grund dafür; anonymisiert wird keiner.
	sollGrund := map[string]string{
		"sauber":   abgaengerSperrgrundKarenz,
		"ausleihe": abgaengerSperrgrundOffen,
		"schaden":  abgaengerSperrgrundOffen,
	}
	for name, grund := range sollGrund {
		var vorname, blockReason string
		var gesperrt, seitGesetzt bool
		if err := pool.QueryRow(ctx,
			`SELECT vorname, ist_gesperrt, COALESCE(block_reason, ''), abgaenger_seit IS NOT NULL FROM schueler WHERE id = $1`, ids[name],
		).Scan(&vorname, &gesperrt, &blockReason, &seitGesetzt); err != nil {
			t.Fatalf("%s nach dem Lauf lesen: %v", name, err)
		}
		if vorname == "Abgänger" {
			t.Errorf("%s: wurde anonymisiert — in der Karenzzeit darf das nicht passieren", name)
		}
		if !gesperrt || blockReason != grund {
			t.Errorf("%s: gesperrt=%v grund=%q, erwartet gesperrt mit %q", name, gesperrt, blockReason, grund)
		}
		if !seitGesetzt {
			t.Errorf("%s: abgaenger_seit fehlt — ohne Stempel läuft keine Karenz-Uhr", name)
		}
	}

	// Karenz 0 = das alte Verhalten: sofort anonymisieren, wer nichts mehr schuldet.
	if _, err := pool.Exec(ctx, `INSERT INTO system_einstellungen (schluessel, wert) VALUES ('abgaenger_karenz_tage', '0')
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`); err != nil {
		t.Fatalf("Karenz 0 setzen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM system_einstellungen WHERE schluessel = 'abgaenger_karenz_tage'`); err != nil {
			t.Logf("Karenz-Einstellung aufräumen: %v", err)
		}
	})
	// Rückkehr + erneuter Abgang derselben drei — ein zweiter Export ohne sie.
	if _, err := pool.Exec(ctx, `UPDATE schueler SET ist_abgaenger = false, ist_gesperrt = false, block_reason = NULL, abgaenger_seit = NULL WHERE id = ANY($1)`,
		[]string{ids["sauber"], ids["ausleihe"], ids["schaden"]}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.computeLusdChanges(ctx, []parsedStudentRow{
		{LusdID: "L-BLEIBT", Vorname: "Ida", Nachname: "Immernoch", Klasse: "8a", LineNum: 1},
	}, true, true); err != nil {
		t.Fatalf("zweiter Abgängerlauf scheiterte: %v", err)
	}
	for name, sollAnonym := range map[string]bool{"sauber": true, "ausleihe": false, "schaden": false} {
		var vorname string
		if err := pool.QueryRow(ctx, `SELECT vorname FROM schueler WHERE id = $1`, ids[name]).Scan(&vorname); err != nil {
			t.Fatal(err)
		}
		if (vorname == "Abgänger") != sollAnonym {
			t.Errorf("Karenz 0: %s anonymisiert=%v, erwartet %v — die Zuordnung der offenen Vorgaenge stimmt nicht",
				name, vorname == "Abgänger", sollAnonym)
		}
	}
}
