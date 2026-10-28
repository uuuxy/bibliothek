package repository

import (
	"context"
	"errors"
	"fmt"

	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrInventurLaeuftBereits meldet, dass für denselben Bereich schon ein offener Lauf besteht
// (durchgesetzt von einem partiellen Unique-Index).
var ErrInventurLaeuftBereits = errors.New("für diesen Bereich läuft bereits eine Inventur")

// ErrInventurUeberlappt meldet, dass sich der gewünschte Bereich mit dem eines offenen Laufs
// überschneidet, ohne derselbe zu sein (den Fall fängt der Unique-Index): Der Abschluss des
// einen buchte sonst die gerade gezählten Bücher des anderen als Verlust. Die Tür antwortet mit
// 409 und nennt die Bezeichnung des anderen Laufs.
var ErrInventurUeberlappt = errors.New("bereich überschneidet sich mit einer laufenden Inventur")

// inventurStartSperre reiht alle Starts hintereinander (Advisory-Lock): Zwei gleichzeitige
// Starts mit überlappendem Bereich bestünden sonst beide die Prüfung. Starts sind selten.
const inventurStartSperre int64 = 749_2026

// Inventurlauf ist ein laufender oder abgeschlossener Durchgang einer Inventur.
type Inventurlauf struct {
	ID           string
	BereichArt   string  // "global" | "signature" | "filter"
	Signatur     *string // Art "signature": Präfix von buecher_titel.signatur
	Fach         *string // Art "filter": Fach
	Jahrgang     *int    // Art "filter": Jahrgang
	Bezeichnung  string
	GestartetVon *string
	GestartetAm  string
	Erfasst      int // in diesem Lauf gescannte Exemplare, auch die, die der Bereich nicht erwartet
	// Nur bei abgeschlossenen Läufen gefüllt (LadeAbgeschlosseneLaeufe):
	AbgeschlossenAm *string
	Verluste        int  // beim Abschluss gebuchte Fehlbestände (inventur_verluste)
	Verworfen       bool // mit „Verwerfen" beendet, ohne Verlustbuchung
}

// Bereich leitet aus den gespeicherten Feldern den InventurBereich ab, die eine Quelle für
// Zählung, Scan-Warnung und Verlustbuchung (inventur_bereich.go).
func (s Inventurlauf) Bereich() InventurBereich {
	return InventurBereich{Signatur: s.Signatur, Fach: s.Fach, Jahrgang: s.Jahrgang}
}

// LegeLaufAn legt einen neuen Lauf an. Ein partieller Unique-Index verhindert einen zweiten
// offenen Lauf im selben Bereich; der Fall kommt als ErrInventurLaeuftBereits zurück.
func (a *InventurAbfragen) LegeLaufAn(ctx context.Context, bereichArt string, bereich InventurBereich, bezeichnung, benutzerID string) (*Inventurlauf, error) {
	var benutzerPtr *string
	if benutzerID != "" {
		benutzerPtr = &benutzerID
	}

	tx, err := a.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("inventur-start-transaktion fehlgeschlagen: %w", err)
	}
	defer db.Verwerfe(ctx, tx)

	// Alle Starts hintereinander, damit die Prüfung auf Überlappung nicht selbst von der
	// Reihenfolge abhängt.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, inventurStartSperre); err != nil {
		return nil, fmt.Errorf("inventur-start sperren fehlgeschlagen: %w", err)
	}

	// Gegen alle offenen Läufe prüfen; gleiche Bereiche fängt zusätzlich der Unique-Index beim
	// INSERT.
	if kollision, err := a.findeUeberlappendenLauf(ctx, tx, bereichArt, bereich); err != nil {
		return nil, err
	} else if kollision != "" {
		return nil, regelVerletzt(ErrInventurUeberlappt, "„%s“ läuft bereits", kollision)
	}

	var s Inventurlauf
	err = tx.QueryRow(ctx, `
		INSERT INTO inventur_sessions (scope_type, scope_signatur, scope_subject, scope_grade, scope_label, gestartet_von)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, scope_type, scope_signatur, scope_subject, scope_grade, scope_label, gestartet_am::text
	`, bereichArt, bereich.Signatur, bereich.Fach, bereich.Jahrgang, bezeichnung, benutzerPtr).
		Scan(&s.ID, &s.BereichArt, &s.Signatur, &s.Fach, &s.Jahrgang, &s.Bezeichnung, &s.GestartetAm)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrInventurLaeuftBereits
		}
		return nil, fmt.Errorf("inventur-session anlegen fehlgeschlagen: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("inventur-start committen fehlgeschlagen: %w", err)
	}
	return &s, nil
}

// LaufStand ist der Fortschritt eines offenen Laufs, gezählt nach den Bedingungen des
// Abschlusses: Erwartet sind die Exemplare im Bereich, die im Regal stehen müssten, Fehlend die
// davon, die der Lauf noch nicht erfasst hat. Fehlend ist genau die Zahl, die SchliesseLaufAb
// jetzt als Verlust buchen würde; ein gescanntes Buch, das der Bereich nicht erwartet
// (verliehen, gesperrt), ändert keine der beiden Zahlen.
type LaufStand struct {
	Erwartet int `json:"erwartet"`
	Fehlend  int `json:"fehlend"`
}

// ZaehleStand zählt den Stand eines Laufs. Beide Zahlen kommen aus einer Abfrage und damit aus
// demselben Moment: Eine Rückgabe an der Theke zwischen zwei Abfragen ließe sonst mehr fehlen,
// als erwartet sind.
func (a *InventurAbfragen) ZaehleStand(ctx context.Context, laufID string, bereich InventurBereich) (LaufStand, error) {
	bedingung, args := bereich.bedingung(1)
	laufPlatz := len(args) + 1
	args = append(args, laufID)
	var stand LaufStand
	err := a.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT count(*), count(*) FILTER (WHERE %s)
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		WHERE %s
	`, nichtErfasst(laufPlatz), bedingung), args...).Scan(&stand.Erwartet, &stand.Fehlend)
	if err != nil {
		return LaufStand{}, fmt.Errorf("stand des laufs zählen fehlgeschlagen: %w", err)
	}
	return stand, nil
}

// LadeLauf lädt einen offenen Lauf mit der Zahl seiner Erfassungen, ohne die teurere Zählung
// des Bereichs: für den Scan. Den Stand zählt ZaehleStand. pgx.ErrNoRows, wenn kein offener Lauf
// diese ID trägt.
func (a *InventurAbfragen) LadeLauf(ctx context.Context, id string) (*Inventurlauf, error) {
	var s Inventurlauf
	err := a.db.QueryRow(ctx, `
		SELECT id, scope_type, scope_signatur, scope_subject, scope_grade, scope_label,
		       gestartet_von::text, gestartet_am::text,
		       (SELECT count(*) FROM inventur_erfassungen WHERE session_id = $1)
		FROM inventur_sessions
		WHERE id = $1 AND abgeschlossen_am IS NULL
	`, id).Scan(&s.ID, &s.BereichArt, &s.Signatur, &s.Fach, &s.Jahrgang, &s.Bezeichnung,
		&s.GestartetVon, &s.GestartetAm, &s.Erfasst)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// LadeAbgeschlosseneLaeufe liefert die zuletzt abgeschlossenen Läufe mit der Zahl ihrer
// Verluste: die Liste, über die sich der Fehlbestand einer früheren Inventur wieder aufrufen
// lässt, auch nach einem Neuladen oder an einem anderen Arbeitsplatz. Verworfene stehen mit
// darin und tragen Verworfen: Sie sind beendet, aber nicht geprüft.
func (a *InventurAbfragen) LadeAbgeschlosseneLaeufe(ctx context.Context, limit int) ([]Inventurlauf, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	// Erst kappen, dann zählen: Stünden die beiden Zählungen in der SELECT-Liste, hinge es am
	// Planer, ob er sie hinter das LIMIT schiebt; gemessen zählte er die Verluste einmal je
	// abgeschlossenem Lauf statt zehnmal. Mit der CTE wird nur für die Zeilen gezählt, die
	// herauskommen.
	rows, err := a.db.Query(ctx, `
		WITH limited_sessions AS (
			SELECT id, scope_type, scope_signatur, scope_subject, scope_grade, scope_label,
			       gestartet_von, gestartet_am, abgeschlossen_am, erfasst_gemeldet, verworfen
			FROM inventur_sessions
			WHERE abgeschlossen_am IS NOT NULL
			ORDER BY abgeschlossen_am DESC
			LIMIT $1
		)
		SELECT s.id, s.scope_type, s.scope_signatur, s.scope_subject, s.scope_grade, s.scope_label,
		       s.gestartet_von::text, s.gestartet_am::text, s.abgeschlossen_am::text,
		       COALESCE(e.count, 0),
		       COALESCE(v.count, 0),
		       s.verworfen
		FROM limited_sessions s
		-- COALESCE auf die eingefrorene Zahl: Nur eine offene Session
		-- zählt live; ein abgeschlossener Durchgang behält, was er gezählt hat.
		LEFT JOIN LATERAL (SELECT COALESCE(s.erfasst_gemeldet,
			(SELECT count(*) FROM inventur_erfassungen WHERE session_id = s.id)) as count) e ON true
		LEFT JOIN LATERAL (SELECT count(*) as count FROM inventur_verluste WHERE session_id = s.id) v ON true
		ORDER BY s.abgeschlossen_am DESC
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("abgeschlossene sessions laden fehlgeschlagen: %w", err)
	}
	defer rows.Close()

	laeufe := make([]Inventurlauf, 0)
	for rows.Next() {
		var s Inventurlauf
		if err := rows.Scan(&s.ID, &s.BereichArt, &s.Signatur, &s.Fach, &s.Jahrgang, &s.Bezeichnung,
			&s.GestartetVon, &s.GestartetAm, &s.AbgeschlossenAm, &s.Erfasst, &s.Verluste, &s.Verworfen); err != nil {
			return nil, fmt.Errorf("session-zeile unlesbar: %w", err)
		}
		laeufe = append(laeufe, s)
	}
	return laeufe, rows.Err()
}

// LadeOffeneLaeufe liefert alle laufenden Läufe, für die Anzeige: damit niemand in einem
// Bereich startet, in dem schon gezählt wird. Ihren Stand zählt ZaehleStand je Lauf; es sind
// wenige (idx_inv_session_offen_*).
func (a *InventurAbfragen) LadeOffeneLaeufe(ctx context.Context) ([]Inventurlauf, error) {
	rows, err := a.db.Query(ctx, `
		SELECT s.id, s.scope_type, s.scope_signatur, s.scope_subject, s.scope_grade, s.scope_label,
		       s.gestartet_von::text, s.gestartet_am::text
		FROM inventur_sessions s
		WHERE s.abgeschlossen_am IS NULL
		ORDER BY s.gestartet_am ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("offene sessions laden fehlgeschlagen: %w", err)
	}
	defer rows.Close()

	laeufe := make([]Inventurlauf, 0)
	for rows.Next() {
		var s Inventurlauf
		if err := rows.Scan(&s.ID, &s.BereichArt, &s.Signatur, &s.Fach, &s.Jahrgang, &s.Bezeichnung,
			&s.GestartetVon, &s.GestartetAm); err != nil {
			return nil, fmt.Errorf("session-zeile unlesbar: %w", err)
		}
		laeufe = append(laeufe, s)
	}
	return laeufe, rows.Err()
}
