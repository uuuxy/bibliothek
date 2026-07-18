package api

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// wendeLusdAenderungenAn führt den zweiten Durchlauf aus: bestehende Schüler
// aktualisieren (Klasse + Kontaktdaten) und Neuzugänge anlegen.
func wendeLusdAenderungenAn(ctx context.Context, tx pgx.Tx, records []parsedStudentRow, dbStudents map[string]lusdDbStudent) error {
	barcodeCounter := 0
	for _, rec := range records {
		if rec.LusdID == "" {
			continue
		}
		if dbRec, exists := dbStudents[rec.LusdID]; exists {
			if err := aktualisiereBestandsschueler(ctx, tx, rec, dbRec.ID); err != nil {
				return err
			}
			continue
		}
		barcodeCounter++
		if err := legeNeuenSchuelerAn(ctx, tx, rec, barcodeCounter); err != nil {
			return err
		}
	}
	return nil
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
	_, err := tx.Exec(ctx, `
		UPDATE schueler SET
			vorname      = COALESCE(NULLIF($1, ''), vorname),
			nachname     = COALESCE(NULLIF($2, ''), nachname),
			klasse       = $3,
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
		rec.Strasse, rec.Hausnummer, rec.PLZ, rec.Ort, rec.ElternEmail, id)
	return err
}

// behandleAbgaenger verarbeitet Schüler, die nicht mehr im Export stehen.
// Mit offenen Ausleihen bleiben Name UND Kontaktdaten erhalten (fürs Mahnwesen und
// die Schadens-Rechnung noch nötig). Ohne offene Ausleihen wird DSGVO-konform
// anonymisiert — dabei werden Adresse und Eltern-E-Mail gelöscht.
func behandleAbgaenger(ctx context.Context, tx pgx.Tx, graduates []StudentDiff, dbStudents map[string]lusdDbStudent) error {
	for _, grad := range graduates {
		dbRec := dbStudents[grad.ID]
		if err := verarbeiteEinenAbgaenger(ctx, tx, dbRec.ID); err != nil {
			return err
		}
	}
	return nil
}

// verarbeiteEinenAbgaenger sperrt oder anonymisiert einen einzelnen Abgänger und räumt
// dessen offene Vormerkungen ab.
func verarbeiteEinenAbgaenger(ctx context.Context, tx pgx.Tx, schuelerID string) error {
	// Offene Buch-Vormerkungen des Abgängers löschen: Ein Schüler, der die Schule verlässt,
	// holt keine reservierten Bücher mehr ab; seine wartenden Vormerkungen würden sonst
	// begehrte Titel dauerhaft für die verbliebenen Schüler blockieren. (Die alte
	// BulkSyncLUSD-Pipeline tat dies; beim Rewrite nach lusd_apply.go ging es verloren.)
	if _, err := tx.Exec(ctx,
		"DELETE FROM vormerkungen WHERE schueler_id = $1 AND status = 'wartend'", schuelerID,
	); err != nil {
		return err
	}

	var pending int
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM ausleihen WHERE schueler_id = $1 AND rueckgabe_am IS NULL", schuelerID).Scan(&pending); err != nil {
		return err
	}

	// Auch offene (unbezahlte, nicht stornierte) Schadensrechnungen verhindern die
	// Anonymisierung — sonst würden Name und Rechnungsadresse gelöscht und die Schule bliebe
	// auf dem Schaden sitzen, weil sie den Schüler nicht mehr anschreiben kann. storniert_am
	// setzt ist_bezahlt = true (repository/audit_system.go), daher genügt ist_bezahlt = false.
	var offeneSchaeden int
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false", schuelerID).Scan(&offeneSchaeden); err != nil {
		return err
	}

	if pending > 0 || offeneSchaeden > 0 {
		return sperreAbgaenger(ctx, tx, schuelerID)
	}
	return anonymisiereAbgaenger(ctx, tx, schuelerID)
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
