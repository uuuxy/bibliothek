package benutzer

import (
	"errors"
	"fmt"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
)

// Das Löschen eines Kontos steht für sich, weil es eigene Regeln hat, die mit dem
// Anlegen und Ändern nichts zu tun haben: kein Selbstlöschen, keine Admin-Konten ohne
// Adminrechte, und aktive Handapparat-Ausleihen sind ein Konflikt (409), kein Fehler.

// KontoLoeschen löscht ein Konto und vermerkt es im Protokoll.
// @Summary      Konto löschen
// @Description  Löscht ein Konto über seine Kennung und vermerkt es im Protokoll.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Kennung des Kontos (UUID)"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /benutzer/{id} [delete]
func (t *Tueren) KontoLoeschen(protokollAbfragen repository.ProtokollAbfragen, kontoAbfragen repository.KontoAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.loescheKonto(w, r, protokollAbfragen, kontoAbfragen)
	}
}

// loescheKonto prüft, ob das Konto gelöscht werden darf, und ordnet die Ablehnungen des
// Löschens ein.
func (t *Tueren) loescheKonto(w http.ResponseWriter, r *http.Request, protokollAbfragen repository.ProtokollAbfragen, kontoAbfragen repository.KontoAbfragen) {
	claims, ok := auth.ClaimsAusKontext(r.Context())
	if !ok {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, auth.MeldungNichtAngemeldet, nil)
		return
	}
	id := r.PathValue("id")
	if id == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung des Kontos fehlt", nil)
		return
	}

	// Das eigene Konto löscht niemand.
	if id == claims.KontoID {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusForbidden, "Das eigene Konto lässt sich nicht löschen", nil)
		return
	}

	ctx := r.Context()

	// Ohne diese Prüfung wäre die Rechtetrennung an der Bearbeitung wirkungslos:
	// Wer den letzten Administrator nicht ändern darf, aber löschen kann, entfernt
	// damit auch jede Instanz, die eine Selbstbeförderung noch zurücknehmen könnte.
	if !pruefeAdminZiel(ctx, w, r, kontoAbfragen, id) {
		return
	}

	err := protokollAbfragen.LoescheKonto(ctx, id, claims.KontoID)
	if err != nil {
		if errors.Is(err, repository.ErrKontoNichtGefunden) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, meldungKontoNichtGefunden, err)
			return
		}
		// Aktive Handapparat-Ausleihen sind ein Konflikt (409), kein Serverfehler:
		// Der Admin muss die Bücher erst zurückbuchen.
		var mitAusleihen *repository.KontoHatAusleihenFehler
		if errors.As(err, &mitAusleihen) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, fmt.Sprintf(
				"Das Konto hat noch ausgeliehene Exemplare (%d offen). Bitte zuerst zurückbuchen.", mitAusleihen.Offen), err)
			return
		}
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set(tuer.HeaderContentType, tuer.ContentTypeJSON)
	httpresp.Schreibe(w, []byte(`{"status":"success"}`))
}
