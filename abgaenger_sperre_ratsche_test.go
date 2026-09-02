package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Abgänger-Sperre: EIN Präfix, EINE Uhr (Rasterdurchgang 02.09.2026). Der Rückkehrer-
// Pfad des LUSD-Imports und das Zusammenführen erkennen die automatische Sperre am
// Präfix des Sperrgrunds; die Karenz vor der Anonymisierung rechnet mit abgaenger_seit.
// Die Versetzung schrieb ein anderes Präfix („Automatische" statt „Automatisierte") und
// stempelte die Uhr nicht — ein Ghost-Block, den nur ein Zwillingsvergleich fand.
// Zwei Regeln, mechanisch:
//  1. Das Präfix steht als Literal nur in repository/abgaenger_sperre.go.
//  2. Jedes SQL, das ist_abgaenger auf true setzt, stempelt im selben Statement abgaenger_seit.
func TestAbgaengerSperre_EinPraefixEineUhr(t *testing.T) {
	praefix := regexp.MustCompile(`Automati(sierte|sche) Abgänger-Sperre`)
	setztAbgaenger := regexp.MustCompile(`(?s)ist_abgaenger\s*=\s*(true|c\.is_graduating)`)
	rohstrings := regexp.MustCompile("(?s)`[^`]*`")
	verkettung := regexp.MustCompile("` *\\+ *[A-Za-z_.]+ *\\+ *`")
	var verstoesse []string
	err := filepath.WalkDir(".", func(pfad string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "frontend" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		inhalt, err := os.ReadFile(pfad) // #nosec G304 -- Repo-Dateien
		if err != nil {
			return err
		}
		pfad = filepath.ToSlash(pfad)
		if praefix.Match(inhalt) && pfad != "repository/abgaenger_sperre.go" {
			verstoesse = append(verstoesse, pfad+": Sperrgrund-Präfix als Literal — repository.AbgaengerSperrPraefix nutzen")
		}
		// Ein SQL, das eine Go-Konstante einbettet (` + repository.X + `), ist EIN Statement.
		zusammen := verkettung.ReplaceAllString(string(inhalt), "'K'")
		for _, s := range rohstrings.FindAllString(zusammen, -1) {
			for _, set := range setKlauseln(s) {
				if setztAbgaenger.MatchString(set) && !strings.Contains(set, "abgaenger_seit") {
					verstoesse = append(verstoesse, pfad+": setzt ist_abgaenger = true ohne abgaenger_seit zu stempeln")
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range verstoesse {
		t.Error(v)
	}
	// Selbstprobe: Der Detektor fasst beide Schreibformen — und lässt Lesezugriffe
	// (WHERE ist_abgaenger = true) in Ruhe.
	for _, form := range []string{"UPDATE schueler SET ist_abgaenger = true WHERE id = $1", "UPDATE schueler SET jahr = EXTRACT(YEAR FROM NOW()), ist_abgaenger = true WHERE id = $1", "UPDATE schueler s SET klasse = 'ABG',\n ist_abgaenger = c.is_graduating FROM calculated c WHERE s.id = c.id"} {
		if sets := setKlauseln(form); len(sets) != 1 || !setztAbgaenger.MatchString(sets[0]) {
			t.Errorf("Selbstprobe: Detektor fasst %q nicht", form)
		}
	}
	if sets := setKlauseln("UPDATE schueler SET klasse = 'x' WHERE ist_abgaenger = true"); len(sets) != 1 || setztAbgaenger.MatchString(sets[0]) {
		t.Error("Selbstprobe: Lesezugriff in WHERE darf nicht als Schreiber gelten")
	}
}

// setKlauseln schneidet aus einem SQL-Text jede SET-Klausel bis zum nächsten WHERE/FROM/
// RETURNING heraus — nur dort wird geschrieben; ein „ist_abgaenger = true" im WHERE ist
// ein Lesezugriff.
func setKlauseln(sql string) []string {
	// EXTRACT(YEAR FROM …) trägt ein FROM, das keine Klausel beendet.
	sql = regexp.MustCompile(`(?i)EXTRACT\([^)]*\)`).ReplaceAllString(sql, "EXTRACT()")
	setRe := regexp.MustCompile(`(?is)\bSET\b(.*?)(\bWHERE\b|\bFROM\b|\bRETURNING\b|$)`)
	var out []string
	for _, m := range setRe.FindAllStringSubmatch(sql, -1) {
		out = append(out, m[1])
	}
	return out
}
