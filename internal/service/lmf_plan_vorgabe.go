package service

import (
	"time"

	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Womit ein neuer LMF-Plan beginnt. Der Büchertausch vor den Sommerferien endet immer am selben
// Tag, dem Donnerstag vor den Ferien zur vierten Stunde; sein Anker liegt deshalb am Ende. Die
// Ausgabe nach den Ferien beginnt am ersten Schultag danach. Die Ferien kommen aus der
// Ferientabelle (pkg/lmfplan); kennt sie das Jahr nicht, sagt die Antwort es, und der Planer
// bittet um den Tag.

// Die Pläne der Schule: Der Tausch endet in der 4. Stunde; die Ausgabe beginnt in der 2. Stunde,
// die 1. gehört der Klassenleitung; 6 Stunden je Tag. Der Planer führt dieselben Zahlen für
// den Zustand vor dem ersten Laden (lmf_plan_vorgabe_paritaet_test.go).
const (
	lmfPlanLetzteStundeVorgabe = 4
	lmfPlanStartstundeVorgabe  = 2
	lmfPlanStundenJeTagVorgabe = 6
)

// LmfPlanFerien sind die Sommerferien, an denen sich ein Plan ausrichtet. Von und Bis sind
// Kalendertage (JJJJ-MM-TT) und leer, wenn die Ferientabelle das Jahr nicht kennt.
type LmfPlanFerien struct {
	Jahr    int
	Von     string
	Bis     string
	Bekannt bool
}

// LmfPlanRahmen ist der Rahmen, mit dem ein neuer Plan beginnt: beim Rückgabe-Plan das Ende
// (Donnerstag vor den Ferien, 4. Stunde), beim Ausgabe-Plan der Beginn (erster Schultag nach
// den Ferien, 2. Stunde). Ein leerer Tag heißt: Ferien nicht bekannt.
type LmfPlanRahmen struct {
	ErsterTag    string
	Startstunde  int
	LetzterTag   string
	LetzteStunde int
	StundenJeTag int
}

// LmfPlanSommerferien nennt die Ferien zum Plan: für einen laufenden Plan die, an denen er
// hängt, sonst die nächsten von heute aus, vor denen der Tausch und nach denen die Ausgabe
// liegt.
func LmfPlanSommerferien(art string, plan *repository.LmfPlan, laufend bool, jetzt time.Time, tab lmfplan.Ferientabelle) (LmfPlanFerien, lmfplan.Zeitraum) {
	rueckgabe := art == repository.LmfTerminRueckgabe
	var z lmfplan.Zeitraum
	var jahr int
	var ok bool
	switch {
	case laufend:
		z, jahr, ok = lmfPlanEigeneFerien(rueckgabe, plan, jetzt, tab)
	default:
		z, jahr, ok = tab.Naechste(jetzt, rueckgabe)
		// Ein Plan, der vorbei ist, dessen Ferien aber noch bevorstehen: Zwischen dem Donnerstag,
		// an dem der Büchertausch endet, und dem Montag, an dem die Ferien beginnen, nennt
		// Naechste(heute) dieselben Ferien wie der abgelaufene Plan. Daraus entstünde derselbe
		// schuljahr_beginn, und „Plan speichern" träfe über ON CONFLICT (art, schuljahr_beginn)
		// den alten, weiterhin veröffentlichten Plan. Der nächste Plan muss über die Ferien des
		// vorigen hinaus. Nicht pauschal „Planjahr + 1": Liegt der vorige Plan Jahre zurück,
		// ist Naechste(heute) schon weiter, und das gilt dann.
		if plan != nil {
			if _, planJahr, planOk := lmfPlanEigeneFerien(rueckgabe, plan, jetzt, tab); planOk && jahr <= planJahr {
				jahr = planJahr + 1
				z, ok = tab.Sommerferien(jahr)
			}
		}
	}
	f := LmfPlanFerien{Jahr: jahr, Bekannt: ok}
	if ok {
		f.Von, f.Bis = z.Von.Format(time.DateOnly), z.Bis.Format(time.DateOnly)
	}
	return f, z
}

// lmfPlanEigeneFerien nennt die Ferien, an denen ein bestehender Plan hängt: Rückgabe die
// nächsten nach seinem Ende (vor ihnen lag der Tausch), Ausgabe die seines Schuljahres. Dieselbe
// Rechnung für den laufenden Plan und für die Frage, worüber ein Nachfolger hinaus muss.
func lmfPlanEigeneFerien(rueckgabe bool, plan *repository.LmfPlan, jetzt time.Time, tab lmfplan.Ferientabelle) (lmfplan.Zeitraum, int, bool) {
	if rueckgabe {
		anker, err := schulzeit.Kalendertag(plan.LetzterTag)
		if err != nil {
			anker = jetzt
		}
		return tab.Naechste(anker, true)
	}
	beginn, err := schulzeit.Kalendertag(plan.ErsterTag)
	if err != nil {
		beginn = jetzt
	}
	jahr := repository.SchuljahrBeginn(beginn).Year()
	z, ok := tab.Sommerferien(jahr)
	return z, jahr, ok
}

// LmfPlanRahmenVorgabe baut den Rahmen eines neuen Plans aus den Ferien.
func LmfPlanRahmenVorgabe(art string, f LmfPlanFerien, z lmfplan.Zeitraum) LmfPlanRahmen {
	v := LmfPlanRahmen{Startstunde: lmfPlanStartstundeVorgabe, LetzteStunde: lmfPlanLetzteStundeVorgabe, StundenJeTag: lmfPlanStundenJeTagVorgabe}
	if !f.Bekannt {
		return v
	}
	if art == repository.LmfTerminRueckgabe {
		v.LetzterTag = lmfplan.DonnerstagVor(z.Von).Format(time.DateOnly)
	} else {
		v.ErsterTag = lmfplan.ErsterSchultagNach(z).Format(time.DateOnly)
	}
	return v
}
