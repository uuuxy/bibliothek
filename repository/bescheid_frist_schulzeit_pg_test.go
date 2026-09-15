package repository

import (
	"context"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
)

// Die Frist eines Bescheids entsteht als Kalendertag in der Schulzeitzone
// (api/bescheid_handler.go, schulzeit.Jetzt). Bis zum 16.09.2026 prüfte die Bedingung
// „Frist abgelaufen" gegen CURRENT_DATE, also gegen den Kalendertag der Datenbank-Sitzung
// (im Image UTC). Zwischen Mitternacht in Berlin und Mitternacht UTC galt eine Frist von
// gestern damit als nicht abgelaufen: Die Übergabe an die Schulaufsicht fand keine Zeile,
// und die Liste zeigte den Bescheid nicht als überfällig. Aufgefallen am 16.09.2026 um
// 00:01 Uhr an TestRueckkehrEinesAbgerechnetenBuches.
//
// Der Test hängt nicht an der Uhrzeit: Er wertet die Bedingung in zwei Sitzungszonen aus,
// die 26 Stunden auseinanderliegen (UTC−12 und UTC+14). Zu jeder Stunde weicht darum
// mindestens eine von beiden vom Berliner Kalendertag ab. Geprüft wird in beiden Zonen:
// Frist gestern → abgelaufen, Frist heute → nicht abgelaufen (Tage in Berlin gerechnet).
func TestBescheidFristAbgelaufen_RechnetInDerSchulzeitzone(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	// Die Frage „ist noch etwas zu zahlen" braucht Forderungszeilen; hier geht es nur um
	// den Kalendertag. Die Ersetzung muss greifen, sonst prüfte der Test etwas anderes.
	if !strings.Contains(bescheidFristAbgelaufen, bescheidHatOffenePosition) {
		t.Fatal("bescheidFristAbgelaufen enthält bescheidHatOffenePosition nicht mehr — Test anpassen")
	}
	praedikat := strings.Replace(bescheidFristAbgelaufen, bescheidHatOffenePosition, "true", 1)

	faelle := []struct {
		name       string
		tageBerlin int
		abgelaufen bool
	}{
		{"Frist gestern", -1, true},
		{"Frist heute", 0, false},
	}

	// SET LOCAL gilt nur bis zum Ende der Transaktion; das Zurückrollen gibt die
	// Verbindung ohne fremde Zeitzone an den geteilten Pool zurück.
	werteAus := func(zone string, tageBerlin int) (bool, error) {
		tx := beginne(t, pool)
		var ergebnis bool
		_, err := tx.Exec(ctx, `SET LOCAL TIME ZONE '`+zone+`'`)
		if err == nil {
			err = tx.QueryRow(ctx, `
				SELECT `+praedikat+`
				FROM (VALUES ('offen', (now() AT TIME ZONE 'Europe/Berlin')::date + $1::int, gen_random_uuid()))
				     AS b(status, frist_bis, id)`, tageBerlin).Scan(&ergebnis)
		}
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			t.Fatalf("zurückrollen: %v", rbErr)
		}
		return ergebnis, err
	}

	for _, zone := range []string{"Etc/GMT+12", "Etc/GMT-14"} {
		for _, f := range faelle {
			ergebnis, err := werteAus(zone, f.tageBerlin)
			if err != nil {
				t.Fatalf("%s in %s: %v", f.name, zone, err)
			}
			if ergebnis != f.abgelaufen {
				t.Errorf("%s, Sitzungszone %s: abgelaufen = %v, erwartet %v", f.name, zone, ergebnis, f.abgelaufen)
			}
		}
	}
}
