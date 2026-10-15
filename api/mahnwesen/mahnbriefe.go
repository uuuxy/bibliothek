package mahnwesen

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/pdf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// ladeMahnVorlageText lädt die Vorlage des Mahnbriefs. Fehlt sie oder ist ein Teil leer, gilt
// der Vorgabetext: Der Editor lässt eine leere Vorlage zu, und der Brief hätte dann weder
// Betreff noch Anschreiben.
func (t *Tueren) ladeMahnVorlageText(ctx context.Context) (betreff, text string) {
	betreff, text, err := repository.LadeMahnVorlage(ctx, t.DB.Pool)
	if err != nil || strings.TrimSpace(betreff) == "" || strings.TrimSpace(text) == "" {
		betreff = "Mahnung: Überfällige Bücher"
		text = "Sehr geehrte Eltern von {{.Vorname}} {{.Nachname}},\n\nbitte geben Sie folgende Bücher umgehend in die Bibliothek zurück:\n\n{{.BuchListe}}\n\nVielen Dank."
	}
	return betreff, text
}

// ladeMahnbriefVorlage liest Betreff und Text aus der Vorlage und die Absenderzeile aus den
// Angaben zur Schule. Sind die Einstellungen nicht lesbar, kommt der Fehler zurück: Mit den
// Vorgaben hätte der Brief keinen Absender.
func (t *Tueren) ladeMahnbriefVorlage(ctx context.Context) (pdf.MahnbriefVorlage, error) {
	betreff, text := t.ladeMahnVorlageText(ctx)
	einstellungen, err := repository.NewEinstellungsAbfragen(t.DB.Pool).Lade(ctx)
	if err != nil {
		return pdf.MahnbriefVorlage{}, fmt.Errorf("einstellungen lesen: %w", err)
	}
	schule := pdf.SchuleInfo{
		Name:    einstellungen.SchuleName,
		Strasse: einstellungen.SchuleStrasse,
		PLZ:     einstellungen.SchulePLZ,
		Ort:     einstellungen.SchuleOrt,
	}
	return pdf.MahnbriefVorlage{Betreff: betreff, Text: text, Absender: schule.Absenderzeile()}, nil
}

// mahnbriefEmpfaenger füllt die Eingabe des Erzeugers aus den Zeilen der Abfrage: je Schüler
// ein Brief mit Anschrift und Büchern, in ihrer Reihenfolge.
func mahnbriefEmpfaenger(briefe []repository.MahnbriefEmpfaenger) []pdf.MahnbriefEmpfaenger {
	eingabe := make([]pdf.MahnbriefEmpfaenger, 0, len(briefe))
	for _, e := range briefe {
		buecher := make([]pdf.MahnbriefBuch, 0, len(e.Buecher))
		for _, b := range e.Buecher {
			buecher = append(buecher, pdf.MahnbriefBuch{
				Titel:            b.Titel,
				Barcode:          b.Barcode,
				AusgeliehenAm:    b.AusgeliehenAm,
				Frist:            b.Frist,
				TageUeberfaellig: b.TageUeberfaellig,
			})
		}
		eingabe = append(eingabe, pdf.MahnbriefEmpfaenger{
			Vorname:    e.Vorname,
			Nachname:   e.Nachname,
			Strasse:    e.Strasse,
			Hausnummer: e.Hausnummer,
			PLZ:        e.PLZ,
			Ort:        e.Ort,
			Buecher:    buecher,
		})
	}
	return eingabe
}

// erzeugeMahnbriefe zählt die Mahnung, liest in derselben Transaktion die Briefe
// und schreibt erst fest, wenn das PDF steht. Läge das Lesen vor dem Zählen, könnte ein
// inzwischen zurückgegebenes Buch auf dem Brief stehen, ohne gezählt zu sein. ok=false: Die
// Fehlerantwort ist geschrieben, das Zählen nimmt der Rollback zurück.
func (t *Tueren) erzeugeMahnbriefe(ctx context.Context, w http.ResponseWriter, ausleihIDs []string) ([]byte, bool) {
	// Vorlage und Absender vor der Transaktion: Sie hält eine Verbindung und Zeilensperren.
	vorlage, err := t.ladeMahnbriefVorlage(ctx)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return nil, false
	}

	tx, err := t.DB.Pool.Begin(ctx)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return nil, false
	}
	defer db.Verwerfe(ctx, tx)

	// Nur dieser Druck zählt die Mahnung, der Mail-Versand nicht (docs/invarianten.md §1).
	// Die Auswahl kommt aus der Oberfläche und kann älter sein als eine Verlängerung oder
	// Rückgabe.
	mahnAbfragen := repository.NewMahnwesenAbfragen(t.DB.Pool)
	gezaehlt, err := mahnAbfragen.ErhoeheMahnstufe(ctx, tx, ausleihIDs)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("fehler beim update der mahnstufen: %w", err))
		return nil, false
	}

	// Der Brief hängt nicht am Zählen: Ein Buch steigt höchstens einmal am Tag, gedruckt
	// wird es nach einem Papierstau auch ein zweites Mal.
	briefe, err := mahnAbfragen.LadeMahnbriefe(ctx, tx, ausleihIDs)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("fehler beim abrufen der daten für pdf: %w", err))
		return nil, false
	}
	if len(briefe) == 0 && gezaehlt > 0 {
		// Zählen und Lesen folgen derselben Bedingung, das ist nicht zu erwarten. Keine
		// Mahnung ohne Brief festschreiben.
		apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("dateninkonsistenz: keine PDF-Daten trotz aktualisierter Mahnstufen"))
		return nil, false
	}
	if len(briefe) == 0 {
		apierrors.AntworteFehler(w, http.StatusNotFound, fmt.Errorf("keine Ausleihe mit abgelaufener Frist in der Auswahl"))
		return nil, false
	}

	pdfDaten, err := pdf.GenerateMahnbriefePDF(mahnbriefEmpfaenger(briefe), vorlage)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("fehler beim generieren des pdfs: %w", err))
		return nil, false
	}

	if err := tx.Commit(ctx); err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("fehler beim commit der transaktion: %w", err))
		return nil, false
	}
	return pdfDaten, true
}

// MahnbriefeAnfrage benennt die Ausleihen, für die in einem Zug gemahnt werden soll.
// Die Auswahl kommt aus der Oberfläche, nicht aus einer serverseitigen Abfrage — wer
// mahnt, soll vorher sehen, wen es trifft.
type MahnbriefeAnfrage struct {
	AusleihIDs []string `json:"ausleih_ids" validate:"omitempty,dive,required,uuid_oder_leer"`
}

// MahnbriefeDrucken druckt die Mahnbriefe zu den gewählten Ausleihen und zählt die
// Mahnung, beides in einer Transaktion.
// POST /api/admin/mahnungen/bulk-print
func (t *Tueren) MahnbriefeDrucken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req MahnbriefeAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}

		if len(req.AusleihIDs) == 0 {
			apierrors.AntworteFehler(w, http.StatusBadRequest, fmt.Errorf("ausleih_ids array darf nicht leer sein"))
			return
		}

		pdfDaten, ok := t.erzeugeMahnbriefe(r.Context(), w, req.AusleihIDs)
		if !ok {
			return
		}

		dateiname := fmt.Sprintf("mahnbriefe_%s.pdf", schulzeit.Jetzt().Format(tuer.DatumsformatISO))

		w.Header().Set(tuer.HeaderContentType, tuer.ContentTypePDF)
		w.Header().Set(tuer.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, dateiname))
		w.Header().Set(tuer.HeaderContentLength, fmt.Sprint(len(pdfDaten)))

		http.ServeContent(w, r, dateiname, time.Now(), bytes.NewReader(pdfDaten))
	}
}
