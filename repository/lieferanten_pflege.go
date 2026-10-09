package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// SetzeHauptlieferant macht in einer laufenden Transaktion genau einen Lieferanten zum
// Hauptlieferanten: erst den bisherigen räumen, dann setzen. Der Teil-Index
// idx_lieferanten_ein_hauptlieferant lässt nur eine Zeile mit dem Merkmal zu; in der anderen
// Reihenfolge bräche das Setzen ab, sobald es schon einen Hauptlieferanten gibt. geaendert
// sagt, ob der Lieferant das Merkmal vorher nicht trug.
func SetzeHauptlieferant(ctx context.Context, tx pgx.Tx, id string) (geaendert bool, err error) {
	if _, err := tx.Exec(ctx,
		`UPDATE lieferanten SET ist_hauptlieferant = false WHERE ist_hauptlieferant AND id <> $1`, id); err != nil {
		return false, err
	}
	tag, err := tx.Exec(ctx,
		`UPDATE lieferanten SET ist_hauptlieferant = true WHERE id = $1 AND NOT ist_hauptlieferant`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// NimmHauptlieferant nimmt einem Lieferanten das Merkmal; geaendert sagt, ob er es trug.
func NimmHauptlieferant(ctx context.Context, db DBQueryer, id string) (geaendert bool, err error) {
	tag, err := db.Exec(ctx,
		`UPDATE lieferanten SET ist_hauptlieferant = false WHERE id = $1 AND ist_hauptlieferant`, id)
	return err == nil && tag.RowsAffected() == 1, err
}

// LieferantenZeile ist ein Lieferant in der Liste der Verwaltung.
type LieferantenZeile struct {
	ID                       string
	Name                     string
	Email                    string
	CustomerNumber           string
	ErstelltAm               time.Time
	IstHauptlieferant        bool
	KundennummerSchultraeger string
}

// ListeLieferanten liefert alle Lieferanten, den Hauptlieferanten zuerst: Das Bestellformular
// wählt den ersten der Liste vor.
func ListeLieferanten(ctx context.Context, db DBQueryer) ([]LieferantenZeile, error) {
	rows, err := db.Query(ctx, `
			SELECT id, name, email, kundennummer, erstellt_am, ist_hauptlieferant, kundennummer_schultraeger
			FROM lieferanten
			ORDER BY ist_hauptlieferant DESC, name ASC
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lieferanten := []LieferantenZeile{}
	for rows.Next() {
		var l LieferantenZeile
		if err := rows.Scan(&l.ID, &l.Name, &l.Email, &l.CustomerNumber, &l.ErstelltAm, &l.IstHauptlieferant, &l.KundennummerSchultraeger); err != nil {
			return nil, err
		}
		lieferanten = append(lieferanten, l)
	}
	return lieferanten, rows.Err()
}

// LegeLieferantAn legt einen Lieferanten ohne das Merkmal Hauptlieferant an und liefert
// Kennung und Zeitpunkt der Anlage.
func LegeLieferantAn(ctx context.Context, db DBQueryer, name, email, kundennummer, kundennummerSchultraeger string) (id string, erstelltAm time.Time, err error) {
	err = db.QueryRow(ctx, `
			INSERT INTO lieferanten (name, email, kundennummer, kundennummer_schultraeger)
			VALUES ($1, $2, $3, $4)
			RETURNING id, erstellt_am
		`, name, email, kundennummer, kundennummerSchultraeger).Scan(&id, &erstelltAm)
	return id, erstelltAm, err
}

// LieferantStand sind die Stammdaten eines Lieferanten vor oder nach einer Änderung.
type LieferantStand struct {
	Name, Email, Kundennummer, Zweitnummer string
	Haupt                                  bool
}

// AendereLieferantStammdaten schreibt die genannten Stammdaten und liefert den Stand davor und
// danach; ein Feld, das nil ist, behält seinen Wert. Den alten Stand liest die Anweisung aus
// der gesperrten Zeile. alt.Haupt füllt sie nicht: Ob das Merkmal sich geändert hat, weiß der
// Aufrufer. Ein unbekannter Lieferant ist pgx.ErrNoRows.
func AendereLieferantStammdaten(ctx context.Context, db DBQueryer, id string, name, email, kundennummer, zweitnummer *string) (alt, neu LieferantStand, err error) {
	err = db.QueryRow(ctx, `
		WITH alt AS (
			SELECT name, email, kundennummer, kundennummer_schultraeger
			  FROM lieferanten WHERE id = $4 FOR UPDATE
		)
		UPDATE lieferanten l
		   SET name = COALESCE($1, l.name), email = COALESCE($2, l.email),
		       kundennummer = COALESCE($3, l.kundennummer),
		       kundennummer_schultraeger = COALESCE($5, l.kundennummer_schultraeger)
		  FROM alt
		 WHERE l.id = $4
		RETURNING l.name, l.email, l.kundennummer, l.kundennummer_schultraeger, l.ist_hauptlieferant,
		          alt.name, alt.email, alt.kundennummer, alt.kundennummer_schultraeger`,
		name, email, kundennummer, id, zweitnummer,
	).Scan(&neu.Name, &neu.Email, &neu.Kundennummer, &neu.Zweitnummer, &neu.Haupt,
		&alt.Name, &alt.Email, &alt.Kundennummer, &alt.Zweitnummer)
	return alt, neu, err
}

// LieferantHauptUndName liest, ob ein Lieferant Hauptlieferant ist, und seinen Namen;
// pgx.ErrNoRows, wenn es ihn nicht gibt.
func LieferantHauptUndName(ctx context.Context, db DBQueryer, id string) (istHaupt bool, name string, err error) {
	err = db.QueryRow(ctx,
		"SELECT ist_hauptlieferant, name FROM lieferanten WHERE id = $1", id).Scan(&istHaupt, &name)
	return istHaupt, name, err
}

// LoescheLieferant löscht einen Lieferanten und liefert die Zahl der gelöschten Zeilen.
func LoescheLieferant(ctx context.Context, db DBQueryer, id string) (int64, error) {
	tag, err := db.Exec(ctx, "DELETE FROM lieferanten WHERE id = $1", id)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
