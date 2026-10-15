package leser

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/kennung"
	"bibliothek/repository"
)

// Zusammenführen zweier Schülerdatensätze, das Sicherheitsnetz hinter der
// Umbenennungs-Paarung des LUSD-Imports (internal/lusd/paarung.go). Regeln und SQL stehen in
// repository/leser_zusammenfuehren.go; hier nur Recht, Rumpf, Fehlerbild und Protokoll.
//
// Ein eigenes Recht, merge_students: Es ist ein Eingriff in die Identität zweier Datensätze,
// unumkehrbar, und festgelegt ist, dass ein Admin ihn vornimmt, nicht die Theke.

type leserZusammenfuehrenAnfrage struct {
	QuelleID string `json:"quelle_id"`
}

// LeserZusammenfuehren bedient POST /api/schueler/{id}/zusammenfuehren — {id}
// bleibt, quelle_id geht darin auf.
func (t *Tueren) LeserZusammenfuehren(protokollAbfragen repository.ProtokollAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		zielID := r.PathValue("id")
		var rumpf leserZusammenfuehrenAnfrage
		if err := json.NewDecoder(r.Body).Decode(&rumpf); err != nil || strings.TrimSpace(rumpf.QuelleID) == "" {
			return apierrors.BadRequest("Die Kennung des zweiten Datensatzes fehlt", err)
		}
		if !kennung.IstUUID(rumpf.QuelleID) {
			return apierrors.BadRequest("Die Kennung des zweiten Datensatzes ist ungültig", nil)
		}

		var bearbeiterID string
		if claims, ok := auth.ClaimsAusKontext(r.Context()); ok {
			bearbeiterID = claims.KontoID
		}
		erg, err := repository.FuehreLeserZusammen(r.Context(), t.DB.Pool, repository.ZusammenfuehrenAuftrag{
			ZielID: zielID, QuelleID: rumpf.QuelleID, BearbeiterID: bearbeiterID,
		})
		switch {
		case errors.Is(err, repository.ErrZusammenfuehrenGleich):
			return apierrors.BadRequest("Ein Datensatz lässt sich nicht mit sich selbst zusammenführen", err)
		case errors.Is(err, repository.ErrZusammenfuehrenNichtGefunden):
			return apierrors.NotFound("Einer der beiden Schüler wurde nicht gefunden oder liegt im Papierkorb", err)
		case errors.Is(err, repository.ErrZusammenfuehrenAnonymisiert):
			return apierrors.Conflict("Ein anonymisierter Datensatz trägt keine Person mehr und lässt sich nicht zusammenführen", err)
		case errors.Is(err, repository.ErrZusammenfuehrenVerschiedeneArten):
			return apierrors.Conflict("Ein Schüler lässt sich nicht mit einem Kollegen zusammenführen", err)
		case errors.Is(err, repository.ErrZusammenfuehrenZweiKonten):
			return apierrors.Conflict("Beide Datensätze haben ein eigenes Zugangskonto, und beim Zusammenführen ginge eine der beiden Anmeldungen verloren. Erst unter „Benutzer & Rechte“ entscheiden, welches Konto bleibt.", err)
		case err != nil:
			return apierrors.Internal("Zusammenführen fehlgeschlagen", err)
		}

		// Protokoll der Verwaltung (audit_logs): welcher Datensatz in welchem aufging, samt
		// Barcodes. Der Rückweg (Stammdaten der Quelle, gewanderte Zeilen) steht in audit_log am
		// Ziel, geschrieben in der Transaktion (repository.schreibeRueckwegEintrag).
		// Schlüssel `schueler_id` für den verbliebenen (Vokabular der Auskunft/Tilgung).
		if claims, ok := auth.ClaimsAusKontext(r.Context()); ok {
			if err := protokollAbfragen.ProtokolliereAdminAktion(r.Context(), claims.KontoID, "SCHUELER_ZUSAMMENGEFUEHRT", clientip.AusAnfrage(r), map[string]any{
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
		tuer.AntworteJSON(w, http.StatusOK, erg)
		return nil
	})
}

// Zusammenfuehrungskandidaten bedient GET /api/schueler/{id}/zusammenfuehren-kandidaten?q=
// — Suche über alle nicht gelöschten Schüler (auch Abgänger und Gesperrte), ohne {id}
// selbst. Die Aktivliste der Schülerdatei blendet Abgänger aus; hier sind sie das Ziel.
func (t *Tueren) Zusammenfuehrungskandidaten() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		treffer, err := repository.FindeZusammenfuehrenKandidaten(r.Context(), t.DB.Pool, r.PathValue("id"), r.URL.Query().Get("q"), 20)
		if err != nil {
			return apierrors.Internal("Kandidatensuche fehlgeschlagen", err)
		}
		tuer.AntworteJSON(w, http.StatusOK, treffer)
		return nil
	})
}
