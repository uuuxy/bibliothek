package repository

import (
	"context"
	"time"

	"bibliothek/db"
)

// Datenzugriff für die Detailansicht EINER Bestellung.
//
// Die Liste (/api/bestellhistorie) ist auf die neuesten 200 gedeckelt und trägt pro
// Position nur Titel, ISBN, Menge und Preis — genug für eine Tabellenzeile, zu wenig,
// um eine Lieferung wiederzuerkennen. Diese Schicht liefert deshalb zusätzlich Autor,
// Verlag, Cover und vor allem die tatsächlich angelegten Exemplare mit ihren Barcodes.
//
// Dass das geht, ist neuer als der Kommentar in bestellhistorie_handler.go behauptet:
// Seit Migration 063 trägt jedes beim Bestellen angelegte Exemplar seine bestellung_id
// (order_service.go schreibt den Kopf bewusst VOR den Exemplaren, damit die ID existiert).
// Für Altbestand aus dem Littera-Import ist die Spalte leer — die Ansicht sagt das dann
// auch, statt eine leere Liste als "keine Exemplare" auszugeben.

// EtikettOffenBedingung ist die EINZIGE Definition von "Etikett steht noch aus".
// Der Zähler im Bestellwesen, die Nachdruck-Liste im Druck-Center und die Positionen
// dieser Detailansicht teilen sie sich — sonst nennt der Verweis irgendwann eine andere
// Zahl, als die Liste hat, die er öffnet, und keiner der beiden Werte ist mehr zu trauen.
// api/etiketten_offen.go übernimmt den Wert von hier.
//
// Ausgesonderte Exemplare stehen nicht mehr im Regal; für sie ein Etikett zu drucken
// wäre immer falsch.
//
// Das Alias `e` ist Teil der Bedingung: Jede Abfrage, die sie einsetzt, muss
// buecher_exemplare als e führen.
const EtikettOffenBedingung = `e.etikett_gedruckt = false AND e.ist_ausgesondert = false`

// BestellungKopf sind die Stammdaten einer Bestellung.
type BestellungKopf struct {
	ID              string     `json:"id"`
	LieferantName   string     `json:"lieferant_name"`
	LieferantEmail  string     `json:"lieferant_email"`
	Kundennummer    string     `json:"kundennummer"`
	Bestelldatum    time.Time  `json:"bestelldatum"`
	Gesamtbetrag    float64    `json:"gesamtbetrag"`
	AnzahlExemplare int        `json:"anzahl_exemplare"`
	MitBestaetigung bool       `json:"mit_bestaetigung"`
	BestaetigtAm    *time.Time `json:"bestaetigt_am,omitempty"`
	BestaetigtDurch *string    `json:"bestaetigt_durch,omitempty"`

	// EtikettenGroesse und LinkAktiv tragen denselben Namen wie in der Bestellhistorie,
	// damit die Detailansicht BestellStatusBlock unverändert weiterverwenden kann. Der
	// Bestätigungs-Ablauf gehört an EINE Stelle — ein nachgebauter zweiter Block wäre
	// genau die Parallelwelt, die diese Seite gerade beseitigt hat.
	EtikettenGroesse *string `json:"etiketten_groesse,omitempty"`
	LinkAktiv        bool    `json:"link_aktiv"`
}

// BestellPositionDetail ist eine bestellte Zeile samt Angaben aus dem Titelsatz.
//
// Autor, Verlag und CoverURL sind in buecher_titel nullbar und werden per COALESCE zu
// leeren Zeichenketten — ein nullbarer Wert in einem nicht-nullbaren Go-Typ ist in
// diesem Projekt eine wiederkehrende 500er-Quelle ("cannot scan NULL"). Leer heisst
// hier "nicht hinterlegt", und die Oberfläche zeigt dann nichts statt eines Fehlers.
type BestellPositionDetail struct {
	TitelID     string  `json:"titel_id,omitempty"`
	TitelName   string  `json:"titel_name"`
	ISBN        string  `json:"isbn"`
	Autor       string  `json:"autor"`
	Verlag      string  `json:"verlag"`
	CoverURL    string  `json:"cover_url"`
	Menge       int     `json:"menge"`
	Einzelpreis float64 `json:"einzelpreis"`
	Gesamtpreis float64 `json:"gesamtpreis"`

	// EtikettenOffen zählt die Exemplare DIESES TITELS ohne Etikett — nicht die dieser
	// Lieferung. Bewusst dieselbe Zahl wie in der Bestellhistorie: Sie trägt den Verweis
	// in die Nachdruck-Liste, und die filtert nach Titel. Eine hier genauere,
	// lieferungsbezogene Zahl führte in eine Liste, die mehr Zeilen zeigt als der Knopf
	// versprochen hat. Wie viele Bücher DIESER Lieferung noch ein Etikett brauchen, steht
	// stattdessen an den Exemplaren selbst (BestellExemplar.EtikettGedruckt).
	EtikettenOffen int `json:"etiketten_offen"`
}

// BestellExemplar ist ein einzelnes, aus dieser Bestellung entstandenes Buch.
type BestellExemplar struct {
	BarcodeID       string `json:"barcode_id"`
	TitelID         string `json:"titel_id"`
	TitelName       string `json:"titel_name"`
	EtikettGedruckt bool   `json:"etikett_gedruckt"`
	IstAusgesondert bool   `json:"ist_ausgesondert"`
}

// BestellungDetail bündelt, was die Detailansicht braucht.
type BestellungDetail struct {
	BestellungKopf
	Positionen []BestellPositionDetail `json:"positionen"`
	Exemplare  []BestellExemplar       `json:"exemplare"`
}

// BestelldetailRepository kapselt die Abfragen der Bestell-Detailansicht.
type BestelldetailRepository interface {
	GetBestellungDetail(ctx context.Context, bestellungID string) (*BestellungDetail, error)
}

type pgBestelldetailRepository struct {
	db db.PgxPoolIface
}

// NewBestelldetailRepository erzeugt das Repository für die Bestell-Detailansicht.
func NewBestelldetailRepository(pool db.PgxPoolIface) BestelldetailRepository {
	return &pgBestelldetailRepository{db: pool}
}

func (r *pgBestelldetailRepository) GetBestellungDetail(ctx context.Context, bestellungID string) (*BestellungDetail, error) {
	kopf, err := r.ladeKopf(ctx, bestellungID)
	if err != nil {
		return nil, err
	}

	positionen, err := r.ladePositionen(ctx, bestellungID)
	if err != nil {
		return nil, err
	}

	exemplare, err := r.ladeExemplare(ctx, bestellungID)
	if err != nil {
		return nil, err
	}

	return &BestellungDetail{BestellungKopf: *kopf, Positionen: positionen, Exemplare: exemplare}, nil
}

// ladeKopf liefert pgx.ErrNoRows, wenn es die Bestellung nicht gibt — der Handler macht
// daraus einen 404 statt einer leeren Seite, die aussieht wie eine leere Bestellung.
func (r *pgBestelldetailRepository) ladeKopf(ctx context.Context, bestellungID string) (*BestellungKopf, error) {
	var k BestellungKopf
	err := r.db.QueryRow(ctx, `
		SELECT id, lieferant_name, lieferant_email, kundennummer, bestelldatum,
		       gesamtbetrag, anzahl_exemplare,
		       bestaetigungs_token_hash IS NOT NULL,
		       bestaetigt_am, bestaetigt_durch, etiketten_groesse,
		       (bestaetigungs_token_hash IS NOT NULL
		        AND (token_gueltig_bis IS NULL OR token_gueltig_bis > now()))
		FROM bestellungen_verlauf
		WHERE id = $1
	`, bestellungID).Scan(&k.ID, &k.LieferantName, &k.LieferantEmail, &k.Kundennummer,
		&k.Bestelldatum, &k.Gesamtbetrag, &k.AnzahlExemplare, &k.MitBestaetigung,
		&k.BestaetigtAm, &k.BestaetigtDurch, &k.EtikettenGroesse, &k.LinkAktiv)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// ladePositionen verbindet die Bestellzeile mit dem Titelsatz.
//
// LEFT JOIN mit Absicht: bestellungen_positionen.titel_id ist ON DELETE SET NULL, eine
// Bestellung bleibt also als Beleg bestehen, wenn der Titel später aus dem Katalog
// verschwindet. Ein INNER JOIN liesse solche Positionen wortlos aus dem Beleg fallen.
func (r *pgBestelldetailRepository) ladePositionen(ctx context.Context, bestellungID string) ([]BestellPositionDetail, error) {
	rows, err := r.db.Query(ctx, `
		SELECT coalesce(p.titel_id::text, ''), p.titel_name, p.isbn, p.menge, p.einzelpreis,
		       coalesce(t.autor, ''), coalesce(t.verlag, ''), coalesce(t.cover_url, ''),
		       (SELECT count(*) FROM buecher_exemplare e
		         WHERE e.titel_id = p.titel_id AND `+EtikettOffenBedingung+`)
		FROM bestellungen_positionen p
		LEFT JOIN buecher_titel t ON t.id = p.titel_id
		WHERE p.bestellung_id = $1
		ORDER BY p.titel_name
	`, bestellungID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positionen := make([]BestellPositionDetail, 0)
	for rows.Next() {
		var p BestellPositionDetail
		if err := rows.Scan(&p.TitelID, &p.TitelName, &p.ISBN, &p.Menge, &p.Einzelpreis,
			&p.Autor, &p.Verlag, &p.CoverURL, &p.EtikettenOffen); err != nil {
			return nil, err
		}
		p.Gesamtpreis = float64(p.Menge) * p.Einzelpreis
		positionen = append(positionen, p)
	}
	return positionen, rows.Err()
}

// ladeExemplare liefert die Bücher, die AUS DIESER Bestellung entstanden sind.
//
// Sortiert nach Titel und Barcode: Der Barcode ist die Nummer, die auf dem Buch klebt,
// und wer sie im Regal sucht, sucht sie in aufsteigender Reihenfolge.
func (r *pgBestelldetailRepository) ladeExemplare(ctx context.Context, bestellungID string) ([]BestellExemplar, error) {
	rows, err := r.db.Query(ctx, `
		SELECT e.barcode_id, e.titel_id::text, t.titel, e.etikett_gedruckt, e.ist_ausgesondert
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		WHERE e.bestellung_id = $1
		ORDER BY t.titel, e.barcode_id
	`, bestellungID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	exemplare := make([]BestellExemplar, 0)
	for rows.Next() {
		var e BestellExemplar
		if err := rows.Scan(&e.BarcodeID, &e.TitelID, &e.TitelName, &e.EtikettGedruckt,
			&e.IstAusgesondert); err != nil {
			return nil, err
		}
		exemplare = append(exemplare, e)
	}
	return exemplare, rows.Err()
}
