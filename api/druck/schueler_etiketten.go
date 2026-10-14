package druck

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/repository"
)

// MaxSchuelerEtiketten deckelt einen Bogenauftrag.
//
// Jedes Etikett erzeugt ein Barcode-PNG; ohne Grenze legt ein versehentliches „alle
// markieren" über die ganze Schule eine Anfrage lahm, die minutenlang läuft und am Ende
// hunderte Seiten liefert, die niemand druckt. 600 sind mehr als zwei volle Jahrgänge
// und knapp 29 Bögen — darüber ist die Absicht fraglicher als die Grenze.
const MaxSchuelerEtiketten = 600

// schuelerEtikettenAnfrage ist die Eingabe des Etikettenbogens.
//
// Nur IDs, keine Namen: Der Server holt die Angaben selbst. Sonst könnte jeder mit
// view_students einen Bogen mit beliebigen Namen zu echten Barcode-Nummern drucken —
// dieselbe Überlegung wie beim Ausweis, dessen Gültigkeitsdatum auch vom Server kommt
// und nicht aus dem Formular.
type schuelerEtikettenAnfrage struct {
	FormatID      string   `json:"formatId,omitempty"`
	Startposition int      `json:"startPosition,omitempty"`
	SchuelerIDs   []string `json:"schuelerIds,omitempty" validate:"omitempty,dive,required,uuid_oder_leer"`
	// Muster druckt ein Beispiel-Etikett statt echter Schüler — der Testdruck des
	// Ausweis-Designers, mit dem man den Sitz auf dem Klebebogen prüft.
	Muster bool `json:"muster,omitempty"`
}

// SchuelerEtiketten liefert den Klebebogen als PDF.
// POST /api/print/schueler-etiketten
func (t *Tueren) SchuelerEtiketten(leserAbfragen repository.LeserAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req schuelerEtikettenAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}

		if fehler := pruefeBogenParameter(&req); fehler != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fehler)
			return
		}

		etiketten, status, fehler := t.etikettenDesBogens(r, leserAbfragen, req)
		if fehler != nil {
			apierrors.SendHTTPError(w, status, fehler)
			return
		}

		bogen, err := pdf.GenerateSchuelerEtikettenPDF(req.FormatID, req.Startposition, etiketten)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("etikettenbogen konnte nicht erzeugt werden: %w", err))
			return
		}

		w.Header().Set(tuer.HeaderContentType, tuer.ContentTypePDF)
		w.Header().Set(tuer.HeaderContentDisposition, "inline; filename=\"schueler_etiketten.pdf\"")
		if err := bogen.Output(w); err != nil {
			log.Printf("Fehler beim Senden des Etikettenbogens: %v", err)
		}
	}
}

// pruefeBogenParameter prüft Format und Startposition und setzt die Vorgabe ein.
//
// Ein unbekanntes Format wird abgewiesen und nicht still auf die Vorgabe gedreht: Wer
// avery_3475 anfordert und zweckform_l4760 bekommt, merkt es erst am verdruckten Bogen.
// Gleiche Regel wie bei den Buch-Etiketten (pdf.IstBekanntesEtikettFormat).
func pruefeBogenParameter(req *schuelerEtikettenAnfrage) error {
	if !pdf.IstBekanntesEtikettFormat(req.FormatID) {
		return fmt.Errorf("unbekanntes Etikettenformat %q", req.FormatID)
	}
	format, _ := pdf.GetLabelFormat(req.FormatID)

	if req.Startposition == 0 {
		req.Startposition = 1
	}
	proSeite := format.Cols * format.Rows
	if req.Startposition < 1 || req.Startposition > proSeite {
		//nolint:staticcheck // ST1005: nutzer-sichtbarer Text im Druckdialog
		return fmt.Errorf("Startposition %d liegt außerhalb des Bogens (1 bis %d)", req.Startposition, proSeite)
	}
	return nil
}

// etikettenDesBogens liefert den Inhalt: ein Muster oder die markierten Schüler.
// Zweiter Rückgabewert ist der HTTP-Status zum Fehler — ein Datenbankausfall ist kein
// Eingabefehler und darf nicht als 400 bei der Theke ankommen.
func (t *Tueren) etikettenDesBogens(r *http.Request, leserAbfragen repository.LeserAbfragen, req schuelerEtikettenAnfrage) ([]pdf.SchuelerEtikett, int, error) {
	if req.Muster {
		return []pdf.SchuelerEtikett{pdf.MusterSchuelerEtikett}, 0, nil
	}

	switch {
	case len(req.SchuelerIDs) == 0:
		return nil, http.StatusBadRequest, errors.New("keine Schüler markiert")
	case len(req.SchuelerIDs) > MaxSchuelerEtiketten:
		return nil, http.StatusBadRequest,
			fmt.Errorf("%d Schüler markiert — höchstens %d je Bogenauftrag", len(req.SchuelerIDs), MaxSchuelerEtiketten)
	}

	zeilen, err := leserAbfragen.LadeEtikettZeilen(r.Context(), req.SchuelerIDs)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("schülerdaten konnten nicht gelesen werden: %w", err)
	}
	if len(zeilen) == 0 {
		//nolint:staticcheck // ST1005: nutzer-sichtbarer Text im Druckdialog
		return nil, http.StatusBadRequest, errors.New("Zu den markierten Schülern gibt es keine Daten mehr — inzwischen gelöscht?")
	}

	etiketten := make([]pdf.SchuelerEtikett, 0, len(zeilen))
	for _, z := range zeilen {
		etiketten = append(etiketten, pdf.SchuelerEtikett{
			BarcodeID: z.BarcodeID,
			Vorname:   z.Vorname,
			Nachname:  z.Nachname,
			Klasse:    z.Klasse,
		})
	}
	return etiketten, 0, nil
}
