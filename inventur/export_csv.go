package inventur

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"bibliothek/pkg/csvutil"
	"bibliothek/repository"
)

// exportSchlagwortTrenner trennt die Schlagworte eines Titels in ihrer einen Zelle.
const exportSchlagwortTrenner = " | "

// handleExportCSV handles the GET /api/admin/books/export route.
//
// Streamt zeilenweise direkt in die HTTP-Antwort: Der Bestand (eine Zeile JE EXEMPLAR,
// die größte Zeilenmenge im System) wird NICHT mehr vollständig als [][]string im
// Speicher materialisiert und dann noch einmal kopiert-sanitisiert. Bei einem großen
// Katalog war das ein Speicher-Risiko; jetzt liegt immer nur eine Zeile im Speicher.
func (handler *APIHandler) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	writer := csv.NewWriter(w)
	writer.Comma = ';' // German Excel standard

	// HTTP-Header und CSV-Kopf erst, NACHDEM die Query erfolgreich gestartet ist
	// (kopf-Callback): Scheitert die Datenbank, ist noch nichts gesendet und der
	// Client bekommt einen echten 500 — statt einer leeren 200er-CSV, die wie ein
	// gelungener (aber leerer) Export aussähe.
	kopfGesendet := false
	kopf := func() error {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="bestand_export_%s.csv"`, time.Now().Format("2006-01-02")))
		// Write UTF-8 BOM so Excel opens it correctly with UTF-8
		_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF}) //nolint:errcheck
		kopfGesendet = true
		return writer.Write([]string{"Titel", "Autor", "Verlag", "ISBN", "Jahr", "Kategorie", "Barcode", "Zustand", "Signatur", "Schlagworte", "Eigentum", "Standort"})
	}

	// Schutz vor Formel-Injection: Titel/Autor/Notizen stammen aus Importen und
	// Nutzereingaben und dürfen beim Öffnen in Excel keine Formel auslösen (je Zeile).
	err := handler.repo.StreamBooksForCSVExport(ctx, kopf, func(row []string) error {
		return writer.Write(csvutil.SanitizeRow(row))
	})
	writer.Flush()
	if err != nil {
		if !kopfGesendet {
			writeError(w, http.StatusInternalServerError, "fehler beim datenbank-export")
			return
		}
		// Header ist längst raus; ein sauberer HTTP-Fehler geht nicht mehr. Die
		// abgebrochene Zeile signalisiert dem Client eine unvollständige Datei.
		return
	}
}

// StreamBooksForCSVExport liest alle Titel×Exemplare und ruft schreibe je Zeile auf —
// ohne die Gesamtmenge im Speicher zu halten. kopf wird genau einmal NACH erfolgreichem
// Query-Start aufgerufen (der Aufrufer sendet dort seine Header). Bricht schreibe ab
// (z. B. Verbindung weg), endet der Stream mit diesem Fehler.
//
// Signatur und Schlagworte stehen am Titel und wiederholen sich je Exemplar; die Schlagworte
// teilen sich eine Zelle (exportSchlagwortTrenner), alphabetisch wie in der Buchmaske. Das
// Eigentum ist das des Exemplars nach repository.ExemplarTopfSQL, dieselbe Regel wie am
// Etikett. Der Standort steht am Exemplar. Eine Zeile ohne Exemplar hat weder Eigentum noch
// Standort.
func (repo *BookRepository) StreamBooksForCSVExport(ctx context.Context, kopf func() error, schreibe func(row []string) error) error {
	query := `
		SELECT
			t.titel,
			coalesce(t.autor, ''),
			coalesce(t.verlag, ''),
			coalesce(t.isbn, ''),
			coalesce(t.erscheinungsjahr, 0),
			coalesce(t.subject, ''),
			coalesce(e.barcode_id, ''),
			coalesce(e.zustand_notiz, ''),
			coalesce(t.signatur, ''),
			coalesce(sw.woerter, ''),
			CASE WHEN e.id IS NULL THEN '' ELSE ` + repository.ExemplarTopfSQL + ` END,
			coalesce(e.standort, '')
		FROM buecher_titel t
		LEFT JOIN buecher_exemplare e ON t.id = e.titel_id AND e.ist_ausgesondert = false
		` + repository.ExemplarTopfJoin + `
		LEFT JOIN (
			SELECT ts.titel_id, string_agg(s.wort, $1 ORDER BY lower(s.wort)) AS woerter
			FROM titel_schlagworte ts
			JOIN schlagworte s ON s.id = ts.schlagwort_id
			GROUP BY ts.titel_id
		) sw ON sw.titel_id = t.id
		ORDER BY t.titel, e.barcode_id;
	`

	pgRows, err := repo.db.Query(ctx, query, exportSchlagwortTrenner)
	if err != nil {
		return err
	}
	defer pgRows.Close()

	if err := kopf(); err != nil {
		return err
	}

	for pgRows.Next() {
		var titel, autor, verlag, isbn, subject, barcode, zustand, signatur, schlagworte, topf, standort string
		var jahr int

		if err := pgRows.Scan(&titel, &autor, &verlag, &isbn, &jahr, &subject, &barcode, &zustand, &signatur, &schlagworte, &topf, &standort); err != nil {
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

		zeile := []string{titel, autor, verlag, isbn, jahrStr, subject, barcode, zustand, signatur, schlagworte, repository.MittelTraeger(topf), standort}
		if err := schreibe(zeile); err != nil {
			return err
		}
	}

	return pgRows.Err()
}
