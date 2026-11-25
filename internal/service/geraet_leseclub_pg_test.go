package service

import (
	"context"
	"testing"
	"time"

	"bibliothek/internal/pkg/schulzeit"
	"bibliothek/internal/testdaten"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Läuft der Ferien-Leseclub, ist ein Gerät eines Schülers wie ein Buch erst zu seinem Datum
// fällig. Ist das Datum vorbei, gilt wieder die Leihfrist der Geräte.
func TestGeraetAusleihe_FerienLeseclub(t *testing.T) {
	g := newGeraeteTheke(t)
	ctx := context.Background()
	ziel := schulzeit.Jetzt().AddDate(0, 2, 0)
	setzeEinstellungenFuerTest(t, g.pool, map[string]*string{
		"ferien_leseclub_aktiv":     strPtr("true"),
		"ferien_leseclub_zieldatum": strPtr(ziel.Format(time.DateOnly)),
	})

	kind := g.schueler(t, "Leseclub")
	nummer := g.geraet(t, "leseclub")
	res, err := g.dienst.verarbeiteScan(ctx, nummer, &kind, true, false, g.ausgabe)
	if err != nil {
		t.Fatalf("Ausleihe im Leseclub: %v", err)
	}
	if want := schulzeit.TagesEnde(ziel); res.Frist == nil || !res.Frist.Equal(want) {
		t.Errorf("Frist %v, erwartet das Datum des Leseclubs %v", res.Frist, want)
	}

	vorbei := schulzeit.Jetzt().AddDate(0, 0, -1).Format(time.DateOnly)
	setzeEinstellungenFuerTest(t, g.pool, map[string]*string{"ferien_leseclub_zieldatum": &vorbei})
	spaeter := g.geraet(t, "leseclub-vorbei")
	res, err = g.dienst.verarbeiteScan(ctx, spaeter, &kind, true, false, g.ausgabe)
	if err != nil {
		t.Fatalf("Ausleihe nach dem Leseclub: %v", err)
	}
	if res.Frist == nil || res.Frist.After(schulzeit.Jetzt().AddDate(0, 0, 60)) {
		t.Errorf("Frist %v nach dem Leseclub, erwartet die Leihfrist der Geräte", res.Frist)
	}
}

// Ein Abgänger bekommt ein Gerät nicht über den letzten Schultag vor den Sommerferien hinaus,
// auch nicht im Ferien-Leseclub. Ein Kind der Klasse 5 bekommt am selben Tag dessen Datum.
func TestGeraetAusleihe_AbgaengerVorDenSommerferien(t *testing.T) {
	g := newGeraeteTheke(t)
	ctx := context.Background()
	g.dienst.jetzt = func() time.Time { return time.Date(2031, time.June, 30, 10, 0, 0, 0, schulzeit.Zone()) }
	setzeEinstellungenFuerTest(t, g.pool, map[string]*string{
		"sommerferien":              strPtr(sommerferien2031),
		"ferien_leseclub_aktiv":     strPtr("true"),
		"ferien_leseclub_zieldatum": strPtr("2031-08-20"),
	})
	abgaenger := testdaten.LegeSchuelerMitNamenAn(t, g.pool, "GT-Abgang-"+g.suffix, "Geraetetheke", "Abgang", "10R1", 2031)
	kind := g.schueler(t, "Fuenfte")

	for _, f := range []struct {
		name, leser string
		erwartet    time.Time
	}{
		{"abgang", abgaenger, fristEnde2031(time.July, 4)},
		{"fuenfte", kind, fristEnde2031(time.August, 20)},
	} {
		res, err := g.dienst.verarbeiteScan(ctx, g.geraet(t, f.name), &f.leser, true, false, g.ausgabe)
		if err != nil {
			t.Fatalf("%s: Ausleihe: %v", f.name, err)
		}
		if res.Frist == nil || !res.Frist.Equal(f.erwartet) {
			t.Errorf("%s: Frist %v, erwartet %v", f.name, res.Frist, f.erwartet)
		}
	}

	// Ein Leseclub über Ostern gilt für den Abgänger auch nicht: 14 Tage ab dem 01.04.2031.
	g.dienst.jetzt = func() time.Time { return time.Date(2031, time.April, 1, 10, 0, 0, 0, schulzeit.Zone()) }
	ostern := "2031-04-25"
	setzeEinstellungenFuerTest(t, g.pool, map[string]*string{"ferien_leseclub_zieldatum": &ostern})
	for _, f := range []struct {
		name, leser string
		erwartet    time.Time
	}{
		{"abgang-ostern", abgaenger, fristEnde2031(time.April, 15)},
		{"fuenfte-ostern", kind, fristEnde2031(time.April, 25)},
	} {
		res, err := g.dienst.verarbeiteScan(ctx, g.geraet(t, f.name), &f.leser, true, false, g.ausgabe)
		if err != nil {
			t.Fatalf("%s: Ausleihe: %v", f.name, err)
		}
		if res.Frist == nil || !res.Frist.Equal(f.erwartet) {
			t.Errorf("%s: Frist %v, erwartet %v", f.name, res.Frist, f.erwartet)
		}
	}
}

// setzeEinstellungenFuerTest setzt Einstellungen (nil = ohne Wert) und stellt am Ende des Tests
// den Stand davor wieder her: den alten Wert oder keine Zeile.
func setzeEinstellungenFuerTest(t *testing.T, pool *pgxpool.Pool, werte map[string]*string) {
	t.Helper()
	ctx := context.Background()
	for schluessel, wert := range werte {
		var vorher *string
		var gab bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM system_einstellungen WHERE schluessel = $1),
			(SELECT wert FROM system_einstellungen WHERE schluessel = $1)`, schluessel).Scan(&gab, &vorher); err != nil {
			t.Fatalf("Einstellung %s lesen: %v", schluessel, err)
		}
		t.Cleanup(func() {
			var err error
			if gab {
				_, err = pool.Exec(context.Background(),
					`UPDATE system_einstellungen SET wert = $2 WHERE schluessel = $1`, schluessel, vorher)
			} else {
				_, err = pool.Exec(context.Background(), `DELETE FROM system_einstellungen WHERE schluessel = $1`, schluessel)
			}
			if err != nil {
				t.Errorf("Einstellung %s zurücksetzen: %v", schluessel, err)
			}
		})
		if _, err := pool.Exec(ctx, `INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)
			ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, schluessel, wert); err != nil {
			t.Fatalf("Einstellung %s setzen: %v", schluessel, err)
		}
	}
}
