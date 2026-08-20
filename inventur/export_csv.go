package inventur

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"bibliothek/pkg/csvutil"
)

// handleExportCSV handles the GET /api/admin/books/export route.
//
// Streamt zeilenweise direkt in die HTTP-Antwort: Der Bestand (eine Zeile JE EXEMPLAR,
// die größte Zeilenmenge im System) wird NICHT mehr vollständig als [][]string im
// Speicher materialisiert und dann noch einmal kopiert-sanitisiert. Bei einem großen
// Katalog war das ein Speicher-Risiko; jetzt liegt immer nur eine Zeile im Speicher.
func (handler *APIHandler) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="bestand_export_%s.csv"`, time.Now().Format("2006-01-02")))

	// Write UTF-8 BOM so Excel opens it correctly with UTF-8
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF}) //nolint:errcheck

	writer := csv.NewWriter(w)
	writer.Comma = ';' // German Excel standard

	if err := writer.Write([]string{"Titel", "Autor", "Verlag", "ISBN", "Jahr", "Kategorie", "Barcode", "Zustand"}); err != nil {
		return // Header bereits gesendet — kein Fehler-Response mehr möglich
	}

	// Schutz vor Formel-Injection: Titel/Autor/Notizen stammen aus Importen und
	// Nutzereingaben und dürfen beim Öffnen in Excel keine Formel auslösen (je Zeile).
	err := handler.repo.StreamBooksForCSVExport(ctx, func(row []string) error {
		return writer.Write(csvutil.SanitizeRow(row))
	})
	writer.Flush()
	if err != nil {
		// Header ist längst raus; ein sauberer HTTP-Fehler geht nicht mehr. Die
		// abgebrochene Zeile signalisiert dem Client eine unvollständige Datei.
		return
	}
}

// StreamBooksForCSVExport liest alle Titel×Exemplare und ruft schreibe je Zeile auf —
// ohne die Gesamtmenge im Speicher zu halten. Bricht schreibe ab (z. B. Verbindung weg),
// endet der Stream mit diesem Fehler.
func (repo *BookRepository) StreamBooksForCSVExport(ctx context.Context, schreibe func(row []string) error) error {
	query := `
		SELECT
			bt.titel,
			coalesce(bt.autor, ''),
			coalesce(bt.verlag, ''),
			coalesce(bt.isbn, ''),
			coalesce(bt.erscheinungsjahr, 0),
			coalesce(bt.subject, ''),
			coalesce(be.barcode_id, ''),
			coalesce(be.zustand_notiz, '')
		FROM buecher_titel bt
		LEFT JOIN buecher_exemplare be ON bt.id = be.titel_id AND be.ist_ausgesondert = false
		ORDER BY bt.titel, be.barcode_id;
	`

	pgRows, err := repo.db.Query(ctx, query)
	if err != nil {
		return err
	}
	defer pgRows.Close()

	for pgRows.Next() {
		var titel, autor, verlag, isbn, subject, barcode, zustand string
		var jahr int

		if err := pgRows.Scan(&titel, &autor, &verlag, &isbn, &jahr, &subject, &barcode, &zustand); err != nil {
			return err
		}

		jahrStr := ""
		if jahr > 0 {
			jahrStr = strconv.Itoa(jahr)
		}

		// Prefix ISBN with a single quote so Excel treats it as text and doesn't remove leading zeros
		if isbn != "" {
			isbn = "'" + isbn
		}

		if err := schreibe([]string{titel, autor, verlag, isbn, jahrStr, subject, barcode, zustand}); err != nil {
			return err
		}
	}

	return pgRows.Err()
}
