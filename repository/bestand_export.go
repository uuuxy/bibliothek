package repository

import (
	"bibliothek/pkg/mitteltopf"
	"context"
	"strconv"
)

// ExportSchlagwortTrenner trennt die Schlagworte eines Titels in ihrer einen Zelle.
const ExportSchlagwortTrenner = " | "

// LadeTitelFuerCSVExport liest alle Titel×Exemplare und ruft schreibe je Zeile auf —
// ohne die Gesamtmenge im Speicher zu halten. kopf wird genau einmal nach erfolgreichem
// Query-Start aufgerufen (der Aufrufer sendet dort seine Header). Bricht schreibe ab
// (z. B. Verbindung weg), endet der Stream mit diesem Fehler.
//
// Signatur und Schlagworte stehen am Titel und wiederholen sich je Exemplar; die Schlagworte
// teilen sich eine Zelle (exportSchlagwortTrenner), alphabetisch wie in der Buchmaske. Das
// Eigentum ist das des Exemplars nach ExemplarTopfSQL, dieselbe Regel wie am
// Etikett. Der Standort steht am Exemplar. Eine Zeile ohne Exemplar hat weder Eigentum noch
// Standort. Die LMF-Nummer ist das alte Etikett eines Lernmittels (lmf_barcode): Bis es
// überklebt ist, trägt das Buch nur sie, und nur über sie fände ein anderes Programm es.
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
			CASE WHEN e.id IS NULL THEN '' ELSE ` + ExemplarTopfSQL + ` END,
			coalesce(e.standort, ''),
			coalesce(e.lmf_barcode, '')
		FROM buecher_titel t
		LEFT JOIN buecher_exemplare e ON t.id = e.titel_id AND e.ist_ausgesondert = false
		` + ExemplarTopfJoin + `
		LEFT JOIN (
			SELECT ts.titel_id, string_agg(s.wort, $1 ORDER BY lower(s.wort) COLLATE deutsch) AS woerter
			FROM titel_schlagworte ts
			JOIN schlagworte s ON s.id = ts.schlagwort_id
			GROUP BY ts.titel_id
		) sw ON sw.titel_id = t.id
		ORDER BY t.titel COLLATE deutsch, e.barcode_id;
	`

	pgZeilen, err := a.pool.Query(ctx, abfrage, ExportSchlagwortTrenner)
	if err != nil {
		return err
	}
	defer pgZeilen.Close()

	if err := kopf(); err != nil {
		return err
	}

	for pgZeilen.Next() {
		var titel, autor, verlag, isbn, fach, barcode, zustand, signatur, schlagworte, topf, standort, lmfNummer string
		var jahr int

		if err := pgZeilen.Scan(&titel, &autor, &verlag, &isbn, &jahr, &fach, &barcode, &zustand, &signatur, &schlagworte, &topf, &standort, &lmfNummer); err != nil {
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

		zeile := []string{titel, autor, verlag, isbn, jahrStr, fach, barcode, zustand, signatur, schlagworte, mitteltopf.Traeger(topf), standort, lmfNummer}
		if err := schreibe(zeile); err != nil {
			return err
		}
	}

	return pgZeilen.Err()
}
