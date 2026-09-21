package main

import (
	"context"
	"fmt"
	"log"

	"bibliothek/internal/crypto"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Der eigentliche Schlüsselwechsel — getrennt von der Kommandozeile in main.go, damit
// die Regel „ein Datensatz nicht lesbar ⇒ gar nichts wird geschrieben" an einer Stelle
// steht und nicht zwischen Flag-Auswertung und Ausgabe verschwindet.

// umschluesselung beschreibt eine Tabelle mit einer verschlüsselten Spalte.
type umschluesselung = repository.VerschluesselteSpalte

// tabellen ist die eine Liste aus dem Repository — dieselbe, an der die
// Betriebsbereitschaft prüft, ob der Schlüssel zum Bestand passt.
var tabellen = repository.VerschluesselteSpalten

// rotiere schlüsselt alle betroffenen Tabellen in EINER Transaktion um.
func rotiere(ctx context.Context, pool *pgxpool.Pool, alt, neu []byte, nurPruefen bool) (int, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("transaktion beginnen: %w", err)
	}
	// Rollback im Fehlerfall UND im Probelauf. Nach einem erfolgreichen Commit ist er
	// wirkungslos (pgx liefert dann ErrTxClosed).
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	gesamt := 0
	for _, t := range tabellen {
		anzahl, err := rotiereTabelle(ctx, tx, t, alt, neu)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", t.Beschreibung, err)
		}
		log.Printf("%s: %d Datensätze umgeschlüsselt.", t.Beschreibung, anzahl)
		gesamt += anzahl
	}

	if nurPruefen {
		return gesamt, nil // defer rollt zurück
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("transaktion abschließen: %w", err)
	}
	return gesamt, nil
}

// rotiereTabelle liest alle verschlüsselten Werte einer Tabelle, schlüsselt sie um und
// schreibt sie zurück. Ein einziger nicht entschlüsselbarer Datensatz bricht den ganzen
// Lauf ab — lieber gar nicht rotieren als einen Bestand mit zwei Schlüsseln hinterlassen.
func rotiereTabelle(ctx context.Context, tx pgx.Tx, t umschluesselung, alt, neu []byte) (int, error) {
	// Tabellen- und Spaltennamen stammen ausschließlich aus der Konstante `tabellen`
	// oben, nie aus einer Eingabe.
	abfrage := fmt.Sprintf(
		`SELECT %s::text, %s FROM %s WHERE %s IS NOT NULL`,
		t.IDSpalte, t.DatenSpalte, t.Tabelle, t.DatenSpalte)

	rows, err := tx.Query(ctx, abfrage)
	if err != nil {
		return 0, fmt.Errorf("lesen: %w", err)
	}

	type datensatz struct {
		id   string
		wert []byte
	}
	var gelesen []datensatz
	for rows.Next() {
		var d datensatz
		if err := rows.Scan(&d.id, &d.wert); err != nil {
			rows.Close()
			return 0, fmt.Errorf("lesen: %w", err)
		}
		gelesen = append(gelesen, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("lesen: %w", err)
	}

	aktualisieren := fmt.Sprintf(`UPDATE %s SET %s = $1 WHERE %s = $2`, t.Tabelle, t.DatenSpalte, t.IDSpalte)

	for _, d := range gelesen {
		klartext, err := crypto.DecryptMit(alt, d.wert)
		if err != nil {
			return 0, fmt.Errorf("datensatz %s ließ sich mit dem alten Schlüssel nicht entschlüsseln "+
				"(steht in %s wirklich der Schlüssel, mit dem diese Daten geschrieben wurden?): %w",
				d.id, crypto.SchluesselVariable, err)
		}
		neuerWert, err := crypto.EncryptMit(neu, klartext)
		if err != nil {
			return 0, fmt.Errorf("datensatz %s ließ sich nicht neu verschlüsseln: %w", d.id, err)
		}
		tag, err := tx.Exec(ctx, aktualisieren, neuerWert, d.id)
		if err != nil {
			return 0, fmt.Errorf("datensatz %s schreiben: %w", d.id, err)
		}
		if tag.RowsAffected() != 1 {
			return 0, fmt.Errorf("datensatz %s: %d Zeilen betroffen statt 1", d.id, tag.RowsAffected())
		}
	}

	return len(gelesen), nil
}
