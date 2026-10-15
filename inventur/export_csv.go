package inventur

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"

	"bibliothek/pkg/csvutil"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// exportSchlagwortTrenner trennt die Schlagworte eines Titels in ihrer einen Zelle.
const exportSchlagwortTrenner = " | "

// CSVExport ist die Tür GET /api/admin/books/export.
//
// Sie schreibt Zeile für Zeile direkt in die Antwort: Der Bestand (eine Zeile je Exemplar, die
// größte Menge an Zeilen im Programm) liegt nie vollständig als [][]string im Speicher und wird
// nicht noch einmal entschärft kopiert; im Speicher liegt immer nur eine Zeile.
func (t *Tueren) CSVExport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	csvSchreiber := csv.NewWriter(w)
	csvSchreiber.Comma = ';' // German Excel standard

	// HTTP-Header und CSV-Kopf erst, nachdem die Query erfolgreich gestartet ist
	// (kopf-Callback): Scheitert die Datenbank, ist noch nichts gesendet und der
	// Client bekommt einen echten 500 — statt einer leeren 200er-CSV, die wie ein
	// gelungener (aber leerer) Export aussähe.
	kopfGesendet := false
	kopf := func() error {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="bestand_export_%s.csv"`, schulzeit.Jetzt().Format("2006-01-02")))
		// Das BOM für UTF-8, damit Excel die Datei richtig öffnet.
		_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF}) //nolint:errcheck
		kopfGesendet = true
		return csvSchreiber.Write([]string{"Titel", "Autor", "Verlag", "ISBN", "Jahr", "Kategorie", "Barcode", "Zustand", "Signatur", "Schlagworte", "Eigentum", "Standort"})
	}

	// Schutz vor Formel-Injection: Titel/Autor/Notizen stammen aus Importen und
	// Nutzereingaben und dürfen beim Öffnen in Excel keine Formel auslösen (je Zeile).
	err := t.abfragen.LadeTitelFuerCSVExport(ctx, kopf, func(zeile []string) error {
		return csvSchreiber.Write(csvutil.EntschaerfeZeile(zeile))
	})
	csvSchreiber.Flush()
	if err != nil {
		if !kopfGesendet {
			schreibeFehler(w, http.StatusInternalServerError, "fehler beim datenbank-export")
			return
		}
		// Header ist längst raus; ein sauberer HTTP-Fehler geht nicht mehr. Die
		// abgebrochene Zeile signalisiert dem Client eine unvollständige Datei.
		return
	}
}

// LadeTitelFuerCSVExport liest alle Titel×Exemplare und ruft schreibe je Zeile auf —
// ohne die Gesamtmenge im Speicher zu halten. kopf wird genau einmal nach erfolgreichem
// Query-Start aufgerufen (der Aufrufer sendet dort seine Header). Bricht schreibe ab
// (z. B. Verbindung weg), endet der Stream mit diesem Fehler.
//
// Signatur und Schlagworte stehen am Titel und wiederholen sich je Exemplar; die Schlagworte
// teilen sich eine Zelle (exportSchlagwortTrenner), alphabetisch wie in der Buchmaske. Das
// Eigentum ist das des Exemplars nach repository.ExemplarTopfSQL, dieselbe Regel wie am
// Etikett. Der Standort steht am Exemplar. Eine Zeile ohne Exemplar hat weder Eigentum noch
// Standort.
func (a *TitelAbfragen) LadeTitelFuerCSVExport(ctx context.Context, kopf func() error, schreibe func(zeile []string) error) error {
	abfrage := `
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

	pgZeilen, err := a.pool.Query(ctx, abfrage, exportSchlagwortTrenner)
	if err != nil {
		return err
	}
	defer pgZeilen.Close()

	if err := kopf(); err != nil {
		return err
	}

	for pgZeilen.Next() {
		var titel, autor, verlag, isbn, fach, barcode, zustand, signatur, schlagworte, topf, standort string
		var jahr int

		if err := pgZeilen.Scan(&titel, &autor, &verlag, &isbn, &jahr, &fach, &barcode, &zustand, &signatur, &schlagworte, &topf, &standort); err != nil {
			return err
		}

		jahrStr := ""
		if jahr > 0 {
			jahrStr = strconv.Itoa(jahr)
		}

		// Ein Hochkomma vor der ISBN, damit Excel sie als Text liest und führende Nullen behält.
		if isbn != "" {
			isbn = "'" + isbn
		}

		zeile := []string{titel, autor, verlag, isbn, jahrStr, fach, barcode, zustand, signatur, schlagworte, mitteltopf.Traeger(topf), standort}
		if err := schreibe(zeile); err != nil {
			return err
		}
	}

	return pgZeilen.Err()
}
