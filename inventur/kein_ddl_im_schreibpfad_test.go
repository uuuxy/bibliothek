package inventur

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Gate gegen Schema-Anweisungen (DDL) im Anwendungscode.
//
// Anlass (07.09.2026): `sys_barcode_seq` — der Nummernkreis der automatisch erzeugten
// Exemplare („SYS-…") — stand in KEINER Migration und in keiner Zeile von schema.sql.
// Sie entstand ausschließlich per `CREATE SEQUENCE IF NOT EXISTS` mitten im
// Schreibpfad, an zwei Stellen (Bestandskorrektur und Sammelimport), jeweils INNERHALB
// der Transaktion, die das Buch anlegt.
//
// Zwei Schäden, die zusammengehören:
//
//  1. Am echten Postgres gemessen: Solange die Sequenz noch nicht existiert, hält dieses
//     DDL eine Sperre bis zum Commit — eine zweite Transaktion, die dasselbe versucht,
//     wartet die volle Laufzeit der ersten ab (gemessen: 3,06 s bei 4 s Vorlauf).
//     Existiert die Sequenz, kostet es nichts (0,11 s). Der Preis fällt also genau
//     einmal an: beim ersten Buch mit Bestand nach einer frischen Installation — dort,
//     wo niemand ihn erwartet und der Client nach 10 s abbricht (frontend/src/lib/apiFetch.js).
//
//  2. Die Schema-Paritäts-Ratsche (db/migrations_schema_paritaet_pg_test.go) vergleicht
//     Sequenzen ausdrücklich mit — und schwieg trotzdem. Sie kann so etwas prinzipiell
//     nicht sehen: Sie hält den gewachsenen gegen den frischen Weg, und in BEIDEN fehlt
//     ein Objekt, das erst der laufende Anwendungscode erzeugt. Ein Schema-Objekt, das
//     keiner der beiden Wege kennt, ist für jeden Vergleich der beiden unsichtbar.
//
// Deshalb prüft dieses Gate die Quelle statt das Ergebnis: Wer eine Struktur braucht,
// legt sie in einer Migration an (und zieht schema.sql nach, wie es die Paritäts-Ratsche
// verlangt). Der Schreibpfad benutzt sie nur.
//
// Reparatur bei Rot: Die gemeldete Anweisung aus dem Go-Code entfernen und das Objekt in
// migrations/ + schema.sql anlegen. Gibt es einen echten Grund für DDL zur Laufzeit,
// gehört er als Kommentar an die Ausnahmeliste unten — nicht in die Ratsche selbst.
func TestKeinDDLImSchreibpfad(t *testing.T) {
	// Anweisungen, die Struktur ändern. ALTER/DROP stehen mit drin, obwohl sie heute
	// nirgends vorkommen: Der Fund war ein CREATE, die Klasse ist größer.
	ddl := regexp.MustCompile(`(?i)\b(CREATE|ALTER|DROP)\s+(SEQUENCE|TABLE|INDEX|VIEW|TYPE|SCHEMA|FUNCTION|TRIGGER)\b`)

	// Die fachlichen Schreibpfade UND der Boot (db/). Draußen bleibt nur cmd/ — Werkzeuge
	// wie die Restore-Probe bauen absichtlich ganze Schemata. db/ stand bis zum 07.09.2026
	// mit der Begründung „stellt das Schema her" außen vor; genau dort lagen aber die
	// nächsten Funde derselben Klasse: db/seed.go legte role_permissions, lieferanten,
	// die pg_trgm-Extension und fünf GIN-Indexe beim Start an — role_permissions in
	// KEINER Migration und keiner Zeile von schema.sql (Migration 106).
	wurzeln := []string{".", "../api", "../repository", "../jobs", "../db"}

	// Einzelne Dateien, die DDL ausführen DÜRFEN — mit dem Grund daneben, damit die
	// Liste nicht zur Ratschen-Lockerung verkommt (jede Erweiterung braucht einen Satz).
	ausnahmen := map[string]string{
		"db/migrations.go": "der Runner selbst: schema_migrations ist die Tabelle, die es VOR jeder Migration geben muss",
	}

	var funde []string
	for _, wurzel := range wurzeln {
		eintraege, err := os.ReadDir(wurzel)
		if err != nil {
			t.Fatalf("%s: %v", wurzel, err)
		}
		for _, e := range eintraege {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			pfad := filepath.Join(wurzel, name)
			if _, erlaubt := ausnahmen[filepath.Join(filepath.Base(wurzel), name)]; erlaubt {
				continue
			}
			inhalt, err := os.ReadFile(pfad) //nolint:gosec // feste Repo-Pfade
			if err != nil {
				t.Fatalf("%s: %v", pfad, err)
			}
			for nr, zeile := range strings.Split(string(inhalt), "\n") {
				// Kommentare erklären das Gate selbst und die Historie — sie sind der
				// häufigste Fehlalarm dieser Bauart (siehe „lügende Ratsche durch
				// Kommentar", 05.09.2026: toContain fand den String im Fließtext).
				if strings.HasPrefix(strings.TrimSpace(zeile), "//") {
					continue
				}
				if ddl.MatchString(zeile) {
					funde = append(funde, filepath.Join(filepath.Base(wurzel), name)+":"+
						itoa(nr+1)+": "+strings.TrimSpace(zeile))
				}
			}
		}
	}

	if len(funde) > 0 {
		t.Errorf("DDL im Anwendungscode — gehört in eine Migration + schema.sql:\n%s",
			strings.Join(funde, "\n"))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
