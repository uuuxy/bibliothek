package jobs

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"bibliothek/pkg/logger"

	"github.com/jackc/pgx/v5"
)

// fehlerAblageAttrappe hält den abgelegten Stand im Speicher; Lesen und Schreiben lassen sich
// scheitern lassen.
type fehlerAblageAttrappe struct {
	wert          string
	vorhanden     bool
	ladeFehler    error
	schreibFehler error
	geschrieben   int
}

func (a *fehlerAblageAttrappe) LadeEinstellungswert(_ context.Context, _ string) (string, error) {
	if a.ladeFehler != nil {
		return "", a.ladeFehler
	}
	if !a.vorhanden {
		return "", pgx.ErrNoRows
	}
	return a.wert, nil
}

func (a *fehlerAblageAttrappe) SchreibeEinstellungswert(_ context.Context, _, wert string) error {
	if a.schreibFehler != nil {
		return a.schreibFehler
	}
	a.wert, a.vorhanden = wert, true
	a.geschrieben++
	return nil
}

// standMitFehler liefert einen Fehlerstand mit einer Zeile der Stufe ERROR je Meldung.
func standMitFehler(t *testing.T, meldungen ...string) *logger.Fehlerstand {
	t.Helper()
	stand := logger.NewFehlerstand()
	zaehler := stand.SchalteVor(slog.DiscardHandler)
	for _, m := range meldungen {
		zeile := slog.NewRecord(time.Now(), slog.LevelError, m, 0)
		zeile.AddAttrs(slog.String("fehler", "x"))
		if err := zaehler.Handle(context.Background(), zeile); err != nil {
			t.Fatalf("Handle: %v", err)
		}
	}
	return stand
}

// Ist der abgelegte Stand nicht lesbar, legt der neue Lauf nichts ab: Er überschriebe sonst die
// Zeilen des vorigen. Sobald das Lesen gelingt, stehen beide Läufe im Stand.
func TestFehlerstandAblage_SchreibtNichtVorDerUebernahme(t *testing.T) {
	ctx := context.Background()
	ablage := &fehlerAblageAttrappe{ladeFehler: errors.New("verbindung abgelehnt")}
	vorher := standMitFehler(t, "Alt")
	if _, err := vorher.Uebernimm("", time.Now()); err != nil {
		t.Fatalf("Uebernimm: %v", err)
	}
	vorher.Beende()
	if err := vorher.LegeAb(time.Now(), func(wert string) error { ablage.wert = wert; return nil }); err != nil {
		t.Fatalf("LegeAb: %v", err)
	}
	ablage.vorhanden = true

	stand := standMitFehler(t, "Neu")
	legeFehlerstandAb(ctx, ablage, stand, time.Now())
	if ablage.geschrieben != 0 {
		t.Fatalf("legeFehlerstandAb hat %d-mal geschrieben, obwohl das Lesen scheiterte, erwartet 0", ablage.geschrieben)
	}

	ablage.ladeFehler = nil
	legeFehlerstandAb(ctx, ablage, stand, time.Now())
	if ablage.geschrieben != 1 || !strings.Contains(ablage.wert, `"Alt"`) || !strings.Contains(ablage.wert, `"Neu"`) {
		t.Errorf("abgelegt = %d-mal %s, erwartet einmal mit beiden Meldungen", ablage.geschrieben, ablage.wert)
	}
}

// Scheitert das Schreiben, versucht es der nächste Takt; ohne neue Zeile schreibt er nicht.
func TestFehlerstandAblage_WiederholtNurWasFehlt(t *testing.T) {
	ctx := context.Background()
	ablage := &fehlerAblageAttrappe{schreibFehler: errors.New("verbindung abgelehnt")}
	stand := standMitFehler(t, "A")

	legeFehlerstandAb(ctx, ablage, stand, time.Now())
	ablage.schreibFehler = nil
	legeFehlerstandAb(ctx, ablage, stand, time.Now())
	legeFehlerstandAb(ctx, ablage, stand, time.Now())
	if ablage.geschrieben != 1 {
		t.Errorf("legeFehlerstandAb hat %d-mal geschrieben, erwartet einmal nach dem gescheiterten Versuch", ablage.geschrieben)
	}
}

// Ein unlesbarer Stand hält das Ablegen nicht auf.
func TestFehlerstandAblage_UnlesbarerStandWirdErsetzt(t *testing.T) {
	ablage := &fehlerAblageAttrappe{wert: "{kaputt", vorhanden: true}
	legeFehlerstandAb(context.Background(), ablage, standMitFehler(t, "A"), time.Now())
	if ablage.geschrieben != 1 || !strings.Contains(ablage.wert, `"A"`) {
		t.Errorf("abgelegt = %d-mal %s, erwartet den Stand dieses Laufs", ablage.geschrieben, ablage.wert)
	}
}

// Ein Lauf, der nicht geordnet herunterfuhr, hinterlässt einen Stand ohne den Vermerk beendet: Er
// stürzte ab, wurde von außen beendet, oder der Strom fiel aus. Der nächste Start meldet das als
// Fehler; nach einem geordneten Herunterfahren meldet er nichts.
func TestFehlerstandAblage_MeldetDenAbgebrochenenLauf(t *testing.T) {
	const meldung = "Voriger Lauf des Servers endete ohne geordnetes Herunterfahren"
	faelle := []struct {
		name          string
		beendet       bool
		meldungAnzahl int
	}{
		{"abgebrochen", false, 1},
		{"heruntergefahren", true, 0},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			ctx := context.Background()
			ablage := &fehlerAblageAttrappe{}
			vorher := logger.NewFehlerstand()
			legeFehlerstandAb(ctx, ablage, vorher, time.Now())
			if f.beendet {
				vorher.Beende()
				legeFehlerstandAb(ctx, ablage, vorher, time.Now())
			}

			nachher := logger.NewFehlerstand()
			vorherLog := slog.Default()
			slog.SetDefault(slog.New(nachher.SchalteVor(slog.DiscardHandler)))
			t.Cleanup(func() { slog.SetDefault(vorherLog) })
			legeFehlerstandAb(ctx, ablage, nachher, time.Now())

			anzahl := 0
			for _, g := range nachher.Gruppen(time.Now()) {
				if g.Meldung == meldung {
					anzahl += g.Anzahl
				}
			}
			if anzahl != f.meldungAnzahl {
				t.Errorf("%d Zeilen „%s“, erwartet %d", anzahl, meldung, f.meldungAnzahl)
			}
		})
	}
}
