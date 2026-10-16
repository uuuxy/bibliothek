package littera

import (
	"bibliothek/internal/uebernahme"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testSchreiberOhneDB(t *testing.T, anpassen func(*Optionen)) (*Schreiber, func() string) {
	t.Helper()
	pfad := filepath.Join(t.TempDir(), "littera_import.log")
	prot, err := uebernahme.NewProtokoll(pfad, "littera_id")
	if err != nil {
		t.Fatalf("Protokoll: %v", err)
	}
	t.Cleanup(prot.Schliesse)

	opt := OptionenVorgabe(time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC))
	if anpassen != nil {
		anpassen(&opt)
	}

	lies := func() string {
		if err := prot.Leere(); err != nil {
			t.Fatalf("Protokoll schreiben: %v", err)
		}
		b, err := os.ReadFile(pfad)
		if err != nil {
			t.Fatalf("Protokoll lesen: %v", err)
		}
		return string(b)
	}
	// Pool ist nil
	return NewSchreiber(nil, prot, opt), lies
}

func TestFristVorAusgabeWirdGemeldet(t *testing.T) {
	s, protokoll := testSchreiberOhneDB(t, nil)

	start := time.Date(2010, 9, 1, 8, 0, 0, 0, time.UTC)
	a := Ausleihe{ID: "A1", ExemplarID: "E1", LeserID: "S1",
		AusgeliehenAm: start.AddDate(0, 0, 10), Frist: start}

	l := &ausleihlauf{
		s:       s,
		bericht: &AusleihBericht{},
	}
	l.meldeFristVorAusgabe(a)

	if l.bericht.FristVorAusgabe != 1 {
		t.Errorf("l.bericht.FristVorAusgabe = %d, erwartet FristVorAusgabe = 1", l.bericht.FristVorAusgabe)
	}

	text := protokoll()
	if !strings.Contains(text, "vor dem Verleihdatum") {
		t.Errorf("Warnung nicht protokolliert: %s", text)
	}
}

func TestKeineFristWirdDurchVerleihdatumErsetzt(t *testing.T) {
	s, protokoll := testSchreiberOhneDB(t, nil)

	start := time.Date(2010, 9, 1, 8, 0, 0, 0, time.UTC)
	// Keine Frist
	a := Ausleihe{ID: "A1", ExemplarID: "E1", LeserID: "S1", AusgeliehenAm: start}

	l := &ausleihlauf{
		s:       s,
		bericht: &AusleihBericht{},
	}
	frist := l.frist(a)

	if !frist.Equal(start) {
		t.Errorf("l.frist(a) = %v, erwartet Frist = Verleihdatum", frist)
	}

	text := protokoll()
	if !strings.Contains(text, "keine Rückgabefrist im Altbestand") {
		t.Errorf("Warnung nicht protokolliert: %s", text)
	}
}

func TestBuchbarIstOffen(t *testing.T) {
	b1 := buchbar{RueckgabeAm: nil}
	if !b1.istOffen() {
		t.Errorf("buchbar{RueckgabeAm: nil}.istOffen() = false, erwartet true")
	}
	jetzt := time.Now()
	b2 := buchbar{RueckgabeAm: &jetzt}
	if b2.istOffen() {
		t.Errorf("buchbar{RueckgabeAm: &jetzt}.istOffen() = true, erwartet false")
	}
}

func TestFristRegulaerBleibtErhalten(t *testing.T) {
	s, _ := testSchreiberOhneDB(t, nil)

	start := time.Date(2010, 9, 1, 8, 0, 0, 0, time.UTC)
	fristVor := start.AddDate(0, 0, 10)
	a := Ausleihe{ID: "A1", ExemplarID: "E1", LeserID: "S1", AusgeliehenAm: start, Frist: fristVor}

	l := &ausleihlauf{
		s:       s,
		bericht: &AusleihBericht{},
	}
	fristNach := l.frist(a)

	if !fristNach.Equal(fristVor) {
		t.Errorf("l.frist(a) = %v, erwartet Frist = %v", fristNach, fristVor)
	}
}

func TestHatBeideEnden(t *testing.T) {
	s, protokoll := testSchreiberOhneDB(t, nil)

	l := &ausleihlauf{
		s:         s,
		bericht:   &AusleihBericht{},
		exemplare: map[string]string{"E1": "UUID1"},
		entleiher: map[string]Zielleser{"L1": {LeserID: "S1", IstSchueler: true}},
	}

	// Gutfall
	aGut := Ausleihe{ID: "A1", ExemplarID: "E1", LeserID: "L1"}
	if !l.hatBeideEnden(aGut) {
		t.Errorf("l.hatBeideEnden(aGut) = false, erwartet true")
	}

	// Ohne Exemplar
	aOhneEx := Ausleihe{ID: "A2", ExemplarID: "E2", LeserID: "L1"}
	if l.hatBeideEnden(aOhneEx) {
		t.Errorf("l.hatBeideEnden(ohne Exemplar) = true, erwartet false")
	}
	if l.bericht.OhneExemplar != 1 {
		t.Errorf("l.bericht.OhneExemplar = %d, erwartet OhneExemplar = 1", l.bericht.OhneExemplar)
	}
	if !strings.Contains(protokoll(), "das ausgeliehene Exemplar steht nicht im übernommenen Bestand") {
		t.Errorf("Protokoll nennt das fehlende Exemplar nicht")
	}

	// Ohne Entleiher
	aOhneEnt := Ausleihe{ID: "A3", ExemplarID: "E1", LeserID: "L2"}
	if l.hatBeideEnden(aOhneEnt) {
		t.Errorf("l.hatBeideEnden(ohne Entleiher) = true, erwartet false")
	}
	if l.bericht.OhneLeser != 1 {
		t.Errorf("l.bericht.OhneLeser = %d, erwartet OhneLeser = 1", l.bericht.OhneLeser)
	}
	if !strings.Contains(protokoll(), "der Entleiher wurde nicht übernommen") {
		t.Errorf("Protokoll nennt den fehlenden Entleiher nicht")
	}
}

func TestRueckgabeAm(t *testing.T) {
	s, protokoll := testSchreiberOhneDB(t, nil)
	l := &ausleihlauf{
		s:       s,
		bericht: &AusleihBericht{},
	}

	start := time.Date(2010, 9, 1, 8, 0, 0, 0, time.UTC)

	// Nicht zurückgegeben
	a1 := Ausleihe{ID: "A1", Zurueckgegeben: false}
	if r1 := l.rueckgabeAm(a1); r1 != nil {
		t.Errorf("l.rueckgabeAm(a1) = %v, erwartet nil für nicht zurückgegebene Ausleihe", r1)
	}

	// Zurückgegeben, aber Datum fehlt
	a2 := Ausleihe{ID: "A2", Zurueckgegeben: true}
	if r2 := l.rueckgabeAm(a2); r2 != nil {
		t.Errorf("l.rueckgabeAm(a2) = %v, erwartet nil für Ausleihe ohne Rückgabedatum", r2)
	}

	// Rückgabedatum liegt vor Verleihdatum
	a3 := Ausleihe{ID: "A3", Zurueckgegeben: true, AusgeliehenAm: start.AddDate(0, 0, 10), RueckgabeAm: start}
	if r3 := l.rueckgabeAm(a3); r3 != nil {
		t.Errorf("l.rueckgabeAm(a3) = %v, erwartet nil für Ausleihe mit unmöglichem Rückgabedatum", r3)
	}
	if !strings.Contains(protokoll(), "Rückgabedatum liegt vor dem Verleihdatum") {
		t.Errorf("Protokoll nennt das unmögliche Rückgabedatum nicht")
	}

	// Erfolgreiche Rückgabe
	a4 := Ausleihe{ID: "A4", Zurueckgegeben: true, AusgeliehenAm: start, RueckgabeAm: start.AddDate(0, 0, 10)}
	if r4 := l.rueckgabeAm(a4); r4 == nil || !r4.Equal(a4.RueckgabeAm) {
		t.Errorf("l.rueckgabeAm(a4) = %v, erwartet %v", r4, a4.RueckgabeAm)
	}
}

func TestAussortieren(t *testing.T) {
	s, protokoll := testSchreiberOhneDB(t, nil)

	l := &ausleihlauf{
		s:       s,
		bericht: &AusleihBericht{},
		exemplare: map[string]string{
			"E1": "UUID1",
			"E2": "UUID2",
		},
		entleiher: map[string]Zielleser{
			"L1": {LeserID: "S1", IstSchueler: true},
			"L2": {LeserID: "S2", IstSchueler: true},
		},
	}

	start := time.Date(2010, 9, 1, 8, 0, 0, 0, time.UTC)

	ausleihen := []Ausleihe{
		// Wird aussortiert, da Entleiher fehlt
		{ID: "A1", ExemplarID: "E1", LeserID: "L3", AusgeliehenAm: start, Frist: start.AddDate(0, 0, 10)},
		// Wird übernommen
		{ID: "A2", ExemplarID: "E1", LeserID: "L1", AusgeliehenAm: start.AddDate(0, 0, 1), Frist: start.AddDate(0, 0, 10)},
		// Wird aussortiert als Doppelbelegung (älter als A2)
		{ID: "A3", ExemplarID: "E1", LeserID: "L2", AusgeliehenAm: start, Frist: start.AddDate(0, 0, 10)},
		// Wird aussortiert, da Exemplar fehlt
		{ID: "A4", ExemplarID: "E3", LeserID: "L2", AusgeliehenAm: start, Frist: start.AddDate(0, 0, 10)},
	}

	ergebnis := l.aussortieren(ausleihen)

	if len(ergebnis) != 1 {
		t.Errorf("len(ergebnis) = %d, erwartet 1 buchbare Ausleihe", len(ergebnis))
	} else if ergebnis[0].ID != "A2" {
		t.Errorf("ergebnis[0].ID = %s, erwartet Ausleihe A2", ergebnis[0].ID)
	}

	if l.bericht.OhneLeser != 1 {
		t.Errorf("l.bericht.OhneLeser = %d, erwartet OhneLeser = 1", l.bericht.OhneLeser)
	}
	if l.bericht.OhneExemplar != 1 {
		t.Errorf("l.bericht.OhneExemplar = %d, erwartet OhneExemplar = 1", l.bericht.OhneExemplar)
	}
	if l.bericht.Doppelbelegung != 1 {
		t.Errorf("l.bericht.Doppelbelegung = %d, erwartet Doppelbelegung = 1", l.bericht.Doppelbelegung)
	}

	protokollText := protokoll()
	erwarteteMeldungen := []string{
		"der Entleiher wurde nicht übernommen",
		"zweite offene Ausleihe desselben Exemplars",
		"das ausgeliehene Exemplar steht nicht im übernommenen Bestand",
	}

	for _, meldung := range erwarteteMeldungen {
		if !strings.Contains(protokollText, meldung) {
			t.Errorf("Erwartete Meldung nicht gefunden: %s", meldung)
		}
	}
}
