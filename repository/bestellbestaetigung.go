package repository

import (
	"context"
	"time"
)

// BestellungImBestaetigungsweg sagt, ob die Bestellung über den Bestelllink bestätigt wird.
// Zwei Wege zählen: Die Bestellung ist mit einem Link hinausgegangen (Token vorhanden); daran
// ändert ein späterer Wechsel des Hauptlieferanten nichts, sie wartet weiter. Oder ihr
// Lieferant ist heute Hauptlieferant und sie hat noch keinen Link, weil beim Bestellen keine
// öffentliche Adresse hinterlegt war.
//
// COALESCE, weil die Bestellung ihren gelöschten Lieferanten als Beleg überlebt
// (lieferant_id NULL): Ohne es scheiterte das Lesen des Wahrheitswerts.
func BestellungImBestaetigungsweg(ctx context.Context, db DBQueryer, id string) (bool, error) {
	var ok bool
	err := db.QueryRow(ctx, `
		SELECT b.bestaetigungs_token_hash IS NOT NULL OR coalesce(l.ist_hauptlieferant, false)
		FROM bestellungen_verlauf b
		LEFT JOIN lieferanten l ON l.id = b.lieferant_id
		WHERE b.id = $1
	`, id).Scan(&ok)
	return ok, err
}

// BestaetigeBestellung trägt die Bestätigung ein; bereits=true heißt, sie lag schon vor.
//
// Die Bedingung bestaetigt_am IS NULL macht das in einer Anweisung: Bestätigen Lieferant und
// Bibliothek zugleich, gewinnt genau einer, und der andere überschreibt nichts. Eine Prüfung
// davor ließe zwischen Lesen und Schreiben ein Fenster. Bestellungen werden nie gelöscht;
// null getroffene Zeilen nach der Prüfung des Aufrufers heißen deshalb immer: schon bestätigt.
// Eine leere Etikettengröße bleibt NULL, wie die Bedingung der Spalte es verlangt.
func BestaetigeBestellung(ctx context.Context, db DBQueryer, bestellungID, groesse, format, durch string) (bereits bool, err error) {
	tag, err := db.Exec(ctx, `
		UPDATE bestellungen_verlauf
		SET bestaetigt_am = now(), etiketten_groesse = NULLIF($1, ''),
		    etiketten_format = NULLIF($2, ''), bestaetigt_durch = $3
		WHERE id = $4 AND bestaetigt_am IS NULL
	`, groesse, format, durch, bestellungID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 0, nil
}

// ErneuereBestaetigungsToken speichert den Hash eines neuen Bestätigungs-Links mit neuer Frist
// und nennt ihr Ende. Gespeichert ist immer nur der Hash des einen gültigen Links; ein
// früherer ist damit ungültig.
func ErneuereBestaetigungsToken(ctx context.Context, db DBQueryer, bestellungID, hash string, tage int) (time.Time, error) {
	var gueltigBis time.Time
	err := db.QueryRow(ctx, `
		UPDATE bestellungen_verlauf
		SET bestaetigungs_token_hash = $1, token_gueltig_bis = now() + make_interval(days => $2)
		WHERE id = $3
		RETURNING token_gueltig_bis
	`, hash, tage, bestellungID).Scan(&gueltigBis)
	return gueltigBis, err
}
