package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// MaxSchuelerEtiketten deckelt einen Bogenauftrag.
//
// Jedes Etikett erzeugt ein Barcode-PNG; ohne Grenze legt ein versehentliches „alle
// markieren" über die ganze Schule eine Anfrage lahm, die minutenlang läuft und am Ende
// hunderte Seiten liefert, die niemand druckt. 600 sind mehr als zwei volle Jahrgänge
// und knapp 29 Bögen — darüber ist die Absicht fraglicher als die Grenze.
const MaxSchuelerEtiketten = 600

// SchuelerEtikettenRequest ist die Eingabe des Etikettenbogens.
//
// Nur IDs, keine Namen: Der Server holt die Angaben selbst. Sonst könnte jeder mit
// view_students einen Bogen mit beliebigen Namen zu echten Barcode-Nummern drucken —
// dieselbe Überlegung wie beim Ausweis, dessen Gültigkeitsdatum auch vom Server kommt
// und nicht aus dem Formular.
type SchuelerEtikettenRequest struct {
	FormatID      string   `json:"formatId,omitempty"`
	StartPosition int      `json:"startPosition,omitempty"`
	SchuelerIDs   []string `json:"schuelerIds,omitempty"`
	// Muster druckt EIN Beispiel-Etikett statt echter Schüler — der Testdruck des
	// Ausweis-Designers, mit dem man den Sitz auf dem Klebebogen prüft.
	Muster bool `json:"muster,omitempty"`
}

// PrintSchuelerEtikettenHandler liefert den Klebebogen als PDF.
// POST /api/print/schueler-etiketten
func (s *Server) PrintSchuelerEtikettenHandler(studentRepo repository.StudentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SchuelerEtikettenRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		if fehler := pruefeBogenParameter(&req); fehler != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fehler)
			return
		}

		etiketten, status, fehler := s.etikettenDesBogens(r, studentRepo, req)
		if fehler != nil {
			apierrors.SendHTTPError(w, status, fehler)
			return
		}

		pdf, err := GenerateSchuelerEtikettenPDF(req.FormatID, req.StartPosition, etiketten)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("etikettenbogen konnte nicht erzeugt werden: %w", err))
			return
		}

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, "inline; filename=\"schueler_etiketten.pdf\"")
		if err := pdf.Output(w); err != nil {
			log.Printf("Fehler beim Senden des Etikettenbogens: %v", err)
		}
	}
}

// pruefeBogenParameter prüft Format und Startposition und setzt die Vorgabe ein.
//
// Ein unbekanntes Format wird abgewiesen und NICHT still auf die Vorgabe gedreht: Wer
// avery_3475 anfordert und zweckform_l4760 bekommt, merkt es erst am verdruckten Bogen.
// Gleiche Regel wie bei den Buch-Etiketten (istBekanntesEtikettFormat).
func pruefeBogenParameter(req *SchuelerEtikettenRequest) error {
	if !istBekanntesEtikettFormat(req.FormatID) {
		return fmt.Errorf("unbekanntes Etikettenformat %q", req.FormatID)
	}
	format, _ := GetLabelFormat(req.FormatID)

	if req.StartPosition == 0 {
		req.StartPosition = 1
	}
	proSeite := format.Cols * format.Rows
	if req.StartPosition < 1 || req.StartPosition > proSeite {
		//nolint:staticcheck // ST1005: nutzer-sichtbarer Text im Druckdialog
		return fmt.Errorf("Startposition %d liegt außerhalb des Bogens (1 bis %d)", req.StartPosition, proSeite)
	}
	return nil
}

// etikettenDesBogens liefert den Inhalt: ein Muster oder die markierten Schüler.
// Zweiter Rückgabewert ist der HTTP-Status zum Fehler — ein Datenbankausfall ist kein
// Eingabefehler und darf nicht als 400 bei der Theke ankommen.
func (s *Server) etikettenDesBogens(r *http.Request, studentRepo repository.StudentRepository, req SchuelerEtikettenRequest) ([]SchuelerEtikett, int, error) {
	if req.Muster {
		return []SchuelerEtikett{MusterSchuelerEtikett}, 0, nil
	}

	switch {
	case len(req.SchuelerIDs) == 0:
		return nil, http.StatusBadRequest, errors.New("keine Schüler markiert")
	case len(req.SchuelerIDs) > MaxSchuelerEtiketten:
		return nil, http.StatusBadRequest,
			fmt.Errorf("%d Schüler markiert — höchstens %d je Bogenauftrag", len(req.SchuelerIDs), MaxSchuelerEtiketten)
	}

	zeilen, err := studentRepo.EtikettenZeilen(r.Context(), req.SchuelerIDs)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("schülerdaten konnten nicht gelesen werden: %w", err)
	}
	if len(zeilen) == 0 {
		//nolint:staticcheck // ST1005: nutzer-sichtbarer Text im Druckdialog
		return nil, http.StatusBadRequest, errors.New("Zu den markierten Schülern gibt es keine Daten mehr — inzwischen gelöscht?")
	}

	etiketten := make([]SchuelerEtikett, 0, len(zeilen))
	for _, z := range zeilen {
		etiketten = append(etiketten, SchuelerEtikett{
			BarcodeID: z.BarcodeID,
			Vorname:   z.Vorname,
			Nachname:  z.Nachname,
			Klasse:    z.Klasse,
		})
	}
	return etiketten, 0, nil
}
