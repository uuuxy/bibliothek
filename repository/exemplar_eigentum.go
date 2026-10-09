package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"bibliothek/db"
	"bibliothek/pkg/mitteltopf"

	"github.com/jackc/pgx/v5"
)

// Das Eigentum von Hand ändern (docs/OFFEN.md 4.24, Stufe 3, freigegeben am 29.09.2026) — wie
// in Littera „Exemplardaten anpassen": Exemplare eines Titels markieren, einen Wert für alle
// setzen. Ein einzelnes Exemplar ist eine Liste mit einem Eintrag: eine Tür, eine Regel.
//
// Das Eigentum wirkt auf Etikett, Zugangs- und Abgangsbuch und Schadensersatz
// (ExemplarTopfSQL). Deshalb ist der Grund Pflicht, und jede Änderung steht mit altem und neuem
// Wert im Protokoll — wie beim Topf einer Bestellung (BestellMittelDialog).

// ExemplarEigentumListeMax begrenzt eine Änderung. Ein Titel mit mehr Exemplaren wird in
// mehreren Schritten geändert; die Buchakte markiert höchstens, was sie zeigt.
const ExemplarEigentumListeMax = 1000

// ErrEigentumUngueltig ist eine Eingabe, die die Tür nicht annimmt (400).
var ErrEigentumUngueltig = errors.New("eigentum ungültig")

// EigentumAenderung ist eine Änderung für mehrere Exemplare.
type EigentumAenderung struct {
	ExemplarIDs []string
	// Eigentum ist mitteltopf.Land oder mitteltopf.Schultraeger. Leer nimmt die Angabe am Exemplar
	// weg: Dann gilt wieder der Topf der Bestellung, sonst die Faustregel aus dem Titel.
	Eigentum     string
	Grund        string
	BearbeiterID string
}

// SetzeExemplarEigentum setzt das Eigentum der genannten Exemplare und liefert, wie viele sich
// geändert haben. Alle oder keins: Fehlt eine Kennung, ändert sich nichts
// (ErrExemplarNichtGefunden). Ein Exemplar, das schon so steht, wird nicht noch einmal
// geschrieben und nicht protokolliert.
func SetzeExemplarEigentum(ctx context.Context, q DBQueryer, a EigentumAenderung) (int, error) {
	if err := pruefeEigentumAenderung(a); err != nil {
		return 0, err
	}
	quelle := ""
	if a.Eigentum != "" {
		quelle = "hand"
	}
	grund := strings.TrimSpace(a.Grund)

	tx, err := q.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("eigentum: transaktion öffnen: %w", err)
	}
	defer db.SafeRollback(ctx, tx)

	// Wie viele verschiedene Exemplare gewählt sind, zählt die Datenbank — sie liest
	// Groß- und Kleinschreibung einer Kennung als dieselbe (wie LoescheSchlagworte).
	var gewaehlt int
	if err := tx.QueryRow(ctx, `SELECT count(DISTINCT k)::int FROM unnest($1::uuid[]) k`, a.ExemplarIDs).
		Scan(&gewaehlt); err != nil {
		return 0, fmt.Errorf("eigentum: kennungen zählen: %w", err)
	}
	alle, err := sperreEigentumStand(ctx, tx, a.ExemplarIDs)
	if err != nil {
		return 0, err
	}
	if len(alle) != gewaehlt {
		return 0, ErrExemplarNichtGefunden
	}

	geaendert := 0
	for _, s := range alle {
		if s.eigentum == a.Eigentum && s.quelle == quelle {
			continue
		}
		if err := schreibeEigentum(ctx, tx, s, a, quelle, grund); err != nil {
			return 0, err
		}
		geaendert++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("eigentum: commit: %w", err)
	}
	return geaendert, nil
}

// eigentumStand ist das Eigentum eines Exemplars vor der Änderung.
type eigentumStand struct{ id, eigentum, quelle string }

// sperreEigentumStand sperrt die Exemplare in fester Reihenfolge, damit sich zwei Änderungen
// mit überlappender Auswahl nicht verklemmen, und liest dabei den alten Wert fürs Protokoll.
func sperreEigentumStand(ctx context.Context, tx pgx.Tx, exemplarIDs []string) ([]eigentumStand, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, coalesce(eigentum, ''), coalesce(eigentum_quelle, '')
		FROM buecher_exemplare WHERE id = ANY($1::uuid[]) ORDER BY id FOR UPDATE`, exemplarIDs)
	if err != nil {
		return nil, fmt.Errorf("eigentum: sperren: %w", err)
	}
	var alle []eigentumStand
	for rows.Next() {
		var s eigentumStand
		if err := rows.Scan(&s.id, &s.eigentum, &s.quelle); err != nil {
			rows.Close()
			return nil, fmt.Errorf("eigentum: alten stand lesen: %w", err)
		}
		alle = append(alle, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("eigentum: alten stand lesen: %w", err)
	}
	return alle, nil
}

// schreibeEigentum setzt das Eigentum eines Exemplars und schreibt den Eintrag mit altem und
// neuem Wert ins Protokoll.
func schreibeEigentum(ctx context.Context, tx pgx.Tx, s eigentumStand, a EigentumAenderung, quelle, grund string) error {
	tag, err := tx.Exec(ctx, `
		UPDATE buecher_exemplare SET eigentum = NULLIF($2, ''), eigentum_quelle = NULLIF($3, '')
		WHERE id = $1`, s.id, a.Eigentum, quelle)
	if err != nil {
		return fmt.Errorf("eigentum: exemplar %s: %w", s.id, err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("eigentum: exemplar %s: %d zeilen statt 1", s.id, tag.RowsAffected())
	}
	audit := &pgAuditRepository{}
	kontext := "Eigentum von Hand geändert"
	if err := audit.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: "buecher_exemplare", Aktion: "UPDATE", DatensatzID: s.id,
		BearbeiterID: &a.BearbeiterID, Akteur: "USER", Kontext: &kontext,
		Details: map[string]any{
			"eigentum_alt": s.eigentum, "eigentum_quelle_alt": s.quelle,
			"eigentum_neu": a.Eigentum, "grund": grund,
		},
	}); err != nil {
		return fmt.Errorf("eigentum: protokoll: %w", err)
	}
	return nil
}

// pruefeEigentumAenderung prüft die Eingabe, bevor die Datenbank gefragt wird.
func pruefeEigentumAenderung(a EigentumAenderung) error {
	switch {
	case len(a.ExemplarIDs) == 0:
		return fmt.Errorf("%w: kein Exemplar gewählt", ErrEigentumUngueltig)
	case len(a.ExemplarIDs) > ExemplarEigentumListeMax:
		return fmt.Errorf("%w: höchstens %d Exemplare auf einmal", ErrEigentumUngueltig, ExemplarEigentumListeMax)
	case a.Eigentum != "" && !mitteltopf.Gueltig(a.Eigentum):
		return fmt.Errorf("%w: unbekanntes Eigentum %q", ErrEigentumUngueltig, a.Eigentum)
	case strings.TrimSpace(a.Grund) == "":
		return fmt.Errorf("%w: der Grund fehlt", ErrEigentumUngueltig)
	case a.BearbeiterID == "":
		return fmt.Errorf("%w: ohne Bearbeiter", ErrEigentumUngueltig)
	}
	return nil
}
