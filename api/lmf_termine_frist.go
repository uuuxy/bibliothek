package api

// lmf_termine_frist.go — die Kopplung des Plans an die Fristen (Register, Entscheidung 3a,
// Peter 05.09.2026: „das wäre doch logisch"). Der Rückgabe-Termin einer Klasse IST die
// Frist ihrer Lernmittel: Beim Ausleihen liest der Ausleihdienst den Termin
// (internal/service, resolveCheckoutDueDate); hier folgt der Bestand dem Plan, wenn er
// sich ändert — sonst nützte der Termin nur den Büchern, die NACH dem Eintrag ausgeliehen
// werden, und der Plan entsteht im Mai, die Bücher sind seit September draußen.
//
// Regeln: Nur Rückgabe-Termine setzen Fristen; Ausgabe-Zeilen nicht. Verliert eine Klasse
// ihren Termin (aus der Zeile genommen, Art gewechselt, Zeile gelöscht), gehen genau die
// Fristen, die auf diesem Tag lagen, an den globalen Stichtag zurück — Fristen von Hand
// bleiben stehen. Angefasst werden nur Fristen im Schuljahr des Termins; eine mehrjährige
// Ausleihe folgt dem Plan nicht.

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"bibliothek/internal/service"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// koppleLmfFristen gleicht die Fristen nach einer Planänderung ab. alt = Stand vor der
// Änderung (nil beim Anlegen), neu = Stand danach (nil beim Löschen). Liefert die Zahl
// der umgeschriebenen Ausleihen.
func (s *Server) koppleLmfFristen(ctx context.Context, alt, neu *repository.LmfTermin) (int64, error) {
	repo := repository.NewLmfTerminRepository(s.DB.Pool)
	var gesamt int64

	// 1. Klassen, die ihren Rückgabe-Termin verlieren → zurück zum Stichtag.
	if alt != nil && alt.Art == repository.LmfTerminRueckgabe {
		verlierer := alt.Klassen
		if neu != nil && neu.Art == repository.LmfTerminRueckgabe {
			verlierer = ohne(alt.Klassen, neu.Klassen)
		}
		if len(verlierer) > 0 {
			altTag, err := planTag(alt.Datum)
			if err != nil {
				return gesamt, err
			}
			einstellungen, err := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(ctx)
			if err != nil {
				return gesamt, err
			}
			stichtag := stichtagImSchuljahr(altTag, einstellungen.LmfStichtag)
			von, bis := schuljahrGrenzen(altTag)
			n, err := repo.SetzeLernmittelFristFuerKlassen(ctx, verlierer,
				service.TagesEndeInSchulzeitzone(stichtag), von, bis, &altTag)
			if err != nil {
				return gesamt, err
			}
			gesamt += n
		}
	}

	// 2. Klassen des (neuen) Rückgabe-Termins → Frist ist der Termin.
	if neu != nil && neu.Art == repository.LmfTerminRueckgabe && len(neu.Klassen) > 0 {
		neuTag, err := planTag(neu.Datum)
		if err != nil {
			return gesamt, err
		}
		von, bis := schuljahrGrenzen(neuTag)
		n, err := repo.SetzeLernmittelFristFuerKlassen(ctx, neu.Klassen,
			service.TagesEndeInSchulzeitzone(neuTag), von, bis, nil)
		if err != nil {
			return gesamt, err
		}
		gesamt += n
	}
	return gesamt, nil
}

// planTag liest das Plan-Datum (JJJJ-MM-TT) als Kalendertag der Schule.
func planTag(datum string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", datum, schulzeit.Zone())
}

// schuljahrGrenzen liefert [1. August des Schuljahres, 1. August des nächsten).
func schuljahrGrenzen(tag time.Time) (time.Time, time.Time) {
	von := repository.SchuljahrBeginn(tag)
	return von, von.AddDate(1, 0, 0)
}

// stichtagImSchuljahr legt den globalen Stichtag („MM-TT", Vorgabe 31.07.) in das
// Schuljahr des Tages — dieselbe Rechnung wie die Fristberechnung beim Ausleihen.
func stichtagImSchuljahr(tag time.Time, stichtag string) time.Time {
	monat, tagImMonat := time.July, 31
	if teile := strings.SplitN(stichtag, "-", 2); len(teile) == 2 {
		m, err1 := strconv.Atoi(teile[0])
		d, err2 := strconv.Atoi(teile[1])
		if err1 == nil && err2 == nil && m >= 1 && m <= 12 && d >= 1 && d <= 31 {
			monat, tagImMonat = time.Month(m), d
		}
	}
	von := repository.SchuljahrBeginn(tag)
	jahr := von.Year() + 1
	if monat >= time.August {
		jahr = von.Year()
	}
	return time.Date(jahr, monat, tagImMonat, 12, 0, 0, 0, schulzeit.Zone())
}

// ohne liefert die Einträge von a, die nicht in b stehen (Klassen über den Normschlüssel).
func ohne(a, b []string) []string {
	drin := map[string]bool{}
	for _, k := range b {
		drin[repository.KlassenSchluessel(k)] = true
	}
	var rest []string
	for _, k := range a {
		if !drin[repository.KlassenSchluessel(k)] {
			rest = append(rest, k)
		}
	}
	return rest
}

// koppleLmfPlanFristen gleicht die Fristen nach dem Speichern oder Verwerfen eines
// ganzen Plans ab (Migration 097): alt = Zeilen vorher, neu = Zeilen nachher (nil beim
// Verwerfen). Nur Rückgabe-Pläne setzen Fristen.
//
// Maßgeblich ist der FRÜHESTE Termin einer Klasse, nicht ihre letzte Zeile. Eine Klasse
// darf mehrfach im Plan stehen — im Plan der Schule steht 9H1 am ersten Tag und noch
// einmal als Nachzügler —, und der Ausleihdienst nimmt beim Ausleihen den nächsten
// Termin ab heute (RueckgabeTerminFuerKlasse, MIN). Bis 05.09.2026 abends lief hier eine
// Schleife über die Zeilen, deren letzte gewann: Dasselbe Schulbuch hatte je nach Weg
// eine andere Frist — schon draußen den späteren Termin, am selben Tag ausgeliehen den
// früheren. Zwei Regeln für dieselbe Frage; Gate:
// TestLmfPlan_KlasseZweimalImPlan_FruehesterTerminGilt.
//
// Zuerst kehren Klassen, die im neuen Plan gar nicht mehr stehen, zum Stichtag zurück —
// genau die Fristen, die auf ihrem alten Termin-Tag lagen —, dann bekommt jede Klasse
// des neuen Plans ihren frühesten Termin.
func (s *Server) koppleLmfPlanFristen(ctx context.Context, art string, alt, neu []repository.LmfPlanZeile) (int64, error) {
	if art != repository.LmfTerminRueckgabe {
		return 0, nil
	}
	termine := fruehesteTermineJeKlasse(neu)
	var gesamt int64
	for _, z := range alt {
		verlierer := ohneTermin(z.Klassen, termine)
		if len(verlierer) == 0 {
			continue
		}
		n, err := s.koppleLmfFristen(ctx, &repository.LmfTermin{Art: art, Datum: z.Datum, Klassen: verlierer}, nil)
		if err != nil {
			return gesamt, err
		}
		gesamt += n
	}
	for _, datum := range sortierteSchluessel(termine) {
		n, err := s.koppleLmfFristen(ctx, nil, &repository.LmfTermin{Art: art, Datum: datum, Klassen: termine[datum]})
		if err != nil {
			return gesamt, err
		}
		gesamt += n
	}
	return gesamt, nil
}

// fruehesteTermineJeKlasse gruppiert die Klassen des Plans nach ihrem FRÜHESTEN Datum:
// Ergebnis ist Datum → Klassen. Verglichen wird über KlassenSchluessel; beide Seiten
// kommen kanonisiert aus dem Vokabular (der Trigger schreibt die registrierte
// Schreibweise), Schreibvarianten können hier also nicht ankommen.
func fruehesteTermineJeKlasse(zeilen []repository.LmfPlanZeile) map[string][]string {
	frueheste := map[string]string{} // Klassenschlüssel → Datum
	namen := map[string]string{}     // Klassenschlüssel → Schreibweise
	for _, z := range zeilen {
		for _, k := range z.Klassen {
			schluessel := repository.KlassenSchluessel(k)
			if bisher, da := frueheste[schluessel]; !da || z.Datum < bisher {
				frueheste[schluessel] = z.Datum
			}
			namen[schluessel] = k
		}
	}
	nachDatum := map[string][]string{}
	for schluessel, datum := range frueheste {
		nachDatum[datum] = append(nachDatum[datum], namen[schluessel])
	}
	for datum := range nachDatum {
		sort.Strings(nachDatum[datum])
	}
	return nachDatum
}

// ohneTermin liefert die Klassen aus a, die im neuen Plan keinen Termin mehr haben.
func ohneTermin(a []string, termine map[string][]string) []string {
	var drin []string
	for _, klassen := range termine {
		drin = append(drin, klassen...)
	}
	return ohne(a, drin)
}

// sortierteSchluessel macht die Reihenfolge der Datums-Gruppen deterministisch — die
// Zahl der angefassten Ausleihen soll nicht von der Map-Reihenfolge abhängen.
func sortierteSchluessel(m map[string][]string) []string {
	schluessel := make([]string, 0, len(m))
	for k := range m {
		schluessel = append(schluessel, k)
	}
	sort.Strings(schluessel)
	return schluessel
}
