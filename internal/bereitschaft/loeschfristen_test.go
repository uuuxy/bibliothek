package bereitschaft

import (
	"strings"
	"testing"
)

// Prüfung 22.08.2026: Die DSGVO-Anonymisierung lief zweimal monatelang ins Leere; nur eine
// Logzeile wusste es. Jetzt urteilt die Selbstprüfung über den ZUSTAND — seit dem
// 23.08.2026 über den ALLER Löschroutinen, nicht nur der Anonymisierung.
func TestPruefeLoeschroutinen(t *testing.T) {
	t.Run("nicht erhoben ist eine Warnung, kein OK", func(t *testing.T) {
		if b := pruefeLoeschroutinen(Lage{LoeschRueckstand: nil}); b.Stufe != StufeWarnung {
			t.Fatalf("nicht erhoben = Warnung: %+v", b)
		}
	})

	t.Run("sauber ist ok", func(t *testing.T) {
		b := pruefeLoeschroutinen(Lage{LoeschRueckstand: rueckstandSauber()})
		if b.Stufe != StufeOK || !strings.Contains(b.Befund, "6 Löschroutinen") {
			t.Fatalf("sauber = ok über alle Routinen: %+v", b)
		}
	})

	// Der eigentliche Zugewinn: JEDE Routine muss den Alarm allein auslösen können.
	// Vorher konnte das nur die Anonymisierung — die anderen liefen unbeobachtet.
	for i, r := range rueckstandSauber() {
		t.Run("überfällig: "+r.Routine, func(t *testing.T) {
			stand := rueckstandSauber()
			stand[i].Zeilen = 3
			b := pruefeLoeschroutinen(Lage{LoeschRueckstand: stand})
			if b.Stufe != StufeKritisch {
				t.Fatalf("%s überfällig muss kritisch sein: %+v", r.Routine, b)
			}
			if !strings.Contains(b.Befund, r.Routine) || !strings.Contains(b.Befund, "3") {
				t.Errorf("Befund nennt Routine und Anzahl nicht: %s", b.Befund)
			}
			if !strings.Contains(b.Befund, r.Frist) {
				t.Errorf("Befund nennt die geltende Frist nicht: %s", b.Befund)
			}
		})
	}

	t.Run("abgeschaltete Frist ist eine Warnung, kein Fehler", func(t *testing.T) {
		stand := rueckstandSauber()
		stand[2].Aus = true
		b := pruefeLoeschroutinen(Lage{LoeschRueckstand: stand})
		if b.Stufe != StufeWarnung || !strings.Contains(b.Befund, "Lesehistorie Schülerbücherei") {
			t.Fatalf("Frist auf 0 = Warnung mit Namen der Routine: %+v", b)
		}
	})

	// Rangfolge: Ein echter Rückstand darf nicht von einer abgeschalteten Frist
	// verdeckt werden — sonst meldete die Seite „Warnung", während Daten stehen bleiben.
	t.Run("Rückstand schlägt abgeschaltete Frist", func(t *testing.T) {
		stand := rueckstandSauber()
		stand[2].Aus = true
		stand[5].Zeilen = 1
		if b := pruefeLoeschroutinen(Lage{LoeschRueckstand: stand}); b.Stufe != StufeKritisch {
			t.Fatalf("Rückstand muss die Warnung überstimmen: %+v", b)
		}
	})
}
