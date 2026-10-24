package jobs

// Der Fehlerstand übersteht einen Neustart.
//
// Er zählt die Zeilen der Stufe ERROR im Speicher (pkg/logger). Ein Update ersetzt den Container,
// und der Wächter mailt drei Minuten nach dem Start; ohne Ablage fehlten in dieser Mail die Zeilen
// vor dem Update. Der Stand liegt deshalb in system_einstellungen, wie das Ergebnis der
// Restore-Probe.

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"bibliothek/db"
	"bibliothek/pkg/logger"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// fehlerstandSchluessel ist der Schlüssel des Fehlerstands in system_einstellungen.
const fehlerstandSchluessel = "fehler_im_log"

// fehlerstandTakt ist der Abstand, in dem der Fehlerstand abgelegt wird; ein Absturz verliert
// höchstens die Zeilen dieser Zeit.
const fehlerstandTakt = 10 * time.Second

// fehlerAblage liest und schreibt den abgelegten Fehlerstand (repository.BetriebszustandAbfragen).
type fehlerAblage interface {
	// LadeEinstellungswert liefert den abgelegten Stand; pgx.ErrNoRows heißt, es gibt keinen.
	LadeEinstellungswert(ctx context.Context, schluessel string) (string, error)
	// SchreibeEinstellungswert legt den Stand ab oder ersetzt ihn.
	SchreibeEinstellungswert(ctx context.Context, schluessel, wert string) error
}

// FuehreFehlerstandAblageAus übernimmt den Stand des vorigen Laufs und legt den Fehlerstand danach
// im Takt ab, sobald eine Zeile dazukam, bis ctx endet. Den letzten Stand legt
// SchreibeFehlerstand beim Herunterfahren ab.
func FuehreFehlerstandAblageAus(ctx context.Context, pool db.Pool, stand *logger.Fehlerstand) {
	ablage := repository.NewBetriebszustandAbfragen(pool)
	takt := time.NewTicker(fehlerstandTakt)
	defer takt.Stop()
	for {
		legeFehlerstandAb(ctx, ablage, stand, time.Now())
		select {
		case <-ctx.Done():
			return
		case <-takt.C:
		}
	}
}

// SchreibeFehlerstand legt den Fehlerstand ab; main ruft es beim Herunterfahren, nach
// logger.Fehlerstand.Beende und noch einmal, wenn der Server keine Anfrage mehr annimmt.
func SchreibeFehlerstand(ctx context.Context, pool db.Pool, stand *logger.Fehlerstand) {
	legeFehlerstandAb(ctx, repository.NewBetriebszustandAbfragen(pool), stand, time.Now())
}

// legeFehlerstandAb übernimmt zuerst den Stand des vorigen Laufs, solange das aussteht, und legt
// dann ab, was neu ist. Scheitert das Lesen, unterbleibt das Ablegen bis zum nächsten Versuch: Es
// überschriebe sonst den Stand des vorigen Laufs. Beides meldet WARN, damit der Fehlerstand nicht
// sein eigenes Scheitern zählt.
//
// Fuhr der vorige Lauf nicht geordnet herunter, steht das als ERROR im Log: Einen Absturz, den kein
// Code abfangen kann (Speicher, Laufzeitfehler), einen Abbruch von außen und einen Stromausfall
// sieht erst der nächste Start.
func legeFehlerstandAb(ctx context.Context, ablage fehlerAblage, stand *logger.Fehlerstand, jetzt time.Time) {
	if !stand.IstUebernommen() {
		abgelegt, err := ablage.LadeEinstellungswert(ctx, fehlerstandSchluessel)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			slog.Warn("Fehlerstand: abgelegter Stand nicht lesbar, nächster Versuch im nächsten Takt", "fehler", err)
			return
		}
		abgebrochen, err := stand.Uebernimm(abgelegt, jetzt)
		if err != nil {
			slog.Warn("Fehlerstand: abgelegter Stand unlesbar, er wird ersetzt", "fehler", err)
		}
		if abgebrochen {
			slog.Error("Voriger Lauf des Servers endete ohne geordnetes Herunterfahren",
				"fehler", "Absturz, Abbruch von außen oder Stromausfall; die Ursache steht im Log des Containers vor diesem Start")
		}
	}
	err := stand.LegeAb(jetzt, func(wert string) error {
		return ablage.SchreibeEinstellungswert(ctx, fehlerstandSchluessel, wert)
	})
	if err != nil {
		slog.Warn("Fehlerstand: nicht abgelegt, nächster Versuch im nächsten Takt", "fehler", err)
	}
}
