package repository

import (
	"bibliothek/db"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrUserHasActiveLoans signalisiert, dass ein Benutzer nicht gelöscht werden kann, weil er
// noch aktive (nicht zurückgegebene) Handapparat-Ausleihen hat. Nutzer-sichtbar (409).
//
//nolint:staticcheck // ST1005: bewusst großgeschrieben, Endnutzer-Meldung
var ErrUserHasActiveLoans = errors.New("Benutzer hat noch aktive Handapparat-Ausleihen — bitte zuerst zurückbuchen")

// DeleteUser löscht einen Systembenutzer endgültig aus der Datenbank und erfasst die Löschung im Audit-Log.
func (r *pgAuditRepository) DeleteUser(ctx context.Context, userID string, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Snapshot erstellen: Benutzerdaten vor dem Löschen sichern
	var vorname, nachname, email, rolle string
	err = tx.QueryRow(ctx,
		`SELECT coalesce(vorname,''), coalesce(nachname,''), coalesce(email,''), coalesce(rolle::text,'')
		 FROM benutzer WHERE id = $1`,
		userID,
	).Scan(&vorname, &nachname, &email, &rolle)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to snapshot user for audit: %w", err)
	}

	// Schutz vor verwaisten Handapparat-Ausleihen: Das Schema hat auf
	// ausleihen.ausleiher_benutzer_id ein ON DELETE SET NULL. Ein DELETE würde die Bücher
	// eines Lehrers also im Status "ausgeliehen" zurücklassen, aber an NULL (niemand)
	// gebunden — dauerhaft blockiert und nicht mehr zuordenbar. Deshalb die Löschung
	// verweigern, solange der Benutzer noch aktive Ausleihen hat.
	var aktiveAusleihen int
	if err = tx.QueryRow(ctx,
		`SELECT count(*) FROM ausleihen WHERE ausleiher_benutzer_id = $1 AND rueckgabe_am IS NULL`,
		userID,
	).Scan(&aktiveAusleihen); err != nil {
		return fmt.Errorf("failed to check active loans for user: %w", err)
	}
	if aktiveAusleihen > 0 {
		return fmt.Errorf("%w (%d offen)", ErrUserHasActiveLoans, aktiveAusleihen)
	}

	if _, err = tx.Exec(ctx, "DELETE FROM benutzer WHERE id = $1", userID); err != nil {
		return err
	}

	if err = r.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: "benutzer", Aktion: "DELETE", DatensatzID: userID,
		BearbeiterID: &bearbeiterID, Akteur: "USER",
		Details: map[string]any{"vorname": vorname, "nachname": nachname, "email": email, "rolle": rolle},
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// DeleteStudent verschiebt einen Schüler in den Papierkorb (Soft-Delete): deleted_at
// wird gesetzt und der Datensatz gesperrt. Die personenbezogenen Daten (PII) bleiben
// zunächst erhalten, damit ein versehentliches Löschen per RestoreStudentHandler
// rückgängig gemacht werden kann.
//
// ACHTUNG: Dies ist KEINE DSGVO-Löschung — die PII (Name, Adresse, Ausleihhistorie,
// Audit-Logs, Schadensfälle) bleibt bestehen. Die endgültige Anonymisierung/Löschung
// macht PurgeStudent (endgültiges Entfernen aus dem Papierkorb).
//
// Der Aufruf ist über die HTTP-API (bearbeiterID = Benutzer-UUID) und den Cronjob
// (bearbeiterID = "" → SYSTEM) möglich.
func (r *pgAuditRepository) DeleteStudent(ctx context.Context, studentID string, bearbeiterID string, grund string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Snapshot erstellen: Daten für das Audit-Log vor dem Löschen sichern
	var vorname, nachname, klasse, barcodeID string
	var abgaengerJahr int
	err = tx.QueryRow(ctx,
		`SELECT coalesce(vorname,''), coalesce(nachname,''), coalesce(klasse,''),
		        coalesce(barcode_id,''), coalesce(abgaenger_jahr, 0)
		 FROM schueler WHERE id = $1`,
		studentID,
	).Scan(&vorname, &nachname, &klasse, &barcodeID, &abgaengerJahr)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to snapshot student for audit: %w", err)
	}

	// Soft-Delete durchführen anstatt physisch zu löschen.
	//
	// COALESCE(NULLIF(...)) statt blindem Überschreiben (31.08.2026): Ein BESTEHENDER
	// Sperrgrund bleibt stehen — dasselbe Muster wie in api/lusd_apply.go und
	// api/student_promotion.go. Vorher war dieser Schreiber der einzige, der den Grund
	// plattmachte, und der Restore erkannte die Zeile dann an seinem eigenen Marker als
	// bloße Lösch-Sperre: Er setzte ist_gesperrt=false und block_reason=NULL, während
	// is_manually_blocked=true stehen blieb — Verstoß gegen chk_schueler_block_reason
	// (gesperrt ⇒ Grund nicht leer), also 23514 → 500. Ein manuell gesperrter Schüler
	// ließ sich nach dem Löschen nie wiederherstellen, und sein echter Grund war weg.
	tag, err := tx.Exec(ctx, `UPDATE schueler
		SET deleted_at = CURRENT_TIMESTAMP,
		    ist_gesperrt = true,
		    block_reason = COALESCE(NULLIF(btrim(block_reason), ''), 'Systematisch gelöscht')
		WHERE id = $1`, studentID)
	if err != nil {
		return fmt.Errorf("soft-deleting student: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("student %s not found", studentID)
	}

	// Akteur ermitteln (entweder manueller Admin-User oder automatische System-Bereinigung)
	var akteur string
	var bearbeiterPtr *string
	if bearbeiterID != "" {
		akteur = "USER"
		bearbeiterPtr = &bearbeiterID
	} else {
		akteur = "SYSTEM"
	}

	kontext := "Soft-Delete Routine"

	// Protokolleintrag schreiben
	if err = r.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: "schueler", Aktion: "UPDATE", DatensatzID: studentID,
		BearbeiterID: bearbeiterPtr, Akteur: akteur, Kontext: &kontext,
		Details: map[string]any{
			"vorname":        vorname,
			"nachname":       nachname,
			"klasse":         klasse,
			"barcode_id":     barcodeID,
			"abgaenger_jahr": abgaengerJahr,
			"grund":          grund,
			"geloescht_am":   time.Now().UTC().Format(time.RFC3339),
			"action":         "soft_delete",
		},
	}); err != nil {
		return fmt.Errorf("writing audit log: %w", err)
	}

	return tx.Commit(ctx)
}

// blockiereBeiOffenenVorgaengen verhindert das endgültige Löschen, solange Bücher
// draußen sind oder eine Gebühr offen ist — in beiden Fällen läuft noch ein
// berechtigtes Interesse, das die Aufbewahrung rechtfertigt.
func blockiereBeiOffenenVorgaengen(ctx context.Context, tx pgx.Tx, studentID string) error {
	var offeneAusleihen int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM ausleihen WHERE schueler_id = $1 AND rueckgabe_am IS NULL`, studentID).Scan(&offeneAusleihen); err != nil {
		return fmt.Errorf("checking open loans: %w", err)
	}
	if offeneAusleihen > 0 {
		return fmt.Errorf("endgültiges Löschen blockiert: %d offene Ausleihe(n)", offeneAusleihen)
	}
	var offeneSchaeden int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false`, studentID).Scan(&offeneSchaeden); err != nil {
		return fmt.Errorf("checking unpaid damages: %w", err)
	}
	if offeneSchaeden > 0 {
		return fmt.Errorf("endgültiges Löschen blockiert: %d unbezahlte(r) Schadensfall/-fälle", offeneSchaeden)
	}
	return nil
}

// entferneSchuelerPIIUndLoesche ist die gemeinsame DSGVO-Löschung für PurgeStudent
// (manueller Papierkorb) und PurgeAbgaenger (Cronjob): Ausleihhistorie anonymisieren
// (schueler_id = NULL — beide Entleiher NULL ist laut check_loan_borrower erlaubt),
// bezahlte Schadensfälle löschen, Schüler-Audit-Details anonymisieren, Datensatz
// entfernen (FK-CASCADE räumt Fotos + Vormerkungen), Löschung ohne PII protokollieren.
func (r *pgAuditRepository) entferneSchuelerPIIUndLoesche(ctx context.Context, tx pgx.Tx, studentID, bearbeiterID, kontextText string) error {
	if _, err := tx.Exec(ctx, `UPDATE ausleihen SET schueler_id = NULL WHERE schueler_id = $1`, studentID); err != nil {
		return fmt.Errorf("anonymizing loans: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM schadensfaelle WHERE schueler_id = $1`, studentID); err != nil {
		return fmt.Errorf("deleting damages: %w", err)
	}
	if err := TilgeSchuelerSpuren(ctx, tx, studentID, "DSGVO-Löschung"); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM schueler WHERE id = $1`, studentID)
	if err != nil {
		return fmt.Errorf("deleting student: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("student %s not found", studentID)
	}

	var bearbeiterPtr *string
	akteur := "SYSTEM"
	if bearbeiterID != "" {
		akteur = "USER"
		bearbeiterPtr = &bearbeiterID
	}
	if err := r.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: "schueler", Aktion: "DELETE", DatensatzID: studentID,
		BearbeiterID: bearbeiterPtr, Akteur: akteur, Kontext: &kontextText,
		Details: map[string]any{"action": "purge", "geloescht_am": time.Now().UTC().Format(time.RFC3339)},
	}); err != nil {
		return fmt.Errorf("writing purge audit log: %w", err)
	}
	return nil
}

// PurgeStudent entfernt einen im Papierkorb liegenden Schüler endgültig und
// DSGVO-konform. Nur aus dem Papierkorb; offene Ausleihen/unbezahlte Schäden blockieren.
func (r *pgAuditRepository) PurgeStudent(ctx context.Context, studentID string, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Nur bereits weichgelöschte Schüler (Papierkorb) dürfen endgültig entfernt werden.
	var imPapierkorb bool
	err = tx.QueryRow(ctx, `SELECT deleted_at IS NOT NULL FROM schueler WHERE id = $1`, studentID).Scan(&imPapierkorb)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("student %s not found", studentID)
	}
	if err != nil {
		return fmt.Errorf("checking trash state: %w", err)
	}
	if !imPapierkorb {
		return fmt.Errorf("student %s ist nicht im Papierkorb — erst löschen, dann endgültig entfernen", studentID)
	}

	if err = blockiereBeiOffenenVorgaengen(ctx, tx, studentID); err != nil {
		return err
	}
	if err = r.entferneSchuelerPIIUndLoesche(ctx, tx, studentID, bearbeiterID, "DSGVO-Löschung (Purge)"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// PurgeAbgaenger entfernt einen ehemaligen Schüler (Abgänger) endgültig und
// DSGVO-konform — der Cronjob-Pendant zu PurgeStudent. Anders als PurgeStudent ist der
// Abgänger NICHT im Papierkorb (ist_abgaenger=true, deleted_at IS NULL); die Auswahl
// (Karenzzeit, ist_abgaenger) trifft der Aufrufer. Die Blockade bei offenen Vorgängen
// wird hier dennoch geprüft — als Sicherheitsnetz gegen einen Race zwischen Auswahl
// und Löschung.
func (r *pgAuditRepository) PurgeAbgaenger(ctx context.Context, studentID string, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	if err = blockiereBeiOffenenVorgaengen(ctx, tx, studentID); err != nil {
		return err
	}
	if err = r.entferneSchuelerPIIUndLoesche(ctx, tx, studentID, bearbeiterID, "DSGVO-Löschung (Abgänger-Cronjob)"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// SpurenExecutor ist der kleinste gemeinsame Nenner von pgx.Tx und dem Pool.
type SpurenExecutor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// SpurTilgung ist EINE Anweisung der Spuren-Tilgung, parametrisiert über eine
// Schüler-MENGE. Die Liste darunter ist die einzige Quelle dafür, welche Spuren eines
// Schülers in den Neben-Tabellen stehen — Purge/LUSD-Abgang (ein Schüler, in der Tx,
// Abbruch beim ersten Fehler) und der nächtliche Cron (alle anonymisierten, am Pool,
// weiter beim Fehler) fahren DIESELBEN Statements und unterscheiden sich nur in
// Menge und Fehlerpolitik.
//
// Anlass (31.08.2026): Cron und Purge pflegten die Liste getrennt; dem Cron fehlte
// genau die Lesehistorie — der Klarname (details->>'entleiher') überlebte die
// Anonymisierung (jobs/dsgvo_spuren_paarung_pg_test.go, am alten Stand rot gesehen).
type SpurTilgung struct {
	Beschreibung string
	sql          string // $1 = Schüler-IDs als text[]; $2 = Grund (nur brauchtGrund)
	brauchtGrund bool
}

// Exec führt die Anweisung für die gegebene Schüler-Menge aus und meldet die Zahl
// der betroffenen Zeilen.
func (st SpurTilgung) Exec(ctx context.Context, ex SpurenExecutor, schuelerIDs []string, grund string) (int64, error) {
	args := []any{schuelerIDs}
	if st.brauchtGrund {
		args = append(args, grund)
	}
	tag, err := ex.Exec(ctx, st.sql, args...)
	return tag.RowsAffected(), err
}

// SpurTilgungen liefert die Statement-Liste für den set-basierten Cron
// (jobs/cron_dsgvo.go). Purge und LUSD-Pfad gehen über TilgeSchuelerSpuren.
func SpurTilgungen() []SpurTilgung { return spurTilgungen }

var spurTilgungen = []SpurTilgung{
	{
		// Datensatz-Historie: DeleteStudent legt Vor-/Nachname, Klasse und Barcode in
		// details ab; das ganze Objekt wird durch den Anonymisierungs-Marker ersetzt.
		// Idempotent über den Marker.
		Beschreibung: "audit_log (Datensatz-Historie, tabelle='schueler')",
		sql: `UPDATE audit_log
			SET details = jsonb_build_object('anonymisiert', true, 'grund', $2::text)
			WHERE tabelle = 'schueler' AND datensatz_id = ANY($1::uuid[])
			  AND (details IS NULL OR NOT (details ? 'anonymisiert'))`,
		brauchtGrund: true,
	},
	{
		// Lesehistorie: dieselben Zeilen, die die Art.-15-Auskunft dem Schüler zurechnet
		// (api/dsgvo_auskunft.go: tabelle='ausleihen' AND details->>'schueler_id' = id).
		// Die Buchungshistorie selbst BLEIBT (Nachweis, dass ein Exemplar unterwegs war),
		// nur ihr Personenbezug fällt — dieselben Schlüssel, die auch die
		// Lesehistorie-Befristung entfernt (jobs/cron_dsgvo_lesehistorie.go).
		Beschreibung: "audit_log (Lesehistorie, tabelle='ausleihen')",
		sql: `UPDATE audit_log
			SET details = details - 'schueler_id' - 'entleiher'
			WHERE tabelle = 'ausleihen' AND details->>'schueler_id' = ANY($1::text[])`,
	},
	{
		// Die staatliche LUSD-ID (LUSD_ID_NACHGETRAGEN). Nur der PII-Schlüssel fällt;
		// Aktion, Zeit und schueler_id (nach Anonymisierung ein Pseudonym) bleiben für
		// die Rechenschaftspflicht erhalten.
		Beschreibung: "audit_logs (LUSD-ID)",
		sql: `UPDATE audit_logs
			SET details = details - 'lusd_id'
			WHERE details ? 'lusd_id' AND details->>'schueler_id' = ANY($1::text[])`,
	},
	{
		// Vormerkungen: die Freitext-Notiz kann personenbezogen sein, und die Vormerkung
		// eines gelöschten/anonymisierten Schülers ist funktionslos. Beim Purge räumt sie
		// auch der FK-CASCADE — hier stehen sie trotzdem, damit der Cron-Pfad (Schüler
		// lebt als anonymisierte Hülle weiter) dieselbe Liste fahren kann.
		Beschreibung: "vormerkungen",
		sql:          `DELETE FROM vormerkungen WHERE schueler_id = ANY($1::uuid[])`,
	},
}

// TilgeSchuelerSpuren entfernt die Personendaten EINES Schülers aus den Neben-Tabellen —
// gemeinsamer Schritt von LUSD-Abgänger-Anonymisierung (api/lusd_apply.go) und Purge.
// Historie der Lücken: Bis 22.08.2026 hatte nur der Cron-Pfad alle damaligen Statements
// (A3: LUSD-ID überlebte 24 Monate); bis 31.08.2026 fehlte hier die Lesehistorie, danach
// fehlte sie dem Cron — seither ist spurTilgungen die eine Liste für beide.
// Idempotent; muss VOR dem DELETE des Schülers laufen (audit_logs hängt nur per
// details->>'schueler_id' am Schüler, nicht per FK).
func TilgeSchuelerSpuren(ctx context.Context, ex SpurenExecutor, schuelerID, grund string) error {
	for _, st := range spurTilgungen {
		if _, err := st.Exec(ctx, ex, []string{schuelerID}, grund); err != nil {
			return fmt.Errorf("tilgung %s: %w", st.Beschreibung, err)
		}
	}
	return nil
}
