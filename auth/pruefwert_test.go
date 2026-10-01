package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestPruefwert_PasstNurZumPasswortUndZumSchluessel(t *testing.T) {
	schluessel := pruefwertSchluessel([]byte(testSecret))
	wert, err := bildePruefwert(schluessel, "Geheim-ÄÖ-123")
	if err != nil {
		t.Fatalf("bildePruefwert: %v", err)
	}
	if strings.Contains(wert, "Geheim") {
		t.Fatalf("der Prüfwert enthält das Passwort: %q", wert)
	}

	faelle := []struct {
		name       string
		schluessel []byte
		passwort   string
		passt      bool
	}{
		{"richtiges Passwort", schluessel, "Geheim-ÄÖ-123", true},
		{"falsches Passwort", schluessel, "Geheim-ÄÖ-124", false},
		{"leeres Passwort", schluessel, "", false},
		// Wer nur die Datenbank hat, hat den Schlüssel nicht — dann passt auch das
		// richtige Passwort nicht.
		{"richtiges Passwort, fremder Schlüssel", pruefwertSchluessel([]byte("ein-anderes-geheimnis-mit-32-zeichen!!")), "Geheim-ÄÖ-123", false},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			passt, err := pruefwertPasst(f.schluessel, wert, f.passwort)
			if err != nil {
				t.Fatalf("pruefwertPasst: %v", err)
			}
			if passt != f.passt {
				t.Errorf("passt = %v, erwartet %v", passt, f.passt)
			}
		})
	}
}

func TestPruefwert_GleichesPasswortErgibtVerschiedeneWerte(t *testing.T) {
	schluessel := pruefwertSchluessel([]byte(testSecret))
	a, errA := bildePruefwert(schluessel, "dasselbe")
	b, errB := bildePruefwert(schluessel, "dasselbe")
	if errA != nil || errB != nil {
		t.Fatalf("bildePruefwert: %v / %v", errA, errB)
	}
	if a == b {
		t.Error("zwei Prüfwerte desselben Passworts sind gleich — ohne eigenes Salz verrät die Tabelle, wer dasselbe Passwort hat")
	}
}

func TestPruefwert_UnlesbarerWertIstEinFehlerKeinTreffer(t *testing.T) {
	schluessel := pruefwertSchluessel([]byte(testSecret))
	for _, wert := range []string{"", "klartext", "argon2id$19456$2$1$nur-fuenf", "bcrypt$1$2$3$YQ$YQ", "argon2id$0$2$1$YQ$YQ", "argon2id$19456$2$1$!!$YQ"} {
		passt, err := pruefwertPasst(schluessel, wert, "egal")
		if passt {
			t.Errorf("Prüfwert %q passt — ein unlesbarer Wert darf nie aufschließen", wert)
		}
		if !errors.Is(err, errPruefwertUnlesbar) {
			t.Errorf("Prüfwert %q: Fehler = %v, erwartet errPruefwertUnlesbar", wert, err)
		}
	}
}
