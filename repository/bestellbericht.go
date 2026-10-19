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
	// Mittel ist der Topf, aus dem die Bestellung bezahlt wird. Leer heißt alte Bestellung ohne
	// eindeutige Zuordnung; sie wird als solche ausgewiesen und keinem Topf zugeschlagen.
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
func LadeBerichtBestellungen(ctx context.Context, db Datenbank, von, bisExklusiv time.Time, lieferantID, mittel string) ([]BerichtBestellung, map[string]int, error) {
	sqlBestellungen := `
		SELECT id, lieferant_name, kundennummer, bestelldatum, gesamtbetrag, anzahl_exemplare,
		       coalesce(mittel, '')
		FROM bestellungen_verlauf
		WHERE bestelldatum >= $1 AND bestelldatum < $2`
	args := []any{von, bisExklusiv}
	if lieferantID != "" {
		args = append(args, lieferantID)
		sqlBestellungen += fmt.Sprintf(" AND lieferant_id = $%d", len(args))
	}
	if bedingung, arg := mittelBedingung(mittel, "mittel", len(args)+1); bedingung != "" {
		sqlBestellungen += bedingung
		if arg != nil {
			args = append(args, arg)
		}
	}
	sqlBestellungen += " ORDER BY bestelldatum ASC"

	zeilen, err := db.Query(ctx, sqlBestellungen, args...)
	if err != nil {
		return nil, nil, err
	}
	defer zeilen.Close()

	bestellungen := make([]BerichtBestellung, 0)
	platzJeBestellung := map[string]int{}
	for zeilen.Next() {
		var o BerichtBestellung
		if err := zeilen.Scan(&o.ID, &o.LieferantName, &o.Kundennummer,
			&o.Bestelldatum, &o.Gesamtbetrag, &o.AnzahlExemplare, &o.Mittel); err != nil {
			return nil, nil, err
		}
		platzJeBestellung[o.ID] = len(bestellungen)
		bestellungen = append(bestellungen, o)
	}
	if err := zeilen.Err(); err != nil {
		return nil, nil, err
	}
	return bestellungen, platzJeBestellung, nil
}

// LadeBerichtPositionen lädt die Positionen aller Bestellungen nach und hängt sie
// den passenden Orders an.
func LadeBerichtPositionen(ctx context.Context, db Datenbank, bestellungen []BerichtBestellung, platzJeBestellung map[string]int) error {
	if len(bestellungen) == 0 {
		return nil
	}
	ids := make([]string, len(bestellungen))
	for i, o := range bestellungen {
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
		if idx, ok := platzJeBestellung[bestellungID]; ok {
			bestellungen[idx].Positionen = append(bestellungen[idx].Positionen, pos)
		}
	}
	return posRows.Err()
}
