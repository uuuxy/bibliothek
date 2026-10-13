package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// ProtokollAbfragen schreibt die Einträge des Protokolls und führt die Vorgänge aus, die mit
// ihrem Eintrag in einer Transaktion stehen: das Löschen von Titel, Konto und Leser, das
// Aussondern eines Exemplars und das Erledigen einer Forderung. So gibt es den Vorgang nicht
// ohne seine Spur.
//
// Dass Einträge nur angefügt werden, ist eine Regel des Codes und keine der Datenbank: Kein
// Weg ändert oder löscht einen Eintrag, außer der Tilgung des Personenbezugs
// (TilgeSchuelerSpuren, jobs/cron_dsgvo.go). Ein Trigger, der jede Änderung verbietet, stünde
// der Löschpflicht entgegen (Art. 17 DSGVO) und ließe genau diese Tilgung scheitern.
type ProtokollAbfragen interface {
	// LoescheTitel löscht einen Titel mit seinen Exemplaren und protokolliert es.
	LoescheTitel(ctx context.Context, titelID string, bearbeiterID string) error
	// SondereExemplarAus setzt ein Exemplar auf ausgesondert und protokolliert es.
	SondereExemplarAus(ctx context.Context, exemplarID string, bearbeiterID string) error
	// LoescheKonto löscht ein Konto und protokolliert es.
	LoescheKonto(ctx context.Context, kontoID string, bearbeiterID string) error

	// LegeLeserInPapierkorb legt einen Leser in den Papierkorb, aus dem er sich zurückholen
	// lässt. Seine Daten bleiben; endgültig löscht LoescheLeserEndgueltig.
	LegeLeserInPapierkorb(ctx context.Context, leserID string, bearbeiterID string, grund string) error

	// LoescheLeserEndgueltig entfernt einen Leser aus dem Papierkorb: Die Ausleihen verlieren
	// ihren Leser, die Einträge des Protokolls ihren Personenbezug, erledigte Schadensfälle
	// und die Zeile des Lesers werden gelöscht. Eine offene Ausleihe oder eine unbezahlte
	// Forderung verhindert es.
	LoescheLeserEndgueltig(ctx context.Context, leserID string, bearbeiterID string) error

	// LoescheAbgaengerEndgueltig tut dasselbe für einen Abgänger, der nicht im Papierkorb
	// liegt; der nächtliche Lauf ruft es. Offene Vorgänge verhindern es ebenso.
	LoescheAbgaengerEndgueltig(ctx context.Context, leserID string, bearbeiterID string) error

	// StorniereForderung storniert eine offene Forderung mit Begründung und protokolliert es.
	// Den Betrag liest es aus der Datenbank, der Rufer übergibt ihn nicht.
	StorniereForderung(ctx context.Context, schadensfallID string, bearbeiterID string, grund string) error

	// VerbucheZahlung verbucht die Zahlung einer offenen Forderung und protokolliert es.
	VerbucheZahlung(ctx context.Context, schadensfallID string, bearbeiterID string) error

	// ProtokolliereAusleihe schreibt den Eintrag zur Ausleihe in die Transaktion der Ausleihe,
	// vor deren Commit: Lässt sich die Spur nicht schreiben, gibt es die Ausleihe nicht.
	ProtokolliereAusleihe(ctx context.Context, tx pgx.Tx, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error
	// ProtokolliereRueckgabe schreibt den Eintrag zur Rückgabe in die Transaktion der Rückgabe.
	ProtokolliereRueckgabe(ctx context.Context, tx pgx.Tx, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error

	// ProtokolliereSystemAktion schreibt einen Eintrag für einen Vorgang des Programms selbst,
	// etwa eine automatische Sperre oder einen nächtlichen Lauf.
	ProtokolliereSystemAktion(ctx context.Context, tabelle string, aktion string, kontext string, details map[string]any) error

	// ProtokolliereAdminAktion schreibt einen Eintrag ins Protokoll der Verwaltung (audit_logs).
	ProtokolliereAdminAktion(ctx context.Context, adminID string, aktion string, ip string, details map[string]any) error
}

type pgProtokollAbfragen struct {
	db db.PgxPoolIface
}

// NewProtokollAbfragen liefert die Abfragen des Protokolls an Postgres.
func NewProtokollAbfragen(db db.PgxPoolIface) ProtokollAbfragen {
	return &pgProtokollAbfragen{db: db}
}

// protokollEintrag bündelt die Felder eines Eintrags im Protokoll der Vorgänge (audit_log).
type protokollEintrag struct {
	Tabelle      string
	Aktion       string
	DatensatzID  string
	BearbeiterID *string
	Akteur       string
	Kontext      *string
	Details      map[string]any
}

// ProtokolliereAdminAktion schreibt einen Eintrag ins Protokoll der Verwaltung. Details, die
// sich nicht als JSON schreiben lassen, stehen als leeres Objekt im Eintrag.
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
	_, err := q.Exec(ctx, query, leerAlsNull(adminID), aktion, details, leerAlsNull(ip))
	return err
}

// ProtokolliereSystemAktion schreibt einen Eintrag ohne Bearbeiter, für die nächtlichen Läufe
// und andere Vorgänge des Programms selbst.
func (a *pgProtokollAbfragen) ProtokolliereSystemAktion(ctx context.Context, tabelle string, aktion string, kontext string, details map[string]any) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	// Ein Vorgang des Programms hat keinen Datensatz; die Spalte bekommt die Null-UUID.
	const nullUUID = "00000000-0000-0000-0000-000000000000"

	if err = schreibeProtokollEintrag(ctx, tx, protokollEintrag{
		Tabelle: tabelle, Aktion: aktion, DatensatzID: nullUUID,
		Akteur: "SYSTEM", Kontext: &kontext, Details: details,
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ProtokolliereAusleihe schreibt den Eintrag CHECKOUT.
func (a *pgProtokollAbfragen) ProtokolliereAusleihe(ctx context.Context, tx pgx.Tx, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error {
	return a.protokolliereLeihvorgang(ctx, tx, "CHECKOUT", exemplarID, schuelerID, benutzerID, bearbeiterID)
}

// ProtokolliereRueckgabe schreibt den Eintrag RETURN.
func (a *pgProtokollAbfragen) ProtokolliereRueckgabe(ctx context.Context, tx pgx.Tx, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error {
	return a.protokolliereLeihvorgang(ctx, tx, "RETURN", exemplarID, schuelerID, benutzerID, bearbeiterID)
}

// protokolliereLeihvorgang schreibt den Eintrag in die übergebene Transaktion und öffnet keine
// eigene: Eintrag und Buchung stehen oder fallen zusammen.
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

	return schreibeProtokollEintrag(ctx, tx, protokollEintrag{
		Tabelle: "ausleihen", Aktion: aktion, DatensatzID: exemplarID,
		BearbeiterID: bearbeiterPtr, Akteur: "USER",
		Details: details,
	})
}

// schreibeProtokollEintrag fügt einen Eintrag an audit_log an. db ist der Pool oder die
// Transaktion des Rufers; ein Vorgang und sein Eintrag gelten so zusammen oder gar nicht.
func schreibeProtokollEintrag(ctx context.Context, db Datenbank, e protokollEintrag) error {
	var detailsJSON []byte
	if e.Details != nil {
		var err error
		detailsJSON, err = json.Marshal(e.Details)
		if err != nil {
			return fmt.Errorf("audit details serialization: %w", err)
		}
	}
	_, err := db.Exec(ctx, `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, akteur, kontext, details)
		VALUES ($1, $2, $3::uuid, $4, $5, $6, $7)
	`, e.Tabelle, e.Aktion, e.DatensatzID, e.BearbeiterID, e.Akteur, e.Kontext,
		func() any {
			if detailsJSON == nil {
				return nil
			}
			return string(detailsJSON)
		}())
	return err
}
