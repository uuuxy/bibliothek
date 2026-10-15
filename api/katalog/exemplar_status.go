package katalog

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/service"
	"bibliothek/repository"
)

// exemplarStatusAnfrage ist der neue Status eines Exemplars.
type exemplarStatusAnfrage struct {
	IstAusleihbar   bool   `json:"ist_ausleihbar"`
	IstAusgesondert bool   `json:"ist_ausgesondert"`
	ZustandNotiz    string `json:"zustand_notiz"`
	// ZustandAbwertungProzent ist der Beschädigungsgrad des Exemplars, 0 bis 100. Ein Zeiger,
	// weil das Feld drei Zustände hat: Ein Wert setzt, 0 setzt zurück, und ein fehlendes Feld
	// lässt den Grad unangetastet. Ohne diese Unterscheidung löschte jeder andere Aufruf dieser
	// Tür einen erfassten Wasserschaden.
	//
	// Die Grenzen hält auch die Spalte (chk_zustand_abwertung_bereich); ohne die Prüfung hier
	// käme eine 140 als 500 zurück statt als Auskunft, was erlaubt ist.
	ZustandAbwertungProzent *int `json:"zustand_abwertung_prozent" validate:"omitempty,min=0,max=100"`
}

// ExemplarStatusAendern ändert Ausleihbarkeit, Aussonderung, Zustandsnotiz und
// Beschädigungsgrad eines Exemplars. Die Antwort nennt den neuen Ersatzwert und ob das Exemplar
// zum Bestand zählt.
//
// @Summary      Status eines Exemplars ändern
// @Description  Ändert ist_ausleihbar, ist_ausgesondert, die Zustandsnotiz und den Beschädigungsgrad eines Exemplars. Die Antwort nennt den neuen Ersatzwert und ob das Exemplar zum Bestand zählt. Ein verliehenes Exemplar lässt sich nicht aussondern (400).
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id    path      string                true  "Kennung des Exemplars (UUID)"
// @Param        body  body      exemplarStatusAnfrage   true  "Der neue Status"
// @Success      200   {object}  map[string]any
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /buecher/exemplare/{id}/status [put]
func (t *Tueren) ExemplarStatusAendern(katalogAbfragen repository.KatalogAbfragen, bescheidAbfragen repository.BescheidAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.aendereExemplarStatus(w, r, katalogAbfragen, bescheidAbfragen)
	}
}

// aendereExemplarStatus speichert Status und Zustand eines Exemplars und nennt den neuen
// Ersatzwert.
func (t *Tueren) aendereExemplarStatus(w http.ResponseWriter, r *http.Request, katalogAbfragen repository.KatalogAbfragen, bescheidAbfragen repository.BescheidAbfragen) {
	id := r.PathValue("id")
	if id == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung des Exemplars fehlt", nil)
		return
	}

	var req exemplarStatusAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return
	}

	// Wer speichert, steht beim Aussondern im Protokoll.
	claims, ok := auth.ClaimsAusKontext(r.Context())
	if !ok {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, auth.MeldungNichtAngemeldet, nil)
		return
	}

	ctx := r.Context()

	// Wer ein Exemplar von Hand auf „verfügbar" setzt, nimmt Notiz und Aussonderung zurück.
	//
	// Der Beschädigungsgrad bleibt dabei stehen: Er ist eine Eigenschaft des Buchs, kein
	// Status. Ein Band mit Wasserrand darf ausleihbar sein und trägt seinen Abschlag weiter;
	// sonst verlangte die Schule beim nächsten Verlust wieder den vollen Zeitwert für ein
	// sichtbar beschädigtes Buch.
	if req.IstAusleihbar {
		req.ZustandNotiz = ""
		req.IstAusgesondert = false
	}

	if err := katalogAbfragen.AendereExemplarStatus(ctx, id, req.IstAusleihbar, req.IstAusgesondert,
		req.ZustandNotiz, req.ZustandAbwertungProzent, claims.KontoID); err != nil {
		if errors.Is(err, repository.ErrExemplarNochVerliehen) {
			apierrors.AntworteFehler(w, http.StatusBadRequest, err)
			return
		}
		if errors.Is(err, repository.ErrExemplarNichtGefunden) {
			apierrors.AntworteFehler(w, http.StatusNotFound, err)
			return
		}
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	// Die Antwort trägt den neuen Ersatzwert: Wer den Wertverlust gerade eingetragen hat,
	// sieht sofort, was das Buch damit noch wert ist. Gerechnet wird am Server, mit derselben
	// Funktion wie im Dialog zum Melden eines Schadens; die Oberfläche rechnet diese Zahl nie
	// selbst.
	//
	// Scheitert die Nachfrage, bleibt es bei der Erfolgsmeldung: Gespeichert ist
	// gespeichert, und eine fehlende Auskunft darf daraus keinen Fehler machen.
	antwort := map[string]any{"status": "success"}
	if g, groessenErr := bescheidAbfragen.LadeGroessenFuerExemplar(ctx, id); groessenErr == nil {
		v := service.ErsatzwertVorschlagAus(g, service.LadeErsatzwertPreisquelle(ctx, t.DB.Pool))
		antwort["ersatzwert"] = v.Betrag
		antwort["ersatzwert_herleitung"] = v.Herleitung
		antwort["ersatzwert_bekannt"] = v.Bekannt
	}
	// Ob das Exemplar jetzt zum Bestand zählt: Freigeben und Aussondern holen ein bestelltes
	// Exemplar aus dem Zulauf, und die Karte der Buchakte benennt den Zustand danach.
	if imBestand, bestandErr := katalogAbfragen.IstExemplarImBestand(ctx, id); bestandErr == nil {
		antwort["im_bestand"] = imBestand
	}
	tuer.AntworteJSON(w, http.StatusOK, antwort)
}
