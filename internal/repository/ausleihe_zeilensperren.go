package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Die Zeilensperren einer Buchung an der Theke. Scan und Nachbuchen nehmen sie in derselben
// Reihenfolge: den Leser, die offene Ausleihe des Exemplars (AusleihAbfragen.SperreOffene), das
// Exemplar. In verschiedener Reihenfolge könnten zwei Buchungen aufeinander warten, bis die
// Datenbank eine abbricht. Die Funktionen nehmen eine Transaktion und keinen Pool: Am Pool
// endete die Sperre mit der Anweisung.

// SperreLeserzeile sperrt die Zeile des Lesers bis zum Ende der Transaktion. Zwei Buchungen
// für denselben Leser laufen so nacheinander, und das Ausleihlimit zählt keine Ausleihe, die
// eine andere Transaktion gerade schreibt.
func SperreLeserzeile(ctx context.Context, tx pgx.Tx, leserID string) error {
	_, err := tx.Exec(ctx, "SELECT id FROM leser WHERE id = $1 FOR UPDATE", leserID)
	return err
}

// ExemplarStand ist der Stand eines Exemplars, den Scan und Nachbuchen unter der Sperre lesen.
type ExemplarStand struct {
	Ausleihbar   bool
	Ausgesondert bool
	// LetzteBewegung fehlt an einem Exemplar, das sich noch nie bewegt hat.
	LetzteBewegung *time.Time
}

// SperreExemplarzeile sperrt die Zeile des Exemplars bis zum Ende der Transaktion und liefert
// seinen Stand.
func SperreExemplarzeile(ctx context.Context, tx pgx.Tx, exemplarID string) (ExemplarStand, error) {
	var s ExemplarStand
	err := tx.QueryRow(ctx, `SELECT ist_ausleihbar, ist_ausgesondert, letzte_bewegung_am
		FROM buecher_exemplare WHERE id = $1 FOR UPDATE`, exemplarID).Scan(&s.Ausleihbar, &s.Ausgesondert, &s.LetzteBewegung)
	return s, err
}

// GeraetStand ist der Stand eines Geräts, den Ausleihe und Schadensmeldung unter der Sperre lesen.
type GeraetStand struct {
	Ausleihbar   bool
	Ausgesondert bool
}

// SperreGeraetezeile sperrt die Zeile des Geräts bis zum Ende der Transaktion und liefert seinen
// Stand, wie SperreExemplarzeile beim Buch, in derselben Reihenfolge nach der Ausleihe.
func SperreGeraetezeile(ctx context.Context, tx pgx.Tx, geraetID string) (GeraetStand, error) {
	var s GeraetStand
	err := tx.QueryRow(ctx, `SELECT ist_ausleihbar, ist_ausgesondert FROM geraete WHERE id = $1 FOR UPDATE`,
		geraetID).Scan(&s.Ausleihbar, &s.Ausgesondert)
	return s, err
}
