package repository

// Die Abfragen der Tresen-Auskunft (api/system/audit_tresen_auskunft.go): der eine Leseweg in
// die Details des Protokolls, an einen Zweck gebunden. Es sind Funktionen über Datenbank und
// keine Methoden an ProtokollAbfragen: Die Schnittstelle trägt die Schreibwege, und für das
// Lesen müsste sonst jede Attrappe wachsen.

import (
	"context"
	"time"
)

// TresenExemplarZeile ist ein Exemplar, das ein Barcode heute oder früher bezeichnete. Status
// ist „im_bestand", „ausgesondert" oder „geloescht"; im letzten Fall gibt es die Zeile in
// buecher_exemplare nicht mehr, nur ihre Spur im Protokoll.
type TresenExemplarZeile struct {
	ExemplarID string
	Titel      string
	Status     string
}

// TresenEreignisZeile ist ein Eintrag zu Ausleihe oder Rückgabe mit den Namen der Beteiligten.
// Ein leerer Name heißt: Der Personenbezug ist getilgt oder die Person gelöscht; wie das zu
// lesen ist, entscheidet die Tür.
//
// LeserName und LeserArt nennen den Entleiher aus details->>'schueler_id', Schüler wie
// Kollegium. KontoName füllen nur alte Einträge, die einen Kollegen als Konto unter benutzer_id
// trugen; heute schreibt das kein Rufer mehr.
type TresenEreignisZeile struct {
	Zeitpunkt      time.Time
	Aktion         string
	LeserName      string
	LeserKlasse    string
	LeserArt       string // Art des Lesers (chk_leser_art); leer ohne Treffer
	KontoName      string
	BearbeiterName string
}

// FindeTresenExemplare nennt die Exemplare zu einem Barcode: zuerst den Bestand (auch ein
// ausgesondertes Exemplar trägt seinen Barcode noch), dann aus dem Protokoll die Exemplare,
// deren Zeile es nicht mehr gibt; der jüngste Eintrag nennt den Titel. Ein Barcode kann
// mehrfach treffen: Nach dem endgültigen Löschen ist die Nummer frei und kann an einem neuen
// Exemplar kleben.
func FindeTresenExemplare(ctx context.Context, db Datenbank, barcode string) ([]TresenExemplarZeile, error) {
	const q = `
		SELECT e.id::text, t.titel,
		       CASE WHEN e.ist_ausgesondert THEN 'ausgesondert' ELSE 'im_bestand' END
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		WHERE e.barcode_id = $1
		UNION ALL
		SELECT * FROM (
			SELECT DISTINCT ON (l.datensatz_id)
			       l.datensatz_id::text, COALESCE(l.details->>'titel', ''), 'geloescht'
			FROM audit_log l
			WHERE l.tabelle = 'buecher_exemplare'
			  AND l.details->>'barcode_id' = $1
			  AND NOT EXISTS (SELECT 1 FROM buecher_exemplare e2 WHERE e2.id = l.datensatz_id)
			ORDER BY l.datensatz_id, l.timestamp DESC
		) geloeschte
		LIMIT 20`
	rows, err := db.Query(ctx, q, barcode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	zeilen := []TresenExemplarZeile{}
	for rows.Next() {
		var z TresenExemplarZeile
		if err := rows.Scan(&z.ExemplarID, &z.Titel, &z.Status); err != nil {
			return nil, err
		}
		zeilen = append(zeilen, z)
	}
	return zeilen, rows.Err()
}

// FindeTresenEreignisse liest die Einträge zu Ausleihe und Rückgabe der Exemplare und löst die
// Personen zu Namen auf. LEFT JOIN, damit ein getilgter Bezug oder eine gelöschte Person den
// Vorgang nicht verschluckt; limit kappt die Liste.
func FindeTresenEreignisse(ctx context.Context, db Datenbank, exemplarIDs []string, limit int) ([]TresenEreignisZeile, error) {
	const q = `
		SELECT l.timestamp, l.aktion,
		       COALESCE(TRIM(sch.vorname || ' ' || sch.nachname), ''),
		       COALESCE(sch.klasse, ''),
		       COALESCE(sch.art, ''),
		       COALESCE(TRIM(lk.vorname || ' ' || lk.nachname), ''),
		       COALESCE(TRIM(bb.vorname || ' ' || bb.nachname), '')
		FROM audit_log l
		-- Die Tabelle leser, nicht die Sicht schueler: Unter schueler_id steht seit
		-- Migration 125 jeder Entleiher, auch ein Kollege. Über die Sicht hieß der
		-- Kollege „Personenbezug getilgt" (TestTresenAuskunftNenntDenKollegen).
		LEFT JOIN leser sch ON sch.id::text = l.details->>'schueler_id'
		LEFT JOIN benutzer lk ON lk.id::text = l.details->>'benutzer_id'
		LEFT JOIN benutzer bb ON bb.id = l.bearbeiter_id
		WHERE l.tabelle = 'ausleihen'
		  AND l.aktion IN ('CHECKOUT', 'RETURN')
		  AND l.datensatz_id = ANY($1::uuid[])
		ORDER BY l.timestamp DESC
		LIMIT $2`
	rows, err := db.Query(ctx, q, exemplarIDs, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	zeilen := []TresenEreignisZeile{}
	for rows.Next() {
		var z TresenEreignisZeile
		if err := rows.Scan(&z.Zeitpunkt, &z.Aktion,
			&z.LeserName, &z.LeserKlasse, &z.LeserArt, &z.KontoName, &z.BearbeiterName); err != nil {
			return nil, err
		}
		zeilen = append(zeilen, z)
	}
	return zeilen, rows.Err()
}
