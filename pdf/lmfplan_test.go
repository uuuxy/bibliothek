package pdf

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

// Der LMF-Plan hängt im Lehrerzimmer und geht als EIN Blatt an die Schulleitung. Peter,
// 06.09.2026: „Das PDF muss zwingend alles auf einer Seite darstellen." Ein zweites Blatt
// ist kein Schönheitsfehler, sondern ein verlorener Termin — wer nur Seite 1 abheftet,
// sieht die Hälfte der Klassen nicht.
//
// Vor dieser Ratsche brach maroto ab rund vierzig Terminen von selbst um: 20 Zeilen je
// Abschnitt ergaben zwei Seiten, 60 ergaben drei, 150 ergaben sieben.

// seitenZahl misst am fertigen Dokument, nicht an der Absicht des Satzes: gofpdf schreibt
// je Seite ein „/Type /Page"-Objekt und einmal „/Type /Pages" für den Baum darüber.
func seitenZahl(t *testing.T, b []byte) int {
	t.Helper()

	seiten := bytes.Count(b, []byte("/Type /Page")) - bytes.Count(b, []byte("/Type /Pages"))
	if seiten < 1 {
		t.Fatalf("kein /Type /Page im Dokument gefunden (%d Bytes)", len(b))
	}
	return seiten
}

const (
	lmfRueckgabeSatz = "Alle Klassen geben die alten Schulbücher ab und bekommen direkt die neuen. " +
		"„Nur Rückgabe“: Abschlussklassen und Klassen, die zum neuen Schuljahr neu gebildet werden."
	lmfAusgabeSatz = "Die Eingangsjahrgänge 5 und 7 bekommen ihre Bücher; die übrigen Klassen haben sie " +
		"beim Tausch vor den Ferien erhalten."
)

// lmfTestPlan baut einen Plan mit je n Terminen in Rückgabe und Ausgabe. vermerk steht in
// jeder vierten Zeile — so schreibt die Bibliothek („Nur Rückgabe"), und lange Vermerke
// sind der Fall, an dem die Rechnerei sonst danebenliegt.
func lmfTestPlan(n int, vermerk string) []LmfPlanAbschnitt {
	tag := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	bauen := func(titel, unter string) LmfPlanAbschnitt {
		a := LmfPlanAbschnitt{Titel: titel, Untertitel: unter}
		for i := range n {
			z := LmfPlanZeile{Datum: tag.AddDate(0, 0, i/6), Stunde: i%6 + 1,
				Klassen: fmt.Sprintf("%02dF%d", 5+i%6, i%3+1)}
			if i%4 == 0 {
				z.Vermerk = vermerk
			}
			a.Zeilen = append(a.Zeilen, z)
		}
		return a
	}
	return []LmfPlanAbschnitt{
		bauen("BÜCHERRÜCKGABE", lmfRueckgabeSatz),
		bauen("BÜCHERAUSGABE", lmfAusgabeSatz),
	}
}

func TestLmfPlanPasstAufEineSeite(t *testing.T) {
	for _, vermerk := range []string{"Nur Rückgabe", "Nur Rückgabe — Abschlussklasse, Bücher bitte gebündelt in die Bibliothek"} {
		for _, zeilen := range []int{1, 10, 20, 30, 45, 60, 100, 150} {
			t.Run(fmt.Sprintf("%d je Abschnitt, Vermerk %d Zeichen", zeilen, len(vermerk)), func(t *testing.T) {
				got, err := GenerateLmfPlan(lmfTestPlan(zeilen, vermerk), time.Now())
				if err != nil {
					t.Fatalf("GenerateLmfPlan: %v", err)
				}
				istPDF(t, got, "LMF-Plan")
				if s := seitenZahl(t, got); s != 1 {
					t.Fatalf("%d Seiten — der Plan muss auf ein Blatt", s)
				}
			})
		}
	}
}

// Ein einzelner, sehr langer Abschnitt bricht innerhalb seiner Tabelle auf die zweite
// Spalte um. Damit dort keine Termine ohne Überschrift stehen, wiederholt der Satz Titel
// und Tabellenkopf — und auch dieser Fall bleibt ein Blatt.
func TestLmfPlanEinLangerAbschnittBleibtEinBlatt(t *testing.T) {
	plan := lmfTestPlan(120, "Nur Rückgabe")
	plan[1].Zeilen = nil

	got, err := GenerateLmfPlan(plan, time.Now())
	if err != nil {
		t.Fatalf("GenerateLmfPlan: %v", err)
	}
	if s := seitenZahl(t, got); s != 1 {
		t.Fatalf("%d Seiten", s)
	}
	satz := lmfSatzWaehlen(plan)
	if len(satz.spalten) < 2 || len(satz.spalten[1]) == 0 {
		t.Fatalf("der Abschnitt sollte auf zwei Spalten umbrechen: %d Spalte(n)", len(satz.spalten))
	}
	if satz.spalten[1][0].art != lmfKopf {
		t.Fatal("die zweite Spalte beginnt ohne Fortsetzungskopf")
	}
}

// Die Verkleinerung ist ein Notnagel, kein Normalfall: Ein Plan in der Größenordnung der
// Schule (zwei Abschnitte, je ein Termin für jede Klasse) steht weiterhin einspaltig in
// voller Größe da — so, wie das Kollegium ihn aus dem Excel kennt.
func TestLmfPlanGewoehnlicherPlanBleibtUnveraendert(t *testing.T) {
	satz := lmfSatzWaehlen(lmfTestPlan(18, "Nur Rückgabe"))

	if satz.modus.spalten != 1 {
		t.Errorf("%d Spalten statt einer", satz.modus.spalten)
	}
	if satz.masse.faktor != 1 {
		t.Errorf("Faktor %.2f statt 1 — ein gewöhnlicher Plan wird nicht verkleinert", satz.masse.faktor)
	}
}

// Ein Abschnitt ohne Termine steht nicht im Plan — auch nicht als leere Überschrift.
func TestLmfPlanLeererAbschnittFaelltWeg(t *testing.T) {
	plan := lmfTestPlan(5, "")
	plan[0].Zeilen = nil

	satz := lmfSatzWaehlen(plan)
	for _, b := range satz.spalten[0] {
		if b.abschnitt == 0 {
			t.Fatal("der leere Abschnitt steht trotzdem im Satz")
		}
	}
}
