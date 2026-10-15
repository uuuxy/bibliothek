package schaden

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// Forderungen erledigen: bar bezahlt oder storniert. Eine offene Forderung sperrt das Löschen
// eines Schülers und den LUSD-Abgleich (pruefeLeserLoeschbar, der Abgleich prüft
// ist_bezahlt = false); deshalb braucht jede einen Weg zu ihrem Ende.
//
// Recht: edit_students wie beim Anlegen (POST /api/damage/report), nicht bloß eine Sitzung:
// Arbeitsplätze am Kiosk sind dauerhaft mit der Rolle Helfer angemeldet, und Forderungen
// erledigen dürfen nur Rollen, die Schülerdaten bearbeiten.

// SchadensfaelleDesSchuelers liefert alle Schadensfälle eines Schülers für
// die Gebühren-Sektion der Schülerakte (GET /api/schueler/{id}/schadensfaelle).
func (t *Tueren) SchadensfaelleDesSchuelers(schadensAbfragen repository.SchadensAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("Schüler-ID fehlt", nil)
		}
		faelle, err := schadensAbfragen.LadeZuLeser(r.Context(), id)
		if err != nil {
			return apierrors.Internal("Fehler beim Laden der Schadensfälle", err)
		}
		tuer.AntworteJSON(w, http.StatusOK, map[string]any{"data": faelle})
		return nil
	})
}

// erledigungsFehler ordnet die Fehler der Abfragen einem Status zu. 409 statt 500 trägt hier:
// Am Mehrplatz-System ist „eine Kollegin war schneller" der Normalfall, und der Sanitizer
// machte aus einem 500 „interner Datenbankfehler" und versteckte die Ursache.
func erledigungsFehler(err error, aktion string) error {
	if errors.Is(err, repository.ErrSchadensfallNichtGefunden) {
		return apierrors.NotFound(err.Error(), err)
	}
	if errors.Is(err, repository.ErrSchadensfallErledigt) {
		return apierrors.Conflict(err.Error(), err)
	}
	return apierrors.Internal("Fehler beim "+aktion+" der Gebühr", err)
}

// ZahlungVerbuchen verbucht die Zahlung (POST /api/schadensfaelle/{id}/bezahlt).
// Ohne Rumpf: Der Betrag kommt immer aus der Datenbank, nie von der Oberfläche.
func (t *Tueren) ZahlungVerbuchen(protokollAbfragen repository.ProtokollAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := auth.ClaimsAusKontext(r.Context())
		if !ok {
			return apierrors.Unauthorized("missing session information", nil)
		}
		if err := protokollAbfragen.VerbucheZahlung(r.Context(), r.PathValue("id"), claims.KontoID); err != nil {
			return erledigungsFehler(err, "Verbuchen")
		}
		tuer.AntworteErfolg(w)
		return nil
	})
}

// ForderungStornieren storniert eine Forderung mit Pflicht-Begründung
// (POST /api/schadensfaelle/{id}/storno).
func (t *Tueren) ForderungStornieren(protokollAbfragen repository.ProtokollAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := auth.ClaimsAusKontext(r.Context())
		if !ok {
			return apierrors.Unauthorized("missing session information", nil)
		}
		var req struct {
			Grund string `json:"grund"`
		}
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}
		grund := strings.TrimSpace(req.Grund)
		if grund == "" {
			return apierrors.BadRequest("Ein Stornierungsgrund ist erforderlich", nil)
		}
		if err := protokollAbfragen.StorniereForderung(r.Context(), r.PathValue("id"), claims.KontoID, grund); err != nil {
			return erledigungsFehler(err, "Stornieren")
		}
		tuer.AntworteErfolg(w)
		return nil
	})
}
