package leser

import (
	"context"
	"net/http"
	"sort"
	"strconv"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/internal/ausweis"
	"bibliothek/repository"
)

// Die Übersetzung „Jahrgang → Klassen" für die Leserdatei.
//
// Das Sichtungsprotokoll des Medienzentrums nennt beides: Die Schülerdatei habe „keine
// Sortier- oder Filteroption nach Klassen bzw. Jahrgängen", und unter Anpassungswünschen:
// „Derzeit ist eine Auswahl nach Klassen möglich. Gemeint ist jedoch eine Auswahl nach
// Jahrgängen."
//
// Warum das in Go steht und nicht im SQL: An dieser Schule heißt der elfte Jahrgang „ET",
// der zwölfte „12T" und der dreizehnte „13T" (Klassenschema, Migration 087). Die Zahl
// am Anfang der Klasse gibt es für „ET" nicht — genau die liest
// LMFAbfragen.LadeKlassenMitSchuelern für die Sortierung des LMF-Plans, wo ein 99 als
// „irgendwo oben" folgenlos ist. Als Filter wäre dieselbe Zeile falsch: Die Oberstufe fiele
// heraus, und zwar lautlos — eine leere Liste sieht aus wie „niemand da".
//
// Gerechnet wird deshalb mit ausweis.AblaufJahrgang, der einen Ableitung, die das
// Schema dieser Schule kennt (05F1→5, 10H1→10, ET→11, 12T→12,
// 13T→13, Q1→12, Q3→13). Eine achte Auslegung des Klassennamens wäre der Anfang genau
// der Geschichte, die dieses Projekt sonst meidet.

// klassenFilter bestimmt, auf welche Klassen eine Listenabfrage eingegrenzt wird.
//
// Die Rückgabe unterscheidet drei Fälle, und diese Unterscheidung ist der Grund, warum
// die Funktion nicht einfach ein []string liefert:
//
//   - kein Filter angefragt        → (nil, false, nil): die Liste zeigt alles
//   - Filter mit Treffern          → (Klassen, false, nil)
//   - Filter ohne eine einzige Klasse → (nil, true, nil): die Liste zeigt nichts
//
// Ohne den dritten Fall zeigte ein Jahrgang, zu dem es keine Klasse gibt, die ganze
// Kartei statt einer leeren Liste — die Sorte Fehler, die niemand bemerkt, weil eine
// volle Liste nach „geht" aussieht.
func klassenFilter(ctx context.Context, lmfAbfragen *repository.LMFAbfragen, klasse, jahrgang string) (klassen []string, leer bool, err error) {
	if klasse != "" {
		return []string{klasse}, false, nil
	}
	if jahrgang == "" {
		return nil, false, nil
	}

	gesucht, wandelFehler := strconv.Atoi(jahrgang)
	if wandelFehler != nil || gesucht < 1 || gesucht > 13 {
		// Ein unlesbarer Jahrgang ist kein Grund, alles zu zeigen: Wer nach „Jahrgang 5"
		// filtert und sich vertippt, bekommt lieber nichts als die ganze Schule.
		return nil, true, nil
	}

	alle, err := lmfAbfragen.LadeKlassenMitSchuelern(ctx)
	if err != nil {
		return nil, false, err
	}
	for _, k := range alle {
		if _, aktuell, ok := ausweis.AblaufJahrgang(k.Name); ok && aktuell == gesucht {
			klassen = append(klassen, k.Name)
		}
	}
	return klassen, len(klassen) == 0, nil
}

// jahrgaengeMitSchuelern nennt die Jahrgänge, in denen tatsächlich jemand ist —
// aufsteigend, ohne Doppelte.
//
// Warum der Server das sagt und nicht der Browser: Die Ableitung „Klassenname →
// Jahrgang" kennt das Schema dieser Schule, und sie soll genau einmal existieren. Eine
// zweite Fassung in JavaScript wäre bis zur nächsten Umbenennung einer Klasse still
// richtig und danach still falsch.
func jahrgaengeMitSchuelern(ctx context.Context, lmfAbfragen *repository.LMFAbfragen) ([]int, error) {
	alle, err := lmfAbfragen.LadeKlassenMitSchuelern(ctx)
	if err != nil {
		return nil, err
	}
	gesehen := map[int]bool{}
	jahrgaenge := []int{}
	for _, k := range alle {
		_, aktuell, ok := ausweis.AblaufJahrgang(k.Name)
		if !ok || gesehen[aktuell] {
			continue
		}
		gesehen[aktuell] = true
		jahrgaenge = append(jahrgaenge, aktuell)
	}
	sort.Ints(jahrgaenge)
	return jahrgaenge, nil
}

// Jahrgaenge liefert die besetzten Jahrgänge für das Filter-Auswahlfeld.
//
// @Summary      Besetzte Jahrgänge
// @Tags         students
// @Produce      json
// @Success      200 {array} int
// @Router       /jahrgaenge [get]
func (t *Tueren) Jahrgaenge(lmfAbfragen *repository.LMFAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		jahrgaenge, err := jahrgaengeMitSchuelern(r.Context(), lmfAbfragen)
		if err != nil {
			return apierrors.Internal("Jahrgänge konnten nicht gelesen werden", err)
		}
		tuer.AntworteJSON(w, http.StatusOK, jahrgaenge)
		return nil
	})
}
