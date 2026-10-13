package api

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"bibliothek/internal/testdaten"
)

// dsgvoAuskunftAbfragen ist die Datei, in der die Abfragen der Auskunft am Leser stehen.
const dsgvoAuskunftAbfragen = "../repository/dsgvo_auskunft.go"

// TestDsgvoAuskunftLiestJedeSchuelerQuelle prüft die Auskunfts-Hälfte des Paars am
// Quelltext: Jede Tabelle mit Schülerbezug muss in repository/dsgvo_auskunft.go in einem
// FROM/JOIN stehen.
func TestDsgvoAuskunftLiestJedeSchuelerQuelle(t *testing.T) {
	quelltext, err := os.ReadFile(dsgvoAuskunftAbfragen)
	if err != nil {
		t.Fatalf("%s lesen: %v", dsgvoAuskunftAbfragen, err)
	}

	for _, q := range testdaten.DsgvoSchuelerQuellen {
		muster := regexp.MustCompile(`(?i)(FROM|JOIN)\s+` + q.Tabelle + `\b`)
		if !muster.Match(quelltext) {
			t.Errorf("die Art.-15-Auskunft liest %s nicht (Bezug: %s) — %s braucht eine "+
				"Abfrage über diese Tabelle, und sammleDsgvoDaten muss sie aufrufen, sonst ist die "+
				"Auskunft unvollständig.", q.Tabelle, q.Bezug, dsgvoAuskunftAbfragen)
		}
	}
}

// TestAuditDetails_EinSchluesselFuerDenSchueler friert das Schlüssel-Vokabular ein:
// Der Verweis auf einen Schüler in Audit-Details heißt `schueler_id` — überall.
//
// Bis zum 31.08.2026 schrieben drei Stellen (DELETE/RESTORE/PURGE_STUDENT) denselben
// Wert als `student_id`. Folge: Jede Abfrage über details->>'schueler_id' — die
// Auskunft ebenso wie die Tilgung — sah diese Einträge nicht. Zwei Namen für denselben
// Wert sind zwei Wahrheitsquellen; Migration 091 hat die Altzeilen umgeschlüsselt.
func TestAuditDetails_EinSchluesselFuerDenSchueler(t *testing.T) {
	var verstoesse []string
	schuelerIDForm := 0

	err := filepath.WalkDir("..", func(pfad string, eintrag fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if eintrag.IsDir() {
			name := eintrag.Name()
			if name == "node_modules" || name == "frontend" || name == "testdata" || name == ".git" {
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
		if strings.Contains(string(inhalt), `{"student_id"`) {
			verstoesse = append(verstoesse, pfad)
		}
		schuelerIDForm += strings.Count(string(inhalt), `{"schueler_id"`)
		return nil
	})
	if err != nil {
		t.Fatalf("Quelltext durchlaufen: %v", err)
	}

	for _, pfad := range verstoesse {
		t.Errorf("%s schreibt den Audit-Detail-Schlüssel `student_id` — das Vokabular ist "+
			"`schueler_id` (Auskunft und Tilgung fragen genau diesen Schlüssel ab).", pfad)
	}
	// Liveness: Verschwindet die Inline-JSON-Form ganz (Refactoring auf Maps o. Ä.),
	// prüft der student_id-Scan womöglich eine Form, die es nicht mehr gibt — dann
	// gehört dieses Gate nachgezogen statt still grün.
	if schuelerIDForm < 3 {
		t.Fatalf("nur %d Inline-Schreibstellen mit `{\"schueler_id\"` gefunden (erwartet >= 3) — "+
			"die Schreibform hat sich geändert, Detektor nachziehen.", schuelerIDForm)
	}
}

// Eine Quelle ohne Fremdschlüssel findet keine Abfrage am Schema. Fällt sie aus der Liste,
// prüfen Auskunft und Tilgung diese Tabelle nicht mehr, und die übrigen Prüfungen bleiben grün.
func TestDsgvoQuellenOhneFremdschluessel_BleibenInDerListe(t *testing.T) {
	ohneFremdschluessel := map[string]bool{}
	for _, q := range testdaten.DsgvoSchuelerQuellen {
		if !q.MitFK {
			ohneFremdschluessel[q.Tabelle] = true
		}
	}
	for _, tabelle := range []string{"leser", "audit_log", "audit_logs"} {
		if !ohneFremdschluessel[tabelle] {
			t.Errorf("testdaten.DsgvoSchuelerQuellen führt %s nicht mehr als Quelle ohne Fremdschlüssel: "+
				"Auskunft und Tilgung prüft an dieser Tabelle dann nichts mehr", tabelle)
		}
	}
}
