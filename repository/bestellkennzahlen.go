package repository

import "context"

// BestellKennzahlen sind die Summen über alle Bestellungen.
type BestellKennzahlen struct {
	Gesamt               int
	Gesamtbetrag         float64
	GesamtExemplare      int
	OffeneBestaetigungen int
}

// LadeBestellKennzahlen liest die Summen aus dem Kopf der Bestellungen, ohne die Positionen.
//
// „Wartet auf Bestätigung" zählt am Token der Bestellung, nicht am Merkmal des Lieferanten:
// Der Token entsteht beim Bestellen, wenn der Lieferant den Bestelllink trägt, und bleibt.
// Am Merkmal gezählt, fielen die offenen Bestellungen aus der Zahl, sobald der Bestelllink an
// einen anderen Händler geht, obwohl weiter auf ihre Bestätigung gewartet wird.
func LadeBestellKennzahlen(ctx context.Context, db DBQueryer) (BestellKennzahlen, error) {
	var k BestellKennzahlen
	err := db.QueryRow(ctx, `
			SELECT count(*), coalesce(sum(b.gesamtbetrag), 0), coalesce(sum(b.anzahl_exemplare), 0),
			       count(*) FILTER (
			           WHERE b.bestaetigt_am IS NULL AND b.bestaetigungs_token_hash IS NOT NULL
			       )
			FROM bestellungen_verlauf b
		`).Scan(&k.Gesamt, &k.Gesamtbetrag, &k.GesamtExemplare, &k.OffeneBestaetigungen)
	return k, err
}

// BestellTopfKennzahlen sind die Summen eines Topfs; Mittel ist leer für Bestellungen ohne
// Zuordnung.
type BestellTopfKennzahlen struct {
	Mittel          string
	Gesamt          int
	Gesamtbetrag    float64
	GesamtExemplare int
}

// LadeBestellKennzahlenJeTopf liefert die Summen je Topf, nach dem Wert der Spalte mittel. Eine
// eigene Abfrage neben LadeBestellKennzahlen: Die Summen über alles sollen nicht davon
// abhängen, dass die Gruppierung gelingt.
func LadeBestellKennzahlenJeTopf(ctx context.Context, db DBQueryer) (map[string]BestellTopfKennzahlen, error) {
	rows, err := db.Query(ctx, `
		SELECT coalesce(mittel, ''), count(*), coalesce(sum(gesamtbetrag), 0),
		       coalesce(sum(anzahl_exemplare), 0)
		FROM bestellungen_verlauf
		GROUP BY coalesce(mittel, '')
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	gezaehlt := map[string]BestellTopfKennzahlen{}
	for rows.Next() {
		var t BestellTopfKennzahlen
		if err := rows.Scan(&t.Mittel, &t.Gesamt, &t.Gesamtbetrag, &t.GesamtExemplare); err != nil {
			return nil, err
		}
		gezaehlt[t.Mittel] = t
	}
	return gezaehlt, rows.Err()
}
