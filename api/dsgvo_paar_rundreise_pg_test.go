package api

import (
	"context"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Die Rundreise: Was die Auskunft zeigt, muss die Tilgung leeren — am echten Postgres,
// über die echten Pfade (DeleteStudent → PurgeStudent), über JEDE Quelle der
// gemeinsamen Liste dsgvoSchuelerQuellen (dsgvo_paar_vollstaendigkeit_test.go).
//
// Drei Prüfungen:
//  1. VORHER zeigt die Art.-15-Auskunft aus jeder Quelle etwas — sonst ist die
//     Auskunfts-Hälfte des Paars unvollständig, und das fällt nie auf, weil eine
//     unvollständige Auskunft gut aussieht.
//  2. NACHHER findet eine wertbasierte Sonde keinen der markanten Werte (Vorname,
//     Barcode, LUSD-ID, Entleiher-Klarname, Eltern-Mail) mehr in den Protokoll-Details —
//     wertbasiert, damit auch ein Schlüssel auffällt, den heute niemand kennt.
//     Die nackte Schüler-UUID bleibt bewusst stehen (Rechenschaftspflicht; sie
//     referenziert nach der Löschung niemanden mehr).
//  3. Die Artefakte eines ZWEITEN Schülers überleben unangetastet (Über-Tilgung).
//
// Wächst dsgvoSchuelerQuellen (FK-Ratsche unten), zwingt der default-Zweig der
// Seed-Schleife diesen Test zum Mitwachsen.
func TestDsgvoRundreise_PurgeTilgtWasDieAuskunftZeigt(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	const (
		vorname  = "Rundreisejakob"
		barcode  = "S-RUND-PROBE"
		lusd     = "LUSD-RUND-4711"
		entl     = "Rundreisejakob Test"
		eltern   = "rundreise-eltern@example.invalid"
		fremdEnt = "Fremdkind Bleibt"
	)

	sid := seedSchueler(t, pool, barcode, vorname, "5a")
	fremd := seedSchueler(t, pool, "S-RUND-FREMD", "Fremdkind", "5a")
	titelID := seedMonitorTitel(t, pool, "Rundreise-Titel", "Aut Or", false, 0)
	exID := exemplar(t, pool, titelID, "RUND-EX-1", true, "")

	if _, err := pool.Exec(ctx, `
		UPDATE schueler SET lusd_id = $2, eltern_email = $3, strasse = 'Rundreiseweg'
		WHERE id = $1`, sid, lusd, eltern); err != nil {
		t.Fatalf("Stammdaten anreichern: %v", err)
	}

	// Je Quelle der gemeinsamen Liste ein Artefakt — der default-Zweig hält die
	// Schleife mit der Liste im Gleichschritt.
	for _, q := range dsgvoSchuelerQuellen {
		var err error
		switch q.Tabelle {
		case "schueler":
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
		case "audit_log":
			// Lesehistorie, wie sie die Ausleihe schreibt (repository/audit_books.go) —
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
			// Verwaltungsprotokolle, wie sie student_update.go (LUSD_ID_NACHGETRAGEN)
			// und student_deleted.go (RESTORE/PURGE) schreiben.
			// … und das Zusammenführen (student_zusammenfuehren.go) legt beide Ausweis-
			// Barcodes ab — der Kanarienwert hier ist der Barcode der Rundreise.
			_, err = pool.Exec(ctx, `INSERT INTO audit_logs (aktion, details)
				VALUES ('LUSD_ID_NACHGETRAGEN', jsonb_build_object('schueler_id', $1::text, 'lusd_id', $2::text)),
				       ('SCHUELER_ZUSAMMENGEFUEHRT', jsonb_build_object('schueler_id', $1::text, 'barcode', $3::text, 'aufgeloest_barcode', 'RUNDREISE-QUELLE'))`,
				sid, lusd, barcode)
		default:
			t.Fatalf("keine Rundreise-Vorbereitung für Quelle %s — Test mit der Liste nachziehen", q.Tabelle)
		}
		if err != nil {
			t.Fatalf("Artefakt für %s anlegen: %v", q.Tabelle, err)
		}
	}

	// Der echte Lösch-Weg beginnt mit dem Soft-Delete — der schreibt den PII-Snapshot
	// (Vorname, Nachname, Barcode) in audit_log.details, genau die Spur, die die
	// Tilgung später neutralisieren muss.
	auditRepo := repository.NewAuditRepository(pool)
	if err := auditRepo.DeleteStudent(ctx, sid, "", "Rundreise"); err != nil {
		t.Fatalf("DeleteStudent: %v", err)
	}

	// ── 1. Die Auskunft zeigt VORHER aus jeder Quelle etwas ──
	srv := &Server{DB: &db.Database{Pool: pool}}
	daten, err := srv.sammleDsgvoDaten(ctx, sid)
	if err != nil {
		t.Fatalf("sammleDsgvoDaten vor der Löschung: %v", err)
	}
	for _, q := range dsgvoSchuelerQuellen {
		leer := false
		switch q.Tabelle {
		case "schueler":
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
			for _, e := range daten.auditEintraege {
				if e.Aktion == "CHECKOUT" {
					lesehistorie = true
				}
			}
			leer = !lesehistorie
		case "audit_logs":
			leer = len(daten.verwaltung) == 0
		default:
			t.Fatalf("keine Auskunfts-Prüfung für Quelle %s — Test mit der Liste nachziehen", q.Tabelle)
		}
		if leer {
			t.Errorf("die Auskunft zeigt nichts aus %s (Bezug: %s), obwohl dort Daten liegen — "+
				"diese Quelle fehlt in sammleDsgvoDaten oder wird falsch abgefragt", q.Tabelle, q.Bezug)
		}
	}

	// ── Der echte Lösch-Weg zu Ende: Purge aus dem Papierkorb ──
	if err := auditRepo.PurgeStudent(ctx, sid, ""); err != nil {
		t.Fatalf("PurgeStudent: %v", err)
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
		"schueler":        `SELECT count(*) FROM schueler WHERE id = $1`,
		"schueler_fotos":  `SELECT count(*) FROM schueler_fotos WHERE schueler_id = $1`,
		"schadensfaelle":  `SELECT count(*) FROM schadensfaelle WHERE schueler_id = $1`,
		"vormerkungen":    `SELECT count(*) FROM vormerkungen WHERE schueler_id = $1`,
		"ausleihen":       `SELECT count(*) FROM ausleihen WHERE schueler_id = $1`,
		"audit_log (LH)":  `SELECT count(*) FROM audit_log WHERE tabelle = 'ausleihen' AND details->>'schueler_id' = $1`,
		"audit_logs (Vw)": `SELECT count(*) FROM audit_logs WHERE (details ? 'lusd_id' OR details ? 'barcode' OR details ? 'aufgeloest_barcode') AND details->>'schueler_id' = $1`,
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
// muss in dsgvoSchuelerQuellen stehen — und umgekehrt. Wer eine neue Tabelle mit
// Schülerbezug baut, wird hier rot, bis Auskunft UND Tilgung sie kennen (die Liste
// erzwingt beides: den Quelltext-Scan der Auskunft und die Seed-Schleife der Rundreise).
//
// Spalten, die Schüler-IDs OHNE FK halten (audit-Details), fängt die Ratsche nicht —
// dafür stehen die wertbasierte Sonde oben und das Schlüssel-Vokabular-Gate
// (dsgvo_paar_vollstaendigkeit_test.go).
func TestSchuelerFremdschluessel_WachsenNurMitDemPaar(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	rows, err := pool.Query(ctx, `
		SELECT c.conrelid::regclass::text, a.attname
		FROM pg_constraint c
		JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY(c.conkey)
		WHERE c.contype = 'f' AND c.confrelid = 'schueler'::regclass
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

	inListe := map[string]bool{}
	for _, q := range dsgvoSchuelerQuellen {
		if q.MitFK {
			inListe[q.Tabelle+".schueler_id"] = true
		}
	}

	for fk := range imSchema {
		if !inListe[fk] {
			t.Errorf("das Schema kennt die Schüler-Referenz %s, die DSGVO-Quellenliste nicht — "+
				"in dsgvoSchuelerQuellen aufnehmen und Auskunft UND Tilgung nachziehen "+
				"(dsgvo_paar_vollstaendigkeit_test.go)", fk)
		}
	}
	for fk := range inListe {
		if !imSchema[fk] {
			t.Errorf("die DSGVO-Quellenliste nennt %s als FK, das Schema kennt ihn nicht (mehr) — "+
				"Liste pflegen", fk)
		}
	}
}
