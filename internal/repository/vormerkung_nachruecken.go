package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// sqlWartenderDarfAbholen ist die Bedingung an den Schüler einer wartenden Vormerkung (Alias
// s), damit ihm ein Exemplar bereitgestellt wird: nicht gelöscht und nicht gesperrt. Sonst läge
// das Buch für jemanden im Abholfach, der es nicht holen kann, und die Nächsten gingen leer aus.
const sqlWartenderDarfAbholen = `s.deleted_at IS NULL AND s.ist_gesperrt = false AND COALESCE(s.is_manually_blocked, false) = false`

// bedieneNaechstenWartenden teilt ein frei gewordenes Exemplar dem nächsten wartenden Schüler
// desselben Titels zu, der abholen darf (neue Abholfrist ab jetzt, BestimmeAbholfrist), aber
// nur, wenn das Exemplar noch frei ist: nicht ausgeliehen, gesperrt oder ausgesondert.
// Verschwindet eine Vormerkung, für die schon ein Exemplar bereitlag, bliebe der Nächste sonst
// für immer wartend, während das Buch im Regal steht.
//
// Gesperrt wird nur die Vormerkung (FOR UPDATE OF v). Eine Vormerkung, die eine Rückgabe gerade
// bedient, wird übergangen; hält eine Ausleihe die Zeile des Schülers, bleibt er trotzdem an
// der Reihe. Liefert true, wenn jemand bedient wurde.
func bedieneNaechstenWartenden(ctx context.Context, db SpurenDatenbank, exemplarID, titelID string, jetzt time.Time) (bool, error) {
	frist, err := BestimmeAbholfrist(ctx, db, jetzt)
	if err != nil {
		return false, err
	}
	tag, err := db.Exec(ctx, `
		UPDATE vormerkungen
		SET status = 'abholbereit', bereitgestellt_exemplar_id = $1,
		    bereitgestellt_bis = $3
		WHERE id = (
			SELECT v.id FROM vormerkungen v JOIN schueler s ON v.schueler_id = s.id
			WHERE v.titel_id = $2 AND v.status = 'wartend'
			  AND `+sqlWartenderDarfAbholen+`
			ORDER BY v.erstellt_am ASC LIMIT 1
			FOR UPDATE OF v SKIP LOCKED
		)
		AND EXISTS (
			SELECT 1 FROM buecher_exemplare e
			WHERE e.id = $1 AND e.ist_ausleihbar = true AND e.ist_ausgesondert = false
			  AND NOT EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = $1 AND a.rueckgabe_am IS NULL)
		)`, exemplarID, titelID, frist)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// loescheVormerkungenUndRueckeNach ist der Schritt der Vormerkungen in der Tilgung der Spuren:
// Es löscht alle Vormerkungen der genannten Schüler (die Notiz ist personenbezogen) und lässt
// für jedes dabei frei werdende Exemplar die Warteschlange nachrücken. Geliefert wird die Zahl
// der gelöschten Zeilen; der nächtliche Lauf weist sie als bereinigt aus.
//
// Erst löschen, dann bedienen: So ist der nächste Wartende nie ein Kind, dessen Vormerkung
// gerade fällt (der Lauf für Abgänger löscht Schüler, die nicht im Papierkorb liegen; sie kämen
// als wartend sonst selbst infrage).
//
// Die Abholfrist des Nachrückenden zählt ab jetzt: Die Tilgung ist ein Schritt der Liste
// spurTilgungen und hat keine eigene Uhr.
func loescheVormerkungenUndRueckeNach(ctx context.Context, db SpurenDatenbank, schuelerIDs []string) (int64, error) {
	rows, err := db.Query(ctx, `
		DELETE FROM vormerkungen WHERE schueler_id = ANY($1::uuid[])
		RETURNING bereitgestellt_exemplar_id, titel_id`, schuelerIDs)
	if err != nil {
		return 0, err
	}
	freigaben, err := sammleFreigaben(rows)
	if err != nil {
		return 0, err
	}
	// Erst nach dem Schließen von rows weiterschreiben; solange es offen ist, ist die
	// Verbindung belegt.
	return int64(len(freigaben)), rueckeNach(ctx, db, freigaben, time.Now())
}

// rueckeNach teilt jedes frei gewordene Exemplar dem Nächsten in der Warteschlange seines Titels
// zu (bedieneNaechstenWartenden). Eine Freigabe ohne Exemplar hatte nichts im Abholfach.
func rueckeNach(ctx context.Context, db SpurenDatenbank, freigaben []freigewordenesExemplar, jetzt time.Time) error {
	for _, f := range freigaben {
		if f.exemplarID == nil || f.titelID == nil {
			continue
		}
		if _, err := bedieneNaechstenWartenden(ctx, db, *f.exemplarID, *f.titelID, jetzt); err != nil {
			return fmt.Errorf("nächsten wartenden bedienen: %w", err)
		}
	}
	return nil
}

// loescheDoppelteVormerkungen sortiert beim Zusammenführen die doppelten Vormerkungen aus: sql
// ist ein DELETE … RETURNING mit Kennung, dem Exemplar im Abholfach und dem Titel. Geliefert
// werden die Kennungen für den Eintrag im Protokoll und die Freigaben; für sie rückt der Nächste
// nach (rueckeNach), sobald alle Vormerkungen umgehängt sind und jeder Wartende in der Schlange
// steht.
func loescheDoppelteVormerkungen(ctx context.Context, tx pgx.Tx, sql string, args ...any) ([]string, []freigewordenesExemplar, error) {
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	ids := []string{}
	var freigaben []freigewordenesExemplar
	for rows.Next() {
		var id string
		var f freigewordenesExemplar
		if err := rows.Scan(&id, &f.exemplarID, &f.titelID); err != nil {
			return nil, nil, err
		}
		ids = append(ids, id)
		freigaben = append(freigaben, f)
	}
	return ids, freigaben, rows.Err()
}

// freigewordenesExemplar ist eine Vormerkung, die gerade verschwunden ist, samt dem
// Exemplar, das sie bereitliegen hatte. Ist exemplarID nil, hing kein Buch daran.
type freigewordenesExemplar struct {
	exemplarID *string
	titelID    *string
}

// sammleFreigaben liest das RETURNING einer Löschung von Vormerkungen ganz aus und schließt
// rows, damit die Verbindung danach wieder schreiben kann.
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
