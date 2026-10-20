package inventurlauf

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

const meldungUnbekannterBarcode = "Zu diesem Barcode gibt es kein Exemplar"

// ladeExemplarFuerScan lädt die Angaben zum Exemplar, die der Scan braucht. Bei false ist die
// Fehlerantwort schon geschrieben: 404 bei unbekanntem Barcode, sonst 500.
func ladeExemplarFuerScan(ctx context.Context, abfragen *repository.InventurAbfragen, w http.ResponseWriter, barcodeID string) (*repository.InventurExemplar, bool) {
	exemplar, err := abfragen.LadeExemplar(ctx, barcodeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Der Satz steht am Scanner unter „Unbekanntes Buch“; der Fehler der Datenbank geht
			// ins Log.
			apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, meldungUnbekannterBarcode, err)
			return nil, false
		}
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return nil, false
	}
	return exemplar, true
}

// scanWarnungen sammelt Hinweise zu einem gescannten Exemplar, die den Scan nicht aufhalten.
// Ein Buch außerhalb des Bereichs ist keine Warnung, sondern wird abgewiesen (verbucheScan).
func scanWarnungen(istVerliehen bool) []string {
	var warnungen []string
	if istVerliehen {
		warnungen = append(warnungen, "Buch ist laut System aktuell ausgeliehen.")
	}
	return warnungen
}

// scanAnfrage nennt den Lauf und den gescannten Barcode.
type scanAnfrage struct {
	LaufID    string `json:"session_id" validate:"omitempty,uuid_oder_leer"`
	BarcodeID string `json:"barcode_id"`
}

// scanAntwort sagt der Oberfläche, was aus dem Scan wurde.
type scanAntwort struct {
	BarcodeID string   `json:"barcode_id"`
	Titel     string   `json:"titel"`
	CoverURL  string   `json:"cover_url,omitempty"`
	Status    string   `json:"status"`
	Warnungen []string `json:"warnungen,omitempty"`
}

// ScanVerbuchen verbucht den Scan eines Exemplars in einem offenen Lauf.
func (t *Tueren) ScanVerbuchen() http.HandlerFunc {
	return t.verbucheScan
}

// verbucheScan verbucht einen Scan. Eine eigene Methode statt einer Funktion in der Tür,
// damit die vielen frühen Rücksprünge nicht zusätzlich als Verschachtelung zählen (SonarQube
// S3776).
func (t *Tueren) verbucheScan(w http.ResponseWriter, r *http.Request) {
	var req scanAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return
	}
	if req.LaufID == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung des Inventurlaufs fehlt", nil)
		return
	}

	ctx := r.Context()
	abfragen := repository.NewInventurAbfragen(t.DB.Pool)

	// Der Lauf muss offen sein, sonst ist der Scan gegenstandslos (404).
	lauf, err := abfragen.LadeLauf(ctx, req.LaufID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Kein laufender Inventurlauf mit dieser Kennung", nil)
			return
		}
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	exemplar, ok := ladeExemplarFuerScan(ctx, abfragen, w, req.BarcodeID)
	if !ok {
		return
	}
	if exemplar.IstAusgesondert {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, fmt.Sprintf("Das Exemplar %s ist bereits ausgesondert", req.BarcodeID), nil)
		return
	}

	imBereich, err := abfragen.IstExemplarImBereich(ctx, exemplar.ExemplarID, lauf.Bereich())
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	// Ein Exemplar außerhalb des Bereichs wird nicht verbucht (409): Läge der Scan in diesem
	// Lauf, fehlte das Exemplar beim Abschluss des zuständigen Laufs und würde dort als
	// Verlust gebucht, obwohl es im Regal steht.
	//
	// Die 409 trägt Titel, Warnung und den Status "ausser_scope", damit die Oberfläche das
	// Buch mit der Warnung zeigt statt "Unbekanntes Buch" und den Scan nicht mitzählt.
	if !imBereich {
		tuer.AntworteJSON(w, http.StatusConflict, scanAntwort{
			BarcodeID: req.BarcodeID,
			Titel:     exemplar.Titel,
			CoverURL:  exemplar.CoverURL,
			Status:    "ausser_scope",
			Warnungen: []string{"Buch gehört nicht zum Bereich dieser Inventur — es wurde NICHT erfasst. Bitte im zuständigen Inventur-Bereich scannen."},
		})
		return
	}

	if err := abfragen.VerbucheScan(ctx, req.LaufID, exemplar.ExemplarID); err != nil {
		// Der Lauf wurde inzwischen an einem anderen Scanner abgeschlossen: Der Scan ist
		// gegenstandslos, aber kein Fehler des Servers (409).
		if errors.Is(err, repository.ErrInventurAbgeschlossen) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, "Die Inventur wurde gerade abgeschlossen. Dieser Scan wurde nicht mehr erfasst.", nil)
			return
		}
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	tuer.AntworteJSON(w, http.StatusOK, scanAntwort{
		BarcodeID: req.BarcodeID,
		Titel:     exemplar.Titel,
		CoverURL:  exemplar.CoverURL,
		Status:    "erfasst",
		Warnungen: scanWarnungen(exemplar.IstVerliehen),
	})
}
