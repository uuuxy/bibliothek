package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// ohneZeitUndZeile vergleicht Gruppen ohne Erstmals und Zeile; beide prüft
// TestFehlerstand_HaeltDieLetzteZeileFest.
var ohneZeitUndZeile = cmpopts.IgnoreFields(Fehlergruppe{}, "Erstmals", "Zeile")

var testStart = time.Date(2026, 10, 24, 10, 5, 0, 0, time.UTC)

// trageZeileEin gibt dem Fehlerstand eine Zeile mit fester Zeit, wie slog sie beim Aufruf setzt.
func trageZeileEin(t *testing.T, stand *Fehlerstand, zeit time.Time, stufe slog.Level, meldung string, attrs ...slog.Attr) {
	t.Helper()
	r := slog.NewRecord(zeit, stufe, meldung, 0)
	r.AddAttrs(attrs...)
	if err := stand.SchalteVor(slog.NewJSONHandler(&bytes.Buffer{}, nil)).Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle: %v", err)
	}
}

// Jede Zeile geht unverändert weiter; gezählt werden ERROR und WARN, je Stufe getrennt, mit der
// Ursache aus dem Feld fehler. INFO zählt nicht.
func TestFehlerstand_ZaehltFehlerUndWarnungenGetrenntUndGibtJedeZeileWeiter(t *testing.T) {
	var ausgabe bytes.Buffer
	stand := NewFehlerstand()
	log := slog.New(stand.SchalteVor(slog.NewJSONHandler(&ausgabe, nil)))

	log.Info("Anfrage", "pfad", "/api/x")
	log.Warn("Vorgabe statt Einstellung")
	log.Error("Mail nicht zugestellt", "empfaenger", "a@schule.example", "fehler", errors.New("dial tcp: i/o timeout"))
	log.With("lauf", "nacht").WithGroup("sicherung").Error("Backup: S3 upload failed", "fehler", "403 AccessDenied")

	if n := strings.Count(ausgabe.String(), "\n"); n != 4 {
		t.Errorf("Zeilen im Log = %d, erwartet 4:\n%s", n, ausgabe.String())
	}
	gruppen := stand.Gruppen(time.Now())
	if len(gruppen) != 3 {
		t.Fatalf("Gruppen = %+v, erwartet die zwei Meldungen der Stufe ERROR und die eine der Stufe WARN", gruppen)
	}
	ursachen := map[string]string{}
	for _, g := range gruppen {
		ursachen[g.Meldung] = g.Ursache
		if g.Anzahl != 1 {
			t.Errorf("Anzahl für %q = %d, erwartet 1", g.Meldung, g.Anzahl)
		}
		if istWarnung := g.Meldung == "Vorgabe statt Einstellung"; g.Warnung != istWarnung {
			t.Errorf("Gruppe %q: Warnung = %v, erwartet %v", g.Meldung, g.Warnung, istWarnung)
		}
	}
	if u := ursachen["Mail nicht zugestellt"]; u != "dial tcp: i/o timeout" {
		t.Errorf("Ursache der Mail = %q, erwartet den Text des Fehlers", u)
	}
	if u := ursachen["Backup: S3 upload failed"]; u != "403 AccessDenied" {
		t.Errorf("Ursache hinter With und WithGroup = %q, erwartet %q", u, "403 AccessDenied")
	}
}

// Gleiche Meldungen stehen zusammen: Anzahl, letzte Zeit und die Ursache der letzten Zeile, die
// jüngste Meldung zuerst.
func TestFehlerstand_FasstJeMeldungZusammen(t *testing.T) {
	stand := NewFehlerstand()
	trageZeileEin(t, stand, testStart, slog.LevelError, "A", slog.String("fehler", "erste"))
	trageZeileEin(t, stand, testStart.Add(2*time.Hour), slog.LevelError, "A", slog.String("fehler", "zweite"))
	trageZeileEin(t, stand, testStart.Add(time.Hour), slog.LevelError, "A", slog.String("fehler", "später eingetroffen, früher geschrieben"))
	trageZeileEin(t, stand, testStart.Add(3*time.Hour), slog.LevelError, "B")

	gruppen := stand.Gruppen(testStart.Add(4 * time.Hour))
	if len(gruppen) != 2 || gruppen[0].Meldung != "B" || gruppen[1].Meldung != "A" {
		t.Fatalf("Gruppen = %+v, erwartet B vor A", gruppen)
	}
	a := gruppen[1]
	if a.Anzahl != 3 || !a.Zuletzt.Equal(testStart.Add(2*time.Hour)) || a.Ursache != "zweite" {
		t.Errorf("Gruppe A = %+v, erwartet 3 Zeilen, zuletzt %v mit Ursache %q", a, testStart.Add(2*time.Hour), "zweite")
	}
	if gruppen[0].Ursache != "" {
		t.Errorf("Ursache von B = %q, erwartet leer, weil die Zeile kein Feld fehler trug", gruppen[0].Ursache)
	}
}

// Eine Zeile zählt 24 Stunden lang, höchstens ein Fach länger, und nie kürzer: Sonst verlöre eine
// Mail, die alle 24 Stunden geht, eine Zeile seit der vorigen.
func TestFehlerstand_ZaehltDenZeitraum(t *testing.T) {
	stand := NewFehlerstand()
	trageZeileEin(t, stand, testStart, slog.LevelError, "A")

	faelle := []struct {
		name  string
		jetzt time.Time
		will  int
	}{
		{"gleich danach", testStart.Add(time.Second), 1},
		{"genau 24 Stunden danach", testStart.Add(FehlerZeitraum), 1},
		{"im Fach nach 24 Stunden", testStart.Add(FehlerZeitraum + 4*time.Minute), 1},
		{"ein Fach nach 24 Stunden", testStart.Add(FehlerZeitraum + fehlerFach), 0},
	}
	for _, f := range faelle {
		anzahl := 0
		for _, g := range stand.Gruppen(f.jetzt) {
			anzahl += g.Anzahl
		}
		if anzahl != f.will {
			t.Errorf("%s: Gruppen(%v) zählt %d Zeilen, erwartet %d", f.name, f.jetzt, anzahl, f.will)
		}
	}
	wert, _, _ := ablageNachUebernahme(t, stand, testStart.Add(2*FehlerZeitraum))
	var abgelegt abgelegterStand
	if err := json.Unmarshal([]byte(wert), &abgelegt); err != nil || len(abgelegt.Gruppen) != 0 || abgelegt.Sammel != nil {
		t.Errorf("Ablage nach Ablauf = %s, erwartet einen leeren Stand", wert)
	}
}

// ablageNachUebernahme übernimmt einen leeren vorigen Stand und liefert die Ablage.
func ablageNachUebernahme(t *testing.T, stand *Fehlerstand, jetzt time.Time) (string, uint64, bool) {
	t.Helper()
	if err := stand.Uebernimm("", jetzt); err != nil {
		t.Fatalf("Uebernimm: %v", err)
	}
	return stand.ablage(jetzt)
}

// Ein Neustart verliert nichts und zählt nichts doppelt: Der neue Lauf rechnet den abgelegten
// Stand einmal ein, und abgelegt wird erst danach.
func TestFehlerstand_UeberstehtDenNeustart(t *testing.T) {
	vorher := NewFehlerstand()
	trageZeileEin(t, vorher, testStart, slog.LevelError, "A", slog.String("fehler", "alt"))
	trageZeileEin(t, vorher, testStart.Add(time.Minute), slog.LevelError, "B")
	abgelegt, _, ok := ablageNachUebernahme(t, vorher, testStart.Add(2*time.Minute))
	if !ok {
		t.Fatal("Ablage nach der Übernahme: nichts geliefert, erwartet den Stand")
	}

	nachher := NewFehlerstand()
	trageZeileEin(t, nachher, testStart.Add(time.Hour), slog.LevelError, "A", slog.String("fehler", "neu"))
	if _, _, ok := nachher.ablage(testStart.Add(time.Hour)); ok {
		t.Fatal("Ablage vor der Übernahme geliefert: Der neue Lauf überschriebe den Stand des vorigen")
	}
	for range 2 {
		if err := nachher.Uebernimm(abgelegt, testStart.Add(time.Hour)); err != nil {
			t.Fatalf("Uebernimm: %v", err)
		}
	}

	gruppen := nachher.Gruppen(testStart.Add(time.Hour))
	if len(gruppen) != 2 {
		t.Fatalf("Gruppen = %+v, erwartet A und B", gruppen)
	}
	if a := gruppen[0]; a.Meldung != "A" || a.Anzahl != 2 || a.Ursache != "neu" {
		t.Errorf("Gruppe A = %+v, erwartet 2 Zeilen (eine je Lauf) mit der Ursache der jüngsten", a)
	}
	if b := gruppen[1]; b.Meldung != "B" || b.Anzahl != 1 {
		t.Errorf("Gruppe B = %+v, erwartet die eine Zeile des vorigen Laufs", b)
	}
}

// Abgelegt wird gleich nach der Übernahme, damit der Stand den laufenden Lauf zeigt, danach nur, was
// neu ist; eine Zeile, die während des Ablegens kommt, bleibt neu.
func TestFehlerstand_AblageNurBeiNeuem(t *testing.T) {
	stand := NewFehlerstand()
	_, fassung, ok := ablageNachUebernahme(t, stand, testStart)
	if !ok {
		t.Fatal("Ablage gleich nach der Übernahme: nichts geliefert, der Stand des vorigen Laufs bliebe stehen")
	}
	stand.merkeAbgelegt(fassung)
	if _, _, ok := stand.ablage(testStart); ok {
		t.Error("Ablage ohne Zeile geliefert, erwartet nichts")
	}
	trageZeileEin(t, stand, testStart, slog.LevelError, "A")
	_, fassung, ok = stand.ablage(testStart)
	if !ok {
		t.Fatal("Ablage nach einer Zeile: nichts geliefert")
	}
	trageZeileEin(t, stand, testStart, slog.LevelError, "A")
	stand.merkeAbgelegt(fassung)
	if _, _, ok := stand.ablage(testStart); !ok {
		t.Error("Ablage nach einer Zeile während des Ablegens: nichts geliefert, die Zeile ginge verloren")
	}
	_, fassung, _ = stand.ablage(testStart)
	stand.merkeAbgelegt(fassung)
	if _, _, ok := stand.ablage(testStart); ok {
		t.Error("Ablage ohne neue Zeile geliefert, erwartet nichts")
	}
}

// Ein unlesbarer abgelegter Stand hält das Ablegen nicht auf; der nächste Stand ersetzt ihn.
func TestFehlerstand_UnlesbarerStandWirdErsetzt(t *testing.T) {
	stand := NewFehlerstand()
	trageZeileEin(t, stand, testStart, slog.LevelError, "A")
	if err := stand.Uebernimm("{kaputt", testStart); err == nil {
		t.Error("Uebernimm eines unlesbaren Stands: kein Fehler, erwartet einen")
	}
	wert, _, ok := stand.ablage(testStart)
	if !ok || !strings.Contains(wert, `"meldung":"A"`) {
		t.Errorf("Ablage = %q, %v, erwartet den Stand dieses Laufs", wert, ok)
	}
}

// Die Ursache steht in einer Zeile und hat ein Ende; den ganzen Text zeigt das Log.
func TestFehlerstand_KuerztDieUrsache(t *testing.T) {
	stand := NewFehlerstand()
	lang := "erste Zeile\n\tzweite Zeile " + strings.Repeat("ä", 300)
	trageZeileEin(t, stand, testStart, slog.LevelError, "A", slog.String("fehler", lang))

	ursache := stand.Gruppen(testStart)[0].Ursache
	if !strings.HasPrefix(ursache, "erste Zeile zweite Zeile ää") {
		t.Errorf("Ursache beginnt mit %q, erwartet die Zeilen durch ein Leerzeichen getrennt", ursache[:30])
	}
	if n := utf8.RuneCountInString(ursache); n != hoechstlaengeUrsache || !strings.HasSuffix(ursache, "…") {
		t.Errorf("Ursache hat %d Zeichen und endet mit %q, erwartet %d Zeichen mit …", n, ursache[len(ursache)-3:], hoechstlaengeUrsache)
	}
}

// Über der Höchstzahl entsteht keine Gruppe mehr, und keine Zeile geht verloren: Die vorhandenen
// Gruppen zählen weiter, die neuen Meldungen zählt die Sammelgruppe, und die Summe stimmt.
func TestFehlerstand_HoechstzahlDerGruppen(t *testing.T) {
	stand := NewFehlerstand()
	for i := range hoechstzahlGruppen + 3 {
		trageZeileEin(t, stand, testStart.Add(time.Duration(i)*time.Second), slog.LevelError, fmt.Sprintf("Meldung %d", i))
	}
	trageZeileEin(t, stand, testStart.Add(time.Hour), slog.LevelError, "Meldung 0")

	gruppen := stand.Gruppen(testStart.Add(time.Hour))
	if len(gruppen) != hoechstzahlGruppen+1 {
		t.Fatalf("Gruppen = %d, erwartet %d und die Sammelgruppe", len(gruppen), hoechstzahlGruppen)
	}
	summe := 0
	for _, g := range gruppen {
		summe += g.Anzahl
		switch {
		case g.Sammel && (g.Anzahl != 3 || g.Ursache != fmt.Sprintf("Meldung %d", hoechstzahlGruppen+2)):
			t.Errorf("Sammelgruppe = %+v, erwartet die drei Zeilen über der Höchstzahl mit der letzten Meldung", g)
		case g.Meldung == "Meldung 0" && g.Anzahl != 2:
			t.Errorf("Gruppe der ersten Meldung = %+v, erwartet 2 Zeilen: Sie zählt über der Höchstzahl weiter", g)
		}
	}
	if summe != hoechstzahlGruppen+4 {
		t.Errorf("Summe der Zeilen = %d, erwartet %d: Keine Zeile darf verloren gehen", summe, hoechstzahlGruppen+4)
	}

	wert, _, _ := ablageNachUebernahme(t, stand, testStart.Add(time.Hour))
	nachher := NewFehlerstand()
	if err := nachher.Uebernimm(wert, testStart.Add(time.Hour)); err != nil {
		t.Fatalf("Uebernimm: %v", err)
	}
	if g := nachher.Gruppen(testStart.Add(time.Hour)); len(g) != len(gruppen) {
		t.Errorf("nach dem Neustart %d Gruppen, erwartet %d mit der Sammelgruppe", len(g), len(gruppen))
	}
}

// Viele Anfragen schreiben zugleich ins Log, während die Seite liest und die Ablage schreibt.
func TestFehlerstand_Nebenlaeufig(t *testing.T) {
	stand := NewFehlerstand()
	log := slog.New(stand.SchalteVor(slog.NewJSONHandler(&sicheresSchreiben{}, nil)))
	if err := stand.Uebernimm("", time.Now()); err != nil {
		t.Fatalf("Uebernimm: %v", err)
	}
	var warten sync.WaitGroup
	for range 8 {
		warten.Add(2)
		go func() {
			defer warten.Done()
			for range 100 {
				log.Error("A", "fehler", "x")
			}
		}()
		go func() {
			defer warten.Done()
			for range 20 {
				stand.Gruppen(time.Now())
				if err := stand.LegeAb(time.Now(), func(string) error { return nil }); err != nil {
					t.Errorf("LegeAb: %v", err)
				}
			}
		}()
	}
	warten.Wait()
	if g := stand.Gruppen(time.Now()); len(g) != 1 || g[0].Anzahl != 800 {
		t.Errorf("Gruppen = %+v, erwartet eine Gruppe mit 800 Zeilen", g)
	}
}

// sicheresSchreiben nimmt Zeilen von mehreren Goroutinen an; bytes.Buffer kann das nicht.
type sicheresSchreiben struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *sicheresSchreiben) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

// Ein Abbruch ist kein Fehler des Servers: Die Zeile geht als WARN weiter und zählt nicht. Eine
// abgelaufene Frist bleibt ein Fehler, sie sagt, dass etwas zu langsam war.
func TestFehlerstand_AbbruchIstKeinFehler(t *testing.T) {
	var ausgabe bytes.Buffer
	stand := NewFehlerstand()
	log := slog.New(stand.SchalteVor(slog.NewJSONHandler(&ausgabe, nil)))

	log.Error("Fehler beim Laden der Bücherliste", "fehler", fmt.Errorf("fehler beim iterieren: %w", context.Canceled))
	log.Error("Fehler beim Laden der Bücherliste", "fehler", fmt.Errorf("lesen: %w", context.DeadlineExceeded))
	log.Warn("Vorschau abgebrochen", "fehler", context.Canceled)

	zeilen := strings.Split(strings.TrimSpace(ausgabe.String()), "\n")
	if len(zeilen) != 3 || !strings.Contains(zeilen[0], `"level":"WARN"`) || !strings.Contains(zeilen[1], `"level":"ERROR"`) ||
		!strings.Contains(zeilen[2], `"level":"WARN"`) {
		t.Errorf("Log = %q, erwartet den Abbruch als WARN, die Frist als ERROR und die Warnung als WARN", zeilen)
	}
	gruppen := stand.Gruppen(time.Now())
	if len(gruppen) != 1 || gruppen[0].Anzahl != 1 || !strings.Contains(gruppen[0].Ursache, "deadline exceeded") {
		t.Errorf("Gruppen = %+v, erwartet nur die abgelaufene Frist: Ein Abbruch zählt auch als Warnung nicht", gruppen)
	}
}

// Dieselbe Meldung als Fehler und als Warnung sind zwei Gruppen, und die Warnung übersteht den
// Neustart als Warnung. Ein Stand ohne das Feld warnung, abgelegt vor der Zählung der Warnungen,
// liest sich als Fehler.
func TestFehlerstand_WarnungUeberstehtDenNeustartAlsWarnung(t *testing.T) {
	stand := NewFehlerstand()
	trageZeileEin(t, stand, testStart, slog.LevelError, "Mail", slog.String("fehler", "kaputt"))
	trageZeileEin(t, stand, testStart.Add(time.Minute), slog.LevelWarn, "Mail", slog.String("fehler", "nochmal versucht"))
	wert, _, _ := ablageNachUebernahme(t, stand, testStart.Add(time.Hour))

	nachher := NewFehlerstand()
	if err := nachher.Uebernimm(wert, testStart.Add(time.Hour)); err != nil {
		t.Fatalf("Uebernimm: %v", err)
	}
	will := []Fehlergruppe{
		{Meldung: "Mail", Anzahl: 1, Zuletzt: testStart.Add(time.Minute), Ursache: "nochmal versucht", Warnung: true},
		{Meldung: "Mail", Anzahl: 1, Zuletzt: testStart, Ursache: "kaputt"},
	}
	if diff := cmp.Diff(will, nachher.Gruppen(testStart.Add(time.Hour)), ohneZeitUndZeile); diff != "" {
		t.Errorf("Gruppen nach dem Neustart (-will +ist):\n%s", diff)
	}

	alt := NewFehlerstand()
	if err := alt.Uebernimm(`{"gruppen":[{"meldung":"A","faecher":{"1792836000":1},"zuletzt":"2026-10-24T10:00:00Z"}]}`, testStart); err != nil {
		t.Fatalf("Uebernimm des alten Stands: %v", err)
	}
	if g := alt.Gruppen(testStart); len(g) != 1 || g[0].Warnung {
		t.Errorf("Gruppen aus dem alten Stand = %+v, erwartet die eine als Fehler", g)
	}
}

// Über der Höchstzahl zählt eine Warnung in ihrer eigenen Sammelgruppe: In der Sammelgruppe der
// Fehler machte sie die Mail kritisch.
func TestFehlerstand_WarnungUeberDerHoechstzahl(t *testing.T) {
	stand := NewFehlerstand()
	for i := range hoechstzahlGruppen {
		trageZeileEin(t, stand, testStart, slog.LevelError, fmt.Sprintf("Meldung %d", i))
	}
	trageZeileEin(t, stand, testStart.Add(time.Minute), slog.LevelWarn, "Warnung 1")
	trageZeileEin(t, stand, testStart.Add(2*time.Minute), slog.LevelWarn, "Warnung 2")

	var sammelFehler, sammelWarnung int
	for _, g := range stand.Gruppen(testStart.Add(time.Hour)) {
		switch {
		case g.Sammel && g.Warnung:
			sammelWarnung += g.Anzahl
		case g.Sammel:
			sammelFehler += g.Anzahl
		}
	}
	if sammelFehler != 0 || sammelWarnung != 2 {
		t.Errorf("Sammelgruppen: Fehler %d, Warnungen %d; erwartet 0 und 2", sammelFehler, sammelWarnung)
	}
	wert, _, _ := ablageNachUebernahme(t, stand, testStart.Add(time.Hour))
	nachher := NewFehlerstand()
	if err := nachher.Uebernimm(wert, testStart.Add(time.Hour)); err != nil {
		t.Fatalf("Uebernimm: %v", err)
	}
	for _, g := range nachher.Gruppen(testStart.Add(time.Hour)) {
		if g.Sammel && (!g.Warnung || g.Anzahl != 2) {
			t.Errorf("Sammelgruppe nach dem Neustart = %+v, erwartet die zwei Warnungen", g)
		}
	}
}

// Die Zeile zu einer Anfrage zählt je Route: Dieselbe Meldung an zwei Routen sind zwei Gruppen,
// und beide überstehen den Neustart mit ihrer Route.
func TestFehlerstand_ZaehltJeRoute(t *testing.T) {
	const meldung = "Anfrage mit Serverfehler beendet"
	stand := NewFehlerstand()
	trageZeileEin(t, stand, testStart, slog.LevelError, meldung, slog.String("route", "GET /api/a/{id}"), slog.String("fehler", "a"))
	trageZeileEin(t, stand, testStart.Add(time.Minute), slog.LevelError, meldung, slog.String("route", "GET /api/a/{id}"))
	trageZeileEin(t, stand, testStart.Add(2*time.Minute), slog.LevelError, meldung, slog.String("route", "POST /api/b"), slog.String("fehler", "b"))
	wert, _, _ := ablageNachUebernahme(t, stand, testStart.Add(time.Hour))

	nachher := NewFehlerstand()
	if err := nachher.Uebernimm(wert, testStart.Add(time.Hour)); err != nil {
		t.Fatalf("Uebernimm: %v", err)
	}
	will := []Fehlergruppe{
		{Meldung: meldung, Route: "POST /api/b", Anzahl: 1, Zuletzt: testStart.Add(2 * time.Minute), Ursache: "b"},
		{Meldung: meldung, Route: "GET /api/a/{id}", Anzahl: 2, Zuletzt: testStart.Add(time.Minute)},
	}
	if diff := cmp.Diff(will, nachher.Gruppen(testStart.Add(time.Hour)), ohneZeitUndZeile); diff != "" {
		t.Errorf("Gruppen nach dem Neustart (-will +ist):\n%s", diff)
	}
}

// Ein Stand in einer früheren Form gilt als unlesbar und wird ersetzt, statt halb gelesen zu werden;
// dazu zählt der Stand mit dem Feld beendet, dessen Vermerk pkg/laufvermerk hält.
func TestFehlerstand_FruehereFormUnlesbar(t *testing.T) {
	for _, abgelegt := range []string{
		`{"A":{"faecher":{"1792836000":1},"zuletzt":"2026-10-24T10:00:00Z"}}`,
		`{"gruppen":[],"beendet":true}`,
	} {
		if err := NewFehlerstand().Uebernimm(abgelegt, testStart); err == nil {
			t.Errorf("Uebernimm(%s) = nil, erwartet unlesbar", abgelegt)
		}
	}
}

// Gezählt wird jede Zeile der Stufen ERROR und WARN, auch wenn die Ausgabe dahinter nichts
// schreibt; sonst verwürfe slog sie, bevor der Fehlerstand sie sieht.
func TestFehlerstand_ZaehltOhneAusgabe(t *testing.T) {
	stand := NewFehlerstand()
	log := slog.New(stand.SchalteVor(slog.DiscardHandler))
	log.Error("Mail nicht zugestellt", "fehler", "x")
	log.Warn("Vorgabe statt Einstellung")
	if g := stand.Gruppen(time.Now()); len(g) != 2 || g[0].Anzahl != 1 || g[1].Anzahl != 1 {
		t.Errorf("Gruppen = %+v, erwartet die Zeile der Stufe ERROR und die der Stufe WARN", g)
	}
}

// Je Meldung hält der Fehlerstand die letzte Zeile so fest, wie das Log sie schreibt, mit den
// Feldern aus With und WithGroup und dem Stack; die Mail zeigt sie ganz, damit sich der Fehler
// ohne Zugang zum Server beheben lässt. Sie übersteht den Neustart, eine zu lange wird gekürzt,
// und Erstmals ist der Beginn des ersten Fachs.
func TestFehlerstand_HaeltDieLetzteZeileFest(t *testing.T) {
	stand := NewFehlerstand()
	log := slog.New(stand.SchalteVor(slog.DiscardHandler)).With("lauf", "nacht").WithGroup("sicherung")
	log.Error("Backup: S3 upload failed", "fehler", "erste")
	log.Error("Backup: S3 upload failed", "fehler", "403 AccessDenied", "stack", "goroutine 1 [running]:\nmain.main()")

	gruppen := stand.Gruppen(time.Now())
	if len(gruppen) != 1 {
		t.Fatalf("Gruppen = %+v, erwartet eine", gruppen)
	}
	zeile := gruppen[0].Zeile
	for _, muss := range []string{`"time":"`, `"level":"ERROR"`, `"msg":"Backup: S3 upload failed"`, `"lauf":"nacht"`,
		`"sicherung":{`, `"fehler":"403 AccessDenied"`, `goroutine 1 [running]:\nmain.main()`} {
		if !strings.Contains(zeile, muss) {
			t.Errorf("Zeile ohne %s: %s", muss, zeile)
		}
	}
	if strings.Contains(zeile, "erste") || strings.HasSuffix(zeile, "\n") {
		t.Errorf("Zeile = %q, erwartet die letzte, ohne Zeilenumbruch am Ende", zeile)
	}
	if e, z := gruppen[0].Erstmals, gruppen[0].Zuletzt; !e.Equal(e.Truncate(fehlerFach)) || e.After(z) || z.Sub(e) >= 2*fehlerFach {
		t.Errorf("Erstmals = %v, erwartet den Beginn des Fachs der ersten Zeile vor %v", e, z)
	}

	wert, _, _ := ablageNachUebernahme(t, stand, time.Now())
	nachher := NewFehlerstand()
	if err := nachher.Uebernimm(wert, time.Now()); err != nil {
		t.Fatalf("Uebernimm: %v", err)
	}
	if g := nachher.Gruppen(time.Now()); len(g) != 1 || g[0].Zeile != zeile {
		t.Errorf("Zeile nach dem Neustart = %+v, erwartet dieselbe", g)
	}

	lang := NewFehlerstand()
	slog.New(lang.SchalteVor(slog.DiscardHandler)).Error("Lang", "fehler", strings.Repeat("x", 2*hoechstlaengeZeile))
	if z := lang.Gruppen(time.Now())[0].Zeile; utf8.RuneCountInString(z) != hoechstlaengeZeile || !strings.HasSuffix(z, "…") {
		t.Errorf("lange Zeile: %d Zeichen, Ende %q; erwartet %d mit …", utf8.RuneCountInString(z), z[len(z)-5:], hoechstlaengeZeile)
	}
}

// Erstmals ist der Beginn des ältesten Fachs im Zeitraum: Ein Fach, das aus dem Zeitraum fällt,
// zählt nicht mehr.
func TestFehlerstand_ErstmalsImZeitraum(t *testing.T) {
	stand := NewFehlerstand()
	trageZeileEin(t, stand, testStart, slog.LevelError, "A")
	trageZeileEin(t, stand, testStart.Add(3*time.Hour), slog.LevelError, "A")
	if g := stand.Gruppen(testStart.Add(4 * time.Hour)); !g[0].Erstmals.Equal(testStart.Truncate(fehlerFach)) {
		t.Errorf("Erstmals = %v, erwartet %v", g[0].Erstmals, testStart.Truncate(fehlerFach))
	}
	spaeter := testStart.Add(FehlerZeitraum + 2*time.Hour)
	if g := stand.Gruppen(spaeter); len(g) != 1 || !g[0].Erstmals.Equal(testStart.Add(3*time.Hour).Truncate(fehlerFach)) {
		t.Errorf("Gruppen %v = %+v, erwartet Erstmals im Fach der zweiten Zeile", spaeter, g)
	}
}
