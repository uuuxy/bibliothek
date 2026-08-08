package api

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// Strukturinvariante der Backend-Schichtung: Ein Handler formuliert kein SQL.
//
// Warum das zaehlt: Am 07.08.2026 enthielten 50 der 109 Dateien in api/ rohes SQL,
// waehrend 56 die repository-Schicht benutzen — und 20 machten BEIDES, in derselben
// Datei. Wer eine Abfrage aendert, muss sie also an zwei Stellen suchen.
//
// Das ist keine Theorie. Die Bugklassen, die dieses Projekt einzeln gefixt hat,
// stammen genau daher: nullbare Spalte in nicht-nullbaren Go-Typ ("cannot scan
// NULL"), Upsert ohne COALESCE-Schutz (autor/verlag/jahr wurden geleert), und
// Read-your-own-writes, das ein Batch-Refactoring still entfernt hat. Jede dieser
// Regeln steht in repository/ genau einmal — und im Handler daneben nochmal nicht.
//
// Diese Ratsche MIGRIERT nichts. Sie friert den Bestand ein, damit er nur noch
// kleiner werden kann: Ein NEUER Handler nimmt repository/, und wer eine Datei
// umstellt, nimmt sie unten heraus.

// Nur Anweisungen, keine Bezeichner: `UPDATE x SET` statt `UPDATE`, sonst schlaegt
// jedes Wort "update" in einem Bezeichner an.
//
// KEIN abschliessendes \b: Es verlangte eine Wortgrenze direkt hinter dem ersten
// Buchstaben des Tabellennamens und traf damit nur einbuchstabige Namen — der erste
// Anlauf zaehlte deshalb eine Datei anders als derselbe Ausdruck in der Shell.
var sqlAnweisung = regexp.MustCompile(`(?i)\b(SELECT\s+[a-z_*(]|INSERT\s+INTO\s+[a-z_]+|DELETE\s+FROM\s+[a-z_]+|UPDATE\s+[a-z_.]+\s+SET)`)

// Kommentare zaehlen nicht: In bestellbestaetigung_handler.go steht "zwischen SELECT
// und UPDATE ein Wettlauf-Fenster" — eine Erklaerung, keine Abfrage.
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

// Bestand vom 07.08.2026. Wer eine Datei auf repository/ umstellt, nimmt sie hier
// heraus — der Test meldet beides, neu hinzugekommene UND inzwischen saubere.
var handlerMitSQL = []string{
	"action.go",
	"audit_handler.go",
	"audit_logs_handler.go",
	"ausleihe.go",
	"ausweis_layout.go",
	"barcode_print.go",
	"bestellbericht_handler.go",
	"bestellbestaetigung_etiketten.go",
	"bestellbestaetigung_handler.go",
	"bestellbestaetigung_link_handler.go",
	"bestellbestaetigung_public.go",
	"bestellhistorie_handler.go",
	"bestellhistorie_uebersicht.go",
	"bestellmail_text.go",
	"book_systematik_handler.go",
	"copy_admin.go",
	"dsgvo_auskunft.go",
	"etiketten_offen.go",
	"graduates.go",
	"graduates_mail.go",
	"isbn_handler.go",
	"klassen_mapping.go",
	"labels.go",
	"littera_import.go",
	"lookups.go",
	"lusd.go",
	"lusd_apply.go",
	"mahnwesen_bulk.go",
	"mahnwesen_bulk_mail.go",
	"mail_routes.go",
	"mail_settings.go",
	"monitor.go",
	"opac.go",
	"order_service.go",
	"pdf.go",
	"permission_middleware.go",
	"photo_serve.go",
	"print.go",
	"reorders.go",
	"reporting_dashboard.go",
	"reports_pdf.go",
	"settings.go",
	"signaturen_handler.go",
	"stats.go",
	"student_create.go",
	"student_deleted.go",
	"student_lock.go",
	"student_promotion.go",
	"student_update.go",
	"supplier_handler.go",
	"systematik_handler.go",
	"user_admin.go",
	"user_admin_permissions.go",
}

func TestHandlerFormulierenKeinNeuesSQL(t *testing.T) {
	eintraege, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("api/ nicht lesbar: %v", err)
	}

	var gefunden []string
	for _, e := range eintraege {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		quelle, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("%s nicht lesbar: %v", name, err)
		}
		if sqlAnweisung.MatchString(ohneKommentare(string(quelle))) {
			gefunden = append(gefunden, name)
		}
	}
	slices.Sort(gefunden)

	// Ohne diese Zusicherung waere ein umbenanntes Verzeichnis ein still gruener Test.
	if len(gefunden) == 0 {
		t.Fatal("kein einziger Handler mit SQL gefunden — der Test misst offenbar nichts mehr")
	}

	for _, f := range gefunden {
		if !slices.Contains(handlerMitSQL, f) {
			t.Errorf("api/%s formuliert SQL. Handler lesen und schreiben ueber repository/ —\n"+
				"dort steht jede Regel (COALESCE-Schutz, NULL-Behandlung, Reihenfolge in der Tx)\n"+
				"genau einmal. Neuer Bedarf gehoert in eine repository-Funktion.", f)
		}
	}
	for _, f := range handlerMitSQL {
		if !slices.Contains(gefunden, f) {
			t.Errorf("api/%s enthaelt kein SQL mehr — bitte aus handlerMitSQL entfernen,\n"+
				"damit die Ratsche greift.", f)
		}
	}
}
