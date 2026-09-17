package api

import (
	"errors"
	"fmt"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/pkg/ersatzwert"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Der Betragsvorschlag für den Dialog „Verlust/Schaden melden" (OFFEN.md 9.3 a).
//
// Bis zum 17.09.2026 stand dort eine feste 15,00 € — eine Zahl ohne jeden Bezug zum Buch.
// Das Protokoll des Medienzentrums vom 16.09.2026 nennt genau das: „weder Einkaufs- bzw.
// Listenpreise noch Beschädigungsgrade hinterlegt, fehlende Restwertberechnung z.Zt.
// Handeingabe."
//
// ZWEI Regeln, nicht eine — das ist der Grund, warum dieser Handler mehr tut als
// bescheidVorschlagAus:
//
//   - Lernmittel (Landesmittel): die Staffel der Arbeitshilfe zum Erlass vom 17.12.2014.
//     1. Verleihjahr voller Kaufpreis, dann 80/60/40/20 %, ab dem 6. Jahr 10 %.
//   - Schülerbücherei (Mittel des Schulträgers): „zuerst Ersatzbeschaffung, sonst Geld in
//     Höhe des NEUWERTS" (Benutzungsordnung, mittel_konzept.md 1.2). Kein Abschlag für
//     das Alter — ein fünf Jahre alter Roman kostet in der Ersatzbeschaffung so viel wie
//     ein neuer.
//
// Die Staffel auf ein Büchereibuch anzuwenden wäre bequem und falsch: Sie steht in einer
// Arbeitshilfe für Schulbücher der Lernmittelfreiheit und gilt dort, weil das Land die
// Bücher bezahlt hat.
//
// In BEIDEN Fällen ist es ein Vorschlag, kein Automat: Der Betrag liegt im Ermessen der
// Schule, der Mensch kann ihn überschreiben, und wenn kein Preis hinterlegt ist, bleibt
// das Feld bei 0 mit einem Satz, der es sagt. Ein geratener Betrag wäre in einer
// Forderung gegen Eltern schlimmer als ein leeres Feld.

// ErsatzwertVorschlag ist die Antwort für den Melde-Dialog.
type ErsatzwertVorschlag struct {
	// Betrag ist der Vorschlag in Euro; 0 heißt „kein Preis hinterlegt".
	Betrag float64 `json:"betrag"`
	// Herleitung sagt dem Personal, WARUM dieser Betrag vorgeschlagen wird — damit im
	// Dialog nachvollziehbar steht, woher die Zahl kommt, bevor sie in einer Forderung
	// landet.
	Herleitung string `json:"herleitung"`
	// IstLernmittel unterscheidet die beiden Regeln; der Dialog benennt sie.
	IstLernmittel bool `json:"ist_lernmittel"`
}

// ErsatzwertVorschlagHandler liefert den Betragsvorschlag für ein Exemplar.
//
// @Summary      Suggest a replacement amount for a copy
// @Tags         schadensersatz
// @Produce      json
// @Param        id path string true "Copy ID"
// @Success      200 {object} ErsatzwertVorschlag
// @Failure      404 {object} apierrors.APIError
// @Router       /buecher/exemplare/{id}/ersatzwert-vorschlag [get]
func (s *Server) ErsatzwertVorschlagHandler(bescheidRepo repository.BescheidRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("id darf nicht leer sein", errors.New("missing id"))
		}

		groessen, err := bescheidRepo.GroessenFuerExemplar(r.Context(), id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apierrors.NotFound("Exemplar nicht gefunden", err)
			}
			return apierrors.Internal("Ersatzwert konnte nicht berechnet werden", err)
		}

		RespondJSON(w, http.StatusOK, ersatzwertVorschlagAus(groessen))
		return nil
	})
}

// ersatzwertVorschlagAus wählt die Regel und formuliert die Herleitung.
func ersatzwertVorschlagAus(g repository.ErsatzwertGroessen) ErsatzwertVorschlag {
	if !g.IstLernmittel {
		return buechereiVorschlag(g.Kaufpreis)
	}

	// Seit Migration 127 mit echtem Listenpreis und dem Zustand des Exemplars. Ist kein
	// Listenpreis erfasst (0), weicht die Staffel auf den Kaufpreis aus und sagt das.
	v := ersatzwert.Rechne(ersatzwert.Verleihjahr(g.SchuljahreMitAusleihe, g.SchuljahreImBestand),
		g.Kaufpreis, g.Listenpreis, g.ZustandAbschlag)
	return ErsatzwertVorschlag{
		Betrag:        v.Betrag,
		Herleitung:    bescheidHerleitung(v),
		IstLernmittel: true,
	}
}

// buechereiVorschlag setzt die Regel der Benutzungsordnung um: Neuwert ohne Abschlag.
func buechereiVorschlag(kaufpreis float64) ErsatzwertVorschlag {
	if kaufpreis <= 0 {
		return ErsatzwertVorschlag{Herleitung: "kein Preis hinterlegt — Betrag bitte eintragen"}
	}
	return ErsatzwertVorschlag{
		Betrag: kaufpreis,
		Herleitung: fmt.Sprintf("Bücherei-Bestand: Neuwert ohne Abschlag (%s, kein Neupreis hinterlegt)",
			euroBetrag(kaufpreis)),
		IstLernmittel: false,
	}
}
