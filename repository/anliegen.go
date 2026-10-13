package repository

import (
	"context"
	"errors"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// Anliegen ist ein Wunsch oder eine Meldung einer Lehrkraft: „Ich möchte in der 8G3 den
// Markl 2" oder „die 8G3 hat die falschen Bücher". Beide Fälle stehen in einer Liste, die die
// LMF abarbeitet; beim Abhaken geht eine Mail an die Lehrkraft.
type Anliegen struct {
	ID         string     `json:"id"`
	Art        string     `json:"art"` // 'wunsch' | 'meldung'
	TitelText  string     `json:"titel_text"`
	Klasse     string     `json:"klasse"`
	Kommentar  string     `json:"kommentar,omitempty"`
	ErstelltAm time.Time  `json:"erstellt_am"`
	ErledigtAm *time.Time `json:"erledigt_am,omitempty"`
	// ErledigtNotiz ist die Zeile der LMF beim Abhaken („bestellt", „getauscht am 3.9."); sie
	// steht auch in der Mail an die Lehrkraft.
	ErledigtNotiz string `json:"erledigt_notiz,omitempty"`
	// Von ist Vorname und Nachname der Lehrkraft und leer, wenn das Konto gelöscht ist.
	Von string `json:"von,omitempty"`
}

// AnliegenErledigt trägt, was die Mail nach dem Abhaken braucht.
type AnliegenErledigt struct {
	Art            string
	TitelText      string
	Klasse         string
	ErledigtNotiz  string
	AnfragendeMail *string
}

// AnliegenAbfragen liest und schreibt die Wünsche und Meldungen der Lehrkräfte.
type AnliegenAbfragen struct {
	pool db.PgxPoolIface
}

// NewAnliegenAbfragen liefert die Abfragen der Anliegen am übergebenen Pool.
func NewAnliegenAbfragen(pool db.PgxPoolIface) *AnliegenAbfragen {
	return &AnliegenAbfragen{pool: pool}
}

// NeuesAnliegen sind die Felder eines neu angelegten Anliegens. Sie stehen in einer Struktur
// und nicht als Zeichenketten in Reihe: Wer dort Klasse und Kommentar vertauscht, bekommt
// keinen Fehler des Compilers, nur eine falsch einsortierte Meldung in der Liste der LMF.
//
// Die Spalten titel_id und isbn der Tabelle schreibt niemand; sie bleiben auf ihrer Vorgabe,
// NULL und leerer Text.
type NeuesAnliegen struct {
	Art            string
	TitelText      string
	Klasse         string
	Kommentar      string
	AngefordertVon string // ID des Kontos der Lehrkraft
}

// LegeAn legt ein Anliegen an und liefert seine ID.
func (a *AnliegenAbfragen) LegeAn(ctx context.Context, neu NeuesAnliegen) (string, error) {
	var id string
	err := a.pool.QueryRow(ctx, `
		INSERT INTO lehrer_anliegen (art, titel_text, klasse, kommentar, angefordert_von)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		neu.Art, neu.TitelText, neu.Klasse, neu.Kommentar, neu.AngefordertVon).Scan(&id)
	return id, err
}

// LadeOffene liefert alle offenen Anliegen für die Liste der LMF, das älteste zuerst: Wer
// zuerst fragt, ist zuerst an der Reihe.
func (a *AnliegenAbfragen) LadeOffene(ctx context.Context) ([]Anliegen, error) {
	return a.lade(ctx, `
		SELECT a.id, a.art, a.titel_text, a.klasse,
		       a.kommentar, a.erstellt_am, a.erledigt_am, a.erledigt_notiz,
		       COALESCE(btrim(b.vorname || ' ' || b.nachname), '')
		FROM lehrer_anliegen a
		LEFT JOIN benutzer b ON a.angefordert_von = b.id
		WHERE a.erledigt_am IS NULL
		ORDER BY a.erstellt_am ASC`)
}

// ZaehleOffene zählt die offenen Anliegen für die Zahl am Menüpunkt „Bestellungen“.
func (a *AnliegenAbfragen) ZaehleOffene(ctx context.Context) (int, error) {
	var n int
	err := a.pool.QueryRow(ctx, `SELECT count(*) FROM lehrer_anliegen WHERE erledigt_am IS NULL`).Scan(&n)
	return n, err
}

// LadeEigene liefert die Anliegen einer Lehrkraft für das Portal, das neueste zuerst und
// höchstens 50: Das Portal ist kein Archiv.
func (a *AnliegenAbfragen) LadeEigene(ctx context.Context, benutzerID string) ([]Anliegen, error) {
	return a.lade(ctx, `
		SELECT a.id, a.art, a.titel_text, a.klasse,
		       a.kommentar, a.erstellt_am, a.erledigt_am, a.erledigt_notiz, ''
		FROM lehrer_anliegen a
		WHERE a.angefordert_von = $1
		ORDER BY a.erstellt_am DESC
		LIMIT 50`, benutzerID)
}

func (a *AnliegenAbfragen) lade(ctx context.Context, query string, args ...any) ([]Anliegen, error) {
	rows, err := a.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	anliegen := []Anliegen{}
	for rows.Next() {
		var eines Anliegen
		if err := rows.Scan(&eines.ID, &eines.Art, &eines.TitelText, &eines.Klasse,
			&eines.Kommentar, &eines.ErstelltAm, &eines.ErledigtAm, &eines.ErledigtNotiz, &eines.Von); err != nil {
			return nil, err
		}
		anliegen = append(anliegen, eines)
	}
	return anliegen, rows.Err()
}

// Erledige hakt ein offenes Anliegen ab und liefert die Angaben für die Mail genau einmal:
// Klicken zwei Arbeitsplätze zugleich, trifft WHERE erledigt_am IS NULL nur den ersten; der
// zweite bekommt nil und verschickt keine zweite Mail.
func (a *AnliegenAbfragen) Erledige(ctx context.Context, id, notiz string) (*AnliegenErledigt, error) {
	var e AnliegenErledigt
	err := a.pool.QueryRow(ctx, `
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
