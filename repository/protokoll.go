package repository

import (
	"bibliothek/db"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ProtokollAbfragen verwaltet die Protokollierung (Audit-Logs) sowie administrative
// Löschungen von Systemressourcen unter Einhaltung der Datenschutzvorgaben (DSGVO).
//
// Append-Only ist hier eine KONVENTION, keine DB-erzwungene Regel: Kein Codepfad ändert
// oder löscht Audit-Einträge — mit einer bewussten Ausnahme, der DSGVO-PII-Tilgung
// (tilgeSpurenUndLoescheLeser, jobs/cron_dsgvo.go). Der frühere Trigger, der jedes
// UPDATE/DELETE hart blockierte (Migration 003), wurde mit Migration 083 entfernt: Er
// stand im Widerspruch zur Löschpflicht (Art. 17) und hätte genau diese Tilgung
// scheitern lassen. Wer echte Manipulationssicherheit braucht, setzt sie über eine
// privilegierte Löschausnahme um, nicht über ein pauschales Änderungsverbot.
type ProtokollAbfragen interface {
	// LoescheTitel protokolliert die administrative Löschung eines Buchtitels.
	LoescheTitel(ctx context.Context, titelID string, bearbeiterID string) error
	// SondereExemplarAus protokolliert die administrative Löschung eines konkreten Buchexemplars.
	SondereExemplarAus(ctx context.Context, exemplarID string, bearbeiterID string) error
	// LoescheKonto protokolliert die administrative Löschung eines Systembenutzers.
	LoescheKonto(ctx context.Context, kontoID string, bearbeiterID string) error

	// LegeLeserInPapierkorb verschiebt einen Schüler in den Papierkorb (Soft-Delete, wiederherstellbar).
	// Die PII bleibt erhalten — die endgültige DSGVO-Anonymisierung macht LoescheLeserEndgueltig.
	LegeLeserInPapierkorb(ctx context.Context, leserID string, bearbeiterID string, grund string) error

	// LoescheLeserEndgueltig entfernt einen bereits im Papierkorb liegenden Schüler endgültig und
	// DSGVO-konform: Ausleihhistorie anonymisiert, Audit-Logs anonymisiert, bezahlte
	// Schadensfälle gelöscht, Schüler-Datensatz entfernt. Offene Ausleihen oder
	// unbezahlte Schadensfälle blockieren die Löschung (liefern einen Fehler).
	LoescheLeserEndgueltig(ctx context.Context, leserID string, bearbeiterID string) error

	// LoescheAbgaengerEndgueltig ist das Cronjob-Pendant für ehemalige Schüler (nicht im Papierkorb,
	// sondern ist_abgaenger=true). Gleiche DSGVO-Löschung wie LoescheLeserEndgueltig; offene
	// Vorgänge blockieren ebenfalls.
	LoescheAbgaengerEndgueltig(ctx context.Context, leserID string, bearbeiterID string) error

	// StorniereForderung storniert eine ausstehende Gebühr mit zwingender Begründung
	// und protokolliert den Vorgang. Der Betrag wird transaktionssicher aus der
	// Datenbank gelesen, nie vom Aufrufer übergeben.
	StorniereForderung(ctx context.Context, schadensfallID string, bearbeiterID string, grund string) error

	// VerbucheZahlung verbucht die Zahlung einer ausstehenden Gebühr und protokolliert den Vorgang.
	VerbucheZahlung(ctx context.Context, schadensfallID string, bearbeiterID string) error

	// ProtokolliereAusleihe protokolliert die Ausleihe eines Exemplars an einen Schüler oder Lehrer —
	// IN der Transaktion der Ausleihe, vor deren Commit. Bis zum 07.09.2026 lief der
	// Eintrag nach dem Commit in eigener Transaktion; brach die Verbindung dazwischen ab,
	// galt die Ausleihe ohne Revisionsspur, und nur eine Logzeile sagte es (Register B).
	// Jetzt gilt: Kann die Spur nicht geschrieben werden, gibt es die Ausleihe nicht.
	ProtokolliereAusleihe(ctx context.Context, tx pgx.Tx, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error
	// ProtokolliereRueckgabe protokolliert die Rückgabe eines Exemplars, ebenfalls in der Transaktion der Rückgabe.
	ProtokolliereRueckgabe(ctx context.Context, tx pgx.Tx, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error

	// ProtokolliereSystemAktion protokolliert systemgesteuerte Batch-Prozesse (z. B. automatische Sperrungen oder Bereinigungen).
	ProtokolliereSystemAktion(ctx context.Context, tabelle string, aktion string, kontext string, details map[string]any) error

	// ProtokolliereAdminAktion protokolliert kritische Admin-Eingriffe in der Tabelle audit_logs.
	ProtokolliereAdminAktion(ctx context.Context, adminID string, aktion string, ip string, details map[string]any) error
}

// pgProtokollAbfragen implementiert das ProtokollAbfragen für PostgreSQL.
type pgProtokollAbfragen struct {
	db db.PgxPoolIface
}

// NewProtokollAbfragen erzeugt eine neue Instanz des PostgreSQL-basierten Audit-Repositorys.
func NewProtokollAbfragen(db db.PgxPoolIface) ProtokollAbfragen {
	return &pgProtokollAbfragen{db: db}
}

// schreibeEintrag ist die zentrale Hilfsfunktion, die alle Logeinträge in die Tabelle `audit_log` schreibt.
// Die Kapselung erzwingt ein konsistentes Schema; sie schreibt nur (kein UPDATE/DELETE) — das ist die
// Append-Only-Konvention (nicht per Trigger erzwungen, siehe ProtokollAbfragen-Doku und Migration 083).
// protokollEintrag bündelt die fachlichen Felder eines Audit-Log-Eintrags (ohne die
// Infrastruktur-Parameter ctx/tx), damit schreibeEintrag nicht neun Einzelargumente führt.
type protokollEintrag struct {
	Tabelle      string
	Aktion       string
	DatensatzID  string
	BearbeiterID *string
	Akteur       string
	Kontext      *string
	Details      map[string]any
}

func (a *pgProtokollAbfragen) schreibeEintrag(ctx context.Context, tx pgx.Tx, e protokollEintrag) error {
	var detailsJSON []byte
	if e.Details != nil {
		var err error
		detailsJSON, err = json.Marshal(e.Details)
		if err != nil {
			return fmt.Errorf("audit details serialization: %w", err)
		}
	}

	const q = `
		INSERT INTO audit_log
		  (tabelle, aktion, datensatz_id, bearbeiter_id, akteur, kontext, details)
		VALUES ($1, $2, $3::uuid, $4, $5, $6, $7)
	`
	_, err := tx.Exec(ctx, q,
		e.Tabelle, e.Aktion, e.DatensatzID,
		e.BearbeiterID, e.Akteur, e.Kontext,
		func() interface{} {
			if detailsJSON == nil {
				return nil
			}
			return string(detailsJSON)
		}(),
	)
	return err
}

// ProtokolliereAdminAktion schreibt einen Eintrag in die Tabelle audit_logs für systemweite oder kritische Admin-Eingriffe.
func (a *pgProtokollAbfragen) ProtokolliereAdminAktion(ctx context.Context, adminID string, aktion string, ip string, details map[string]any) error {
	var detailsJSON []byte
	if details != nil {
		var err error
		detailsJSON, err = json.Marshal(details)
		if err != nil {
			detailsJSON = []byte("{}")
		}
	} else {
		detailsJSON = []byte("{}")
	}
	return SchreibeAdminProtokoll(ctx, a.db, adminID, aktion, ip, string(detailsJSON))
}

// SchreibeAdminProtokoll schreibt einen Eintrag in audit_logs, dessen Details als JSON-Text
// vorliegen; ProtokolliereAdminAktion nimmt sie als Tabelle und geht denselben Weg. Ein leerer
// Bearbeiter und eine leere Adresse stehen als NULL in der Zeile.
func SchreibeAdminProtokoll(ctx context.Context, q Datenbank, adminID, aktion, ip, details string) error {
	query := `
		INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse, zeitstempel)
		VALUES ($1, $2, $3::jsonb, $4, CURRENT_TIMESTAMP)
	`
	_, err := q.Exec(ctx, query, leererStringAlsNull(adminID), aktion, details, leererStringAlsNull(ip))
	return err
}

// ProtokolliereSystemAktion schreibt einen Audit-Eintrag für eine rein systemgesteuerte Aktion (ohne Bearbeiter-ID).
// Dies wird typischerweise von Cronjobs, DSGVO-Routinebereinigungen oder Datensicherungen verwendet.
func (a *pgProtokollAbfragen) ProtokolliereSystemAktion(ctx context.Context, tabelle string, aktion string, kontext string, details map[string]any) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Systemaktionen verwenden eine Standard-Null-UUID als Datensatz-ID
	const nullUUID = "00000000-0000-0000-0000-000000000000"

	if err = a.schreibeEintrag(ctx, tx, protokollEintrag{
		Tabelle: tabelle, Aktion: aktion, DatensatzID: nullUUID,
		Akteur: "SYSTEM", Kontext: &kontext, Details: details,
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ProtokolliereAusleihe schreibt einen neuen Ausleiheintrag (CHECKOUT) in das Audit-Log.
func (a *pgProtokollAbfragen) ProtokolliereAusleihe(ctx context.Context, tx pgx.Tx, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error {
	return a.protokolliereLeihvorgang(ctx, tx, "CHECKOUT", exemplarID, schuelerID, benutzerID, bearbeiterID)
}

// ProtokolliereRueckgabe schreibt einen neuen Rückgabeeintrag (RETURN) in das Audit-Log.
func (a *pgProtokollAbfragen) ProtokolliereRueckgabe(ctx context.Context, tx pgx.Tx, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error {
	return a.protokolliereLeihvorgang(ctx, tx, "RETURN", exemplarID, schuelerID, benutzerID, bearbeiterID)
}

// protokolliereLeihvorgang schreibt den Ausleih-/Rückgabe-Eintrag in die ÜBERGEBENE Transaktion —
// keine eigene: Eintrag und Buchung stehen oder fallen zusammen.
func (a *pgProtokollAbfragen) protokolliereLeihvorgang(ctx context.Context, tx pgx.Tx, aktion, exemplarID, schuelerID, benutzerID, bearbeiterID string) error {
	var bearbeiterPtr *string
	if bearbeiterID != "" {
		bearbeiterPtr = &bearbeiterID
	}

	details := map[string]any{
		"exemplar_id": exemplarID,
		"zeitpunkt":   time.Now().UTC().Format(time.RFC3339),
	}
	if schuelerID != "" {
		details["schueler_id"] = schuelerID
	}
	if benutzerID != "" {
		details["benutzer_id"] = benutzerID
	}

	return a.schreibeEintrag(ctx, tx, protokollEintrag{
		Tabelle: "ausleihen", Aktion: aktion, DatensatzID: exemplarID,
		BearbeiterID: bearbeiterPtr, Akteur: "USER",
		Details: details,
	})
}
