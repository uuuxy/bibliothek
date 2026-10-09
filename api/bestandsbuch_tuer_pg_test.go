package api

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pdftest"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// GET /api/bestand/abgangsbuch/pdf und /api/bestand/zugangsbuch/pdf liefern das Blatt zum
// Abheften. Die Erzeuger prüft pdf/bestandsbuch_test.go, ihre Eingabe bestandsbuch_test.go;
// hier steht die Tür davor, über den Router und mit Sitzung: Das Blatt trägt die Zeilen der
// Datenbank im Zeitraum, je Topf unter seiner Überschrift, und keine Zeile von außerhalb.
func TestBestandsbuecher_BlattUeberDieTuer(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	_, rufe := protokollWelt(t, pool)
	ctx := t.Context()
	tag := func(m time.Month, d int) time.Time {
		return time.Date(2026, m, d, 12, 0, 0, 0, schulzeit.Zone())
	}

	lernmittel := titelMitSignatur(t, pool, "Mathebuch 7", "Mat 7", 0)
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET ist_lernmittel = true WHERE id = $1`, lernmittel); err != nil {
		t.Fatalf("Lernmittel setzen: %v", err)
	}
	buecherei := titelMitSignatur(t, pool, "Gregs Tagebuch", "Jug Gre", 0)

	abgang := func(titelID, barcode string, wann time.Time) {
		t.Helper()
		id := exemplar(t, pool, titelID, barcode, true, "")
		if _, err := pool.Exec(ctx, `
			UPDATE buecher_exemplare
			SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'VERLUST'
			WHERE id = $1`, id); err != nil {
			t.Fatalf("aussondern %s: %v", barcode, err)
		}
		// Der Auslöser setzt den Zeitpunkt des Aussonderns; der Test braucht ein festes Datum.
		if _, err := pool.Exec(ctx,
			`UPDATE buecher_exemplare SET ausgesondert_am = $2 WHERE id = $1`, id, wann); err != nil {
			t.Fatalf("datieren %s: %v", barcode, err)
		}
	}
	abgang(lernmittel, "TUER-AB-LAND", tag(time.April, 12))
	abgang(buecherei, "TUER-AB-BUE", tag(time.May, 3))
	abgang(lernmittel, "TUER-AB-SPAET", tag(time.September, 16))

	var bestellung string
	if err := pool.QueryRow(ctx, `
		INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email, mittel)
		VALUES ('Buchhandlung Land', 'haendler@example.org', $1) RETURNING id`,
		repository.MittelLand).Scan(&bestellung); err != nil {
		t.Fatalf("Bestellung anlegen: %v", err)
	}
	zugang := func(barcode string, wann time.Time, bestellID *string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, erworben_am, bestellung_id)
			VALUES ($1, $2, $3::date, $4)`,
			buecherei, barcode, wann.Format(dateFormatISO), bestellID); err != nil {
			t.Fatalf("Zugang %s: %v", barcode, err)
		}
	}
	zugang("TUER-ZU-LAND", tag(time.April, 12), &bestellung)
	zugang("TUER-ZU-OHNE", tag(time.May, 2), nil)
	zugang("TUER-ZU-SPAET", tag(time.September, 16), &bestellung)

	blatt := func(t *testing.T, buch, dateiname string) string {
		t.Helper()
		rec := rufe(t, http.MethodGet, "/api/bestand/"+buch+"/pdf?von=2026-03-16&bis=2026-09-15", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: Status %d, erwartet 200: %s", buch, rec.Code, rec.Body.String())
		}
		if typ := rec.Header().Get("Content-Type"); !strings.Contains(typ, "application/pdf") {
			t.Errorf("%s: Content-Type %q, erwartet ein PDF", buch, typ)
		}
		if ablage := rec.Header().Get("Content-Disposition"); !strings.Contains(ablage, dateiname) {
			t.Errorf("%s: Content-Disposition %q nennt %q nicht", buch, ablage, dateiname)
		}
		return strings.Join(pdftest.Texte(t, rec.Body.Bytes()), " ")
	}
	pruefe := func(t *testing.T, text string, muss []string, darfNicht string) {
		t.Helper()
		for _, m := range muss {
			if !strings.Contains(text, m) {
				t.Errorf("auf dem Blatt fehlt %q:\n%s", m, text)
			}
		}
		if strings.Contains(text, darfNicht) {
			t.Errorf("auf dem Blatt steht %q, es liegt außerhalb des Zeitraums:\n%s", darfNicht, text)
		}
	}

	t.Run("Abgangsbuch", func(t *testing.T) {
		pruefe(t, blatt(t, "abgangsbuch", "Abgangsbuch_2026-03-16_bis_2026-09-15.pdf"), []string{
			"Zeitraum: 16.03.2026 bis 15.09.2026",
			"TUER-AB-LAND", "Mathebuch 7", "Mat 7", "12.04.2026", "Verlust",
			"Summe " + mittelBeschriftung(repository.MittelLand) + ": 1 Exemplare",
			"TUER-AB-BUE", "Gregs Tagebuch",
			"Summe " + mittelBeschriftung(repository.MittelSchultraeger) + ": 1 Exemplare",
			"Abgänge im Zeitraum: 2 Exemplare",
		}, "TUER-AB-SPAET")
	})

	t.Run("Zugangsbuch", func(t *testing.T) {
		pruefe(t, blatt(t, "zugangsbuch", "Zugangsbuch_2026-03-16_bis_2026-09-15.pdf"), []string{
			"Zeitraum: 16.03.2026 bis 15.09.2026",
			"TUER-ZU-LAND", "12.04.2026", "Buchhandlung Land",
			"Summe " + mittelBeschriftung(repository.MittelLand) + ": 1 Exemplare",
			"TUER-ZU-OHNE",
			"Summe " + mittelOhneZuordnung + ": 1 Exemplare",
			"keine Bestellung hinterlegt",
			"Zugänge im Zeitraum: 2 Exemplare",
		}, "TUER-ZU-SPAET")
	})
}
