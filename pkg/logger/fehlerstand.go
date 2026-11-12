package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// FehlerZeitraum ist der Zeitraum, über den der Fehlerstand zählt. Der Wächter mailt alle 24
// Stunden; so sieht jede Mail jede Zeile seit der vorigen.
const FehlerZeitraum = 24 * time.Hour

// fehlerFach ist die Breite der Fächer, in denen der Fehlerstand Zeilen zählt. Gezählt wird jedes
// Fach, das in den Zeitraum reicht: höchstens zehn Minuten zu viel, nie zu wenig, damit keine
// Mail eine Zeile seit der vorigen verliert.
const fehlerFach = 10 * time.Minute

// hoechstzahlGruppen begrenzt die Gruppen, die der Fehlerstand einzeln zählt; jede weitere Zeile
// zählt die Sammelgruppe, verworfen wird keine (OpenTelemetry, „Cardinality limits":
// „Measurements MUST NOT be double-counted or dropped during an overflow"). Feste Meldungen
// (docs/STIL.md, Abschnitt 11) und die Routen des Servers erreichen die Zahl nicht.
// hoechstlaengeUrsache kürzt die Ursache für Seite und Mail; den ganzen Text zeigt das Log.
// hoechstlaengeZeile begrenzt die letzte Logzeile je Gruppe, die die Mail ganz zeigt: Sie trägt,
// was zum Beheben nötig ist (Zeit, Route, Ursache, bei einem Panic den Stack), und bleibt
// begrenzt, weil der Stand alle zehn Sekunden in die Datenbank geht.
const (
	hoechstzahlGruppen   = 500
	hoechstlaengeUrsache = 200
	hoechstlaengeZeile   = 4000
)

// Fehlerstand zählt die Zeilen der Stufen ERROR und WARN im Log, je Stufe, Meldung und Route,
// über den FehlerZeitraum. Die Selbstprüfung zeigt die Fehler, ihr Wächter mailt sie, die
// Warnungen nur mit dem Schalter alarm_warnungen; so fällt beides auf, ohne dass jemand das Log
// liest.
type Fehlerstand struct {
	mu      sync.Mutex
	gruppen map[schluessel]*gruppe
	// sammel und sammelWarnung zählen je Stufe die Zeilen, für die über hoechstzahlGruppen hinaus
	// keine eigene Gruppe entstand: Eine Warnung darf nicht als Fehler gezählt werden.
	sammel        *gruppe
	sammelWarnung *gruppe
	// fassung zählt die Änderungen, abgelegt ist die zuletzt abgelegte Fassung. Bis der Stand des
	// vorigen Laufs übernommen ist, legt der Fehlerstand nichts ab: Er überschriebe sonst, was der
	// vorige Lauf gezählt hat.
	fassung     uint64
	abgelegt    uint64
	uebernommen bool
	// ablageSperre reiht das Ablegen: Beim Herunterfahren legen der Takt und main zugleich ab, und
	// ein älterer Stand darf den jüngeren nicht überschreiben.
	ablageSperre sync.Mutex
}

// schluessel trennt die Gruppen: die Stufe, die Meldung und, bei der Zeile zu einer Anfrage, ihre
// Route.
type schluessel struct {
	warnung        bool
	meldung, route string
}

// gruppe sind die Zeilen einer Meldung. Die Felder tragen Namen für JSON, weil der Stand so
// abgelegt wird.
type gruppe struct {
	// Faecher zählt die Zeilen je Fach, unter dem Beginn des Fachs in Unix-Sekunden.
	Faecher map[int64]int `json:"faecher"`
	Zuletzt time.Time     `json:"zuletzt"`
	Ursache string        `json:"ursache,omitempty"`
	// Zeile ist die letzte Zeile, wie das Log sie schreibt, mit allen Feldern.
	Zeile string `json:"zeile,omitempty"`
}

// abgelegterStand ist die Form des Stands in system_einstellungen.
type abgelegterStand struct {
	Gruppen       []abgelegteGruppe `json:"gruppen"`
	Sammel        *gruppe           `json:"sammel,omitempty"`
	SammelWarnung *gruppe           `json:"sammel_warnung,omitempty"`
}

type abgelegteGruppe struct {
	Meldung string `json:"meldung"`
	Route   string `json:"route,omitempty"`
	// Warnung fehlt in einem Stand, den ein Lauf vor der Zählung der Warnungen abgelegt hat; er
	// kennt nur Fehler.
	Warnung bool `json:"warnung,omitempty"`
	gruppe
}

// Fehlergruppe fasst die Zeilen einer Meldung im FehlerZeitraum zusammen.
type Fehlergruppe struct {
	Meldung string
	// Route ist das Muster der Route bei der Zeile zu einer Anfrage, sonst leer.
	Route  string
	Anzahl int
	// Erstmals ist der Beginn des ältesten Fachs im Zeitraum: die erste Zeile auf zehn Minuten
	// genau.
	Erstmals time.Time
	Zuletzt  time.Time
	// Ursache ist das Feld fehler der letzten Zeile, gekürzt; leer, wenn sie keins trug. In der
	// Sammelgruppe steht die Meldung davor.
	Ursache string
	// Zeile ist die letzte Zeile, wie das Log sie schreibt, mit allen Feldern; die Mail zeigt sie
	// ganz, damit sich der Fehler ohne Zugang zum Server beheben lässt.
	Zeile string
	// Sammel heißt, die Gruppe zählt die Zeilen über hoechstzahlGruppen; Meldung ist dann leer.
	Sammel bool
	// Warnung heißt, die Gruppe zählt Zeilen der Stufe WARN, sonst ERROR.
	Warnung bool
}

// NewFehlerstand legt einen leeren Fehlerstand an.
func NewFehlerstand() *Fehlerstand {
	return &Fehlerstand{gruppen: map[schluessel]*gruppe{}}
}

// SchalteVor legt den Fehlerstand vor innen: Jede Zeile geht an innen weiter, die der Stufen ERROR
// und WARN zählt er dabei. Ist ihre Ursache ein Abbruch (context.Canceled), geht sie als WARN
// weiter und zählt nicht: Der Browser hat die Anfrage aufgegeben, oder der Server fährt herunter;
// ein Fehler des Servers ist das nicht (docs/STIL.md, Abschnitt 11).
func (f *Fehlerstand) SchalteVor(innen slog.Handler) slog.Handler {
	return zaehler{innen: innen, stand: f}
}

// Gruppen liefert die Gruppen mit Zeilen im FehlerZeitraum vor jetzt, die jüngste zuerst.
func (f *Fehlerstand) Gruppen(jetzt time.Time) []Fehlergruppe {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.raeumeAuf(jetzt)
	gruppen := make([]Fehlergruppe, 0, len(f.gruppen)+1)
	for k, g := range f.gruppen {
		gruppen = append(gruppen, g.fehlergruppe(Fehlergruppe{Meldung: k.meldung, Route: k.route, Warnung: k.warnung}))
	}
	if f.sammel != nil {
		gruppen = append(gruppen, f.sammel.fehlergruppe(Fehlergruppe{Sammel: true}))
	}
	if f.sammelWarnung != nil {
		gruppen = append(gruppen, f.sammelWarnung.fehlergruppe(Fehlergruppe{Sammel: true, Warnung: true}))
	}
	slices.SortFunc(gruppen, func(a, b Fehlergruppe) int {
		if c := b.Zuletzt.Compare(a.Zuletzt); c != 0 {
			return c
		}
		if c := strings.Compare(a.Meldung, b.Meldung); c != 0 {
			return c
		}
		if c := strings.Compare(a.Route, b.Route); c != 0 {
			return c
		}
		// Fehler vor Warnung, damit die Reihenfolge nicht von der Map abhängt.
		switch {
		case a.Warnung == b.Warnung:
			return 0
		case b.Warnung:
			return -1
		}
		return 1
	})
	return gruppen
}

// Uebernimm rechnet den abgelegten Stand des vorigen Laufs ein; leer heißt, es gibt keinen. Erst
// nach Uebernimm legt LegeAb etwas ab, und die erste Ablage kommt gleich. Ein unlesbarer Stand
// zählt als keiner und wird ersetzt; ein zweiter Aufruf ändert nichts, sonst zählte der vorige Lauf
// doppelt.
func (f *Fehlerstand) Uebernimm(abgelegt string, jetzt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.uebernommen {
		return nil
	}
	f.uebernommen = true
	f.fassung++
	if abgelegt == "" {
		return nil
	}
	var vorige abgelegterStand
	leser := json.NewDecoder(strings.NewReader(abgelegt))
	leser.DisallowUnknownFields()
	if err := leser.Decode(&vorige); err != nil {
		return fmt.Errorf("abgelegter Fehlerstand unlesbar: %w", err)
	}
	for _, g := range vorige.Gruppen {
		rechneEin(f.gruppeFuer(schluessel{warnung: g.Warnung, meldung: g.Meldung, route: g.Route}), g.gruppe)
	}
	if vorige.Sammel != nil {
		rechneEin(f.sammelgruppe(false), *vorige.Sammel)
	}
	if vorige.SammelWarnung != nil {
		rechneEin(f.sammelgruppe(true), *vorige.SammelWarnung)
	}
	f.raeumeAuf(jetzt)
	return nil
}

// IstUebernommen sagt, ob der Stand des vorigen Laufs eingerechnet ist.
func (f *Fehlerstand) IstUebernommen() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.uebernommen
}

// LegeAb übergibt den Stand an schreibe, wenn seit dem letzten Ablegen etwas dazukam und der Stand
// des vorigen Laufs übernommen ist. Zwei Aufrufe zugleich legen nacheinander ab.
func (f *Fehlerstand) LegeAb(jetzt time.Time, schreibe func(wert string) error) error {
	f.ablageSperre.Lock()
	defer f.ablageSperre.Unlock()
	wert, fassung, ok := f.ablage(jetzt)
	if !ok {
		return nil
	}
	if err := schreibe(wert); err != nil {
		return err
	}
	f.merkeAbgelegt(fassung)
	return nil
}

// ablage liefert den Stand zum Ablegen und seine Fassung. ok ist falsch, solange der Stand des
// vorigen Laufs nicht übernommen ist oder seit dem letzten Ablegen nichts dazukam.
func (f *Fehlerstand) ablage(jetzt time.Time) (wert string, fassung uint64, ok bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.uebernommen || f.fassung == f.abgelegt {
		return "", 0, false
	}
	f.raeumeAuf(jetzt)
	stand := abgelegterStand{Gruppen: make([]abgelegteGruppe, 0, len(f.gruppen)), Sammel: f.sammel, SammelWarnung: f.sammelWarnung}
	for k, g := range f.gruppen {
		stand.Gruppen = append(stand.Gruppen, abgelegteGruppe{Meldung: k.meldung, Route: k.route, Warnung: k.warnung, gruppe: *g})
	}
	roh, err := json.Marshal(stand)
	if err != nil {
		return "", 0, false
	}
	return string(roh), f.fassung, true
}

// merkeAbgelegt merkt sich, dass die Fassung abgelegt ist; eine Zeile, die während des Ablegens
// kam, hat eine höhere Fassung und bleibt neu.
func (f *Fehlerstand) merkeAbgelegt(fassung uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.abgelegt = max(f.abgelegt, fassung)
}

func (f *Fehlerstand) trageEin(zeit time.Time, k schluessel, ursache, zeile string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, vorhanden := f.gruppen[k]; !vorhanden && len(f.gruppen) >= hoechstzahlGruppen {
		f.raeumeAuf(zeit)
	}
	g := f.gruppeFuer(k)
	if g == f.sammel || g == f.sammelWarnung {
		ursache = strings.TrimSuffix(strings.TrimSpace(k.meldung+" "+k.route)+": "+ursache, ": ")
	}
	g.Faecher[fachBeginn(zeit)]++
	if !zeit.Before(g.Zuletzt) {
		g.Zuletzt = zeit
		g.Ursache = kuerzeUrsache(ursache)
		g.Zeile = kuerzeZeile(zeile)
	}
	f.fassung++
}

// gruppeFuer liefert die Gruppe zu k und legt sie an; über hoechstzahlGruppen ist es die
// Sammelgruppe seiner Stufe.
func (f *Fehlerstand) gruppeFuer(k schluessel) *gruppe {
	if g := f.gruppen[k]; g != nil {
		return g
	}
	if len(f.gruppen) >= hoechstzahlGruppen {
		return f.sammelgruppe(k.warnung)
	}
	g := &gruppe{Faecher: map[int64]int{}}
	f.gruppen[k] = g
	return g
}

func (f *Fehlerstand) sammelgruppe(warnung bool) *gruppe {
	ziel := &f.sammel
	if warnung {
		ziel = &f.sammelWarnung
	}
	if *ziel == nil {
		*ziel = &gruppe{Faecher: map[int64]int{}}
	}
	return *ziel
}

// rechneEin zählt die Zeilen einer abgelegten Gruppe zu denen von g.
func rechneEin(g *gruppe, alt gruppe) {
	for fach, n := range alt.Faecher {
		g.Faecher[fach] += n
	}
	if alt.Zuletzt.After(g.Zuletzt) {
		g.Zuletzt = alt.Zuletzt
		g.Ursache = alt.Ursache
		g.Zeile = alt.Zeile
	}
}

// fehlergruppe ergänzt die Angaben der Gruppe um ihre Zahlen und Zeiten.
func (g *gruppe) fehlergruppe(fg Fehlergruppe) Fehlergruppe {
	fg.Anzahl, fg.Zuletzt, fg.Ursache, fg.Zeile = g.anzahl(), g.Zuletzt, g.Ursache, g.Zeile
	for fach := range g.Faecher {
		if beginn := time.Unix(fach, 0); fg.Erstmals.IsZero() || beginn.Before(fg.Erstmals) {
			fg.Erstmals = beginn
		}
	}
	return fg
}

// raeumeAuf entfernt die Fächer vor dem FehlerZeitraum und die Gruppen, die dann leer sind.
func (f *Fehlerstand) raeumeAuf(jetzt time.Time) {
	grenze := fachBeginn(jetzt.Add(-FehlerZeitraum))
	for k, g := range f.gruppen {
		if g.raeumeAuf(grenze) {
			delete(f.gruppen, k)
		}
	}
	if f.sammel != nil && f.sammel.raeumeAuf(grenze) {
		f.sammel = nil
	}
	if f.sammelWarnung != nil && f.sammelWarnung.raeumeAuf(grenze) {
		f.sammelWarnung = nil
	}
}

// raeumeAuf entfernt die Fächer vor grenze und sagt, ob die Gruppe dann leer ist.
func (g *gruppe) raeumeAuf(grenze int64) bool {
	for fach := range g.Faecher {
		if fach < grenze {
			delete(g.Faecher, fach)
		}
	}
	return len(g.Faecher) == 0
}

func (g *gruppe) anzahl() int {
	anzahl := 0
	for _, n := range g.Faecher {
		anzahl += n
	}
	return anzahl
}

func fachBeginn(zeit time.Time) int64 {
	return zeit.Truncate(fehlerFach).Unix()
}

// kuerzeZeile begrenzt die letzte Logzeile auf hoechstlaengeZeile Zeichen, ohne den Zeilenumbruch
// am Ende.
func kuerzeZeile(zeile string) string {
	zeile = strings.TrimRight(zeile, "\n")
	if utf8.RuneCountInString(zeile) <= hoechstlaengeZeile {
		return zeile
	}
	return string([]rune(zeile)[:hoechstlaengeZeile-1]) + "…"
}

// kuerzeUrsache fasst eine Ursache auf eine Zeile von höchstens hoechstlaengeUrsache Zeichen: Sie
// steht in einer Mail aus reinem Text, und den ganzen Text zeigt das Log.
func kuerzeUrsache(ursache string) string {
	ursache = strings.Join(strings.Fields(ursache), " ")
	if utf8.RuneCountInString(ursache) <= hoechstlaengeUrsache {
		return ursache
	}
	return string([]rune(ursache)[:hoechstlaengeUrsache-1]) + "…"
}

// zaehler gibt jede Zeile an innen weiter und trägt die der Stufen ERROR und WARN in den
// Fehlerstand ein. schritte merkt sich With und WithGroup in ihrer Reihenfolge, damit die
// festgehaltene Zeile dieselben Felder trägt wie die im Log.
type zaehler struct {
	innen    slog.Handler
	stand    *Fehlerstand
	schritte []func(slog.Handler) slog.Handler
}

// Enabled lässt jede Zeile der Stufen ERROR und WARN zu, auch wenn innen sie nicht schreibt:
// Gezählt wird sie trotzdem.
func (z zaehler) Enabled(ctx context.Context, stufe slog.Level) bool {
	return stufe >= slog.LevelWarn || z.innen.Enabled(ctx, stufe)
}

func (z zaehler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level < slog.LevelWarn {
		return z.innen.Handle(ctx, r)
	}
	ursache, gesetzt, route := felderAus(r)
	if gesetzt && istAbbruch(ursache) {
		r.Level = min(r.Level, slog.LevelWarn)
		return z.weiter(ctx, r)
	}
	zeit := r.Time
	if zeit.IsZero() {
		zeit = time.Now()
	}
	text := ""
	if gesetzt {
		text = ursache.Resolve().String()
	}
	z.stand.trageEin(zeit, schluessel{warnung: r.Level < slog.LevelError, meldung: r.Message, route: route}, text, z.zeileAus(ctx, r))
	return z.weiter(ctx, r)
}

// zeileAus schreibt die Zeile, wie der JSON-Handler des Logs sie schreibt, samt der Felder aus
// With und WithGroup. Scheitert das, bleibt sie leer; gezählt wird trotzdem.
func (z zaehler) zeileAus(ctx context.Context, r slog.Record) string {
	var puffer bytes.Buffer
	var h slog.Handler = slog.NewJSONHandler(&puffer, &slog.HandlerOptions{Level: slog.LevelDebug})
	for _, schritt := range z.schritte {
		h = schritt(h)
	}
	if h.Handle(ctx, r.Clone()) != nil {
		return ""
	}
	return puffer.String()
}

// weiter gibt eine Zeile an innen, wenn innen ihre Stufe schreibt.
func (z zaehler) weiter(ctx context.Context, r slog.Record) error {
	if !z.innen.Enabled(ctx, r.Level) {
		return nil
	}
	return z.innen.Handle(ctx, r)
}

func (z zaehler) WithAttrs(attrs []slog.Attr) slog.Handler {
	schritt := func(h slog.Handler) slog.Handler { return h.WithAttrs(attrs) }
	return zaehler{innen: z.innen.WithAttrs(attrs), stand: z.stand, schritte: append(slices.Clip(z.schritte), schritt)}
}

func (z zaehler) WithGroup(name string) slog.Handler {
	schritt := func(h slog.Handler) slog.Handler { return h.WithGroup(name) }
	return zaehler{innen: z.innen.WithGroup(name), stand: z.stand, schritte: append(slices.Clip(z.schritte), schritt)}
}

// felderAus liefert die Felder fehler und route einer Zeile: Unter fehler steht die Ursache
// (docs/STIL.md, Abschnitt 11), unter route das Muster der Route, wenn die Zeile zu einer Anfrage
// gehört.
func felderAus(r slog.Record) (ursache slog.Value, gesetzt bool, route string) {
	r.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "fehler":
			ursache, gesetzt = a.Value, true
		case "route":
			route = a.Value.Resolve().String()
		}
		return true
	})
	return ursache, gesetzt, route
}

// istAbbruch erkennt eine Ursache, die einen abgebrochenen Kontext einwickelt.
func istAbbruch(ursache slog.Value) bool {
	fehler, ok := ursache.Resolve().Any().(error)
	return ok && errors.Is(fehler, context.Canceled)
}
