package schaden

import (
	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/pdf"
	"bibliothek/repository"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/google/uuid"
)

// queryRechnungItems lädt die offenen Forderungen eines Schülers ohne Bescheid als
// Rechnungspositionen (repository.ListeOffeneForderungenOhneBescheid).
func queryRechnungItems(ctx context.Context, dbPool db.PgxPoolIface, schuelerID uuid.UUID) ([]pdf.RechnungItem, error) {
	posten, err := repository.ListeOffeneForderungenOhneBescheid(ctx, dbPool, schuelerID)
	if err != nil {
		return nil, err
	}
	var items []pdf.RechnungItem
	for _, p := range posten {
		items = append(items, pdf.RechnungItem(p))
	}
	return items, nil
}

// PrintRechnungHandler generates the invoice for lost books of a student.
func PrintRechnungHandler(dbPool db.PgxPoolIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handlePrintRechnung(w, r, dbPool)
	}
}

// handlePrintRechnung lädt Anschrift und offene Forderungen ohne Bescheid und setzt den Brief.
func handlePrintRechnung(w http.ResponseWriter, r *http.Request, dbPool db.PgxPoolIface) {
	ctx := r.Context()

	// extract schueler_id from the URL path: /api/print/rechnung/{id}
	idStr := r.PathValue("schueler_id")
	schuelerID, err := uuid.Parse(idStr)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, err)
		return
	}

	// Anschrift mitladen: Die Rechnung ist wie der Eltern-Mahnbrief ein Brief für das
	// DIN-Fensterkuvert (VVT-Zweck „gedruckte Rechnung / Elternbrief"). COALESCE ist
	// Pflicht: Die Spalten sind nullbar, die Go-Strings nicht.
	anschrift, err := repository.LadeBriefAnschrift(ctx, dbPool, schuelerID)
	s := pdf.Schueler(anschrift)
	if errors.Is(err, pgx.ErrNoRows) {
		apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("schüler nicht gefunden"))
		return
	}
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	items, err := queryRechnungItems(ctx, dbPool, schuelerID)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	if len(items) == 0 {
		// Stehen die offenen Forderungen alle auf einem Bescheid, sagt die Meldung das.
		// Sonst hieße es „keine offenen Schadensfälle", während die Akte offene Beträge zeigt.
		aufBescheid, err := repository.HatOffeneForderungAufBescheid(ctx, dbPool, schuelerID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if aufBescheid {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New(
				"die offenen Forderungen stehen auf einem Schadensersatz-Bescheid — dafür gilt der Bescheid, keine Ersatzforderung"))
			return
		}
		apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("no open damage records found for student"))
		return
	}

	settingsRepo := repository.NewSystemSettingsRepository(dbPool)
	settings, _ := settingsRepo.GetSettings(ctx) //nolint:errcheck
	schule := pdf.SchuleInfo{
		Name:    settings.SchuleName,
		Strasse: settings.SchuleStrasse,
		PLZ:     settings.SchulePLZ,
		Ort:     settings.SchuleOrt,
	}

	// Zahlstelle und Bankverbindung des Landes aus derselben Einstellung wie im
	// Bescheid — eine zweite Kontoangabe im selben Haus wäre eine zweite Wahrheit.
	angaben := repository.BescheidAngabenAus(settings)
	zahlung := pdf.Zahlungsangaben{Zahlstelle: angaben.Zahlstelle, Bankverbindung: angaben.Bankverbindung}

	pdfBytes, err := pdf.GenerateRechnung(s, items, schule, zahlung)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set(tuer.HeaderContentType, tuer.ContentTypePDF)
	w.Header().Set(tuer.HeaderContentDisposition, `inline; filename="Rechnung.pdf"`)
	w.Header().Set(tuer.HeaderContentLength, fmt.Sprint(len(pdfBytes)))

	http.ServeContent(w, r, "Rechnung.pdf", time.Now(), bytes.NewReader(pdfBytes))
}
