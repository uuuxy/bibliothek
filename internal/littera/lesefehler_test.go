package littera

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

// Ein kaputter Export ist ein Fehler, kein leerer Bestand.
//
// mdb-export schreibt die Kopfzeile auch für eine leere Tabelle. Eine Datei ohne Kopfzeile
// ist deshalb abgebrochen oder falsch erzeugt — und ein Import, der daraus „0 Titel"
// macht, meldet Erfolg für einen Lauf, der nichts gelesen hat. Dasselbe gilt für einen
// Abbruch mitten in der Datei: Die Zeilen davor sind nicht der Bestand, sondern sein
// Anfang.
//
// Alle drei Leser gehen durch leseTabelle; geprüft wird trotzdem jeder, weil jeder den
// Fehler selbst weiterreichen muss. (Zusammengeführt aus den Jules-PRs #603, #605, #612,
// die denselben Kopfzeilen-Zweig dreimal abdeckten und den Abbruch nach der Kopfzeile
// keinmal.)
func TestLeseTabellen_KaputterExportIstEinFehler(t *testing.T) {
	simuliert := errors.New("simulierter Lesefehler")

	leser := map[string]func(io.Reader) (int, error){
		"Titel": func(r io.Reader) (int, error) {
			z, err := LeseTitel(r)
			return len(z), err
		},
		"Exemplare": func(r io.Reader) (int, error) {
			z, err := LeseExemplare(r)
			return len(z), err
		},
		"Leser": func(r io.Reader) (int, error) {
			z, err := LeseLeser(r, nil)
			return len(z), err
		},
	}

	eingaben := []struct {
		name    string
		neu     func() io.Reader
		ursache error
		teil    string
	}{
		{"leere Datei", func() io.Reader { return strings.NewReader("") }, io.EOF, "kopfzeile unlesbar"},
		{"Abbruch in der Kopfzeile", func() io.Reader { return iotest.ErrReader(simuliert) }, simuliert, "kopfzeile unlesbar"},
		{"Abbruch nach der ersten Zeile", func() io.Reader {
			return io.MultiReader(strings.NewReader("Buchungsnummer,Titel\n1,2\n"), iotest.ErrReader(simuliert))
		}, simuliert, "zeile unlesbar"},
	}

	for tabelle, lies := range leser {
		for _, e := range eingaben {
			t.Run(tabelle+"/"+e.name, func(t *testing.T) {
				anzahl, err := lies(e.neu())
				if err == nil {
					t.Fatalf("kein Fehler, %d Zeilen gelesen — ein kaputter Export sähe aus wie ein leerer Bestand", anzahl)
				}
				if !errors.Is(err, e.ursache) {
					t.Errorf("Ursache fehlt in der Kette: %v", err)
				}
				if !strings.Contains(err.Error(), e.teil) {
					t.Errorf("Meldung sagt nicht, wo es brach (%q fehlt): %v", e.teil, err)
				}
				if anzahl != 0 {
					t.Errorf("%d Zeilen trotz Fehler zurückgegeben", anzahl)
				}
			})
		}
	}
}
