package service

import (
	"context"
	"sort"
	"time"

	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Die Kopplung des LMF-Plans an die Fristen: Der Rückgabe-Termin einer Klasse ist die Frist
// ihrer Lernmittel. Beim Ausleihen liest resolveCheckoutDueDate den Termin; hier folgt der
// Bestand dem Plan, wenn er sich ändert. Sonst gälte der Termin nur für Bücher, die nach dem
// Eintrag ausgeliehen werden, und der Plan entsteht im Mai, wenn die Bücher seit September
// draußen sind.
//
// Nur Rückgabe-Termine setzen Fristen. Verliert eine Klasse ihren Termin (aus der Zeile
// genommen, Zeile gelöscht, Plan verworfen), gehen genau die Fristen, die auf diesem Tag lagen,
// an den Stichtag zurück; eine Frist von Hand bleibt stehen. Angefasst werden nur Fristen im
// Schuljahr des Termins: Eine mehrjährige Ausleihe folgt dem Plan nicht.

// KoppleLmfPlanFristen gleicht die Fristen ab, nachdem ein ganzer Plan gespeichert,
// veröffentlicht oder verworfen wurde: alt sind die Zeilen vorher, neu die Zeilen nachher (nil
// beim Verwerfen). Gelesen und geschrieben wird über ex, die Transaktion der Tür: Plan und
// Fristen gelten zusammen oder zusammen nicht. Liefert die Zahl der umgeschriebenen Ausleihen.
//
// Maßgeblich ist der früheste Termin einer Klasse, nicht ihre letzte Zeile: Ein Plan kann eine
// Klasse zweimal nennen, etwa mit einem nachgeschobenen Termin, und der Ausleihdienst nimmt
// beim Ausleihen ebenfalls den nächsten Termin nach heute (RueckgabeTerminLage). Mit zwei
// Regeln hätte dasselbe Schulbuch je nach Weg eine andere Frist.
//
// Zuerst kehren die Klassen, die im neuen Plan nicht mehr stehen, zum Stichtag zurück, dann
// bekommt jede Klasse des neuen Plans ihren frühesten Termin.
func KoppleLmfPlanFristen(ctx context.Context, repo *repository.LmfTerminRepository, ex repository.DBQueryer, art string, alt, neu []repository.LmfPlanZeile) (int64, error) {
	if art != repository.LmfTerminRueckgabe {
		return 0, nil
	}
	termine := fruehesteTermineJeKlasse(neu)
	var gesamt int64
	for _, z := range alt {
		n, err := loeseLmfFristenVomTermin(ctx, repo, ex, z.Datum, ohneTermin(z.Klassen, termine))
		if err != nil {
			return gesamt, err
		}
		gesamt += n
	}
	for _, datum := range sortierteSchluessel(termine) {
		n, err := setzeLmfFristenAufTermin(ctx, repo, ex, datum, termine[datum])
		if err != nil {
			return gesamt, err
		}
		gesamt += n
	}
	return gesamt, nil
}

// loeseLmfFristenVomTermin bringt die Fristen der Klassen, die ihren Rückgabe-Termin an diesem
// Tag verlieren, zurück zum Stichtag.
func loeseLmfFristenVomTermin(ctx context.Context, repo *repository.LmfTerminRepository, ex repository.DBQueryer, datum string, verlierer []string) (int64, error) {
	if len(verlierer) == 0 {
		return 0, nil
	}
	altTag, err := schulzeit.Kalendertag(datum)
	if err != nil {
		return 0, err
	}
	einstellungen, err := repository.EinstellungenUeber(ctx, ex)
	if err != nil {
		return 0, err
	}
	// Der Stichtag des Schuljahres, in dem der Termin lag, nicht der nächste ab heute: Angefasst
	// werden nur Fristen dieses Schuljahres, und sie gehen dorthin zurück, wo sie ohne Plan
	// gestanden hätten. Und nur Fristen, die auf dem Termin lagen: Eine Frist von Hand bleibt.
	stichtag := repository.LmfStichtagImSchuljahr(altTag, einstellungen.LmfStichtag)
	von, bis := schuljahrGrenzen(altTag)
	return repo.SetzeLernmittelFristFuerKlassenIn(ctx, ex, verlierer,
		TagesEndeInSchulzeitzone(stichtag), von, bis, &altTag)
}

// setzeLmfFristenAufTermin setzt die Fristen der Klassen eines Rückgabe-Termins auf dessen Tag.
func setzeLmfFristenAufTermin(ctx context.Context, repo *repository.LmfTerminRepository, ex repository.DBQueryer, datum string, klassen []string) (int64, error) {
	if len(klassen) == 0 {
		return 0, nil
	}
	neuTag, err := schulzeit.Kalendertag(datum)
	if err != nil {
		return 0, err
	}
	von, bis := schuljahrGrenzen(neuTag)
	return repo.SetzeLernmittelFristFuerKlassenIn(ctx, ex, klassen,
		TagesEndeInSchulzeitzone(neuTag), von, bis, nil)
}

// schuljahrGrenzen liefert [1. August des Schuljahres, 1. August des nächsten).
func schuljahrGrenzen(tag time.Time) (time.Time, time.Time) {
	von := repository.SchuljahrBeginn(tag)
	return von, von.AddDate(1, 0, 0)
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

// fruehesteTermineJeKlasse gruppiert die Klassen des Plans nach ihrem frühesten Datum: Datum →
// Klassen. Verglichen wird über KlassenSchluessel; beide Seiten kommen aus dem Vokabular der
// Klassen, Schreibvarianten kommen hier nicht an.
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

// sortierteSchluessel ordnet die Datums-Gruppen: Die Zahl der angefassten Ausleihen soll nicht
// von der Reihenfolge einer Map abhängen.
func sortierteSchluessel(m map[string][]string) []string {
	schluessel := make([]string, 0, len(m))
	for k := range m {
		schluessel = append(schluessel, k)
	}
	sort.Strings(schluessel)
	return schluessel
}
