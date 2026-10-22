package repository

import (
	"context"
)

// EtikettDaten sind die Angaben eines Exemplars für sein Etikett. AnschaffungsJahr ist das
// Jahr des Zugangs, Topf entscheidet über den Eigentumsvermerk (ExemplarTopfSQL). Zwei
// Abfragen füllen sie: je Bestellung (LadeEtikettDatenDerBestellung) und je Barcode
// (LadeEtikettServerfelderZuBarcodes). Wer einer ein Feld gibt, gibt es beiden;
// buchetikett.TestEtikettenDatenSindAufBeidenWegenGleich hält sie zusammen.
type EtikettDaten struct {
	BarcodeID        string
	Titel            string
	Autor            string
	ISBN             string
	AnschaffungsJahr string
	Signatur         string
	Topf             string
}

// LadeEtikettDatenDerBestellung liefert die Exemplare einer Bestellung, beschränkt auf die
// Positionen mit Vorab-Barcode: Die übrigen beklebt die Bibliothek selbst. Das Jahr steht auf
// der Vorlage der Schule als „Ansch.J." unter dem Titel.
func LadeEtikettDatenDerBestellung(ctx context.Context, db Datenbank, bestellungID string) ([]EtikettDaten, error) {
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
		ORDER BY t.titel COLLATE deutsch, e.barcode_id
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

// EtikettServerfelder sind die Etikettenangaben, die der Server selbst kennt und deshalb nie
// aus einer Anfrage übernimmt.
type EtikettServerfelder struct {
	Jahr     string
	Signatur string
	Topf     string
}

// LadeEtikettServerfelderZuBarcodes liefert Jahr, Signatur und Topf je bekanntem Barcode in einer
// Abfrage. Ein unbekannter Barcode fehlt in der Karte: Beim Vorab-Druck gibt es das Exemplar
// noch nicht. Bricht das Lesen der Zeilen ab, trägt der Fehler ErrZeileUnlesbar.
//
// Das Jahr ist das des Zugangs (zugang_am): Im Bestellweg ist erworben_am der Bestelltag, und
// ein im Dezember bestelltes Buch käme sonst mit dem alten Jahr aufs Etikett und mit dem neuen
// ins Zugangsbuch. Der Rückfall auf erworben_am (NOT NULL, Vorgabe der Schultag) deckt Zeilen
// ohne Zugangsdatum; to_char liefert damit immer vier Ziffern und nie NULL.
func LadeEtikettServerfelderZuBarcodes(ctx context.Context, db Datenbank, barcodes []string) (map[string]EtikettServerfelder, error) {
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
		if err := rows.Scan(&barcode, &felder.Jahr, &felder.Signatur, &felder.Topf); err != nil {
			return nil, zeileUnlesbar(err)
		}
		bekannt[barcode] = felder
	}
	if err := rows.Err(); err != nil {
		return nil, zeileUnlesbar(err)
	}
	return bekannt, nil
}
