package repository

// audit_tresen.go — die Abfragen der Tresen-Auskunft (api/audit_tresen_auskunft.go):
// der eine zweckgebundene Leseweg in audit_log.details (Betreiber-Entscheidung
// 01.09.2026, Befund-Register). Paketfunktionen über DBQueryer statt Methoden auf
// AuditRepository — dieselbe Bauart wie TilgeSchuelerSpuren: Das Interface trägt
// die Schreibpfade, dieser Leseweg braucht keinen Mock-Zwang für alle Fakes.

import (
	"context"
	"time"
)

// TresenExemplarZeile ist ein Exemplar, das ein Barcode heute oder früher
// bezeichnete. Status: "im_bestand", "ausgesondert" oder "geloescht" (die Zeile in
// buecher_exemplare existiert nicht mehr, nur noch der Audit-Snapshot).
type TresenExemplarZeile struct {
	ExemplarID string
	Titel      string
	Status     string
}

// TresenEreignisZeile ist ein CHECKOUT/RETURN-Protokolleintrag mit aufgelösten
// Klarnamen. Leere Namen heißen: Personenbezug getilgt oder Person gelöscht —
// die Deutung trifft der Handler, nicht die Abfrage.
type TresenEreignisZeile struct {
	Zeitpunkt      time.Time
	Aktion         string
	SchuelerName   string
	SchuelerKlasse string
	LehrkraftName  string
	BearbeiterName string
}

// SucheTresenExemplare löst einen Barcode zu Exemplaren auf: erst der Bestand
// (auch ausgesonderte Zeilen tragen ihren Barcode noch), dann die Snapshot-Details
// des Protokolls für Exemplare, deren Zeile es nicht mehr gibt (DISTINCT ON: der
// jüngste Snapshot benennt den Titel). Ein Barcode kann mehrfach treffen — nach
// endgültigem Löschen wird die Nummer frei und kann an einem neuen Exemplar kleben.
func SucheTresenExemplare(ctx context.Context, db DBQueryer, barcode string) ([]TresenExemplarZeile, error) {
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

// SucheTresenEreignisse liest die CHECKOUT/RETURN-Protokolle der Exemplare und löst
// die Personen zu Klarnamen auf. LEFT JOIN, nicht JOIN: Ein getilgter Bezug oder
// eine gelöschte Person darf den Vorgang nicht verschlucken. limit kappt die Liste
// (unbegrenzte Listen-Endpunkte sind eine bekannte Bugklasse dieses Projekts).
func SucheTresenEreignisse(ctx context.Context, db DBQueryer, exemplarIDs []string, limit int) ([]TresenEreignisZeile, error) {
	const q = `
		SELECT l.timestamp, l.aktion,
		       COALESCE(TRIM(sch.vorname || ' ' || sch.nachname), ''),
		       COALESCE(sch.klasse, ''),
		       COALESCE(TRIM(lk.vorname || ' ' || lk.nachname), ''),
		       COALESCE(TRIM(bb.vorname || ' ' || bb.nachname), '')
		FROM audit_log l
		LEFT JOIN schueler sch ON sch.id::text = l.details->>'schueler_id'
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
			&z.SchuelerName, &z.SchuelerKlasse, &z.LehrkraftName, &z.BearbeiterName); err != nil {
			return nil, err
		}
		zeilen = append(zeilen, z)
	}
	return zeilen, rows.Err()
}
