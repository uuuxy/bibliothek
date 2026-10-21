package lernmittel

// Die Türen des LMF-Plans: lesen, rechnen und speichern, verwerfen.
//
// Der Plan ist ein Rahmen und eine Reihenfolge von Zeilen (Klassen, Vermerk). Datum und Stunde
// jeder Zeile rechnet der Server über die Schultage (service.VerteileLMFPlan, pkg/lmfplan); eine
// Zeile kann ihren Platz fest vorgeben, die anderen fließen um sie herum. Der Rahmen hängt an
// der Art: Der Büchertausch vor den Sommerferien endet (letzter_tag, letzte_stunde) und fließt
// rückwärts, sein Beginn ist gerechnet; die Ausgabe nach den Ferien beginnt (erster_tag,
// startstunde) und fließt vorwärts. Die Vorschau im Planer ist derselbe Aufruf mit
// "vorschau": true, damit es keinen JavaScript-Zwilling der Verteilung gibt.
//
// Je Art und Schuljahr gibt es einen Plan. GET liefert den neuesten der Art und, wenn keiner
// läuft, die Reihenfolge für den nächsten (service.LMFPlanReihenfolgeVorgabe). Gespeichert wird
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

// lmfPlanVorschlag ist die Reihenfolge, mit der ein neuer Plan beginnt
// (service.LMFPlanReihenfolgeVorgabe), mit seinem Rahmen.
type lmfPlanVorschlag struct {
	// Quelle: "vorjahr" (der letzte Plan der Art) oder "regel" (Abschluss zuerst).
	Quelle      string                    `json:"quelle"`
	Zeilen      []repository.LMFPlanZeile `json:"zeilen"`
	Ausgelassen []string                  `json:"ausgelassen"`
	// Rahmen: womit der neue Plan beginnt — aus den Sommerferien (service.LMFPlanRahmenVorgabe).
	Rahmen lmfPlanRahmenVorgabe `json:"rahmen"`
}

// lmfPlanSommerferien sind die Sommerferien Hessen, an denen sich der Plan ausrichtet.
type lmfPlanSommerferien struct {
	Jahr int    `json:"jahr"`
	Von  string `json:"von"` // JJJJ-MM-TT, leer, wenn nicht bekannt
	Bis  string `json:"bis"`
	// Bekannt: false, wenn das Jahr weder im Programm noch in der Einstellung
	// „Sommerferien" steht — der Planer nennt dann das Jahr und bittet um den letzten
	// bzw. ersten Tag von Hand.
	Bekannt bool `json:"bekannt"`
}

// lmfPlanRahmenVorgabe ist der Rahmen, mit dem ein neuer Plan im Planer beginnt: beim
// Rückgabe-Plan das Ende (Donnerstag vor den Ferien, 4. Stunde), beim Ausgabe-Plan der
// Beginn (erster Schultag nach den Ferien, 2. Stunde). Leere Tage: Ferien nicht bekannt.
type lmfPlanRahmenVorgabe struct {
	ErsterTag    string `json:"erster_tag"`
	Startstunde  int    `json:"startstunde"`
	LetzterTag   string `json:"letzter_tag"`
	LetzteStunde int    `json:"letzte_stunde"`
	StundenJeTag int    `json:"stunden_je_tag"`
}

// lmfPlanStandAntwort ist die Antwort von GET /api/lmf-plan/{art}.
type lmfPlanStandAntwort struct {
	// Plan ist der neueste Plan der Art; nil, wenn es noch keinen gibt.
	Plan        *repository.LMFPlan       `json:"plan"`
	Zeilen      []repository.LMFPlanZeile `json:"zeilen"`
	Ausgelassen []string                  `json:"ausgelassen"`
	// Vorbei: der letzte Termin des Plans liegt hinter heute — der Planer bietet dann
	// den nächsten Plan an, mit derselben Reihenfolge als Vorschlag.
	Vorbei bool `json:"vorbei"`
	// Vorschlag ist gesetzt, wenn es keinen laufenden Plan gibt (kein Plan oder vorbei).
	Vorschlag *lmfPlanVorschlag `json:"vorschlag,omitempty"`
	// Klassen: die Klassen mit aktiven Schülern, für die Auswahl im Planer. Eine Klasse
	// im Plan, die hier fehlt, hat (noch) keine Schüler — der Planer markiert sie.
	Klassen []string `json:"klassen"`
	// AusgelassenRegel: die Klassen aus Klassen, die die Regel dieser Art auslässt
	// (service.IstAusgelassenImLMFPlan). Der Planer klappt sie unter „bleiben draußen" ein,
	// auch bei einem laufenden Plan, der sie nie kannte; ein Plan mit alten Klassennamen böte
	// sonst jede Klasse der Schule offen an.
	AusgelassenRegel []string `json:"ausgelassen_regel"`
	// Eingangsjahrgaenge: die Jahrgänge, die nach den Ferien Bücher bekommen (Einstellung).
	Eingangsjahrgaenge []int `json:"eingangsjahrgaenge"`
	// Sommerferien: die Ferien Hessen, an denen sich der Plan ausrichtet — der Planer
	// zeigt sie neben dem Rahmen, oder den Hinweis, dass das Jahr nicht hinterlegt ist.
	Sommerferien lmfPlanSommerferien `json:"sommerferien"`
}

// leseLMFPlanArt liest {art} aus dem Pfad und prüft sie.
func leseLMFPlanArt(r *http.Request) (string, error) {
	art := r.PathValue("art")
	if art != repository.LMFTerminRueckgabe && art != repository.LMFTerminAusgabe {
		return "", apierrors.BadRequest(fmt.Sprintf("Die Art „%s“ gibt es nicht. Möglich sind rueckgabe und ausgabe.", art), nil)
	}
	return art, nil
}

// LMFPlan liefert den neuesten Plan der Art und ggf. den Vorschlag.
func (t *Tueren) LMFPlan() http.HandlerFunc {
	return apierrors.MitFehlerantwort(t.ladeLMFPlanStand)
}

// ladeLMFPlanStand stellt den Stand für den Planer zusammen: Plan, Klassen, Sommerferien und,
// wenn kein Plan läuft, den Vorschlag.
func (t *Tueren) ladeLMFPlanStand(w http.ResponseWriter, r *http.Request) error {
	art, err := leseLMFPlanArt(r)
	if err != nil {
		return err
	}
	abfragen := repository.NewLMFAbfragen(t.DB.Pool)
	einstellungen, err := repository.NewEinstellungsAbfragen(t.DB.Pool).Lade(r.Context())
	if err != nil {
		return apierrors.Internal("Einstellungen laden", err)
	}
	eingang := repository.EingangsjahrgaengeAus(einstellungen.LMFEingangsjahrgaenge)
	// Programmtabelle plus die Jahre, die die Schule selbst eingetragen hat.
	ferientabelle := lmfplan.FerientabelleAus(einstellungen.Sommerferien)
	antwort := lmfPlanStandAntwort{Zeilen: []repository.LMFPlanZeile{}, Ausgelassen: []string{}, Eingangsjahrgaenge: eingang}
	stand, err := abfragen.LadeNeuestenPlan(r.Context(), art)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// kein Plan
	case err != nil:
		return apierrors.Internal("LMF-Plan laden", err)
	default:
		plan := stand.Plan
		antwort.Plan, antwort.Zeilen, antwort.Ausgelassen = &plan, stand.Zeilen, stand.Ausgelassen
		antwort.Vorbei = service.IstLMFPlanVorbei(stand, t.Jetzt())
	}
	// Klassen mit aktiven Schülern. LeserAbfragen.LadeKlassen zählt Abgänger mit; die Klasse eines
	// Abgängers ohne aktive Schüler stünde im Planer sonst als „mit Schülern".
	klassen, err := abfragen.LadeKlassenMitSchuelern(r.Context())
	if err != nil {
		return apierrors.Internal("Klassen laden", err)
	}
	antwort.Klassen = make([]string, 0, len(klassen))
	antwort.AusgelassenRegel = []string{}
	for _, k := range klassen {
		antwort.Klassen = append(antwort.Klassen, k.Name)
		if service.IstAusgelassenImLMFPlan(art, eingang, k) {
			antwort.AusgelassenRegel = append(antwort.AusgelassenRegel, k.Name)
		}
	}
	laufend := antwort.Plan != nil && !antwort.Vorbei
	sommerferien, ferien := service.LMFPlanSommerferien(art, antwort.Plan, laufend, t.Jetzt(), ferientabelle)
	antwort.Sommerferien = lmfPlanSommerferien(sommerferien)
	if !laufend {
		reihenfolge := service.LMFPlanReihenfolgeVorgabe(art, eingang, antwort.Plan != nil, stand, klassen)
		antwort.Vorschlag = &lmfPlanVorschlag{
			Quelle:      reihenfolge.Quelle,
			Zeilen:      reihenfolge.Zeilen,
			Ausgelassen: reihenfolge.Ausgelassen,
			Rahmen:      lmfPlanRahmenVorgabe(service.LMFPlanRahmenVorgabe(art, sommerferien, ferien)),
		}
	}
	if antwort.Vorschlag != nil {
		// „nur Rückgabe" belegt den Vermerk vor und ist keine gerechnete Marke: Der Text
		// steht im Feld und lässt sich dort ändern.
		nurRueckgabe, err := t.lmfPlanNurRueckgabe(r.Context(), abfragen, art, antwort, eingang)
		if err != nil {
			return apierrors.Internal("Klassen einordnen", err)
		}
		service.BelegeVermerkNurRueckgabeVor(antwort.Vorschlag.Zeilen, nurRueckgabe)
	}
	tuer.AntworteJSON(w, http.StatusOK, antwort)
	return nil
}

// lmfPlanAnfrage ist der Körper von PUT /api/lmf-plan/{art}.
type lmfPlanAnfrage struct {
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
	Zeilen      []lmfPlanAnfrageZeile `json:"zeilen"`
	Ausgelassen []string              `json:"ausgelassen"`
	// Vorschau: nur rechnen, nichts schreiben — die Verteilung für den Planer.
	Vorschau bool `json:"vorschau"`
	// AktualisiertAm ist der Stempel des Plans, den der Planer geladen hat, leer ohne Plan.
	AktualisiertAm string `json:"aktualisiert_am"`
}

// lmfPlanAnfrageZeile ist eine Zeile der Anfrage.
type lmfPlanAnfrageZeile struct {
	Klassen []string `json:"klassen"`
	Vermerk string   `json:"vermerk"`
	// Fest: Datum und Stunde dieser Zeile von Hand — null, wenn sie fließt.
	Fest *struct {
		Datum  string `json:"datum"`
		Stunde int    `json:"stunde"`
	} `json:"fest"`
}

// lmfPlanAusfall ist ein Werktag im Plan-Zeitraum, an dem der Plan nicht läuft — mit
// Grund, damit der Planer den fehlenden Donnerstag erklärt.
type lmfPlanAusfall struct {
	Datum string `json:"datum"`
	Grund string `json:"grund"`
}

// lmfPlanSpeicherAntwort ist die Antwort von PUT und von POST …/veroeffentlichen: der
// Stand, die Ausfälle im Plan-Zeitraum und die Zahl der Ausleihen, deren Frist dem Plan
// gefolgt ist (0 bei Vorschau, bei Ausgabe-Plänen und bei einem Entwurf).
type lmfPlanSpeicherAntwort struct {
	repository.LMFPlanStand
	Vorschau         bool             `json:"vorschau"`
	Ausfaelle        []lmfPlanAusfall `json:"ausfaelle"`
	FristenAngepasst int64            `json:"fristen_angepasst"`
}

const lmfPlanMaxZeilen = 400

// pruefeLMFPlan prüft die Anfrage und füllt daraus den Entwurf, den der Dienst verteilt.
func pruefeLMFPlan(art string, req lmfPlanAnfrage) (service.LMFPlanEntwurf, error) {
	e := service.LMFPlanEntwurf{Plan: repository.LMFPlan{Art: art, StundenJeTag: req.StundenJeTag, FreieTage: []repository.LMFFreierTag{}}}
	if e.Plan.StundenJeTag < 1 || e.Plan.StundenJeTag > 12 {
		return e, apierrors.BadRequest("Die Stunden je Tag müssen zwischen 1 und 12 liegen", nil)
	}
	if err := pruefeLMFAnker(&e.Plan, req); err != nil {
		return e, err
	}
	if len(req.Zeilen) > lmfPlanMaxZeilen || len(req.FreieTage) > lmfPlanMaxZeilen {
		return e, apierrors.BadRequest(fmt.Sprintf("Der Plan hat zu viele Zeilen, höchstens %d", lmfPlanMaxZeilen), nil)
	}
	for i, f := range req.FreieTage {
		datum := strings.TrimSpace(f.Datum)
		if _, err := time.Parse(tuer.DatumsformatISO, datum); err != nil {
			return e, apierrors.BadRequest(fmt.Sprintf("Freier Tag %d: Das Datum muss die Form JJJJ-MM-TT haben.", i+1), err)
		}
		e.Plan.FreieTage = append(e.Plan.FreieTage, repository.LMFFreierTag{Datum: datum, Grund: strings.TrimSpace(f.Grund)})
	}
	e.Zeilen = make([]repository.LMFPlanZeile, 0, len(req.Zeilen))
	e.Fest = make([]*lmfplan.Platz, 0, len(req.Zeilen))
	for i, z := range req.Zeilen {
		zeile, fest, err := pruefeLMFPlanZeile(i+1, z)
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

// pruefeLMFPlanZeile prüft die Zeile mit der Nummer nr und liefert sie mit ihrem festen
// Platz; ohne festen Platz (nil) fließt sie.
func pruefeLMFPlanZeile(nr int, z lmfPlanAnfrageZeile) (repository.LMFPlanZeile, *lmfplan.Platz, error) {
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
		return zeile, nil, apierrors.BadRequest(fmt.Sprintf("Zeile %d hat weder Klasse noch Vermerk", nr), nil)
	}
	if z.Fest == nil {
		return zeile, nil, nil
	}
	tag, err := schulzeit.Kalendertag(strings.TrimSpace(z.Fest.Datum))
	if err != nil {
		return zeile, nil, apierrors.BadRequest(fmt.Sprintf("Zeile %d: Ein fester Termin braucht ein Datum der Form JJJJ-MM-TT.", nr), err)
	}
	if z.Fest.Stunde < 1 || z.Fest.Stunde > 12 {
		return zeile, nil, apierrors.BadRequest(fmt.Sprintf("Zeile %d: Die feste Stunde muss zwischen 1 und 12 liegen.", nr), nil)
	}
	return zeile, &lmfplan.Platz{Datum: tag, Stunde: z.Fest.Stunde}, nil
}

// pruefeLMFAnker prüft den Anker je Art. Der Rückgabe-Plan endet (letzter_tag,
// letzte_stunde): Beginn-Angaben der Anfrage werden verworfen, der Server rechnet den Beginn;
// bis dahin steht das Ende auch als Beginn, damit ein Plan ohne Zeilen nicht ohne Tag ist. Der
// Ausgabe-Plan beginnt (erster_tag, startstunde).
func pruefeLMFAnker(p *repository.LMFPlan, req lmfPlanAnfrage) error {
	if p.Art == repository.LMFTerminRueckgabe {
		p.LetzterTag, p.LetzteStunde = strings.TrimSpace(req.LetzterTag), req.LetzteStunde
		if _, err := time.Parse(tuer.DatumsformatISO, p.LetzterTag); err != nil {
			return apierrors.BadRequest("Der letzte Tag muss ein Datum der Form JJJJ-MM-TT sein", err)
		}
		if p.LetzteStunde < 1 || p.LetzteStunde > p.StundenJeTag {
			return apierrors.BadRequest("Die letzte Stunde muss zwischen 1 und der Zahl der Stunden je Tag liegen", nil)
		}
		p.ErsterTag, p.Startstunde = p.LetzterTag, p.LetzteStunde
		return nil
	}
	p.ErsterTag, p.Startstunde = strings.TrimSpace(req.ErsterTag), req.Startstunde
	if _, err := time.Parse(tuer.DatumsformatISO, p.ErsterTag); err != nil {
		return apierrors.BadRequest("Der erste Tag muss ein Datum der Form JJJJ-MM-TT sein", err)
	}
	if p.Startstunde < 1 || p.Startstunde > p.StundenJeTag {
		return apierrors.BadRequest("Die erste Stunde muss zwischen 1 und der Zahl der Stunden je Tag liegen", nil)
	}
	return nil
}

// LMFPlanSchreiben rechnet die Verteilung und speichert den Plan (oder zeigt sie nur).
func (t *Tueren) LMFPlanSchreiben() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		antwort, ok, err := t.rechneOderSchreibeLMFPlan(w, r)
		if err != nil || !ok {
			return err
		}
		// Das Signal beschreibt einen neuen Stand: Es kommt erst nach dem Schreiben, bei der
		// Vorschau gar nicht.
		if !antwort.Vorschau {
			t.meldeLMFPlanGeaendert()
		}
		tuer.AntworteJSON(w, http.StatusOK, antwort)
		return nil
	})
}

// rechneOderSchreibeLMFPlan prüft die Anfrage, rechnet die Verteilung und speichert den
// Plan, wenn es keine Vorschau ist. ok ist false, wenn die Anfrage schon beantwortet ist.
func (t *Tueren) rechneOderSchreibeLMFPlan(w http.ResponseWriter, r *http.Request) (lmfPlanSpeicherAntwort, bool, error) {
	var keine lmfPlanSpeicherAntwort
	art, err := leseLMFPlanArt(r)
	if err != nil {
		return keine, false, err
	}
	var req lmfPlanAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return keine, false, nil
	}
	e, err := pruefeLMFPlan(art, req)
	if err != nil {
		return keine, false, err
	}
	abfragen := repository.NewLMFAbfragen(t.DB.Pool)
	plaetze, ausgefallen, err := service.VerteileLMFPlan(&e)
	if err != nil {
		return keine, false, apierrors.Internal("Verteilung rechnen", err)
	}
	ausfaelle := lmfPlanAusfaelle(ausgefallen)
	if req.Vorschau {
		return lmfPlanVorschauAntwort(e, plaetze, ausfaelle), true, nil
	}
	// Den alten Stand dieses Schuljahres lesen: Klassen, die ihren Termin verlieren,
	// kehren zum Stichtag zurück (service.KoppleLMFPlanFristen).
	alt, err := service.LadeLMFPlanZeilenVorher(r.Context(), abfragen, art, e.Plan.ErsterTag)
	if err != nil {
		return keine, false, apierrors.Internal("alten Plan lesen", err)
	}
	// Plan und Fristen stehen in einer Transaktion. Scheiterte die Kopplung nach einem schon
	// geschriebenen Plan, wären die Fristen halb gesetzt, und ein zweiter Anlauf fände den
	// alten Plan nicht mehr, aus dem die Klassen ohne Termin kämen.
	tx, err := t.DB.Pool.Begin(r.Context())
	if err != nil {
		return keine, false, apierrors.Internal("Transaktion", err)
	}
	defer db.Verwerfe(r.Context(), tx)
	// Von einer Änderung an einem anderen Platz erfährt der Planer nur über die Live-Leitung.
	// War sie unterbrochen, überschriebe das Speichern die Änderung still; es gilt deshalb nur
	// auf dem Stand, den der Planer geladen hat.
	stempel, err := abfragen.SperreNeuestenPlan(r.Context(), tx, art)
	if err != nil {
		return keine, false, apierrors.Internal("LMF-Plan lesen", err)
	}
	if stempel != req.AktualisiertAm {
		return keine, false, apierrors.Conflict("Der Plan wurde inzwischen an einem anderen Platz geändert. Nichts gespeichert.", nil)
	}
	stand, err := abfragen.SchreibePlan(r.Context(), tx, e.Plan, e.Zeilen, plaetze, e.Ausgelassen)
	if err != nil {
		return keine, false, apierrors.Internal("LMF-Plan speichern", err)
	}
	// Ein Entwurf setzt keine Fristen — das tut erst das Veröffentlichen. Ein schon
	// veröffentlichter Plan bleibt es, und seine Korrektur gilt sofort.
	var angepasst int64
	if stand.Plan.VeroeffentlichtAm != nil {
		if angepasst, err = service.KoppleLMFPlanFristen(r.Context(), abfragen, tx, art, alt, stand.Zeilen); err != nil {
			return keine, false, apierrors.Internal("Fristen koppeln", err)
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		return keine, false, apierrors.Internal("LMF-Plan speichern", err)
	}
	// Die Spur kommt erst nach dem Commit: Sie beschreibt einen Zustand, den es vorher nicht
	// gab. Nur der veröffentlichte Plan greift in Fristen ein; ein Entwurf braucht keine Spur.
	if stand.Plan.VeroeffentlichtAm != nil {
		t.protokolliereLMFPlan(r, aktionLMFPlanGespeichert, art, stand.Plan.ID, angepasst)
	}
	return lmfPlanSpeicherAntwort{LMFPlanStand: stand, Ausfaelle: ausfaelle, FristenAngepasst: angepasst}, true, nil
}

// lmfPlanAusfaelle wandelt die Ausfälle der Verteilung in die Form der Antwort. Die Liste ist
// nie nil: Der Planer bekommt `[]` und nicht `null`.
func lmfPlanAusfaelle(ausfaelle []lmfplan.Ausfall) []lmfPlanAusfall {
	aus := make([]lmfPlanAusfall, 0, len(ausfaelle))
	for _, a := range ausfaelle {
		aus = append(aus, lmfPlanAusfall{Datum: a.Datum.Format(tuer.DatumsformatISO), Grund: a.Grund})
	}
	return aus
}

// lmfPlanVorschauAntwort trägt die gerechneten Plätze in die Zeilen des Entwurfs ein.
func lmfPlanVorschauAntwort(e service.LMFPlanEntwurf, plaetze []lmfplan.Platz, ausfaelle []lmfPlanAusfall) lmfPlanSpeicherAntwort {
	for i := range e.Zeilen {
		e.Zeilen[i].Position = i + 1
		e.Zeilen[i].Datum = plaetze[i].Datum.Format(tuer.DatumsformatISO)
		e.Zeilen[i].Stunde = plaetze[i].Stunde
	}
	return lmfPlanSpeicherAntwort{Vorschau: true, Ausfaelle: ausfaelle,
		LMFPlanStand: repository.LMFPlanStand{Plan: e.Plan, Zeilen: e.Zeilen, Ausgelassen: e.Ausgelassen}}
}

// LMFPlanVerwerfen verwirft den neuesten Plan der Art; die Fristen seiner Klassen
// gehen an den Stichtag zurück (nur die, die auf den Terminen lagen).
func (t *Tueren) LMFPlanVerwerfen() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		angepasst, err := t.verwerfeLMFPlan(r)
		if err != nil {
			return err
		}
		t.meldeLMFPlanGeaendert()
		tuer.AntworteJSON(w, http.StatusOK, map[string]int64{"fristen_angepasst": angepasst})
		return nil
	})
}

// verwerfeLMFPlan löscht den neuesten Plan der Art, bringt die Fristen seiner Klassen
// zurück und nennt, wie viele Ausleihen das traf.
func (t *Tueren) verwerfeLMFPlan(r *http.Request) (int64, error) {
	art, err := leseLMFPlanArt(r)
	if err != nil {
		return 0, err
	}
	abfragen := repository.NewLMFAbfragen(t.DB.Pool)
	stand, err := abfragen.LadeNeuestenPlan(r.Context(), art)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, apierrors.NotFound("Kein Plan vorhanden", pgx.ErrNoRows)
	}
	if err != nil {
		return 0, apierrors.Internal("LMF-Plan laden", err)
	}
	// Löschen und Rückkehr der Fristen stehen in einer Transaktion, wie beim Speichern.
	tx, err := t.DB.Pool.Begin(r.Context())
	if err != nil {
		return 0, apierrors.Internal("Transaktion", err)
	}
	defer db.Verwerfe(r.Context(), tx)
	if _, err := abfragen.LoeschePlan(r.Context(), tx, stand.Plan.ID); err != nil {
		return 0, apierrors.Internal("LMF-Plan löschen", err)
	}
	// Nur ein veröffentlichter Plan hat Fristen gesetzt, die zurückkehren müssen.
	var angepasst int64
	if stand.Plan.VeroeffentlichtAm != nil {
		if angepasst, err = service.KoppleLMFPlanFristen(r.Context(), abfragen, tx, art, stand.Zeilen, nil); err != nil {
			return 0, apierrors.Internal("Fristen koppeln", err)
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		return 0, apierrors.Internal("LMF-Plan löschen", err)
	}
	// Auch das Verwerfen eines Entwurfs wird protokolliert: Es löscht einen Plan, den
	// jemand gebaut hat, und die Antwort nennt nur eine Zahl.
	t.protokolliereLMFPlan(r, aktionLMFPlanVerworfen, art, stand.Plan.ID, angepasst)
	return angepasst, nil
}
