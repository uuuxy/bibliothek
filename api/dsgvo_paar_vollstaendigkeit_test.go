package api

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Gate: Auskunft und Tilgung sind ein PAAR über derselben Tabellenliste.
//
// Anlass (31.08.2026): Beide Seiten zählten ihre Tabellen von Hand auf —
// api/dsgvo_auskunft.go die eine Liste, repository/audit_users.go die andere — und
// nichts hielt sie zusammen. Genau daher kam der Lesehistorie-Fund des
// Komplett-Durchgangs (Tilgung kannte tabelle='ausleihen' nicht), und die Gegenrichtung
// war ebenso offen: Die Auskunft kannte audit_logs nicht, obwohl dort Einträge mit
// details->>'schueler_id' liegen (LUSD_ID_NACHGETRAGEN, DELETE/RESTORE/PURGE_STUDENT).
// Eine Art.-15-Auskunft, die eine Tabelle nicht liest, ist unvollständig — und fällt
// niemandem auf, weil die Antwort trotzdem gut aussieht.
//
// dsgvoSchuelerQuellen ist seither DIE Liste: Der Quelltext-Scan unten erzwingt, dass
// die Auskunft jede Quelle liest; die Rundreise (dsgvo_paar_rundreise_pg_test.go)
// beweist am echten Postgres, dass die Tilgung jede Quelle leert; die FK-Ratsche
// (ebendort) wird rot, sobald das Schema eine neue Schüler-Referenz bekommt, die hier
// fehlt.
var dsgvoSchuelerQuellen = []struct {
	Tabelle string
	Bezug   string // wie die Tabelle auf den Schüler verweist (Doku + Fehlermeldung)
	MitFK   bool   // taucht im information_schema-Scan der FK-Ratsche auf
}{
	{"schueler", "id", false},
	{"schueler_fotos", "schueler_id (FK, ON DELETE CASCADE)", true},
	{"ausleihen", "schueler_id (FK, bei Tilgung -> NULL)", true},
	{"schadensfaelle", "schueler_id (FK, bei Tilgung geloescht)", true},
	{"vormerkungen", "schueler_id (FK, bei Tilgung geloescht)", true},
	{"audit_log", "datensatz_id (tabelle='schueler') bzw. details->>'schueler_id'", false},
	{"audit_logs", "details->>'schueler_id'", false},
}

// TestDsgvoAuskunftLiestJedeSchuelerQuelle prüft die Auskunfts-Hälfte des Paars am
// Quelltext: Jede Tabelle mit Schülerbezug muss in api/dsgvo_auskunft.go in einem
// FROM/JOIN stehen. Rot gesehen am Stand vor dem 31.08.2026 (audit_logs fehlte).
func TestDsgvoAuskunftLiestJedeSchuelerQuelle(t *testing.T) {
	quelltext, err := os.ReadFile("dsgvo_auskunft.go")
	if err != nil {
		t.Fatalf("dsgvo_auskunft.go lesen: %v", err)
	}

	for _, q := range dsgvoSchuelerQuellen {
		muster := regexp.MustCompile(`(?i)(FROM|JOIN)\s+` + q.Tabelle + `\b`)
		if !muster.Match(quelltext) {
			t.Errorf("die Art.-15-Auskunft liest %s nicht (Bezug: %s) — sammleDsgvoDaten "+
				"braucht eine Query über diese Tabelle, sonst ist die Auskunft unvollständig.",
				q.Tabelle, q.Bezug)
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
