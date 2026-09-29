package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Abholfrist, entschieden am 28.09.2026: drei Tage ab der Zuteilung, das Ende fällt wie
// bei der Leihfrist auf den nächsten Schultag, bis Tagesende. Geprüft an den drei Wegen, auf
// denen repository die Warteschlange nachrücken lässt — Verfall-Lauf, Löschen von Hand,
// Spuren-Tilgung —, über ein Wochenende, über die Herbstferien und über Sommerferien, die nur
// in der Einstellung stehen. Die Rückgabe prüft internal/service
// (TestRueckgabe_AbholfristMitFesterUhr).

func abholfristEnde(j int, m time.Month, d int) time.Time {
	return time.Date(j, m, d, 23, 59, 59, 0, schulzeit.Zone())
}

func um10(j int, m time.Month, d int) time.Time {
	return time.Date(j, m, d, 10, 0, 0, 0, schulzeit.Zone())
}

// abholLage: Für Nora lag ein Exemplar abholbereit, ihre Frist ist seit gestern vorbei; Nils
// wartet als Nächster am selben Titel.
func abholLage(t *testing.T, pool *pgxpool.Pool, praefix string) (noraVormerkung, nora, nils string) {
	t.Helper()
	ctx := context.Background()
	ex := seedSignaturMitExemplaren(t, pool, praefix, 1)
	titelID := titelIDVonExemplar(t, pool, ex[0])
	nora = seedSchueler(t, pool, praefix+"-NORA", "Nora", "7a")
	nils = seedSchueler(t, pool, praefix+"-NILS", "Nils", "7b")
	if err := pool.QueryRow(ctx, `
		INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis, erstellt_am)
		VALUES ($1, $2, 'abholbereit', $3, CURRENT_TIMESTAMP - INTERVAL '1 day', CURRENT_TIMESTAMP - INTERVAL '5 days')
		RETURNING id`, titelID, nora, ex[0]).Scan(&noraVormerkung); err != nil {
		t.Fatalf("Noras Vormerkung: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO vormerkungen (titel_id, schueler_id, status, erstellt_am)
		VALUES ($1, $2, 'wartend', CURRENT_TIMESTAMP - INTERVAL '2 days')`, titelID, nils); err != nil {
		t.Fatalf("Nils' Vormerkung: %v", err)
	}
	return noraVormerkung, nora, nils
}

// abholfristVon liest die Frist der Vormerkung eines Lesers; sie muss abholbereit sein.
func abholfristVon(t *testing.T, pool *pgxpool.Pool, schuelerID string) time.Time {
	t.Helper()
	var status string
	var bis *time.Time
	if err := pool.QueryRow(context.Background(),
		`SELECT status, bereitgestellt_bis FROM vormerkungen WHERE schueler_id = $1`, schuelerID).Scan(&status, &bis); err != nil {
		t.Fatalf("Vormerkung lesen: %v", err)
	}
	if status != "abholbereit" || bis == nil {
		t.Fatalf("nicht nachgerückt: Status %q, Frist %v", status, bis)
	}
	return *bis
}

func vormerkungenMitUhr(pool *pgxpool.Pool, jetzt time.Time) *pgVormerkungRepository {
	return &pgVormerkungRepository{db: pool, jetzt: func() time.Time { return jetzt }}
}

// setzeSommerferien trägt die eigenen Sommerferien der Schule ein und stellt den alten Stand
// danach wieder her.
func setzeSommerferien(t *testing.T, pool *pgxpool.Pool, wert string) {
	t.Helper()
	ctx := context.Background()
	var alt *string
	err := pool.QueryRow(ctx, `SELECT wert FROM system_einstellungen WHERE schluessel = $1`,
		lmfplan.SommerferienSchluessel).Scan(&alt)
	hatteZeile := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("Einstellung lesen: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, lmfplan.SommerferienSchluessel, wert); err != nil {
		t.Fatalf("Einstellung setzen: %v", err)
	}
	t.Cleanup(func() {
		var err error
		if hatteZeile {
			_, err = pool.Exec(ctx, `UPDATE system_einstellungen SET wert = $2 WHERE schluessel = $1`,
				lmfplan.SommerferienSchluessel, alt)
		} else {
			_, err = pool.Exec(ctx, `DELETE FROM system_einstellungen WHERE schluessel = $1`, lmfplan.SommerferienSchluessel)
		}
		if err != nil {
			t.Errorf("Einstellung zurücksetzen: %v", err)
		}
	})
}

// Der Verfall-Lauf räumt Noras Vormerkung ab und teilt Nils das Exemplar zu; seine Frist
// zählt ab der Uhr des Laufs. Rot am Rückbau zu CURRENT_TIMESTAMP + INTERVAL '3 days'.
func TestAbholfrist_VerfallLauf(t *testing.T) {
	pool := pgTestPool(t)
	for _, fall := range []struct {
		name        string
		jetzt, soll time.Time
	}{
		// Dienstag plus drei Tage ist Freitag, ein Schultag: bis zu seinem Ende.
		{"an einem Schultag", um10(2026, time.September, 22), abholfristEnde(2026, time.September, 25)},
		// Mittwoch plus drei Tage ist Samstag: der Montag danach.
		{"über ein Wochenende", um10(2026, time.September, 16), abholfristEnde(2026, time.September, 21)},
		// Freitag vor den Herbstferien (05.10.–17.10.2026): plus drei Tage ist ihr erster
		// Tag, der nächste Schultag der Montag nach den Ferien.
		{"über die Herbstferien", um10(2026, time.October, 2), abholfristEnde(2026, time.October, 19)},
	} {
		t.Run(fall.name, func(t *testing.T) {
			resetInventurDaten(t, pool)
			_, _, nils := abholLage(t, pool, "AFV")
			verfallen, neuBereit, err := vormerkungenMitUhr(pool, fall.jetzt).VerfalleAbgelaufeneVormerkungen(context.Background())
			if err != nil {
				t.Fatalf("Verfall: %v", err)
			}
			if verfallen != 1 || neuBereit != 1 {
				t.Fatalf("erwartet 1 verfallen und 1 nachgerückt, war %d/%d", verfallen, neuBereit)
			}
			if ist := abholfristVon(t, pool, nils); !ist.Equal(fall.soll) {
				t.Errorf("Abholfrist %s, erwartet %s", ist.In(schulzeit.Zone()), fall.soll)
			}
		})
	}
}

// Die Sommerferien der Schule kommen aus der Einstellung, wie bei der Leihfrist: 2031 steht
// nicht in der Programmtabelle. Freitag 11.07.2031 plus drei Tage ist Montag 14.07., der
// erste Tag der eingetragenen Ferien; ohne die Einstellung wäre es ein Schultag.
func TestAbholfrist_SommerferienAusDerEinstellung(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	setzeSommerferien(t, pool, `[{"jahr":2031,"von":"2031-07-14","bis":"2031-08-22"}]`)
	_, _, nils := abholLage(t, pool, "AFS")
	if _, _, err := vormerkungenMitUhr(pool, um10(2031, time.July, 11)).VerfalleAbgelaufeneVormerkungen(context.Background()); err != nil {
		t.Fatalf("Verfall: %v", err)
	}
	if ist, soll := abholfristVon(t, pool, nils), abholfristEnde(2031, time.August, 25); !ist.Equal(soll) {
		t.Errorf("Abholfrist %s, erwartet %s (erster Schultag nach den eingetragenen Ferien)", ist.In(schulzeit.Zone()), soll)
	}
}

// Wer Noras Vormerkung von Hand löscht, lässt Nils nachrücken — mit derselben Frist.
func TestAbholfrist_LoeschenVonHand(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	noraVormerkung, _, nils := abholLage(t, pool, "AFL")
	if err := vormerkungenMitUhr(pool, um10(2026, time.September, 16)).Delete(context.Background(), noraVormerkung); err != nil {
		t.Fatalf("Löschen: %v", err)
	}
	if ist, soll := abholfristVon(t, pool, nils), abholfristEnde(2026, time.September, 21); !ist.Equal(soll) {
		t.Errorf("Abholfrist %s, erwartet %s", ist.In(schulzeit.Zone()), soll)
	}
}

// Die Spuren-Tilgung löscht Noras Vormerkungen und lässt Nils nachrücken. Sie hat keine
// eigene Uhr: Die Frist ist die Tagesfrist ab dem Aufruf.
func TestAbholfrist_SpurenTilgung(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	_, nora, nils := abholLage(t, pool, "AFT")
	vorher := time.Now()
	if _, err := loescheVormerkungenUndRuecktNach(context.Background(), pool, []string{nora}); err != nil {
		t.Fatalf("Tilgung: %v", err)
	}
	nachher := time.Now()
	ist := abholfristVon(t, pool, nils)
	kalender := lmfplan.FerientabelleAus(sommerferienImTest(t, pool))
	if a, b := kalender.Tagesfrist(vorher, 3), kalender.Tagesfrist(nachher, 3); !ist.Equal(a) && !ist.Equal(b) {
		t.Errorf("Abholfrist %s, erwartet %s", ist.In(schulzeit.Zone()), a)
	}
}

// sommerferienImTest liest die Einstellung, wie sie in der Testdatenbank steht.
func sommerferienImTest(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	einstellungen, err := EinstellungenUeber(context.Background(), pool)
	if err != nil {
		t.Fatalf("Einstellungen lesen: %v", err)
	}
	return einstellungen.Sommerferien
}
