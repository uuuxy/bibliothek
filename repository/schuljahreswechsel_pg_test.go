package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Sperre gegen einen zweiten Schuljahreswechsel hängt allein am Eintrag des Laufs im
// Protokoll der Verwaltung. Ein Lauf von gestern sperrt, ebenso einer kurz vor Ende des
// Abstands, und den löscht der Nachtlauf auch bei der kürzesten Aufbewahrung nicht, die die
// Einstellungen zulassen; nach dem Abstand sperrt der Lauf nicht mehr.
func TestSchuljahreswechsel_SperreHaeltDenAbstand(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	abstand := schuljahreswechselAbstandMonate
	for _, fall := range []struct {
		name   string
		alter  string // wie lange der Lauf her ist, als Zeitspanne der Datenbank
		sperrt bool
	}{
		{name: "gestern", alter: "1 day", sperrt: true},
		{name: "einen Tag vor Ende des Abstands", alter: fmt.Sprintf("%d months -1 day", abstand), sperrt: true},
		{name: "einen Tag nach Ende des Abstands", alter: fmt.Sprintf("%d months 1 day", abstand), sperrt: false},
	} {
		t.Run(fall.name, func(t *testing.T) {
			tx := beginneOhneLauf(t, pool)
			if _, err := tx.Exec(ctx, `INSERT INTO audit_logs (aktion, zeitstempel) VALUES ($1, NOW() - $2::interval)`,
				AktionSchuljahreswechsel, fall.alter); err != nil {
				t.Fatalf("Lauf eintragen: %v", err)
			}
			if _, err := LoescheAltesVerwaltungsProtokoll(ctx, tx, protokollAufbewahrungMonateMinimum); err != nil {
				t.Fatalf("Nachtlauf: %v", err)
			}

			lauf, err := FindeJuengstenSchuljahreswechsel(ctx, tx)
			if err != nil {
				t.Fatalf("FindeJuengstenSchuljahreswechsel: %v", err)
			}
			if gesperrt := lauf != nil; gesperrt != fall.sperrt {
				t.Errorf("Ein Lauf %s: sperrt = %v, erwartet %v", fall.name, gesperrt, fall.sperrt)
			}
		})
	}
}

// Der nächste Lauf geht ein halbes Jahr später zur selben Uhrzeit der Schule, auch wenn dazwischen
// die Uhr umgestellt wird. Die Zeitpunkte liegen in der Zukunft, damit der Lauf sperrt, gleich an
// welchem Tag der Test läuft.
func TestSchuljahreswechsel_NaechsterLaufZurSelbenUhrzeit(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	zone := schulzeit.Zone()

	for _, fall := range []struct {
		lief, naechster time.Time
	}{
		{time.Date(2099, time.July, 20, 10, 15, 0, 0, zone), time.Date(2100, time.January, 20, 10, 15, 0, 0, zone)},
		{time.Date(2099, time.November, 2, 23, 30, 0, 0, zone), time.Date(2100, time.May, 2, 23, 30, 0, 0, zone)},
	} {
		t.Run(fall.lief.Format(time.RFC3339), func(t *testing.T) {
			tx := beginneOhneLauf(t, pool)
			if _, err := tx.Exec(ctx, `INSERT INTO audit_logs (aktion, zeitstempel) VALUES ($1, $2)`,
				AktionSchuljahreswechsel, fall.lief); err != nil {
				t.Fatalf("Lauf eintragen: %v", err)
			}

			lauf, err := FindeJuengstenSchuljahreswechsel(ctx, tx)
			if err != nil || lauf == nil {
				t.Fatalf("FindeJuengstenSchuljahreswechsel = %v, %v; erwartet den Lauf", lauf, err)
			}
			if !lauf.Lief.Equal(fall.lief) || !lauf.NaechsterAb.Equal(fall.naechster) {
				t.Errorf("Lauf %s, nächster ab %s; erwartet %s und %s", lauf.Lief.In(zone), lauf.NaechsterAb.In(zone),
					fall.lief, fall.naechster)
			}
		})
	}
}

// beginneOhneLauf öffnet eine Transaktion, die am Ende des Tests zurückrollt, und nimmt darin die
// Läufe anderer Tests aus dem Protokoll.
func beginneOhneLauf(t *testing.T, pool *pgxpool.Pool) pgx.Tx {
	t.Helper()
	ctx := context.Background()
	tx := beginne(t, pool)
	t.Cleanup(func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	})
	if _, err := tx.Exec(ctx, `DELETE FROM audit_logs WHERE aktion = $1`, AktionSchuljahreswechsel); err != nil {
		t.Fatalf("Protokoll leeren: %v", err)
	}
	return tx
}
