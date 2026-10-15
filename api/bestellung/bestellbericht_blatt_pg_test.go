package bestellung

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pdftest"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/pkg/schulzeit"
)

// Der Bestellbericht über seine Tür: Was in der Datenbank steht, steht auf dem Blatt, je Topf in
// einem Block mit eigener Summe. Die Tür liest die Zeilen, ordnet sie den Töpfen zu und reicht
// dem Erzeuger den Kopf der Schule, Überschrift, Zeitraum und die Wahl zwischen Beträgen und
// Mengen.
func TestBestellbericht_TuerDrucktDieBestellungenJeTopf(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	// Die Proben tragen ihren Namen und liegen im Jahr 2019, vor jeder anderen Bestellung der
	// Testdatenbank: Der Bericht zählt alle Bestellungen seines Zeitraums.
	raeume := func() {
		for _, sql := range []string{
			`DELETE FROM bestellungen_verlauf WHERE lieferant_name LIKE 'Bericht-Probe %'`,
			`DELETE FROM lieferanten WHERE name = 'Bericht-Probe Händler'`,
			`DELETE FROM system_einstellungen WHERE schluessel = 'preise_erfassen'`,
			// Die vier Zeilen der Schule legt das Schema mit leerem Wert an: zurück auf diesen
			// Stand, nicht löschen.
			`INSERT INTO system_einstellungen (schluessel, wert)
			 VALUES ('schule_name', ''), ('schule_strasse', ''), ('schule_plz', ''), ('schule_ort', '')
			 ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`,
		} {
			if _, err := pool.Exec(context.Background(), sql); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		}
	}
	raeume()
	t.Cleanup(raeume)

	var haendler string
	if err := pool.QueryRow(ctx, `
		INSERT INTO lieferanten (name, email, kundennummer) VALUES ('Bericht-Probe Händler', 'haendler@example.invalid', 'KD-H')
		RETURNING id`).Scan(&haendler); err != nil {
		t.Fatalf("Lieferant anlegen: %v", err)
	}
	for schluessel, wert := range map[string]string{
		"schule_name": "Probeschule Nord", "schule_strasse": "Probeweg 5", "schule_plz": "34117", "schule_ort": "Kassel",
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)
			ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, schluessel, wert); err != nil {
			t.Fatalf("Einstellung %s setzen: %v", schluessel, err)
		}
	}

	anlegen := func(name, mittel string, tag time.Time, menge int, einzelpreis float64) {
		t.Helper()
		var spalte any
		if mittel != "" {
			spalte = mittel
		}
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO bestellungen_verlauf
			    (lieferant_name, lieferant_email, kundennummer, bestelldatum, gesamtbetrag, anzahl_exemplare, mittel)
			VALUES ($1, 'haendler@example.invalid', $2, $3, $4, $5, $6) RETURNING id
		`, "Bericht-Probe "+name, "KD-"+name, tag,
			float64(menge)*einzelpreis, menge, spalte).Scan(&id); err != nil {
			t.Fatalf("Bestellung %s anlegen: %v", name, err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO bestellungen_positionen (bestellung_id, titel_name, isbn, menge, einzelpreis)
			VALUES ($1, $2, $3, $4, $5)`, id, "Titel "+name, "isbn-"+name, menge, einzelpreis); err != nil {
			t.Fatalf("Position %s anlegen: %v", name, err)
		}
	}
	mittags := func(monat time.Month, tag int) time.Time { return time.Date(2019, monat, tag, 10, 0, 0, 0, time.UTC) }
	// In der Folge der Abfrage (nach dem Bestelldatum) stehen die Töpfe durcheinander.
	anlegen("Alt", "", mittags(time.September, 2), 2, 5.00)
	anlegen("Kreis", mitteltopf.Schultraeger, mittags(time.September, 3), 5, 9.10)
	anlegen("Land-A", mitteltopf.Land, mittags(time.September, 4), 30, 4.00)
	anlegen("Land-B", mitteltopf.Land, mittags(time.September, 30), 20, 4.00)
	// Außerhalb des Zeitraums: am Tag vor „von" und am Tag nach „bis".
	anlegen("August", mitteltopf.Land, mittags(time.August, 31), 7, 1.00)
	anlegen("Oktober", mitteltopf.Land, mittags(time.October, 1), 9, 1.00)
	// Um Mitternacht der Schule, in UTC gespeichert: 00:30 Uhr am 01.11.2019 und am 01.12.2019.
	anlegen("Nacht-Anfang", mitteltopf.Land, time.Date(2019, time.October, 31, 23, 30, 0, 0, time.UTC), 1, 2.00)
	anlegen("Nacht-Ende", mitteltopf.Land, time.Date(2019, time.November, 30, 23, 30, 0, 0, time.UTC), 1, 3.00)
	if _, err := pool.Exec(ctx, `UPDATE bestellungen_verlauf SET lieferant_id = $1
		WHERE lieferant_name IN ('Bericht-Probe Land-A', 'Bericht-Probe Oktober')`, haendler); err != nil {
		t.Fatalf("Lieferant an zwei Bestellungen setzen: %v", err)
	}

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	rufe := func(t *testing.T, query string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/bestellungen/bericht.pdf?"+query, nil)
		rec := httptest.NewRecorder()
		srv.BestellberichtPDF().ServeHTTP(rec, req)
		return rec
	}
	blatt := func(t *testing.T, query string) []string {
		t.Helper()
		rec := rufe(t, query)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: Status %d: %s", query, rec.Code, rec.Body.String())
		}
		return pdftest.TexteInReihenfolge(t, rec.Body.Bytes())
	}
	// folge verlangt die Texte als eigene Zellen in dieser Folge.
	folge := func(t *testing.T, texte []string, will ...string) {
		t.Helper()
		stelle := 0
		for _, w := range will {
			i := slices.Index(texte[stelle:], w)
			if i < 0 {
				t.Errorf("%q steht nicht hinter den Texten davor; gedruckt ab dort: %q", w, texte[stelle:])
				return
			}
			stelle += i + 1
		}
	}
	setzePreise := func(t *testing.T, wert string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO system_einstellungen (schluessel, wert) VALUES ('preise_erfassen', $1)
			ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, wert); err != nil {
			t.Fatalf("Einstellung setzen: %v", err)
		}
	}

	t.Run("mit Preisen: je Topf ein Block mit Summe, darunter die Gesamtsumme", func(t *testing.T) {
		setzePreise(t, "true")
		texte := blatt(t, "von=2019-09-01&bis=2019-09-30")
		folge(t, texte,
			"Probeschule Nord", "Probeschule Nord · Probeweg 5 · 34117 Kassel", "Kassel, den "+schulzeit.Jetzt().Format("02.01.2006"),
			"Bestellbericht", "Zeitraum: 01.09.2019 bis 30.09.2019",
			"Bestellungen: 4", "Exemplare: 57", "Gesamtbetrag: 255,50 €",
			"Lernmittelfreiheit (Land)",
			"04.09.2019  ·  Bericht-Probe Land-A  ·  Kd.-Nr. KD-Land-A", "Titel Land-A", "isbn-Land-A", "30", "4,00 €", "120,00 €",
			"30.09.2019  ·  Bericht-Probe Land-B  ·  Kd.-Nr. KD-Land-B", "Titel Land-B",
			"Summe Lernmittelfreiheit (Land) (2 Bestellungen, 50 Exemplare)", "200,00 €",
			"Schülerbücherei (Schulträger)", "03.09.2019  ·  Bericht-Probe Kreis  ·  Kd.-Nr. KD-Kreis",
			"Summe Schülerbücherei (Schulträger) (1 Bestellungen, 5 Exemplare)", "45,50 €",
			"ohne Zuordnung", "02.09.2019  ·  Bericht-Probe Alt  ·  Kd.-Nr. KD-Alt",
			"Summe ohne Zuordnung (1 Bestellungen, 2 Exemplare)", "10,00 €",
			"Gesamtbetrag 01.09.2019 – 30.09.2019", "255,50 €")
		text := strings.Join(texte, "\n")
		if strings.Contains(text, "Probe August") || strings.Contains(text, "Probe Oktober") {
			t.Error("eine Bestellung außerhalb des Zeitraums steht auf dem Blatt")
		}
		if strings.Contains(text, "nach Monat") || strings.Contains(text, "nach Lieferant") {
			t.Error("der Bericht ohne den Parameter jahresansicht trägt die Übersichten der Jahresansicht")
		}
	})

	t.Run("ohne Preise: Mengen statt Beträge", func(t *testing.T) {
		setzePreise(t, "false")
		texte := blatt(t, "von=2019-09-01&bis=2019-09-30")
		folge(t, texte, "Bestellungen: 4", "Exemplare: 57", "Lernmittelfreiheit (Land)",
			"Summe Lernmittelfreiheit (Land) (2 Bestellungen, 50 Exemplare)", "Gesamt 01.09.2019 – 30.09.2019", "57")
		if text := strings.Join(texte, "\n"); strings.Contains(text, "€") {
			t.Errorf("der Mengenbericht nennt einen Betrag: %q", texte)
		}
	})

	t.Run("auf einen Topf eingeengt: Überschrift nennt ihn, der andere fehlt", func(t *testing.T) {
		setzePreise(t, "true")
		texte := blatt(t, "von=2019-09-01&bis=2019-09-30&mittel=schultraeger")
		folge(t, texte, "Bestellbericht — Schülerbücherei (Schulträger)", "Bestellungen: 1", "Gesamtbetrag: 45,50 €",
			"Schülerbücherei (Schulträger)", "Summe Schülerbücherei (Schulträger) (1 Bestellungen, 5 Exemplare)", "45,50 €")
		if text := strings.Join(texte, "\n"); strings.Contains(text, "Lernmittelfreiheit") || strings.Contains(text, "ohne Zuordnung") {
			t.Errorf("der Bericht über einen Topf nennt einen anderen: %q", texte)
		}
	})

	t.Run("auf einen Lieferanten eingeengt: nur seine Bestellung im Zeitraum", func(t *testing.T) {
		setzePreise(t, "true")
		texte := blatt(t, "von=2019-09-01&bis=2019-09-30&lieferant_id="+haendler)
		folge(t, texte, "Lieferantenabrechnung", "Bestellungen: 1", "Exemplare: 30", "Gesamtbetrag: 120,00 €",
			"04.09.2019  ·  Bericht-Probe Land-A  ·  Kd.-Nr. KD-Land-A", "Gesamtbetrag 01.09.2019 – 30.09.2019", "120,00 €")
		if text := strings.Join(texte, "\n"); strings.Contains(text, "Land-B") || strings.Contains(text, "Probe Oktober") {
			t.Errorf("die Abrechnung eines Lieferanten nennt eine fremde Bestellung oder eine außerhalb des Zeitraums: %q", texte)
		}
	})

	t.Run("Jahresansicht mit eigenem Titel: Übersichten nach Monat und Lieferant", func(t *testing.T) {
		setzePreise(t, "true")
		texte := blatt(t, "von=2019-09-01&bis=2019-09-30&jahresansicht=true&titel=Abrechnung+Herbst")
		folge(t, texte, "Abrechnung Herbst", "Übersicht nach Monat", "September", "4", "57", "255,50 €",
			"Ausgaben nach Lieferant", "Bericht-Probe Alt", "Bericht-Probe Kreis", "Bericht-Probe Land-A", "Bericht-Probe Land-B",
			"Bestellungen im Detail")
	})

	t.Run("der Zeitraum zählt am Kalendertag der Schule", func(t *testing.T) {
		setzePreise(t, "true")
		texte := blatt(t, "von=2019-11-01&bis=2019-11-30&jahresansicht=true")
		folge(t, texte, "Zeitraum: 01.11.2019 bis 30.11.2019", "Bestellungen: 1", "Gesamtbetrag: 2,00 €",
			"Übersicht nach Monat", "November", "1",
			"01.11.2019  ·  Bericht-Probe Nacht-Anfang  ·  Kd.-Nr. KD-Nacht-Anfang", "Gesamtbetrag 01.11.2019 – 30.11.2019", "2,00 €")
		if text := strings.Join(texte, "\n"); strings.Contains(text, "Nacht-Ende") || strings.Contains(text, "Oktober") {
			t.Errorf("der November nennt die Bestellung vom 01.12. oder zählt eine im Oktober: %q", texte)
		}
		// Die Bestellung aus der ersten halben Stunde des Dezembers steht im Dezember.
		folge(t, blatt(t, "von=2019-12-01&bis=2019-12-31"), "Bestellungen: 1",
			"01.12.2019  ·  Bericht-Probe Nacht-Ende  ·  Kd.-Nr. KD-Nacht-Ende", "3,00 €")
		// Und der Oktober endet vor ihr: Er führt nur die Bestellung vom 01.10.
		oktober := blatt(t, "von=2019-10-01&bis=2019-10-31")
		folge(t, oktober, "Bestellungen: 1", "01.10.2019  ·  Bericht-Probe Oktober  ·  Kd.-Nr. KD-Oktober")
		if strings.Contains(strings.Join(oktober, "\n"), "Nacht-Anfang") {
			t.Errorf("der Oktober nennt die Bestellung vom 01.11.: %q", oktober)
		}
	})

	t.Run("Kopf der Antwort und Eingabefehler", func(t *testing.T) {
		rec := rufe(t, "von=2019-09-01&bis=2019-09-30")
		if ist := rec.Header().Get("Content-Type"); ist != "application/pdf" {
			t.Errorf("Content-Type %q, erwartet application/pdf", ist)
		}
		if ist, will := rec.Header().Get("Content-Disposition"), `attachment; filename="bestellbericht_2019-09-01_2019-09-30.pdf"`; ist != will {
			t.Errorf("Content-Disposition %q, erwartet %q", ist, will)
		}
		for query, nennt := range map[string]string{
			"von=2019-09-01":                             "Anfang und Ende des Zeitraums",
			"von=2019-09-01&bis=30.09.2019":              "Das Ende des Zeitraums",
			"von=2019-09-01&bis=2019-09-30&mittel=kreis": "kreis",
		} {
			rec := rufe(t, query)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s: Status %d, erwartet 400: %s", query, rec.Code, rec.Body.String())
			} else if !strings.Contains(rec.Body.String(), nennt) {
				t.Errorf("%s: die Antwort nennt %s nicht: %s", query, nennt, rec.Body.String())
			}
		}
	})
}
