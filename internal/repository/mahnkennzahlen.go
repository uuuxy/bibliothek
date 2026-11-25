package repository

import "context"

// MahnKennzahlen zählen die offenen, überfälligen Ausleihen: zusammen, die längste Dauer in
// Tagen und die Verteilung nach Dauer. Sie tragen keinen Namen und keinen Titel.
type MahnKennzahlen struct {
	Ueberfaellig, MaxTage        int
	Bis14, Bis30, Bis60, Ueber60 int
}

// LadeMahnKennzahlen liest alle Zahlen in einem Durchgang über die Ausleihen, ohne Verbindung
// zu Lesern: Die Statistik soll kein personenbezogenes Feld erreichen. Der Titel dient nur dem
// Bestandsfilter; die Antwort trägt nur Zahlen. Dauerleihen zählen nicht, sie werden nicht
// überfällig.
//
// typFilter ist das Stück aus BestandsFilterBedingung, mit dem die Statistik auch ihre übrigen
// Kacheln filtert: Die Quote „überfällig ÷ laufende Ausleihen" teilte sonst eine Zahl der ganzen
// Schule durch eine Zahl aus Freihand oder LMF. Aus demselben Grund zählen Geräte nicht: Der
// Nenner („aktuell verliehen“) zählt Bücher; überfällige Geräte stehen im Mahnwesen.
//
// Die Dauer zählt in vollen Tagen wie im Mahnwesen (tage_ueberfaellig, ab 15 Tagen „Mahnung“):
// Ein Verzug von 14 Tagen und ein paar Stunden steht in beiden unter „bis 14 Tage“.
func LadeMahnKennzahlen(ctx context.Context, db Datenbank, typFilter string) (MahnKennzahlen, error) {
	var k MahnKennzahlen
	err := db.QueryRow(ctx, `
			WITH offen AS (
				SELECT GREATEST(0, EXTRACT(DAY FROM (CURRENT_TIMESTAMP - a.rueckgabe_frist))::int) AS tage
				FROM ausleihen a
				JOIN buecher_exemplare e ON e.id = a.exemplar_id
				JOIN buecher_titel t ON t.id = e.titel_id
				-- Ohne Dauerleihen: Sie werden nicht überfällig (dieselbe Regel wie in der
				-- Sperr-Automatik und in der Leserliste). Sonst zählt die Übersicht
				-- Mahnfälle, die es nicht gibt — ein Kollege wird nicht gemahnt.
				WHERE a.rueckgabe_am IS NULL AND a.rueckgabe_frist < CURRENT_TIMESTAMP
				  AND a.ist_handapparat = false `+typFilter+`
			)
			SELECT
				COUNT(*)::int,
				COALESCE(MAX(tage), 0)::int,
				COUNT(*) FILTER (WHERE tage <= 14)::int,
				COUNT(*) FILTER (WHERE tage BETWEEN 15 AND 30)::int,
				COUNT(*) FILTER (WHERE tage BETWEEN 31 AND 60)::int,
				COUNT(*) FILTER (WHERE tage > 60)::int
			FROM offen
		`).Scan(&k.Ueberfaellig, &k.MaxTage, &k.Bis14, &k.Bis30, &k.Bis60, &k.Ueber60)
	return k, err
}
