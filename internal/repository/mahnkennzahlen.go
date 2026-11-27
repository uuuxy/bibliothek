package repository

import "context"

// MahnKennzahlen zählen die offenen, überfälligen Ausleihen: zusammen, die längste Dauer in
// Tagen und die Verteilung nach Dauer. Sie tragen keinen Namen und keinen Titel.
type MahnKennzahlen struct {
	Ueberfaellig, MaxTage        int
	Bis14, Bis30, Bis60, Ueber60 int
	// Laufend zählt die offenen Ausleihen derselben Grundmenge, überfällig oder nicht: der Nenner
	// der Quote „überfällig ÷ laufende Ausleihen“.
	Laufend int
}

// LadeMahnKennzahlen liest alle Zahlen in einem Durchgang über die Ausleihen, ohne Verbindung
// zu Lesern: Die Statistik soll kein personenbezogenes Feld erreichen. Der Titel dient nur dem
// Bestandsfilter; die Antwort trägt nur Zahlen. Dauerleihen zählen nicht, sie werden nicht
// überfällig.
//
// typFilter ist das Stück aus BestandsFilterBedingung, mit dem die Statistik auch ihre übrigen
// Kacheln filtert. Zähler und Nenner der Quote „überfällig ÷ laufende Ausleihen“ kommen beide aus
// dieser Abfrage, aus derselben Grundmenge: nur Bücher (überfällige Geräte stehen im Mahnwesen),
// ohne Dauerleihen, mit demselben Filter. „Aktuell verliehen“ der Statistik zählt Dauerleihen mit
// und machte die Quote zu klein.
//
// Die Dauer zählt in vollen Tagen wie im Mahnwesen (tage_ueberfaellig, ab 15 Tagen „Mahnung“):
// Ein Verzug von 14 Tagen und ein paar Stunden steht in beiden unter „bis 14 Tage“.
func LadeMahnKennzahlen(ctx context.Context, db Datenbank, typFilter string) (MahnKennzahlen, error) {
	var k MahnKennzahlen
	err := db.QueryRow(ctx, `
			WITH laufend AS (
				SELECT a.rueckgabe_frist < CURRENT_TIMESTAMP AS ueberfaellig,
				       GREATEST(0, EXTRACT(DAY FROM (CURRENT_TIMESTAMP - a.rueckgabe_frist))::int) AS tage
				FROM ausleihen a
				JOIN buecher_exemplare e ON e.id = a.exemplar_id
				JOIN buecher_titel t ON t.id = e.titel_id
				-- Ohne Dauerleihen: Sie werden nicht überfällig (dieselbe Regel wie in der
				-- Sperr-Automatik und in der Leserliste). Sonst zählt die Übersicht
				-- Mahnfälle, die es nicht gibt — ein Kollege wird nicht gemahnt.
				WHERE a.rueckgabe_am IS NULL
				  AND a.ist_handapparat = false `+typFilter+`
			)
			SELECT
				COUNT(*) FILTER (WHERE ueberfaellig)::int,
				COALESCE(MAX(tage) FILTER (WHERE ueberfaellig), 0)::int,
				COUNT(*) FILTER (WHERE ueberfaellig AND tage <= 14)::int,
				COUNT(*) FILTER (WHERE ueberfaellig AND tage BETWEEN 15 AND 30)::int,
				COUNT(*) FILTER (WHERE ueberfaellig AND tage BETWEEN 31 AND 60)::int,
				COUNT(*) FILTER (WHERE ueberfaellig AND tage > 60)::int,
				COUNT(*)::int
			FROM laufend
		`).Scan(&k.Ueberfaellig, &k.MaxTage, &k.Bis14, &k.Bis30, &k.Bis60, &k.Ueber60, &k.Laufend)
	return k, err
}
