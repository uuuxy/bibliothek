package repository

import (
	"context"
	"errors"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// DamageRepository defines operations for managing book damages and related loan actions.
type DamageRepository interface {
	MarkCopyDefekt(ctx context.Context, copyID string, loanID, schuelerID *string, benutzerID string, betrag float64, beschreibung string) (string, error)
	ReportDamage(ctx context.Context, copyID, loanID, schuelerID string, benutzerID string, beschreibung string, betrag float64) (string, error)
}

type pgDamageRepository struct {
	db db.PgxPoolIface
}

// NewDamageRepository returns a new PostgreSQL implementation of DamageRepository.
func NewDamageRepository(db db.PgxPoolIface) DamageRepository {
	return &pgDamageRepository{db: db}
}

// MarkCopyDefekt marks a book copy as defective and records a damage entry.
func (r *pgDamageRepository) MarkCopyDefekt(ctx context.Context, copyID string, loanID, schuelerID *string, benutzerID string, betrag float64, beschreibung string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer db.SafeRollback(ctx, tx)

	res, err := tx.Exec(ctx, `
		UPDATE buecher_exemplare
		SET ist_ausleihbar = false,
		    zustand_notiz = $1,
		    aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $2
	`, beschreibung, copyID)
	if err != nil {
		return "", err
	}
	if res.RowsAffected() == 0 {
		return "", pgx.ErrNoRows
	}

	var schadensID string
	if schuelerID != nil && *schuelerID != "" {
		err = tx.QueryRow(ctx, `
			INSERT INTO schadensfaelle
			    (exemplar_id, ausleihe_id, schueler_id, beschreibung, betrag)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, copyID, loanID, schuelerID, beschreibung, betrag).Scan(&schadensID)
	} else {
		err = tx.QueryRow(ctx, `
			INSERT INTO schadensfaelle
			    (exemplar_id, ausleihe_id, benutzer_id, beschreibung, betrag)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, copyID, loanID, benutzerID, beschreibung, betrag).Scan(&schadensID)
	}
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return schadensID, nil
}

// ErrExemplarNeuVerliehen signalisiert, dass das zu meldende Exemplar zwischenzeitlich
// (nach dem Öffnen des Schadensformulars) an jemand anderen ausgeliehen wurde. Der Text
// ist nutzer-sichtbar (wird als 409-Meldung ausgeliefert), daher deutsche Großschreibung.
//
//nolint:staticcheck // ST1005: bewusst großgeschrieben, Endnutzer-Meldung
var ErrExemplarNeuVerliehen = errors.New("Exemplar wurde zwischenzeitlich neu ausgeliehen — bitte den Vorgang neu laden")

// ReportDamage sets ist_ausgesondert = true, inserts a damage record, and ends the loan.
func (r *pgDamageRepository) ReportDamage(ctx context.Context, copyID, loanID, schuelerID string, benutzerID string, beschreibung string, betrag float64) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer db.SafeRollback(ctx, tx)

	// Idempotenz + Serialisierung gegen Doppelklick: Zwei parallel abgeschickte
	// "Schaden melden"-Klicks mit derselben ausleihe_id würden sonst beide den
	// fremdeAktive-Check passieren und JE einen Schadensfall anlegen — der Schüler würde
	// für dasselbe Buch doppelt belastet. Wir sperren zuerst die Ausleihe-Zeile
	// (FOR UPDATE): der zweite Aufruf blockiert, bis der erste committet hat, und liest
	// danach den bereits angelegten Schadensfall. Existiert für diese Ausleihe schon ein
	// (nicht stornierter) Schadensfall, geben wir dessen ID idempotent zurück, statt einen
	// zweiten anzulegen.
	var vorhanden bool
	if err := tx.QueryRow(ctx,
		`SELECT true FROM ausleihen WHERE id = $1 FOR UPDATE`, loanID,
	).Scan(&vorhanden); err != nil {
		return "", err // pgx.ErrNoRows: Ausleihe existiert nicht
	}

	var bestehenderSchaden string
	err = tx.QueryRow(ctx,
		`SELECT id FROM schadensfaelle WHERE ausleihe_id = $1 AND storniert_am IS NULL LIMIT 1`,
		loanID,
	).Scan(&bestehenderSchaden)
	if err == nil {
		// Schadensfall existiert bereits — idempotent zurückgeben, nichts doppelt buchen.
		return bestehenderSchaden, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	// Race-Schutz: Bleibt das Schadensformular offen, während das Buch zurückgegeben
	// und neu ausgeliehen wird, würde der "Melden"-Klick ein aktiv verliehenes Exemplar
	// aussondern. Gibt es für dieses Exemplar eine aktive Ausleihe, die NICHT die hier
	// gemeldete ist, brechen wir ab, statt die neue Ausleihe blind zu überschreiben.
	var fremdeAktive int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM ausleihen
		WHERE exemplar_id = $1 AND rueckgabe_am IS NULL AND id <> $2
	`, copyID, loanID).Scan(&fremdeAktive); err != nil {
		return "", err
	}
	if fremdeAktive > 0 {
		return "", ErrExemplarNeuVerliehen
	}

	_, err = tx.Exec(ctx, `
		UPDATE buecher_exemplare
		SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'BESCHAEDIGUNG',
		    zustand_notiz = $1, aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $2
	`, beschreibung, copyID)
	if err != nil {
		return "", err
	}

	var schadensID string
	err = tx.QueryRow(ctx, `
		INSERT INTO schadensfaelle (exemplar_id, ausleihe_id, schueler_id, beschreibung, betrag)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, copyID, loanID, schuelerID, beschreibung, betrag).Scan(&schadensID)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx, `
		UPDATE ausleihen
		SET rueckgabe_am = CURRENT_TIMESTAMP, rueckgabe_bearbeiter_id = $1
		WHERE id = $2
	`, benutzerID, loanID)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return schadensID, nil
}
