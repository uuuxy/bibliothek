package leser

import (
	"context"
	"regexp"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// Die Rundreise: Was die Auskunft zeigt, muss die Tilgung leeren — am echten Postgres,
// über die echten Pfade (LegeLeserInPapierkorb → LoescheLeserEndgueltig), über jede Quelle der
// gemeinsamen Liste testdaten.DSGVOLeserQuellen.
//
// Drei Prüfungen:
//  1. Vorher zeigt die Art.-15-Auskunft aus jeder Quelle etwas — sonst ist die
//     Auskunfts-Hälfte des Paars unvollständig, und das fällt nie auf, weil eine
//     unvollständige Auskunft gut aussieht.
//  2. Nachher findet eine wertbasierte Sonde keinen der markanten Werte (Vorname,
//     Barcode, LUSD-ID, Entleiher-Klarname, Eltern-Mail) mehr in den Protokoll-Details —
//     wertbasiert, damit auch ein Schlüssel auffällt, den heute niemand kennt.
//     Die nackte Schüler-UUID bleibt bewusst stehen (Rechenschaftspflicht; sie
//     referenziert nach der Löschung niemanden mehr).
//  3. Die Artefakte eines zweiten Schülers überleben unangetastet (Über-Tilgung).
//
// Wächst testdaten.DSGVOLeserQuellen (FK-Ratsche unten), zwingt der default-Zweig der
// Seed-Schleife diesen Test zum Mitwachsen.
func TestDSGVORundreise_EndgueltigesLoeschenTilgtWasDieAuskunftZeigt(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	const (
		vorname  = "Rundreisejakob"
		barcode  = "S-RUND-PROBE"
		lusd     = "LUSD-RUND-4711"
		entl     = "Rundreisejakob Test"
		eltern   = "rundreise-eltern@example.invalid"
		fremdEnt = "Fremdkind Bleibt"
	)

	sid := testdaten.LegeSchuelerAn(t, pool, barcode, vorname, "5a")
	fremd := testdaten.LegeSchuelerAn(t, pool, "S-RUND-FREMD", "Fremdkind", "5a")
	titelID := testdaten.LegeTitelMitCoverUndAlterAn(t, pool, "Rundreise-Titel", "Aut Or", false, 0)
	exID := testdaten.LegeExemplarAn(t, pool, titelID, "RUND-EX-1", true, "")

	if _, err := pool.Exec(ctx, `
		UPDATE schueler SET lusd_id = $2, eltern_email = $3, strasse = 'Rundreiseweg'
		WHERE id = $1`, sid, lusd, eltern); err != nil {
		t.Fatalf("Stammdaten anreichern: %v", err)
	}

	// Je Quelle der gemeinsamen Liste ein Artefakt — der default-Zweig hält die
	// Schleife mit der Liste im Gleichschritt.
	for _, q := range testdaten.DSGVOLeserQuellen {
		var err error
		switch q.Tabelle {
		case "leser":
			// oben angelegt
		case "schueler_fotos":
			_, err = pool.Exec(ctx, `INSERT INTO schueler_fotos (schueler_id, foto_encrypted)
				VALUES ($1, '\x00'::bytea)`, sid)
		case "ausleihen":
			_, err = pool.Exec(ctx, `INSERT INTO ausleihen
				(exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
				VALUES ($1, $2, NOW() - interval '10 days', NOW() + interval '4 days', NOW() - interval '2 days')`,
				exID, sid)
		case "schadensfaelle":
			_, err = pool.Exec(ctx, `INSERT INTO schadensfaelle
				(exemplar_id, schueler_id, beschreibung, betrag, ist_bezahlt)
				VALUES ($1, $2, 'Rundreise-Schaden', 3.50, true)`, exID, sid)
		case "vormerkungen":
			_, err = pool.Exec(ctx, `INSERT INTO vormerkungen (titel_id, schueler_id, status)
				VALUES ($1, $2, 'wartend')`, titelID, sid)
		case "nachbuch_meldungen":
			_, err = pool.Exec(ctx, `INSERT INTO nachbuch_meldungen
				(idempotency_key, exemplar_id, barcode, ergebnis, grund, vorbesitzer_schueler_id, gescannt_am)
				VALUES (gen_random_uuid(), $1, 'B-RUNDREISE', 'umgebucht', 'Rundreise', $2, NOW() - interval '1 hour')`, exID, sid)
		case "audit_log":
			// Lesehistorie, wie sie die Ausleihe schreibt (repository/protokoll.go) —
			// je eine Zeile für den Prüfling und den fremden Schüler.
			for _, z := range []struct{ id, name string }{{sid, entl}, {fremd, fremdEnt}} {
				if _, err = pool.Exec(ctx, `
					INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, details)
					VALUES ('ausleihen', 'CHECKOUT', $1::uuid, 'USER',
					        jsonb_build_object('exemplar_id', $1::text, 'schueler_id', $2::text, 'entleiher', $3::text))`,
					exID, z.id, z.name); err != nil {
					break
				}
			}
		case "audit_logs":
			// Verwaltungsprotokolle, wie sie leser_aendern.go (LUSD_ID_NACHGETRAGEN)
			// und papierkorb.go (RESTORE/PURGE) schreiben.
			// … und das Zusammenführen (leser_zusammenfuehren.go) legt beide Ausweis-
			// Barcodes ab — der Kanarienwert hier ist der Barcode der Rundreise.
			_, err = pool.Exec(ctx, `INSERT INTO audit_logs (aktion, details)
				VALUES ('LUSD_ID_NACHGETRAGEN', jsonb_build_object('schueler_id', $1::text, 'lusd_id', $2::text)),
				       ('SCHUELER_ZUSAMMENGEFUEHRT', jsonb_build_object('schueler_id', $1::text, 'barcode', $3::text, 'aufgeloest_barcode', 'RUNDREISE-QUELLE'))`,
				sid, lusd, barcode)
		case "schadensersatz_bescheide":
			// Ein Bescheid mit PII im Snapshot: Er bleibt nach dem endgültigen Löschen als
			// Beleg (die Referenznummer ordnet Zahlungen zu), aber ohne Personenbezug.
			_, err = pool.Exec(ctx, `
				INSERT INTO schadensersatz_bescheide
					(schueler_id, mittel, kassenjahr, laufende_nr, referenznummer, frist_bis,
					 gesamtbetrag, empfaenger_snapshot)
				VALUES ($1, 'land', 2026, 1, '5830 2026 1234 0001', CURRENT_DATE + 28, 24.90,
					jsonb_build_object('name', $2::text, 'strasse', 'Rundreiseweg'))`, sid, entl)
		case "benutzer":
			// Ein Zugangskonto, das auf diesen Leser zeigt (Migration 123). Nach der
			// Tilgung muss die Verknüpfung weg sein, das Konto selbst bleibt: Es ist die
			// Anmeldung samt Rechten und gehört der Anlage, nicht dem Leser.
			_, err = pool.Exec(ctx, `
				INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, leser_id)
				VALUES ('Rundreise', $2::text, 'rundreise-' || $1::text || '@test.invalid',
				        'kollegium', true, $1::uuid)`, sid, entl)
		default:
			t.Fatalf("keine Rundreise-Vorbereitung für Quelle %s — Test mit der Liste nachziehen", q.Tabelle)
		}
		if err != nil {
			t.Fatalf("Artefakt für %s anlegen: %v", q.Tabelle, err)
		}
	}

	// Der echte Lösch-Weg beginnt mit dem Papierkorb; der schreibt den PII-Snapshot
	// (Vorname, Nachname, Barcode) in audit_log.details, genau die Spur, die die
	// Tilgung später neutralisieren muss.
	auditRepo := repository.NewProtokollAbfragen(pool)
	if err := auditRepo.LegeLeserInPapierkorb(ctx, sid, "", "Rundreise"); err != nil {
		t.Fatalf("LegeLeserInPapierkorb: %v", err)
	}

	// ── 1. Die Auskunft zeigt vorher aus jeder Quelle etwas ──
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}
	daten, err := srv.sammleDSGVODaten(ctx, sid, true)
	if err != nil {
		t.Fatalf("sammleDSGVODaten vor der Löschung: %v", err)
	}
	for _, q := range testdaten.DSGVOLeserQuellen {
		leer := false
		switch q.Tabelle {
		case "leser":
			leer = daten.stammdaten == nil || daten.stammdaten.Vorname != vorname
		case "schueler_fotos":
			leer = !daten.foto.Vorhanden
		case "ausleihen":
			leer = len(daten.ausleihen) == 0
		case "schadensfaelle":
			leer = len(daten.schaeden) == 0
		case "vormerkungen":
			leer = len(daten.vormerkungen) == 0
		case "audit_log":
			lesehistorie := false
			for _, e := range daten.protokollEintraege {
				if e.Aktion == "CHECKOUT" {
					lesehistorie = true
				}
			}
			leer = !lesehistorie
		case "audit_logs":
			leer = len(daten.verwaltung) == 0
		case "schadensersatz_bescheide":
			leer = len(daten.bescheide) == 0
		case "nachbuch_meldungen":
			leer = len(daten.nachbuchMeldungen) == 0
		case "benutzer":
			// Die Auskunft nennt, dass ein Zugangskonto auf diesen Leser zeigt, und auch das
			// Konto selbst (repository/dsgvo_konto.go).
			leer = daten.stammdaten == nil || !daten.stammdaten.HatZugangskonto ||
				daten.zugangskonto == nil || daten.zugangskonto.Email != "rundreise-"+sid+"@test.invalid"
		default:
			t.Fatalf("keine Auskunfts-Prüfung für Quelle %s — Test mit der Liste nachziehen", q.Tabelle)
		}
		if leer {
			t.Errorf("die Auskunft zeigt nichts aus %s (Bezug: %s), obwohl dort Daten liegen — "+
				"diese Quelle fehlt in sammleDSGVODaten oder wird falsch abgefragt", q.Tabelle, q.Bezug)
		}
	}

	// ── Der echte Lösch-Weg zu Ende: endgültig löschen aus dem Papierkorb ──
	if err := auditRepo.LoescheLeserEndgueltig(ctx, sid, ""); err != nil {
		t.Fatalf("LoescheLeserEndgueltig: %v", err)
	}

	// ── 2. Wertbasierte Sonde: keiner der markanten Werte überlebt in den Protokollen ──
	for _, wert := range []string{vorname, barcode, lusd, entl, eltern} {
		var treffer int
		if err := pool.QueryRow(ctx, `
			SELECT (SELECT count(*) FROM audit_log  WHERE details::text LIKE '%' || $1 || '%')
			     + (SELECT count(*) FROM audit_logs WHERE details::text LIKE '%' || $1 || '%')`,
			wert).Scan(&treffer); err != nil {
			t.Fatalf("Sonde für %q: %v", wert, err)
		}
		if treffer != 0 {
			t.Errorf("der Wert %q steht nach dem Purge noch in %d Protokoll-Details — "+
				"eine Spur, die keine Tilgungs-Anweisung kennt (repository.SpurTilgungen erweitern)", wert, treffer)
		}
	}

	// Und die Tabellen selbst sind leer bzw. entkoppelt.
	for tabelle, zaehler := range map[string]string{
		"leser":           `SELECT count(*) FROM leser WHERE id = $1`,
		"schueler_fotos":  `SELECT count(*) FROM schueler_fotos WHERE schueler_id = $1`,
		"schadensfaelle":  `SELECT count(*) FROM schadensfaelle WHERE schueler_id = $1`,
		"vormerkungen":    `SELECT count(*) FROM vormerkungen WHERE schueler_id = $1`,
		"ausleihen":       `SELECT count(*) FROM ausleihen WHERE schueler_id = $1`,
		"audit_log (LH)":  `SELECT count(*) FROM audit_log WHERE tabelle = 'ausleihen' AND details->>'schueler_id' = $1`,
		"audit_logs (Vw)": `SELECT count(*) FROM audit_logs WHERE (details ? 'lusd_id' OR details ? 'barcode' OR details ? 'aufgeloest_barcode' OR details ? 'grund' OR details ? 'reason') AND details->>'schueler_id' = $1`,
		// Der Bescheid bleibt als Beleg, aber ohne Person: schueler_id ist NULL
		// (ON DELETE SET NULL) und der Snapshot geleert.
		"schadensersatz_bescheide": `SELECT count(*) FROM schadensersatz_bescheide WHERE schueler_id = $1`,
		// Das Konto bleibt (Anmeldung samt Rechten), seine Verknüpfung zum getilgten
		// Leser nicht — erst durch die Tilgung, dann durch ON DELETE SET NULL.
		"benutzer": `SELECT count(*) FROM benutzer WHERE leser_id = $1`,
	} {
		var n int
		if err := pool.QueryRow(ctx, zaehler, sid).Scan(&n); err != nil {
			t.Fatalf("%s zählen: %v", tabelle, err)
		}
		if n != 0 {
			t.Errorf("%s: %d Zeilen referenzieren den Schüler nach dem Purge noch", tabelle, n)
		}
	}

	// ── 3. Über-Tilgungs-Gegenprobe: der fremde Schüler ist unangetastet ──
	var fremdeDa int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM audit_log
		WHERE tabelle = 'ausleihen' AND details->>'entleiher' = $1`, fremdEnt).Scan(&fremdeDa); err != nil {
		t.Fatalf("fremde Zeile zählen: %v", err)
	}
	if fremdeDa != 1 {
		t.Errorf("fremde Lesehistorie-Zeile: %d (erwartet 1) — die Tilgung greift zu weit", fremdeDa)
	}
}

// Die Wachstums-Ratsche des Paars: Jede FK-Spalte auf schueler, die das Schema kennt,
// muss in testdaten.DSGVOLeserQuellen stehen — und umgekehrt. Wer eine neue Tabelle mit
// Schülerbezug baut, wird hier rot, bis Auskunft und Tilgung sie kennen (die Liste
// erzwingt beides: den Quelltext-Scan der Auskunft und die Seed-Schleife der Rundreise).
//
// Spalten, die Schüler-IDs ohne FK halten (Details im Protokoll), fängt die Ratsche nicht —
// dafür stehen die wertbasierte Sonde oben und das Schlüssel-Vokabular-Gate
// (repository/dsgvo_paar_vollstaendigkeit_test.go).
func TestSchuelerFremdschluessel_WachsenNurMitDemPaar(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	rows, err := pool.Query(ctx, `
		SELECT c.conrelid::regclass::text, a.attname
		FROM pg_constraint c
		JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY(c.conkey)
		-- 'leser'::regclass, nicht 'schueler': Seit Migration 124 ist schueler eine
		-- SICHT, und auf eine Sicht kann kein Fremdschlüssel zeigen. Stünde hier
		-- weiter der alte Name, fände der Scan nichts — und der Detektor meldete
		-- „alles gut", weil er nichts mehr messen kann. Genau davor schützt die
		-- Notbremse unten (len(imSchema) == 0).
		WHERE c.contype = 'f' AND c.confrelid = 'leser'::regclass
		ORDER BY 1, 2`)
	if err != nil {
		t.Fatalf("FK-Scan: %v", err)
	}
	defer rows.Close()

	imSchema := map[string]bool{}
	for rows.Next() {
		var tabelle, spalte string
		if err := rows.Scan(&tabelle, &spalte); err != nil {
			t.Fatalf("FK-Zeile: %v", err)
		}
		imSchema[tabelle+"."+spalte] = true
	}
	if rows.Err() != nil {
		t.Fatalf("FK-Scan: %v", rows.Err())
	}
	if len(imSchema) == 0 {
		t.Fatal("der FK-Scan fand keine einzige Referenz auf schueler — der Detektor misst nichts mehr")
	}

	// Welche Spalte(n) der Tabelle auf schueler zeigen, steht im Bezug-Text der Liste:
	// jedes Wort, das auf schueler_id oder leser_id endet (Migration 117 hat zwei je
	// Tabelle — ausleiher_schueler_id und vorbesitzer_schueler_id). Ohne Treffer gilt
	// schueler_id.
	//
	// leser_id kam mit Migration 123 dazu: Die Tabelle führt alle Leser, und die
	// stillschweigende Annahme „jede Referenz auf schueler heißt *schueler_id" gilt nicht.
	// Der Detektor erkennt damit mehr Spalten, nicht weniger; eine
	// unbekannte Referenz fiele sonst durch, und genau das ist der Fall, den er sucht.
	spaltenImBezug := regexp.MustCompile(`\b[a-z_]*(?:schueler_id|leser_id)\b`)
	inListe := map[string]bool{}
	for _, q := range testdaten.DSGVOLeserQuellen {
		if !q.MitFremdschluessel {
			continue
		}
		spalten := spaltenImBezug.FindAllString(q.Bezug, -1)
		if len(spalten) == 0 {
			spalten = []string{"schueler_id"}
		}
		for _, s := range spalten {
			inListe[q.Tabelle+"."+s] = true
		}
	}

	for fk := range imSchema {
		if !inListe[fk] {
			t.Errorf("das Schema kennt die Schüler-Referenz %s, die DSGVO-Quellenliste nicht — "+
				"in testdaten.DSGVOLeserQuellen aufnehmen und Auskunft UND Tilgung nachziehen "+
				"(internal/testdaten/dsgvo.go)", fk)
		}
	}
	for fk := range inListe {
		if !imSchema[fk] {
			t.Errorf("die DSGVO-Quellenliste nennt %s als FK, das Schema kennt ihn nicht (mehr) — "+
				"Liste pflegen", fk)
		}
	}
}
