package testdaten

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"bibliothek/internal/pgtest"
)

// LegeLieferantAn legt einen Lieferanten an und liefert seine Kennung. Als Hauptlieferant räumt er
// zuerst den bisherigen: Es darf höchstens einen geben, und LeereBestandsdaten leert die
// Lieferanten nicht.
func LegeLieferantAn(t *testing.T, pool *pgxpool.Pool, name string, istHaupt bool) string {
	t.Helper()
	ctx := context.Background()
	if istHaupt {
		if _, err := pool.Exec(ctx,
			`UPDATE lieferanten SET ist_hauptlieferant = false WHERE ist_hauptlieferant`); err != nil {
			t.Fatalf("bisherigen Hauptlieferanten räumen: %v", err)
		}
	}
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO lieferanten (name, email, kundennummer, ist_hauptlieferant)
		VALUES ($1, $2, $3, $4) RETURNING id
	`, name, name+"@example.invalid", "K-"+name, istHaupt).Scan(&id)
	if err != nil {
		t.Fatalf("Lieferant %s anlegen: %v", name, err)
	}
	return id
}

// LegeBestellungAn legt eine Bestellung mit Topf an.
func LegeBestellungAn(t *testing.T, mittel string) string {
	t.Helper()
	var id string
	if err := pgtest.Pool(t).QueryRow(context.Background(), `
		INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email, mittel)
		VALUES ('Buchhandlung', 'haendler@example.org', $1) RETURNING id`, mittel).Scan(&id); err != nil {
		t.Fatalf("Bestellung: %v", err)
	}
	return id
}

// LadeBarcodesDerBestellung liest die Nummern der nicht ausleihbaren Exemplare, wie eine Bestellung
// sie anlegt.
func LadeBarcodesDerBestellung(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		`SELECT barcode_id FROM buecher_exemplare WHERE ist_ausleihbar = false ORDER BY barcode_id`)
	if err != nil {
		t.Fatalf("Barcodes lesen: %v", err)
	}
	defer rows.Close()

	var liste []string
	for rows.Next() {
		var b string
		if err := rows.Scan(&b); err != nil {
			t.Fatalf("Barcode lesen: %v", err)
		}
		liste = append(liste, b)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Barcodes lesen: %v", err)
	}
	return liste
}

// LadeExemplarIDsDerBestellung liest die Kennungen, mit denen der Wareneingang arbeitet.
func LadeExemplarIDsDerBestellung(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		`SELECT id FROM buecher_exemplare WHERE ist_ausleihbar = false ORDER BY barcode_id`)
	if err != nil {
		t.Fatalf("Exemplar-IDs lesen: %v", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("Exemplar-ID lesen: %v", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Exemplar-IDs lesen: %v", err)
	}
	return ids
}

// SetzeOeffentlicheAdresse hinterlegt die Adresse, aus der der Bestätigungs-Link entsteht; ein
// leerer Wert entfernt sie.
func SetzeOeffentlicheAdresse(t *testing.T, pool *pgxpool.Pool, adresse string) {
	t.Helper()
	ctx := context.Background()
	var err error
	if adresse == "" {
		_, err = pool.Exec(ctx, `DELETE FROM system_einstellungen WHERE schluessel = 'oeffentliche_adresse'`)
	} else {
		_, err = pool.Exec(ctx, `
			INSERT INTO system_einstellungen (schluessel, wert) VALUES ('oeffentliche_adresse', $1)
			ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, adresse)
	}
	if err != nil {
		t.Fatalf("öffentliche Adresse setzen: %v", err)
	}
}

// LegeBestellungOhneTopfAn legt eine Bestellung an, wie sie vor der Zuordnung zu einem Topf entstand: ohne
// Mittel, mit je einer Position für die genannten Titel. Ein leerer Eintrag steht für eine
// Position, deren Titel gelöscht ist.
func LegeBestellungOhneTopfAn(t *testing.T, pool *pgxpool.Pool, name string, titelIDs []string) string {
	t.Helper()
	ctx := context.Background()
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email) VALUES ($1, 'alt@example.invalid')
		RETURNING id`, name).Scan(&id); err != nil {
		t.Fatalf("Alt-Bestellung anlegen: %v", err)
	}
	for _, titelID := range titelIDs {
		if _, err := pool.Exec(ctx, `
			INSERT INTO bestellungen_positionen (bestellung_id, titel_id, titel_name, menge)
			VALUES ($1, NULLIF($2, '')::uuid, 'Titel', 1)`, id, titelID); err != nil {
			t.Fatalf("Position anlegen: %v", err)
		}
	}
	return id
}
