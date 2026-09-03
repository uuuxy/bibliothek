package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"

	"github.com/google/uuid"
)

// Zusammenführen zweier Schülerdatensätze — das Sicherheitsnetz hinter der Umbenennungs-
// Paarung des LUSD-Imports (lusd_paarung.go). Regeln und SQL stehen in
// repository/schueler_zusammenfuehren.go; hier nur Recht, Rumpf, Fehlerbild und Audit.
//
// Recht merge_students (eigenes Recht seit 03.09.2026, vorher manage_students_admin wie
// Purge und DSGVO-Auskunft): Es ist ein Eingriff in die
// Identität zweier Datensätze, unumkehrbar, und Peters Vorgabe vom 02.09.2026 ist, dass
// ein Admin das tut — nicht der Tresen.

type zusammenfuehrenRumpf struct {
	QuelleID string `json:"quelle_id"`
}

// ZusammenfuehrenSchuelerHandler bedient POST /api/schueler/{id}/zusammenfuehren — {id}
// bleibt, quelle_id geht darin auf.
func (s *Server) ZusammenfuehrenSchuelerHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		zielID := r.PathValue("id")
		var rumpf zusammenfuehrenRumpf
		if err := json.NewDecoder(r.Body).Decode(&rumpf); err != nil || strings.TrimSpace(rumpf.QuelleID) == "" {
			return apierrors.BadRequest("quelle_id fehlt", err)
		}
		if _, err := uuid.Parse(rumpf.QuelleID); err != nil {
			return apierrors.BadRequest("quelle_id ist keine gültige Kennung", nil)
		}

		var bearbeiterID string
		if claims, ok := auth.GetClaims(r.Context()); ok {
			bearbeiterID = claims.UserID
		}
		erg, err := repository.ZusammenfuehrenSchueler(r.Context(), s.DB.Pool, repository.ZusammenfuehrenAuftrag{
			ZielID: zielID, QuelleID: rumpf.QuelleID, AbgaengerJahr: calculateAbgaengerJahr, BearbeiterID: bearbeiterID,
		})
		switch {
		case errors.Is(err, repository.ErrZusammenfuehrenGleich):
			return apierrors.BadRequest("Ein Datensatz lässt sich nicht mit sich selbst zusammenführen.", err)
		case errors.Is(err, repository.ErrZusammenfuehrenNichtGefunden):
			return apierrors.NotFound("Einer der beiden Schüler wurde nicht gefunden oder liegt im Papierkorb.", err)
		case errors.Is(err, repository.ErrZusammenfuehrenAnonymisiert):
			return apierrors.Conflict("Ein anonymisierter Datensatz trägt keine Person mehr und lässt sich nicht zusammenführen.", err)
		case err != nil:
			return apierrors.Internal("Zusammenführen fehlgeschlagen", err)
		}

		// Admin-Protokoll (audit_logs): welcher Datensatz in welchem aufging, samt Barcodes.
		// Der RÜCKWEG (Stammdaten der Quelle, gewanderte Zeilen) steht in audit_log am
		// Ziel, geschrieben in der Transaktion (repository.schreibeRueckwegEintrag).
		// Schlüssel `schueler_id` für den verbliebenen (Vokabular der Auskunft/Tilgung).
		if claims, ok := auth.GetClaims(r.Context()); ok {
			if err := auditRepo.LogAdminAktion(r.Context(), claims.UserID, "SCHUELER_ZUSAMMENGEFUEHRT", getIP(r), map[string]any{
				"schueler_id":        erg.ZielID,
				"barcode":            erg.BarcodeID,
				"aufgeloest_id":      rumpf.QuelleID,
				"aufgeloest_barcode": erg.QuelleBarcode,
				"ausleihen":          erg.Ausleihen,
				"schaeden":           erg.Schaeden,
				"vormerkungen":       erg.Vormerkungen,
			}); err != nil {
				log.Printf("Zusammenführen: Audit-Eintrag fehlgeschlagen: %v", err)
			}
		}
		RespondJSON(w, http.StatusOK, erg)
		return nil
	})
}

// ZusammenfuehrenKandidatenHandler bedient GET /api/schueler/{id}/zusammenfuehren-kandidaten?q=
// — Suche über ALLE nicht gelöschten Schüler (auch Abgänger und Gesperrte), ohne {id}
// selbst. Die Aktivliste der Schülerdatei blendet Abgänger aus; hier sind sie das Ziel.
func (s *Server) ZusammenfuehrenKandidatenHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		treffer, err := repository.SucheZusammenfuehrenKandidaten(r.Context(), s.DB.Pool, r.PathValue("id"), r.URL.Query().Get("q"), 20)
		if err != nil {
			return apierrors.Internal("Kandidatensuche fehlgeschlagen", err)
		}
		RespondJSON(w, http.StatusOK, treffer)
		return nil
	})
}
