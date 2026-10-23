package repository

import (
	"context"
	"fmt"
)

// Vergabe der Exemplarnummern (Präfix "B-").
//
// Eine Quelle für alle Wege (Bestellwesen, Handvergabe in der Exemplarkarte, Littera-Import):
// die Postgres-Sequenz barcode_seq. Sie zählt streng aufwärts und sieht die Tabelle nie an.
// Eine einmal vergebene Nummer kehrt deshalb auch dann nicht zurück, wenn ihr Exemplar hart
// gelöscht wird (Titel löschen, Massenlöschung in der Bestandsverwaltung, „endgültig löschen"
// im Fehlbestand). Das ist gewollt: Die Nummer klebt am Buch, und ein Etikett mit einer
// wiederverwendeten Nummer bucht auf das falsche Exemplar.
//
// Eine zweite Vergabestelle, die MAX(Nummer)+1 aus dem Bestand rechnet, darf es nicht geben:
// Beide lieferten dieselbe nächste Nummer, aber nur die Sequenz zählte weiter. Die von Hand
// vergebene Nummer käme bei der nächsten Bestellung erneut und liefe in den UNIQUE-Constraint;
// weil die Exemplare einer Bestellung in einer Transaktion per CopyFrom entstehen, kostete das
// die ganze Bestellung. Der Ausweis-Generator (ausweis_nummer_start in der Datenbank) bedient
// nur die Ausweisnummern („A-").

// barcodeZiehVersuche begrenzt die Nachzieh-Runden. Nach dem Heben der Sequenz über den
// Bestand liegt der nächste Zug garantiert im freien Bereich; mehr als eine
// Wiederholung braucht es nur, wenn parallel von Hand eine Nummer eingetragen wird.
const barcodeZiehVersuche = 3

// barcodeNummerMuster erkennt die maschinell vergebenen Nummern. Die Ziffernlänge ist
// begrenzt, damit eine von Hand eingetippte Fantasienummer ("B-999999999999999999999")
// den Cast auf bigint nicht sprengt und damit die Bestellung mitreißt.
const barcodeNummerMuster = `^B-[0-9]{1,15}$`

// ZieheFreienExemplarBarcode liefert die nächste freie Exemplarnummer für die
// Handvergabe in der Exemplarkarte.
//
// Die Nummer ist danach verbraucht, auch wenn der Vorschlag nicht gespeichert wird.
// Eine Lücke im Nummernkreis ist harmlos — eine doppelt vergebene Nummer nicht.
func ZieheFreienExemplarBarcode(ctx context.Context, db Datenbank) (string, error) {
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
func ZieheFreieExemplarBarcodes(ctx context.Context, db Datenbank, anzahl int) ([]string, error) {
	if anzahl <= 0 {
		return nil, nil
	}

	frei := make([]string, 0, anzahl)
	for versuch := 0; versuch < barcodeZiehVersuche; versuch++ {
		kandidaten, err := ZieheExemplarNummern(ctx, db, anzahl-len(frei))
		if err != nil {
			return nil, err
		}
		belegt, err := ladeBelegteBarcodes(ctx, db, kandidaten)
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

// exemplarNummer setzt eine Zahl aus barcode_seq in die gedruckte Form: „B-" und mindestens fünf
// Ziffern, wie AusweisNummer. Gesetzt wird in Go, weil LPAD eine längere Zahl abschneidet: Aus
// 100000 würde B-10000, die Nummer eines anderen Buchs.
func exemplarNummer(n int64) string { return fmt.Sprintf("B-%05d", n) }

// ZieheExemplarNummern holt anzahl Nummern aus barcode_seq, ohne den Bestand anzusehen. Wer sie
// vergibt, prüft, ob eine schon an einem Exemplar hängt: ZieheFreieExemplarBarcodes und die
// Littera-Übernahme mit ihrer Liste belegter Nummern.
func ZieheExemplarNummern(ctx context.Context, db Datenbank, anzahl int) ([]string, error) {
	rows, err := db.Query(ctx, `SELECT nextval('barcode_seq') FROM generate_series(1, $1)`, anzahl)
	if err != nil {
		return nil, fmt.Errorf("konnte keine Nummern aus barcode_seq ziehen: %w", err)
	}
	defer rows.Close()

	codes := make([]string, 0, anzahl)
	for rows.Next() {
		var n int64
		if err := rows.Scan(&n); err != nil {
			return nil, fmt.Errorf("konnte keine Nummern aus barcode_seq ziehen: %w", err)
		}
		codes = append(codes, exemplarNummer(n))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("konnte keine Nummern aus barcode_seq ziehen: %w", err)
	}
	if len(codes) != anzahl {
		return nil, fmt.Errorf("barcode_seq lieferte %d von %d Nummern", len(codes), anzahl)
	}
	return codes, nil
}

// ladeBelegteBarcodes meldet, welche der Kandidaten bereits an einem Exemplar hängen.
func ladeBelegteBarcodes(ctx context.Context, db Datenbank, kandidaten []string) (map[string]bool, error) {
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
func hebeSequenzUeberBestand(ctx context.Context, db Datenbank) error {
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
