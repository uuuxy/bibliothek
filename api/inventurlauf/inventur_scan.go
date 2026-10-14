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

const meldungUnbekannterBarcode = "Zu diesem Barcode gibt es kein Exemplar."

// ladeExemplarFuerScan lädt die für die Inventur-Logik nötigen Exemplardetails.
// ok=false: die Fehlerantwort (404 bei unbekanntem Barcode, sonst 500) wurde bereits geschrieben.
func ladeExemplarFuerScan(ctx context.Context, abfragen *repository.InventurAbfragen, w http.ResponseWriter, barcodeID string) (*repository.InventurExemplar, bool) {
	exemplar, err := abfragen.LadeExemplar(ctx, barcodeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Der Satz steht am Scanner unter „Unbekanntes Buch“; der Fehler der Datenbank geht ins Log.
			apierrors.SendHTTPErrorMitMeldung(w, http.StatusNotFound, meldungUnbekannterBarcode, err)
			return nil, false
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return nil, false
	}
	return exemplar, true
}

// scanWarnungen sammelt nicht-blockierende Hinweise zu einem gescannten Exemplar.
// Der Scope ist KEINE Warnung mehr, sondern eine harte Abweisung (siehe Handler): ein
// Buch außerhalb des Scopes darf gar nicht erst in dieser Session verbucht werden.
func scanWarnungen(istVerliehen bool) []string {
	var warnungen []string
	if istVerliehen {
		warnungen = append(warnungen, "Buch ist laut System aktuell ausgeliehen.")
	}
	return warnungen
}

// scanAnfrage is the payload for checking in an item during inventory.
type scanAnfrage struct {
	LaufID    string `json:"session_id" validate:"omitempty,uuid_oder_leer"`
	BarcodeID string `json:"barcode_id"`
}

// scanAntwort provides feedback to the frontend after a scan attempt.
type scanAntwort struct {
	BarcodeID string   `json:"barcode_id"`
	Titel     string   `json:"titel"`
	CoverURL  string   `json:"cover_url,omitempty"`
	Status    string   `json:"status"`
	Warnungen []string `json:"warnungen,omitempty"`
}

// ScanVerbuchen verbucht einen Exemplar-Scan in einer laufenden Session.
// @Summary      Scan a copy during inventory
// @Description  Records that a physical copy was present, bound to the given session.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        body  body      scanAnfrage   true  "Session and barcode"
// @Success      200   {object}  scanAntwort
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /inventur/scan [post]
func (t *Tueren) ScanVerbuchen() http.HandlerFunc {
	return t.verbucheScan
}

// verbucheScan verbucht einen einzelnen Inventur-Scan. Bewusst als Top-Level-Methode
// statt Inline-Closure, damit die vielen Frühabbrüche nicht zusätzlich als Verschachtelung
// zählen (SonarQube S3776).
func (t *Tueren) verbucheScan(w http.ResponseWriter, r *http.Request) {
	var req scanAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return
	}
	if req.LaufID == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("session_id fehlt"))
		return
	}

	ctx := r.Context()
	abfragen := repository.NewInventurAbfragen(t.DB.Pool)

	// Session muss offen sein — sonst ist der Scan gegenstandslos (404).
	lauf, err := abfragen.LadeLauf(ctx, req.LaufID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("keine laufende Inventur zu dieser Session"))
			return
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	exemplar, ok := ladeExemplarFuerScan(ctx, abfragen, w, req.BarcodeID)
	if !ok {
		return
	}
	if exemplar.IstAusgesondert {
		apierrors.SendHTTPError(w, http.StatusConflict, fmt.Errorf("exemplar %s ist bereits ausgesondert", req.BarcodeID))
		return
	}

	imBereich, err := abfragen.IstExemplarImBereich(ctx, exemplar.ExemplarID, lauf.Bereich())
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	// Cross-Contamination-Schutz: Ein Exemplar außerhalb des Session-Scopes darf NICHT
	// in dieser Session verbucht werden (409, nicht stillschweigend absorbieren). Täte es
	// das doch, läge der Scan session-gebunden im fremden Scope — beim Abschluss der
	// ZUSTÄNDIGEN Fach-Session fehlte das Exemplar dann in deren Erfassungen und würde
	// dort fälschlich als VERLUST gebucht, obwohl das Buch physisch vorliegt.
	//
	// Bewusst als STRUKTURIERTE 409-Antwort (nicht als rohe Fehlermeldung): Das Buch
	// existiert ja, es liegt nur im falschen Scope. Das Frontend soll den echten Titel und
	// den Warntext zeigen — nicht "Unbekanntes Buch" — und den Scan als Warnung rendern,
	// ohne ihn mitzuzählen. Der Status "ausser_scope" macht den Fall clientseitig
	// unterscheidbar.
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
		// Die Session wurde inzwischen abgeschlossen (Multi-Scanner-Wettlauf) — der Scan
		// ist gegenstandslos, aber kein Serverfehler. 409 mit klarer Meldung.
		if errors.Is(err, repository.ErrInventurAbgeschlossen) {
			apierrors.SendHTTPError(w, http.StatusConflict,
				errors.New("die Inventur wurde soeben abgeschlossen — dieser Scan wurde nicht mehr erfasst"))
			return
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
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
