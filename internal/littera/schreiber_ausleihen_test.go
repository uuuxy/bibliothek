package littera

import (
	"strings"
	"testing"
	"time"
    "path/filepath"
    "os"
    "bibliothek/internal/uebernahme"
)

func testSchreiberOhneDB(t *testing.T, anpassen func(*Optionen)) (*Schreiber, func() string) {
	t.Helper()
	pfad := filepath.Join(t.TempDir(), "littera_import.log")
	prot, err := uebernahme.NeuesProtokoll(pfad, "littera_id")
	if err != nil {
		t.Fatalf("Protokoll: %v", err)
	}
	t.Cleanup(prot.Schliessen)

	opt := StandardOptionen(time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC))
	if anpassen != nil {
		anpassen(&opt)
	}

	lies := func() string {
		if err := prot.Leeren(); err != nil {
			t.Fatalf("Protokoll schreiben: %v", err)
		}
		b, err := os.ReadFile(pfad)
		if err != nil {
			t.Fatalf("Protokoll lesen: %v", err)
		}
		return string(b)
	}
	// Pool ist nil
	return NeuerSchreiber(nil, prot, opt), lies
}

func TestFristVorAusgabeWirdGemeldet(t *testing.T) {
	s, protokoll := testSchreiberOhneDB(t, nil)

	start := time.Date(2010, 9, 1, 8, 0, 0, 0, time.UTC)
	a := Ausleihe{ID: "A1", ExemplarID: "E1", LeserID: "S1",
		AusgeliehenAm: start.AddDate(0, 0, 10), Frist: start}

	l := &ausleihlauf{
		s: s,
		bericht: &AusleihBericht{},
	}
	l.meldeFristVorAusgabe(a)

	if l.bericht.FristVorAusgabe != 1 {
		t.Errorf("erwartet FristVorAusgabe = 1, bekommen %d", l.bericht.FristVorAusgabe)
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
		s: s,
		bericht: &AusleihBericht{},
	}
	frist := l.frist(a)

	if !frist.Equal(start) {
		t.Errorf("erwartet Frist = Verleihdatum, bekommen %v", frist)
	}

	text := protokoll()
	if !strings.Contains(text, "keine Rückgabefrist im Altbestand") {
		t.Errorf("Warnung nicht protokolliert: %s", text)
	}
}

func TestBuchbarBleibtOffen(t *testing.T) {
    b1 := buchbar{RueckgabeAm: nil}
    if !b1.bleibtOffen() {
        t.Errorf("Erwartet, dass Buchung ohne RueckgabeAm offen bleibt")
    }
    jetzt := time.Now()
    b2 := buchbar{RueckgabeAm: &jetzt}
    if b2.bleibtOffen() {
        t.Errorf("Erwartet, dass Buchung mit RueckgabeAm nicht offen bleibt")
    }
}

func TestFristRegulaerBleibtErhalten(t *testing.T) {
	s, _ := testSchreiberOhneDB(t, nil)

	start := time.Date(2010, 9, 1, 8, 0, 0, 0, time.UTC)
	fristVor := start.AddDate(0, 0, 10)
	a := Ausleihe{ID: "A1", ExemplarID: "E1", LeserID: "S1", AusgeliehenAm: start, Frist: fristVor}

	l := &ausleihlauf{
		s: s,
		bericht: &AusleihBericht{},
	}
	fristNach := l.frist(a)

	if !fristNach.Equal(fristVor) {
		t.Errorf("erwartet Frist = %v, bekommen %v", fristVor, fristNach)
	}
}

func TestBeideEndenGefunden(t *testing.T) {
	s, protokoll := testSchreiberOhneDB(t, nil)

	l := &ausleihlauf{
		s: s,
		bericht: &AusleihBericht{},
		exemplare: map[string]string{"E1": "UUID1"},
		entleiher: map[string]Entleiher{"L1": {SchuelerID: "S1"}},
	}

	// Gutfall
	aGut := Ausleihe{ID: "A1", ExemplarID: "E1", LeserID: "L1"}
	if !l.beideEndenGefunden(aGut) {
		t.Errorf("Erwartet true, wenn beide Enden gefunden wurden")
	}

	// Ohne Exemplar
	aOhneEx := Ausleihe{ID: "A2", ExemplarID: "E2", LeserID: "L1"}
	if l.beideEndenGefunden(aOhneEx) {
		t.Errorf("Erwartet false, wenn Exemplar fehlt")
	}
	if l.bericht.OhneExemplar != 1 {
		t.Errorf("Erwartet OhneExemplar = 1, bekommen %d", l.bericht.OhneExemplar)
	}
	if !strings.Contains(protokoll(), "das ausgeliehene Exemplar steht nicht im übernommenen Bestand") {
		t.Errorf("Erwartet Protokollierung des fehlenden Exemplars")
	}

	// Ohne Entleiher
	aOhneEnt := Ausleihe{ID: "A3", ExemplarID: "E1", LeserID: "L2"}
	if l.beideEndenGefunden(aOhneEnt) {
		t.Errorf("Erwartet false, wenn Entleiher fehlt")
	}
	if l.bericht.OhneEntleiher != 1 {
		t.Errorf("Erwartet OhneEntleiher = 1, bekommen %d", l.bericht.OhneEntleiher)
	}
	if !strings.Contains(protokoll(), "der Entleiher wurde nicht übernommen") {
		t.Errorf("Erwartet Protokollierung des fehlenden Entleihers")
	}
}

func TestRueckgabeAm(t *testing.T) {
	s, protokoll := testSchreiberOhneDB(t, nil)
	l := &ausleihlauf{
		s: s,
		bericht: &AusleihBericht{},
	}

	start := time.Date(2010, 9, 1, 8, 0, 0, 0, time.UTC)

	// Nicht zurückgegeben
	a1 := Ausleihe{ID: "A1", Zurueckgegeben: false}
	if r1 := l.rueckgabeAm(a1); r1 != nil {
		t.Errorf("Erwartet nil für nicht zurückgegebene Ausleihe, bekommen %v", r1)
	}

	// Zurückgegeben, aber Datum fehlt
	a2 := Ausleihe{ID: "A2", Zurueckgegeben: true}
	if r2 := l.rueckgabeAm(a2); r2 != nil {
		t.Errorf("Erwartet nil für Ausleihe ohne Rückgabedatum, bekommen %v", r2)
	}

	// Rückgabedatum liegt vor Verleihdatum
	a3 := Ausleihe{ID: "A3", Zurueckgegeben: true, AusgeliehenAm: start.AddDate(0, 0, 10), RueckgabeAm: start}
	if r3 := l.rueckgabeAm(a3); r3 != nil {
		t.Errorf("Erwartet nil für Ausleihe mit unmöglichem Rückgabedatum, bekommen %v", r3)
	}
	if !strings.Contains(protokoll(), "Rückgabedatum liegt vor dem Verleihdatum") {
		t.Errorf("Erwartet Protokollierung des unmöglichen Rückgabedatums")
	}

	// Erfolgreiche Rückgabe
	a4 := Ausleihe{ID: "A4", Zurueckgegeben: true, AusgeliehenAm: start, RueckgabeAm: start.AddDate(0, 0, 10)}
	if r4 := l.rueckgabeAm(a4); r4 == nil || !r4.Equal(a4.RueckgabeAm) {
		t.Errorf("Erwartet %v, bekommen %v", a4.RueckgabeAm, r4)
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
		entleiher: map[string]Entleiher{
			"L1": {SchuelerID: "S1"},
			"L2": {SchuelerID: "S2"},
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
		t.Errorf("Erwartet 1 buchbare Ausleihe, bekommen %d", len(ergebnis))
	} else if ergebnis[0].ID != "A2" {
		t.Errorf("Erwartet Ausleihe A2, bekommen %s", ergebnis[0].ID)
	}

	if l.bericht.OhneEntleiher != 1 {
		t.Errorf("Erwartet OhneEntleiher = 1, bekommen %d", l.bericht.OhneEntleiher)
	}
	if l.bericht.OhneExemplar != 1 {
		t.Errorf("Erwartet OhneExemplar = 1, bekommen %d", l.bericht.OhneExemplar)
	}
	if l.bericht.Doppelbelegung != 1 {
		t.Errorf("Erwartet Doppelbelegung = 1, bekommen %d", l.bericht.Doppelbelegung)
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
