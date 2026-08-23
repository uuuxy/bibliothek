package repository

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Die Löschbedingungen an echtem Postgres ausgerechnet: Wo genau liegt die Grenze?
//
// Der Compiler verhindert seit dem 23.08.2026, dass Zahlen und Bedingung getrennt
// voneinander falsch gesetzt werden (Loeschbedingung trägt beides). Was er NICHT
// verhindert, ist eine falsche Einheit IN der Bedingung — `days` statt `months`. Genau
// dort ist der Schaden am größten: Aus 24 Monaten Aufbewahrung würden 24 Tage, und der
// nächtliche Lauf räumte das Protokoll bis auf den letzten Monat ab. Unumkehrbar.
//
// Deshalb wird die Grenze hier ausgerechnet und mit dem Kalender verglichen, statt die
// SQL-Zeichenkette anzusehen. Ein Test, der `months =>` im Text sucht, hielte auch dann
// still, wenn der Parameter daneben landet.
func TestLoeschbedingungen_GrenzenStimmenMitDemKalender(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	// grenze rechnet aus, ab welchem Zeitpunkt eine Bedingung greift: Sie fragt
	// Postgres, für welches Datum das Prädikat gerade noch wahr wird.
	grenze := func(b Loeschbedingung, spalte string) time.Time {
		t.Helper()
		// Die Bedingung bezieht sich auf eine Spalte; wir setzen den Kandidaten als
		// Wert ein und lassen Postgres dieselbe Rechnung machen wie im Lauf.
		var g time.Time
		sql := `SELECT ` + zeitAusdruck(b.Where, spalte)
		if err := pool.QueryRow(ctx, sql, b.Args...).Scan(&g); err != nil {
			t.Fatalf("Grenze berechnen (%s): %v", sql, err)
		}
		return g
	}

	jetzt := time.Now()

	t.Run("Audit-Aufbewahrung sind MONATE", func(t *testing.T) {
		g := grenze(PredikatAuditLog(StandardAuditAufbewahrungMonate, KulanzJob), "timestamp")
		erwartet := jetzt.AddDate(0, -StandardAuditAufbewahrungMonate, 0)
		if abstandTage(g, erwartet) > 2 {
			t.Fatalf("Grenze %s, erwartet ~%s — die Einheit stimmt nicht (Tage statt Monate?)",
				g.Format("2006-01-02"), erwartet.Format("2006-01-02"))
		}
		// Gegenprobe: 24 TAGE lägen mehr als ein Jahr daneben. Ohne sie könnte der Test
		// auch bei einer viel zu kurzen Frist grün bleiben.
		if abstandTage(g, jetzt.AddDate(0, 0, -StandardAuditAufbewahrungMonate)) < 300 {
			t.Fatal("Grenze liegt bei ~24 Tagen statt 24 Monaten")
		}
	})

	t.Run("Lesehistorie ist TAGE plus Kulanz", func(t *testing.T) {
		g := grenze(PredikatAnliegen(StandardAnliegenTage, KulanzWaechter), "erledigt_am")
		erwartet := jetzt.AddDate(0, 0, -(StandardAnliegenTage + KulanzWaechter))
		if abstandTage(g, erwartet) > 1 {
			t.Fatalf("Grenze %s, erwartet %s", g.Format("2006-01-02"), erwartet.Format("2006-01-02"))
		}
	})

	t.Run("Kulanz verschiebt die Grenze weiter zurück, nie nach vorn", func(t *testing.T) {
		job := grenze(PredikatAnliegen(StandardAnliegenTage, KulanzJob), "erledigt_am")
		waechter := grenze(PredikatAnliegen(StandardAnliegenTage, KulanzWaechter), "erledigt_am")
		if !waechter.Before(job) {
			t.Fatalf("Wächter-Grenze %s liegt nicht VOR der Job-Grenze %s — die Kulanz wirkt "+
				"falsch herum und die Selbstprüfung mahnte an, was der Job noch gar nicht sah",
				waechter.Format("2006-01-02"), job.Format("2006-01-02"))
		}
	})
}

func abstandTage(a, b time.Time) float64 {
	d := a.Sub(b).Hours() / 24
	if d < 0 {
		return -d
	}
	return d
}

// zeitAusdruck schneidet aus einer Bedingung den Zeitvergleich der genannten Spalte
// heraus — also alles hinter „spalte < ". Bewusst am ECHTEN Bedingungstext, nicht an
// einer Kopie: Ein Test, der die Rechnung nachbaut, prüft seine eigene Annahme.
func zeitAusdruck(where, spalte string) string {
	marke := spalte + " < "
	i := strings.Index(where, marke)
	if i < 0 {
		panic("Zeitvergleich für " + spalte + " nicht gefunden in: " + where)
	}
	rest := where[i+len(marke):]
	// Der Ausdruck endet am Zeilenumbruch oder am nächsten AND — je nachdem, was zuerst
	// kommt. Beides beendet in diesen Bedingungen zuverlässig den Vergleich.
	if j := strings.IndexAny(rest, "\n"); j >= 0 {
		rest = rest[:j]
	}
	return strings.TrimSpace(rest)
}
