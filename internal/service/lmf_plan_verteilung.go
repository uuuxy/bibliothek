package service

import (
	"fmt"
	"time"

	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// LMFPlanEntwurf ist ein geprüfter Plan vor dem Rechnen: Rahmen, Zeilen, je Zeile der feste
// Platz (nil, wenn sie fließt) und die ausgelassenen Klassen. Die Tür füllt ihn aus der Anfrage.
type LMFPlanEntwurf struct {
	Plan        repository.LMFPlan
	Zeilen      []repository.LMFPlanZeile
	Fest        []*lmfplan.Platz
	Ausgelassen []string
}

// VerteileLMFPlan rechnet die Plätze der Zeilen über die Schultage, also ohne die freien Tage
// des Plans und die Feiertage und Ferien aus pkg/lmfplan, und nennt die Ausfälle vom ersten Tag
// bis zum letzten Platz. Der Rückgabe-Plan fließt vom Ende her rückwärts; sein Beginn
// (e.Plan.ErsterTag und Startstunde) ist danach der früheste Platz.
func VerteileLMFPlan(entwurf *LMFPlanEntwurf) ([]lmfplan.Platz, []lmfplan.Ausfall, error) {
	rueckwaerts := entwurf.Plan.Art == repository.LMFTerminRueckgabe
	ankerTag := entwurf.Plan.ErsterTag
	if rueckwaerts {
		ankerTag = entwurf.Plan.LetzterTag
	}
	anker, err := schulzeit.Kalendertag(ankerTag)
	if err != nil {
		return nil, nil, err
	}
	frei, err := lmfPlanFreieZeitraeume(entwurf.Plan.FreieTage)
	if err != nil {
		return nil, nil, err
	}
	var plaetze []lmfplan.Platz
	if rueckwaerts {
		ende := lmfplan.Ende{LetzterTag: anker, LetzteStunde: entwurf.Plan.LetzteStunde, StundenJeTag: entwurf.Plan.StundenJeTag}
		plaetze = lmfplan.VerteileRueckwaerts(ende, entwurf.Fest, lmfplan.Schultage(frei))
		if b, ok := lmfplan.Beginn(plaetze); ok {
			entwurf.Plan.ErsterTag, entwurf.Plan.Startstunde = b.Datum.Format(time.DateOnly), b.Stunde
		}
	} else {
		r := lmfplan.Rahmen{ErsterTag: anker, Startstunde: entwurf.Plan.Startstunde, StundenJeTag: entwurf.Plan.StundenJeTag}
		plaetze = lmfplan.VerteileMit(r, entwurf.Fest, lmfplan.Schultage(frei))
	}
	ersterTag, err := schulzeit.Kalendertag(entwurf.Plan.ErsterTag)
	if err != nil {
		return nil, nil, err
	}
	ausfaelle := lmfplan.Ausfaelle(ersterTag, spaetesterPlatz(ersterTag, plaetze), frei)
	// Je Zeile ein Platz: Beide Verteiler liefern einen Platz je Eintrag in e.Fest, und e.Fest
	// hat einen je Zeile, außer bei weniger als einer Stunde je Tag; dann geben beide eine leere
	// Liste zurück. Die Tür verlangt 1 bis 12. Die Zusicherung steht trotzdem hier, weil Vorschau
	// und Speichern mit plaetze[i] zugreifen und ein Indexfehler keine lesbare Meldung ist.
	if len(plaetze) != len(entwurf.Zeilen) {
		return nil, nil, fmt.Errorf("verteilung: %d Plätze für %d Zeilen", len(plaetze), len(entwurf.Zeilen))
	}
	return plaetze, ausfaelle, nil
}

// lmfPlanFreieZeitraeume macht aus den freien Tagen des Plans eintägige Zeiträume; ohne Grund
// heißt der Tag „freier Tag".
func lmfPlanFreieZeitraeume(tage []repository.LMFFreierTag) ([]lmfplan.Zeitraum, error) {
	frei := []lmfplan.Zeitraum{}
	for _, f := range tage {
		tag, err := schulzeit.Kalendertag(f.Datum)
		if err != nil {
			return nil, err
		}
		grund := f.Grund
		if grund == "" {
			grund = "freier Tag"
		}
		frei = append(frei, lmfplan.Zeitraum{Von: tag, Bis: tag, Name: grund})
	}
	return frei, nil
}

// spaetesterPlatz nennt den spätesten Tag der Plätze, mindestens ab.
func spaetesterPlatz(ab time.Time, plaetze []lmfplan.Platz) time.Time {
	letzter := ab
	for _, p := range plaetze {
		if p.Datum.After(letzter) {
			letzter = p.Datum
		}
	}
	return letzter
}
