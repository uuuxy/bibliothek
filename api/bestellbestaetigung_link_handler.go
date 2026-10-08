package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// NeuerBestaetigungsLinkHandler erzeugt den Bestätigungs-Link einer Bestellung neu
// (PUT /api/bestellungen/{id}/bestaetigungs-link).
//
// Zwei Anlässe: Die Mail ging an die falsche Adresse und der alte Link soll sterben —
// oder es gab nie einen, weil beim Bestellen noch keine öffentliche Adresse hinterlegt
// war. Der alte Token wird dabei überschrieben und ist sofort tot; das ist der Zweck.
//
// Ein „Link nochmal anzeigen" kann es nicht geben: In der Datenbank liegt nur der Hash.
// Das ist keine Lücke, sondern der Preis dafür, dass ein Datenbank-Auszug keine
// benutzbaren Links enthält.
func (s *Server) NeuerBestaetigungsLinkHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing bestellung id"))
			return
		}

		ctx := r.Context()

		bietet, err := s.bestellungImBestaetigungsweg(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("bestellung not found"))
			return
		}
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if !bietet {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("diese Bestellung hat keinen Bestätigungsschritt"))
			return
		}

		basis := s.oeffentlicheAdresse(ctx)
		if basis == "" {
			// 400 statt 500: Der Sanitizer ersetzt jede 500-Meldung durch „interner
			// Datenbankfehler" — die Anleitung, was zu tun ist, erreichte den Bildschirm
			// dann nie. Das hier ist kein Fehler, sondern eine fehlende Einstellung.
			apierrors.SendHTTPError(w, http.StatusBadRequest,
				errors.New("in den Einstellungen ist keine öffentliche Adresse hinterlegt — ohne sie kann kein Link erzeugt werden"))
			return
		}

		token, gueltigBis, err := s.erneuereBestaetigungsToken(ctx, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, map[string]any{
			"link":        bestaetigungsLink(basis, token),
			"gueltig_bis": gueltigBis,
		})
	}
}

// erneuereBestaetigungsToken gibt der Bestellung einen neuen Bestätigungs-Link mit neuer
// Frist und liefert seinen Klartext. Ein früherer Link ist damit ungültig: Gespeichert ist
// immer nur der Hash des einen gültigen. „Neuen Link erzeugen" und der erneute Versand der
// Bestellmail gehen beide hier durch.
func (s *Server) erneuereBestaetigungsToken(ctx context.Context, bestellungID string) (token string, gueltigBis time.Time, err error) {
	token, hash, err := neuerBestaetigungsToken()
	if err != nil {
		return "", time.Time{}, err
	}
	gueltigBis, err = repository.ErneuereBestaetigungsToken(ctx, s.DB.Pool, bestellungID, hash, s.bestellinkGueltigkeitTage(ctx))
	if err != nil {
		return "", time.Time{}, err
	}
	return token, gueltigBis, nil
}

// oeffentlicheAdresse liest die Adresse, unter der Dritte das System erreichen. Leer =
// nicht hinterlegt; dann verschickt das System keine Links, statt kaputte zu erzeugen.
func (s *Server) oeffentlicheAdresse(ctx context.Context) string {
	settings, err := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(ctx)
	if err != nil || settings.OeffentlicheAdresse == nil {
		return ""
	}
	return *settings.OeffentlicheAdresse
}
