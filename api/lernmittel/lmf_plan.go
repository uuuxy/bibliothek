package lernmittel

// lmf_plan.go — die Türen des LMF-Plans: lesen, rechnen und speichern, verwerfen.
//
// Der Plan ist ein Rahmen und eine Reihenfolge von Zeilen (Klassen, Vermerk). Datum und Stunde
// jeder Zeile rechnet der Server über die Schultage (service.VerteileLmfPlan, pkg/lmfplan); eine
// Zeile kann ihren Platz fest vorgeben, die anderen fließen um sie herum. Der Rahmen hängt an
// der Art: Der Büchertausch vor den Sommerferien endet (letzter_tag, letzte_stunde) und fließt
// rückwärts, sein Beginn ist gerechnet; die Ausgabe nach den Ferien beginnt (erster_tag,
// startstunde) und fließt vorwärts. Die Vorschau im Planer ist derselbe Aufruf mit
// "vorschau": true, damit es keinen JavaScript-Zwilling der Verteilung gibt.
//
// Je Art und Schuljahr gibt es einen Plan. GET liefert den neuesten der Art und, wenn keiner
// läuft, die Reihenfolge für den nächsten (service.LmfPlanReihenfolgeVorgabe). Gespeichert wird
// ein Entwurf; erst das Veröffentlichen macht ihn sichtbar und setzt Fristen
// (lmf_plan_veroeffentlichung.go).

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// LmfPlanVorschlag ist die Reihenfolge, mit der ein neuer Plan beginnt
// (service.LmfPlanReihenfolgeVorgabe), mit seinem Rahmen.
type LmfPlanVorschlag struct {
	// Quelle: "vorjahr" (der letzte Plan der Art) oder "regel" (Abschluss zuerst).
	Quelle      string                    `json:"quelle"`
	Zeilen      []repository.LMFPlanZeile `json:"zeilen"`
	Ausgelassen []string                  `json:"ausgelassen"`
	// Rahmen: womit der neue Plan beginnt — aus den Sommerferien (service.LmfPlanRahmenVorgabe).
	Rahmen LmfPlanRahmenVorgabe `json:"rahmen"`
}

// LmfPlanSommerferien sind die Sommerferien Hessen, an denen sich der Plan ausrichtet.
type LmfPlanSommerferien struct {
	Jahr int    `json:"jahr"`
	Von  string `json:"von"` // YYYY-MM-DD, leer wenn nicht bekannt
	Bis  string `json:"bis"`
	// Bekannt: false, wenn das Jahr weder im Programm noch in der Einstellung
	// „Sommerferien" steht — der Planer nennt dann das Jahr und bittet um den letzten
	// bzw. ersten Tag von Hand.
	Bekannt bool `json:"bekannt"`
}

// LmfPlanRahmenVorgabe ist der Rahmen, mit dem ein neuer Plan im Planer beginnt: beim
// Rückgabe-Plan das Ende (Donnerstag vor den Ferien, 4. Stunde), beim Ausgabe-Plan der
// Beginn (erster Schultag nach den Ferien, 2. Stunde). Leere Tage: Ferien nicht bekannt.
type LmfPlanRahmenVorgabe struct {
	ErsterTag    string `json:"erster_tag"`
	Startstunde  int    `json:"startstunde"`
	LetzterTag   string `json:"letzter_tag"`
	LetzteStunde int    `json:"letzte_stunde"`
	StundenJeTag int    `json:"stunden_je_tag"`
}

// LmfPlanStandAntwort ist die Antwort von GET /api/lmf-plan/{art}.
type LmfPlanStandAntwort struct {
	// Plan ist der neueste Plan der Art; nil, wenn es noch keinen gibt.
	Plan        *repository.LMFPlan       `json:"plan"`
	Zeilen      []repository.LMFPlanZeile `json:"zeilen"`
	Ausgelassen []string                  `json:"ausgelassen"`
	// Vorbei: der letzte Termin des Plans liegt hinter heute — der Planer bietet dann
	// den nächsten Plan an, mit derselben Reihenfolge als Vorschlag.
	Vorbei bool `json:"vorbei"`
	// Vorschlag ist gesetzt, wenn es keinen laufenden Plan gibt (kein Plan oder vorbei).
	Vorschlag *LmfPlanVorschlag `json:"vorschlag,omitempty"`
	// Klassen: die Klassen mit aktiven Schülern, für die Auswahl im Planer. Eine Klasse
	// im Plan, die hier fehlt, hat (noch) keine Schüler — der Planer markiert sie.
	Klassen []string `json:"klassen"`
	// AusgelassenRegel: die Klassen aus Klassen, die die Regel dieser Art auslässt
	// (service.LmfPlanLaesstAus). Der Planer klappt sie unter „bleiben draußen" ein, auch bei
	// einem laufenden Plan, der sie nie kannte; ein Plan mit alten Klassennamen böte sonst
	// jede Klasse der Schule offen an.
	AusgelassenRegel []string `json:"ausgelassen_regel"`
	// Eingangsjahrgaenge: die Jahrgänge, die nach den Ferien Bücher bekommen (Einstellung).
	Eingangsjahrgaenge []int `json:"eingangsjahrgaenge"`
	// Sommerferien: die Ferien Hessen, an denen sich der Plan ausrichtet — der Planer
	// zeigt sie neben dem Rahmen, oder den Hinweis, dass das Jahr nicht hinterlegt ist.
	Sommerferien LmfPlanSommerferien `json:"sommerferien"`
}

// lmfPlanArt liest {art} aus dem Pfad und prüft sie.
func lmfPlanArt(r *http.Request) (string, error) {
	art := r.PathValue("art")
	if art != repository.LMFTerminRueckgabe && art != repository.LMFTerminAusgabe {
		return "", errors.New("art muss 'rueckgabe' oder 'ausgabe' sein")
	}
	return art, nil
}

// GetLmfPlanHandler liefert den neuesten Plan der Art und ggf. den Vorschlag.
// @Summary      LMF-Plan (Reihenfolge) lesen
// @Tags         lernmittel
// @Produce      json
// @Success      200  {object}  LmfPlanStandAntwort
// @Router       /lmf-plan/{art} [get]
func (s *Tueren) GetLmfPlanHandler() http.HandlerFunc {
	return apierrors.Wrap(s.handleGetLmfPlan)
}

// handleGetLmfPlan stellt den Stand für den Planer zusammen: Plan, Klassen, Sommerferien und,
// wenn kein Plan läuft, den Vorschlag.
func (s *Tueren) handleGetLmfPlan(w http.ResponseWriter, r *http.Request) error {
	art, err := lmfPlanArt(r)
	if err != nil {
		return apierrors.BadRequest(err.Error(), err)
	}
	repo := repository.NewLMFAbfragen(s.DB.Pool)
	einstellungen, err := repository.NewEinstellungsAbfragen(s.DB.Pool).Lade(r.Context())
	if err != nil {
		return apierrors.Internal("Einstellungen laden", err)
	}
	eingang := repository.EingangsjahrgaengeAus(einstellungen.LMFEingangsjahrgaenge)
	// Programmtabelle plus die Jahre, die die Schule selbst eingetragen hat.
	ferientabelle := lmfplan.FerientabelleAus(einstellungen.Sommerferien)
	antwort := LmfPlanStandAntwort{Zeilen: []repository.LMFPlanZeile{}, Ausgelassen: []string{}, Eingangsjahrgaenge: eingang}
	stand, err := repo.LadeNeuestenPlan(r.Context(), art)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// kein Plan
	case err != nil:
		return apierrors.Internal("LMF-Plan laden", err)
	default:
		plan := stand.Plan
		antwort.Plan, antwort.Zeilen, antwort.Ausgelassen = &plan, stand.Zeilen, stand.Ausgelassen
		antwort.Vorbei = service.LmfPlanVorbei(stand, s.Jetzt())
	}
	// Klassen mit aktiven Schülern. LeserAbfragen.LadeKlassen zählt Abgänger mit; die Klasse eines
	// Abgängers ohne aktive Schüler stünde im Planer sonst als „mit Schülern".
	klassen, err := repo.LadeKlassenMitSchuelern(r.Context())
	if err != nil {
		return apierrors.Internal("Klassen laden", err)
	}
	antwort.Klassen = make([]string, 0, len(klassen))
	antwort.AusgelassenRegel = []string{}
	for _, k := range klassen {
		antwort.Klassen = append(antwort.Klassen, k.Name)
		if service.LmfPlanLaesstAus(art, eingang, k) {
			antwort.AusgelassenRegel = append(antwort.AusgelassenRegel, k.Name)
		}
	}
	laufend := antwort.Plan != nil && !antwort.Vorbei
	sommerferien, ferien := service.LmfPlanSommerferien(art, antwort.Plan, laufend, s.Jetzt(), ferientabelle)
	antwort.Sommerferien = LmfPlanSommerferien(sommerferien)
	if !laufend {
		reihenfolge := service.LmfPlanReihenfolgeVorgabe(art, eingang, antwort.Plan != nil, stand, klassen)
		antwort.Vorschlag = &LmfPlanVorschlag{
			Quelle:      reihenfolge.Quelle,
			Zeilen:      reihenfolge.Zeilen,
			Ausgelassen: reihenfolge.Ausgelassen,
			Rahmen:      LmfPlanRahmenVorgabe(service.LmfPlanRahmenVorgabe(art, sommerferien, ferien)),
		}
	}
	if antwort.Vorschlag != nil {
		// „nur Rückgabe" belegt den Vermerk vor und ist keine gerechnete Marke: Der Text
		// steht im Feld und lässt sich dort ändern.
		nurRueckgabe, err := s.lmfPlanNurRueckgabe(r.Context(), repo, art, antwort, eingang)
		if err != nil {
			return apierrors.Internal("Klassen einordnen", err)
		}
		service.BelegeLmfPlanVermerkNurRueckgabe(antwort.Vorschlag.Zeilen, nurRueckgabe)
	}
	tuer.RespondJSON(w, http.StatusOK, antwort)
	return nil
}

// lmfPlanRequest ist der Körper von PUT /api/lmf-plan/{art}.
type lmfPlanRequest struct {
	// Ausgabe-Plan: Beginn. Beim Rückgabe-Plan werden beide verworfen — der Server
	// rechnet den Beginn aus dem Ende.
	ErsterTag   string `json:"erster_tag"`
	Startstunde int    `json:"startstunde"`
	// Rückgabe-Plan: Ende (Donnerstag vor den Sommerferien, 4. Stunde).
	LetzterTag   string `json:"letzter_tag"`
	LetzteStunde int    `json:"letzte_stunde"`
	StundenJeTag int    `json:"stunden_je_tag"`
	// FreieTage: Tage, die der Plan überspringt (Brückentag, pädagogischer Tag).
	FreieTage []struct {
		Datum string `json:"datum"`
		Grund string `json:"grund"`
	} `json:"freie_tage"`
	Zeilen      []lmfPlanRequestZeile `json:"zeilen"`
	Ausgelassen []string              `json:"ausgelassen"`
	// Vorschau: nur rechnen, nichts schreiben — die Verteilung für den Planer.
	Vorschau bool `json:"vorschau"`
}

// lmfPlanRequestZeile ist eine Zeile der Anfrage.
type lmfPlanRequestZeile struct {
	Klassen []string `json:"klassen"`
	Vermerk string   `json:"vermerk"`
	// Fest: Datum und Stunde dieser Zeile von Hand — null, wenn sie fließt.
	Fest *struct {
		Datum  string `json:"datum"`
		Stunde int    `json:"stunde"`
	} `json:"fest"`
}

// LmfPlanAusfall ist ein Werktag im Plan-Zeitraum, an dem der Plan nicht läuft — mit
// Grund, damit der Planer den fehlenden Donnerstag erklärt.
type LmfPlanAusfall struct {
	Datum string `json:"datum"`
	Grund string `json:"grund"`
}

// LmfPlanSpeicherAntwort ist die Antwort von PUT und von POST …/veroeffentlichen: der
// Stand, die Ausfälle im Plan-Zeitraum und die Zahl der Ausleihen, deren Frist dem Plan
// gefolgt ist (0 bei Vorschau, bei Ausgabe-Plänen und bei einem Entwurf).
type LmfPlanSpeicherAntwort struct {
	repository.LMFPlanStand
	Vorschau         bool             `json:"vorschau"`
	Ausfaelle        []LmfPlanAusfall `json:"ausfaelle"`
	FristenAngepasst int64            `json:"fristen_angepasst"`
}

const lmfPlanMaxZeilen = 400

// pruefeLmfPlan prüft die Anfrage und füllt daraus den Entwurf, den der Dienst verteilt.
func pruefeLmfPlan(art string, req lmfPlanRequest) (service.LmfPlanEntwurf, error) {
	e := service.LmfPlanEntwurf{Plan: repository.LMFPlan{Art: art, StundenJeTag: req.StundenJeTag, FreieTage: []repository.LMFFreierTag{}}}
	if e.Plan.StundenJeTag < 1 || e.Plan.StundenJeTag > 12 {
		return e, errors.New("stunden_je_tag muss zwischen 1 und 12 liegen")
	}
	if err := pruefeLmfAnker(&e.Plan, req); err != nil {
		return e, err
	}
	if len(req.Zeilen) > lmfPlanMaxZeilen || len(req.FreieTage) > lmfPlanMaxZeilen {
		return e, errors.New("zu viele Zeilen")
	}
	for i, f := range req.FreieTage {
		datum := strings.TrimSpace(f.Datum)
		if _, err := time.Parse(tuer.DateFormatISO, datum); err != nil {
			return e, fmt.Errorf("freier Tag %d: Datum muss als JJJJ-MM-TT angegeben sein", i+1)
		}
		e.Plan.FreieTage = append(e.Plan.FreieTage, repository.LMFFreierTag{Datum: datum, Grund: strings.TrimSpace(f.Grund)})
	}
	e.Zeilen = make([]repository.LMFPlanZeile, 0, len(req.Zeilen))
	e.Fest = make([]*lmfplan.Platz, 0, len(req.Zeilen))
	for i, z := range req.Zeilen {
		zeile, fest, err := pruefeLmfPlanZeile(i+1, z)
		if err != nil {
			return e, err
		}
		e.Zeilen = append(e.Zeilen, zeile)
		e.Fest = append(e.Fest, fest)
	}
	for _, k := range req.Ausgelassen {
		if k = strings.TrimSpace(k); k != "" {
			e.Ausgelassen = append(e.Ausgelassen, k)
		}
	}
	return e, nil
}

// pruefeLmfPlanZeile prüft die Zeile mit der Nummer nr und liefert sie mit ihrem festen
// Platz; ohne festen Platz (nil) fließt sie.
func pruefeLmfPlanZeile(nr int, z lmfPlanRequestZeile) (repository.LMFPlanZeile, *lmfplan.Platz, error) {
	// Klassen als leere Liste anlegen, nicht als nil: Die Vorschau lieferte sonst
	// `klassen: null`, wo der gespeicherte Plan `[]` liefert (schreibeKlassen); die Oberfläche
	// müsste zwei Formen derselben Zeile kennen. Betroffen sind die Zeilen ohne Klasse, die es
	// im Plan der Schule gibt („Nachzügler", „Aufräumen").
	zeile := repository.LMFPlanZeile{
		Vermerk: strings.TrimSpace(z.Vermerk),
		Fest:    z.Fest != nil,
		Klassen: make([]string, 0, len(z.Klassen)),
	}
	for _, k := range z.Klassen {
		if k = strings.TrimSpace(k); k != "" {
			zeile.Klassen = append(zeile.Klassen, k)
		}
	}
	if len(zeile.Klassen) == 0 && zeile.Vermerk == "" {
		return zeile, nil, fmt.Errorf("zeile %d hat weder Klasse noch Vermerk", nr)
	}
	if z.Fest == nil {
		return zeile, nil, nil
	}
	tag, err := schulzeit.Kalendertag(strings.TrimSpace(z.Fest.Datum))
	if err != nil {
		return zeile, nil, fmt.Errorf("zeile %d: fester Termin braucht ein Datum (JJJJ-MM-TT)", nr)
	}
	if z.Fest.Stunde < 1 || z.Fest.Stunde > 12 {
		return zeile, nil, fmt.Errorf("zeile %d: feste Stunde muss zwischen 1 und 12 liegen", nr)
	}
	return zeile, &lmfplan.Platz{Datum: tag, Stunde: z.Fest.Stunde}, nil
}

// pruefeLmfAnker prüft den Anker je Art. Der Rückgabe-Plan endet (letzter_tag,
// letzte_stunde): Beginn-Angaben der Anfrage werden verworfen, der Server rechnet den Beginn;
// bis dahin steht das Ende auch als Beginn, damit ein Plan ohne Zeilen nicht ohne Tag ist. Der
// Ausgabe-Plan beginnt (erster_tag, startstunde).
func pruefeLmfAnker(p *repository.LMFPlan, req lmfPlanRequest) error {
	if p.Art == repository.LMFTerminRueckgabe {
		p.LetzterTag, p.LetzteStunde = strings.TrimSpace(req.LetzterTag), req.LetzteStunde
		if _, err := time.Parse(tuer.DateFormatISO, p.LetzterTag); err != nil {
			return errors.New("letzter_tag muss als JJJJ-MM-TT angegeben sein")
		}
		if p.LetzteStunde < 1 || p.LetzteStunde > p.StundenJeTag {
			return errors.New("letzte_stunde muss zwischen 1 und stunden_je_tag liegen")
		}
		p.ErsterTag, p.Startstunde = p.LetzterTag, p.LetzteStunde
		return nil
	}
	p.ErsterTag, p.Startstunde = strings.TrimSpace(req.ErsterTag), req.Startstunde
	if _, err := time.Parse(tuer.DateFormatISO, p.ErsterTag); err != nil {
		return errors.New("erster_tag muss als JJJJ-MM-TT angegeben sein")
	}
	if p.Startstunde < 1 || p.Startstunde > p.StundenJeTag {
		return errors.New("startstunde muss zwischen 1 und stunden_je_tag liegen")
	}
	return nil
}

// PutLmfPlanHandler rechnet die Verteilung und speichert den Plan (oder zeigt sie nur).
// @Summary      LMF-Plan (Reihenfolge) speichern oder als Vorschau rechnen
// @Tags         lernmittel
// @Accept       json
// @Produce      json
// @Success      200  {object}  LmfPlanSpeicherAntwort
// @Router       /lmf-plan/{art} [put]
func (s *Tueren) PutLmfPlanHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		antwort, ok, err := s.rechneOderSpeichereLmfPlan(w, r)
		if err != nil || !ok {
			return err
		}
		// Das Signal beschreibt einen neuen Stand: Es kommt erst nach dem Schreiben, bei der
		// Vorschau gar nicht.
		if !antwort.Vorschau {
			s.meldeLmfPlanGeaendert()
		}
		tuer.RespondJSON(w, http.StatusOK, antwort)
		return nil
	})
}

// rechneOderSpeichereLmfPlan prüft die Anfrage, rechnet die Verteilung und speichert den
// Plan, wenn es keine Vorschau ist. ok ist false, wenn die Anfrage schon beantwortet ist.
func (s *Tueren) rechneOderSpeichereLmfPlan(w http.ResponseWriter, r *http.Request) (LmfPlanSpeicherAntwort, bool, error) {
	var keine LmfPlanSpeicherAntwort
	art, err := lmfPlanArt(r)
	if err != nil {
		return keine, false, apierrors.BadRequest(err.Error(), err)
	}
	var req lmfPlanRequest
	if !tuer.DecodeAndValidate(w, r, &req) {
		return keine, false, nil
	}
	e, err := pruefeLmfPlan(art, req)
	if err != nil {
		return keine, false, apierrors.BadRequest(err.Error(), err)
	}
	repo := repository.NewLMFAbfragen(s.DB.Pool)
	plaetze, ausgefallen, err := service.VerteileLmfPlan(&e)
	if err != nil {
		return keine, false, apierrors.Internal("Verteilung rechnen", err)
	}
	ausfaelle := lmfPlanAusfaelle(ausgefallen)
	if req.Vorschau {
		return lmfPlanVorschauAntwort(e, plaetze, ausfaelle), true, nil
	}
	// Den alten Stand dieses Schuljahres lesen: Klassen, die ihren Termin verlieren,
	// kehren zum Stichtag zurück (service.KoppleLmfPlanFristen).
	alt, err := service.LmfPlanZeilenVorher(r.Context(), repo, art, e.Plan.ErsterTag)
	if err != nil {
		return keine, false, apierrors.Internal("alten Plan lesen", err)
	}
	// Plan und Fristen stehen in einer Transaktion. Scheiterte die Kopplung nach einem schon
	// geschriebenen Plan, wären die Fristen halb gesetzt, und ein zweiter Anlauf fände den
	// alten Plan nicht mehr, aus dem die Klassen ohne Termin kämen.
	tx, err := s.DB.Pool.Begin(r.Context())
	if err != nil {
		return keine, false, apierrors.Internal("Transaktion", err)
	}
	defer db.SafeRollback(r.Context(), tx)
	stand, err := repo.SchreibePlan(r.Context(), tx, e.Plan, e.Zeilen, plaetze, e.Ausgelassen)
	if err != nil {
		return keine, false, apierrors.Internal("LMF-Plan speichern", err)
	}
	// Ein Entwurf setzt keine Fristen — das tut erst das Veröffentlichen. Ein schon
	// veröffentlichter Plan bleibt es, und seine Korrektur gilt sofort.
	var angepasst int64
	if stand.Plan.VeroeffentlichtAm != nil {
		if angepasst, err = service.KoppleLmfPlanFristen(r.Context(), repo, tx, art, alt, stand.Zeilen); err != nil {
			return keine, false, apierrors.Internal("Fristen koppeln", err)
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		return keine, false, apierrors.Internal("LMF-Plan speichern", err)
	}
	// Die Spur kommt erst nach dem Commit: Sie beschreibt einen Zustand, den es vorher nicht
	// gab. Nur der veröffentlichte Plan greift in Fristen ein; ein Entwurf braucht keine Spur.
	if stand.Plan.VeroeffentlichtAm != nil {
		s.auditiereLmfPlan(r, auditLmfPlanGespeichert, art, stand.Plan.ID, angepasst)
	}
	return LmfPlanSpeicherAntwort{LMFPlanStand: stand, Ausfaelle: ausfaelle, FristenAngepasst: angepasst}, true, nil
}

// lmfPlanAusfaelle wandelt die Ausfälle der Verteilung in die Form der Antwort. Die Liste ist
// nie nil: Der Planer bekommt `[]` und nicht `null`.
func lmfPlanAusfaelle(ausfaelle []lmfplan.Ausfall) []LmfPlanAusfall {
	aus := make([]LmfPlanAusfall, 0, len(ausfaelle))
	for _, a := range ausfaelle {
		aus = append(aus, LmfPlanAusfall{Datum: a.Datum.Format(tuer.DateFormatISO), Grund: a.Grund})
	}
	return aus
}

// lmfPlanVorschauAntwort trägt die gerechneten Plätze in die Zeilen des Entwurfs ein.
func lmfPlanVorschauAntwort(e service.LmfPlanEntwurf, plaetze []lmfplan.Platz, ausfaelle []LmfPlanAusfall) LmfPlanSpeicherAntwort {
	for i := range e.Zeilen {
		e.Zeilen[i].Position = i + 1
		e.Zeilen[i].Datum = plaetze[i].Datum.Format(tuer.DateFormatISO)
		e.Zeilen[i].Stunde = plaetze[i].Stunde
	}
	return LmfPlanSpeicherAntwort{Vorschau: true, Ausfaelle: ausfaelle,
		LMFPlanStand: repository.LMFPlanStand{Plan: e.Plan, Zeilen: e.Zeilen, Ausgelassen: e.Ausgelassen}}
}

// DeleteLmfPlanHandler verwirft den neuesten Plan der Art; die Fristen seiner Klassen
// gehen an den Stichtag zurück (nur die, die auf den Terminen lagen).
// @Summary      LMF-Plan (Reihenfolge) verwerfen
// @Tags         lernmittel
// @Produce      json
// @Success      200  {object}  map[string]int64
// @Router       /lmf-plan/{art} [delete]
func (s *Tueren) DeleteLmfPlanHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		angepasst, err := s.verwerfeLmfPlan(r)
		if err != nil {
			return err
		}
		s.meldeLmfPlanGeaendert()
		tuer.RespondJSON(w, http.StatusOK, map[string]int64{"fristen_angepasst": angepasst})
		return nil
	})
}

// verwerfeLmfPlan löscht den neuesten Plan der Art, bringt die Fristen seiner Klassen
// zurück und nennt, wie viele Ausleihen das traf.
func (s *Tueren) verwerfeLmfPlan(r *http.Request) (int64, error) {
	art, err := lmfPlanArt(r)
	if err != nil {
		return 0, apierrors.BadRequest(err.Error(), err)
	}
	repo := repository.NewLMFAbfragen(s.DB.Pool)
	st, err := repo.LadeNeuestenPlan(r.Context(), art)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, apierrors.NotFound("kein Plan vorhanden", pgx.ErrNoRows)
	}
	if err != nil {
		return 0, apierrors.Internal("LMF-Plan laden", err)
	}
	// Löschen und Rückkehr der Fristen stehen in einer Transaktion, wie beim Speichern.
	tx, err := s.DB.Pool.Begin(r.Context())
	if err != nil {
		return 0, apierrors.Internal("Transaktion", err)
	}
	defer db.SafeRollback(r.Context(), tx)
	if _, err := repo.LoeschePlan(r.Context(), tx, st.Plan.ID); err != nil {
		return 0, apierrors.Internal("LMF-Plan löschen", err)
	}
	// Nur ein veröffentlichter Plan hat Fristen gesetzt, die zurückkehren müssen.
	var angepasst int64
	if st.Plan.VeroeffentlichtAm != nil {
		if angepasst, err = service.KoppleLmfPlanFristen(r.Context(), repo, tx, art, st.Zeilen, nil); err != nil {
			return 0, apierrors.Internal("Fristen koppeln", err)
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		return 0, apierrors.Internal("LMF-Plan löschen", err)
	}
	// Auch das Verwerfen eines Entwurfs wird protokolliert: Es löscht einen Plan, den
	// jemand gebaut hat, und die Antwort nennt nur eine Zahl.
	s.auditiereLmfPlan(r, auditLmfPlanVerworfen, art, st.Plan.ID, angepasst)
	return angepasst, nil
}
