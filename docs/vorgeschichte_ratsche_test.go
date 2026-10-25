package docs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Ratsche gegen Vorgeschichte in den Dokumenten.
//
// Ein Dokument nennt, was gilt; wie es dazu kam, steht in der Commit-Nachricht. Einschübe wie
// „seit dem 07.10.2026" oder „bis zum …" und Absätze mit Anlass und Nachtrag wachsen sonst mit
// jeder Änderung nach, bis die Regel zwischen ihrer Geschichte nicht mehr zu finden ist. Ein
// Datum bleibt, wo es eine Entscheidung, eine Quelle, eine Frist oder eine Messung begrenzt
// (PFLEGEKONZEPT.md 7.2).
//
// Gezählt werden je Dokument die Zeilen mit einem vollen Datum (TT.MM.JJJJ, JJJJ-MM-TT), in
// docs/ und den Dokumenten der Wurzel. Nicht gezählt werden die Kopfzeile mit dem Stand (sie
// hält stand_angaben_test.go) und das Datum einer Quelle, in derselben Form wie in
// stil_ratsche_test.go.
//
// Blindheit: Vorgeschichte ohne volles Datum („früher", „bis dahin", „im Juli") und ein zweites
// Datum in einer Zeile, die schon eines trägt. Die Ratsche zählt nur; ob ein Datum im Bestand
// Vorgeschichte ist, liest ein Mensch. OFFEN.md zählt sie mit Absicht nicht, .jules/ und
// bewertung/ liegen außerhalb.

// bestandVorgeschichte nennt je Dokument die Zeilen mit Datum und was sie sind. Die Zahl darf
// sinken, nie steigen; wer eine Zeile streicht, trägt die neue Zahl hier ein.
var bestandVorgeschichte = map[string]struct {
	anzahl int
	grund  string
}{
	"docs/ARCHITEKTUR.md":             {48, "Entscheidungen der ADR, Messungen mit Tag, der Stand einzelner Abschnitte"},
	"docs/FACHKONZEPT.md":             {48, "Entscheidungen, darunter die verworfenen Vorschläge in §20, und Messungen"},
	"docs/littera_schema_befund.md":   {20, "Zählungen an den Sicherungen und Exporten von Littera, Entscheidungen"},
	"docs/sweeps.md":                  {14, "Durchsichten über den Bestand und Messungen mit Tag"},
	"docs/PFLEGEKONZEPT.md":           {11, "Entscheidungen der Schule, die Messungen in Abschnitt 6"},
	"docs/invarianten.md":             {11, "Entscheidungen und Messungen zu einzelnen Invarianten"},
	"docs/mittel_konzept.md":          {10, "Entscheidungen der Schule, Antworten der Servicestelle"},
	"docs/SECURITY.md":                {8, "Entscheidungen, Proben am Mailserver, Fassung der Excel-Bibliothek"},
	"docs/SCRIPTS.md":                 {7, "Entscheidung, Ergebnisse der Generalprobe, Messungen, Grenze einer Reparatur"},
	"docs/LUSD.md":                    {5, "Entscheidungen, Prüfung der Feldliste, der Lauf in Schulgröße"},
	"docs/PII_MATRIX.de.md":           {5, "Erhebung, Zählung der Schreibrouten, Grenze gespeicherter Protokolle"},
	"docs/datenschutz/nachweis.md":    {5, "Entscheidung, Grenzen gespeicherter Protokolle, Rechtsquelle, Zählung"},
	"docs/resilience_and_recovery.md": {5, "Grenzen alter Sicherungen, Entscheidung, Probe des Totalverlusts"},
	"README.md":                       {1, "Messung des Umfangs"},
	"docs/DEPLOYMENT.md":              {2, "zwei Entscheidungen"},
	"docs/HANDBUCH.md":                {2, "ein Beispiel der Oberfläche, die Grenze einer Reparatur"},
	"docs/STIL.md":                    {1, "Beispiel für das Datum einer Quelle, über zwei Zeilen"},
	"docs/api_inventar.md":            {1, "Tag der Erzeugung durch scripts/api_inventar.sh"},
}

// ausgenommenVorgeschichte nennt die Dokumente, die die Ratsche nicht zählt, je mit Grund.
var ausgenommenVorgeschichte = map[string]string{
	"docs/OFFEN.md": "die Liste der offenen Arbeit: Ein Punkt trägt Fundtag und Messungen, bis er " +
		"erledigt ist und ganz herausfällt (CLAUDE.md, Abschnitt 4)",
}

var (
	vorgeschichteDatum  = regexp.MustCompile(`\b\d{1,2}\.\d{1,2}\.20\d{2}\b|\b20\d{2}-\d{2}-\d{2}\b`)
	vorgeschichteQuelle = regexp.MustCompile(`(Erlass|Beschluss|Gesetz|Verordnung) vom \d{1,2}\.\d{1,2}\.20\d{2}|` +
		`abgerufen am \d{1,2}\.\d{1,2}\.20\d{2}`)
	vorgeschichteKopf = regexp.MustCompile(`(?i)(Zuletzt aktualisiert|Stand):?\s*\**\s*(\d{4}-\d{2}-\d{2}|\d{2}\.\d{2}\.\d{4})`)
)

// vorgeschichteKopfZeilen ist der Bereich, in dem stand_angaben_test.go die Kopfzeile sucht.
const vorgeschichteKopfZeilen = 24

// zeilenMitDatum zählt die Zeilen eines Dokuments mit einem vollen Datum, ohne die erste
// Kopfzeile mit dem Stand und ohne das Datum einer Quelle.
func zeilenMitDatum(inhalt string) int {
	kopfGesehen := false
	n := 0
	for i, zeile := range strings.Split(inhalt, "\n") {
		if !kopfGesehen && i < vorgeschichteKopfZeilen && vorgeschichteKopf.MatchString(zeile) {
			kopfGesehen = true
			continue
		}
		if vorgeschichteDatum.MatchString(vorgeschichteQuelle.ReplaceAllString(zeile, "")) {
			n++
		}
	}
	return n
}

// dokumenteMitInhalt liest die Dokumente in docs/ und in der Wurzel, je unter ihrem Pfad ab der
// Wurzel des Repositorys.
func dokumenteMitInhalt(t *testing.T) map[string]string {
	t.Helper()
	dokumente := map[string]string{}
	inDocs := 0
	err := filepath.WalkDir(".", func(pfad string, eintrag fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if eintrag.IsDir() || !strings.HasSuffix(pfad, ".md") {
			return nil
		}
		roh, err := os.ReadFile(pfad) //nolint:gosec // Dokumente aus dem Repo
		if err != nil {
			return err
		}
		dokumente["docs/"+filepath.ToSlash(pfad)] = string(roh)
		inDocs++
		return nil
	})
	if err != nil {
		t.Fatalf("docs/ lesen: %v", err)
	}
	wurzel, err := filepath.Glob("../*.md")
	if err != nil {
		t.Fatalf("Wurzel lesen: %v", err)
	}
	for _, pfad := range wurzel {
		roh, err := os.ReadFile(pfad) //nolint:gosec // Dokumente aus dem Repo
		if err != nil {
			t.Fatalf("%s lesen: %v", pfad, err)
		}
		dokumente[filepath.Base(pfad)] = string(roh)
	}
	// Nicht-leer-Garantie je Quelle: Greift ein Sammler ins Leere, wäre die Ratsche still grün.
	if inDocs < 10 || len(wurzel) < 2 {
		t.Fatalf("nur %d Dokumente in docs/ und %d in der Wurzel gefunden — der Sammler greift ins Leere",
			inDocs, len(wurzel))
	}
	return dokumente
}

func TestVorgeschichte_ZeilenMitDatumWachsenNicht(t *testing.T) {
	dokumente := dokumenteMitInhalt(t)

	gezaehlt := map[string]int{}
	summe := 0
	for pfad, inhalt := range dokumente {
		if _, aus := ausgenommenVorgeschichte[pfad]; aus {
			continue
		}
		if n := zeilenMitDatum(inhalt); n > 0 {
			gezaehlt[pfad] = n
			summe += n
		}
	}
	if summe == 0 {
		t.Fatal("keine Zeile mit Datum in allen Dokumenten — der Detektor misst nichts")
	}

	var meldungen []string
	for pfad, n := range gezaehlt {
		bestand, bekannt := bestandVorgeschichte[pfad]
		switch {
		case !bekannt:
			meldungen = append(meldungen, fmt.Sprintf("%s: %s mit Datum, das Dokument steht nicht im Bestand. "+
				"Ein Dokument nennt, was gilt; wie es dazu kam, gehört in die Commit-Nachricht. Begrenzt das Datum "+
				"eine Entscheidung, eine Quelle, eine Frist oder eine Messung, das Dokument mit Zahl und Grund in "+
				"bestandVorgeschichte eintragen.", pfad, zeilenWort(n)))
		case n > bestand.anzahl:
			meldungen = append(meldungen, fmt.Sprintf("%s: %s mit Datum statt %d. Vorgeschichte („seit dem …“, "+
				"„bis zum …“, Anlass, Nachtrag) gehört in die Commit-Nachricht. Begrenzt das neue Datum eine "+
				"Entscheidung, eine Quelle, eine Frist oder eine Messung, die Zahl erhöhen und den Grund in die "+
				"Commit-Nachricht schreiben.", pfad, zeilenWort(n), bestand.anzahl))
		case n < bestand.anzahl:
			meldungen = append(meldungen, fmt.Sprintf("%s: nur noch %s mit Datum statt %d — die Zahl in "+
				"bestandVorgeschichte nachziehen, damit die Ratsche weiter greift.", pfad, zeilenWort(n), bestand.anzahl))
		}
	}
	for pfad, bestand := range bestandVorgeschichte {
		if _, gibt := dokumente[pfad]; !gibt {
			meldungen = append(meldungen, pfad+" steht im Bestand, das Dokument gibt es nicht mehr — austragen.")
			continue
		}
		if gezaehlt[pfad] == 0 {
			meldungen = append(meldungen, pfad+" hat keine Zeile mit Datum mehr — aus bestandVorgeschichte austragen.")
		}
		if bestand.grund == "" {
			meldungen = append(meldungen, pfad+" steht ohne Grund im Bestand.")
		}
	}
	for pfad := range ausgenommenVorgeschichte {
		if _, gibt := dokumente[pfad]; !gibt {
			meldungen = append(meldungen, pfad+" ist ausgenommen, das Dokument gibt es nicht mehr — austragen.")
		}
	}
	sort.Strings(meldungen)
	for _, m := range meldungen {
		t.Error(m)
	}
}

// Selbstprobe über die Formen, die gezählt werden, und die, die es nicht werden.
func TestVorgeschichte_Selbstprobe(t *testing.T) {
	// Ohne Kopf davor: 30 Leerzeilen, damit kein Satz als Kopfzeile gilt.
	rumpf := strings.Repeat("\n", 30)
	zaehlt := []string{
		"Seit dem 07.10.2026 gilt die Regel.",
		"(Stand 21.09.2026) mitten im Text",
		"Stand: 01.10.2026, ein Abschnitt weit unten",
		"Nachtrag 28.09.2026: Die Übernahme schreibt …",
		"am 1.9.2026 behoben",
		"| Eintrag | 2026-10-07 |",
		"Vom 05.10. bis zum 07.10.2026 und am 08.10.2026",
		"Erlass vom 17.12.2014, umgesetzt seit dem 17.09.2026",
	}
	for _, zeile := range zaehlt {
		if n := zeilenMitDatum(rumpf + zeile); n != 1 {
			t.Errorf("%q zählt %s, erwartet 1", zeile, zeilenWort(n))
		}
	}
	zaehltNicht := []string{
		"die Arbeitshilfe zum Erlass vom 17.12.2014",
		"Beschluss vom 21.09.2022 für 2025 bis 2030",
		"kultus.hessen.de, abgerufen am 24.09.2026",
		"Stichtag 31.07., Bestandskartei zum 15.3. und 15.9.",
		"im Juli 2026, Katalog vom Juni 2026",
		"„Der Plan beginnt Donnerstag, 11.06.26 in der 3. Stunde\"",
		"`v2.11.1-0.20261003002531-6258dcebc4e2`",
		"Version 2.14.0, Abschnitt 11.1",
	}
	for _, zeile := range zaehltNicht {
		if n := zeilenMitDatum(rumpf + zeile); n != 0 {
			t.Errorf("%q zählt %s, erwartet 0", zeile, zeilenWort(n))
		}
	}
	// Die Kopfzeile zählt nicht, in beiden Schreibweisen; eine zweite im Kopfbereich schon.
	for _, kopf := range []string{"# Titel\n\nStand: 16.10.2026 (Entwurf)\n", "> Zuletzt aktualisiert: 2026-10-16\n"} {
		if n := zeilenMitDatum(kopf); n != 0 {
			t.Errorf("Kopfzeile %q zählt %d, erwartet 0", kopf, n)
		}
	}
	if n := zeilenMitDatum("Stand: 16.10.2026\n\nStand: 08.10.2026 des Abschnitts\n"); n != 1 {
		t.Errorf("eine zweite Stand-Zeile im Kopfbereich zählt %d, erwartet 1", n)
	}
}

// zeilenWort nennt eine Zahl von Zeilen mit dem passenden Wort.
func zeilenWort(n int) string {
	if n == 1 {
		return "1 Zeile"
	}
	return fmt.Sprintf("%d Zeilen", n)
}
