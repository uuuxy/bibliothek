package repository

import (
	"context"
	"fmt"
	"time"
)

// BerichtBestellung ist eine Bestellung des Bestellberichts mit ihren Positionen.
type BerichtBestellung struct {
	ID              string
	LieferantName   string
	Kundennummer    string
	Bestelldatum    time.Time
	Gesamtbetrag    float64
	AnzahlExemplare int
	// Mittel ist der Topf, aus dem die Bestellung bezahlt wird (Migration 109). Leer =
	// Alt-Bestellung ohne eindeutige Zuordnung; sie wird als solche ausgewiesen und
	// niemals einem Topf zugeschlagen.
	Mittel     string
	Positionen []BerichtPosition
}

// BerichtPosition ist eine Position einer Bestellung im Bestellbericht.
type BerichtPosition struct {
	TitelName   string
	ISBN        string
	Menge       int
	Einzelpreis float64
}

// LadeBerichtBestellungen liest die Bestellungen im Zeitraum (optional je Lieferant) und
// liefert zusätzlich einen Index ID→Position für das Nachladen der Positionen.
func LadeBerichtBestellungen(ctx context.Context, db DBQueryer, von, bisExklusiv time.Time, lieferantID, mittel string) ([]BerichtBestellung, map[string]int, error) {
	orderQuery := `
		SELECT id, lieferant_name, kundennummer, bestelldatum, gesamtbetrag, anzahl_exemplare,
		       coalesce(mittel, '')
		FROM bestellungen_verlauf
		WHERE bestelldatum >= $1 AND bestelldatum < $2`
	args := []any{von, bisExklusiv}
	if lieferantID != "" {
		args = append(args, lieferantID)
		orderQuery += fmt.Sprintf(" AND lieferant_id = $%d", len(args))
	}
	if bedingung, arg := MittelBedingung(mittel, "mittel", len(args)+1); bedingung != "" {
		orderQuery += bedingung
		if arg != nil {
			args = append(args, arg)
		}
	}
	orderQuery += " ORDER BY bestelldatum ASC"

	orderRows, err := db.Query(ctx, orderQuery, args...)
	if err != nil {
		return nil, nil, err
	}
	defer orderRows.Close()

	orders := make([]BerichtBestellung, 0)
	orderIndex := map[string]int{}
	for orderRows.Next() {
		var o BerichtBestellung
		if err := orderRows.Scan(&o.ID, &o.LieferantName, &o.Kundennummer,
			&o.Bestelldatum, &o.Gesamtbetrag, &o.AnzahlExemplare, &o.Mittel); err != nil {
			return nil, nil, err
		}
		orderIndex[o.ID] = len(orders)
		orders = append(orders, o)
	}
	if err := orderRows.Err(); err != nil {
		return nil, nil, err
	}
	return orders, orderIndex, nil
}

// LadeBerichtPositionen lädt die Positionen aller Bestellungen nach und hängt sie
// den passenden Orders an.
func LadeBerichtPositionen(ctx context.Context, db DBQueryer, orders []BerichtBestellung, orderIndex map[string]int) error {
	if len(orders) == 0 {
		return nil
	}
	ids := make([]string, len(orders))
	for i, o := range orders {
		ids[i] = o.ID
	}
	posRows, err := db.Query(ctx, `
		SELECT bestellung_id, titel_name, isbn, menge, einzelpreis
		FROM bestellungen_positionen
		WHERE bestellung_id = ANY($1::uuid[])
		ORDER BY bestellung_id, titel_name`, ids)
	if err != nil {
		return err
	}
	defer posRows.Close()
	for posRows.Next() {
		var bestellungID string
		var pos BerichtPosition
		if err := posRows.Scan(&bestellungID, &pos.TitelName, &pos.ISBN, &pos.Menge, &pos.Einzelpreis); err != nil {
			return err
		}
		if idx, ok := orderIndex[bestellungID]; ok {
			orders[idx].Positionen = append(orders[idx].Positionen, pos)
		}
	}
	return posRows.Err()
}
