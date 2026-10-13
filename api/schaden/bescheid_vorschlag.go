package schaden

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/internal/service"
	"bibliothek/pkg/ersatzwert"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Der Vorschlag für einen neuen Bescheid: was auf den Brief kann, je mit dem Betrag der Staffel.

// BescheidVorschlagPosition ist eine offene Forderung mit dem vorgeschlagenen Betrag und
// seiner Herleitung — der Dialog zeigt beides, damit der Mensch die Zahl beurteilen kann.
type BescheidVorschlagPosition struct {
	SchadensfallID string  `json:"schadensfall_id"`
	Art            string  `json:"art"`
	Titel          string  `json:"titel"`
	ISBN           string  `json:"isbn"`
	Betrag         float64 `json:"betrag"`
	// Herleitung: „3. Verleihjahr → 60 % von 41,50 €" bzw. der Hinweis, dass kein Preis
	// hinterlegt ist.
	Herleitung string `json:"herleitung"`
	// Topf ist das Eigentum des Exemplars (repository.ExemplarTopfSQL): Bücher des Landes
	// gehen an das Land, alles andere an den Schulträger; leer bei einem Geräteschaden. Ein
	// Brief trägt genau einen Topf. Bis zum 29.09.2026 stand hier ist_lernmittel des Titels.
	Topf string `json:"topf"`
}

// BescheidVorschlagAusleihe ist ein überfälliges Buch ohne Forderung: Es kommt mit dem
// Brief als Verlust in die Bücher (Stufe 2). Betrag und Herleitung wie bei einer Position.
type BescheidVorschlagAusleihe struct {
	AusleiheID  string  `json:"ausleihe_id"`
	Titel       string  `json:"titel"`
	ISBN        string  `json:"isbn"`
	Betrag      float64 `json:"betrag"`
	Herleitung  string  `json:"herleitung"`
	Topf        string  `json:"topf"`
	FaelligSeit string  `json:"faellig_seit"`
	// Mahnstufe und LetztesMahndatum (JJJJ-MM-TT, leer ohne Datum): wie oft und wann
	// zuletzt ein Mahnbrief zu diesem Buch gedruckt wurde.
	Mahnstufe        int    `json:"mahnstufe"`
	LetztesMahndatum string `json:"letztes_mahndatum,omitempty"`
}

// BescheidVorschlag ist die Antwort für den Dialog.
type BescheidVorschlag struct {
	SchuelerName string                      `json:"schueler_name"`
	Klasse       string                      `json:"klasse"`
	FristBis     string                      `json:"frist_bis"`
	Positionen   []BescheidVorschlagPosition `json:"positionen"`
	// Ausleihen: überfällige Bücher, die noch keine Forderung tragen. Seit dem
	// 15.09.2026 (Stufe 2) braucht es keine Verlustmeldung je Buch mehr vor dem Brief.
	Ausleihen []BescheidVorschlagAusleihe `json:"ausleihen"`
	// FehlendeAngaben nennt die Einstellungen, ohne die kein Bescheid entstehen kann.
	// Der Dialog zeigt sie, statt den Knopf stumm zu sperren.
	FehlendeAngaben []string `json:"fehlende_angaben"`
}

// BescheidVorschlagHandler liefert die offenen Forderungen eines Schülers mit
// Betragsvorschlag.
//
// @Summary      Proposal for a new compensation notice
// @Tags         schadensersatz
// @Produce      json
// @Param        id path string true "Student ID"
// @Success      200 {object} BescheidVorschlag
// @Router       /schueler/{id}/bescheid-vorschlag [get]
func (s *Tueren) BescheidVorschlagHandler(bescheidRepo repository.BescheidAbfragen) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		return s.handleBescheidVorschlag(w, r, bescheidRepo)
	})
}

// handleBescheidVorschlag sammelt, was auf einen Bescheid kann: offene Forderungen und
// überfällige Bücher, je mit Betragsvorschlag.
func (s *Tueren) handleBescheidVorschlag(w http.ResponseWriter, r *http.Request, bescheidRepo repository.BescheidAbfragen) error {
	id := r.PathValue("id")
	if id == "" {
		return apierrors.BadRequest(tuer.MeldungIDLeer, errors.New("missing id"))
	}
	ctx := r.Context()

	angaben, schule, err := s.bescheidAngaben(ctx)
	if err != nil {
		return apierrors.Internal(tuer.MeldungEinstellungenUnlesbar, err)
	}
	// Welcher Preis die Grundlage ist, steht in denselben Einstellungen (Stufe 4).
	quelle := service.ErsatzwertPreisquelle(ctx, s.DB.Pool)

	vorschlag := BescheidVorschlag{
		FristBis:        schulzeit.Jetzt().AddDate(0, 0, angaben.FristTage).Format(tuer.DateFormatISO),
		FehlendeAngaben: angaben.FehlendeAngaben(schule),
		Positionen:      []BescheidVorschlagPosition{},
		Ausleihen:       []BescheidVorschlagAusleihe{},
	}
	empfaenger, err := bescheidRepo.LadeEmpfaenger(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apierrors.NotFound("Schüler nicht gefunden", err)
		}
		return apierrors.Internal("Schüler konnte nicht gelesen werden", err)
	}
	vorschlag.SchuelerName = strings.TrimSpace(empfaenger.Vorname + " " + empfaenger.Nachname)
	vorschlag.Klasse = empfaenger.Klasse

	offene, err := bescheidRepo.LadeOffeneForderungen(ctx, id)
	if err != nil {
		return apierrors.Internal("Offene Forderungen konnten nicht gelesen werden", err)
	}
	for _, f := range offene {
		vorschlag.Positionen = append(vorschlag.Positionen, bescheidVorschlagAus(f, quelle))
	}
	ueberfaellig, err := bescheidRepo.LadeUeberfaelligeAusleihen(ctx, id)
	if err != nil {
		return apierrors.Internal("Überfällige Ausleihen konnten nicht gelesen werden", err)
	}
	for _, a := range ueberfaellig {
		vorschlag.Ausleihen = append(vorschlag.Ausleihen, bescheidVorschlagAusAusleihe(a, quelle))
	}

	tuer.RespondJSON(w, http.StatusOK, vorschlag)
	return nil
}

// bescheidVorschlagAus rechnet den Staffel-Vorschlag für eine offene Forderung.
//
// Seit Migration 127 mit echtem Listenpreis und dem Zustand des Exemplars: Die
// Arbeitshilfe verlangt ab dem zweiten Verleihjahr den Neupreis zum Zeitpunkt des
// Verlusts. Bis zum 17.09.2026 stand hier eine harte 0, und die Staffel wich immer auf
// den Kaufpreis aus. Ist kein Listenpreis erfasst, tut sie das weiterhin — und sagt es.
func bescheidVorschlagAus(f repository.OffeneForderung, quelle ersatzwert.Preisquelle) BescheidVorschlagPosition {
	v := service.ErsatzwertEingabe{
		Kaufpreis: f.Kaufpreis, Listenpreis: f.Listenpreis, ZustandAbschlag: f.ZustandAbschlag,
		SchuljahreMitAusleihe: f.SchuljahreMitAusleihe, SchuljahreImBestand: f.SchuljahreImBestand,
		Topf: f.Topf,
	}.Rechne(quelle)
	return BescheidVorschlagPosition{
		SchadensfallID: f.SchadensfallID,
		Art:            f.Art,
		Titel:          f.Titel,
		ISBN:           f.ISBN,
		Betrag:         v.Betrag,
		Herleitung:     service.ErsatzwertHerleitung(v),
		Topf:           f.Topf,
	}
}

// bescheidVorschlagAusAusleihe rechnet den Staffel-Vorschlag für ein überfälliges Buch —
// dieselbe Rechnung wie für eine Forderung, das Buch trägt nur noch keine.
func bescheidVorschlagAusAusleihe(a repository.UeberfaelligeAusleihe, quelle ersatzwert.Preisquelle) BescheidVorschlagAusleihe {
	v := service.ErsatzwertEingabe{
		Kaufpreis: a.Kaufpreis, Listenpreis: a.Listenpreis, ZustandAbschlag: a.ZustandAbschlag,
		SchuljahreMitAusleihe: a.SchuljahreMitAusleihe, SchuljahreImBestand: a.SchuljahreImBestand,
		Topf: a.Topf,
	}.Rechne(quelle)
	antwort := BescheidVorschlagAusleihe{
		AusleiheID:  a.AusleiheID,
		Titel:       a.Titel,
		ISBN:        a.ISBN,
		Betrag:      v.Betrag,
		Herleitung:  service.ErsatzwertHerleitung(v),
		Topf:        a.Topf,
		FaelligSeit: a.FaelligSeit.In(schulzeit.Zone()).Format(tuer.DateFormatISO),
		Mahnstufe:   a.Mahnstufe,
	}
	if a.LetztesMahndatum != nil {
		antwort.LetztesMahndatum = a.LetztesMahndatum.In(schulzeit.Zone()).Format(tuer.DateFormatISO)
	}
	return antwort
}
