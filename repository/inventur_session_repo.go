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

// inventurStartLockKey serialisiert alle Inventur-Starts (Advisory-Lock), damit die
// Überlappungsprüfung race-sicher ist — zwei gleichzeitige, sich überlappende Starts
// könnten sonst beide die Prüfung passieren. Starts sind selten; Serialisieren ist billig.
const inventurStartLockKey int64 = 749_2026

// InventurSession beschreibt eine laufende oder abgeschlossene Inventur.
type InventurSession struct {
	ID           string
	ScopeType    string  // "global" | "signature" | "filter"
	Signatur     *string // 'signature'-Scope: Präfix von buecher_titel.signatur
	Subject      *string // 'filter'-Scope: Fach
	Grade        *int    // 'filter'-Scope: Klasse
	ScopeLabel   string
	GestartetVon *string
	GestartetAm  string
	Erwartet     int // physisch erwartbare Exemplare im Scope (dynamisch)
	Erfasst      int // in dieser Session gescannte Exemplare
	// Nur bei abgeschlossenen Sessions gefüllt (ListAbgeschlosseneInventurSessions):
	AbgeschlossenAm *string
	Verluste        int // beim Abschluss gebuchte Fehlbestände (inventur_verluste)
}

// Scope leitet aus den gespeicherten Feldern den auswertbaren InventurScope ab —
// die eine Quelle für Zählung, Scan-Warnung und Verlustbuchung (siehe inventur_scope.go).
func (s InventurSession) Scope() InventurScope {
	return InventurScope{Signatur: s.Signatur, Subject: s.Subject, Grade: s.Grade}
}

// CreateInventurSession legt eine neue Session an. Der partielle Unique-Index aus
// Migration 045 verhindert eine zweite offene Session im selben Scope; dieser Fall
// wird als ErrInventurLaeuftBereits zurückgegeben (nicht als roher DB-Fehler).
func (r *InventoryRepository) CreateInventurSession(ctx context.Context, scopeType string, scope InventurScope, scopeLabel, benutzerID string) (*InventurSession, error) {
	var benutzerPtr *string
	if benutzerID != "" {
		benutzerPtr = &benutzerID
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("inventur-start-transaktion fehlgeschlagen: %w", err)
	}
	defer db.SafeRollback(ctx, tx)

	// Alle Starts serialisieren, damit die Überlappungsprüfung nicht selbst ein Race hat.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, inventurStartLockKey); err != nil {
		return nil, fmt.Errorf("inventur-start sperren fehlgeschlagen: %w", err)
	}

	// Gegen alle offenen Sessions auf Überlappung prüfen (identische Scopes fängt
	// zusätzlich der partielle Unique-Index beim INSERT ab).
	if label, err := r.findeUeberlappendeSession(ctx, tx, scopeType, scope); err != nil {
		return nil, err
	} else if label != "" {
		return nil, fmt.Errorf("%w: %q läuft bereits", ErrInventurUeberlappt, label)
	}

	var s InventurSession
	err = tx.QueryRow(ctx, `
		INSERT INTO inventur_sessions (scope_type, scope_signatur, scope_subject, scope_grade, scope_label, gestartet_von)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, scope_type, scope_signatur, scope_subject, scope_grade, scope_label, gestartet_am::text
	`, scopeType, scope.Signatur, scope.Subject, scope.Grade, scopeLabel, benutzerPtr).
		Scan(&s.ID, &s.ScopeType, &s.Signatur, &s.Subject, &s.Grade, &s.ScopeLabel, &s.GestartetAm)
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

// ZaehleScope liefert die Anzahl physisch erwartbarer Exemplare im Scope.
func (r *InventoryRepository) ZaehleScope(ctx context.Context, scope InventurScope) (int, error) {
	bedingung, args := scope.Bedingung(1)
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT count(*)
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		WHERE `+bedingung, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("scope-zählung fehlgeschlagen: %w", err)
	}
	return count, nil
}

// LadeInventurSession lädt die Stammdaten einer offenen Session plus die Zahl der in
// ihr erfassten Exemplare (eine schlanke Query, ohne die teurere Scope-Zählung —
// gedacht für den Scan-Pfad). Erwartet bleibt hier 0; wer es braucht (Start/Status),
// ruft zusätzlich ZaehleScope. Liefert pgx.ErrNoRows, wenn keine offene Session.
func (r *InventoryRepository) LadeInventurSession(ctx context.Context, id string) (*InventurSession, error) {
	var s InventurSession
	err := r.db.QueryRow(ctx, `
		SELECT id, scope_type, scope_signatur, scope_subject, scope_grade, scope_label,
		       gestartet_von::text, gestartet_am::text,
		       (SELECT count(*) FROM inventur_erfassungen WHERE session_id = $1)
		FROM inventur_sessions
		WHERE id = $1 AND abgeschlossen_am IS NULL
	`, id).Scan(&s.ID, &s.ScopeType, &s.Signatur, &s.Subject, &s.Grade, &s.ScopeLabel,
		&s.GestartetVon, &s.GestartetAm, &s.Erfasst)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetInventurSession lädt eine offene Session inklusive erwartet UND erfasst.
func (r *InventoryRepository) GetInventurSession(ctx context.Context, id string) (*InventurSession, error) {
	s, err := r.LadeInventurSession(ctx, id)
	if err != nil {
		return nil, err
	}
	erwartet, err := r.ZaehleScope(ctx, s.Scope())
	if err != nil {
		return nil, err
	}
	s.Erwartet = erwartet
	return s, nil
}

// ListAbgeschlosseneInventurSessions liefert die zuletzt abgeschlossenen Inventuren
// samt Anzahl gebuchter Verluste — die Auswahlliste, über die man den Fehlbestand
// einer früheren Inventur wieder aufrufen kann.
//
// Ohne sie war GET /api/inventur/fehlbestand unerreichbar: Der Endpunkt braucht eine
// session_id, und die einzige Session-Liste filterte auf abgeschlossen_am IS NULL —
// eine fertige Inventur tauchte dort also nie auf. Der Bericht existierte damit nur im
// Arbeitsspeicher des Browsers, der "Abschließen" gedrückt hatte, und war nach einem
// Neuladen oder an einem anderen Arbeitsplatz weg.
func (r *InventoryRepository) ListAbgeschlosseneInventurSessions(ctx context.Context, limit int) ([]InventurSession, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	// Erst kappen, dann zählen. Standen die beiden count-Unterabfragen wie früher in der
	// SELECT-Liste, HING es am Planer, ob er sie hinter das LIMIT schiebt: gemessen am
	// 09.08.2026 zog er die Erfassungs-Zählung auf zehn Durchläufe zusammen, die
	// Verlust-Zählung aber nicht — die lief 400-mal, einmal je abgeschlossener Inventur.
	// Die CTE macht aus dieser Planer-Laune eine Struktur: gezählt wird nur für die
	// Zeilen, die auch herauskommen.
	rows, err := r.db.Query(ctx, `
		WITH limited_sessions AS (
			SELECT id, scope_type, scope_signatur, scope_subject, scope_grade, scope_label,
			       gestartet_von, gestartet_am, abgeschlossen_am
			FROM inventur_sessions
			WHERE abgeschlossen_am IS NOT NULL
			ORDER BY abgeschlossen_am DESC
			LIMIT $1
		)
		SELECT s.id, s.scope_type, s.scope_signatur, s.scope_subject, s.scope_grade, s.scope_label,
		       s.gestartet_von::text, s.gestartet_am::text, s.abgeschlossen_am::text,
		       COALESCE(e.count, 0),
		       COALESCE(v.count, 0)
		FROM limited_sessions s
		LEFT JOIN LATERAL (SELECT count(*) as count FROM inventur_erfassungen WHERE session_id = s.id) e ON true
		LEFT JOIN LATERAL (SELECT count(*) as count FROM inventur_verluste WHERE session_id = s.id) v ON true
		ORDER BY s.abgeschlossen_am DESC
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("abgeschlossene sessions laden fehlgeschlagen: %w", err)
	}
	defer rows.Close()

	sessions := make([]InventurSession, 0)
	for rows.Next() {
		var s InventurSession
		if err := rows.Scan(&s.ID, &s.ScopeType, &s.Signatur, &s.Subject, &s.Grade, &s.ScopeLabel,
			&s.GestartetVon, &s.GestartetAm, &s.AbgeschlossenAm, &s.Erfasst, &s.Verluste); err != nil {
			return nil, fmt.Errorf("session-zeile unlesbar: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// ListOffeneInventurSessions liefert alle laufenden Sessions (für die Anzeige, damit
// niemand versehentlich in einen fremden, bereits laufenden Scope startet).
func (r *InventoryRepository) ListOffeneInventurSessions(ctx context.Context) ([]InventurSession, error) {
	// Hier bewusst OHNE die CTE-/LATERAL-Konstruktion der Schwesterfunktion oben: Die
	// lohnt sich nur, weil dort ein LIMIT die Zählung auf zehn Zeilen eindampft. Diese
	// Abfrage hat kein LIMIT — jede offene Session wird ohnehin gezählt, die Umbauten
	// ergäben denselben Plan und nur mehr SQL. Offene Sessions sind zudem per
	// idx_inv_session_offen_* auf eine Handvoll begrenzt.
	rows, err := r.db.Query(ctx, `
		SELECT s.id, s.scope_type, s.scope_signatur, s.scope_subject, s.scope_grade, s.scope_label,
		       s.gestartet_von::text, s.gestartet_am::text,
		       (SELECT count(*) FROM inventur_erfassungen WHERE session_id = s.id)
		FROM inventur_sessions s
		WHERE s.abgeschlossen_am IS NULL
		ORDER BY s.gestartet_am ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("offene sessions laden fehlgeschlagen: %w", err)
	}
	defer rows.Close()

	sessions := make([]InventurSession, 0)
	for rows.Next() {
		var s InventurSession
		if err := rows.Scan(&s.ID, &s.ScopeType, &s.Signatur, &s.Subject, &s.Grade, &s.ScopeLabel,
			&s.GestartetVon, &s.GestartetAm, &s.Erfasst); err != nil {
			return nil, fmt.Errorf("session-zeile unlesbar: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}
