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

// bescheidVorschlagPosition ist eine offene Forderung mit dem vorgeschlagenen Betrag und
// seiner Herleitung — der Dialog zeigt beides, damit der Mensch die Zahl beurteilen kann.
type bescheidVorschlagPosition struct {
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
	// Brief trägt genau einen Topf.
	Topf string `json:"topf"`
}

// bescheidVorschlagAusleihe ist ein überfälliges Buch ohne Forderung: Es kommt mit dem
// Brief als Verlust in die Bücher. Betrag und Herleitung wie bei einer Position.
type bescheidVorschlagAusleihe struct {
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

// bescheidVorschlagAntwort ist die Antwort für den Dialog.
type bescheidVorschlagAntwort struct {
	SchuelerName string                      `json:"schueler_name"`
	Klasse       string                      `json:"klasse"`
	FristBis     string                      `json:"frist_bis"`
	Positionen   []bescheidVorschlagPosition `json:"positionen"`
	// Ausleihen: überfällige Bücher, die noch keine Forderung tragen; sie brauchen vor dem
	// Brief keine Verlustmeldung je Buch.
	Ausleihen []bescheidVorschlagAusleihe `json:"ausleihen"`
	// FehlendeAngaben nennt die Einstellungen, ohne die kein Bescheid entstehen kann.
	// Der Dialog zeigt sie, statt den Knopf stumm zu sperren.
	FehlendeAngaben []string `json:"fehlende_angaben"`
}

// BescheidVorschlag liefert die offenen Forderungen eines Schülers mit
// Betragsvorschlag.
func (t *Tueren) BescheidVorschlag(bescheidAbfragen *repository.BescheidAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		return t.schlageBescheidVor(w, r, bescheidAbfragen)
	})
}

// schlageBescheidVor sammelt, was auf einen Bescheid kann: offene Forderungen und
// überfällige Bücher, je mit Betragsvorschlag.
func (t *Tueren) schlageBescheidVor(w http.ResponseWriter, r *http.Request, bescheidAbfragen *repository.BescheidAbfragen) error {
	id := r.PathValue("id")
	if id == "" {
		return apierrors.BadRequest(tuer.MeldungIDLeer, errors.New("missing id"))
	}
	ctx := r.Context()

	angaben, schule, err := t.bescheidAngaben(ctx)
	if err != nil {
		return apierrors.Internal(tuer.MeldungEinstellungenUnlesbar, err)
	}
	// Welcher Preis die Grundlage ist, steht in denselben Einstellungen.
	quelle := service.LadeErsatzwertPreisquelle(ctx, t.DB.Pool)

	vorschlag := bescheidVorschlagAntwort{
		FristBis:        schulzeit.Jetzt().AddDate(0, 0, angaben.FristTage).Format(tuer.DatumsformatISO),
		FehlendeAngaben: angaben.FehlendeAngaben(schule),
		Positionen:      []bescheidVorschlagPosition{},
		Ausleihen:       []bescheidVorschlagAusleihe{},
	}
	empfaenger, err := bescheidAbfragen.LadeEmpfaenger(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apierrors.NotFound("Schüler nicht gefunden", err)
		}
		return apierrors.Internal("Schüler konnte nicht gelesen werden", err)
	}
	vorschlag.SchuelerName = strings.TrimSpace(empfaenger.Vorname + " " + empfaenger.Nachname)
	vorschlag.Klasse = empfaenger.Klasse

	offene, err := bescheidAbfragen.LadeOffeneForderungen(ctx, id)
	if err != nil {
		return apierrors.Internal("Offene Forderungen konnten nicht gelesen werden", err)
	}
	for _, f := range offene {
		vorschlag.Positionen = append(vorschlag.Positionen, bescheidVorschlagAus(f, quelle))
	}
	ueberfaellig, err := bescheidAbfragen.LadeUeberfaelligeAusleihen(ctx, id)
	if err != nil {
		return apierrors.Internal("Überfällige Ausleihen konnten nicht gelesen werden", err)
	}
	for _, a := range ueberfaellig {
		vorschlag.Ausleihen = append(vorschlag.Ausleihen, bescheidVorschlagAusAusleihe(a, quelle))
	}

	tuer.AntworteJSON(w, http.StatusOK, vorschlag)
	return nil
}

// bescheidVorschlagAus rechnet den Staffel-Vorschlag für eine offene Forderung, mit
// Listenpreis und Zustand des Exemplars: Die Arbeitshilfe verlangt ab dem zweiten Verleihjahr
// den Neupreis zum Zeitpunkt des Verlusts. Ist kein Listenpreis erfasst, weicht die Staffel
// auf den Kaufpreis aus und sagt es.
func bescheidVorschlagAus(f repository.OffeneForderung, quelle ersatzwert.Preisquelle) bescheidVorschlagPosition {
	v := service.ErsatzwertEingabe{
		Kaufpreis: f.Kaufpreis, Listenpreis: f.Listenpreis, ZustandAbschlag: f.ZustandAbschlag,
		SchuljahreMitAusleihe: f.SchuljahreMitAusleihe, SchuljahreImBestand: f.SchuljahreImBestand,
		Topf: f.Topf,
	}.Rechne(quelle)
	return bescheidVorschlagPosition{
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
func bescheidVorschlagAusAusleihe(a repository.UeberfaelligeAusleihe, quelle ersatzwert.Preisquelle) bescheidVorschlagAusleihe {
	v := service.ErsatzwertEingabe{
		Kaufpreis: a.Kaufpreis, Listenpreis: a.Listenpreis, ZustandAbschlag: a.ZustandAbschlag,
		SchuljahreMitAusleihe: a.SchuljahreMitAusleihe, SchuljahreImBestand: a.SchuljahreImBestand,
		Topf: a.Topf,
	}.Rechne(quelle)
	antwort := bescheidVorschlagAusleihe{
		AusleiheID:  a.AusleiheID,
		Titel:       a.Titel,
		ISBN:        a.ISBN,
		Betrag:      v.Betrag,
		Herleitung:  service.ErsatzwertHerleitung(v),
		Topf:        a.Topf,
		FaelligSeit: a.FaelligSeit.In(schulzeit.Zone()).Format(tuer.DatumsformatISO),
		Mahnstufe:   a.Mahnstufe,
	}
	if a.LetztesMahndatum != nil {
		antwort.LetztesMahndatum = a.LetztesMahndatum.In(schulzeit.Zone()).Format(tuer.DatumsformatISO)
	}
	return antwort
}
