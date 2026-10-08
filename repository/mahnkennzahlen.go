package repository

import "context"

// MahnKennzahlen zählen die offenen, überfälligen Ausleihen: zusammen, die längste Dauer in
// Tagen und die Verteilung nach Dauer. Sie tragen keinen Namen und keinen Titel.
type MahnKennzahlen struct {
	Ueberfaellig, MaxTage        int
	Bis14, Bis30, Bis60, Ueber60 int
}

// LadeMahnKennzahlen liest alle Zahlen in einem Durchgang über die Ausleihen, ohne Verbindung
// zu Lesern oder Titeln: Die Statistik soll kein personenbezogenes Feld erreichen. Dauerleihen
// zählen nicht, sie werden nicht überfällig.
func LadeMahnKennzahlen(ctx context.Context, db DBQueryer) (MahnKennzahlen, error) {
	var k MahnKennzahlen
	err := db.QueryRow(ctx, `
			WITH offen AS (
				SELECT (CURRENT_TIMESTAMP - rueckgabe_frist) AS verzug
				FROM ausleihen
				-- Ohne Dauerleihen: Sie werden nicht überfällig (dieselbe Regel wie in der
				-- Sperr-Automatik und in der Leserliste). Sonst zählt die Übersicht
				-- Mahnfälle, die es nicht gibt — ein Kollege wird nicht gemahnt.
				WHERE rueckgabe_am IS NULL AND rueckgabe_frist < CURRENT_TIMESTAMP
				  AND ist_handapparat = false
			)
			SELECT
				COUNT(*)::int,
				COALESCE(MAX(GREATEST(0, EXTRACT(DAY FROM verzug)::int)), 0)::int,
				COUNT(*) FILTER (WHERE verzug <= INTERVAL '14 days')::int,
				COUNT(*) FILTER (WHERE verzug > INTERVAL '14 days' AND verzug <= INTERVAL '30 days')::int,
				COUNT(*) FILTER (WHERE verzug > INTERVAL '30 days' AND verzug <= INTERVAL '60 days')::int,
				COUNT(*) FILTER (WHERE verzug > INTERVAL '60 days')::int
			FROM offen
		`).Scan(&k.Ueberfaellig, &k.MaxTage, &k.Bis14, &k.Bis30, &k.Bis60, &k.Ueber60)
	return k, err
}
