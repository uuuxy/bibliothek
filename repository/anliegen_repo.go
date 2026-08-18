package repository

import (
	"context"
	"errors"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// Anliegen ist ein Wunsch oder eine Meldung einer Lehrkraft (Migration 075):
// "Ich möchte in der 8G3 den Markl 2" bzw. "die 8G3 hat die falschen Bücher".
// EIN Mechanismus für beide Fälle — die LMF arbeitet die Liste in Ruhe ab,
// beim Abhaken geht eine Mail an die Lehrkraft.
type Anliegen struct {
	ID         string     `json:"id"`
	Art        string     `json:"art"` // 'wunsch' | 'meldung'
	TitelText  string     `json:"titel_text"`
	TitelID    *string    `json:"titel_id,omitempty"`
	ISBN       string     `json:"isbn,omitempty"`
	Klasse     string     `json:"klasse"`
	Kommentar  string     `json:"kommentar,omitempty"`
	ErstelltAm time.Time  `json:"erstellt_am"`
	ErledigtAm *time.Time `json:"erledigt_am,omitempty"`
	// ErledigtNotiz ist der Einzeiler der LMF beim Abhaken ("bestellt",
	// "getauscht am 3.9.") — er steht auch in der Mail an die Lehrkraft.
	ErledigtNotiz string `json:"erledigt_notiz,omitempty"`
	// Von: "Vorname Nachname" der Lehrkraft; leer, wenn das Konto gelöscht wurde.
	Von string `json:"von,omitempty"`
}

// AnliegenErledigt trägt die Daten für die Abhak-Mail.
type AnliegenErledigt struct {
	Art            string
	TitelText      string
	Klasse         string
	ErledigtNotiz  string
	AnfragendeMail *string
}

// AnliegenRepository verwaltet die Wünsche und Meldungen der Lehrkräfte.
type AnliegenRepository struct {
	pool db.PgxPoolIface
}

// NewAnliegenRepository erzeugt das Repository auf dem übergebenen Pool.
func NewAnliegenRepository(pool db.PgxPoolIface) *AnliegenRepository {
	return &AnliegenRepository{pool: pool}
}

// Create legt ein Anliegen an. angefordertVon ist die Benutzer-ID der Lehrkraft.
func (r *AnliegenRepository) Create(ctx context.Context, art, titelText, titelID, isbn, klasse, kommentar, angefordertVon string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO lehrer_anliegen (art, titel_text, titel_id, isbn, klasse, kommentar, angefordert_von)
		VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, $6, $7)
		RETURNING id`,
		art, titelText, titelID, isbn, klasse, kommentar, angefordertVon).Scan(&id)
	return id, err
}

// ListOffene liefert alle offenen Anliegen für die LMF-Liste, älteste zuerst
// (wie die Klassensatz-Warteschlange: wer zuerst fragt, ist zuerst dran).
func (r *AnliegenRepository) ListOffene(ctx context.Context) ([]Anliegen, error) {
	return r.list(ctx, `
		SELECT a.id, a.art, a.titel_text, a.titel_id::text, COALESCE(a.isbn, ''), a.klasse,
		       a.kommentar, a.erstellt_am, a.erledigt_am, a.erledigt_notiz,
		       COALESCE(btrim(b.vorname || ' ' || b.nachname), '')
		FROM lehrer_anliegen a
		LEFT JOIN benutzer b ON a.angefordert_von = b.id
		WHERE a.erledigt_am IS NULL
		ORDER BY a.erstellt_am ASC`)
}

// ListEigene liefert die Anliegen EINER Lehrkraft (Portal-Status), neueste
// zuerst und auf die letzten 50 begrenzt — das Portal ist kein Archiv.
func (r *AnliegenRepository) ListEigene(ctx context.Context, benutzerID string) ([]Anliegen, error) {
	return r.list(ctx, `
		SELECT a.id, a.art, a.titel_text, a.titel_id::text, COALESCE(a.isbn, ''), a.klasse,
		       a.kommentar, a.erstellt_am, a.erledigt_am, a.erledigt_notiz, ''
		FROM lehrer_anliegen a
		WHERE a.angefordert_von = $1
		ORDER BY a.erstellt_am DESC
		LIMIT 50`, benutzerID)
}

func (r *AnliegenRepository) list(ctx context.Context, query string, args ...any) ([]Anliegen, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	anliegen := []Anliegen{}
	for rows.Next() {
		var a Anliegen
		if err := rows.Scan(&a.ID, &a.Art, &a.TitelText, &a.TitelID, &a.ISBN, &a.Klasse,
			&a.Kommentar, &a.ErstelltAm, &a.ErledigtAm, &a.ErledigtNotiz, &a.Von); err != nil {
			return nil, err
		}
		anliegen = append(anliegen, a)
	}
	return anliegen, rows.Err()
}

// Erledige hakt ein offenes Anliegen ab und liefert die Mail-Daten GENAU EINMAL:
// Das WHERE erledigt_am IS NULL macht den Doppelklick zweier Arbeitsplätze
// unschädlich — der zweite bekommt nil und verschickt keine zweite Mail
// (gleiches Muster wie ErledigeKlassensatzReservierung).
func (r *AnliegenRepository) Erledige(ctx context.Context, id, notiz string) (*AnliegenErledigt, error) {
	var e AnliegenErledigt
	err := r.pool.QueryRow(ctx, `
		WITH erledigt AS (
			UPDATE lehrer_anliegen
			SET erledigt_am = CURRENT_TIMESTAMP, erledigt_notiz = $2
			WHERE id = $1 AND erledigt_am IS NULL
			RETURNING art, titel_text, klasse, erledigt_notiz, angefordert_von
		)
		SELECT e.art, e.titel_text, e.klasse, e.erledigt_notiz, b.email
		FROM erledigt e
		LEFT JOIN benutzer b ON e.angefordert_von = b.id`,
		id, notiz).Scan(&e.Art, &e.TitelText, &e.Klasse, &e.ErledigtNotiz, &e.AnfragendeMail)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}
