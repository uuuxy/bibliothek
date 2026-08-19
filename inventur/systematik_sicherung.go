package inventur

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// dbSchreiber ist der minimale Datenbankzugriff der Fach-Registrierung. Pool
// (db.PgxPoolIface) und pgx.Tx erfüllen ihn beide — der Sammelimport registriert
// innerhalb seiner Transaktion.
type dbSchreiber interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// StelleFaecherSicher registriert unbekannte Fächer in systematik_kategorien, BEVOR
// ein Schreibpfad sie in buecher_titel.subject einträgt — subject ist seit Migration
// 078 ein Fremdschlüssel auf systematik_kategorien(bezeichnung). Die Lehre aus
// Migration 021→060: Ein FK, den die Importe nicht bedienen, lässt jeden Import mit
// neuem Fach scheitern — und wird beim nächsten Aufräumen wieder abgerissen.
//
// Liefert je ROH-Eingabe die kanonische Schreibweise; eine bestehende Registrierung
// gewinnt case-insensitiv. Der Aufrufer schreibt den GELIEFERTEN Wert, nicht seinen
// rohen — "deutsch" neben "Deutsch" scheiterte sonst am FK. Leere Eingaben fehlen in
// der Karte und liefern beim Zugriff "", das die Schreibpfade per NULLIF zu NULL machen.
func StelleFaecherSicher(ctx context.Context, db dbSchreiber, faecher []string) (map[string]string, error) {
	kanonisch := make(map[string]string, len(faecher))
	for _, roh := range faecher {
		if _, erledigt := kanonisch[roh]; erledigt {
			continue
		}
		fach := strings.TrimSpace(roh)
		if fach == "" {
			continue
		}
		bez, err := sichereEinFach(ctx, db, fach)
		if err != nil {
			return nil, err
		}
		kanonisch[roh] = bez
	}
	return kanonisch, nil
}

// sichereEinFach liest die kanonische Bezeichnung oder legt sie an. Zwei Durchläufe:
// Verliert die Registrierung ein Rennen gegen einen parallelen Import (ON CONFLICT
// DO NOTHING), findet der zweite Lauf die Zeile des Gewinners.
func sichereEinFach(ctx context.Context, db dbSchreiber, fach string) (string, error) {
	for versuch := 0; versuch < 2; versuch++ {
		var bez string
		err := db.QueryRow(ctx,
			`SELECT bezeichnung FROM systematik_kategorien WHERE lower(bezeichnung) = lower($1)`,
			fach).Scan(&bez)
		if err == nil {
			return bez, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("fach %q nachschlagen fehlgeschlagen: %w", fach, err)
		}
		if err := registriereFach(ctx, db, fach); err != nil {
			return "", fmt.Errorf("fach %q registrieren fehlgeschlagen: %w", fach, err)
		}
	}
	return "", fmt.Errorf("fach %q nach registrierung nicht auffindbar", fach)
}

// registriereFach legt das Fach als Sachgruppe an. Kürzel-Kandidat ist die Bezeichnung
// ohne Leerzeichen (Kürzel bilden Signatur-Vorschläge und dürfen keine tragen, siehe
// systematikRequest.pruefe im api-Paket); kollidiert er mit einem bestehenden Kürzel,
// entscheidet ein Hash-Suffix. Der Konflikt auf lower(bezeichnung) ist dagegen das
// benigne Rennen zweier Importe — DO NOTHING, der Aufrufer liest danach den Gewinner.
func registriereFach(ctx context.Context, db dbSchreiber, fach string) error {
	const einfuegen = `
		INSERT INTO systematik_kategorien (kuerzel, bezeichnung)
		VALUES ($1, $2)
		ON CONFLICT (lower(bezeichnung)) DO NOTHING`

	kuerzel := kuerzeRunen(strings.ReplaceAll(fach, " ", ""), 50)
	_, err := db.Exec(ctx, einfuegen, kuerzel, fach)
	if !istKuerzelKollision(err) {
		return err
	}

	h := sha256.Sum256([]byte(strings.ToLower(fach)))
	suffix := fmt.Sprintf("~%x", h[:4]) // "~" + 8 Hex-Zeichen
	kuerzel = kuerzeRunen(kuerzel, 50-len(suffix)) + suffix
	_, err = db.Exec(ctx, einfuegen, kuerzel, fach)
	return err
}

// istKuerzelKollision erkennt den Unique-Konflikt auf systematik_kategorien.kuerzel.
func istKuerzelKollision(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" &&
		pgErr.ConstraintName == "systematik_kategorien_kuerzel_key"
}

// kuerzeRunen kappt auf max Zeichen (nicht Bytes — VARCHAR zählt Zeichen, und ein
// mittig zerschnittener Umlaut wäre ohnehin kein gültiges Kürzel).
func kuerzeRunen(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
