package katalog

import (
	"encoding/csv"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pkg/csvutil"
	"bibliothek/pkg/schulzeit"
)

// CSVExport ist die Tür GET /api/admin/books/export.
//
// Sie schreibt Zeile für Zeile direkt in die Antwort: Der Bestand (eine Zeile je Exemplar, die
// größte Menge an Zeilen im Programm) liegt nie vollständig als [][]string im Speicher und wird
// nicht noch einmal entschärft kopiert; im Speicher liegt immer nur eine Zeile.
func (t *Tueren) CSVExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		csvSchreiber := csv.NewWriter(w)
		csvSchreiber.Comma = ';' // German Excel standard

		// HTTP-Header und CSV-Kopf erst, nachdem die Query erfolgreich gestartet ist
		// (kopf-Callback): Scheitert die Datenbank, ist noch nichts gesendet und der
		// Client bekommt einen echten 500 — statt einer leeren 200er-CSV, die wie ein
		// gelungener (aber leerer) Export aussähe.
		kopfGesendet := false
		kopf := func() error {
			tuer.SetzeDateiKopf(w, "text/csv; charset=utf-8", tuer.Speichern,
				"bestand_export_"+schulzeit.Jetzt().Format(tuer.DatumsformatISO)+".csv")
			// Das BOM für UTF-8, damit Excel die Datei richtig öffnet.
			_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF}) //nolint:errcheck
			kopfGesendet = true
			return csvSchreiber.Write([]string{"Titel", "Autor", "Verlag", "ISBN", "Jahr", "Kategorie", "Barcode", "Zustand", "Signatur", "Schlagworte", "Eigentum", "Standort", "LMF-Nummer"})
		}

		// Schutz vor Formel-Injection: Titel/Autor/Notizen stammen aus Importen und
		// Nutzereingaben und dürfen beim Öffnen in Excel keine Formel auslösen (je Zeile).
		err := t.titelAbfragen().LadeTitelFuerCSVExport(ctx, kopf, func(zeile []string) error {
			return csvSchreiber.Write(csvutil.EntschaerfeZeile(zeile))
		})
		csvSchreiber.Flush()
		if err != nil {
			if !kopfGesendet {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "Der Export ließ sich nicht erstellen", err)
				return
			}
			// Header ist längst raus; ein sauberer HTTP-Fehler geht nicht mehr. Die
			// abgebrochene Zeile signalisiert dem Client eine unvollständige Datei.
			return
		}
	}
}
