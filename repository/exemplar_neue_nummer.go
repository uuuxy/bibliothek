package repository

import (
	"context"
	"errors"
	"fmt"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// protokollAktionNeueNummer kennzeichnet in details->>'action' den Eintrag zu einem Exemplar,
// das eine neue Nummer bekam; details->>'barcode_id' ist die alte, details->>'neue_nummer' die
// neue.
const protokollAktionNeueNummer = "neue_nummer"

// kontextNeueNummer steht im Feld kontext des Eintrags.
const kontextNeueNummer = "Exemplar bekam für ein neues Etikett eine kurze Nummer"

// errNeueNummerOhneBearbeiter lehnt eine neue Nummer ab, die keine Person nennt.
var errNeueNummerOhneBearbeiter = errors.New("neue nummer ohne bearbeiter")

// gesperrteNummer ist ein Exemplar mit seiner Nummer vor der Änderung.
type gesperrteNummer struct{ id, nummer string }

// VergebeNeueNummern gibt den Exemplaren mit den genannten Nummern je eine neue aus barcode_seq
// (ZieheFreieExemplarBarcodes) und liefert zu jeder alten Nummer die neue. Alle oder keins: Trägt
// eine der Nummern kein Exemplar, ändert sich nichts (ErrExemplarNichtGefunden).
//
// Das Etikett steht danach aus (etikett_gedruckt = false), weil die neue Nummer noch an keinem
// Buch klebt; bricht der Druck ab, führt die Liste „Fehlende Etiketten" zu dem Exemplar. Je
// Exemplar steht ein Eintrag mit beiden Nummern und der Person im Protokoll. Die Theken-Auskunft
// liest ihn nicht, sie sucht dort nur gelöschte Exemplare (FindeThekenExemplare): Ein altes
// Etikett, das noch auftaucht, führt nur über eine Abfrage an audit_log zu seinem Exemplar.
func VergebeNeueNummern(ctx context.Context, q Datenbank, nummern []string, bearbeiterID string) (map[string]string, error) {
	if bearbeiterID == "" {
		return nil, errNeueNummerOhneBearbeiter
	}
	tx, err := q.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("neue nummer: transaktion öffnen: %w", err)
	}
	defer db.Verwerfe(ctx, tx)

	gewaehlt := make(map[string]bool, len(nummern))
	for _, n := range nummern {
		gewaehlt[n] = true
	}
	gesperrt, err := sperreNummern(ctx, tx, nummern)
	if err != nil {
		return nil, err
	}
	if len(gewaehlt) == 0 || len(gesperrt) != len(gewaehlt) {
		return nil, ErrExemplarNichtGefunden
	}

	neue, err := ZieheFreieExemplarBarcodes(ctx, tx, len(gesperrt))
	if err != nil {
		return nil, fmt.Errorf("neue nummer: %w", err)
	}
	zuordnung := make(map[string]string, len(gesperrt))
	for i, e := range gesperrt {
		if err := schreibeNeueNummer(ctx, tx, e, neue[i], bearbeiterID); err != nil {
			return nil, err
		}
		zuordnung[e.nummer] = neue[i]
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("neue nummer: commit: %w", err)
	}
	return zuordnung, nil
}

// schreibeNeueNummer setzt die neue Nummer eines Exemplars, nimmt den Vermerk „Etikett gedruckt"
// zurück und schreibt den Eintrag mit beiden Nummern ins Protokoll.
func schreibeNeueNummer(ctx context.Context, tx pgx.Tx, e gesperrteNummer, neu, bearbeiterID string) error {
	tag, err := tx.Exec(ctx, `
		UPDATE buecher_exemplare
		SET barcode_id = $1, etikett_gedruckt = false, aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $2`, neu, e.id)
	if err != nil {
		return fmt.Errorf("neue nummer: %s umschreiben: %w", e.nummer, err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("neue nummer: %s: %d zeilen statt 1", e.nummer, tag.RowsAffected())
	}
	kontext := kontextNeueNummer
	if err := schreibeProtokollEintrag(ctx, tx, protokollEintrag{
		Tabelle: "buecher_exemplare", Aktion: "UPDATE", DatensatzID: e.id,
		BearbeiterID: &bearbeiterID, Akteur: "USER", Kontext: &kontext,
		Details: map[string]any{"action": protokollAktionNeueNummer, "barcode_id": e.nummer, "neue_nummer": neu},
	}); err != nil {
		return fmt.Errorf("neue nummer: protokoll: %w", err)
	}
	return nil
}

// sperreNummern sperrt die Exemplare mit den Nummern in fester Reihenfolge, damit sich zwei
// Aufträge mit überlappender Auswahl nicht verklemmen. Hat ein anderer Auftrag die Nummer
// inzwischen ersetzt, fehlt ihr Exemplar in der Antwort.
func sperreNummern(ctx context.Context, q Datenbank, nummern []string) ([]gesperrteNummer, error) {
	rows, err := q.Query(ctx, `
		SELECT id::text, barcode_id FROM buecher_exemplare
		WHERE barcode_id = ANY($1) ORDER BY id FOR UPDATE`, nummern)
	if err != nil {
		return nil, fmt.Errorf("neue nummer: sperren: %w", err)
	}
	defer rows.Close()
	var gesperrt []gesperrteNummer
	for rows.Next() {
		var e gesperrteNummer
		if err := rows.Scan(&e.id, &e.nummer); err != nil {
			return nil, fmt.Errorf("neue nummer: sperren: %w", err)
		}
		gesperrt = append(gesperrt, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("neue nummer: sperren: %w", err)
	}
	return gesperrt, nil
}
