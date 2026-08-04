package littera

import (
	"context"
	"fmt"

	"bibliothek/internal/uebernahme"
)

// klaereBarcodes bestimmt für jedes Exemplar den Barcode, mit dem es geschrieben wird.
//
// Das geschieht VOR dem Schreiben und in einem Zug, aus dem gleichen Grund, aus dem
// KlaereISBN vor dem Titel-INSERT läuft: buecher_exemplare.barcode_id ist UNIQUE, und ein
// 23505 mitten in einem Titel kostet nicht das Exemplar, sondern den ganzen Titel — bei
// einem Klassensatz mit 354 Stück also 354 Bücher wegen einer doppelten Nummer.
//
// Im Altbestand tragen 13 Exemplare eine von 5 doppelt vergebenen Nummern. Sie bekommen
// hier eine frische Nummer aus barcode_seq und eine WARNUNG — abgewertet übernommen
// statt verloren, dieselbe Haltung wie bei der ISBN.
//
// Auch der VORHANDENE Bestand wird eingelesen. Läuft der Import in eine Datenbank, in der
// schon Exemplare stehen, wäre jede Überschneidung sonst ein Titelverlust.
func (s *Schreiber) klaereBarcodes(
	ctx context.Context, exemplare []Exemplar, fremd map[string]string,
) (map[string]string, error) {
	belegt, err := s.vorhandeneBarcodes(ctx)
	if err != nil {
		return nil, err
	}

	zuweisung := make(map[string]string, len(exemplare))
	var brauchtNeue []Exemplar

	for _, e := range exemplare {
		wunsch := s.barcodeWunsch(e, fremd[e.ID])
		switch {
		case wunsch == "":
			brauchtNeue = append(brauchtNeue, e)
		case belegt[wunsch]:
			s.prot.Warnung(e.ID, wunsch,
				"Barcode bereits vergeben – Exemplar bekommt eine neue Nummer aus barcode_seq")
			brauchtNeue = append(brauchtNeue, e)
		default:
			belegt[wunsch] = true
			zuweisung[e.ID] = wunsch
		}
	}

	if err := s.vergebeNeueBarcodes(ctx, brauchtNeue, belegt, zuweisung); err != nil {
		return nil, err
	}
	return zuweisung, nil
}

// barcodeWunsch liefert die gewünschte Nummer oder "" für „bitte neu vergeben".
//
// Geliefert wird der Wert, den ein SCANNER vom Etikett liest — also der EAN-13, nicht die
// im Klartext daneben gedruckte Exemplarnummer. Steht in Litteras FremdBarcode ein
// Ersatzetikett, gewinnt das.
func (s *Schreiber) barcodeWunsch(e Exemplar, fremd string) string {
	if s.opt.Barcodes != BarcodeLittera {
		return ""
	}
	nummer, ok := EtikettBarcode(e.Exemplarnummer, e.Bibliotheksnummer)
	if !ok {
		s.prot.Warnung(e.ID, e.Exemplarnummer,
			"aus Exemplar- und Bibliotheksnummer lässt sich kein Etikett-Barcode bilden – "+
				"neue Nummer aus barcode_seq, das Buch braucht ein neues Etikett")
		return ""
	}
	// Ein hinterlegter Fremdbarcode gewinnt: Dann klebt ein Ersatzetikett am Buch, und
	// der gerechnete EAN-13 stünde nirgends.
	if fremd != "" {
		nummer = fremd
	}
	if len(nummer) > uebernahme.MaxBarcode {
		s.prot.Warnung(e.ID, nummer,
			"Barcode ist länger als die Spalte barcode_id – neue Nummer aus barcode_seq")
		return ""
	}
	return nummer
}

func (s *Schreiber) vorhandeneBarcodes(ctx context.Context) (map[string]bool, error) {
	rows, err := s.pool.Query(ctx, `SELECT barcode_id FROM buecher_exemplare`)
	if err != nil {
		return nil, fmt.Errorf("konnte die vorhandenen Barcodes nicht lesen: %w", err)
	}
	defer rows.Close()

	belegt := map[string]bool{}
	for rows.Next() {
		var bc string
		if err := rows.Scan(&bc); err != nil {
			return nil, fmt.Errorf("konnte die vorhandenen Barcodes nicht lesen: %w", err)
		}
		belegt[bc] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("konnte die vorhandenen Barcodes nicht lesen: %w", err)
	}
	return belegt, nil
}

// vergebeNeueBarcodes zieht die fehlenden Nummern aus barcode_seq.
//
// Bewusst aus der Postgres-Sequenz und nicht aus einem Zähler in Go: Aus derselben
// Sequenz bedient sich die laufende Anwendung (repository/book_inventory.go). Ein
// eigener Zähler im Werkzeug wäre am Tag nach dem Import wieder eingeholt worden.
func (s *Schreiber) vergebeNeueBarcodes(
	ctx context.Context, exemplare []Exemplar, belegt map[string]bool, zuweisung map[string]string,
) error {
	if len(exemplare) == 0 {
		return nil
	}
	vorrat, err := s.zieheBarcodes(ctx, len(exemplare))
	if err != nil {
		return err
	}

	for _, e := range exemplare {
		bc, rest, err := s.naechsterFreier(ctx, vorrat, belegt)
		if err != nil {
			return err
		}
		vorrat = rest
		belegt[bc] = true
		zuweisung[e.ID] = bc
	}
	return nil
}

// naechsterFreier nimmt die erste noch unbelegte Nummer aus dem Vorrat und zieht nach,
// wenn er leer ist.
//
// Die Prüfung gegen belegt ist nötig, weil barcode_seq nur die von der Anwendung
// erzeugten B-Nummern kennt: Ein von Hand vergebenes „B-00042" steht in der Tabelle,
// aber nicht in der Sequenz. Die Sequenz wächst streng, der Vorrat läuft also nach
// endlich vielen Versuchen an den vorhandenen Nummern vorbei.
func (s *Schreiber) naechsterFreier(
	ctx context.Context, vorrat []string, belegt map[string]bool,
) (string, []string, error) {
	for versuch := 0; versuch <= len(belegt)+1; versuch++ {
		for i, bc := range vorrat {
			if !belegt[bc] {
				return bc, vorrat[i+1:], nil
			}
		}
		nachschlag, err := s.zieheBarcodes(ctx, barcodeNachschlag)
		if err != nil {
			return "", nil, err
		}
		vorrat = nachschlag
	}
	return "", nil, fmt.Errorf("barcode_seq liefert nur bereits vergebene Nummern – Sequenz prüfen")
}

const barcodeNachschlag = 256

func (s *Schreiber) zieheBarcodes(ctx context.Context, anzahl int) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT 'B-' || LPAD(nextval('barcode_seq')::TEXT, 5, '0') FROM generate_series(1, $1)`, anzahl)
	if err != nil {
		return nil, fmt.Errorf("konnte keine neuen Barcodes aus barcode_seq ziehen: %w", err)
	}
	defer rows.Close()

	codes := make([]string, 0, anzahl)
	for rows.Next() {
		var bc string
		if err := rows.Scan(&bc); err != nil {
			return nil, fmt.Errorf("konnte keine neuen Barcodes aus barcode_seq ziehen: %w", err)
		}
		codes = append(codes, bc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("konnte keine neuen Barcodes aus barcode_seq ziehen: %w", err)
	}
	return codes, nil
}

// vorhandeneISBNs füllt die Dublettenerkennung mit dem, was schon in der Datenbank steht.
//
// buecher_titel.isbn ist UNIQUE. Ohne diese Vorbelegung liefe ein Titel, dessen ISBN
// bereits vergeben ist, in einen 23505 — und ginge samt aller Exemplare verloren, obwohl
// eine verworfene ISBN völlig ausgereicht hätte.
func (s *Schreiber) vorhandeneISBNs(ctx context.Context) (map[string]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT isbn FROM buecher_titel WHERE isbn IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("konnte die vorhandenen ISBNs nicht lesen: %w", err)
	}
	defer rows.Close()

	gesehen := map[string]string{}
	for rows.Next() {
		var isbn string
		if err := rows.Scan(&isbn); err != nil {
			return nil, fmt.Errorf("konnte die vorhandenen ISBNs nicht lesen: %w", err)
		}
		gesehen[uebernahme.NormalisiereISBN(isbn)] = "(Bestand)"
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("konnte die vorhandenen ISBNs nicht lesen: %w", err)
	}
	return gesehen, nil
}
