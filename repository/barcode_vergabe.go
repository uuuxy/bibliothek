package repository

import (
	"context"
	"fmt"
)

// Vergabe der Exemplarnummern (Präfix "B-").
//
// EINE Quelle für alle Wege — Bestellwesen, Handvergabe in der Exemplarkarte,
// Littera-Import: die Postgres-Sequenz barcode_seq. Sie zählt streng aufwärts und sieht
// die Tabelle nie an. Eine einmal vergebene Nummer kann deshalb auch dann nicht
// zurückkehren, wenn ihr Exemplar hart gelöscht wird (Titel löschen, Massenlöschung in
// der Bestandsverwaltung, „endgültig löschen" im Fehlbestand). Genau das ist gewollt:
// Die Nummer klebt physisch am Buch, und ein Etikett mit recycelter Nummer bucht auf
// das falsche Exemplar.
//
// Vorher gab es hier eine zweite Vergabestelle: Der Knopf „Interne ID generieren"
// rechnete MAX(Nummer)+1 aus dem Bestand (SequenceRepository.GetNextSequence). Beide
// Stellen lieferten dieselbe nächste Nummer, aber nur die Sequenz zählte weiter — die
// von Hand vergebene Nummer kam bei der nächsten Bestellung erneut und lief in den
// UNIQUE-Constraint. Weil die Exemplare einer Bestellung in EINER Transaktion per
// CopyFrom entstehen, kostete das nicht die Position, sondern die ganze Bestellung.
// GetNextSequence bedient nur noch die Schülerausweise ("S-").

// barcodeZiehVersuche begrenzt die Nachzieh-Runden. Nach dem Heben der Sequenz über den
// Bestand liegt der nächste Zug garantiert im freien Bereich; mehr als eine
// Wiederholung braucht es nur, wenn parallel von Hand eine Nummer eingetragen wird.
const barcodeZiehVersuche = 3

// barcodeNummerMuster erkennt die maschinell vergebenen Nummern. Die Ziffernlänge ist
// begrenzt, damit eine von Hand eingetippte Fantasienummer ("B-999999999999999999999")
// den Cast auf bigint nicht sprengt und damit die Bestellung mitreißt.
const barcodeNummerMuster = `^B-[0-9]{1,15}$`

// NaechsterFreierExemplarBarcode liefert die nächste freie Exemplarnummer für die
// Handvergabe in der Exemplarkarte.
//
// Die Nummer ist danach verbraucht, auch wenn der Vorschlag nicht gespeichert wird.
// Eine Lücke im Nummernkreis ist harmlos — eine doppelt vergebene Nummer nicht.
func NaechsterFreierExemplarBarcode(ctx context.Context, db DBQueryer) (string, error) {
	barcodes, err := ZieheFreieExemplarBarcodes(ctx, db, 1)
	if err != nil {
		return "", err
	}
	return barcodes[0], nil
}

// ZieheFreieExemplarBarcodes liefert anzahl garantiert unbelegte Exemplarnummern.
//
// Der Abgleich gegen den Bestand ist nötig, weil barcode_seq nur die maschinell
// erzeugten Nummern kennt: Ein in der Exemplarkarte von Hand eingetipptes "B-12345"
// steht in der Tabelle, aber nicht in der Sequenz. Kollidiert ein gezogener Wert, wird
// die Sequenz über den belegten Bereich gehoben und nachgezogen.
func ZieheFreieExemplarBarcodes(ctx context.Context, db DBQueryer, anzahl int) ([]string, error) {
	if anzahl <= 0 {
		return nil, nil
	}

	frei := make([]string, 0, anzahl)
	for versuch := 0; versuch < barcodeZiehVersuche; versuch++ {
		kandidaten, err := zieheAusSequenz(ctx, db, anzahl-len(frei))
		if err != nil {
			return nil, err
		}
		belegt, err := belegteBarcodes(ctx, db, kandidaten)
		if err != nil {
			return nil, err
		}
		for _, bc := range kandidaten {
			if !belegt[bc] {
				frei = append(frei, bc)
			}
		}
		if len(frei) == anzahl {
			return frei, nil
		}
		if err := hebeSequenzUeberBestand(ctx, db); err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("barcode_seq liefert nur bereits vergebene Nummern – Sequenzstand prüfen")
}

// zieheAusSequenz holt anzahl Nummern aus barcode_seq. LPAD auf fünf Stellen hält das
// gewohnte Format B-10001; jenseits von 99999 wächst die Nummer einfach mit.
func zieheAusSequenz(ctx context.Context, db DBQueryer, anzahl int) ([]string, error) {
	rows, err := db.Query(ctx,
		`SELECT 'B-' || LPAD(nextval('barcode_seq')::TEXT, 5, '0') FROM generate_series(1, $1)`,
		anzahl)
	if err != nil {
		return nil, fmt.Errorf("konnte keine Nummern aus barcode_seq ziehen: %w", err)
	}
	defer rows.Close()

	codes := make([]string, 0, anzahl)
	for rows.Next() {
		var bc string
		if err := rows.Scan(&bc); err != nil {
			return nil, fmt.Errorf("konnte keine Nummern aus barcode_seq ziehen: %w", err)
		}
		codes = append(codes, bc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("konnte keine Nummern aus barcode_seq ziehen: %w", err)
	}
	if len(codes) != anzahl {
		return nil, fmt.Errorf("barcode_seq lieferte %d von %d Nummern", len(codes), anzahl)
	}
	return codes, nil
}

// belegteBarcodes meldet, welche der Kandidaten bereits an einem Exemplar hängen.
func belegteBarcodes(ctx context.Context, db DBQueryer, kandidaten []string) (map[string]bool, error) {
	rows, err := db.Query(ctx,
		`SELECT barcode_id FROM buecher_exemplare WHERE barcode_id = ANY($1)`, kandidaten)
	if err != nil {
		return nil, fmt.Errorf("konnte die vergebenen Barcodes nicht prüfen: %w", err)
	}
	defer rows.Close()

	belegt := map[string]bool{}
	for rows.Next() {
		var bc string
		if err := rows.Scan(&bc); err != nil {
			return nil, fmt.Errorf("konnte die vergebenen Barcodes nicht prüfen: %w", err)
		}
		belegt[bc] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("konnte die vergebenen Barcodes nicht prüfen: %w", err)
	}
	return belegt, nil
}

// hebeSequenzUeberBestand zieht barcode_seq auf die höchste tatsächlich vergebene
// Nummer nach.
//
// GREATEST mit dem aktuellen Stand: Die Sequenz darf nur steigen. Ein Senken würde
// Nummern erneut ausgeben, die eine parallel laufende Bestellung bereits gezogen, aber
// noch nicht eingefügt hat.
func hebeSequenzUeberBestand(ctx context.Context, db DBQueryer) error {
	_, err := db.Exec(ctx, `
		SELECT setval('barcode_seq', GREATEST(
			(SELECT last_value FROM barcode_seq),
			(SELECT coalesce(max(substr(barcode_id, 3)::bigint), 0)
			   FROM buecher_exemplare
			  WHERE barcode_id ~ $1)))
	`, barcodeNummerMuster)
	if err != nil {
		return fmt.Errorf("barcode_seq konnte nicht über den Bestand gehoben werden: %w", err)
	}
	return nil
}
