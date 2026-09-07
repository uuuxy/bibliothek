package api

import (
	"testing"
	"time"

	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Drei Tage im Jahr überschrieb „Plan speichern" den laufenden Plan.
//
// Register 06.09.2026: Zwischen dem Donnerstag, an dem der Büchertausch endet, und dem
// Montag, an dem die Sommerferien beginnen, ist der Rückgabe-Plan „vorbei" (letzter
// Platz < heute) — der Planer bietet den Vorschlag fürs nächste Jahr an. Der Vorschlag
// fragte aber Naechste(heute), und das nennt in diesen drei Tagen noch DIESELBEN Ferien
// wie der abgelaufene Plan. Daraus wurde derselbe schuljahr_beginn, und der Upsert
// (ON CONFLICT art, schuljahr_beginn) traf den alten, weiterhin veröffentlichten Plan.
//
// Gerechnet an der echten Tabelle (lmfplan.Hessen): Ferien 2027 beginnen Montag, den
// 28.06.; der Tausch endet Donnerstag, den 24.06.; das Fenster ist Fr 25. bis So 27.06.
// Ab dem ersten Ferientag stimmte die Rechnung von allein wieder — deshalb steht der
// 28.06. als Gegenprobe mit drin: Der Umbau darf dort nichts ändern.
func TestLmfPlanVorschlagGehtUeberDieFerienDesVorigenPlansHinaus(t *testing.T) {
	tab := lmfplan.Hessen()
	tag := func(datum string) time.Time {
		t.Helper()
		d, err := time.ParseInLocation("2006-01-02", datum, schulzeit.Zone())
		if err != nil {
			t.Fatal(err)
		}
		return d.Add(10 * time.Hour)
	}
	plan2027 := &repository.LmfPlan{Art: repository.LmfTerminRueckgabe, ErsterTag: "2027-06-14", LetzterTag: "2027-06-24"}

	faelle := []struct {
		name  string
		plan  *repository.LmfPlan
		heute string
		jahr  int
	}{
		{"Fr nach dem Tausch, vor den Ferien", plan2027, "2027-06-25", 2028},
		{"Sa im Fenster", plan2027, "2027-06-26", 2028},
		{"So vor Ferienbeginn", plan2027, "2027-06-27", 2028},
		{"erster Ferientag: rechnete schon vorher richtig", plan2027, "2027-06-28", 2028},
		{"vorher am Donnerstag ist der Plan noch nicht vorbei — nicht dieser Zweig", nil, "2027-06-24", 2027},
		{"kein Plan: nächste Ferien von heute aus", nil, "2027-06-25", 2027},
		{"alter Plan von 2025, heute Mai 2027: nicht stur Planjahr+1, sondern die von heute",
			&repository.LmfPlan{Art: repository.LmfTerminRueckgabe, ErsterTag: "2025-06-16", LetzterTag: "2025-06-26"}, "2027-05-03", 2027},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			// laufend=false: In all diesen Fällen ist der Plan vorbei (oder es gibt keinen).
			ferien, _ := lmfPlanSommerferien(repository.LmfTerminRueckgabe, f.plan, false, tag(f.heute), tab)
			if !ferien.Bekannt {
				t.Fatalf("heute %s: Ferienjahr %d unbekannt — Tabelle zu kurz?", f.heute, ferien.Jahr)
			}
			if ferien.Jahr != f.jahr {
				t.Errorf("heute %s: Vorschlag richtet sich an den Ferien %d aus, erwartet %d — "+
					"derselbe schuljahr_beginn wie der vorige Plan hiesse: Speichern überschreibt ihn",
					f.heute, ferien.Jahr, f.jahr)
			}
		})
	}
}
