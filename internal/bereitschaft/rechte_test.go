package bereitschaft

import (
	"strings"
	"testing"
)

// Freigestellte Rechte (db.RechteOptional) sind zum Umschalten da: Schaltet ein Admin dem
// Helfer die Vormerkungen zu, ist das keine Abweichung, sonst stünde jede Anlage, die das
// nutzt, dauerhaft auf Warnung. Verschwindet die Zeile ganz, fehlt die Tür in der
// Rechte-Matrix, und die Warnung kommt.
func TestBetriebsbereitschaft_OptionalesRechtIstKeineDrift(t *testing.T) {
	t.Run("eingeschaltet ist kein Befund", func(t *testing.T) {
		l := lageEingerichtet()
		l.RechteLive["HELFER"]["manage_vormerkungen"] = true
		b := befundZu(t, Pruefe(l), "Rechte-Vorgabe")
		if b.Stufe != StufeOK {
			t.Errorf("optionales Recht eingeschaltet → %q statt OK: %s", b.Stufe, b.Befund)
		}
	})
	t.Run("fehlende Zeile warnt weiterhin", func(t *testing.T) {
		l := lageEingerichtet()
		delete(l.RechteLive["HELFER"], "manage_vormerkungen")
		b := befundZu(t, Pruefe(l), "Rechte-Vorgabe")
		if b.Stufe != StufeWarnung || !strings.Contains(b.Befund+b.Folge, "HELFER/manage_vormerkungen fehlt live") {
			t.Errorf("fehlende optionale Zeile muss warnen, got %q: %s", b.Stufe, b.Befund)
		}
	})
}

// Viele Abweichungen dürfen die Auskunft nicht fluten: Gezeigt werden sechs, der Rest wird
// gezählt. Eine leere, aber lesbare Tabelle meldet jede Zeile der Vorgabe als fehlend.
func TestBetriebsbereitschaft_RechteAbweichungenWerdenGekappt(t *testing.T) {
	l := lageEingerichtet()
	l.RechteLive = map[string]map[string]bool{}

	b := befundZu(t, Pruefe(l), "Rechte-Vorgabe")
	if b.Stufe != StufeWarnung {
		t.Fatalf("Stufe %q statt Warnung: %s", b.Stufe, b.Befund)
	}
	if !strings.Contains(b.Befund, "weitere") {
		t.Errorf("die Kappung fehlt — der Befund listet alles: %s", b.Befund)
	}
	if strings.Count(b.Befund, ";") > 6 {
		t.Errorf("mehr als sechs Einträge gezeigt: %s", b.Befund)
	}
}
