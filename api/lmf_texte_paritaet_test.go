package api

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"bibliothek/repository"
)

// Dieselben Worte auf beiden Wegen zum Plan.
//
// Überschrift und Erklärungssatz jedes Plans stehen zweimal: in Go (LmfArtTitel /
// LmfArtErklaerung — Quelle für das Kollegiums-PDF) und in JavaScript (ARTEN /
// artErklaerung in lmfplanDienst.js — Quelle für Planer und Portal-Reiter). Der
// Go-Kommentar sagt „dieselben Worte wie ARTEN in lmfplanDienst.js" — bis zum
// Rasterdurchgang am 06.09.2026 war das eine Verabredung zwischen zwei Dateien, die
// niemand prüfte (Raster, Frage 3: zwei Wahrheitsquellen).
//
// Wirkung ohne Gate: Peter hat die Begriffe am 06.09.2026 schon einmal umbenannt („nicht
// Rückgabe und Ausgabe, das sind zwei verschiedene Dinge"). Wird nur die eine Seite
// angefasst, zeigt der Portal-Reiter den neuen Titel und das PDF, das derselbe Knopf im
// selben Reiter herunterlädt, den alten.
//
// Gelesen wird die JS-Datei als Text — ein Node-Lauf im Go-Test wäre eine zweite
// Werkzeugkette für eine Zeichenkette. Findet der Test die Stelle nicht mehr, ist er rot
// und nicht still grün.
func TestLmfTexte_GoUndJavaScriptSagenDasselbe(t *testing.T) {
	quelle, err := os.ReadFile("../frontend/src/lib/lmfplanDienst.js")
	if err != nil {
		t.Fatalf("lmfplanDienst.js lesen: %v", err)
	}
	js := string(quelle)

	// 1. Die zwei Überschriften: label je Art aus ARTEN.
	for _, f := range []struct{ art, jsWert string }{
		{repository.LmfTerminRueckgabe, "rueckgabe"},
		{repository.LmfTerminAusgabe, "ausgabe"},
	} {
		muster := regexp.MustCompile(`\{ wert: '` + f.jsWert + `', label: '([^']+)' \}`)
		treffer := muster.FindStringSubmatch(js)
		if treffer == nil {
			t.Fatalf("ARTEN-Eintrag %q in lmfplanDienst.js nicht gefunden — Form geändert? "+
				"Dann diesen Test nachziehen, sonst prüft er nichts.", f.jsWert)
		}
		if got := LmfArtTitel(f.art); got != treffer[1] {
			t.Errorf("Überschrift %q: Go %q, JavaScript %q — PDF und Portal nennen den Plan verschieden",
				f.art, got, treffer[1])
		}
	}

	// 2. Der Erklärungssatz. Der Ausgabe-Satz trägt die Eingangsjahrgänge; verglichen wird
	//    mit dem Text, den auch das PDF druckt, also mit derselben Liste.
	eingang := []int{5, 7}
	// Ohne Jahrgänge fällt das Klammerpaar weg — auf beiden Seiten gleich, sonst stünde im
	// Portal „(Jahrgang )", während das PDF den ganzen Satz druckt.
	if got, will := LmfArtErklaerung(repository.LmfTerminAusgabe, nil),
		"Nur die neu gebildeten Klassen bekommen ihre Schulbücher."; got != will {
		t.Errorf("Ausgabe-Satz ohne Jahrgänge: %q", got)
	}
	if !strings.Contains(js, "'Nur die neu gebildeten Klassen bekommen ihre Schulbücher.'") {
		t.Error("lmfplanDienst.js kennt den Satz ohne Jahrgänge nicht — dann zeigt das Portal ein leeres Klammerpaar")
	}
	sätze := map[string]string{
		repository.LmfTerminAusgabe: "Nur die neu gebildeten Klassen (Jahrgang ${jahrgaengeText(eingang)}) bekommen ihre Schulbücher.",
		repository.LmfTerminRueckgabe: "Alle Klassen geben die alten Schulbücher ab und bekommen direkt die neuen. " +
			"„Nur Rückgabe“: Abschlussklassen und Klassen, die zum neuen Schuljahr neu gebildet werden.",
	}
	for art, jsSatz := range sätze {
		if !strings.Contains(js, jsSatz) {
			t.Errorf("Erklärungssatz %q steht so nicht in lmfplanDienst.js:\n  Go: %s",
				art, LmfArtErklaerung(art, eingang))
			continue
		}
		// Die Go-Fassung mit denselben Jahrgängen muss den JS-Satz ergeben, sobald die
		// Einsetzung ausgeführt ist.
		erwartet := strings.ReplaceAll(jsSatz, "${jahrgaengeText(eingang)}", jahrgaengeText(eingang))
		if got := LmfArtErklaerung(art, eingang); got != erwartet {
			t.Errorf("Erklärungssatz %q:\n  Go:         %s\n  JavaScript: %s", art, got, erwartet)
		}
	}
}
