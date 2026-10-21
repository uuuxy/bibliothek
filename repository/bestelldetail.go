package repository

import (
	"context"
	"time"

	"bibliothek/db"
)

// EtikettOffenBedingung ist die eine Bestimmung von „Etikett steht noch aus". Der Zähler im
// Bestellwesen, die Liste zum Nachdrucken im Druck-Center und die Positionen der Detailansicht
// teilen sie; sonst nennte der Verweis eine andere Zahl, als die Liste hat, die er öffnet.
// api/druck/etiketten_offen.go übernimmt den Wert von hier.
//
// Ausgesonderte Exemplare stehen nicht mehr im Regal; für sie wäre ein Etikett immer falsch.
// Der Alias e gehört zur Bedingung: Jede Abfrage, die sie einsetzt, führt buecher_exemplare als
// e.
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

	// EtikettenGroesse und LinkAktiv heißen wie in der Bestellhistorie, damit die Detailansicht
	// denselben Baustein der Oberfläche nutzen kann (BestellStatusBlock): Der Ablauf der
	// Bestätigung steht an einer Stelle.
	EtikettenGroesse *string `json:"etiketten_groesse,omitempty"`
	LinkAktiv        bool    `json:"link_aktiv"`

	// Mittel ist der Topf der Bestellung ('land' oder 'schultraeger'). Leer bei einer alten
	// Bestellung ohne eindeutige Zuordnung; die Oberfläche zeigt dann „ohne Zuordnung" und
	// keinen geratenen Topf.
	Mittel string `json:"mittel"`

	// MailGescheitertAm: Zeitpunkt des letzten gescheiterten Versands der Bestellmail,
	// solange die Bestellung weder bestätigt noch geliefert ist (sqlBestellmailOffen). nil =
	// kein offener gescheiterter Versand.
	MailGescheitertAm *time.Time `json:"mail_gescheitert_am,omitempty"`

	// LinkNichtErhalten: Der Bestätigungs-Link stand in einer Mail, die nicht rausging
	// (sqlBestelllinkNichtErhalten). Der Bestätigungsblock sagt dann nicht, er sei unterwegs.
	LinkNichtErhalten bool `json:"link_nicht_erhalten"`
}

// BestellPositionDetail ist eine bestellte Zeile mit den Angaben aus dem Titel. Autor, Verlag
// und CoverURL können in buecher_titel leer sein und kommen über COALESCE als leerer Text: NULL
// ließe sich nicht in einen Text lesen. Leer heißt „nicht hinterlegt".
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

	// EtikettenOffen zählt die Exemplare dieses Titels ohne Etikett, nicht die dieser
	// Lieferung, wie in der Bestellhistorie: Die Zahl trägt den Verweis in die Liste zum
	// Nachdrucken, und die filtert nach Titel. Eine Zahl je Lieferung führte in eine Liste mit
	// mehr Zeilen, als der Knopf nennt. Wie viele Bücher dieser Lieferung noch ein Etikett
	// brauchen, steht an den Exemplaren (BestellExemplar.EtikettGedruckt).
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

// BestelldetailAbfragen liest, was die Detailansicht einer Bestellung zeigt. Die Liste der
// Bestellhistorie trägt je Position nur Titel, ISBN, Menge und Preis; hier kommen Autor,
// Verlag, Cover und die angelegten Exemplare mit ihren Barcodes dazu, an denen sich eine
// Lieferung wiedererkennen lässt. Jedes beim Bestellen angelegte Exemplar trägt seine
// bestellung_id; am Altbestand aus Littera ist die Spalte leer.
type BestelldetailAbfragen struct {
	db db.Pool
}

// NewBestelldetailAbfragen liefert die Abfragen der Detailansicht an Postgres.
func NewBestelldetailAbfragen(pool db.Pool) *BestelldetailAbfragen {
	return &BestelldetailAbfragen{db: pool}
}

// Lade lädt Kopf, Positionen und Exemplare einer Bestellung; pgx.ErrNoRows, wenn es sie nicht
// gibt.
func (a *BestelldetailAbfragen) Lade(ctx context.Context, bestellungID string) (*BestellungDetail, error) {
	kopf, err := a.ladeKopf(ctx, bestellungID)
	if err != nil {
		return nil, err
	}

	positionen, err := a.ladePositionen(ctx, bestellungID)
	if err != nil {
		return nil, err
	}

	exemplare, err := a.ladeExemplare(ctx, bestellungID)
	if err != nil {
		return nil, err
	}

	return &BestellungDetail{BestellungKopf: *kopf, Positionen: positionen, Exemplare: exemplare}, nil
}

// ladeKopf liefert pgx.ErrNoRows, wenn es die Bestellung nicht gibt; die Tür macht daraus 404
// statt einer Seite, die wie eine leere Bestellung aussieht.
func (a *BestelldetailAbfragen) ladeKopf(ctx context.Context, bestellungID string) (*BestellungKopf, error) {
	var k BestellungKopf
	err := a.db.QueryRow(ctx, `
		SELECT id, lieferant_name, lieferant_email, kundennummer, bestelldatum,
		       gesamtbetrag, anzahl_exemplare,
		       `+sqlBestellungMitBestaetigung+`,
		       bestaetigt_am, bestaetigt_durch, etiketten_groesse,
		       (bestaetigungs_token_hash IS NOT NULL
		        AND (token_gueltig_bis IS NULL OR token_gueltig_bis > now())),
		       coalesce(mittel, ''), `+sqlBestellmailOffen+`, `+sqlBestelllinkNichtErhalten+`
		FROM bestellungen_verlauf b
		WHERE id = $1
	`, bestellungID).Scan(&k.ID, &k.LieferantName, &k.LieferantEmail, &k.Kundennummer,
		&k.Bestelldatum, &k.Gesamtbetrag, &k.AnzahlExemplare, &k.MitBestaetigung,
		&k.BestaetigtAm, &k.BestaetigtDurch, &k.EtikettenGroesse, &k.LinkAktiv, &k.Mittel,
		&k.MailGescheitertAm, &k.LinkNichtErhalten)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// ladePositionen verbindet die Bestellzeile mit dem Titel. Mit LEFT JOIN:
// bestellungen_positionen.titel_id steht auf ON DELETE SET NULL, die Bestellung bleibt als
// Beleg, wenn der Titel aus dem Katalog verschwindet, und mit INNER JOIN fielen solche
// Positionen aus dem Beleg.
func (a *BestelldetailAbfragen) ladePositionen(ctx context.Context, bestellungID string) ([]BestellPositionDetail, error) {
	rows, err := a.db.Query(ctx, `
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

// ladeExemplare liefert die Bücher, die aus dieser Bestellung entstanden sind, sortiert nach
// Titel und Barcode: Der Barcode klebt auf dem Buch, und wer es im Regal sucht, sucht in
// aufsteigender Reihenfolge.
func (a *BestelldetailAbfragen) ladeExemplare(ctx context.Context, bestellungID string) ([]BestellExemplar, error) {
	rows, err := a.db.Query(ctx, `
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
