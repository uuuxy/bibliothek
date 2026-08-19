package api

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestPrivilegierteRequestFelderRegistriert ist die Ratsche zur Bugklasse
// „privilegiertes Request-Feld" (override_block-Fund, 18.08.2026): Ein Feld im
// Request-Body hebt eine Regel auf, und der einzige Schutz ist, dass die
// Oberfläche das Feld nicht anbietet — der Server prüft kein Recht. So konnte
// ein HELFER per {"override_block":true} jede Schülersperre aushebeln.
//
// Der einmalige Rollen×Aktionen-Sweep hat den damaligen Bestand geprüft (drei
// Funde, alle gefixt). Diese Ratsche hält den Zustand: Jedes NEUE JSON-Feld,
// dessen Name nach Regel-Aufhebung klingt, muss hier mit dem Recht registriert
// werden, das es serverseitig verlangt — oder mit der Begründung, warum es
// unbedenklich ist. Wer das Feld einführt, beantwortet die Frage; nicht der,
// der den Bypass später ausbaden muss.
//
// Die Liste steht BEWUSST hier im Test, nicht in einer Sammeldatei — eine
// ausgelagerte Bestandsliste versteckt sich vor dem Gate (siehe die Lehre aus
// dem Verwaisten-Komponenten-Test des Frontends).
var registriertePrivilegFelder = map[string]string{
	"api/action_types.go:override_block":        "Gate an der HTTP-Grenze: nur mit BesitztRecht(edit_students), sonst wird das Feld ignoriert (154c1f85)",
	"api/mahnwesen_bulk_mail.go:override_email": "Route hinter create_orders (nur Mitarbeiter/Admin); Umleitung dokumentiert und namentlich auditiert",
	"api/lusd.go:skipped_no_id":                 "Antwort-Zähler, kein Request-Feld",
	"api/mahnwesen_bulk_mail.go:skipped_count":  "Antwort-Zähler, kein Request-Feld",
	"api/mahnwesen_bulk_mail.go:skipped":        "Antwort-Zähler, kein Request-Feld",
}

var (
	jsonTagMuster       = regexp.MustCompile(`json:"([a-zA-Z0-9_]+)`)
	privilegNamenMuster = regexp.MustCompile(`(?i)(override|force|bypass|skip|ignore|unlock)`)
)

func TestPrivilegierteRequestFelderRegistriert(t *testing.T) {
	wurzel := ".."
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
			if _, registriert := registriertePrivilegFelder[schluessel]; !registriert {
				t.Errorf("neues override-artiges JSON-Feld %q — prüfe: Welche Regel hebt es auf, und welches "+
					"Recht erzwingt der SERVER dafür (nicht die UI)? Dann in registriertePrivilegFelder "+
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
	for schluessel := range registriertePrivilegFelder {
		if !gefunden[schluessel] {
			t.Errorf("registriertes Feld %q existiert im Code nicht mehr — Eintrag entfernen", schluessel)
		}
	}
}
