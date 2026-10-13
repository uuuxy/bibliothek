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
// Eine dritte Form derselben Klasse (13.10.2026): Ein Datum aus einer Eingabe wird mit
// time.Parse als Tag in UTC gelesen und als Zeitpunkt gegen eine Spalte mit Zeitpunkten
// verglichen. Der Bestellbericht führte so eine Bestellung aus der ersten Stunde eines Monats
// im Monat davor. schulzeit.Kalendertag liest denselben Text als Tag der Schule.
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
	"cmd/migrate/pg_writer.go":               {1, "erworben_am beim Alt-Import"},
	"repository/statistik.go":                {5, "Statistik-Fenster (30 Tage, 12 Monate): verschiebt ein Ranking, keine Entscheidung; das Schuljahr dort rechnet bereits in der Schulzeitzone"},
	"cmd/seed/main.go":                       {1, "erfundene Abgangsjahre für Testdaten"},
	"internal/uebernahme/protokoll.go":       {1, "Zeitstempel einer Protokollzeile"},
	"internal/service/bestellung_anlegen.go": {1, "Anschaffungsjahr eines Exemplars"},
	// Ein Datum aus einer Eingabe, als Tag in UTC gelesen: geduldet, wo danach kein Zeitpunkt
	// in UTC verglichen wird.
	"internal/service/loan_rules.go":      {1, "Ferien-Leseclub: nimmt Jahr, Monat und Tag und baut daraus das Tagesende der Schule"},
	"api/theke/ausleihe.go":               {1, "Frist von Hand: geht durch TagesEnde in der Zone der Schule"},
	"api/lernmittel/lmf_verlaengerung.go": {1, "neues Rückgabedatum: geht durch TagesEnde in der Zone der Schule"},
	"api/lernmittel/lmf_plan.go":          {3, "prüft nur die Form der Tage; gerechnet wird mit schulzeit.Kalendertag"},
	"api/leser/student_create.go":         {1, "Geburtsdatum: trifft eine Datumsspalte"},
	"api/leser/student_update.go":         {1, "Geburtsdatum: trifft eine Datumsspalte"},
	"api/druck/etiketten_offen.go":        {2, "Stichtag der Etiketten: verglichen mit Datumsspalten"},
	"pkg/lmfplan/sommerferien.go":         {2, "Ferientage als Kalendertage in UTC, verglichen mit Tagen derselben Form (kalendertagUTC)"},
	"pkg/lmfplan/ferien_einstellung.go":   {2, "Ferientage als Kalendertage in UTC, verglichen mit Tagen derselben Form (kalendertagUTC)"},
}

var (
	musterCurrentDate = regexp.MustCompile(`\bCURRENT_DATE\b`)
	musterRoheUhr     = regexp.MustCompile(`time\.Now\(\)\.(Format|Year)\(`)
	musterDatumInUTC  = regexp.MustCompile(`time\.Parse\((tuer\.DateFormatISO|"2006-01-02"|time\.DateOnly),`)
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
		n := len(musterCurrentDate.FindAllString(quelle, -1)) + len(musterRoheUhr.FindAllString(quelle, -1)) +
			len(musterDatumInUTC.FindAllString(quelle, -1))
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
			"Container-Uhr (time.Now().Format/Year), oder liest ein Datum mit time.Parse als "+
			"Tag in UTC. Zwischen Mitternacht in Berlin und Mitternacht UTC ist das der VORTAG. "+
			"Entweder schulzeit.SQLHeute, schulzeit.Jetzt() bzw. schulzeit.Kalendertag "+
			"benutzen — oder die Stelle mit Begründung in bestandKalendertag eintragen.", p)
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
