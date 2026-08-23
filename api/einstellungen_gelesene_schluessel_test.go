package api

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"bibliothek/repository"
)

// Die dritte Richtung: Was der Server LIEST, muss auch einstellbar sein.
//
// Das Schwester-Gate (einstellungen_schluessel_paritaet_test.go) hält Oberfläche und
// Patch-Struct zusammen. Für einen Schlüssel, den nur der Server liest, ist es blind —
// und genau dort saß der Fund vom 23.08.2026: `audit_aufbewahrung_monate` wurde vom
// Aufbewahrungs-Job gelesen und im Dateikopf als einstellbar beschrieben ("Untergrenze
// 6"), stand aber in keiner Kategorie, keinem Struct, keinem Seed und keiner Migration.
// Kein Weg konnte ihn schreiben; die Frist war faktisch fest verdrahtet, und die
// Selbstprüfung meldete "Frist 24 Monate", als wäre sie konfiguriert.
func TestGeleseneEinstellungenSindEinstellbar(t *testing.T) {
	gelesen := schluesselAusUebernahme(t)
	// Gegenprobe am Detektor: Findet er überhaupt etwas? Ein Regex, der ins Leere
	// greift, meldet fröhlich "alles einstellbar".
	if len(gelesen) < 12 {
		t.Fatalf("nur %d gelesene Schlüssel gefunden — der Detektor greift ins Leere", len(gelesen))
	}
	imStruct := jsonTags(repository.EinstellungenPatch{})

	for _, k := range sortiereMenge(gelesen) {
		if !imStruct[k] {
			t.Errorf("der Server liest die Einstellung %q, aber repository.EinstellungenPatch "+
				"kennt sie nicht — niemand kann sie setzen, der Wert ist faktisch fest "+
				"verdrahtet. Entweder anbieten oder die Behauptung aus dem Code nehmen.", k)
		}
	}
}

// schluesselAusUebernahme sammelt die `case "..."`-Schlüssel der Funktionen, die einen
// gespeicherten Wert in ein Feld übernehmen. Bewusst NUR diese: Ein Detektor, der jedes
// Vorkommen einer Zeichenkette zählt, fände auch Kommentare und Testdaten.
func schluesselAusUebernahme(t *testing.T) map[string]bool {
	t.Helper()
	caseSchluessel := regexp.MustCompile(`case\s+"([a-z_]+)"`)

	menge := map[string]bool{}
	for _, q := range []string{
		filepath.Join("..", "internal", "service", "loan_rules.go"),
		filepath.Join("..", "repository", "system_settings_datenschutz.go"),
		filepath.Join("..", "repository", "system_settings.go"),
	} {
		roh, err := os.ReadFile(q)
		if err != nil {
			t.Fatalf("%s lesen: %v", q, err)
		}
		for _, m := range caseSchluessel.FindAllStringSubmatch(ohneKommentare(string(roh)), -1) {
			menge[m[1]] = true
		}
	}
	// Der Audit-Schlüssel steht als Konstante im case, nicht als Literal — über den Wert
	// dazu, sonst rutschte genau der Fall durch, der diesen Test veranlasst hat.
	menge[repository.AuditAufbewahrungSchluessel] = true
	return menge
}

// TestKeineNeuenRohenEinstellungsLeser friert die Stellen ein, die an den Einstellungen
// VORBEI direkt in system_einstellungen greifen.
//
// Warum als Ratsche und nicht als Verbot: Die fünf bekannten Stellen sind berechtigt —
// zwei davon lesen einen Wert, den das Programm selbst geschrieben hat (Restore-Probe,
// Ausweis-Layout), die anderen sind die Einstellungs-Schicht selbst. Aber genau auf
// diesem Weg entsteht die tote Einstellung: Wer künftig roh liest, umgeht das Gate
// darüber, weil sein Schlüssel in keinem `case` auftaucht. Die Liste darf schrumpfen,
// nicht wachsen.
func TestKeineNeuenRohenEinstellungsLeser(t *testing.T) {
	erlaubt := map[string]string{
		"repository/betriebszustand.go":  "LadeEinstellungswert — generisch, liest das Ergebnis der Restore-Probe",
		"repository/system_settings.go":  "die Einstellungs-Schicht selbst",
		"internal/service/loan_rules.go": "liest ALLE Zeilen und bildet sie über applyEinstellung ab (vom Gate oben erfasst)",
		"api/ausweis_layout.go":          "eigener Endpunkt, schreibt und liest dieselbe Zeile",
		"jobs/restore_probe.go":          "schreibt und liest sein eigenes Ergebnis",
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
		t.Fatal("keine einzige Lesestelle gefunden — der Detektor greift ins Leere")
	}

	var neue []string
	for datei := range gefunden {
		if _, ok := erlaubt[datei]; !ok {
			neue = append(neue, datei)
		}
	}
	sort.Strings(neue)
	for _, datei := range neue {
		t.Errorf("%s liest roh aus system_einstellungen. Damit umgeht der Schlüssel das "+
			"Gate darüber und kann eine Einstellung werden, die niemand einstellen kann. "+
			"Über repository.SystemSettingsRepository lesen — oder hier mit Begründung "+
			"eintragen.", datei)
	}
	// Die Gegenrichtung: Ein Eintrag, der nicht mehr zutrifft, ist eine Ausnahme für
	// etwas, das es nicht gibt — und verdeckt beim nächsten Mal einen echten Fall.
	for datei, grund := range erlaubt {
		if !gefunden[datei] {
			t.Errorf("%s steht als Ausnahme (%q), liest aber gar nicht mehr roh — "+
				"Eintrag entfernen.", datei, grund)
		}
	}
}
