package main

import (
	"bytes"
	"errors"
	"log"
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
				Bestand:   littera.BestandBericht{AbgleichOK: true},
				Personen:  littera.PersonenBericht{AbgleichOK: true},
				Ausleihen: littera.AusleihBericht{AbgleichOK: true},
				Fehler:    0,
				Abbruch:   nil,
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
		Titel: []littera.Titel{{}, {}},
		Signaturen: map[string]string{"1": "A", "2": "B"},
		Leser: []littera.Leser{
			{Art: littera.ArtSchueler},
			{Art: littera.ArtLehrkraft},
		},
		Ausleihen: []littera.Ausleihe{{Frist: time.Now()}},
	}

	trockenlauf(ab)

	out := buf.String()
	if !strings.Contains(out, "TROCKENLAUF: es wird nichts geschrieben") {
		t.Errorf("Trockenlauf Meldung fehlt")
	}
	if !strings.Contains(out, "Leser: 1 Schüler, 1 Lehrkräfte") {
		t.Errorf("Leser Statistik fehlt oder falsch, log: %s", out)
	}
}
