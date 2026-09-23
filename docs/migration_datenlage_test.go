package docs

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Der Anker zu Raster-Frage 14: „Datenlage — was steht in den Zeilen, auf die der Pfad
// trifft, gemessen am echten Bestand und nicht an der Seed-Datenbank?"
//
// Migration 135 („Klasse N wird Spanne N bis N", 22.09.2026) war an der lokalen Datenbank
// entworfen, in der ein einziger Testtitel eine Klasse trug. Auf dem Testserver trugen 153
// Titel eine, darunter mehrjährige Bände mit nur einem Jahr. Die Migration lief dort, wurde
// zurückgenommen und brauchte ein Reparaturskript aus der Vorab-Sicherung (7981e347,
// bf91c54d). Keine der dreizehn Fragen davor sah in die Zeilen; alle lesen Code und Schema.
//
// Die Regel: Eine neue Migration, die vorhandene Zeilen ändert, umdeutet oder Daten
// wegnimmt, nennt im Kopf ihre Messung am echten Bestand — ein Satz mit „gemessen" und dem
// Datum („Gemessen am Testserver am 23.09.2026: …") — oder, wo es nichts zu messen gibt,
// „Ohne Messung: <Grund>". Die Zeile beweist nicht, dass gemessen wurde; sie stellt die
// Frage dort, wo die Migration entsteht.
//
// BLINDHEIT: Nur migrations/*.sql ab Nummer 145; die älteren sind Bestand, ein Teil nennt
// seine Messung, ein Teil nicht. Nur Anweisungen in Textform: dynamisches SQL (EXECUTE
// format(…) in einem DO-Block) nicht. Eine Anweisung in einem DO-Block zählt, eine in einem
// Funktionskörper bewusst nicht — sie läuft erst später, nicht bei der Migration. Das
// Auffüllen einer neuen Spalte über ihren DEFAULT zählt nicht. Datenskripte außerhalb von
// migrations/ (scripts/*.sql, cmd/) und Importe aus Go sieht sie nicht. Ob die genannte
// Messung die richtige ist, liest ein Mensch.

// ersteGepruefteMigration: Ab hier gilt die Regel (Rasterdurchgang 23.09.2026).
const ersteGepruefteMigration = 145

// datenaenderungen: die Formen, mit denen eine Migration vorhandene Zeilen ändert, umdeutet
// oder Daten wegnimmt — geprüft am Text ohne Kommentare und ohne Funktionskörper.
var datenaenderungen = []struct {
	name   string
	muster *regexp.Regexp
}{
	{"UPDATE", regexp.MustCompile(`(?is)\bUPDATE\s+(ONLY\s+)?[a-z_][a-z0-9_."]*(\s+(AS\s+)?[a-z_][a-z0-9_]*)?\s+SET\b`)},
	{"DELETE", regexp.MustCompile(`(?is)\bDELETE\s+FROM\b`)},
	{"INSERT … SELECT", regexp.MustCompile(`(?is)\bINSERT\s+INTO\s+[a-z_][a-z0-9_."]*\s*(\([^)]*\))?\s*(SELECT|WITH)\b`)},
	{"ON CONFLICT DO UPDATE", regexp.MustCompile(`(?is)\bON\s+CONFLICT\b[^;]*?\bDO\s+UPDATE\b`)},
	{"DROP COLUMN", regexp.MustCompile(`(?is)\bDROP\s+COLUMN\b`)},
	{"DROP TABLE", regexp.MustCompile(`(?is)\bDROP\s+TABLE\b`)},
	{"ALTER COLUMN … TYPE", regexp.MustCompile(`(?is)\bALTER\s+COLUMN\s+[a-z_][a-z0-9_"]*\s+(SET\s+DATA\s+)?TYPE\b`)},
	{"RENAME VALUE", regexp.MustCompile(`(?is)\bALTER\s+TYPE\b[^;]*?\bRENAME\s+VALUE\b`)},
	{"TRUNCATE", regexp.MustCompile(`(?is)\bTRUNCATE\b`)},
}

var (
	blockKommentar   = regexp.MustCompile(`(?s)/\*.*?\*/`)
	zeilenKommentar  = regexp.MustCompile(`--[^\n]*`)
	dollarMarke      = regexp.MustCompile(`\$[A-Za-z_]*\$`)
	funktionsAnfang  = regexp.MustCompile(`(?is)\bCREATE\s+(OR\s+REPLACE\s+)?(FUNCTION|PROCEDURE)\b`)
	kommentarZeile   = regexp.MustCompile(`(?m)^\s*--\s?(.*)$`)
	messungGenannt   = regexp.MustCompile(`(?is)gemessen\b.{0,80}?\b\d{1,2}\.\d{1,2}\.\d{4}\b`)
	ohneMessungGrund = regexp.MustCompile(`(?i)ohne\s+messung:\s*\S`)
)

// ohneFunktionskoerper entfernt Kommentare und die Körper von CREATE FUNCTION/PROCEDURE.
// Ein DO-Block bleibt stehen: Er läuft bei der Migration.
func ohneFunktionskoerper(sql string) string {
	sql = zeilenKommentar.ReplaceAllString(blockKommentar.ReplaceAllString(sql, " "), " ")
	var aus strings.Builder
	rest := sql
	for {
		auf := dollarMarke.FindStringIndex(rest)
		if auf == nil {
			aus.WriteString(rest)
			return aus.String()
		}
		marke := rest[auf[0]:auf[1]]
		zu := strings.Index(rest[auf[1]:], marke)
		if zu < 0 {
			aus.WriteString(rest)
			return aus.String()
		}
		davor := rest[:auf[0]]
		koerper := rest[auf[1] : auf[1]+zu]
		// Die Anweisung, zu der der Körper gehört, beginnt nach dem letzten Semikolon davor.
		anweisung := davor[strings.LastIndex(davor, ";")+1:]
		aus.WriteString(davor)
		if !funktionsAnfang.MatchString(anweisung) {
			aus.WriteString(" " + koerper + " ")
		}
		rest = rest[auf[1]+zu+len(marke):]
	}
}

// datenaenderungIn nennt die erste Form, mit der die Migration vorhandene Daten ändert,
// oder "" — dann gibt es nichts zu messen.
func datenaenderungIn(sql string) string {
	text := ohneFunktionskoerper(sql)
	for _, d := range datenaenderungen {
		if d.muster.MatchString(text) {
			return d.name
		}
	}
	return ""
}

// messungIm sagt, ob der Kommentartext der Migration eine Messung mit Datum nennt oder
// begründet, warum es keine gibt. Kommentarzeilen werden zu einem Text verbunden: Ein Satz
// darf über den Zeilenumbruch laufen („Gemessen am Testserver am\n-- 22.09.2026").
func messungIm(sql string) bool {
	var zeilen []string
	for _, z := range kommentarZeile.FindAllStringSubmatch(sql, -1) {
		zeilen = append(zeilen, strings.TrimSpace(z[1]))
	}
	kommentar := strings.Join(zeilen, " ")
	return messungGenannt.MatchString(kommentar) || ohneMessungGrund.MatchString(kommentar)
}

func migrationsNummer(name string) (int, bool) {
	teil, _, ok := strings.Cut(name, "_")
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(teil)
	return n, err == nil
}

func TestMigrationDatenlage_NeueMigrationNenntIhreMessung(t *testing.T) {
	dateien, err := filepath.Glob("../migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	// Nicht-leer-Garantie: Findet der Sammler die Migrationen nicht, prüft er nichts.
	if len(dateien) < 100 {
		t.Fatalf("nur %d Migrationen gefunden — der Pfad stimmt nicht mehr", len(dateien))
	}
	sort.Strings(dateien)
	for _, pfad := range dateien {
		name := filepath.Base(pfad)
		nr, ok := migrationsNummer(name)
		if !ok || nr < ersteGepruefteMigration {
			continue
		}
		roh, err := os.ReadFile(pfad)
		if err != nil {
			t.Fatal(err)
		}
		sql := string(roh)
		if form := datenaenderungIn(sql); form != "" && !messungIm(sql) {
			t.Errorf("%s ändert vorhandene Daten (%s) und nennt keine Messung am echten Bestand. "+
				"Raster-Frage 14 (docs/invarianten.md): vorher lesend am Testserver messen und im Kopf "+
				"einen Satz mit „gemessen\" und dem Datum eintragen — oder „Ohne Messung: <Grund>\".", name, form)
		}
	}
}

// Die Selbstprobe über alle Formen (sweeps.md, Regel 6): jede Datenänderung, jede sichere
// Form daneben und beide Arten, die Messung zu nennen.
func TestMigrationDatenlage_Selbstprobe(t *testing.T) {
	aendern := map[string]string{
		"UPDATE":                        `UPDATE buecher_titel SET titel = titel;`,
		"update klein":                  `update leser set klasse = '05F1' where klasse = '5f1';`,
		"UPDATE mit Alias":              `UPDATE leser l SET barcode_id = NULL FROM x WHERE l.id = x.id;`,
		"UPDATE ONLY":                   `UPDATE ONLY leser SET art = 'schueler';`,
		"UPDATE in CTE":                 `WITH z AS (SELECT id FROM leser) UPDATE leser SET art = 'liv' WHERE id IN (SELECT id FROM z);`,
		"DELETE":                        `DELETE FROM klassen WHERE name = 'X';`,
		"INSERT … SELECT":               `INSERT INTO titel_schlagworte (titel_id, schlagwort_id) SELECT id, id FROM t;`,
		"INSERT … SELECT ohne Spalten":  `INSERT INTO kopie SELECT * FROM quelle;`,
		"INSERT … WITH":                 `INSERT INTO kopie (a) WITH q AS (SELECT 1) SELECT * FROM q;`,
		"Seed mit Überschreiben":        `INSERT INTO system_settings (k, v) VALUES ('a', 'b') ON CONFLICT (k) DO UPDATE SET v = EXCLUDED.v;`,
		"DROP COLUMN":                   `ALTER TABLE buecher_titel DROP COLUMN IF EXISTS ziel_jahrgang;`,
		"DROP TABLE":                    `DROP TABLE IF EXISTS benutzer_rollen;`,
		"ALTER COLUMN TYPE":             `ALTER TABLE leser ALTER COLUMN klasse TYPE varchar(10);`,
		"ALTER COLUMN SET DATA TYPE":    `ALTER TABLE leser ALTER COLUMN klasse SET DATA TYPE text;`,
		"RENAME VALUE":                  `ALTER TYPE benutzer_rolle RENAME VALUE 'lehrer' TO 'kollegium';`,
		"TRUNCATE":                      `TRUNCATE nachbuch_meldungen;`,
		"UPDATE im DO-Block":            "DO $$\nBEGIN\n  UPDATE leser SET art = 'liv';\nEND $$;",
		"UPDATE im DO-Block mit Marke":  "DO $body$ BEGIN DELETE FROM x; END $body$;",
		"UPDATE nach einer Funktion":    "CREATE FUNCTION f() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RETURN NEW; END $$;\nUPDATE leser SET art = 'liv';",
		"UPDATE über mehrere Zeilen":    "UPDATE\n  buecher_exemplare e\nSET\n  zugang_am = erworben_am;",
		"DELETE nach Kommentar im Satz": "DELETE /* alte Zeilen */ FROM klassen;",
	}
	for name, sql := range aendern {
		if datenaenderungIn(sql) == "" {
			t.Errorf("verbotene Form %q nicht erkannt: %s", name, sql)
		}
	}

	sicher := map[string]string{
		"Funktionskörper":              "CREATE OR REPLACE FUNCTION f() RETURNS trigger LANGUAGE plpgsql AS $$\nBEGIN\n  UPDATE leser SET x = 1;\n  DELETE FROM y;\n  RETURN NEW;\nEND $$;",
		"Funktion mit benannter Marke": "CREATE FUNCTION g() RETURNS void LANGUAGE sql AS $fn$ DELETE FROM x; TRUNCATE y $fn$;",
		"Prozedur":                     "CREATE PROCEDURE p() LANGUAGE sql AS $$ UPDATE t SET a = 1 $$;",
		"Kommentar":                    "-- UPDATE leser SET art = 'liv';\n/* DELETE FROM x; */\nCREATE INDEX i ON t (a);",
		"Fremdschlüssel":               `ALTER TABLE a ADD CONSTRAINT fk FOREIGN KEY (b) REFERENCES c (id) ON UPDATE CASCADE ON DELETE CASCADE;`,
		"Trigger":                      `CREATE TRIGGER t BEFORE INSERT OR UPDATE OF barcode_id, deleted_at ON leser FOR EACH ROW EXECUTE FUNCTION f();`,
		"Sperre":                       `SELECT id FROM leser WHERE id = 1 FOR UPDATE;`,
		"Seed ohne Überschreiben":      `INSERT INTO klassen (name) VALUES ('05F1') ON CONFLICT DO NOTHING;`,
		"neue Spalte":                  `ALTER TABLE leser ADD COLUMN IF NOT EXISTS letzter_vorgang_am TIMESTAMPTZ;`,
		"Vorgabe":                      `ALTER TABLE buecher_exemplare ALTER COLUMN erworben_am SET DEFAULT CURRENT_DATE;`,
		"Sicht":                        `CREATE OR REPLACE VIEW schueler AS SELECT * FROM leser WHERE art = 'schueler' WITH CHECK OPTION;`,
		"CHECK im DO-Block":            "DO $$\nBEGIN\n  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'c') THEN\n    ALTER TABLE t ADD CONSTRAINT c CHECK (a > 0);\n  END IF;\nEND $$;",
	}
	for name, sql := range sicher {
		if form := datenaenderungIn(sql); form != "" {
			t.Errorf("sichere Form %q als Datenänderung (%s) gemeldet: %s", name, form, sql)
		}
	}

	mitMessung := map[string]string{
		"Satz in einer Zeile":    "-- Gemessen am Testserver am 23.09.2026: 3 Zeilen betroffen.\nUPDATE t SET a = 1;",
		"über den Zeilenumbruch": "-- Rückgabetermin. Gemessen am Testserver am\n-- 22.09.2026: kein Titel trug einen Wert.\nALTER TABLE t DROP COLUMN c;",
		"nachgemessen":           "-- Zahlen vorher nachgemessen (Testserver, 16.09.2026): 0 Zeilen.\nDELETE FROM t;",
		"begründet ohne":         "-- Ohne Messung: Die Tabelle entsteht in dieser Migration und ist leer.\nINSERT INTO t (a) SELECT 1;",
	}
	for name, sql := range mitMessung {
		if !messungIm(sql) {
			t.Errorf("Messung %q nicht erkannt: %s", name, sql)
		}
	}
	ohneMessung := map[string]string{
		"kein Satz":           "-- Rückfüllung über den ganzen Bestand.\nUPDATE t SET a = 1;",
		"gemessen ohne Datum": "-- Gemessen am Testserver.\nUPDATE t SET a = 1;",
		"leerer Grund":        "-- Ohne Messung:\nUPDATE t SET a = 1;",
		"Datum nur im SQL":    "UPDATE t SET gemessen = '23.09.2026';",
	}
	for name, sql := range ohneMessung {
		if messungIm(sql) {
			t.Errorf("Form %q als Messung gelesen: %s", name, sql)
		}
	}
}

// Die Gegenprobe am echten Bestand: Der Detektor liest die Migrationen des Hauses so, wie
// sie gemeint sind. 129, 136 und 140 ändern Zeilen (Rückfüllung, Nachtrag, Normalform), 134
// nimmt eine Spalte weg; 131, 133 und 144 legen nur Funktionen und Trigger an. 134, 136
// und 140 nennen ihre Messung, 129 nicht — ab Nummer 145 wäre 129 rot. 131 nennt auch eine
// („nachgemessen am 22.09.2026: 65.000 Exemplare …"), nur misst sie die Sperrtabelle, nicht
// den Bestand: genau die Blindheit aus dem Kopfkommentar.
func TestMigrationDatenlage_LiestDenBestandRichtig(t *testing.T) {
	erwartet := []struct {
		praefix           string
		aendert, gemessen bool
	}{
		{"129_", true, false},
		{"131_", false, true},
		{"133_", false, false},
		{"134_", true, true},
		{"136_", true, true},
		{"140_", true, true},
		{"144_", false, false},
	}
	for _, e := range erwartet {
		treffer, err := filepath.Glob("../migrations/" + e.praefix + "*.sql")
		if err != nil || len(treffer) != 1 {
			t.Fatalf("Migration %s*: %d Treffer (%v)", e.praefix, len(treffer), err)
		}
		roh, err := os.ReadFile(treffer[0])
		if err != nil {
			t.Fatal(err)
		}
		sql := string(roh)
		if got := datenaenderungIn(sql) != ""; got != e.aendert {
			t.Errorf("%s: ändert Daten = %v, erwartet %v (Form %q)", filepath.Base(treffer[0]), got, e.aendert, datenaenderungIn(sql))
		}
		if got := messungIm(sql); got != e.gemessen {
			t.Errorf("%s: nennt Messung = %v, erwartet %v", filepath.Base(treffer[0]), got, e.gemessen)
		}
	}
}
