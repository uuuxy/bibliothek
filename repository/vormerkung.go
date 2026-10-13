package repository

import (
	"context"
	"errors"
	"time"

	"bibliothek/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrTitelBereitsAusgeliehen meldet, dass ein Schüler einen Titel vormerken will, den er selbst
// ausgeliehen hat: Er könnte das Buch sonst bei der Rückgabe sofort wieder für sich bekommen
// und die Warteschlange auf Dauer besetzen. Der Text steht so vor dem Menschen (409).
//
//nolint:staticcheck // ST1005: bewusst großgeschrieben, Endnutzer-Meldung
var ErrTitelBereitsAusgeliehen = errors.New("Buch wird aktuell bereits von diesem Schüler ausgeliehen")

// ErrVormerkungBereitsVorhanden meldet die zweite Vormerkung desselben Schülers auf denselben
// Titel. UNIQUE(titel_id, schueler_id) fängt sie in der Datenbank ab; ohne den eigenen
// Fehlerwert käme für „steht schon auf der Liste" eine Störungsmeldung (500). Der Text steht so
// vor dem Menschen (409).
//
//nolint:staticcheck // ST1005: bewusst großgeschrieben, Endnutzer-Meldung
var ErrVormerkungBereitsVorhanden = errors.New("Dieser Schüler hat den Titel bereits vorgemerkt")

// ErrVormerkungNurFuerSchueler meldet eine Vormerkung für einen Leser, der kein Schüler ist
// oder den es nicht gibt. Die Warteschlange liest die Sicht schueler: Eine Vormerkung für einen
// Kollegen würde nie bedient und stünde ohne Namen in der Liste. Die Regel steht deshalb an
// LegeAn und nicht nur in der Oberfläche, die zum Vormerken ohnehin nur Schüler anbietet. Der
// Text steht so vor dem Menschen (409).
//
//nolint:staticcheck // ST1005: bewusst großgeschrieben, Endnutzer-Meldung
var ErrVormerkungNurFuerSchueler = errors.New("Vormerken lässt sich nur für Schüler")

// Vormerkung ist die Vormerkung eines Schülers auf einen Titel.
type Vormerkung struct {
	ID           string    `json:"id"`
	TitelID      string    `json:"titel_id"`
	TitelName    string    `json:"titel"`
	Notiz        string    `json:"notiz,omitempty"`
	ErstelltAm   time.Time `json:"erstellt_am"`
	SchuelerID   string    `json:"schueler_id,omitempty"`
	SchuelerName string    `json:"schueler_name,omitempty"`
	// Status ist 'wartend' oder 'abholbereit' (chk_vormerkung_status). Die Oberfläche
	// unterscheidet daran „wartet seit …" von einer Vormerkung, die zur Abholung bereitliegt
	// und deren Frist läuft.
	Status string `json:"status"`
	// BereitgestelltBis ist nur bei 'abholbereit' gesetzt: das Ende der Abholfrist
	// (BestimmeAbholfrist).
	BereitgestelltBis *time.Time `json:"bereitgestellt_bis,omitempty"`
}

// VormerkungsAbfragen liest und schreibt die Vormerkungen.
type VormerkungsAbfragen interface {
	LadeListe(ctx context.Context, titelID, schuelerID string) ([]Vormerkung, error)
	LegeAn(ctx context.Context, titelID, notiz, schuelerID string) (string, error)
	Loesche(ctx context.Context, id string) error
	// LasseAbgelaufeneVerfallen löscht bereitgelegte Vormerkungen, deren Abholfrist abgelaufen
	// ist, und bedient den Nächsten.
	LasseAbgelaufeneVerfallen(ctx context.Context) (verfallen, neuBereitgestellt int, err error)
}

// LasseAbgelaufeneVerfallen behandelt bereitgelegte Vormerkungen, deren Abholfrist abgelaufen
// ist: Wer nicht abholt, verliert seinen Platz, und der Nächste ist an der Reihe. Ohne diesen
// Lauf bliebe die Vormerkung für immer abholbereit, und das Exemplar würde nie wieder
// zugeteilt. Geliefert wird, wie viele verfallen sind und wie viele neu bereitgestellt wurden.
//
//   - Die abgelaufene Vormerkung wird gelöscht; RETURNING liefert Exemplar und Titel.
//   - Ist das Exemplar noch frei (nicht ausgeliehen, nicht ausgesondert), bekommt es der
//     nächste wartende Schüler, der abholen darf, mit neuer Abholfrist ab a.uhr(). Sonst
//     bedient die nächste Rückgabe die Warteschlange.
//
// Alles in einer Transaktion; FOR UPDATE SKIP LOCKED verhindert, dass der Lauf und eine
// gleichzeitige Rückgabe aufeinander warten oder dasselbe Exemplar zweimal zuteilen.
func (a *pgVormerkungsAbfragen) LasseAbgelaufeneVerfallen(ctx context.Context) (int, int, error) {
	tx, err := a.db.Begin(ctx)
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
	freigaben, err := sammleFreigaben(rows)
	if err != nil {
		return 0, 0, err
	}

	neuBereit := 0
	for _, f := range freigaben {
		if f.exemplarID == nil || f.titelID == nil {
			continue
		}
		// Den nächsten Wartenden nur bedienen, wenn das Exemplar noch frei ist, wie beim
		// endgültigen Löschen eines Schülers (vormerkung_nachruecken.go).
		bedient, err := bedieneNaechstenWartenden(ctx, tx, *f.exemplarID, *f.titelID, a.uhr())
		if err != nil {
			return 0, 0, err
		}
		if bedient {
			neuBereit++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}
	return len(freigaben), neuBereit, nil
}

type pgVormerkungsAbfragen struct {
	db db.PgxPoolIface
	// jetzt ist die Uhr, ab der eine neue Abholfrist zählt (Nachrücken im Verfall-Lauf und
	// beim Löschen); nil heißt time.Now. Tests setzen einen festen Tag vor einem Wochenende
	// oder vor den Ferien.
	jetzt func() time.Time
}

// uhr liefert den Zeitpunkt, ab dem eine neue Abholfrist zählt.
func (a *pgVormerkungsAbfragen) uhr() time.Time {
	if a.jetzt != nil {
		return a.jetzt()
	}
	return time.Now()
}

// NewVormerkungsAbfragen liefert die Abfragen der Vormerkungen an Postgres.
func NewVormerkungsAbfragen(db db.PgxPoolIface) VormerkungsAbfragen {
	return &pgVormerkungsAbfragen{db: db}
}

// ErrVormerkungFilterFehlt meldet einen Aufruf von LadeListe ohne Titel und ohne Schüler. Er
// lieferte sonst alle Vormerkungen mit Namen und Klassen: einen Abzug, den die Oberfläche nie
// anfordert und der Personendaten preisgäbe, sobald das Recht manage_vormerkungen an eine
// Helfer-Rolle vergeben ist. Der Filter ist deshalb Pflicht.
//
//nolint:staticcheck // ST1005: bewusst großgeschrieben, Endnutzer-Meldung
var ErrVormerkungFilterFehlt = errors.New("Vormerkungen brauchen einen Titel- oder Schüler-Filter")

// LadeListe liefert die Vormerkungen eines Titels oder eines Schülers, die ältesten zuerst.
// Ohne einen der beiden Filter liefert es ErrVormerkungFilterFehlt.
func (a *pgVormerkungsAbfragen) LadeListe(ctx context.Context, titelID, schuelerID string) ([]Vormerkung, error) {
	var rows pgx.Rows
	var err error

	if titelID != "" {
		rows, err = a.db.Query(ctx, `
			SELECT v.id, v.titel_id, bt.titel, COALESCE(v.notiz, ''), v.erstellt_am,
			       COALESCE(s.id::text, ''), COALESCE(s.vorname || ' ' || s.nachname || ', ' || s.klasse, ''),
			       v.status, v.bereitgestellt_bis
			FROM vormerkungen v
			JOIN buecher_titel bt ON bt.id = v.titel_id
			LEFT JOIN schueler s ON s.id = v.schueler_id
			WHERE v.titel_id = $1
			ORDER BY v.erstellt_am ASC
		`, titelID)
	} else if schuelerID != "" {
		rows, err = a.db.Query(ctx, `
			SELECT v.id, v.titel_id, bt.titel, COALESCE(v.notiz, ''), v.erstellt_am,
			       COALESCE(s.id::text, ''), COALESCE(s.vorname || ' ' || s.nachname || ', ' || s.klasse, ''),
			       v.status, v.bereitgestellt_bis
			FROM vormerkungen v
			JOIN buecher_titel bt ON bt.id = v.titel_id
			LEFT JOIN schueler s ON s.id = v.schueler_id
			WHERE v.schueler_id = $1
			ORDER BY v.erstellt_am ASC
		`, schuelerID)
	} else {
		return nil, ErrVormerkungFilterFehlt
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vormerkungen []Vormerkung
	for rows.Next() {
		var v Vormerkung
		if err := rows.Scan(&v.ID, &v.TitelID, &v.TitelName, &v.Notiz, &v.ErstelltAm, &v.SchuelerID, &v.SchuelerName,
			&v.Status, &v.BereitgestelltBis); err != nil {
			return nil, err
		}
		vormerkungen = append(vormerkungen, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return vormerkungen, nil
}

// LegeAn legt eine Vormerkung an. Ein Schüler, der ein Exemplar des Titels selbst ausgeliehen
// hat, bekommt ErrTitelBereitsAusgeliehen; die Prüfung läuft nur mit schuelerID, eine
// Vormerkung ohne Schüler ist davon nicht betroffen.
func (a *pgVormerkungsAbfragen) LegeAn(ctx context.Context, titelID, notiz, schuelerID string) (string, error) {
	if schuelerID != "" {
		// Gegen die Sicht schueler, mit Absicht: Es ist dieselbe Frage, die die Warteschlange
		// beim Bedienen stellt (ErrVormerkungNurFuerSchueler).
		var istSchueler bool
		if err := a.db.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM schueler WHERE id = $1)`, schuelerID,
		).Scan(&istSchueler); err != nil {
			return "", err
		}
		if !istSchueler {
			return "", ErrVormerkungNurFuerSchueler
		}

		var bereitsAusgeliehen bool
		if err := a.db.QueryRow(ctx, `
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
	err := a.db.QueryRow(ctx, `
		INSERT INTO vormerkungen (titel_id, notiz, schueler_id)
		VALUES ($1, NULLIF($2, ''), NULLIF($3, '')::uuid)
		RETURNING id
	`, titelID, notiz, schuelerID).Scan(&id)
	// Die zweite Vormerkung ist kein Serverfehler, sondern eine Antwort: Der Schüler steht
	// schon auf der Liste. Erkannt wird sie am Constraint der Tabelle und nicht an einer
	// Abfrage davor; die ließe zwischen Prüfen und Schreiben ein Fenster (zwei Arbeitsplätze,
	// derselbe Titel).
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "vormerkungen_titel_id_schueler_id_key" {
		return "", ErrVormerkungBereitsVorhanden
	}
	return id, err
}

// Loesche löscht eine Vormerkung, gleich wem sie gehört: Das Recht manage_vormerkungen ist das
// Verwaltungsrecht der Theke, und Vormerkungen anderer auf Zuruf zu löschen ist sein Zweck.
//
// Lag für die Vormerkung schon ein Exemplar im Abholfach, rückt der Nächste in derselben
// Transaktion nach (vormerkung_nachruecken.go); sonst bliebe er für immer wartend, und das
// Exemplar gälte als frei.
func (a *pgVormerkungsAbfragen) Loesche(ctx context.Context, id string) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)
	var titelID string
	var bereitgestellt *string
	err = tx.QueryRow(ctx, `
		DELETE FROM vormerkungen WHERE id = $1
		RETURNING titel_id::text, CASE WHEN status = 'abholbereit' THEN bereitgestellt_exemplar_id::text END`,
		id).Scan(&titelID, &bereitgestellt)
	// Keine Zeile heißt unbekannte ID; „gelöscht" wäre dann falsch.
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrVormerkungNichtGefunden
	}
	if err != nil {
		return err
	}
	if bereitgestellt != nil {
		if _, err := bedieneNaechstenWartenden(ctx, tx, *bereitgestellt, titelID, a.uhr()); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ErrVormerkungNichtGefunden meldet eine unbekannte ID einer Vormerkung.
var ErrVormerkungNichtGefunden = errors.New("vormerkung nicht gefunden")
