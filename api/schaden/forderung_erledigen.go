package schaden

import (
	"context"
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

// zahlungsBuchung verbucht die Zahlung einer offenen Forderung (repository.SchadensAbfragen);
// repository.ErrSchadensfallNichtGefunden oder repository.ErrSchadensfallErledigt, wenn es keine
// offene Forderung dieser Kennung gibt.
type zahlungsBuchung interface {
	VerbucheZahlung(ctx context.Context, schadensfallID string, bearbeiterID string) error
}

// forderungsStorno storniert eine offene Forderung mit Begründung (repository.SchadensAbfragen);
// den Betrag liest es selbst. Gibt es keine offene Forderung dieser Kennung, liefert es dieselben
// Fehler wie zahlungsBuchung.
type forderungsStorno interface {
	StorniereForderung(ctx context.Context, schadensfallID string, bearbeiterID string, grund string) error
}

// SchadensfaelleDesSchuelers liefert alle Schadensfälle eines Schülers für
// die Gebühren-Sektion der Schülerakte (GET /api/schueler/{id}/schadensfaelle).
func (t *Tueren) SchadensfaelleDesSchuelers(schadensAbfragen *repository.SchadensAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("Schüler-ID fehlt", nil)
		}
		faelle, err := schadensAbfragen.LadeZuLeser(r.Context(), id)
		if err != nil {
			return apierrors.Internal("Fehler beim Laden der Schadensfälle", err)
		}
		tuer.AntworteJSON(w, http.StatusOK, faelle)
		return nil
	})
}

// erledigungsFehler ordnet die Fehler der Abfragen einem Status zu. 409 statt 500 trägt hier:
// Am Mehrplatz-System ist „eine Kollegin war schneller" der Normalfall, und der Sanitizer
// machte aus einem 500 „interner Datenbankfehler" und versteckte die Ursache.
func erledigungsFehler(err error, aktion string) error {
	if errors.Is(err, repository.ErrSchadensfallNichtGefunden) {
		return apierrors.NotFound("Schadensfall nicht gefunden", err)
	}
	if errors.Is(err, repository.ErrSchadensfallErledigt) {
		return apierrors.Conflict("Schadensfall wurde bereits bezahlt oder storniert", err)
	}
	return apierrors.Internal("Fehler beim "+aktion+" der Gebühr", err)
}

// ZahlungVerbuchen verbucht die Zahlung (POST /api/schadensfaelle/{id}/bezahlt).
// Ohne Rumpf: Der Betrag kommt immer aus der Datenbank, nie von der Oberfläche.
func (t *Tueren) ZahlungVerbuchen(schadensAbfragen zahlungsBuchung) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := auth.ClaimsAusKontext(r.Context())
		if !ok {
			return apierrors.Unauthorized(auth.MeldungNichtAngemeldet, nil)
		}
		if err := schadensAbfragen.VerbucheZahlung(r.Context(), r.PathValue("id"), claims.KontoID); err != nil {
			return erledigungsFehler(err, "Verbuchen")
		}
		tuer.AntworteErfolg(w)
		return nil
	})
}

// ForderungStornieren storniert eine Forderung mit Pflicht-Begründung
// (POST /api/schadensfaelle/{id}/storno).
func (t *Tueren) ForderungStornieren(schadensAbfragen forderungsStorno) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := auth.ClaimsAusKontext(r.Context())
		if !ok {
			return apierrors.Unauthorized(auth.MeldungNichtAngemeldet, nil)
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
		if err := schadensAbfragen.StorniereForderung(r.Context(), r.PathValue("id"), claims.KontoID, grund); err != nil {
			return erledigungsFehler(err, "Stornieren")
		}
		tuer.AntworteErfolg(w)
		return nil
	})
}
