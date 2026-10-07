package littera

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Littera schließt den Artikel am Anfang eines Titels in Nichtsortierzeichen ein („¬Die¬
// schwarze Katze"), damit er beim Ordnen übersprungen wird. Die Formen stammen aus der
// Sicherung von 2010: 1.403 von 10.732 Titeln und 3 von 7.364 Verfassern tragen das Zeichen,
// die übrigen Tabellen der Übernahme keines.
func TestLeseTitel_OhneNichtsortierzeichen(t *testing.T) {
	const kopf = "Buchungsnummer,Haupttitel,Untertitel,Urheber,Verfasserangabe\n"
	faelle := []struct{ name, roh, titel string }{
		{"Paar am Anfang", "¬Die¬ schwarze Katze", "Die schwarze Katze"},
		{"ohne schließendes Zeichen", "¬Les nougats", "Les nougats"},
		{"Paar mitten im Titel", "Mumienherz. ¬Die¬ Rückkehr des Seth / 1", "Mumienherz. Die Rückkehr des Seth / 1"},
		// Wie in Littera getippt: Das Leerzeichen fehlt dort, die Übernahme erfindet keins.
		{"kein Leerzeichen nach dem Paar", "¬The¬Scarlett Letter", "TheScarlett Letter"},
		// Die Übernahme liest sie mit; ein Leerzeichen daraus macht die Datenbank (Migration 160).
		{"zwei Leerzeichen in Folge", "¬La¬  Peste", "La  Peste"},
		{"Zeichen am Rand", " ¬ Das IGL-Buch 1 ¬ ", "Das IGL-Buch 1"},
		{"ohne Zeichen", "Die Welle", "Die Welle"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			titel, err := LeseTitel(strings.NewReader(kopf + `7,"` + f.roh + `","¬Ein¬ Untertitel","","¬Der¬ Verfasser"` + "\n"))
			if err != nil {
				t.Fatalf("Titel lesen: %v", err)
			}
			if len(titel) != 1 {
				t.Fatalf("ein Titel erwartet, waren %d", len(titel))
			}
			if titel[0].Haupttitel != f.titel {
				t.Errorf("Haupttitel: erwartet %q, war %q", f.titel, titel[0].Haupttitel)
			}
			if titel[0].Untertitel != "Ein Untertitel" {
				t.Errorf("Untertitel: %q", titel[0].Untertitel)
			}
			if titel[0].Autor != "Der Verfasser" {
				t.Errorf("Verfasserangabe: %q", titel[0].Autor)
			}
		})
	}
}

// Ein Titel, der nur aus den Zeichen besteht, ist leer und bekommt den Platzhalter des
// Schreibers („[ohne Titel, Littera …]").
func TestLeseTitel_NurNichtsortierzeichenIstLeer(t *testing.T) {
	titel, err := LeseTitel(strings.NewReader("Buchungsnummer,Haupttitel\n7,\"¬ ¬\"\n"))
	if err != nil {
		t.Fatalf("Titel lesen: %v", err)
	}
	if len(titel) != 1 || titel[0].Haupttitel != "" {
		t.Errorf("leerer Haupttitel erwartet, war %+v", titel)
	}
}

func TestLesePersonen_OhneNichtsortierzeichen(t *testing.T) {
	const csv = `Buchungsnummer,Name
1,"Grün, Max ¬von der¬"
2,"¬Der¬ Rechtsstaat"
3,"Gruber, Hans-Martin ¬"
4,"¬"
`
	namen, err := LesePersonen(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("Personen lesen: %v", err)
	}
	erwartet := map[string]string{"1": "Grün, Max von der", "2": "Der Rechtsstaat", "3": "Gruber, Hans-Martin"}
	for id, name := range erwartet {
		if namen[id] != name {
			t.Errorf("Person %s: erwartet %q, war %q", id, name, namen[id])
		}
	}
	// Ein Name, der nur aus dem Zeichen besteht, ist leer und erzeugt keinen Verfasser.
	if _, da := namen["4"]; da {
		t.Errorf("leerer Name darf keinen Eintrag erzeugen, war %q", namen["4"])
	}
}

// TestEchterAltbestand_OhneNichtsortierzeichen stellt dieselbe Frage dem echten Export
// (LITTERA_CSV_DIR wie bei TestEchterAltbestand; gebraucht werden titel.csv und personen.csv).
func TestEchterAltbestand_OhneNichtsortierzeichen(t *testing.T) {
	basis := os.Getenv("LITTERA_CSV_DIR")
	if basis == "" {
		t.Skip("LITTERA_CSV_DIR nicht gesetzt — Lauf gegen den echten Altbestand übersprungen")
	}
	roh, err := os.ReadFile(filepath.Join(basis, "titel.csv")) //nolint:gosec // Pfad kommt aus der Testumgebung
	if err != nil {
		t.Fatalf("titel.csv: %v", err)
	}
	if !bytes.Contains(roh, []byte("¬")) {
		t.Fatal("der Export trägt kein Nichtsortierzeichen — die Prüfung sähe nichts")
	}

	var mit int
	for _, ti := range ladeTitel(t, filepath.Join(basis, "titel.csv")) {
		if strings.Contains(ti.Haupttitel+ti.Untertitel+ti.Autor, "¬") {
			mit++
		}
	}
	if mit > 0 {
		t.Errorf("%d Titel tragen nach dem Lesen noch ein Nichtsortierzeichen", mit)
	}

	pf, err := os.Open(filepath.Join(basis, "personen.csv")) //nolint:gosec // Pfad kommt aus der Testumgebung
	if err != nil {
		t.Fatalf("personen.csv: %v", err)
	}
	t.Cleanup(func() {
		if err := pf.Close(); err != nil {
			t.Logf("Datei schliessen: %v", err)
		}
	})
	personen, err := LesePersonen(pf)
	if err != nil {
		t.Fatalf("Personen lesen: %v", err)
	}
	for _, name := range personen {
		if strings.Contains(name, "¬") {
			t.Errorf("ein Verfasser trägt nach dem Lesen noch ein Nichtsortierzeichen")
			break
		}
	}
}
