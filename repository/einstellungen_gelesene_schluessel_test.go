package repository_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"bibliothek/repository"
)

// Was der Server liest, ist auch einstellbar.
//
// Die Schwesterprüfung (einstellungen_schluessel_paritaet_test.go) hält Oberfläche und Patch
// zusammen. Für einen Schlüssel, den nur der Server liest, ist sie blind: Der nächtliche Lauf
// las `audit_aufbewahrung_monate`, der Schlüssel stand aber in keiner Kategorie, keinem Typ
// und keiner Migration. Kein Weg konnte ihn schreiben, die Frist stand fest, und die
// Selbstprüfung meldete „Frist 24 Monate", als wäre sie eingestellt.
func TestGeleseneEinstellungenSindEinstellbar(t *testing.T) {
	gelesen := schluesselAusUebernahme(t)
	// Gegenprobe am Detektor: Ein Muster, das ins Leere greift, meldete „alles einstellbar".
	if len(gelesen) < 12 {
		t.Fatalf("nur %d gelesene Schlüssel gefunden: Der Detektor greift ins Leere", len(gelesen))
	}
	imStruct := jsonTags(repository.EinstellungenPatch{})

	for _, k := range sortiereMenge(gelesen) {
		if !imStruct[k] {
			t.Errorf("der Server liest die Einstellung %q, aber repository.EinstellungenPatch "+
				"kennt sie nicht: Niemand kann sie setzen, der Wert steht fest. "+
				"Entweder anbieten oder die Behauptung aus dem Code nehmen.", k)
		}
	}
}

// schluesselAusUebernahme sammelt die Schlüssel aus den Zweigen `case "..."` der Funktionen,
// die einen gespeicherten Wert in ein Feld übernehmen, und nur diese: Ein Detektor, der jede
// Zeichenkette zählt, fände auch Kommentare und Testdaten.
func schluesselAusUebernahme(t *testing.T) map[string]bool {
	t.Helper()
	caseSchluessel := regexp.MustCompile(`case\s+"([a-z_]+)"`)

	menge := map[string]bool{}
	for _, q := range []string{
		filepath.Join("..", "repository", "einstellungen_datenschutz.go"),
		filepath.Join("..", "repository", "einstellungen_bescheid.go"),
		filepath.Join("..", "repository", "einstellungen.go"),
	} {
		roh, err := os.ReadFile(q)
		if err != nil {
			t.Fatalf("%s lesen: %v", q, err)
		}
		for _, m := range caseSchluessel.FindAllStringSubmatch(ohneKommentare(string(roh)), -1) {
			menge[m[1]] = true
		}
	}
	// Der Schlüssel für die Aufbewahrung des Protokolls steht als Konstante im case und kommt
	// deshalb über ihren Wert dazu; sonst fehlte der Fall, für den es diesen Test gibt.
	menge[repository.ProtokollAufbewahrungSchluessel] = true
	return menge
}

// TestKeineNeuenRohenEinstellungsLeser hält die Stellen fest, die an den Einstellungen vorbei
// in system_einstellungen greifen.
//
// Eine Liste statt eines Verbots, weil die bekannten Stellen berechtigt sind: Zwei lesen einen
// Wert, den das Programm selbst geschrieben hat (Restore-Probe, Ausweis-Layout), eine ist das
// Lesen der Einstellungen selbst. Auf diesem Weg entsteht aber die Einstellung, die niemand
// einstellen kann: Wer roh liest, umgeht die Prüfung darüber, weil sein Schlüssel in keinem
// `case` steht. Die Liste schrumpft nur.
func TestKeineNeuenRohenEinstellungsLeser(t *testing.T) {
	erlaubt := map[string]string{
		"repository/betriebszustand.go":    "LadeEinstellungswert: liest das Ergebnis der Restore-Probe für die Selbstprüfung und für den Start",
		"repository/einstellungen.go":      "das Lesen der Einstellungen selbst",
		"repository/ausweis_layout.go":     "eigene Tür (api/druck/ausweis_layout.go), schreibt und liest dieselbe Zeile",
		"internal/testdaten/bestellung.go": "Prüfhilfe der Tests: setzt und entfernt die öffentliche Adresse",
		"internal/testdaten/etikett.go":    "Prüfhilfe der Tests: setzt und entfernt den Eigentumsvermerk der Schülerbücherei",
		"internal/testdaten/schaden.go":    "Prüfhilfe der Tests: setzt und entfernt die Pflichtangaben des Bescheids",
		"internal/testweg/lesefehler.go":   "Prüfhilfe der Tests: nennt die Abfrage der Einstellungen, um sie scheitern zu lassen; liest nichts",
	}

	gefunden := map[string]bool{}
	wurzel := ".."
	err := filepath.Walk(wurzel, func(pfad string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(pfad, ".go") ||
			strings.HasSuffix(pfad, "_test.go") || strings.Contains(pfad, "node_modules") {
			return nil
		}
		roh, leseErr := os.ReadFile(pfad)
		if leseErr != nil {
			return nil
		}
		if strings.Contains(ohneKommentare(string(roh)), "FROM system_einstellungen") {
			rel, relErr := filepath.Rel(wurzel, pfad)
			if relErr != nil {
				return nil
			}
			gefunden[filepath.ToSlash(rel)] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Baum durchlaufen: %v", err)
	}
	if len(gefunden) == 0 {
		t.Fatal("keine einzige Lesestelle gefunden: Der Detektor greift ins Leere")
	}

	var neue []string
	for datei := range gefunden {
		if _, ok := erlaubt[datei]; !ok {
			neue = append(neue, datei)
		}
	}
	sort.Strings(neue)
	for _, datei := range neue {
		t.Errorf("%s liest roh aus system_einstellungen. Damit umgeht der Schlüssel die "+
			"Prüfung darüber und kann eine Einstellung werden, die niemand einstellen kann. "+
			"Über repository.EinstellungsAbfragen lesen oder hier mit Begründung "+
			"eintragen.", datei)
	}
	// Die Gegenrichtung: Ein Eintrag, der nicht mehr zutrifft, ist eine Ausnahme für etwas,
	// das es nicht gibt, und verdeckt beim nächsten Mal einen echten Fall.
	for datei, grund := range erlaubt {
		if !gefunden[datei] {
			t.Errorf("%s steht als Ausnahme (%q), liest aber nicht mehr roh: "+
				"Eintrag entfernen.", datei, grund)
		}
	}
}

// ohneKommentare nimmt aus einem Quelltext, was hinter „//" steht: Ein Satz, der einen Schlüssel
// erklärt, ist keine Lesestelle.
func ohneKommentare(quelle string) string {
	var b strings.Builder
	for line := range strings.Lines(quelle) {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}
