package inventurlauf

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// startAnfrage nennt den Bereich einer neuen Inventur: den ganzen Bestand, ein Regal über den
// Anfang der Signatur oder Fach und Klasse.
type startAnfrage struct {
	Art string `json:"type"` // "global" | "signature" | "filter"
	// Signatur ist bei "signature" der Anfang von buecher_titel.signatur, also eine
	// Regaladresse: "BIB Deu" erfasst auch "BIB Deu 5 KRÜ".
	Signatur *string `json:"signatur,omitempty"`
	Fach     *string `json:"subject,omitempty"` // bei "filter": Fach (buecher_titel.subject)
	Jahrgang *int    `json:"grade,omitempty"`   // bei "filter": Klasse (Jahrgangsbereich enthält)
}

// signaturNormalisiert liefert die Signatur ohne Leerzeichen am Rand. Bezeichnung und Bereich
// nehmen sie beide von hier, damit die Bezeichnung nie einen anderen Bereich nennt, als der
// Lauf erfasst.
func (a startAnfrage) signaturNormalisiert() string {
	if a.Signatur == nil {
		return ""
	}
	return strings.TrimSpace(*a.Signatur)
}

// bereich leitet aus der Anfrage den Bereich ab, den die Abfragen auswerten.
func (a startAnfrage) bereich() repository.InventurBereich {
	switch a.Art {
	case "signature":
		signatur := a.signaturNormalisiert()
		return repository.InventurBereich{Signatur: &signatur}
	case "filter":
		return repository.InventurBereich{Fach: a.Fach, Jahrgang: a.Jahrgang}
	default:
		return repository.InventurBereich{}
	}
}

// startAntwort nennt den neuen Lauf und die Zahl der Exemplare, die er erwartet.
type startAntwort struct {
	LaufID      string `json:"session_id"`
	Bereich     string `json:"scope"`
	Bezeichnung string `json:"label"`
	Erwartet    int    `json:"erwartet"`
}

// pruefeStartAnfrage prüft die Art des Bereichs und die Felder, die sie verlangt.
func pruefeStartAnfrage(w http.ResponseWriter, req startAnfrage) bool {
	switch req.Art {
	case "global":
		return true
	case "signature":
		// Eine leere Signatur abweisen: Sie wäre der Anfang jeder Signatur und machte aus der
		// Inventur eines Regals unbemerkt eine über den ganzen Bestand.
		if req.signaturNormalisiert() == "" {
			apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New("signatur ist erforderlich, wenn type 'signature' ist"))
			return false
		}
		return true
	case "filter":
		if req.Fach == nil && req.Jahrgang == nil {
			apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New("filter scope requires at least 'subject' or 'grade'"))
			return false
		}
		return true
	default:
		apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New("invalid type, must be 'global', 'signature' or 'filter'"))
		return false
	}
}

// bezeichnungFuer bestimmt die Bezeichnung des Bereichs, die die Oberfläche zeigt. Für eine
// Signatur braucht sie keine Datenbank: Die Signatur ist ihre Bezeichnung.
func bezeichnungFuer(req startAnfrage) string {
	switch req.Art {
	case "signature":
		return "Signatur " + req.signaturNormalisiert()
	case "filter":
		return filterBezeichnung(req)
	default:
		return "Gesamtbestand"
	}
}

// filterBezeichnung baut eine lesbare Bezeichnung, z. B. "Mathematik · Kl. 5", "Deutsch" oder
// "Kl. 7".
func filterBezeichnung(req startAnfrage) string {
	var teile []string
	if req.Fach != nil && *req.Fach != "" {
		teile = append(teile, *req.Fach)
	}
	if req.Jahrgang != nil {
		teile = append(teile, fmt.Sprintf("Kl. %d", *req.Jahrgang))
	}
	if len(teile) == 0 {
		return "Teilbestand"
	}
	return strings.Join(teile, " · ")
}

// InventurStarten beginnt einen Lauf für den gewählten Bereich.
//
// Jeder Lauf zählt seinen eigenen Fortschritt. Ist der Bereich schon von einem offenen Lauf
// belegt, antwortet die Tür mit 409, statt etwas zurückzusetzen: Dort könnte gerade jemand
// scannen.
//
// @Summary      Inventur beginnen
// @Description  Beginnt einen Lauf für den gewählten Bereich: den ganzen Bestand, eine Signatur oder Fach und Klasse.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        body  body      startAnfrage   true  "Bereich"
// @Success      200   {object}  startAntwort
// @Failure      400   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /inventur/start [post]
func (t *Tueren) InventurStarten() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req startAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}
		if !pruefeStartAnfrage(w, req) {
			return
		}

		ctx := r.Context()
		var benutzerID string
		if claims, ok := auth.ClaimsAusKontext(ctx); ok {
			benutzerID = claims.KontoID
		}

		abfragen := repository.NewInventurAbfragen(t.DB.Pool)

		lauf, err := abfragen.LegeLaufAn(ctx, req.Art, req.bereich(), bezeichnungFuer(req), benutzerID)
		if err != nil {
			// Derselbe Bereich läuft schon, oder der gewünschte überschneidet sich mit einem
			// offenen Lauf: beides 409, kein Fehler des Servers.
			if errors.Is(err, repository.ErrInventurLaeuftBereits) || errors.Is(err, repository.ErrInventurUeberlappt) {
				apierrors.AntworteFehler(w, http.StatusConflict, err)
				return
			}
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		erwartet, err := abfragen.LadeLaufMitErwartung(ctx, lauf.ID)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		tuer.AntworteJSON(w, http.StatusOK, startAntwort{
			LaufID:      lauf.ID,
			Bereich:     lauf.BereichArt,
			Bezeichnung: lauf.Bezeichnung,
			Erwartet:    erwartet.Erwartet,
		})
	}
}
