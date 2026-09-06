package api

// lmf_plan_vorgabe.go — womit ein neuer LMF-Plan beginnt. Peter, 06.09.2026, mit dem
// Excel „Bücherrückgabe" in der Hand: „Das Programm kann das doch sicherlich automatisch
// setzen — es endet immer am gleichen Tag: Donnerstags vor den Ferien zur vierten
// Stunde." Der Büchertausch vor den Sommerferien hat seinen Anker deshalb am ENDE
// (Migration 101), vorbelegt aus den Sommerferien Hessen (pkg/lmfplan/ferien.go); die
// Bücherausgabe nach den Ferien beginnt am ersten Schultag danach. Ist das Jahr nicht
// hinterlegt, sagt die Antwort es (Bekannt=false), und der Planer bittet um den Termin.

import (
	"time"

	"bibliothek/pkg/lmfplan"
	"bibliothek/repository"
)

// Peters Pläne 2026: Der Tausch endet in der 4. Stunde; die Ausgabe beginnt am ersten
// Schultag nach den Ferien in der 2. Stunde (die 1. gehört der Klassenleitung); 6 je Tag.
const (
	lmfPlanLetzteStundeVorgabe = 4
	lmfPlanStartstundeVorgabe  = 2
	lmfPlanStundenJeTagVorgabe = 6
)

// LmfPlanSommerferien sind die Sommerferien Hessen, an denen sich der Plan ausrichtet.
type LmfPlanSommerferien struct {
	Jahr int    `json:"jahr"`
	Von  string `json:"von"` // YYYY-MM-DD, leer wenn nicht bekannt
	Bis  string `json:"bis"`
	// Bekannt: false, wenn das Jahr nicht in pkg/lmfplan hinterlegt ist — der Planer
	// nennt dann das Jahr und bittet um den letzten bzw. ersten Tag von Hand.
	Bekannt bool `json:"bekannt"`
}

// LmfPlanRahmenVorgabe ist der Rahmen, mit dem ein neuer Plan im Planer beginnt: beim
// Rückgabe-Plan das Ende (Donnerstag vor den Ferien, 4. Stunde), beim Ausgabe-Plan der
// Beginn (erster Schultag nach den Ferien, 2. Stunde). Leere Tage: Ferien nicht bekannt.
type LmfPlanRahmenVorgabe struct {
	ErsterTag    string `json:"erster_tag"`
	Startstunde  int    `json:"startstunde"`
	LetzterTag   string `json:"letzter_tag"`
	LetzteStunde int    `json:"letzte_stunde"`
	StundenJeTag int    `json:"stunden_je_tag"`
}

// lmfPlanSommerferien nennt die Ferien zum Plan: Für einen laufenden Plan die, an denen
// er hängt (Rückgabe: die nächsten nach seinem Ende; Ausgabe: die seines Schuljahres),
// sonst die nächsten von heute aus — vor denen der Tausch, nach denen die Ausgabe liegt.
func lmfPlanSommerferien(art string, plan *repository.LmfPlan, laufend bool, jetzt time.Time) (LmfPlanSommerferien, lmfplan.Zeitraum) {
	rueckgabe := art == repository.LmfTerminRueckgabe
	var z lmfplan.Zeitraum
	var jahr int
	var ok bool
	switch {
	case laufend && rueckgabe:
		anker, err := planTag(plan.LetzterTag)
		if err != nil {
			anker = jetzt
		}
		z, jahr, ok = lmfplan.NaechsteSommerferien(anker, true)
	case laufend:
		beginn, err := planTag(plan.ErsterTag)
		if err != nil {
			beginn = jetzt
		}
		jahr = repository.SchuljahrBeginn(beginn).Year()
		z, ok = lmfplan.SommerferienHessen(jahr)
	default:
		z, jahr, ok = lmfplan.NaechsteSommerferien(jetzt, rueckgabe)
	}
	f := LmfPlanSommerferien{Jahr: jahr, Bekannt: ok}
	if ok {
		f.Von, f.Bis = z.Von.Format("2006-01-02"), z.Bis.Format("2006-01-02")
	}
	return f, z
}

// lmfPlanRahmenVorgabe baut den Rahmen eines neuen Plans aus den Ferien.
func lmfPlanRahmenVorgabe(art string, f LmfPlanSommerferien, z lmfplan.Zeitraum) LmfPlanRahmenVorgabe {
	v := LmfPlanRahmenVorgabe{Startstunde: lmfPlanStartstundeVorgabe, LetzteStunde: lmfPlanLetzteStundeVorgabe, StundenJeTag: lmfPlanStundenJeTagVorgabe}
	if !f.Bekannt {
		return v
	}
	if art == repository.LmfTerminRueckgabe {
		v.LetzterTag = lmfplan.DonnerstagVor(z.Von).Format("2006-01-02")
	} else {
		v.ErsterTag = lmfplan.ErsterSchultagNach(z).Format("2006-01-02")
	}
	return v
}
