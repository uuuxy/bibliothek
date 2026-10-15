package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Append-only auf audit_log/audit_logs ist kein Trigger, sondern Konvention: Die Tilgung nach
// DSGVO braucht die Schreibtür. Eine Konvention ohne Ratsche ist keine; dieser Test hält die
// Liste der Dateien fest, die audit_log(s) verändern oder löschen dürfen. Wer eine neue Tür baut,
// trägt sie hier ein und begründet sie im Commit.
func TestProtokollSchreibtueren_NurBekannteDateien(t *testing.T) {
	erlaubt := map[string]string{
		// Auch der Lauf nach DSGVO (jobs/cron_dsgvo.go) fährt diese Statements, über
		// repository.SpurTilgungen, die eine Liste; er hat keine eigene Tür.
		"repository/spur_tilgung.go": "SpurTilgungen (DSGVO: Purge + LUSD-Anonymisierung + Cron)",
		// Die Anweisungen, die SpurTilgungen aus den Schlüssellisten baut: Personenbezug neben
		// der Kennung eines Lesers und in den Einträgen seiner früheren Zugangskonten.
		"repository/protokoll_personenbezug.go": "Anweisungen der SpurTilgungen (Schlüssel mit Personenbezug)",
		// Die Anweisungen der Nachtläufe: Leser aus dem Protokoll der Ausleihen und aus den
		// Spuren gelöschter Vormerkungen nach der Frist der Lesehistorie, und das Löschen nach
		// der Aufbewahrungsfrist beider Protokolle.
		"repository/nachtlauf.go": "Lesehistorie-Frist und Aufbewahrung der Protokolle",
		// Zusammenführen zweier Schülerdatensätze: Die Protokollspuren der aufgelösten Quelle
		// (details->>'schueler_id', datensatz_id) werden auf das Ziel umgeschlüsselt, nicht gelöscht
		// oder geändert; sonst verlöre die Auskunft nach Art. 15 die Lesehistorie, und die Tilgung nach
		// DSGVO fände sie nicht mehr (dieselben Schlüssel wie SpurTilgungen). Aussage und Zeitstempel
		// der Einträge bleiben.
		"repository/leser_zusammenfuehren.go": "verschiebeVorgaenge (Umschlüsselung beim Zusammenführen)",
	}
	muster := regexp.MustCompile(`(?i)\b(UPDATE|DELETE\s+FROM)\s+audit_logs?\b`)
	var verstoesse []string
	gefunden := map[string]bool{}
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
		if !muster.Match(inhalt) {
			return nil
		}
		gefunden[filepath.ToSlash(pfad)] = true
		if _, ok := erlaubt[filepath.ToSlash(pfad)]; !ok {
			verstoesse = append(verstoesse, pfad)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(verstoesse) > 0 {
		t.Fatalf("neue Schreibtür auf audit_log/audit_logs in %v — Append-only ist Konvention; "+
			"nur DSGVO-Tilgung und Aufbewahrung dürfen das. Begründen und in die Liste in diesem Test aufnehmen.", verstoesse)
	}
	// Ein Eintrag, dessen Datei keine solche Anweisung mehr trägt, deckte beim nächsten Mal
	// eine neue Tür in derselben Datei.
	for datei := range erlaubt {
		if !gefunden[datei] {
			t.Errorf("erlaubte Datei %s ändert oder löscht audit_log/audit_logs nicht mehr — Eintrag entfernen", datei)
		}
	}
}
