package docs

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Der Anker zu Raster-Frage 12: Was ein körperliches DELETE tut, das kein Lesepfad ahnt.
//
// Ein Exemplar verlässt den Bestand normalerweise als AUSSONDERUNG: Die Zeile bleibt,
// `ist_ausgesondert` wird wahr, der Trigger aus Migration 128 stempelt das Abgangsdatum,
// und das Abgangsbuch führt sie. Drei Türen tun etwas anderes — sie löschen die Zeile.
// Danach steht das Exemplar in KEINER Abfrage über `buecher_exemplare` mehr, auch nicht
// im Nachweis, den die Schule zum Stichtag ausdruckt und abheftet.
//
// Gefunden am 17.09.2026 und an der Datenbank nachgestellt: Ein ausgesondertes Exemplar
// mit Abgangsdatum steht vor dem Löschen seines Titels im Abgangsbuch und danach nicht
// mehr — rückwirkend, in einem Halbjahr, das vielleicht schon unterschrieben ist.
//
// Die Antwort darauf ist KEIN Verbot: Dass „Titel löschen" wirklich alles löscht, ist eine
// Entscheidung des Betreibers vom 23.08.2026. Die Antwort ist die Zahl unter der Liste
// (repository/abgangsbuch.go), und die kann nur zählen, was eine Protokollspur hinterlässt.
// Deshalb diese Ratsche: **Eine fünfte Anweisung ist eine Frage.** Wer eine hinzufügt,
// schreibt daneben, welche Spur sie legt — oder merkt beim Schreiben, dass sie keine legt
// und der Nachweis ab dann lügt.
var koerperlicheLoeschungBestand = map[string]struct {
	anzahl int
	grund  string
}{
	"repository/audit_books.go": {2, "Titel löschen (einzeln): erst die Exemplare, dann der Titel. " +
		"Spur je Exemplar mit AuditAktionTitelGeloescht."},
	"inventur/db_books_delete.go": {1, "Titel löschen (Massenaktion): die Exemplare fallen per " +
		"ON DELETE CASCADE. Spur je Exemplar in db_books_delete_spur.go, ebenfalls AuditAktionTitelGeloescht."},
	"repository/inventur_verlust_aktionen.go": {1, "Verlust endgültig löschen (Inventur). " +
		"Spur je Exemplar mit AuditAktionVerlustEndgueltigGeloescht."},
}

var musterKoerperlicheLoeschung = regexp.MustCompile(`(?i)DELETE\s+FROM\s+buecher_(exemplare|titel)\b`)

func TestKoerperlicheLoeschung_FuenfteAnweisungIstEineFrage(t *testing.T) {
	gefunden := map[string]int{}
	wurzel := ".."
	err := filepath.Walk(wurzel, func(pfad string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if name := info.Name(); name == "node_modules" || name == ".git" || name == "frontend" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		roh, err := os.ReadFile(pfad)
		if err != nil {
			return err
		}
		// Kommentare zuerst weg: Eine Ratsche, die einen erklärenden Satz mitzählt, meldet
		// eine Tür, die es nicht gibt — und wer sie einmal als Fehlalarm abtut, tut es beim
		// nächsten Mal wieder.
		quelle := musterSQLKomm.ReplaceAllString(musterZeilenKomm.ReplaceAllString(string(roh), ""), "")
		if n := len(musterKoerperlicheLoeschung.FindAllString(quelle, -1)); n > 0 {
			rel := strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(pfad, wurzel)), "/")
			gefunden[rel] = n
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Baum lesen: %v", err)
	}
	// Sanity-Floor: Ein Detektor, der nichts findet, ist grün und wertlos.
	if len(gefunden) == 0 {
		t.Fatal("keine einzige Lösch-Anweisung gefunden — der Detektor misst nichts; " +
			"heißen die Tabellen noch buecher_exemplare/buecher_titel?")
	}

	var neu, gewachsen, veraltet []string
	for pfad, n := range gefunden {
		eintrag, bekannt := koerperlicheLoeschungBestand[pfad]
		switch {
		case !bekannt:
			neu = append(neu, pfad)
		case n > eintrag.anzahl:
			gewachsen = append(gewachsen, pfad)
		}
	}
	for pfad, eintrag := range koerperlicheLoeschungBestand {
		if gefunden[pfad] < eintrag.anzahl {
			veraltet = append(veraltet, pfad)
		}
		if eintrag.grund == "" {
			t.Errorf("%s steht ohne Begründung im Bestand", pfad)
		}
	}
	sort.Strings(neu)
	sort.Strings(gewachsen)
	sort.Strings(veraltet)

	for _, p := range neu {
		t.Errorf("%s löscht Bestandszeilen KÖRPERLICH. Damit fällt das Exemplar aus dem "+
			"Abgangsbuch, das es sonst führen würde. Bitte eine Protokollspur legen (siehe "+
			"AuditAktionTitelGeloescht) und mit diesem Satz in koerperlicheLoeschungBestand "+
			"eintragen — sonst verliert der Nachweis stillschweigend eine Zeile.", p)
	}
	for _, p := range gewachsen {
		t.Errorf("%s löscht öfter körperlich als eingetragen (%d statt %d) — jede Anweisung "+
			"ist eine eigene Frage nach ihrer Spur.", p, gefunden[p], koerperlicheLoeschungBestand[p].anzahl)
	}
	for _, p := range veraltet {
		t.Errorf("%s löscht seltener als eingetragen (%d statt %d) — bitte den Bestand "+
			"nachziehen, damit er weiter etwas aussagt.", p, gefunden[p], koerperlicheLoeschungBestand[p].anzahl)
	}
}
