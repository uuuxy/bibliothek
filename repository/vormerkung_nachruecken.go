package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Nachrücken in der Warteschlange: Verschwindet eine Vormerkung, die bereits ein
// Exemplar bereitliegen hatte, muss dieses Exemplar dem NÄCHSTEN Wartenden zugeteilt
// werden. Sonst bleibt der Nächste für immer 'wartend', während das Buch verfügbar
// herumsteht — genau der Zustand, für den der Verfall-Lauf gebaut wurde
// (VerfalleAbgelaufeneVormerkungen).
//
// Dieselbe Bewegung braucht die DSGVO-Tilgung: Purge, Abgänger-Cronjob und
// LUSD-Anonymisierung löschen die Vormerkungen eines Schülers (die Freitext-Notiz ist
// personenbezogen) — und ließen dabei bis 06.09.2026 ein bereitgelegtes Exemplar
// unbedient auf dem Abholregal liegen. Gefunden über Frage 12 (Gegenrichtung Schema)
// beim Befragen von `vormerkungen.schueler_id -> schueler ON DELETE CASCADE`: Der
// CASCADE ist dort gar nicht der Handelnde — die Spuren-Tilgung löscht selbst, und ihr
// fehlte der Schritt.

// bedieneNaechstenWartenden teilt ein freigewordenes Exemplar dem nächsten wartenden,
// abholberechtigten Schüler desselben Titels zu (neue 3-Tage-Frist) — aber nur, wenn
// das Exemplar wirklich noch frei ist (nicht zwischenzeitlich ausgeliehen, gesperrt
// oder ausgesondert). FOR UPDATE SKIP LOCKED verhindert Doppelzuteilung gegen
// gleichzeitige Rückgaben. Liefert true, wenn jemand bedient wurde.
func bedieneNaechstenWartenden(ctx context.Context, ex SpurenExecutor, exemplarID, titelID string) (bool, error) {
	tag, err := ex.Exec(ctx, `
		UPDATE vormerkungen
		SET status = 'abholbereit', bereitgestellt_exemplar_id = $1,
		    bereitgestellt_bis = CURRENT_TIMESTAMP + INTERVAL '3 days'
		WHERE id = (
			SELECT v.id FROM vormerkungen v JOIN schueler s ON v.schueler_id = s.id
			WHERE v.titel_id = $2 AND v.status = 'wartend'
			  AND s.deleted_at IS NULL AND s.ist_gesperrt = false
			  AND COALESCE(s.is_manually_blocked, false) = false
			ORDER BY v.erstellt_am ASC LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		AND EXISTS (
			SELECT 1 FROM buecher_exemplare e
			WHERE e.id = $1 AND e.ist_ausleihbar = true AND e.ist_ausgesondert = false
			  AND NOT EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = $1 AND a.rueckgabe_am IS NULL)
		)`, exemplarID, titelID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// loescheVormerkungenUndRuecktNach ist der Vormerkungs-Schritt der Spuren-Tilgung: Es
// löscht ALLE Vormerkungen der gegebenen Schüler und lässt für jedes dabei frei
// werdende Exemplar die Warteschlange nachrücken. Gemeldet wird die Zahl der GELÖSCHTEN
// Zeilen — das ist die Zahl, die das Cron-Protokoll als „bereinigt" ausweist.
//
// Warum erst löschen, dann bedienen: Nur so kann der nächste Wartende nicht wieder ein
// Kind sein, dessen Vormerkung gerade fällt (der Abgänger-Cronjob löscht Schüler, die
// nicht weichgelöscht sind — sie kämen als „wartend" sonst selbst infrage).
func loescheVormerkungenUndRuecktNach(ctx context.Context, ex SpurenExecutor, schuelerIDs []string) (int64, error) {
	rows, err := ex.Query(ctx, `
		DELETE FROM vormerkungen WHERE schueler_id = ANY($1::uuid[])
		RETURNING bereitgestellt_exemplar_id, titel_id`, schuelerIDs)
	if err != nil {
		return 0, err
	}
	freigaben, err := sammleFreigaben(rows)
	if err != nil {
		return 0, err
	}
	// Erst nach dem Schließen der Rows weiterschreiben — solange sie offen sind, ist
	// die Verbindung belegt.
	for _, f := range freigaben {
		if f.exemplarID == nil || f.titelID == nil {
			continue
		}
		if _, err := bedieneNaechstenWartenden(ctx, ex, *f.exemplarID, *f.titelID); err != nil {
			return int64(len(freigaben)), fmt.Errorf("nächsten wartenden bedienen: %w", err)
		}
	}
	return int64(len(freigaben)), nil
}

// freigewordenesExemplar ist eine Vormerkung, die gerade verschwunden ist, samt dem
// Exemplar, das sie bereitliegen hatte. Ist exemplarID nil, hing kein Buch daran.
type freigewordenesExemplar struct {
	exemplarID *string
	titelID    *string
}

// sammleFreigaben liest das RETURNING einer Vormerkungs-Löschung vollständig aus und
// schließt die Rows — damit die Verbindung danach wieder schreiben kann.
func sammleFreigaben(rows pgx.Rows) ([]freigewordenesExemplar, error) {
	defer rows.Close()
	var freigaben []freigewordenesExemplar
	for rows.Next() {
		var f freigewordenesExemplar
		if err := rows.Scan(&f.exemplarID, &f.titelID); err != nil {
			return nil, err
		}
		freigaben = append(freigaben, f)
	}
	return freigaben, rows.Err()
}
