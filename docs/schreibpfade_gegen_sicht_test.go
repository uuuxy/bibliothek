package docs

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Der Anker zu Raster-Frage 13: „Bedeutungswechsel unter gleichem Namen."
//
// `schueler` war bis Migration 124 eine Tabelle und ist seither eine SICHT auf `leser`
// mit `WHERE art = 'schueler'`. Der Name blieb, die Bedeutung nicht — und ein UPDATE
// gegen eine gefilterte Sicht trifft beim Kollegium null Zeilen. Wer das nicht weiß,
// schreibt eine stille 404 (OFFEN.md 5.17/5.18).
//
// Beim Durchgang vom 16.09.2026 gab es 15 Schreibpfade gegen diese Sicht, heute sind es
// neun; die sechs, die gewandert sind, waren Zeile für Zeile die Funde jenes Durchgangs.
// Genau deshalb steht die Frage im Raster — und genau deshalb braucht sie diesen Anker:
// Eine Frage ohne Mechanik verrottet zum Spruch.
//
// Die Regel ist NICHT „gegen eine Sicht schreibt man nicht". Alle neun sind richtig: Sie
// führen die Schülerarbeit, und dort ist die Filterung der Sicht das Gewollte. Die Regel
// ist: **Eine zehnte Zeile ist eine Frage.** Wer eine hinzufügt, schreibt daneben, warum
// die Filterung für seinen Pfad richtig ist — oder merkt beim Schreiben, dass sie es
// nicht ist.

// schreibpfadeGegenSicht: je Datei die Anzahl der Schreibzugriffe auf die Sicht
// `schueler` und der Grund, warum die Sicht dort das Richtige ist.
var schreibpfadeGegenSicht = map[string]struct {
	anzahl int
	grund  string
}{
	"cmd/seed/main.go":                       {1, "Testdaten: Schüler anlegen"},
	"internal/littera/schreiber_personen.go": {1, "Littera-Schülerlauf; der Personenlauf des Kollegiums schreibt in benutzer"},
	"api/student_promotion.go":               {1, "Versetzung zum Schuljahresende — betrifft ausschließlich Klassen, also Schüler"},
	"api/lusd_apply.go":                      {5, "LUSD-Abgleich: anlegen, ändern, Abgang buchen — die LUSD kennt nur Schüler"},
	"jobs/cron_dsgvo.go":                     {1, "Anonymisierung der Abgänger; das Kollegium hat keine Abgangslogik"},
}

var musterSchreibpfad = regexp.MustCompile(`(?i)\b(INSERT\s+INTO|UPDATE|DELETE\s+FROM)\s+schueler\b`)

func TestSchreibpfadeGegenDieSicht_ZehnteZeileIstEineFrage(t *testing.T) {
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
		quelle := musterZeilenKomm.ReplaceAllString(string(roh), "")
		// `schueler_fotos` ist eine eigene TABELLE und nicht die Sicht — die Wortgrenze
		// im Muster trennt beide.
		if n := len(musterSchreibpfad.FindAllString(quelle, -1)); n > 0 {
			rel := strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(pfad, wurzel)), "/")
			gefunden[rel] = n
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Baum lesen: %v", err)
	}
	if len(gefunden) == 0 {
		t.Fatal("kein einziger Schreibpfad gefunden — der Detektor misst nichts; " +
			"heißt die Sicht noch `schueler`?")
	}

	var neu, gewachsen, veraltet []string
	for pfad, n := range gefunden {
		eintrag, bekannt := schreibpfadeGegenSicht[pfad]
		switch {
		case !bekannt:
			neu = append(neu, pfad)
		case n > eintrag.anzahl:
			gewachsen = append(gewachsen, pfad)
		}
	}
	for pfad, eintrag := range schreibpfadeGegenSicht {
		if gefunden[pfad] < eintrag.anzahl {
			veraltet = append(veraltet, pfad)
		}
		if eintrag.grund == "" {
			t.Errorf("%s steht ohne Begründung im Bestand", pfad)
		}
	}
	sort.Strings(neu)
	sort.Strings(gewachsen)
	sort.Strings(veraltet)

	for _, p := range neu {
		t.Errorf("%s schreibt neu gegen die SICHT `schueler` (gefiltert auf art = 'schueler'). "+
			"Beim Kollegium trifft so ein Schreibzugriff null Zeilen — eine stille 404. Wenn "+
			"die Filterung hier richtig ist, bitte mit diesem Satz in schreibpfadeGegenSicht "+
			"eintragen; wenn nicht, gehört der Pfad an die Tabelle `leser`.", p)
	}
	for _, p := range gewachsen {
		t.Errorf("%s hat mehr Schreibpfade gegen die Sicht als eingetragen (%d statt %d) — "+
			"jeder einzelne ist eine eigene Frage.", p, gefunden[p], schreibpfadeGegenSicht[p].anzahl)
	}
	for _, p := range veraltet {
		t.Errorf("%s hat weniger Schreibpfade als eingetragen (%d statt %d) — bitte den "+
			"Bestand nachziehen, damit er weiter etwas aussagt.", p, gefunden[p], schreibpfadeGegenSicht[p].anzahl)
	}
}
