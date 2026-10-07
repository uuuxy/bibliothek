package repository

import (
	"context"
	"errors"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// Der Versand der Bestellmail an der Bestellung (Migration 161). mail_gescheitert_am trägt
// den Zeitpunkt des letzten gescheiterten Versuchs; NULL heißt, dass kein gescheiterter
// Versand vermerkt ist. Geschrieben wird die Spalte nur über diese Datei.

// ErrBestellmailNichtOffen meldet: Die Bestellung trägt keinen gescheiterten Versand. Ihre
// Mail ging raus, ein anderer Arbeitsplatz sendet sie gerade erneut, oder die Bestellung ist
// älter als der Vermerk.
var ErrBestellmailNichtOffen = errors.New("für diese Bestellung ist kein gescheiterter Versand vermerkt")

// BestellmailAuftrag ist, was der erneute Versand über die Bestellung wissen muss.
type BestellmailAuftrag struct {
	LieferantName string
	// Empfaenger ist die heutige Adresse des Lieferanten. Ist er gelöscht, bleibt es die
	// Adresse, die am Beleg steht.
	Empfaenger   string
	Kundennummer string
	Mittel       string
	Positionen   []BestellmailPosition
}

// BestellmailPosition ist eine bestellte Zeile, wie sie im Anschreiben steht.
type BestellmailPosition struct {
	Titel, Autor, ISBN, Verlag string
	Menge                      int
}

// MerkeBestellmailGescheitert vermerkt an der Bestellung, dass ihre Mail nicht rausging.
// pgx.ErrNoRows, wenn es die Bestellung nicht gibt.
func MerkeBestellmailGescheitert(ctx context.Context, pool db.PgxPoolIface, bestellungID string) error {
	tag, err := pool.Exec(ctx,
		`UPDATE bestellungen_verlauf SET mail_gescheitert_am = now() WHERE id = $1`, bestellungID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// BeanspruchBestellmail nimmt der Bestellung den Vermerk und liefert, was der erneute Versand
// braucht. Nur eine Anfrage bekommt den Auftrag: Ein zweiter Klick und ein zweiter
// Arbeitsplatz finden keinen Vermerk mehr und bekommen ErrBestellmailNichtOffen, die Mail
// geht nicht doppelt an den Lieferanten. Scheitert der Versand danach, setzt
// MerkeBestellmailGescheitert den Vermerk wieder. pgx.ErrNoRows, wenn es die Bestellung
// nicht gibt.
func BeanspruchBestellmail(ctx context.Context, pool db.PgxPoolIface, bestellungID string) (*BestellmailAuftrag, error) {
	var a BestellmailAuftrag
	err := pool.QueryRow(ctx, `
		UPDATE bestellungen_verlauf b
		   SET mail_gescheitert_am = NULL
		  FROM (SELECT v.id, COALESCE(l.email, v.lieferant_email) AS empfaenger
		          FROM bestellungen_verlauf v
		          LEFT JOIN lieferanten l ON l.id = v.lieferant_id
		         WHERE v.id = $1 AND v.mail_gescheitert_am IS NOT NULL
		           FOR UPDATE OF v) offen
		 WHERE b.id = offen.id
		RETURNING b.lieferant_name, offen.empfaenger, COALESCE(b.kundennummer, ''), COALESCE(b.mittel, '')`,
		bestellungID).Scan(&a.LieferantName, &a.Empfaenger, &a.Kundennummer, &a.Mittel)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, bestellmailNichtOffenOderUnbekannt(ctx, pool, bestellungID)
	}
	if err != nil {
		return nil, err
	}
	a.Positionen, err = leseBestellmailPositionen(ctx, pool, bestellungID)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// bestellmailNichtOffenOderUnbekannt unterscheidet die zwei Gründe, aus denen kein Auftrag
// zu haben war: Die Bestellung gibt es nicht, oder sie trägt keinen Vermerk.
func bestellmailNichtOffenOderUnbekannt(ctx context.Context, pool db.PgxPoolIface, bestellungID string) error {
	var vorhanden bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM bestellungen_verlauf WHERE id = $1)`, bestellungID).Scan(&vorhanden); err != nil {
		return err
	}
	if !vorhanden {
		return pgx.ErrNoRows
	}
	return ErrBestellmailNichtOffen
}

// leseBestellmailPositionen liest die Zeilen der Bestellung für das Anschreiben. Titel und
// ISBN stehen an der Position, Autor und Verlag am Titel; ist er inzwischen gelöscht,
// bleiben sie leer.
func leseBestellmailPositionen(ctx context.Context, pool db.PgxPoolIface, bestellungID string) ([]BestellmailPosition, error) {
	rows, err := pool.Query(ctx, `
		SELECT p.titel_name, COALESCE(t.autor, ''), COALESCE(p.isbn, ''), COALESCE(t.verlag, ''), p.menge
		  FROM bestellungen_positionen p
		  LEFT JOIN buecher_titel t ON t.id = p.titel_id
		 WHERE p.bestellung_id = $1
		 ORDER BY p.titel_name`, bestellungID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	positionen := []BestellmailPosition{}
	for rows.Next() {
		var p BestellmailPosition
		if err := rows.Scan(&p.Titel, &p.Autor, &p.ISBN, &p.Verlag, &p.Menge); err != nil {
			return nil, err
		}
		positionen = append(positionen, p)
	}
	return positionen, rows.Err()
}

// MerkeBestellmailVersendet hält an der Bestellung die Adresse fest, an die der erneute
// Versand ging: Am Beleg steht, wohin die Bestellung gegangen ist. pgx.ErrNoRows, wenn es
// die Bestellung nicht gibt.
func MerkeBestellmailVersendet(ctx context.Context, pool db.PgxPoolIface, bestellungID, empfaenger string) error {
	tag, err := pool.Exec(ctx, `
		UPDATE bestellungen_verlauf SET lieferant_email = $2, mail_gescheitert_am = NULL WHERE id = $1`,
		bestellungID, empfaenger)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
