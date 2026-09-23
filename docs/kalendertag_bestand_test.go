package docs

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Ratsche gegen die Bugklasse „Tag in der falschen Zeitzone".
//
// Die Datenbank-Sitzung und der Container laufen in UTC, die Schule in Berlin. Zwischen
// Mitternacht in Berlin und Mitternacht UTC liefern `CURRENT_DATE` und `time.Now()`
// deshalb den VORTAG. Was das kostet, hat dieses Projekt inzwischen siebenmal gesehen:
//
//   - die Frist eines Bescheids galt zu spät als abgelaufen (16.09.2026),
//   - am 18. Geburtstag ging der Bescheid an die Eltern eines Erwachsenen,
//   - ein Mahnlauf kurz nach Mitternacht wurde als „heute schon gemahnt" übersprungen,
//   - Rückgaben zwischen 0 und 2 Uhr fehlten auf dem Dashboard,
//   - der Stornierungsgrund „Rückgabe am …" trug das Datum von gestern,
//   - Mahnliste und Kontoauszüge trugen im Betreff das Datum von gestern,
//   - die Bestellmail an den Händler nannte ein Bestelldatum von gestern.
//
// Jedes Mal war es dieselbe Zeile in einer anderen Datei. Ein Kommentar hält das nicht
// auf; eine Liste, die nur wachsen darf, wenn jemand einen Grund dazuschreibt, schon.
//
// Was die Ratsche NICHT verbietet: Zeitpunkt-Vergleiche. „Überfällig" ist
// `rueckgabe_frist < CURRENT_TIMESTAMP` und bleibt es — ein Instant ist in jeder Zone
// derselbe. Verboten ist nur der KALENDERTAG aus einer fremden Zone.

// bestandKalendertag: Stellen, die heute noch mit der Zeitzone der Sitzung bzw. des
// Containers rechnen dürfen — je Datei die Anzahl und der Grund.
//
// Die Zahl darf sinken, nie steigen. Wer eine Stelle umstellt, trägt sie hier aus.
var bestandKalendertag = map[string]struct {
	anzahl int
	grund  string
}{
	// Einmal-Werkzeug der Alt-Übernahme: Anschaffungsdatum eines importierten Exemplars.
	// Ein um zwei Stunden verschobenes Kaufdatum ändert keine Entscheidung über einen
	// Menschen, und es läuft ohnehin von Hand und tagsüber. Der Listen-Import stand hier
	// bis zum 23.09.2026; seitdem nimmt er die Vorgabe der Spalte (Migration 139).
	"cmd/migrate/pg_writer.go":         {1, "erworben_am beim Alt-Import"},
	"api/stats.go":                     {5, "Statistik-Fenster (30 Tage, 12 Monate): verschiebt ein Ranking, keine Entscheidung; das Schuljahr dort rechnet bereits in der Schulzeitzone"},
	"cmd/seed/main.go":                 {1, "erfundene Abgangsjahre für Testdaten"},
	"inventur/lernmittel_handler.go":   {1, "Datum im DATEINAMEN eines Downloads"},
	"inventur/export_csv.go":           {1, "Datum im DATEINAMEN eines Downloads"},
	"internal/uebernahme/protokoll.go": {1, "Zeitstempel einer Protokollzeile"},
	"api/pdf_service.go":               {1, "Datum im DATEINAMEN eines Downloads"},
	"api/mahnwesen_bulk.go":            {1, "Datum im DATEINAMEN der Sammel-PDF"},
	"api/order_service.go":             {1, "Anschaffungsjahr eines Exemplars"},
}

var (
	musterCurrentDate = regexp.MustCompile(`\bCURRENT_DATE\b`)
	musterRoheUhr     = regexp.MustCompile(`time\.Now\(\)\.(Format|Year)\(`)
	musterZeilenKomm  = regexp.MustCompile(`(?m)^\s*//.*$`)
	musterSQLKomm     = regexp.MustCompile(`(?m)^\s*--.*$`)
)

// TestKalendertag_BestandWaechstNicht hält die Liste gegen den Baum.
func TestKalendertag_BestandWaechstNicht(t *testing.T) {
	gefunden := map[string]int{}
	wurzel := ".."
	err := filepath.Walk(wurzel, func(pfad string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if name := info.Name(); name == "node_modules" || name == ".git" || name == "frontend" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		roh, err := os.ReadFile(pfad)
		if err != nil {
			return err
		}
		// Kommentare zuerst weg: Ein Fund, der nur eine Erklärung ist, wäre ein Fehlalarm —
		// und ein Register mit Fehlalarmen wird nach dem zweiten Mal nicht mehr gelesen.
		quelle := musterSQLKomm.ReplaceAllString(musterZeilenKomm.ReplaceAllString(string(roh), ""), "")
		n := len(musterCurrentDate.FindAllString(quelle, -1)) + len(musterRoheUhr.FindAllString(quelle, -1))
		if n > 0 {
			rel := strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(pfad, wurzel)), "/")
			gefunden[rel] = n
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Baum lesen: %v", err)
	}

	if len(gefunden) == 0 {
		t.Fatal("kein einziger Fund im ganzen Baum — der Detektor misst nichts, und diese " +
			"Ratsche wäre ab sofort immer grün")
	}

	var neu, gewachsen, ausgetragen []string
	for pfad, n := range gefunden {
		erlaubt, bekannt := bestandKalendertag[pfad]
		switch {
		case !bekannt:
			neu = append(neu, pfad)
		case n > erlaubt.anzahl:
			gewachsen = append(gewachsen, pfad)
		}
	}
	for pfad, erlaubt := range bestandKalendertag {
		if gefunden[pfad] < erlaubt.anzahl {
			ausgetragen = append(ausgetragen, pfad)
		}
	}
	sort.Strings(neu)
	sort.Strings(gewachsen)
	sort.Strings(ausgetragen)

	for _, p := range neu {
		t.Errorf("%s rechnet neu mit dem Kalendertag der Sitzung (CURRENT_DATE) oder der "+
			"Container-Uhr (time.Now().Format/Year). Zwischen Mitternacht in Berlin und "+
			"Mitternacht UTC ist das der VORTAG. Entweder schulzeit.SQLHeute bzw. "+
			"schulzeit.Jetzt() benutzen — oder die Stelle mit Begründung in "+
			"bestandKalendertag eintragen.", p)
	}
	for _, p := range gewachsen {
		t.Errorf("%s hat mehr solche Stellen als geduldet (%d statt %d). Eine Ausnahme ist "+
			"kein Freibrief.", p, gefunden[p], bestandKalendertag[p].anzahl)
	}
	for _, p := range ausgetragen {
		t.Errorf("%s hat weniger solche Stellen als der Bestand behauptet (%d statt %d) — "+
			"bitte in bestandKalendertag nachziehen, damit die Ratsche weiter greift.",
			p, gefunden[p], bestandKalendertag[p].anzahl)
	}

	for pfad, eintrag := range bestandKalendertag {
		if eintrag.grund == "" {
			t.Errorf("%s steht ohne Grund im Bestand — eine Duldung ohne Begründung ist keine", pfad)
		}
	}
}
