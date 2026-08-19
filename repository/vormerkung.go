package repository

import (
	"context"
	"errors"
	"time"

	"bibliothek/db"
	"github.com/jackc/pgx/v5"
)

// ErrTitelBereitsAusgeliehen signalisiert, dass ein Schüler einen Titel vormerken will, den
// er aktuell bereits selbst ausgeliehen hat. Ohne diese Sperre könnte er das Buch bei der
// Rückgabe sofort wieder für sich selbst abgreifen und die Vormerkungs-Warteschlange
// dauerhaft für sich monopolisieren. Nutzer-sichtbar (409).
//
//nolint:staticcheck // ST1005: bewusst großgeschrieben, Endnutzer-Meldung
var ErrTitelBereitsAusgeliehen = errors.New("Buch wird aktuell bereits von diesem Schüler ausgeliehen")

// Vormerkung represents a pending book reservation entry for a student.
type Vormerkung struct {
	ID           string    `json:"id"`
	TitelID      string    `json:"titel_id"`
	TitelName    string    `json:"titel"`
	Notiz        string    `json:"notiz,omitempty"`
	ErstelltAm   time.Time `json:"erstellt_am"`
	SchuelerID   string    `json:"schueler_id,omitempty"`
	SchuelerName string    `json:"schueler_name,omitempty"`
}

// VormerkungRepository defines operations for managing individual book reservations.
type VormerkungRepository interface {
	List(ctx context.Context, titelID, schuelerID string) ([]Vormerkung, error)
	Create(ctx context.Context, titelID, notiz, schuelerID string) (string, error)
	Delete(ctx context.Context, id string) error
	// VerfalleAbgelaufeneVormerkungen räumt No-Show-Reservierungen ab.
	VerfalleAbgelaufeneVormerkungen(ctx context.Context) (verfallen, neuBereitgestellt int, err error)
}

// VerfalleAbgelaufeneVormerkungen behandelt „abholbereit"-Reservierungen, deren
// 3-Tage-Abholfrist abgelaufen ist (No-Show). Betreiber-Entscheidung 19.08.2026:
// „Verfall + nächsten bedienen". Ohne diesen Lauf blieb eine solche Vormerkung für
// immer 'abholbereit' — der Schüler fiel still aus der Warteschlange, ohne bedient zu
// werden, und das Exemplar wurde nie wieder zugeteilt.
//
//  1. Die abgelaufene Reservierung wird GELÖSCHT (der No-Show verliert seinen Platz,
//     wie in Bibliotheken üblich) — RETURNING liefert Exemplar und Titel.
//  2. Ist das freigewordene Exemplar noch verfügbar (nicht zwischenzeitlich ausgeliehen
//     oder ausgesondert), wird es dem NÄCHSTEN wartenden, abholberechtigten Schüler
//     zugeteilt (Status 'abholbereit', neue 3-Tage-Frist). Ist es weg, geschieht nichts
//     Weiteres — der reguläre Rückgabe-Pfad bedient die Warteschlange dann später.
//
// Alles in EINER Transaktion; FOR UPDATE SKIP LOCKED verhindert Deadlocks/Doppelzuteilung
// gegen gleichzeitige Rückgaben.
func (r *pgVormerkungRepository) VerfalleAbgelaufeneVormerkungen(ctx context.Context) (int, int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer db.SafeRollback(ctx, tx)

	rows, err := tx.Query(ctx, `
		DELETE FROM vormerkungen
		WHERE status = 'abholbereit' AND bereitgestellt_bis < CURRENT_TIMESTAMP
		RETURNING bereitgestellt_exemplar_id, titel_id`)
	if err != nil {
		return 0, 0, err
	}
	type freigabe struct{ exemplarID, titelID *string }
	var freigaben []freigabe
	for rows.Next() {
		var f freigabe
		if err := rows.Scan(&f.exemplarID, &f.titelID); err != nil {
			rows.Close()
			return 0, 0, err
		}
		freigaben = append(freigaben, f)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}

	neuBereit := 0
	for _, f := range freigaben {
		if f.exemplarID == nil || f.titelID == nil {
			continue
		}
		// Nächsten Wartenden nur dann bedienen, wenn das Exemplar wirklich noch frei ist.
		tag, err := tx.Exec(ctx, `
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
			)`, *f.exemplarID, *f.titelID)
		if err != nil {
			return 0, 0, err
		}
		if tag.RowsAffected() > 0 {
			neuBereit++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}
	return len(freigaben), neuBereit, nil
}

type pgVormerkungRepository struct {
	db db.PgxPoolIface
}

// NewVormerkungRepository returns a new PostgreSQL implementation of VormerkungRepository.
func NewVormerkungRepository(db db.PgxPoolIface) VormerkungRepository {
	return &pgVormerkungRepository{db: db}
}

// List retrieves reservations filtered by either title or student.
func (r *pgVormerkungRepository) List(ctx context.Context, titelID, schuelerID string) ([]Vormerkung, error) {
	var rows pgx.Rows
	var err error

	if titelID != "" {
		rows, err = r.db.Query(ctx, `
			SELECT v.id, v.titel_id, bt.titel, COALESCE(v.notiz, ''), v.erstellt_am,
			       COALESCE(s.id::text, ''), COALESCE(s.vorname || ' ' || s.nachname || ', ' || s.klasse, '')
			FROM vormerkungen v
			JOIN buecher_titel bt ON bt.id = v.titel_id
			LEFT JOIN schueler s ON s.id = v.schueler_id
			WHERE v.titel_id = $1
			ORDER BY v.erstellt_am ASC
		`, titelID)
	} else if schuelerID != "" {
		rows, err = r.db.Query(ctx, `
			SELECT v.id, v.titel_id, bt.titel, COALESCE(v.notiz, ''), v.erstellt_am,
			       COALESCE(s.id::text, ''), COALESCE(s.vorname || ' ' || s.nachname || ', ' || s.klasse, '')
			FROM vormerkungen v
			JOIN buecher_titel bt ON bt.id = v.titel_id
			LEFT JOIN schueler s ON s.id = v.schueler_id
			WHERE v.schueler_id = $1
			ORDER BY v.erstellt_am ASC
		`, schuelerID)
	} else {
		rows, err = r.db.Query(ctx, `
			SELECT v.id, v.titel_id, bt.titel, COALESCE(v.notiz, ''), v.erstellt_am,
			       COALESCE(s.id::text, ''), COALESCE(s.vorname || ' ' || s.nachname || ', ' || s.klasse, '')
			FROM vormerkungen v
			JOIN buecher_titel bt ON bt.id = v.titel_id
			LEFT JOIN schueler s ON s.id = v.schueler_id
			ORDER BY v.erstellt_am ASC
		`)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Vormerkung
	for rows.Next() {
		var v Vormerkung
		if err := rows.Scan(&v.ID, &v.TitelID, &v.TitelName, &v.Notiz, &v.ErstelltAm, &v.SchuelerID, &v.SchuelerName); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// Create creates a new reservation.
//
// Monopolisierungs-Schutz: Ein Schüler, der ein Exemplar dieses Titels aktuell selbst
// ausgeliehen hat, darf ihn nicht zusätzlich für sich vormerken (siehe
// ErrTitelBereitsAusgeliehen). Der Check läuft nur bei gesetztem schuelerID — anonyme
// Vormerkungen (ohne Schüler) sind davon nicht betroffen.
func (r *pgVormerkungRepository) Create(ctx context.Context, titelID, notiz, schuelerID string) (string, error) {
	if schuelerID != "" {
		var bereitsAusgeliehen bool
		if err := r.db.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM ausleihen a
				JOIN buecher_exemplare e ON e.id = a.exemplar_id
				WHERE e.titel_id = $1 AND a.schueler_id = $2 AND a.rueckgabe_am IS NULL
			)
		`, titelID, schuelerID).Scan(&bereitsAusgeliehen); err != nil {
			return "", err
		}
		if bereitsAusgeliehen {
			return "", ErrTitelBereitsAusgeliehen
		}
	}

	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO vormerkungen (titel_id, notiz, schueler_id)
		VALUES ($1, NULLIF($2, ''), NULLIF($3, '')::uuid)
		RETURNING id
	`, titelID, notiz, schuelerID).Scan(&id)
	return id, err
}

// Delete removes a reservation.
func (r *pgVormerkungRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM vormerkungen WHERE id = $1`, id)
	return err
}
