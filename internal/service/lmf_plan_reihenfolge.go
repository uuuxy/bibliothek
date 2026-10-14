package service

import (
	"slices"
	"strings"
	"time"

	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Die Reihenfolge, mit der ein neuer LMF-Plan beginnt, und was die Tür dazu über einen Plan
// fragt: welche Klassen seine Art auslässt, ob er vorbei ist, welcher Vermerk vorbelegt wird.
// Den Rahmen eines neuen Plans rechnet lmf_plan_vorgabe.go.

// Woher die Reihenfolge eines neuen Plans kommt. Die Werte stehen in der Antwort an den Planer.
const (
	lmfPlanQuelleVorjahr = "vorjahr"
	lmfPlanQuelleRegel   = "regel"
)

// LMFPlanReihenfolge ist die Reihenfolge, mit der ein neuer Plan beginnt: die Zeilen und die
// Klassen, die draußen bleiben.
type LMFPlanReihenfolge struct {
	// Quelle nennt, woher sie kommt: aus dem letzten Plan der Art oder aus der Regel.
	Quelle      string
	Zeilen      []repository.LMFPlanZeile
	Ausgelassen []string
}

// IstAusgelassenImLMFPlan ist die eine Regel, welche Klasse nicht in den Plan einer Art gehört. Der
// Büchertausch lässt die Oberstufe aus, sie organisiert sich selbst; die Ausgabe nach den
// Ferien alles außer den Eingangsjahrgängen, nur die neu gebildeten Klassen bekommen Bücher.
// Der Vorschlag und die Liste „bleiben draußen" des Planers fragen sie hier.
func IstAusgelassenImLMFPlan(art string, eingang []int, k repository.KlasseImPlan) bool {
	if art == repository.LMFTerminAusgabe {
		return !slices.Contains(eingang, k.Jahrgang)
	}
	return k.Oberstufe
}

// LMFPlanReihenfolgeVorgabe baut die Reihenfolge, mit der ein neuer Plan beginnt: das Vorjahr,
// wenn es eines gibt, ergänzt um Klassen, die inzwischen neu sind; sonst die Regel. Klassen ohne
// Schüler bleiben aus dem Vorjahr erhalten, eine „7G6" kann wiederkommen; der Planer markiert
// sie. Beim Ausgabe-Plan kommen nur die Eingangsjahrgänge in die Reihenfolge, alle anderen
// Klassen stehen unter den ausgelassenen.
func LMFPlanReihenfolgeVorgabe(art string, eingang []int, vorjahr bool, stand repository.LMFPlanStand, klassen []repository.KlasseImPlan) LMFPlanReihenfolge {
	v := LMFPlanReihenfolge{Quelle: lmfPlanQuelleRegel, Zeilen: []repository.LMFPlanZeile{}, Ausgelassen: []string{}}
	bekannt := map[string]bool{}
	if vorjahr {
		v.Quelle = lmfPlanQuelleVorjahr
		for _, z := range stand.Zeilen {
			v.Zeilen = append(v.Zeilen, repository.LMFPlanZeile{Klassen: z.Klassen, Vermerk: z.Vermerk})
			for _, k := range z.Klassen {
				bekannt[repository.KlassenSchluessel(k)] = true
			}
		}
		v.Ausgelassen = append(v.Ausgelassen, stand.Ausgelassen...)
		for _, k := range stand.Ausgelassen {
			bekannt[repository.KlassenSchluessel(k)] = true
		}
	}
	for _, k := range klassen {
		if bekannt[repository.KlassenSchluessel(k.Name)] {
			continue
		}
		if IstAusgelassenImLMFPlan(art, eingang, k) {
			v.Ausgelassen = append(v.Ausgelassen, k.Name)
			continue
		}
		v.Zeilen = append(v.Zeilen, repository.LMFPlanZeile{Klassen: []string{k.Name}})
	}
	if v.Quelle == lmfPlanQuelleRegel {
		// Die zwei Zeilen ohne Klasse, mit denen die Pläne der Schule enden: wer nicht dran war,
		// und dann das Aufräumen. Das Vorjahr bringt sie von selbst mit; die Regel setzt sie,
		// damit der erste Plan aussieht wie der der Schule.
		v.Zeilen = append(v.Zeilen,
			repository.LMFPlanZeile{Klassen: []string{}, Vermerk: "Nachzügler"},
			repository.LMFPlanZeile{Klassen: []string{}, Vermerk: "Aufräumen"})
	}
	return v
}

// IstLMFPlanVorbei sagt, ob der letzte Platz des Plans vor dem heutigen Kalendertag der Schule
// liegt; bei einem Plan ohne Zeilen zählt sein erster Tag. Der Planer bietet dann den nächsten
// Plan an.
func IstLMFPlanVorbei(stand repository.LMFPlanStand, jetzt time.Time) bool {
	letzter := stand.Plan.ErsterTag
	for _, z := range stand.Zeilen {
		if z.Datum > letzter {
			letzter = z.Datum
		}
	}
	heute := jetzt.In(schulzeit.Zone()).Format(time.DateOnly)
	return letzter < heute
}

// lmfPlanVermerkNurRueckgabe ist der Text, den der Vorschlag Zeilen voranstellt, deren Klassen
// vor den Ferien nur abgeben: Abschlussklassen und Klassen, die neu gebildet werden.
const lmfPlanVermerkNurRueckgabe = "nur Rückgabe"

// BelegeVermerkNurRueckgabeVor belegt den Vermerk der Zeilen vor, in denen jede Klasse nur
// abgibt, außer der Vermerk aus dem Vorjahr sagt es schon. Danach gehört der Text der
// Bibliothek: Sie kann ihn ändern oder löschen, nichts rechnet ihn nach.
func BelegeVermerkNurRueckgabeVor(zeilen []repository.LMFPlanZeile, nurRueckgabe []string) {
	ist := map[string]bool{}
	for _, k := range nurRueckgabe {
		ist[repository.KlassenSchluessel(k)] = true
	}
	for i := range zeilen {
		z := &zeilen[i]
		if len(z.Klassen) == 0 || strings.Contains(strings.ToLower(z.Vermerk), strings.ToLower(lmfPlanVermerkNurRueckgabe)) {
			continue
		}
		alle := true
		for _, k := range z.Klassen {
			if !ist[repository.KlassenSchluessel(k)] {
				alle = false
				break
			}
		}
		if !alle {
			continue
		}
		if z.Vermerk == "" {
			z.Vermerk = lmfPlanVermerkNurRueckgabe
		} else {
			z.Vermerk = lmfPlanVermerkNurRueckgabe + " · " + z.Vermerk
		}
	}
}
