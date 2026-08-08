package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bibliothek/pkg/closeutil"

	"github.com/jackc/pgx/v5"
)

// wendeLusdAenderungenAn führt den zweiten Durchlauf aus: bestehende Schüler
// aktualisieren (Klasse + Kontaktdaten) und Neuzugänge anlegen.
func wendeLusdAenderungenAn(ctx context.Context, tx pgx.Tx, records []parsedStudentRow, dbStudents map[string]lusdDbStudent) error {
	barcodeCounter := 0

	var batchRecords []parsedStudentRow
	var batchIDs []string

	for _, rec := range records {
		if rec.LusdID == "" {
			continue
		}
		if dbRec, exists := dbStudents[rec.LusdID]; exists {
			batchRecords = append(batchRecords, rec)
			batchIDs = append(batchIDs, dbRec.ID)
			continue
		}

		// Nicht in der Aktiven-Liste (ladeAktiveSchueler filtert ist_abgaenger = false):
		// Es kann aber ein zurückkehrender Abgänger sein, dessen NICHT soft-gelöschte Zeile
		// die lusd_id weiterhin hält. Ein blindes INSERT (legeNeuenSchuelerAn) kollidiert
		// dann am partiellen Unique-Index uniq_schueler_lusd_id_active und ließe den GESAMTEN
		// Import scheitern (SQLSTATE 23505). Solche Rückkehrer werden reaktiviert statt neu
		// angelegt — aktualisiereBestandsschueler setzt ist_abgaenger zurück und hebt die
		// Abgänger-Sperre auf, sofern keine Vorgänge mehr offen sind (Ghost-Block).
		// Soft-gelöschte Zeilen (deleted_at IS NOT NULL) blockieren den Index NICHT und
		// sollen bewusst als frischer Datensatz neu entstehen — daher hier ausgeklammert.
		rueckkehrerID, err := findeAktivenSchuelerNachLusdID(ctx, tx, rec.LusdID)
		if err != nil {
			return err
		}
		if rueckkehrerID != "" {
			batchRecords = append(batchRecords, rec)
			batchIDs = append(batchIDs, rueckkehrerID)
			continue
		}

		barcodeCounter++
		if err := legeNeuenSchuelerAn(ctx, tx, rec, barcodeCounter); err != nil {
			return err
		}
	}

	if len(batchRecords) > 0 {
		return aktualisiereBestandsschuelerBatch(ctx, tx, batchRecords, batchIDs)
	}

	return nil
}

// findeAktivenSchuelerNachLusdID sucht die (höchstens eine — garantiert durch den partiellen
// Unique-Index) nicht soft-gelöschte Schülerzeile zu einer lusd_id. Leerer String, wenn keine
// existiert. Dient dazu, einen zurückkehrenden Abgänger zu erkennen, der nicht in der
// Aktiven-Liste steht, dessen Zeile aber die lusd_id noch belegt.
func findeAktivenSchuelerNachLusdID(ctx context.Context, tx pgx.Tx, lusdID string) (string, error) {
	var id string
	err := tx.QueryRow(ctx,
		"SELECT id FROM schueler WHERE lusd_id = $1 AND deleted_at IS NULL LIMIT 1", lusdID,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

// legeNeuenSchuelerAn legt einen per LUSD neu hinzugekommenen Schüler an.
// Leere Adress-/Kontaktwerte werden als NULL gespeichert.
func legeNeuenSchuelerAn(ctx context.Context, tx pgx.Tx, rec parsedStudentRow, barcodeCounter int) error {
	year := time.Now().Year() + 5 // Default-Abgangsjahr
	_, err := tx.Exec(ctx, `
		INSERT INTO schueler
			(barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id, geburtsdatum,
			 strasse, hausnummer, plz, ort, eltern_email)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		generateImportBarcode(barcodeCounter), rec.Vorname, rec.Nachname, rec.Klasse, year,
		rec.LusdID, rec.GebDatum,
		nullableString(rec.Strasse), nullableString(rec.Hausnummer), nullableString(rec.PLZ),
		nullableString(rec.Ort), nullableString(rec.ElternEmail))
	return err
}

// aktualisiereBestandsschueler übernimmt Klasse, Name und Kontaktdaten aus dem Export.
// COALESCE(NULLIF(...)) sorgt dafür, dass ein LEERER Export-Wert bestehende Daten
// NICHT überschreibt — ein Export ohne Adressspalten löscht also nichts.
//
// Rückkehrer-Behandlung: Steht ein Schüler wieder im aktiven Export, ist er per
// Definition KEIN Abgänger mehr. War er zuvor als Abgänger anonymisiert worden
// (Vorname 'Abgänger', Nachname 'Anonymisiert-…', gesperrt), bliebe er sonst dauerhaft
// unter diesem Namen und gesperrt im System — obwohl sein echter Name im Export steht
// (z. B. Wechsel in die Oberstufe). Deshalb: Name IMMER aus dem Export übernehmen und
// den Abgänger-/Anonymisierungs-Status zurücksetzen.
//
// Zwei Entsperr-Fälle beim Rückkehrer:
//  1. Anonymisierter Abgänger (Vorname 'Abgänger', Nachname 'Anonymisiert-…'): immer
//     entsperren und Grund räumen — er hatte per Definition keine offenen Vorgänge, sonst
//     wäre er gesperrt statt anonymisiert worden.
//  2. Nur gesperrter Abgänger (block_reason beginnt mit 'Automatisierte Abgänger-Sperre'):
//     dynamisch prüfen, ob NOCH offene Ausleihen oder unbezahlte Schäden bestehen. Sind
//     alle Vorgänge beglichen → entsperren und Grund räumen. Bestehen noch Vorgänge →
//     gesperrt lassen, aber den irreführenden „Abgänger"-Grund in einen sachlichen
//     „Sperre wegen offener Vorgänge" umbenennen (sonst „Permanent Ghost-Block").
//
// Eine Sperre aus ANDEREM Grund (manuell / nicht die Abgänger-Automatik) bleibt unangetastet.
// Die CASE-Ausdrücke lesen die ALTEN Zeilenwerte (Postgres wertet SET-RHS vor der Zuweisung
// aus), daher greifen die Namens-/Grund-Checks noch auf den Zustand VOR dem Namens-Update.
func aktualisiereBestandsschueler(ctx context.Context, tx pgx.Tx, rec parsedStudentRow, id string) error {
	return aktualisiereBestandsschuelerBatch(ctx, tx, []parsedStudentRow{rec}, []string{id})
}

// aktualisiereBestandsschuelerBatch aktualisiert mehrere Bestandsschüler oder Rückkehrer effizient als Batch.
func aktualisiereBestandsschuelerBatch(ctx context.Context, tx pgx.Tx, records []parsedStudentRow, ids []string) error {
	batch := &pgx.Batch{}
	for i, rec := range records {
		batch.Queue(`
		UPDATE schueler SET
			vorname      = COALESCE(NULLIF($1, ''), vorname),
			nachname     = COALESCE(NULLIF($2, ''), nachname),
			-- Wie die sieben Felder ringsum: ein LEERER Exportwert darf den Bestand nicht
			-- loeschen. klasse stand hier als EINZIGES ungeschuetzt da (seit 4219a2e, nie
			-- bewusst entschieden). Dieselbe Bugklasse hat 96c2f8c im Buch-Importer bereits
			-- behoben (autor/verlag/jahr). Heute faengt lusd_parser.go:82 eine leere Klasse
			-- schon beim Parsen ab — aber diese Zeile ist die zweite Tuer zum selben Zustand,
			-- und die Klasse ist die Angabe, an der Ausleihlimit, Mahnweg und Abgaengerlauf
			-- haengen. Ein leerer Wert hier ist nie eine gewollte Aussage.
			klasse       = COALESCE(NULLIF($3, ''), klasse),
			strasse      = COALESCE(NULLIF($4, ''), strasse),
			hausnummer   = COALESCE(NULLIF($5, ''), hausnummer),
			plz          = COALESCE(NULLIF($6, ''), plz),
			ort          = COALESCE(NULLIF($7, ''), ort),
			eltern_email = COALESCE(NULLIF($8, ''), eltern_email),
			ist_abgaenger = false,
			ist_gesperrt = CASE
				WHEN vorname = 'Abgänger' AND nachname LIKE 'Anonymisiert-%' THEN false
				WHEN block_reason LIKE 'Automatisierte Abgänger-Sperre%'
				     AND NOT EXISTS (SELECT 1 FROM ausleihen WHERE schueler_id = $9 AND rueckgabe_am IS NULL)
				     AND NOT EXISTS (SELECT 1 FROM schadensfaelle WHERE schueler_id = $9 AND ist_bezahlt = false)
				THEN false
				ELSE ist_gesperrt END,
			-- block_reason konsistent zu ist_gesperrt setzen: chk_schueler_block_reason
			-- verlangt einen Grund NUR solange ist_gesperrt = true.
			block_reason = CASE
				WHEN vorname = 'Abgänger' AND nachname LIKE 'Anonymisiert-%' THEN NULL
				WHEN block_reason LIKE 'Automatisierte Abgänger-Sperre%'
				     AND NOT EXISTS (SELECT 1 FROM ausleihen WHERE schueler_id = $9 AND rueckgabe_am IS NULL)
				     AND NOT EXISTS (SELECT 1 FROM schadensfaelle WHERE schueler_id = $9 AND ist_bezahlt = false)
				THEN NULL
				WHEN block_reason LIKE 'Automatisierte Abgänger-Sperre%'
				THEN 'Sperre wegen offener Vorgänge'
				ELSE block_reason END,
			aktualisiert_am = NOW()
		WHERE id = $9`,
			rec.Vorname, rec.Nachname, rec.Klasse,
			rec.Strasse, rec.Hausnummer, rec.PLZ, rec.Ort, rec.ElternEmail, ids[i])
	}

	// Die Verbindung bleibt bis zum Close() vom Batch belegt. Erst danach darf der
	// Aufrufer die Transaktion weiterbenutzen (behandleAbgaenger läuft anschließend
	// auf derselben tx) — ein pgx-Aufruf davor quittiert mit "conn busy".
	br := tx.SendBatch(ctx, batch)

	for i := range records {
		if _, err := br.Exec(); err != nil {
			// Close vor dem Rückweg, sonst geht die Verbindung belegt an den
			// Rollback des Aufrufers.
			closeutil.LogClose(br, "LUSD-Batch nach Fehler")
			return fmt.Errorf("schüler %s konnte nicht aktualisiert werden: %w", ids[i], err)
		}
	}

	// Close meldet Fehler, die beim Abräumen der restlichen Ergebnisse auffallen —
	// die darf der Import nicht verschlucken, sonst gilt er als geglückt.
	if err := br.Close(); err != nil {
		return fmt.Errorf("LUSD-Sammelaktualisierung abschließen: %w", err)
	}
	return nil
}

// behandleAbgaenger verarbeitet Schüler, die nicht mehr im Export stehen.
// Mit offenen Ausleihen bleiben Name UND Kontaktdaten erhalten (fürs Mahnwesen und
// die Schadens-Rechnung noch nötig). Ohne offene Ausleihen wird DSGVO-konform
// anonymisiert — dabei werden Adresse und Eltern-E-Mail gelöscht.
func behandleAbgaenger(ctx context.Context, tx pgx.Tx, graduates []StudentDiff, dbStudents map[string]lusdDbStudent) error {
	if len(graduates) == 0 {
		return nil
	}

	gradIDs := make([]string, 0, len(graduates))
	for _, grad := range graduates {
		gradIDs = append(gradIDs, dbStudents[grad.ID].ID)
	}

	// Offene Buch-Vormerkungen der Abgänger löschen: Ein Schüler, der die Schule verlässt,
	// holt keine reservierten Bücher mehr ab; seine wartenden Vormerkungen würden sonst
	// begehrte Titel dauerhaft für die verbliebenen Schüler blockieren. (Die alte
	// BulkSyncLUSD-Pipeline tat dies; beim Rewrite nach lusd_apply.go ging es verloren.)
	if _, err := tx.Exec(ctx,
		"DELETE FROM vormerkungen WHERE schueler_id = ANY($1) AND status = 'wartend'", gradIDs,
	); err != nil {
		return err
	}

	// Offene Ausleihen pre-loaden
	pendingCounts := make(map[string]int, len(gradIDs))
	rows, err := tx.Query(ctx, "SELECT schueler_id, COUNT(*) FROM ausleihen WHERE schueler_id = ANY($1) AND rueckgabe_am IS NULL GROUP BY schueler_id", gradIDs)
	if err != nil {
		return err
	}
	for rows.Next() {
		var sID string
		var count int
		if err := rows.Scan(&sID, &count); err != nil {
			rows.Close()
			return err
		}
		pendingCounts[sID] = count
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// Auch offene (unbezahlte, nicht stornierte) Schadensrechnungen verhindern die
	// Anonymisierung — sonst würden Name und Rechnungsadresse gelöscht und die Schule bliebe
	// auf dem Schaden sitzen, weil sie den Schüler nicht mehr anschreiben kann. storniert_am
	// setzt ist_bezahlt = true (repository/audit_system.go), daher genügt ist_bezahlt = false.
	offeneSchaedenCounts := make(map[string]int, len(gradIDs))
	schaedenRows, err := tx.Query(ctx, "SELECT schueler_id, COUNT(*) FROM schadensfaelle WHERE schueler_id = ANY($1) AND ist_bezahlt = false GROUP BY schueler_id", gradIDs)
	if err != nil {
		return err
	}
	for schaedenRows.Next() {
		var sID string
		var count int
		if err := schaedenRows.Scan(&sID, &count); err != nil {
			schaedenRows.Close()
			return err
		}
		offeneSchaedenCounts[sID] = count
	}
	schaedenRows.Close()
	if err := schaedenRows.Err(); err != nil {
		return err
	}

	for _, sID := range gradIDs {
		pending := pendingCounts[sID]
		offeneSchaeden := offeneSchaedenCounts[sID]
		if pending > 0 || offeneSchaeden > 0 {
			if err := sperreAbgaenger(ctx, tx, sID); err != nil {
				return err
			}
		} else {
			if err := anonymisiereAbgaenger(ctx, tx, sID); err != nil {
				return err
			}
		}
	}

	return nil
}

// sperreAbgaenger markiert einen Abgänger mit offenen Vorgängen (nicht zurückgegebene
// Bücher ODER unbezahlte Schäden) als gesperrt, lässt Name und Kontaktdaten aber
// unangetastet (Mahnung/Rechnung laufen noch).
//
// abgaenger_jahr wird auf das TATSÄCHLICHE Abgangsjahr (jetzt) gesetzt. Ohne das blieb
// es auf dem Default vom Anlegen (Jahr+5, in der Zukunft), und der DSGVO-Cronjob
// (RunGDPRDeleteAbgaenger, Filter abgaenger_jahr < cutoffYear) hätte den Abgänger nach
// der Buchrückgabe NIE erfasst — die PII wäre für immer geblieben.
func sperreAbgaenger(ctx context.Context, tx pgx.Tx, schuelerID string) error {
	// block_reason MUSS gesetzt sein (chk_schueler_block_reason). Ein bereits vorhandener
	// (z. B. manueller) Grund bleibt erhalten — sonst greift der Abgänger-Standardgrund,
	// damit das Personal im Profil sofort sieht, WARUM gesperrt wurde.
	_, err := tx.Exec(ctx,
		`UPDATE schueler SET ist_abgaenger = true, ist_gesperrt = true,
		        block_reason = COALESCE(NULLIF(block_reason, ''), 'Automatisierte Abgänger-Sperre (offene Vorgänge)'),
		        abgaenger_jahr = EXTRACT(YEAR FROM NOW())::int, aktualisiert_am = NOW()
		 WHERE id = $1`,
		schuelerID)
	return err
}

// anonymisiereAbgaenger entfernt personenbezogene Daten (Name, Adresse, E-Mail)
// eines Abgängers ohne offene Vorgänge. Die interne DB-UUID hängt am Nachnamen,
// um Unique-Constraint-Verletzungen zu vermeiden.
func anonymisiereAbgaenger(ctx context.Context, tx pgx.Tx, schuelerID string) error {
	// Verschlüsseltes Passfoto sofort mit der Anonymisierung löschen — nicht erst beim
	// späteren Hard-Purge (ON DELETE CASCADE). Der Foto-Zweck (Wiedererkennung an der Theke)
	// entfällt mit dem Abgang; Name/Adresse werden hier ohnehin geleert, das Foto ist das
	// personenbezogenste verbleibende Datum und gehört mit weg (Datensparsamkeit).
	if _, err := tx.Exec(ctx, "DELETE FROM schueler_fotos WHERE schueler_id = $1", schuelerID); err != nil {
		return err
	}

	anonymisiertName := fmt.Sprintf("Anonymisiert-%s", schuelerID)
	// abgaenger_jahr aufs echte Abgangsjahr setzen — damit der DSGVO-Cronjob den
	// (bereits namens-anonymisierten) Datensatz nach Karenzzeit endgültig entfernt,
	// statt ihn unbegrenzt zu behalten.
	// block_reason wird auf einen festen Text gesetzt (nicht der alte erhalten): Bei der
	// Anonymisierung wird jeglicher personenbezogene Kontext geleert, ein evtl. alter Grund
	// könnte solchen enthalten. chk_schueler_block_reason verlangt zudem einen Grund.
	_, err := tx.Exec(ctx, `
		UPDATE schueler SET
			vorname = 'Abgänger', nachname = $1, klasse = 'ABG',
			strasse = NULL, hausnummer = NULL, plz = NULL, ort = NULL, eltern_email = NULL,
			ist_abgaenger = true, ist_gesperrt = true, block_reason = 'Abgänger anonymisiert',
			abgaenger_jahr = EXTRACT(YEAR FROM NOW())::int, aktualisiert_am = NOW()
		WHERE id = $2`,
		anonymisiertName, schuelerID)
	return err
}
