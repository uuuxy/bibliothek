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
	if err := vorher.Uebernimm("", time.Now()); err != nil {
		t.Fatalf("Uebernimm: %v", err)
	}
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
