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

// BestellungZuTokenHash liefert die Bestellung, zu der der Hash eines gültigen
// Bestätigungs-Links gehört. Unbekannt und abgelaufen sind derselbe Fall: pgx.ErrNoRows.
func BestellungZuTokenHash(ctx context.Context, db DBQueryer, tokenHash string) (string, error) {
	var id string
	err := db.QueryRow(ctx, `
		SELECT id FROM bestellungen_verlauf
		WHERE bestaetigungs_token_hash = $1
		  AND (token_gueltig_bis IS NULL OR token_gueltig_bis > now())
	`, tokenHash).Scan(&id)
	return id, err
}

// OeffentlicherBestellkopf sind die Angaben einer Bestellung für die Seite hinter dem
// Bestätigungs-Link: ohne Preise. EtikettenVorhanden sagt, ob eine Position mit Vorab-Barcode
// bestellt wurde; Mittel ist leer bei einer Alt-Bestellung ohne Zuordnung.
type OeffentlicherBestellkopf struct {
	LieferantName      string
	Kundennummer       string
	Bestelldatum       time.Time
	AnzahlExemplare    int
	BestaetigtAm       *time.Time
	LinkGueltigBis     *time.Time
	EtikettenVorhanden bool
	Mittel             string
}

// LadeOeffentlichenBestellkopf liest den Kopf einer Bestellung für die Seite hinter dem Link;
// pgx.ErrNoRows, wenn es die Bestellung nicht gibt.
func LadeOeffentlichenBestellkopf(ctx context.Context, db DBQueryer, bestellungID string) (OeffentlicherBestellkopf, error) {
	var k OeffentlicherBestellkopf
	err := db.QueryRow(ctx, `
		SELECT b.lieferant_name, b.kundennummer, b.bestelldatum, b.anzahl_exemplare, b.bestaetigt_am,
		       b.token_gueltig_bis,
		       EXISTS (SELECT 1 FROM bestellungen_positionen p
		                WHERE p.bestellung_id = b.id AND p.mit_vorab_barcode),
		       COALESCE(b.mittel, '')
		FROM bestellungen_verlauf b WHERE b.id = $1
	`, bestellungID).Scan(&k.LieferantName, &k.Kundennummer, &k.Bestelldatum, &k.AnzahlExemplare,
		&k.BestaetigtAm, &k.LinkGueltigBis, &k.EtikettenVorhanden, &k.Mittel)
	return k, err
}

// OeffentlichePosition ist eine Bestellzeile ohne Preis.
type OeffentlichePosition struct {
	TitelName string
	ISBN      string
	Menge     int
}

// ListeOeffentlichePositionen liefert die Positionen einer Bestellung nach Titel geordnet.
func ListeOeffentlichePositionen(ctx context.Context, db DBQueryer, bestellungID string) ([]OeffentlichePosition, error) {
	rows, err := db.Query(ctx, `
		SELECT titel_name, isbn, menge FROM bestellungen_positionen
		WHERE bestellung_id = $1 ORDER BY titel_name
	`, bestellungID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positionen := []OeffentlichePosition{}
	for rows.Next() {
		var p OeffentlichePosition
		if err := rows.Scan(&p.TitelName, &p.ISBN, &p.Menge); err != nil {
			return nil, err
		}
		positionen = append(positionen, p)
	}
	return positionen, rows.Err()
}
