package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bibliothek/db"
	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5"
)

// Der LMF-Plan (Migration 096): Rückgabe- und Ausgabetermine je Klasse — die Excel-
// Tabelle der Schule als Tabelle im System. Ein Termin hat Datum, Stunde, Art und
// Vermerk und 0..n Klassen aus dem Vokabular; „Bücher setzen" ist ein Termin ohne
// Klasse, „6F1/6F2" einer mit zwei.

// Die zwei Arten eines Termins: Rückgabe (setzt die Frist der Klasse) und Ausgabe.
const (
	LmfTerminRueckgabe = "rueckgabe"
	LmfTerminAusgabe   = "ausgabe"
)

// LmfTermin ist eine Zeile des Plans, so wie Oberfläche und PDF sie lesen.
type LmfTermin struct {
	ID      string   `json:"id"`
	Datum   string   `json:"datum"` // YYYY-MM-DD
	Stunde  int      `json:"stunde"`
	Art     string   `json:"art"`
	Klassen []string `json:"klassen"`
	Vermerk string   `json:"vermerk"`
}

// SchuljahrBeginn liefert den 1. August des Schuljahres, in dem t liegt (Hessen:
// 1. August bis 31. Juli) — in der Schulzeitzone, weil daraus Kalendertage werden.
func SchuljahrBeginn(t time.Time) time.Time {
	t = t.In(schulzeit.Zone())
	jahr := t.Year()
	if t.Month() < time.August {
		jahr--
	}
	return time.Date(jahr, time.August, 1, 0, 0, 0, 0, schulzeit.Zone())
}

// LmfTerminRepository liest und schreibt den Plan.
type LmfTerminRepository struct {
	db db.PgxPoolIface
}

// NewLmfTerminRepository baut das Repository über dem Pool.
func NewLmfTerminRepository(pool db.PgxPoolIface) *LmfTerminRepository {
	return &LmfTerminRepository{db: pool}
}

// ListLmfTermine liefert die Termine ab einem Datum (einschließlich), nach Datum und
// Stunde sortiert; ab = Nullzeit liefert alle. Die Klassen kommen sortiert mit, damit
// „6F1/6F2" in Oberfläche und PDF gleich aussieht.
func (r *LmfTerminRepository) ListLmfTermine(ctx context.Context, ab time.Time) ([]LmfTermin, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.id, to_char(t.datum, 'YYYY-MM-DD'), t.stunde, t.art, t.vermerk,
		       COALESCE((SELECT array_agg(k.klasse ORDER BY klassen_normkey(k.klasse))
		                 FROM lmf_termin_klassen k WHERE k.termin_id = t.id), '{}')
		FROM lmf_termine t
		WHERE $1::date IS NULL OR t.datum >= $1::date
		ORDER BY t.datum, t.stunde, t.id`, nullbaresDatum(ab))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	termine := []LmfTermin{}
	for rows.Next() {
		var t LmfTermin
		if err := rows.Scan(&t.ID, &t.Datum, &t.Stunde, &t.Art, &t.Vermerk, &t.Klassen); err != nil {
			return nil, err
		}
		termine = append(termine, t)
	}
	return termine, rows.Err()
}

// nullbaresDatum macht aus der Nullzeit ein SQL-NULL, damit „alle" ohne zweites
// Statement geht.
func nullbaresDatum(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// SaveLmfTermin legt einen Termin an (ID leer) oder schreibt ihn um (ID gesetzt) —
// Kopf und Klassen in einer Transaktion, die Klassen vollständig ersetzt. Liefert die
// gespeicherte Zeile mit ID und den kanonisierten Klassennamen zurück.
func (r *LmfTerminRepository) SaveLmfTermin(ctx context.Context, t LmfTermin) (LmfTermin, error) {
	datum, err := time.Parse("2006-01-02", t.Datum)
	if err != nil {
		return t, fmt.Errorf("datum: %w", err)
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return t, err
	}
	defer db.SafeRollback(ctx, tx)

	if t.ID == "" {
		err = tx.QueryRow(ctx, `
			INSERT INTO lmf_termine (datum, stunde, art, vermerk)
			VALUES ($1, $2, $3, $4) RETURNING id`, datum, t.Stunde, t.Art, t.Vermerk).Scan(&t.ID)
	} else {
		var tag pgconnTag
		tag, err = tx.Exec(ctx, `
			UPDATE lmf_termine SET datum = $1, stunde = $2, art = $3, vermerk = $4
			WHERE id = $5`, datum, t.Stunde, t.Art, t.Vermerk, t.ID)
		if err == nil && tag.RowsAffected() == 0 {
			err = pgx.ErrNoRows
		}
	}
	if err != nil {
		return t, err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM lmf_termin_klassen WHERE termin_id = $1`, t.ID); err != nil {
		return t, err
	}
	kanonisch := make([]string, 0, len(t.Klassen))
	for _, k := range t.Klassen {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		// RETURNING liefert den vom Vokabular-Trigger kanonisierten Namen („5f1" → „05F1").
		// ON CONFLICT: dieselbe Klasse zweimal in einer Zeile ist einmal.
		var name string
		err := tx.QueryRow(ctx, `
			INSERT INTO lmf_termin_klassen (termin_id, klasse) VALUES ($1, $2)
			ON CONFLICT DO NOTHING RETURNING klasse`, t.ID, k).Scan(&name)
		if err == pgx.ErrNoRows {
			continue
		}
		if err != nil {
			return t, err
		}
		kanonisch = append(kanonisch, name)
	}
	t.Klassen = kanonisch
	return t, tx.Commit(ctx)
}

// pgconnTag ist das, was tx.Exec zurückgibt — hier nur für RowsAffected.
type pgconnTag interface{ RowsAffected() int64 }

// DeleteLmfTermin entfernt einen Termin samt Klassen; false, wenn es ihn nicht gab.
func (r *LmfTerminRepository) DeleteLmfTermin(ctx context.Context, id string) (bool, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM lmf_termine WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// GetLmfTermin liest einen Termin; pgx.ErrNoRows, wenn es ihn nicht gibt.
func (r *LmfTerminRepository) GetLmfTermin(ctx context.Context, id string) (LmfTermin, error) {
	var t LmfTermin
	err := r.db.QueryRow(ctx, `
		SELECT t.id, to_char(t.datum, 'YYYY-MM-DD'), t.stunde, t.art, t.vermerk,
		       COALESCE((SELECT array_agg(k.klasse ORDER BY klassen_normkey(k.klasse))
		                 FROM lmf_termin_klassen k WHERE k.termin_id = t.id), '{}')
		FROM lmf_termine t WHERE t.id = $1`, id).
		Scan(&t.ID, &t.Datum, &t.Stunde, &t.Art, &t.Vermerk, &t.Klassen)
	return t, err
}

// KlassenOhneRueckgabeTermin nennt die Klassen mit aktiven Schülern, für die ab dem
// Datum kein Rückgabe-Termin eingetragen ist — der Hinweis auf der Planseite (Register,
// Entscheidung 3c: leer starten, aber zeigen, wer noch fehlt). Verglichen wird über den
// Normschlüssel, damit „5f1" und „05F1" dieselbe Klasse sind.
func (r *LmfTerminRepository) KlassenOhneRueckgabeTermin(ctx context.Context, ab time.Time) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.klasse
		FROM schueler s
		WHERE s.deleted_at IS NULL AND s.ist_abgaenger = false AND s.klasse ~ '^\d'
		  AND NOT EXISTS (
		      SELECT 1 FROM lmf_termin_klassen k
		      JOIN lmf_termine t ON t.id = k.termin_id
		      WHERE t.art = 'rueckgabe' AND t.datum >= $1::date
		        AND klassen_normkey(k.klasse) = klassen_normkey(s.klasse))
		GROUP BY s.klasse
		ORDER BY substring(s.klasse from '^\d+')::int, s.klasse`, ab)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	klassen := []string{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		klassen = append(klassen, k)
	}
	return klassen, rows.Err()
}
