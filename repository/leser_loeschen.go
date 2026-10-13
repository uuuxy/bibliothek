package repository

import (
	"bibliothek/db"
	"bibliothek/pkg/leserart"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// LegeLeserInPapierkorb verschiebt einen Schüler in den Papierkorb (Soft-Delete): deleted_at
// wird gesetzt und der Datensatz gesperrt. Die personenbezogenen Daten (PII) bleiben
// zunächst erhalten, damit ein versehentliches Löschen per RestoreStudentHandler
// rückgängig gemacht werden kann.
//
// ACHTUNG: Dies ist KEINE DSGVO-Löschung — die PII (Name, Adresse, Ausleihhistorie,
// Audit-Logs, Schadensfälle) bleibt bestehen. Die endgültige Anonymisierung/Löschung
// macht LoescheLeserEndgueltig (endgültiges Entfernen aus dem Papierkorb).
//
// Der Aufruf ist über die HTTP-API (bearbeiterID = Benutzer-UUID) und den Cronjob
// (bearbeiterID = "" → SYSTEM) möglich.
func (a *pgProtokollAbfragen) LegeLeserInPapierkorb(ctx context.Context, leserID string, bearbeiterID string, grund string) error {
	tx, err := a.db.Begin(ctx)
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
		leserID,
	).Scan(&vorname, &nachname, &klasse, &barcodeID, &abgaengerJahr, &art)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to snapshot student for audit: %w", err)
	}

	// Soft-Delete durchführen anstatt physisch zu löschen.
	//
	// COALESCE(NULLIF(...)) statt blindem Überschreiben (31.08.2026): Ein BESTEHENDER
	// Sperrgrund bleibt stehen — dasselbe Muster wie in SperreAbgaenger (lusd_import.go) und
	// schuljahreswechsel.go. Vorher war dieser Schreiber der einzige, der den Grund
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
		WHERE id = $1`, leserID)
	if err != nil {
		return fmt.Errorf("soft-deleting student: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %s", ErrLeserNichtGefunden, leserID)
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
	// (api/leser/student_schul_email.go) — derselbe Weg wie beim Altbestand.
	var kontenGeloescht int64
	if !leserart.IstSchueler(art) {
		if kontenGeloescht, err = a.loescheKontenDerLeserzeile(ctx, tx, leserID, bearbeiterID); err != nil {
			return err
		}
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
	if err = a.schreibeEintrag(ctx, tx, protokollEintrag{
		Tabelle: "schueler", Aktion: "UPDATE", DatensatzID: leserID,
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

// pruefeKeineOffenenVorgaenge verhindert das endgültige Löschen, solange Bücher
// draußen sind oder eine Gebühr offen ist — in beiden Fällen läuft noch ein
// berechtigtes Interesse, das die Aufbewahrung rechtfertigt.
func pruefeKeineOffenenVorgaenge(ctx context.Context, tx pgx.Tx, leserID string) error {
	var offeneAusleihen int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM ausleihen WHERE schueler_id = $1 AND rueckgabe_am IS NULL`, leserID).Scan(&offeneAusleihen); err != nil {
		return fmt.Errorf("checking open loans: %w", err)
	}
	if offeneAusleihen > 0 {
		return fmt.Errorf("%w: %d offene Ausleihe(n)", ErrLoeschenBlockiert, offeneAusleihen)
	}
	var offeneSchaeden int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false`, leserID).Scan(&offeneSchaeden); err != nil {
		return fmt.Errorf("checking unpaid damages: %w", err)
	}
	if offeneSchaeden > 0 {
		return fmt.Errorf("%w: %d unbezahlte(r) Schadensfall/-fälle", ErrLoeschenBlockiert, offeneSchaeden)
	}
	return nil
}

// tilgeSpurenUndLoescheLeser ist die gemeinsame DSGVO-Löschung für LoescheLeserEndgueltig
// (manueller Papierkorb) und LoescheAbgaengerEndgueltig (Cronjob): Ausleihhistorie anonymisieren
// (schueler_id = NULL — beide Entleiher NULL ist laut check_loan_borrower erlaubt),
// bezahlte Schadensfälle löschen, Schüler-Audit-Details anonymisieren (dabei fallen
// auch die Vormerkungen, siehe TilgeSchuelerSpuren), Datensatz entfernen (der
// FK-CASCADE räumt dann nur noch das Foto), Löschung ohne PII protokollieren.
func (a *pgProtokollAbfragen) tilgeSpurenUndLoescheLeser(ctx context.Context, tx pgx.Tx, leserID, bearbeiterID, kontextText string) error {
	if _, err := tx.Exec(ctx, `UPDATE ausleihen SET schueler_id = NULL WHERE schueler_id = $1`, leserID); err != nil {
		return fmt.Errorf("anonymizing loans: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM schadensfaelle WHERE schueler_id = $1`, leserID); err != nil {
		return fmt.Errorf("deleting damages: %w", err)
	}
	if err := TilgeSchuelerSpuren(ctx, tx, leserID, "DSGVO-Löschung"); err != nil {
		return err
	}
	// `leser` statt der Sicht `schueler`: Sonst bliebe die Zeile eines Kollegen beim
	// endgültigen Löschen stehen — mit anonymisierter Historie, aber vorhandenem Namen.
	tag, err := tx.Exec(ctx, `DELETE FROM leser WHERE id = $1`, leserID)
	if err != nil {
		return fmt.Errorf("deleting student: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %s", ErrLeserNichtGefunden, leserID)
	}

	var bearbeiterPtr *string
	akteur := "SYSTEM"
	if bearbeiterID != "" {
		akteur = "USER"
		bearbeiterPtr = &bearbeiterID
	}
	if err := a.schreibeEintrag(ctx, tx, protokollEintrag{
		Tabelle: "schueler", Aktion: "DELETE", DatensatzID: leserID,
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

// LoescheLeserEndgueltig entfernt einen im Papierkorb liegenden Schüler endgültig und
// DSGVO-konform. Nur aus dem Papierkorb; offene Ausleihen/unbezahlte Schäden blockieren.
func (a *pgProtokollAbfragen) LoescheLeserEndgueltig(ctx context.Context, leserID string, bearbeiterID string) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Nur bereits weichgelöschte Schüler (Papierkorb) dürfen endgültig entfernt werden.
	var imPapierkorb bool
	err = tx.QueryRow(ctx, `SELECT deleted_at IS NOT NULL FROM leser WHERE id = $1`, leserID).Scan(&imPapierkorb)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %s", ErrLeserNichtGefunden, leserID)
	}
	if err != nil {
		return fmt.Errorf("checking trash state: %w", err)
	}
	if !imPapierkorb {
		return fmt.Errorf("%w: der Datensatz liegt nicht im Papierkorb — erst löschen, dann endgültig entfernen", ErrLoeschenBlockiert)
	}

	if err = pruefeKeineOffenenVorgaenge(ctx, tx, leserID); err != nil {
		return err
	}
	if err = a.tilgeSpurenUndLoescheLeser(ctx, tx, leserID, bearbeiterID, "DSGVO-Löschung (Purge)"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// LoescheAbgaengerEndgueltig entfernt einen ehemaligen Schüler (Abgänger) endgültig und
// DSGVO-konform — der Cronjob-Pendant zu LoescheLeserEndgueltig. Anders als LoescheLeserEndgueltig ist der
// Abgänger NICHT im Papierkorb (ist_abgaenger=true, deleted_at IS NULL); die Auswahl
// (Karenzzeit, ist_abgaenger) trifft der Aufrufer. Die Blockade bei offenen Vorgängen
// wird hier dennoch geprüft — als Sicherheitsnetz gegen einen Race zwischen Auswahl
// und Löschung.
func (a *pgProtokollAbfragen) LoescheAbgaengerEndgueltig(ctx context.Context, leserID string, bearbeiterID string) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	if err = pruefeKeineOffenenVorgaenge(ctx, tx, leserID); err != nil {
		return err
	}
	if err = a.tilgeSpurenUndLoescheLeser(ctx, tx, leserID, bearbeiterID, "DSGVO-Löschung (Abgänger-Cronjob)"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
