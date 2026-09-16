package repository

import (
	"context"
	"errors"
	"strings"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Die Leserzeile und ihr Anmeldekonto — die Abfragen, die beide zusammen betreffen.
//
// Eine Person steht seit Migration 125 in EINER Leserzeile (`leser`); wer sich anmelden
// können soll, hat zusätzlich ein Konto (`benutzer.leser_id`). Die Adresse am Konto ist
// dabei keine Kontaktangabe, sondern der Schlüssel: An ihr erkennt die Anmeldung (IMAP)
// die Person, und `benutzer_email_unique` verhindert darüber den Doppeleintrag.
//
// Warum die Abfragen hier und nicht im Handler stehen: In api/ ist SQL eingefroren
// (api/schichtung_test.go). Die Regeln — welche Zeile bei mehreren Konten gilt, dass ein
// fehlendes Konto kein Fehler ist, dass die Namen aus der LESERZEILE kommen — gehören an
// eine Stelle.

// KontoSchreiber ist der kleinste gemeinsame Nenner von pgx.Tx und Pool: Beim ANLEGEN
// entsteht das Konto in derselben Transaktion wie die Leserzeile, beim NACHTRAGEN steht
// die Zeile längst.
type KontoSchreiber interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// ErrLeserNichtGefunden meldet eine Leser-ID, zu der es keine Zeile gibt. Der Handler
// macht daraus eine 404 statt einer 500 — „nicht gefunden" ist keine Störung.
var ErrLeserNichtGefunden = errors.New("leser nicht gefunden")

// LeserArt liefert die Art einer Leserzeile (schueler | lehrkraft | liv).
func LeserArt(ctx context.Context, pool db.PgxPoolIface, leserID string) (string, error) {
	var art string
	err := pool.QueryRow(ctx, `SELECT art FROM leser WHERE id = $1`, leserID).Scan(&art)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrLeserNichtGefunden
	}
	return art, err
}

// LeserArtUndKontoEmail liefert die Art der Leserzeile und die Adresse an ihrem Konto
// ("" = kein Konto) in EINER Abfrage — der Handler entscheidet an genau diesem Paar.
//
// Bei mehreren Konten an derselben Zeile gilt das älteste. Das ist der Normalfall-Fall
// von einem Konto (trg_benutzer_hat_leserzeile), aber die Reihenfolge macht die Antwort
// eindeutig, statt sich auf eine Zusage zu verlassen, die kein Index hält.
func LeserArtUndKontoEmail(ctx context.Context, pool db.PgxPoolIface, leserID string) (art, email string, err error) {
	err = pool.QueryRow(ctx, `
		SELECT l.art,
		       COALESCE((SELECT b.email FROM benutzer b
		                  WHERE b.leser_id = l.id
		                  ORDER BY b.erstellt_am, b.id LIMIT 1), '')
		FROM leser l
		WHERE l.id = $1`, leserID).Scan(&art, &email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrLeserNichtGefunden
	}
	return art, email, err
}

// KontoEmail liefert die Schul-Adresse am Konto dieser Leserzeile ("" = kein Konto).
func KontoEmail(ctx context.Context, pool db.PgxPoolIface, leserID string) (string, error) {
	var email string
	err := pool.QueryRow(ctx, `
		SELECT COALESCE(email, '') FROM benutzer
		WHERE leser_id = $1
		ORDER BY erstellt_am, id LIMIT 1`, leserID).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil // kein Konto ist kein Fehler
	}
	return email, err
}

// LeserName liefert Vor- und Nachnamen einer Leserzeile.
func LeserName(ctx context.Context, pool db.PgxPoolIface, leserID string) (vorname, nachname string, err error) {
	err = pool.QueryRow(ctx,
		`SELECT COALESCE(vorname, ''), nachname FROM leser WHERE id = $1`, leserID).Scan(&vorname, &nachname)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrLeserNichtGefunden
	}
	return vorname, nachname, err
}

// ErrKontoNichtEntstanden meldet ein INSERT ohne Zeile. Es ist der Phantom-Erfolg, den
// die Ratsche phantom_erfolg_test.go abfängt: Der Dialog meldete „angelegt", die Person
// stünde in der Leserdatei und käme nie ins Portal.
var ErrKontoNichtEntstanden = errors.New("das Konto ist nicht entstanden")

// LegeKollegiumskonto hängt an eine Leserzeile das Anmeldekonto.
//
// `leserID` wird ausdrücklich mitgegeben, damit der Wächter trg_benutzer_hat_leserzeile
// NICHT anspringt: Er legt zu jedem Konto ohne Leserzeile eine frische an, und das wäre
// die zweite — genau der Doppeleintrag, um den es geht.
//
// aktiv=false hinterlässt einen Antrag (zugang_beantragt_am), den die Freischaltungs-Zeile
// der Benutzerverwaltung zeigt.
//
// Eine belegte Adresse kommt als pgconn.PgError 23505 zurück; der Aufrufer macht daraus
// eine Auskunft (409), keine Störung.
func LegeKollegiumskonto(ctx context.Context, q KontoSchreiber, vorname, nachname, email, leserID string, aktiv bool) error {
	tag, err := q.Exec(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, leser_id, zugang_beantragt_am)
		VALUES ($1, $2, $3, 'kollegium', $4, $5, CASE WHEN $4 THEN NULL ELSE CURRENT_TIMESTAMP END)
	`, vorname, nachname, strings.ToLower(strings.TrimSpace(email)), aktiv, leserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrKontoNichtEntstanden
	}
	return nil
}

// IstAdressenKollision erkennt die belegte E-Mail-Adresse (benutzer_email_unique).
func IstAdressenKollision(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// LeserIDVonKonto liefert die Leserzeile zu einem Konto ("" = unbekanntes Konto).
//
// Gebraucht für den Selbstschutz beim Löschen: Wer seine eigene Leserzeile löscht,
// verliert beim Kollegium im selben Vorgang sein Konto.
func LeserIDVonKonto(ctx context.Context, pool db.PgxPoolIface, kontoID string) (string, error) {
	var leserID string
	err := pool.QueryRow(ctx,
		`SELECT COALESCE(leser_id::text, '') FROM benutzer WHERE id = $1`, kontoID).Scan(&leserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return leserID, err
}
