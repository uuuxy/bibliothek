package repository

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/pkg/schulzeit"
)

// Der Monatstrend zählt nach dem Kalendermonat der Schule. Eine Ausleihe am Monatsersten um
// 0:30 Uhr Berliner Zeit gehört in diesen Monat; in der Zone der Sitzung gerechnet (im Image
// UTC) fiel sie in den Vormonat. Der Test hängt nicht an der Uhrzeit: Er rechnet in drei
// Sitzungszonen, UTC−12 und UTC+14 liegen 26 Stunden auseinander.
func TestLadeMonatsTrend_ZaehltImKalendermonatDerSchule(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	monat := schulzeit.Jetzt().Format("2006-01")

	for _, zone := range []string{"UTC", "Etc/GMT+12", "Etc/GMT-14"} {
		t.Run(zone, func(t *testing.T) {
			tx := beginne(t, pool)
			defer func() {
				if err := tx.Rollback(ctx); err != nil {
					t.Errorf("zurückrollen: %v", err)
				}
			}()
			if _, err := tx.Exec(ctx, `SET LOCAL TIME ZONE '`+zone+`'`); err != nil {
				t.Fatal(err)
			}
			// Gezählt wird der Unterschied: Die Testdatenbank trägt Ausleihen anderer Tests.
			zaehle := func() (letzter MonatsTrendPunkt) {
				t.Helper()
				trend, err := LadeMonatsTrend(ctx, tx, "")
				if err != nil || len(trend) != 12 {
					t.Fatalf("LadeMonatsTrend: %d Monate, %v", len(trend), err)
				}
				return trend[11]
			}
			vorher := zaehle()

			_, leserID, bearbeiterID := geraeteAufbau(t, tx, "trend")
			if _, err := tx.Exec(ctx, `
				WITH ti AS (INSERT INTO buecher_titel (titel) VALUES ('Trend-Probe') RETURNING id),
				     e AS (INSERT INTO buecher_exemplare (titel_id, barcode_id) SELECT id, 'TREND-1' FROM ti RETURNING id)
				INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, ausgeliehen_am, rueckgabe_frist)
				SELECT id, $1, $2,
				       (date_trunc('month', now() AT TIME ZONE 'Europe/Berlin') + INTERVAL '30 minutes') AT TIME ZONE 'Europe/Berlin',
				       now() + INTERVAL '14 days'
				FROM e`, leserID, bearbeiterID); err != nil {
				t.Fatalf("Ausleihe anlegen: %v", err)
			}

			nachher := zaehle()
			if nachher.Monat != monat || nachher.Ausleihen != vorher.Ausleihen+1 {
				t.Errorf("letzter Monat %s mit %d Ausleihen (vorher %d), erwartet %s mit einer mehr",
					nachher.Monat, nachher.Ausleihen, vorher.Ausleihen, monat)
			}
		})
	}
}

// Die Zeiträume der Rangliste beginnen um Mitternacht der Schule: Eine Ausleihe um 0:30 Uhr
// Berliner Zeit am ersten Tag zählt mit, eine um 23:30 Uhr am Tag davor nicht, in jeder
// Sitzungszone.
func TestAusleihZeitraumBedingung_BeginntUmMitternachtDerSchule(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	beginn := map[string]string{
		"schuljahr": "'" + schuljahresBeginn(schulzeit.Jetzt()).Format("2006-01-02") + "'::timestamp",
		"monat":     "(" + schulzeit.SQLHeute + " - 30)::timestamp",
	}
	for _, zone := range []string{"UTC", "Etc/GMT+12", "Etc/GMT-14"} {
		for zeitraum, erster := range beginn {
			for _, f := range []struct {
				versatz string
				zaehlt  bool
			}{{"30 minutes", true}, {"-30 minutes", false}} {
				tx := beginne(t, pool)
				var zaehlt bool
				_, err := tx.Exec(ctx, `SET LOCAL TIME ZONE '`+zone+`'`)
				if err == nil {
					err = tx.QueryRow(ctx, `
						SELECT EXISTS (SELECT 1 FROM (VALUES ((`+erster+` + INTERVAL '`+f.versatz+`') AT TIME ZONE 'Europe/Berlin'))
						AS a(ausgeliehen_am) WHERE true `+AusleihZeitraumBedingung(zeitraum)+`)`).Scan(&zaehlt)
				}
				if rbErr := tx.Rollback(ctx); rbErr != nil {
					t.Fatalf("zurückrollen: %v", rbErr)
				}
				if err != nil || zaehlt != f.zaehlt {
					t.Errorf("%s, %s, Beginn %s: zählt %v (%v), erwartet %v", zone, zeitraum, f.versatz, zaehlt, err, f.zaehlt)
				}
			}
		}
	}
}
