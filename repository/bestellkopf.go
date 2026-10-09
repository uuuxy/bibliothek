package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// BestellkopfNeu sind die Werte eines neuen Bestellkopfs. Name, Adresse und Kundennummer des
// Lieferanten stehen als Abschrift an der Bestellung. TokenHash ist leer, wenn die Bestellung
// keinen Bestätigungs-Link bekommt; IdempotenzSchluessel ist nil ohne Schlüssel.
type BestellkopfNeu struct {
	LieferantID          string
	LieferantName        string
	LieferantEmail       string
	Kundennummer         string
	Gesamtbetrag         float64
	Menge                int
	TokenHash            string
	LinkTage             int
	IdempotenzSchluessel *string
	Mittel               string
}

// LegeBestellkopfAn schreibt den Bestellkopf in der Transaktion der Bestellung und liefert seine
// Kennung und den Ablauf des Bestätigungs-Links. Gibt es zum Idempotenz-Schlüssel schon eine
// Bestellung, entsteht keine Zeile: pgx.ErrNoRows. Ein leerer TokenHash wird NULL, damit der
// Teil-Index zwei Bestellungen ohne Link nicht als Dublette ablehnt.
func LegeBestellkopfAn(ctx context.Context, tx pgx.Tx, k BestellkopfNeu) (id string, linkGueltigBis *time.Time, err error) {
	err = tx.QueryRow(ctx, `
		INSERT INTO bestellungen_verlauf
			(lieferant_id, lieferant_name, lieferant_email, kundennummer, gesamtbetrag, anzahl_exemplare,
			 bestaetigungs_token_hash, token_gueltig_bis, idempotenz_schluessel, mittel)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''),
		        CASE WHEN $7 = '' THEN NULL ELSE now() + make_interval(days => $8) END,
		        $9, $10)
		ON CONFLICT (idempotenz_schluessel) WHERE idempotenz_schluessel IS NOT NULL DO NOTHING
		RETURNING id, token_gueltig_bis`,
		k.LieferantID, k.LieferantName, k.LieferantEmail, k.Kundennummer,
		k.Gesamtbetrag, k.Menge, k.TokenHash, k.LinkTage,
		k.IdempotenzSchluessel, k.Mittel,
	).Scan(&id, &linkGueltigBis)
	return id, linkGueltigBis, err
}

// BestellungZuIdempotenzSchluessel liest die Bestellung, die mit diesem Schlüssel schon
// angelegt wurde: Kennung, Name des Lieferanten und Zahl der Exemplare.
func BestellungZuIdempotenzSchluessel(ctx context.Context, db DBQueryer, schluessel string) (id, lieferantName string, anzahlExemplare int, err error) {
	err = db.QueryRow(ctx, `
		SELECT id, lieferant_name, anzahl_exemplare
		FROM bestellungen_verlauf WHERE idempotenz_schluessel = $1`,
		schluessel).Scan(&id, &lieferantName, &anzahlExemplare)
	return id, lieferantName, anzahlExemplare, err
}

// BestellpositionNeu ist eine Position einer neuen Bestellung. TitelName und ISBN stehen als
// Abschrift an der Position, damit die Bestellung lesbar bleibt, wenn der Titel gelöscht ist.
// MitVorabBarcode hält fest, ob die Position auf dem Barcodebogen der Bestellmail stand: Ohne
// die Angabe druckte die Etikettenseite hinter dem Lieferanten-Link auch Exemplare, die ohne
// Vorab-Etikett bestellt wurden.
type BestellpositionNeu struct {
	TitelID         string
	TitelName       string
	ISBN            string
	Menge           int
	Einzelpreis     float64
	MitVorabBarcode bool
}

// SchreibeBestellpositionen schreibt die Positionen eines Bestellkopfs in einem Zug, in der
// Transaktion der Bestellung.
func SchreibeBestellpositionen(ctx context.Context, tx pgx.Tx, bestellungID string, positionen []BestellpositionNeu) error {
	if len(positionen) == 0 {
		return nil
	}
	zeilen := make([][]any, 0, len(positionen))
	for _, p := range positionen {
		zeilen = append(zeilen, []any{
			bestellungID, p.TitelID, p.TitelName, p.ISBN, p.Menge, p.Einzelpreis, p.MitVorabBarcode,
		})
	}
	_, err := tx.CopyFrom(ctx,
		pgx.Identifier{"bestellungen_positionen"},
		[]string{"bestellung_id", "titel_id", "titel_name", "isbn", "menge", "einzelpreis", "mit_vorab_barcode"},
		pgx.CopyFromRows(zeilen),
	)
	return err
}
