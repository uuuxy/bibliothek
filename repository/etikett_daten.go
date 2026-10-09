package repository

import (
	"context"
	"fmt"
)

// EtikettDaten sind die Angaben eines Exemplars für sein Etikett. AnschaffungsJahr ist das
// Jahr des Zugangs, Topf entscheidet über den Eigentumsvermerk (ExemplarTopfSQL). Drei
// Abfragen füllen sie: je Bestellung, je Titel und je Barcode. Wer einer ein Feld gibt, gibt
// es allen; api.TestEtikettenWegeDruckenDasselbe hält die Wege am fertigen PDF zusammen.
type EtikettDaten struct {
	BarcodeID        string
	Titel            string
	Autor            string
	ISBN             string
	AnschaffungsJahr string
	Signatur         string
	Topf             string
}

// EtikettDatenDerBestellung liefert die Exemplare einer Bestellung, beschränkt auf die
// Positionen mit Vorab-Barcode: Die übrigen beklebt die Bibliothek selbst. Das Jahr steht auf
// der Vorlage der Schule als „Ansch.J." unter dem Titel.
func EtikettDatenDerBestellung(ctx context.Context, db DBQueryer, bestellungID string) ([]EtikettDaten, error) {
	rows, err := db.Query(ctx, `
		SELECT e.barcode_id, t.titel, coalesce(t.autor, ''), coalesce(t.isbn, ''), coalesce(t.signatur, ''),
		       to_char(COALESCE(e.zugang_am, e.erworben_am), 'YYYY'),
		       `+ExemplarTopfSQL+`
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		`+ExemplarTopfJoin+`
		WHERE e.bestellung_id = $1
		  AND EXISTS (SELECT 1 FROM bestellungen_positionen p
		               WHERE p.bestellung_id = e.bestellung_id
		                 AND p.titel_id = e.titel_id
		                 AND p.mit_vorab_barcode)
		ORDER BY t.titel, e.barcode_id
	`, bestellungID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	etiketten := []EtikettDaten{}
	for rows.Next() {
		var d EtikettDaten
		if err := rows.Scan(&d.BarcodeID, &d.Titel, &d.Autor, &d.ISBN, &d.Signatur, &d.AnschaffungsJahr, &d.Topf); err != nil {
			return nil, err
		}
		etiketten = append(etiketten, d)
	}
	return etiketten, rows.Err()
}

// EtikettDatenDesTitels liefert die Exemplare eines Titels, die nicht ausgesondert sind.
func EtikettDatenDesTitels(ctx context.Context, db DBQueryer, titelID string) ([]EtikettDaten, error) {
	// Das Jahr ist das des Zugangs (zugang_am, Migration 129) — im Bestellweg ist erworben_am
	// der Bestelltag, und ein im Dezember bestelltes Buch käme sonst mit dem alten Jahr aufs
	// Etikett und mit dem neuen ins Zugangsbuch. Der Rückfall auf erworben_am (NOT NULL,
	// Vorgabe der Schultag, Migration 139) deckt Zeilen ohne Zugangsdatum; to_char liefert damit immer
	// vier Ziffern und nie NULL.
	//
	// Ausgesonderte Exemplare bleiben draußen (17.09.2026, OFFEN.md 5.5): Wer die Etiketten
	// eines Titels druckt, klebt sie auf Bücher, die im Regal stehen. Ein ausgesondertes
	// Exemplar gibt es dort nicht mehr — sein Etikett ist ein Blatt Papier für ein Buch,
	// das niemand findet, und auf einem Bogen mit fortlaufenden Plätzen verschiebt es alle
	// folgenden. Die Zeile bleibt in der Datenbank; nur gedruckt wird sie nicht.
	query := `
		SELECT e.barcode_id, t.titel, coalesce(t.autor, ''), to_char(COALESCE(e.zugang_am, e.erworben_am), 'YYYY'), coalesce(t.signatur, ''),
		       ` + ExemplarTopfSQL + `
		FROM buecher_exemplare e
		JOIN buecher_titel t ON e.titel_id = t.id
		` + ExemplarTopfJoin + `
		WHERE e.titel_id = $1 AND e.ist_ausgesondert = false
		ORDER BY e.barcode_id
	`
	rows, err := db.Query(ctx, query, titelID)
	if err != nil {
		return nil, fmt.Errorf("fehler beim laden der exemplare: %w", err)
	}
	defer rows.Close()

	var items []EtikettDaten
	for rows.Next() {
		var item EtikettDaten
		if err := rows.Scan(&item.BarcodeID, &item.Titel, &item.Autor, &item.AnschaffungsJahr, &item.Signatur, &item.Topf); err == nil {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("datenbankfehler: %w", err)
	}
	return items, nil
}

// EtikettServerfelder sind die Etikettenangaben, die der Server selbst kennt und deshalb nie
// aus einer Anfrage übernimmt.
type EtikettServerfelder struct {
	Jahr     string
	Signatur string
	Topf     string
}

// EtikettServerfelderZuBarcodes liefert Jahr, Signatur und Topf je bekanntem Barcode in einer
// Abfrage. Ein unbekannter Barcode fehlt in der Karte: Beim Vorab-Druck gibt es das Exemplar
// noch nicht. Bricht das Lesen der Zeilen ab, trägt der Fehler ErrZeileUnlesbar.
func EtikettServerfelderZuBarcodes(ctx context.Context, db DBQueryer, barcodes []string) (map[string]EtikettServerfelder, error) {
	rows, err := db.Query(ctx, `
		SELECT e.barcode_id, to_char(COALESCE(e.zugang_am, e.erworben_am), 'YYYY'), coalesce(t.signatur, ''),
		       `+ExemplarTopfSQL+`
		FROM buecher_exemplare e
		JOIN buecher_titel t ON e.titel_id = t.id
		`+ExemplarTopfJoin+`
		WHERE e.barcode_id = ANY($1)
	`, barcodes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bekannt := make(map[string]EtikettServerfelder, len(barcodes))
	for rows.Next() {
		var barcode string
		var felder EtikettServerfelder
		if err := rows.Scan(&barcode, &felder.Jahr, &felder.Signatur, &felder.Topf); err == nil {
			bekannt[barcode] = felder
		}
	}
	if err := rows.Err(); err != nil {
		return nil, zeileUnlesbar(err)
	}
	return bekannt, nil
}
