package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// Die Spur des Aussonderns (docs/invarianten.md, Frage 19): Das Abgangsbuch nennt Datum und
// Grund eines ausgesonderten Exemplars, der Eintrag hier nennt die Person. Geschrieben wird er
// auf den Wegen, die ein Exemplar ohne eigene Protokollzeile aussondern: der Status in der
// Buchakte, ein kleinerer Bestand in „Buch bearbeiten", der Abschluss einer Inventur.
//
// Die Zustandsnotiz steht nicht im Eintrag: Sie ist Freitext und kann eine Person nennen, und
// ein Protokolleintrag bleibt bis zur Audit-Aufbewahrung.

// AuditAktionAusgesondert kennzeichnet den Eintrag in details->>'action'.
const AuditAktionAusgesondert = "ausgesondert"

// Die Wege stehen im Feld kontext des Eintrags.
const (
	AussonderungsWegStatus            = "Exemplar ausgesondert: Status in der Buchakte"
	AussonderungsWegBestandskorrektur = "Exemplar ausgesondert: Bestand in „Buch bearbeiten“ verkleinert"
	AussonderungsWegInventur          = "Exemplar ausgesondert: Abschluss einer Inventur"
)

// ErrAussonderungOhneBearbeiter lehnt ein Aussondern ab, das keine Person nennt.
var ErrAussonderungOhneBearbeiter = errors.New("aussondern ohne bearbeiter")

// ProtokolliereAussonderung schreibt je Exemplar einen Eintrag mit Bearbeiter, Weg und Grund.
// Den Grund liest die Anweisung vom Exemplar: Im Eintrag steht, was das Abgangsbuch zeigt.
// Der Aufrufer ruft sie nach dem Aussondern in derselben Transaktion; ein Exemplar, das nicht
// ausgesondert ist, bekommt keinen Eintrag, und der Aufruf scheitert. zusatz ergänzt die
// Einzelheiten, etwa um die Inventur.
func ProtokolliereAussonderung(ctx context.Context, q DBQueryer, exemplarIDs []string, bearbeiterID, weg string, zusatz map[string]any) error {
	if bearbeiterID == "" {
		return ErrAussonderungOhneBearbeiter
	}
	if len(exemplarIDs) == 0 {
		return nil
	}
	if zusatz == nil {
		zusatz = map[string]any{}
	}
	zusatzJSON, err := json.Marshal(zusatz)
	if err != nil {
		return fmt.Errorf("aussonderung protokollieren: %w", err)
	}
	tag, err := q.Exec(ctx, `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, akteur, kontext, details)
		SELECT 'buecher_exemplare', 'UPDATE', e.id, $2::uuid, 'USER', $3,
		       jsonb_build_object('action', $4::text, 'grund', e.aussonderung_grund) || $5::jsonb
		FROM buecher_exemplare e
		WHERE e.id = ANY($1::uuid[]) AND e.ist_ausgesondert`,
		exemplarIDs, bearbeiterID, weg, AuditAktionAusgesondert, string(zusatzJSON))
	if err != nil {
		return fmt.Errorf("aussonderung protokollieren: %w", err)
	}
	if tag.RowsAffected() != int64(len(exemplarIDs)) {
		return fmt.Errorf("aussonderung protokollieren: %d einträge für %d exemplare", tag.RowsAffected(), len(exemplarIDs))
	}
	return nil
}
