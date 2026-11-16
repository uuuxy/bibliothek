package pgtest

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// grundzustand hält fest, wie schema.sql die Datenbank hinterlässt, und bringt sie dorthin
// zurück: Tabellen mit Zeilen werden leer, die Tabellen, die schema.sql füllt, bekommen ihre
// Zeilen wieder, und jede Folge (sequence) steht wieder auf ihrem Wert.
type grundzustand struct {
	belegtAbfrage string             // fragt je leere Tabelle, ob sie Zeilen hat
	leere         []string           // Namen der Tabellen, die schema.sql leer lässt, maskiert
	gefuellt      []gefuellteTabelle // die Tabellen, die schema.sql füllt
	folgen        string             // setzt jede Folge auf ihren Wert nach dem Schema
}

type gefuellteTabelle struct {
	name   string // maskiert
	zeilen string // die Zeilen als JSON
}

// leseGrundzustand liest den Zustand gleich nach dem Einspielen des Schemas.
func leseGrundzustand(ctx context.Context, p *pgxpool.Pool) (grundzustand, error) {
	var g grundzustand
	zeilen, err := p.Query(ctx, `
		SELECT c.relname FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p') AND NOT c.relispartition
		ORDER BY c.relname`)
	if err != nil {
		return g, err
	}
	namen, err := pgx.CollectRows(zeilen, pgx.RowTo[string])
	if err != nil {
		return g, err
	}
	var belegt []string
	for _, name := range namen {
		tabelle := pgx.Identifier{name}.Sanitize()
		var inhalt *string
		if err := p.QueryRow(ctx, `SELECT json_agg(t)::text FROM `+tabelle+` t`).Scan(&inhalt); err != nil {
			return g, fmt.Errorf("%s lesen: %w", tabelle, err)
		}
		if inhalt != nil {
			g.gefuellt = append(g.gefuellt, gefuellteTabelle{name: tabelle, zeilen: *inhalt})
			continue
		}
		g.leere = append(g.leere, tabelle)
		belegt = append(belegt, `EXISTS (SELECT 1 FROM `+tabelle+`)`)
	}
	g.belegtAbfrage = `SELECT ` + strings.Join(belegt, ", ")
	if err := p.QueryRow(ctx, `
		SELECT coalesce('SELECT ' || string_agg(format('setval(%L, %s, %L)', format('%I', sequencename),
		       coalesce(last_value, start_value), last_value IS NOT NULL), ', '), '')
		FROM pg_sequences WHERE schemaname = 'public'`).Scan(&g.folgen); err != nil {
		return g, fmt.Errorf("pg_sequences lesen: %w", err)
	}
	return g, nil
}

// setzeZurueck bringt die Datenbank in einer Transaktion in den Grundzustand.
func (g grundzustand) setzeZurueck(ctx context.Context, p *pgxpool.Pool) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // nach dem Commit ohne Wirkung
	// Hält ein Test noch eine Sperre, scheitert das Zurücksetzen mit Meldung, statt den Lauf
	// anzuhalten.
	if _, err := tx.Exec(ctx, `SET LOCAL lock_timeout = '10s'`); err != nil {
		return err
	}
	if err := g.leereBelegte(ctx, tx); err != nil {
		return err
	}
	if err := g.schreibeZeilenZurueck(ctx, tx); err != nil {
		return err
	}
	if g.folgen != "" {
		if _, err := tx.Exec(ctx, g.folgen); err != nil {
			return fmt.Errorf("setval der Folgen: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// leereBelegte leert die Tabellen, die schema.sql leer lässt und die jetzt Zeilen haben. Ein
// TRUNCATE über alle kostet rund doppelt so viel Zeit.
func (g grundzustand) leereBelegte(ctx context.Context, tx pgx.Tx) error {
	if len(g.leere) == 0 {
		return nil
	}
	ziele := make([]any, len(g.leere))
	belegt := make([]bool, len(g.leere))
	for i := range belegt {
		ziele[i] = &belegt[i]
	}
	if err := tx.QueryRow(ctx, g.belegtAbfrage).Scan(ziele...); err != nil {
		return fmt.Errorf("belegte Tabellen: %w", err)
	}
	var leeren []string
	for i, ja := range belegt {
		if ja {
			leeren = append(leeren, g.leere[i])
		}
	}
	if len(leeren) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `TRUNCATE `+strings.Join(leeren, ", ")+` CASCADE`)
	return err
}

// schreibeZeilenZurueck stellt die Tabellen, die schema.sql füllt, auf ihre Zeilen von damals.
func (g grundzustand) schreibeZeilenZurueck(ctx context.Context, tx pgx.Tx) error {
	for _, f := range g.gefuellt {
		if _, err := tx.Exec(ctx, `DELETE FROM `+f.name); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO `+f.name+` SELECT * FROM json_populate_recordset(NULL::`+
			f.name+`, $1::json)`, f.zeilen); err != nil {
			return fmt.Errorf("%s zurückschreiben: %w", f.name, err)
		}
	}
	return nil
}
