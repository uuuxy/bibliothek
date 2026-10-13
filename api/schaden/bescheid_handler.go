package schaden

import (
	"context"
	"errors"
	"log"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/clientip"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Die Türen des Schadensersatz-Bescheids.
//
// Der Ablauf hat drei Schritte, und jeder ist ein eigener Aufruf: Der Dialog holt den Vorschlag
// (offene Forderungen mit Staffel-Betrag, bescheid_vorschlag.go), der Mensch prüft und erstellt
// (bescheid_erstellen.go), und danach lässt sich der Brief nachdrucken (bescheid_druck.go). Nach
// Fristablauf kommt die Übergabe dazu. Hier stehen die Listen, die Übergabe und was mehrere
// Türen brauchen; was der Brief einfriert, steht in bescheid_schnappschuss.go.
//
// Nichts davon läuft automatisch: Jede Referenznummer wird unwiderruflich verbraucht, und
// jeder Betrag ist eine Entscheidung der Schule.

// auditBescheidErstellt und auditBescheidUebergeben sind die Aktionen im Admin-Audit-Log.
const (
	auditBescheidErstellt   = "SCHADENSERSATZ_BESCHEID_ERSTELLT"
	auditBescheidUebergeben = "SCHADENSERSATZ_BESCHEID_UEBERGEBEN"
)

// bescheidAngaben liest Einstellungen und Schul-Identität in einem Zug.
func (s *Tueren) bescheidAngaben(ctx context.Context) (repository.BescheidAngaben, repository.SchuleAngaben, error) {
	einst, err := repository.NewEinstellungsAbfragen(s.DB.Pool).Lade(ctx)
	if err != nil {
		return repository.BescheidAngaben{}, repository.SchuleAngaben{}, err
	}
	return repository.BescheidAngabenAus(einst), repository.SchuleAngabenAus(einst), nil
}

// bescheidAkteur liefert die ID des angemeldeten Bearbeiters (leer ohne Anmeldung).
func bescheidAkteur(ctx context.Context) string {
	if claims, ok := auth.GetClaims(ctx); ok {
		return claims.UserID
	}
	return ""
}

// bescheidAudit protokolliert einen Eingriff. Dasselbe Muster wie der LMF-Plan: ohne
// Anmeldung oder ohne Datenbank gibt es nichts zu schreiben.
func bescheidAudit(ctx context.Context, s *Tueren, auditRepo repository.ProtokollAbfragen, aktion string, details map[string]any, ip string) {
	claims, ok := auth.GetClaims(ctx)
	if !ok || s.DB == nil || s.DB.Pool == nil || auditRepo == nil {
		return
	}
	if err := auditRepo.ProtokolliereAdminAktion(ctx, claims.UserID, aktion, ip, details); err != nil {
		log.Printf("Audit %s fehlgeschlagen: %v", aktion, err)
	}
}

// BescheidListeHandler liefert die Bescheide — die Arbeitsliste des Sekretariats.
//
// @Summary      List compensation notices
// @Tags         schadensersatz
// @Produce      json
// @Param        status query string false "offen = nur nicht übergebene"
// @Success      200 {array} repository.Bescheid
// @Router       /bescheide [get]
func (s *Tueren) BescheidListeHandler(bescheidRepo repository.BescheidAbfragen) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		liste, err := bescheidRepo.LadeListe(r.Context(), r.URL.Query().Get("status") == "offen")
		if err != nil {
			return apierrors.Internal("Bescheide konnten nicht gelesen werden", err)
		}
		tuer.RespondJSON(w, http.StatusOK, liste)
		return nil
	})
}

// BescheidSchuelerListeHandler liefert die Bescheide eines Schülers für die Akte.
//
// @Summary      List a student's compensation notices
// @Tags         schadensersatz
// @Produce      json
// @Param        id path string true "Student ID"
// @Success      200 {array} repository.Bescheid
// @Router       /schueler/{id}/bescheide [get]
func (s *Tueren) BescheidSchuelerListeHandler(bescheidRepo repository.BescheidAbfragen) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest(tuer.MeldungIDLeer, errors.New("missing id"))
		}
		liste, err := bescheidRepo.LadeZuSchueler(r.Context(), id)
		if err != nil {
			return apierrors.Internal("Bescheide konnten nicht gelesen werden", err)
		}
		tuer.RespondJSON(w, http.StatusOK, liste)
		return nil
	})
}

// BescheidUebergebenHandler setzt einen Bescheid nach Fristablauf auf „übergeben".
//
// Kein Cron: Die Übergabe ist eine Handlung des Sekretariats (Original und Buchungsbeleg
// gehen aus dem Haus). Eine Automatik über Nacht wäre eine stille Zustandsänderung.
//
// @Summary      Mark a notice as handed over after the deadline
// @Tags         schadensersatz
// @Produce      json
// @Param        id path string true "Notice ID"
// @Success      200 {object} map[string]any
// @Router       /bescheide/{id}/uebergeben [post]
func (s *Tueren) BescheidUebergebenHandler(bescheidRepo repository.BescheidAbfragen, auditRepo repository.ProtokollAbfragen) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest(tuer.MeldungIDLeer, errors.New("missing id"))
		}
		ctx := r.Context()
		if err := bescheidRepo.Uebergebe(ctx, id); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Entweder gibt es den Bescheid nicht, oder sein Zustand passt nicht.
				// Die Meldung nennt beides, statt einen 404 zu behaupten.
				return apierrors.Conflict(
					"Übergabe nicht möglich: Der Bescheid ist nicht (mehr) offen oder seine Frist läuft noch.",
					err)
			}
			return apierrors.Internal("Übergabe konnte nicht gebucht werden", err)
		}

		bescheidAudit(ctx, s, auditRepo, auditBescheidUebergeben, map[string]any{"bescheid_id": id}, clientip.AusAnfrage(r))
		tuer.RespondJSON(w, http.StatusOK, map[string]any{"id": id, "status": "uebergeben"})
		return nil
	})
}
