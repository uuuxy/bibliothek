package repository

import (
	"context"
	"errors"
	"fmt"

	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrInventurLaeuftBereits signalisiert, dass für denselben Scope bereits eine
// offene Inventur-Session existiert (durchgesetzt per partiellem Unique-Index).
var ErrInventurLaeuftBereits = errors.New("für diesen Bereich läuft bereits eine Inventur")

// ErrInventurUeberlappt signalisiert, dass der gewünschte Scope sich mit einer bereits
// offenen Inventur ÜBERSCHNEIDET (nicht identisch — das fängt der Unique-Index). Der
// Abschluss der einen würde sonst die gerade gezählten Bücher der anderen als Verlust
// buchen. Wird als 409 ausgeliefert, mit dem Label der kollidierenden Session im Text.
var ErrInventurUeberlappt = errors.New("bereich überschneidet sich mit einer laufenden Inventur")

// inventurStartSperre serialisiert alle Inventur-Starts (Advisory-Lock), damit die
// Überlappungsprüfung race-sicher ist — zwei gleichzeitige, sich überlappende Starts
// könnten sonst beide die Prüfung passieren. Starts sind selten; Serialisieren ist billig.
const inventurStartSperre int64 = 749_2026

// Inventurlauf beschreibt eine laufende oder abgeschlossene Inventur.
type Inventurlauf struct {
	ID           string
	BereichArt   string  // "global" | "signature" | "filter"
	Signatur     *string // 'signature'-Scope: Präfix von buecher_titel.signatur
	Fach         *string // 'filter'-Scope: Fach
	Jahrgang     *int    // 'filter'-Scope: Klasse
	Bezeichnung  string
	GestartetVon *string
	GestartetAm  string
	Erwartet     int // physisch erwartbare Exemplare im Scope (dynamisch)
	Erfasst      int // in dieser Session gescannte Exemplare
	// Nur bei abgeschlossenen Sessions gefüllt (LadeAbgeschlosseneLaeufe):
	AbgeschlossenAm *string
	Verluste        int  // beim Abschluss gebuchte Fehlbestände (inventur_verluste)
	Verworfen       bool // mit „Verwerfen" beendet, ohne Verlustbuchung (Migration 149)
}

// Bereich leitet aus den gespeicherten Feldern den auswertbaren InventurBereich ab —
// die eine Quelle für Zählung, Scan-Warnung und Verlustbuchung (siehe inventur_bereich.go).
func (s Inventurlauf) Bereich() InventurBereich {
	return InventurBereich{Signatur: s.Signatur, Fach: s.Fach, Jahrgang: s.Jahrgang}
}

// LegeLaufAn legt eine neue Session an. Der partielle Unique-Index aus
// Migration 045 verhindert eine zweite offene Session im selben Scope; dieser Fall
// wird als ErrInventurLaeuftBereits zurückgegeben (nicht als roher DB-Fehler).
func (a *InventurAbfragen) LegeLaufAn(ctx context.Context, bereichArt string, bereich InventurBereich, bezeichnung, benutzerID string) (*Inventurlauf, error) {
	var benutzerPtr *string
	if benutzerID != "" {
		benutzerPtr = &benutzerID
	}

	tx, err := a.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("inventur-start-transaktion fehlgeschlagen: %w", err)
	}
	defer db.SafeRollback(ctx, tx)

	// Alle Starts serialisieren, damit die Überlappungsprüfung nicht selbst ein Race hat.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, inventurStartSperre); err != nil {
		return nil, fmt.Errorf("inventur-start sperren fehlgeschlagen: %w", err)
	}

	// Gegen alle offenen Sessions auf Überlappung prüfen (identische Scopes fängt
	// zusätzlich der partielle Unique-Index beim INSERT ab).
	if kollision, err := a.findeUeberlappendenLauf(ctx, tx, bereichArt, bereich); err != nil {
		return nil, err
	} else if kollision != "" {
		return nil, fmt.Errorf("%w: %q läuft bereits", ErrInventurUeberlappt, kollision)
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

// ZaehleBereich liefert die Anzahl physisch erwartbarer Exemplare im Scope.
func (a *InventurAbfragen) ZaehleBereich(ctx context.Context, bereich InventurBereich) (int, error) {
	bedingung, args := bereich.bedingung(1)
	var anzahl int
	err := a.db.QueryRow(ctx, `
		SELECT count(*)
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		WHERE `+bedingung, args...).Scan(&anzahl)
	if err != nil {
		return 0, fmt.Errorf("scope-zählung fehlgeschlagen: %w", err)
	}
	return anzahl, nil
}

// LadeLauf lädt die Stammdaten einer offenen Session plus die Zahl der in
// ihr erfassten Exemplare (eine schlanke Query, ohne die teurere Scope-Zählung —
// gedacht für den Scan-Pfad). Erwartet bleibt hier 0; wer es braucht (Start/Status),
// ruft zusätzlich ZaehleBereich. Liefert pgx.ErrNoRows, wenn keine offene Session.
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

// LadeLaufMitErwartung lädt eine offene Session inklusive erwartet UND erfasst.
func (a *InventurAbfragen) LadeLaufMitErwartung(ctx context.Context, id string) (*Inventurlauf, error) {
	s, err := a.LadeLauf(ctx, id)
	if err != nil {
		return nil, err
	}
	erwartet, err := a.ZaehleBereich(ctx, s.Bereich())
	if err != nil {
		return nil, err
	}
	s.Erwartet = erwartet
	return s, nil
}

// LadeAbgeschlosseneLaeufe liefert die zuletzt abgeschlossenen Inventuren
// samt Anzahl gebuchter Verluste — die Auswahlliste, über die man den Fehlbestand
// einer früheren Inventur wieder aufrufen kann. Verworfene stehen mit darin und tragen
// Verworfen: Sie sind beendet, aber nicht geprüft (Migration 149).
//
// Ohne sie war GET /api/inventur/fehlbestand unerreichbar: Der Endpunkt braucht eine
// session_id, und die einzige Session-Liste filterte auf abgeschlossen_am IS NULL —
// eine fertige Inventur tauchte dort also nie auf. Der Bericht existierte damit nur im
// Arbeitsspeicher des Browsers, der "Abschließen" gedrückt hatte, und war nach einem
// Neuladen oder an einem anderen Arbeitsplatz weg.
func (a *InventurAbfragen) LadeAbgeschlosseneLaeufe(ctx context.Context, limit int) ([]Inventurlauf, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	// Erst kappen, dann zählen. Standen die beiden count-Unterabfragen wie früher in der
	// SELECT-Liste, HING es am Planer, ob er sie hinter das LIMIT schiebt: gemessen am
	// 09.08.2026 zog er die Erfassungs-Zählung auf zehn Durchläufe zusammen, die
	// Verlust-Zählung aber nicht — die lief 400-mal, einmal je abgeschlossener Inventur.
	// Die CTE macht aus dieser Planer-Laune eine Struktur: gezählt wird nur für die
	// Zeilen, die auch herauskommen.
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
		-- COALESCE auf die eingefrorene Zahl (Migration 103): Nur eine OFFENE Session
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

// LadeOffeneLaeufe liefert alle laufenden Sessions (für die Anzeige, damit
// niemand versehentlich in einen fremden, bereits laufenden Scope startet).
func (a *InventurAbfragen) LadeOffeneLaeufe(ctx context.Context) ([]Inventurlauf, error) {
	// Hier bewusst OHNE die CTE-/LATERAL-Konstruktion der Schwesterfunktion oben: Die
	// lohnt sich nur, weil dort ein LIMIT die Zählung auf zehn Zeilen eindampft. Diese
	// Abfrage hat kein LIMIT — jede offene Session wird ohnehin gezählt, die Umbauten
	// ergäben denselben Plan und nur mehr SQL. Offene Sessions sind zudem per
	// idx_inv_session_offen_* auf eine Handvoll begrenzt.
	rows, err := a.db.Query(ctx, `
		SELECT s.id, s.scope_type, s.scope_signatur, s.scope_subject, s.scope_grade, s.scope_label,
		       s.gestartet_von::text, s.gestartet_am::text,
		       COALESCE(s.erfasst_gemeldet,
		                (SELECT count(*) FROM inventur_erfassungen WHERE session_id = s.id))
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
			&s.GestartetVon, &s.GestartetAm, &s.Erfasst); err != nil {
			return nil, fmt.Errorf("session-zeile unlesbar: %w", err)
		}
		laeufe = append(laeufe, s)
	}
	return laeufe, rows.Err()
}
