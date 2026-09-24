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
// noch Bücher ausgeliehen hat (offene Ausleihen auf seiner Leserzeile).
// Nutzer-sichtbar (409) — deshalb ohne Wörter aus dem Code (audit_users_meldung_test.go).
//
//nolint:staticcheck // ST1005: bewusst großgeschrieben, Endnutzer-Meldung
var ErrUserHasActiveLoans = errors.New("Das Konto hat noch ausgeliehene Bücher — bitte zuerst zurückbuchen")

// DeleteUser löscht einen Systembenutzer endgültig aus der Datenbank und erfasst die Löschung im Audit-Log.
func (r *pgAuditRepository) DeleteUser(ctx context.Context, userID string, bearbeiterID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Snapshot erstellen: Benutzerdaten vor dem Löschen sichern. Die `leser_id` wird
	// mitgelesen, weil sie nach dem DELETE nicht mehr zu finden ist (ON DELETE SET NULL am
	// Konto, und die Zeile selbst ist dann weg).
	var vorname, nachname, email, rolle string
	var leserID *string
	err = tx.QueryRow(ctx,
		`SELECT coalesce(vorname,''), coalesce(nachname,''), coalesce(email,''), coalesce(rolle::text,''), leser_id
		 FROM benutzer WHERE id = $1`,
		userID,
	).Scan(&vorname, &nachname, &email, &rolle, &leserID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to snapshot user for audit: %w", err)
	}

	// Schutz vor verwaisten Ausleihen: Die Bücher hängen an der LESERZEILE des Kontos,
	// und benutzer.leser_id trägt ON DELETE SET NULL. Ein DELETE ließe die Bücher im
	// Status "ausgeliehen" zurück, während die Leserzeile ihren Namen verliert — dauerhaft
	// blockiert und nicht mehr zuordenbar. Deshalb die Löschung verweigern, solange offene
	// Ausleihen bestehen.
	var aktiveAusleihen int
	if err = tx.QueryRow(ctx,
		`SELECT count(*) FROM ausleihen a
		  JOIN benutzer b ON b.leser_id = a.schueler_id
		 WHERE b.id = $1 AND a.rueckgabe_am IS NULL`,
		userID,
	).Scan(&aktiveAusleihen); err != nil {
		return fmt.Errorf("failed to check active loans for user: %w", err)
	}
	if aktiveAusleihen > 0 {
		return fmt.Errorf("%w (%d offen)", ErrUserHasActiveLoans, aktiveAusleihen)
	}

	// 0 Zeilen = unbekannte ID: Der Snapshot oben toleriert ErrNoRows, der Ausleihen-Zähler
	// steht auf 0 — eine unbekannte Benutzer-ID lief glatt durch und hinterließ einen
	// DELETE-Audit-Eintrag über ein Konto, das es nie gab (Phantom-Erfolg-Sweep 31.08.2026;
	// der Schüler-Purge 140 Zeilen weiter unten prüft längst).
	tag, err := tx.Exec(ctx, "DELETE FROM benutzer WHERE id = $1", userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrBenutzerNichtGefunden
	}

	// Die Leserzeile des Kontos geht MIT, wenn sie unberührt ist.
	//
	// Anlass: Eine abgelehnte Zugangsanfrage. Die Selbstanmeldung legt ein Konto ohne
	// Leserzeile an, der Wächter `trg_benutzer_hat_leserzeile` hängt eine frische daran.
	// Wird die Anfrage abgelehnt und das Konto gelöscht, blieb diese Zeile als Waise in der
	// Leserdatei stehen: ohne Ausweis, ohne Vorgänge, mit dem aus der E-Mail-Adresse
	// geratenen Namen (Rasterdurchgang 16.09.2026, OFFEN.md 5.18, Fund 1).
	//
	// „Unberührt" heißt: kein Ausweis und nichts, was an ihr hängt. Hängt doch etwas daran —
	// Bücher, ein Schadensfall, ein Foto, eine Vormerkung —, BLEIBT die Zeile stehen; sie
	// gehört dann einem Menschen, der in der Leserdatei steht, und nicht dem Konto.
	geloescht, grund, err := loescheUnberuehrteLeserzeile(ctx, tx, leserID)
	if err != nil {
		return err
	}

	if err = r.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: "benutzer", Aktion: "DELETE", DatensatzID: userID,
		BearbeiterID: &bearbeiterID, Akteur: "USER",
		Details: map[string]any{"vorname": vorname, "nachname": nachname, "email": email, "rolle": rolle,
			"leserzeile_geloescht": geloescht, "leserzeile_bleibt_wegen": grund},
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// loescheUnberuehrteLeserzeile entfernt die Leserzeile eines gelöschten Kontos, sofern an
// ihr nichts hängt. Liefert zurück, ob gelöscht wurde, und sonst den Grund.
//
// Die Liste der Tabellen kommt aus dem KATALOG der Datenbank, nicht aus dem Go-Code. Das
// ist der Kern und keine Spielerei: An `leser` hängen Fremdschlüssel mit gemischter
// Löschwirkung — RESTRICT bei Ausleihen und Schadensfällen, CASCADE bei Fotos und
// Vormerkungen, SET NULL bei Bescheiden, Nachbuch-Meldungen und Konten. Eine Aufzählung in
// Go hielte das nur bis zur nächsten Tabelle, die jemand anhängt: Bei CASCADE verschwänden
// deren Zeilen still mit, bei SET NULL verlören sie ihren Bezug. Wer eine Tabelle anhängt,
// bekommt die Prüfung hier geschenkt.
//
// Der Ausweis ist der Sonderfall, den keine Fremdschlüssel-Abfrage sieht: Er steht als
// Spalte in der Zeile selbst. Eine Nummer ist vergeben und wird nie recycelt — wer eine
// hat, steht in der Leserdatei.
func loescheUnberuehrteLeserzeile(ctx context.Context, tx pgx.Tx, leserID *string) (bool, string, error) {
	if leserID == nil || *leserID == "" {
		return false, "", nil
	}

	var hatAusweis bool
	if err := tx.QueryRow(ctx,
		`SELECT barcode_id IS NOT NULL AND barcode_id <> '' FROM leser WHERE id = $1`, *leserID,
	).Scan(&hatAusweis); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", nil // schon weg (Papierkorb endgültig geleert)
		}
		return false, "", fmt.Errorf("leserzeile lesen: %w", err)
	}
	if hatAusweis {
		return false, "ausweis", nil
	}

	rows, err := tx.Query(ctx, `
		SELECT c.conrelid::regclass::text, a.attname
		  FROM pg_constraint c
		  JOIN unnest(c.conkey) AS k(attnum) ON true
		  JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = k.attnum
		 WHERE c.contype = 'f' AND c.confrelid = 'leser'::regclass`)
	if err != nil {
		return false, "", fmt.Errorf("fremdschlüssel auf leser lesen: %w", err)
	}
	type kind struct{ tabelle, spalte string }
	var kinder []kind
	for rows.Next() {
		var k kind
		if err := rows.Scan(&k.tabelle, &k.spalte); err != nil {
			rows.Close()
			return false, "", err
		}
		kinder = append(kinder, k)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return false, "", err
	}
	// Kein Fremdschlüssel gefunden heißt nicht „nichts hängt daran", sondern dass die
	// Abfrage nicht getan hat, was sie soll. Dann lieber die Zeile stehen lassen.
	if len(kinder) == 0 {
		return false, "katalog leer", nil
	}

	for _, k := range kinder {
		// Die Namen stammen aus dem Katalog, nicht aus einer Eingabe; `regclass` liefert sie
		// bereits so, wie Postgres sie wieder liest.
		var haengt bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM `+k.tabelle+` WHERE `+k.spalte+` = $1)`, *leserID,
		).Scan(&haengt); err != nil {
			return false, "", fmt.Errorf("%s prüfen: %w", k.tabelle, err)
		}
		if haengt {
			return false, k.tabelle, nil
		}
	}

	tag, err := tx.Exec(ctx, `DELETE FROM leser WHERE id = $1`, *leserID)
	if err != nil {
		return false, "", fmt.Errorf("leserzeile löschen: %w", err)
	}
	// Null Zeilen heißt: Die Zeile war zwischen Prüfung und Löschung schon weg. Kein Fehler,
	// aber auch kein „gelöscht" — der Audit-Eintrag soll nicht behaupten, was nicht geschah.
	if tag.RowsAffected() == 0 {
		return false, "schon weg", nil
	}
	return true, "", nil
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

	// Snapshot erstellen: Daten für das Audit-Log vor dem Löschen sichern.
	//
	// Gelesen wird die TABELLE `leser`, nicht die Sicht `schueler`: Die zeigt nur Schüler,
	// und für einen Kollegen lief dieser ganze Weg bis zum 16.09.2026 ins Leere — der
	// Snapshot blieb leer, das UPDATE traf null Zeilen, und der Handler antwortete „nicht
	// gefunden". Deshalb stand der Löschknopf in seiner Akte gar nicht erst (docs/OFFEN.md
	// 5.16 C).
	var vorname, nachname, klasse, barcodeID, art string
	var abgaengerJahr int
	err = tx.QueryRow(ctx,
		`SELECT coalesce(vorname,''), coalesce(nachname,''), coalesce(klasse,''),
		        coalesce(barcode_id,''), coalesce(abgaenger_jahr, 0), art
		 FROM leser WHERE id = $1`,
		studentID,
	).Scan(&vorname, &nachname, &klasse, &barcodeID, &abgaengerJahr, &art)
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
	//
	// COALESCE auf deleted_at (12.09.2026, Register): Liegt die Zeile schon im
	// Papierkorb, bleibt der ERSTE Zeitpunkt stehen. Er ist die Uhr der Anonymisierung
	// (repository.PredikatAnonymisierung, 180 Tage); ein zweites Löschen schob sie um
	// die ganze bereits abgelaufene Zeit nach hinten — still, denn die Antwort lautete
	// beide Male „success". Löschen bleibt damit wiederholbar, ohne die Frist zu
	// verlängern.
	tag, err := tx.Exec(ctx, `UPDATE leser
		SET deleted_at = COALESCE(deleted_at, CURRENT_TIMESTAMP),
		    ist_gesperrt = true,
		    block_reason = COALESCE(NULLIF(btrim(block_reason), ''), 'Systematisch gelöscht')
		WHERE id = $1`, studentID)
	if err != nil {
		return fmt.Errorf("soft-deleting student: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %s", ErrLeserNichtGefunden, studentID)
	}

	// Beim Kollegium geht das KONTO mit — Entschieden am 16.09.2026: „wenn ein kollege gelöscht
	// wird dann wird alles gelöscht."
	//
	// Es bleibt nicht als abgeschaltete Hülle stehen, und das aus einem nachprüfbaren
	// Grund: Die Adresse ist der Schlüssel der Anmeldung (`benutzer_email_unique`). Ein
	// stehengebliebenes Konto hielte sie besetzt, und wer die Person danach neu anlegt,
	// bekäme „Unter … steht bereits ein Zugang" — über einen Eintrag, der im Papierkorb
	// liegt und nirgends zu sehen ist. Die Spuren gehen dabei nicht verloren: Jeder
	// Fremdschlüssel auf `benutzer` trägt ON DELETE SET NULL, die Protokollzeilen bleiben
	// also stehen und verlieren nur den Verweis.
	//
	// Wird die Leserzeile aus dem Papierkorb zurückgeholt, kommt sie OHNE Zugang zurück.
	// Den legt man an, indem man in der Akte die Schul-E-Mail nachträgt
	// (api/student_schul_email.go) — derselbe Weg wie beim Altbestand.
	var kontenGeloescht int64
	if art != "schueler" {
		kontoTag, kontoErr := tx.Exec(ctx, `DELETE FROM benutzer WHERE leser_id = $1`, studentID)
		if kontoErr != nil {
			return fmt.Errorf("deleting account of reader: %w", kontoErr)
		}
		kontenGeloescht = kontoTag.RowsAffected()
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
			"art":            art,
			// Wie viele Zugänge dabei erloschen sind — ohne diese Zahl liesse sich später
			// nicht mehr sagen, ob die Person je einen hatte.
			"konten_geloescht": kontenGeloescht,
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
		return fmt.Errorf("%w: %d offene Ausleihe(n)", ErrLoeschenBlockiert, offeneAusleihen)
	}
	var offeneSchaeden int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false`, studentID).Scan(&offeneSchaeden); err != nil {
		return fmt.Errorf("checking unpaid damages: %w", err)
	}
	if offeneSchaeden > 0 {
		return fmt.Errorf("%w: %d unbezahlte(r) Schadensfall/-fälle", ErrLoeschenBlockiert, offeneSchaeden)
	}
	return nil
}

// entferneSchuelerPIIUndLoesche ist die gemeinsame DSGVO-Löschung für PurgeStudent
// (manueller Papierkorb) und PurgeAbgaenger (Cronjob): Ausleihhistorie anonymisieren
// (schueler_id = NULL — beide Entleiher NULL ist laut check_loan_borrower erlaubt),
// bezahlte Schadensfälle löschen, Schüler-Audit-Details anonymisieren (dabei fallen
// auch die Vormerkungen, siehe TilgeSchuelerSpuren), Datensatz entfernen (der
// FK-CASCADE räumt dann nur noch das Foto), Löschung ohne PII protokollieren.
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
	// `leser` statt der Sicht `schueler`: Sonst bliebe die Zeile eines Kollegen beim
	// endgültigen Löschen stehen — mit anonymisierter Historie, aber vorhandenem Namen.
	tag, err := tx.Exec(ctx, `DELETE FROM leser WHERE id = $1`, studentID)
	if err != nil {
		return fmt.Errorf("deleting student: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %s", ErrLeserNichtGefunden, studentID)
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

// ErrLoeschenBlockiert meldet: Die Löschung ist nicht möglich, WEIL noch etwas offen ist —
// eine Ausleihe, ein unbezahlter Schaden, oder die Zeile liegt gar nicht im Papierkorb.
//
// Der Handler braucht den Unterschied (17.09.2026, OFFEN.md 5.6): Bis hierher beantwortete
// er JEDEN Fehler des Purge mit 409 „Konflikt" — auch einen Verbindungsabbruch oder einen
// kaputten Constraint. Ein Serverfehler, der sich als Konflikt ausgibt, schickt die
// Bibliothek los, ein Problem zu suchen, das es nicht gibt, und verdeckt das echte
// (Bugklasse „Fehler-Kollaps", docs/sweeps.md).
//
//nolint:staticcheck // ST1005: die Meldung steht so vor dem Menschen.
var ErrLoeschenBlockiert = errors.New("Endgültiges Löschen blockiert")

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
	err = tx.QueryRow(ctx, `SELECT deleted_at IS NOT NULL FROM leser WHERE id = $1`, studentID).Scan(&imPapierkorb)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %s", ErrLeserNichtGefunden, studentID)
	}
	if err != nil {
		return fmt.Errorf("checking trash state: %w", err)
	}
	if !imPapierkorb {
		return fmt.Errorf("%w: der Datensatz liegt nicht im Papierkorb — erst löschen, dann endgültig entfernen", ErrLoeschenBlockiert)
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

// SpurenExecutor ist der kleinste gemeinsame Nenner von pgx.Tx und dem Pool. Query
// steht mit drin, seit eine Tilgung nicht mehr nur schreibt, sondern das Ergebnis ihres
// eigenen DELETE braucht (Vormerkungen, siehe vormerkung_nachruecken.go).
type SpurenExecutor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
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
	// schritt steht STATT sql, wenn eine Tilgung mehr ist als eine Anweisung — sie
	// bleibt trotzdem ein Eintrag DIESER Liste, damit Purge, LUSD-Abgang und Cron
	// weiterhin dasselbe tun.
	schritt func(ctx context.Context, ex SpurenExecutor, schuelerIDs []string) (int64, error)
}

// Exec führt die Anweisung für die gegebene Schüler-Menge aus und meldet die Zahl
// der betroffenen Zeilen.
func (st SpurTilgung) Exec(ctx context.Context, ex SpurenExecutor, schuelerIDs []string, grund string) (int64, error) {
	if st.schritt != nil {
		return st.schritt(ctx, ex, schuelerIDs)
	}
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
		// Zugangskonto (Migration 123): Das KONTO bleibt — es ist die Anmeldung samt
		// Rechten und gehoert nicht dem Leser, sondern der Anlage. Was faellt, ist die
		// Verknuepfung: Nach der Tilgung soll keine Anmeldung mehr auf diese getilgte
		// Person zeigen. Der Fremdschluessel steht auf ON DELETE SET NULL und greift erst,
		// wenn die Leserzeile ganz verschwindet; diese Zeile raeumt den Fall davor
		// (Anonymisierung, Zeile bleibt). Idempotent: NULL bleibt NULL.
		Beschreibung: "benutzer (Verknuepfung zur Leserzeile)",
		sql: `UPDATE benutzer SET leser_id = NULL, aktualisiert_am = NOW()
			WHERE leser_id = ANY($1::uuid[])`,
	},
	{
		// Nachbuch-Meldungen (Migration 117): Die Meldung bleibt als Vorgang — Barcode,
		// Ergebnis, Grund —, der Personenbezug fällt: beide Schüler-Spalten auf NULL. Die
		// Fremdschlüssel stehen auf ON DELETE SET NULL, sobald der Datensatz verschwindet;
		// diese Zeile räumt die Anonymisierung davor (anonymized_at, Zeile bleibt).
		// Idempotent: NULL bleibt NULL.
		Beschreibung: "nachbuch_meldungen (Ausleiher und Vorbesitzer)",
		sql: `UPDATE nachbuch_meldungen
			SET ausleiher_schueler_id = CASE WHEN ausleiher_schueler_id = ANY($1::uuid[]) THEN NULL ELSE ausleiher_schueler_id END,
			    vorbesitzer_schueler_id = CASE WHEN vorbesitzer_schueler_id = ANY($1::uuid[]) THEN NULL ELSE vorbesitzer_schueler_id END
			WHERE ausleiher_schueler_id = ANY($1::uuid[]) OR vorbesitzer_schueler_id = ANY($1::uuid[])`,
	},
	{
		// Schadensersatz-Bescheid (Migration 110): Der Brief BLEIBT als Beleg — über seine
		// Referenznummer werden Zahlungen zugeordnet, und Rechnungsunterlagen liegen
		// Jahre. Was fällt, ist der Personenbezug: der Empfänger-Snapshot (Anrede, Name,
		// Anschrift zum Briefdatum). Der Fremdschlüssel wird von ON DELETE SET NULL
		// geleert, sobald der Schülerdatensatz verschwindet; diese Zeile räumt den
		// Klartext, der sonst im JSONB stehen bliebe. Idempotent: leeres Objekt bleibt leer.
		Beschreibung: "schadensersatz_bescheide (Empfänger-Snapshot)",
		sql: `UPDATE schadensersatz_bescheide
			SET empfaenger_snapshot = '{}'::jsonb
			WHERE schueler_id = ANY($1::uuid[]) AND empfaenger_snapshot <> '{}'::jsonb`,
	},
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
		// Die staatliche LUSD-ID (LUSD_ID_NACHGETRAGEN) und die Ausweis-Barcodes
		// (SCHUELER_ZUSAMMENGEFUEHRT, Tresen-Auskunft): Nur die PII-Schlüssel fallen;
		// Aktion, Zeit und schueler_id (nach Anonymisierung ein Pseudonym) bleiben für
		// die Rechenschaftspflicht erhalten. Der Barcode gehört dazu, weil die anonyme
		// Hülle einen ANON-Barcode bekommt, damit die physische Karte nicht mehr aufgeht —
		// im Verwaltungsprotokoll stand die alte Nummer sonst bis zu 24 Monate neben der
		// UUID (Rasterdurchgang 02.09.2026).
		// Der Grund einer Sperre ebenso: Die Anonymisierung ersetzt block_reason, weil er
		// andere Personen nennen kann, und die Sperr-Tür schreibt ihn als grund ins Protokoll
		// (LESER_GESPERRT, LESER_ENTSPERRT), das Übergehen an der Theke bis zum 24.09.2026 als
		// reason (OVERRIDE_BLOCK). Rasterdurchgang 24.09.2026,
		// api/sperrgrund_tilgung_pg_test.go.
		Beschreibung: "audit_logs (LUSD-ID, Barcodes, Sperrgrund)",
		sql: `UPDATE audit_logs
			SET details = details - 'lusd_id' - 'barcode' - 'aufgeloest_barcode' - 'grund' - 'reason'
			WHERE (details ? 'lusd_id' OR details ? 'barcode' OR details ? 'aufgeloest_barcode'
			       OR details ? 'grund' OR details ? 'reason')
			  AND details->>'schueler_id' = ANY($1::text[])`,
	},
	{
		// Schadensfall-Freitext (entschieden am 16.09.2026, OFFEN.md 4.15): Der FALL bleibt
		// als Beleg stehen — Betrag, Datum, bezahlt oder storniert; daran hängen
		// Kassenbuch und Bescheid. Was fällt, ist die Geschichte dazu: „Buch im Bus liegen
		// gelassen, Mutter angerufen" ist Personenbezug, der die Anonymisierung sonst
		// überlebt.
		//
		// Nur bei ERLEDIGTEN Fällen (ist_bezahlt deckt bezahlt UND storniert ab, siehe
		// audit_system.go). Eine offene Forderung behält ihre Begründung: Sie wird noch
		// gebraucht — jemand muss sie einziehen, erklären oder stornieren können. Dass ein
		// Schüler mit offener Forderung überhaupt anonymisiert wird, verhindert das
		// Prädikat (loeschfristen.go); diese Bedingung ist der Gürtel dazu.
		//
		// Idempotent über `beschreibung <> ''`.
		Beschreibung: "schadensfaelle (Freitext erledigter Fälle)",
		sql: `UPDATE schadensfaelle
			SET beschreibung = ''
			WHERE schueler_id = ANY($1::uuid[]) AND ist_bezahlt = true AND beschreibung <> ''`,
	},
	{
		// Vormerkungen: die Freitext-Notiz kann personenbezogen sein, und die Vormerkung
		// eines gelöschten/anonymisierten Schülers ist funktionslos. Beim Purge räumt sie
		// auch der FK-CASCADE — hier steht sie trotzdem, damit der Cron-Pfad (Schüler
		// lebt als anonymisierte Hülle weiter) dieselbe Liste fahren kann.
		//
		// Kein reines DELETE: Lag für das Kind schon ein Exemplar abholbereit, muss es
		// an den nächsten Wartenden gehen — sonst bleibt das Buch auf dem Abholregal
		// liegen und die Schlange rückt nie nach (06.09.2026).
		Beschreibung: "vormerkungen (gelöscht, Warteschlange nachgerückt)",
		schritt:      loescheVormerkungenUndRuecktNach,
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
