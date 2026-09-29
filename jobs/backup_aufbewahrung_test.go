package jobs

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"
	"testing"
	"time"
)

// Die Aufbewahrung (entschieden am 28.09.2026): die 14 jüngsten Nachtsicherungen, dazu von den
// älteren je Kalenderwoche die jüngste für 12 Wochen. Eine Nachtsicherung je Tag um 02:30 UTC
// (jobs/cron.go) vom 01.05. bis 29.09.2026, 152 Stück. Die erwartete Liste ist unabhängig vom
// Programm ausgezählt: 16.09.–29.09. jede Nacht, davor Dienstag 15.09. (die jüngste ältere,
// Woche 38) und die Sonntage 13.09. zurück bis 05.07. (Wochen 37 bis 27).
func TestZuLoeschen_NaechteUndWochen(t *testing.T) {
	var dateien []BackupDatei
	for tag := time.Date(2026, time.May, 1, 2, 30, 0, 0, time.UTC); !tag.After(time.Date(2026, time.September, 29, 3, 0, 0, 0, time.UTC)); tag = tag.AddDate(0, 0, 1) {
		dateien = append(dateien, BackupDatei{Name: backupPraefix + tag.Format(sicherungsStempel) + backupEndung})
	}
	weg := zuLoeschen(dateien, BehalteNaechte, BehalteWochen)
	var bleibt []string
	for _, d := range dateien {
		if !slices.ContainsFunc(weg, func(w BackupDatei) bool { return w.Name == d.Name }) {
			bleibt = append(bleibt, d.Name[len(backupPraefix):len(backupPraefix)+10])
		}
	}
	soll := []string{
		"2026-07-05", "2026-07-12", "2026-07-19", "2026-07-26", "2026-08-02", "2026-08-09",
		"2026-08-16", "2026-08-23", "2026-08-30", "2026-09-06", "2026-09-13", "2026-09-15",
	}
	for tag := 16; tag <= 29; tag++ {
		soll = append(soll, time.Date(2026, time.September, tag, 0, 0, 0, 0, time.UTC).Format("2006-01-02"))
	}
	if !slices.Equal(bleibt, soll) {
		t.Errorf("es bleiben %d:\n%v\nerwartet %d:\n%v", len(bleibt), bleibt, len(soll), soll)
	}
}

// Passt ein Name nicht zum Stempel, zählt die Änderungszeit: Von zwei solchen Dateien in
// derselben Woche bleibt eine.
func TestZuLoeschen_OhneStempelZaehltDieAenderungszeit(t *testing.T) {
	dateien := []BackupDatei{
		{Name: "backup_alt-a.sql.gz.enc", GeaendertAm: time.Date(2026, time.August, 4, 3, 0, 0, 0, time.UTC)},
		{Name: "backup_alt-b.sql.gz.enc", GeaendertAm: time.Date(2026, time.August, 5, 3, 0, 0, 0, time.UTC)},
		{Name: "backup_neu.sql.gz.enc", GeaendertAm: time.Date(2026, time.September, 29, 3, 0, 0, 0, time.UTC)},
	}
	weg := zuLoeschen(dateien, 1, 12)
	if len(weg) != 1 || weg[0].Name != "backup_alt-a.sql.gz.enc" {
		t.Errorf("gelöscht %v, erwartet nur backup_alt-a (dieselbe Woche wie alt-b, älter)", namen(weg))
	}
}

// Die Auskunft nennt BehalteNaechte und BehalteWochen (api, dsgvoSicherungen). Rotierte die
// Nachtsicherung mit anderen Zahlen, stimmte sie nicht mehr; deshalb steht der Aufruf hier fest.
// Gelesen am Syntaxbaum von backup.go, nicht am Text — ein Kommentar zählt nicht.
func TestRunDatabaseBackup_RotiertMitDenKonstanten(t *testing.T) {
	datei, err := parser.ParseFile(token.NewFileSet(), "backup.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var aufrufe []string
	ast.Inspect(datei, func(n ast.Node) bool {
		aufruf, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if name, ok := aufruf.Fun.(*ast.Ident); ok && name.Name == "rotateBackups" {
			var argumente []string
			for _, a := range aufruf.Args {
				if bezeichner, ok := a.(*ast.Ident); ok {
					argumente = append(argumente, bezeichner.Name)
				} else {
					argumente = append(argumente, "?")
				}
			}
			aufrufe = append(aufrufe, strings.Join(argumente, ", "))
		}
		return true
	})
	if len(aufrufe) != 1 || aufrufe[0] != "backupDir, BehalteNaechte, BehalteWochen" {
		t.Errorf("rotateBackups in backup.go: %q, erwartet genau einmal (backupDir, BehalteNaechte, BehalteWochen)", aufrufe)
	}
}
