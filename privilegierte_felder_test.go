package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestPrivilegierteAnfrageFelderEingetragen ist die Ratsche zur Bugklasse „privilegiertes Feld
// der Anfrage": Ein Feld im Rumpf der Anfrage hebt eine Regel auf, und der einzige Schutz ist,
// dass die Oberfläche das Feld nicht anbietet; der Server prüft kein Recht. So könnte ein Helfer
// per {"override_block":true} jede Schülersperre aushebeln.
//
// Die Ratsche hält den Zustand: Jedes neue JSON-Feld, dessen Name nach der Aufhebung einer Regel
// klingt, muss hier mit dem Recht eingetragen werden, das es auf dem Server verlangt, oder mit
// der Begründung, warum es unbedenklich ist. Wer das Feld einführt, beantwortet die Frage, nicht
// der, der die Umgehung später ausbaden muss.
//
// Die Liste steht absichtlich hier im Test, nicht in einer Sammeldatei: Eine ausgelagerte Liste
// des Bestands versteckt sich vor der Prüfung.
var eingetragenePrivilegFelder = map[string]string{
	"api/theke/eingabe_anfrage_antwort.go:override_block": "Gate an der HTTP-Grenze: nur mit HatRecht(edit_students), sonst wird das Feld ignoriert (154c1f85)",
	"api/mahnwesen/mahnlisten_versand.go:override_email":  "Route hinter create_orders (nur Mitarbeiter/Admin); Umleitung dokumentiert und namentlich auditiert",
	"internal/lusd/lauf.go:skipped_no_id":                 "Antwort-Zähler, kein Request-Feld",
	"api/mahnwesen/mahnlisten_versand.go:skipped_count":   "Antwort-Zähler, kein Request-Feld",
	"api/mahnwesen/mahnlisten_versand.go:skipped":         "Antwort-Zähler, kein Request-Feld",
}

var (
	jsonTagMuster       = regexp.MustCompile(`json:"([a-zA-Z0-9_]+)`)
	privilegNamenMuster = regexp.MustCompile(`(?i)(override|force|bypass|skip|ignore|unlock)`)
)

func TestPrivilegierteAnfrageFelderEingetragen(t *testing.T) {
	wurzel := "."
	gefunden := map[string]bool{}

	err := filepath.Walk(wurzel, func(pfad string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case "frontend", "node_modules", ".git", "docs", "scripts", "migrations":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		inhalt, err := os.ReadFile(pfad)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(wurzel, pfad)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		for _, treffer := range jsonTagMuster.FindAllStringSubmatch(string(inhalt), -1) {
			name := treffer[1]
			if !privilegNamenMuster.MatchString(name) {
				continue
			}
			schluessel := rel + ":" + name
			gefunden[schluessel] = true
			if _, registriert := eingetragenePrivilegFelder[schluessel]; !registriert {
				t.Errorf("neues override-artiges JSON-Feld %q — prüfe: Welche Regel hebt es auf, und welches "+
					"Recht erzwingt der Server dafür (nicht die Oberfläche)? Dann in eingetragenePrivilegFelder "+
					"mit Recht bzw. Begründung eintragen (Bugklasse override_block, 18.08.2026).", schluessel)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Quelltext-Scan fehlgeschlagen: %v", err)
	}

	// Gegenrichtung: Ein registrierter Eintrag, den es im Code nicht mehr gibt, ist
	// Karteileiche — die Liste soll den Ist-Zustand beschreiben, nicht Geschichte.
	for schluessel := range eingetragenePrivilegFelder {
		if !gefunden[schluessel] {
			t.Errorf("registriertes Feld %q existiert im Code nicht mehr — Eintrag entfernen", schluessel)
		}
	}
}
