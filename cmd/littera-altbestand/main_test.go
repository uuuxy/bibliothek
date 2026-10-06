package main

import (
	"bytes"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/littera"
)

func TestRueckgabewert(t *testing.T) {
	tests := []struct {
		name    string
		bericht littera.Bericht
		want    int
	}{
		{
			name: "Alles OK",
			bericht: littera.Bericht{
				Bestand:     littera.BestandBericht{AbgleichOK: true},
				Schlagworte: littera.SchlagwortBericht{AbgleichOK: true},
				Personen:    littera.PersonenBericht{AbgleichOK: true},
				Ausleihen:   littera.AusleihBericht{AbgleichOK: true},
				Fehler:      0,
				Abbruch:     nil,
			},
			want: exitOK,
		},
		{
			name: "Mit Abbruch",
			bericht: littera.Bericht{
				Abbruch: errors.New("ein fehler"),
			},
			want: exitAbgebrochen,
		},
		{
			name: "Mit Fehler (unvollständig)",
			bericht: littera.Bericht{
				Bestand:   littera.BestandBericht{AbgleichOK: true},
				Personen:  littera.PersonenBericht{AbgleichOK: true},
				Ausleihen: littera.AusleihBericht{AbgleichOK: true},
				Fehler:    1,
			},
			want: exitUnvollstaendig,
		},
		{
			name: "Abgleich fehlgeschlagen (unvollständig)",
			bericht: littera.Bericht{
				Bestand:   littera.BestandBericht{AbgleichOK: false},
				Personen:  littera.PersonenBericht{AbgleichOK: true},
				Ausleihen: littera.AusleihBericht{AbgleichOK: true},
				Fehler:    0,
			},
			want: exitUnvollstaendig,
		},
		{
			// Seit dem 30.09.2026 (docs/OFFEN.md 4.20): Fehlen Schlagworte in der Datenbank, ist
			// der Lauf nicht vollständig, auch wenn Bestand und Personen stimmen.
			name: "Abgleich der Schlagworte fehlgeschlagen (unvollständig)",
			bericht: littera.Bericht{
				Bestand:     littera.BestandBericht{AbgleichOK: true},
				Schlagworte: littera.SchlagwortBericht{AbgleichOK: false},
				Personen:    littera.PersonenBericht{AbgleichOK: true},
				Ausleihen:   littera.AusleihBericht{AbgleichOK: true},
			},
			want: exitUnvollstaendig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rueckgabewert(tt.bericht); got != tt.want {
				t.Errorf("rueckgabewert() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOptionen(t *testing.T) {
	s := schalter{
		batch:         100,
		barcodes:      "neu",
		lehrerInaktiv: true,
		schuljahrEnde: 2025,
	}

	opt := optionen(s)

	if opt.BatchGroesse != 100 {
		t.Errorf("BatchGroesse = %v, want 100", opt.BatchGroesse)
	}
	if string(opt.Barcodes) != "neu" {
		t.Errorf("Barcodes = %v, want 'neu'", opt.Barcodes)
	}
	if !opt.LehrerInaktiv {
		t.Error("LehrerInaktiv = false, want true")
	}
	if opt.SchuljahrEnde != 2025 {
		t.Errorf("SchuljahrEnde = %v, want 2025", opt.SchuljahrEnde)
	}
}

func TestAbgleich(t *testing.T) {
	var buf bytes.Buffer
	defer log.SetOutput(log.Writer())
	log.SetOutput(&buf)

	abgleich(true, "100 neu")
	if !strings.Contains(buf.String(), "✓ Abgleich: 100 neu") {
		t.Errorf("Erwartete Erfolgsmeldung nicht gefunden, log: %s", buf.String())
	}

	buf.Reset()
	abgleich(false, "50 neu")
	if !strings.Contains(buf.String(), "⚠ ABGLEICH FEHLGESCHLAGEN: 50 neu") {
		t.Errorf("Erwartete Fehlermeldung nicht gefunden, log: %s", buf.String())
	}
}

func TestTrockenlauf(t *testing.T) {
	var buf bytes.Buffer
	defer log.SetOutput(log.Writer())
	log.SetOutput(&buf)

	ab := &littera.Altbestand{
		Titel:      []littera.Titel{{}, {}},
		Signaturen: map[string]string{"1": "A", "2": "B"},
		Leser: []littera.Leser{
			{Art: littera.ArtSchueler},
			{Art: littera.ArtLehrkraft},
			{Art: littera.ArtFachbereich},
		},
		Ausleihen: []littera.Ausleihe{{Frist: time.Now()}},
	}

	trockenlauf(ab)

	out := buf.String()
	if !strings.Contains(out, "TROCKENLAUF: es wird nichts geschrieben") {
		t.Errorf("Trockenlauf Meldung fehlt")
	}
	if !strings.Contains(out, "Leser: 1 Schüler, 0 abgegangen; Kollegium: 1 Lehrkräfte, 0 LiV, 0 Praktikum, "+
		"0 Sekretariat, 0 U-plus, 1 Fachbereich; ohne Zuordnung: 0") {
		t.Errorf("Leser Statistik fehlt oder falsch, log: %s", out)
	}
}

// ohneZuordnung ist ein Export mit einem Leser der Gruppe „Undefinierte Untergruppe" und seiner
// Ausleihe — so, wie die Sicherung von 2010 fünf davon trägt.
func ohneZuordnung() *littera.Altbestand {
	return &littera.Altbestand{
		Leser: []littera.Leser{
			{ID: "1", Vorname: "Geheim", Nachname: "Person", Art: littera.ArtSchueler, Klasse: "07H1"},
			{ID: "2", Vorname: "Auch", Nachname: "Geheim", Art: littera.ArtUnbekannt, Klasse: "UNDEF",
				Gruppe: "Undefinierte Untergruppe", GruppeNr: "1"},
		},
		Ausleihen: []littera.Ausleihe{{ID: "A", LeserID: "2"}},
	}
}

// TestLeserOhneZuordnungHaeltVorDemErstenSchreibenAn: Entscheidung vom 28.09.2026 — eine Gruppe
// ohne Zuordnung hält den Lauf an, bevor irgendetwas geschrieben ist. Belegt am Weg selbst: Der
// Lauf endet mit 1, legt kein Protokoll an und versucht keine Verbindung (die Adresse wäre nicht
// auflösbar, die Meldung „Datenbank" stünde im Log). Ausgegeben werden Gruppe und Zahlen, kein Name.
func TestLeserOhneZuordnungHaeltVorDemErstenSchreibenAn(t *testing.T) {
	t.Chdir(t.TempDir())
	var buf bytes.Buffer
	defer log.SetOutput(log.Writer())
	log.SetOutput(&buf)

	s := schalter{personen: true, ausleihen: true, bestand: true, dbURL: "postgres://nicht-erreichbar.invalid/x"}
	if code := uebertrage(s, ohneZuordnung()); code != exitAbgebrochen {
		t.Errorf("Rückgabe %d, erwartet %d (abgebrochen)", code, exitAbgebrochen)
	}
	out := buf.String()
	if !strings.Contains(out, "Lesergruppe 1 „Undefinierte Untergruppe“ (UNDEF): 1 Person, 1 Ausleihe") {
		t.Errorf("die Gruppe muss mit Zahlen genannt werden, log: %s", out)
	}
	for _, verboten := range []string{"Geheim", "Datenbank"} {
		if strings.Contains(out, verboten) {
			t.Errorf("%q hat im Log nichts zu suchen — Name oder Verbindungsversuch, log: %s", verboten, out)
		}
	}
	if _, err := os.Stat(protokollPfad); !os.IsNotExist(err) {
		t.Errorf("vor dem Halt darf kein Protokoll entstehen (%s): %v", protokollPfad, err)
	}
}

// TestTrockenlaufMeldetFehlendeZuordnung: Der Trockenlauf sagt voraus, ob der echte Lauf mit
// denselben Schaltern anhielte — mit -personen endet er dann mit 1. Ohne -personen betrifft die
// Gruppe nichts, der Lauf schreibt keine Person.
func TestTrockenlaufMeldetFehlendeZuordnung(t *testing.T) {
	defer log.SetOutput(log.Writer())
	log.SetOutput(&bytes.Buffer{})

	if code := zuordnungGeprueft(ohneZuordnung(), true); code != exitAbgebrochen {
		t.Errorf("mit -personen: Rückgabe %d, erwartet %d", code, exitAbgebrochen)
	}
	if code := zuordnungGeprueft(ohneZuordnung(), false); code != exitOK {
		t.Errorf("ohne -personen: Rückgabe %d, erwartet %d", code, exitOK)
	}
}

// TestTrockenlaufListetStandorte: Der Trockenlauf nennt jeden Wert, der als Standort ankäme,
// mit seiner Zahl — nur an dieser Liste fällt ein Verfasser ohne Komma auf.
func TestTrockenlaufListetStandorte(t *testing.T) {
	var buf bytes.Buffer
	defer log.SetOutput(log.Writer())
	log.SetOutput(&buf)

	trockenlauf(&littera.Altbestand{
		Signaturen:       map[string]string{},
		Standortvermerke: map[string][]string{"1": {"LMF"}, "2": {"Louise Carleton-Gertsch"}},
		Exemplare: []littera.Exemplar{
			{ID: "a", TitelID: "1"}, {ID: "b", TitelID: "1", Sonderstandort: "Lehrerschrank"},
			{ID: "c", TitelID: "2"},
		},
	})

	out := buf.String()
	for _, zeile := range []string{
		"1 Exemplare mit eigenem Sonderstandort, 2 über den Vermerk am Titel; 1 Titel sind nur nach dem Vermerk Lernmittel",
		"Vermerke am Titel (2 Werte",
		"    1  LMF", "    1  Louise Carleton-Gertsch",
		"Sonderstandorte der Exemplare (1 Werte",
		"    1  Lehrerschrank",
	} {
		if !strings.Contains(out, zeile) {
			t.Errorf("im Trockenlauf fehlt %q, log: %s", zeile, out)
		}
	}
}

// TestKeinStandortSchalter: Der Schalter darf mehrfach stehen, jeder Wert geht in die Regel.
func TestKeinStandortSchalter(t *testing.T) {
	var s schalter
	for _, wert := range []string{"Louise Carleton-Gertsch", "Hörbuch"} {
		if err := s.keinStandort.Set(wert); err != nil {
			t.Fatalf("Set(%q): %v", wert, err)
		}
	}
	regel := s.standortregel()
	if len(regel.KeinStandort) != 2 || !regel.KeinStandort["Hörbuch"] || !regel.KeinStandort["Louise Carleton-Gertsch"] {
		t.Errorf("Regel aus zwei Schaltern: %v", regel.KeinStandort)
	}
	if got := s.keinStandort.String(); got != "Louise Carleton-Gertsch, Hörbuch" {
		t.Errorf("String() = %q", got)
	}
}
