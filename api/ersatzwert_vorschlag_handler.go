package api

import (
	"context"
	"errors"
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

		RespondJSON(w, http.StatusOK, ersatzwertVorschlagAus(groessen, s.preisquelle(r.Context())))
		return nil
	})
}

// ersatzwertEingabe sind die Größen, aus denen JEDER Betragsvorschlag entsteht — egal ob
// er aus einem Exemplar (Melde-Dialog), einer offenen Forderung oder einer überfälligen
// Ausleihe kommt (Bescheid-Vorschlag).
//
// Bis zum 17.09.2026 waren es drei Wege mit ZWEI Rechnungen: Der Melde-Dialog wählte die
// Regel nach IstLernmittel, die beiden Bescheid-Wege wendeten die Staffel der
// Arbeitshilfe auf ALLES an — auch auf Büchereibücher, für die sie nicht gilt. Für
// dasselbe Buch nannten Dialog und Brief damit verschiedene Beträge, und der Brief war
// der falsche. Ein Typ, eine Rechnung, eine Herleitung.
type ersatzwertEingabe struct {
	Kaufpreis       float64
	Listenpreis     float64
	ZustandAbschlag int
	// Die beiden Größen des Verleihjahrs — nur die Staffel benutzt sie.
	SchuljahreMitAusleihe int
	SchuljahreImBestand   int
	// IstLernmittel entscheidet über die REGEL, nicht über einen Faktor: Staffel des
	// Landes oder Neuwert der Benutzungsordnung.
	IstLernmittel bool
}

func (e ersatzwertEingabe) rechne(quelle ersatzwert.Preisquelle) ersatzwert.Vorschlag {
	if !e.IstLernmittel {
		return ersatzwert.RechneNeuwert(e.Kaufpreis, e.Listenpreis, e.ZustandAbschlag, quelle)
	}
	return ersatzwert.Rechne(ersatzwert.Verleihjahr(e.SchuljahreMitAusleihe, e.SchuljahreImBestand),
		e.Kaufpreis, e.Listenpreis, e.ZustandAbschlag, quelle)
}

// eingabeAusGroessen: der Weg des Melde-Dialogs.
func eingabeAusGroessen(g repository.ErsatzwertGroessen) ersatzwertEingabe {
	return ersatzwertEingabe{
		Kaufpreis: g.Kaufpreis, Listenpreis: g.Listenpreis, ZustandAbschlag: g.ZustandAbschlag,
		SchuljahreMitAusleihe: g.SchuljahreMitAusleihe, SchuljahreImBestand: g.SchuljahreImBestand,
		IstLernmittel: g.IstLernmittel,
	}
}

// ersatzwertVorschlagAus rechnet und formuliert die Herleitung.
func ersatzwertVorschlagAus(g repository.ErsatzwertGroessen, quelle ersatzwert.Preisquelle) ErsatzwertVorschlag {
	v := eingabeAusGroessen(g).rechne(quelle)
	return ErsatzwertVorschlag{
		Betrag:        v.Betrag,
		Herleitung:    bescheidHerleitung(v),
		IstLernmittel: g.IstLernmittel,
	}
}

// preisquelle liest aus den Einstellungen, welcher Preis ab dem zweiten Verleihjahr die
// Grundlage ist (Anforderungsliste Nr. 3, OFFEN.md 9.8 Stufe 4).
//
// Scheitert das Lesen, gilt die Regel der Arbeitshilfe. Das ist die einzige Richtung, die
// hier zu vertreten ist: Ein Fehler beim Lesen einer Einstellung darf nicht dazu führen,
// dass eine Forderung nach einer anderen Grundlage entsteht als die davor.
func (s *Server) preisquelle(ctx context.Context) ersatzwert.Preisquelle {
	einst, err := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(ctx)
	if err != nil || einst == nil || !einst.ErsatzwertImmerKaufpreis {
		return ersatzwert.PreisquelleListenpreis
	}
	return ersatzwert.PreisquelleKaufpreis
}
