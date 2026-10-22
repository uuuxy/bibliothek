package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ExemplarNeu beschreibt die Datenstruktur für das Einfügen neuer Buchexemplare im Bulk-Verfahren.
type ExemplarNeu struct {
	// TitelID verweist auf die Metadaten des Buchtitels.
	TitelID string
	// BarcodeID ist der eindeutige Barcode des neuen Exemplars.
	BarcodeID string
	// ZustandNotiz dokumentiert den Initialzustand des Buchs (optional).
	ZustandNotiz string
	// IstAusleihbar gibt an, ob das Buch direkt verliehen werden darf.
	IstAusleihbar bool
	// EtikettGedruckt speichert, ob das Barcode-Etikett bereits gedruckt wurde.
	EtikettGedruckt bool
	// Einkaufspreis speichert den Netto-Anschaffungspreis des Exemplars.
	Einkaufspreis float64
	// Bestellstatus setzt neue Exemplare aus dem Bestellwesen in den Bestellweg ('bestellt'
	// oder 'im_zulauf'); leer heißt kein Bestellvorgang (NULL). Nur diese Spalte entscheidet
	// über die Sichtbarkeit im OPAC, die zustand_notiz ist Text für Menschen.
	Bestellstatus string
	// BestellungID verweist auf die Bestellung, aus der das Exemplar entstanden ist.
	// Leer bei Handanlage und Import — dann wird NULL geschrieben. Der Verweis trägt die
	// Etikettenseite des Lieferanten-Links: Sie muss genau die Exemplare dieser Lieferung
	// drucken, nicht alle Exemplare des Titels.
	BestellungID string
}

// Exemplar ist das einzelne Stück eines Titels (Tabelle `buecher_exemplare`), mit den Angaben
// seines Titels, die Theke und Akte daneben zeigen.
type Exemplar struct {
	// ID ist die UUID des konkreten Exemplars.
	ID string `json:"id"`
	// TitelID verweist auf die Metadaten des Buchtitels.
	TitelID string `json:"titel_id"`
	// BarcodeID ist die physische Inventar- oder Barcode-Nummer des Exemplars.
	BarcodeID string `json:"barcode_id"`
	// ZustandNotiz dokumentiert eventuelle Beschädigungen (z. B. "Wasserschaden") oder Reservierungen.
	ZustandNotiz string `json:"zustand_notiz"`
	// ErworbenAm ist das Kaufdatum oder Zugangsdatum des Exemplars.
	ErworbenAm time.Time `json:"erworben_am"`
	// IstAusleihbar gibt an, ob das Buch verliehen werden darf.
	IstAusleihbar bool `json:"ist_ausleihbar"`
	// IstAusgesondert markiert verloren gegangene, beschädigte oder ausgemusterte Bücher.
	IstAusgesondert bool `json:"ist_ausgesondert"`
	// ErstelltAm ist das System-Erfassungsdatum.
	ErstelltAm time.Time `json:"erstellt_am"`
	// AktualisiertAm ist das letzte Änderungsdatum.
	AktualisiertAm time.Time `json:"aktualisiert_am"`

	// Gejointe Felder aus der Tabelle buecher_titel:

	// Titel ist der Haupttitel des Werks.
	Titel string `json:"titel"`
	// Autor ist der Autor des Werks.
	Autor string `json:"autor"`
	// Verlag ist der Verlag des Werks.
	Verlag string `json:"verlag"`
	// ISBN ist die ISBN des Werks.
	ISBN string `json:"isbn"`
	// CoverURL ist das Coverbild des Werks.
	CoverURL string `json:"cover_url,omitempty"`
	// Medientyp ist die Medienart (z. B. "Buch").
	Medientyp string `json:"medientyp,omitempty"`
	// Signatur speichert die Bibliothekssignatur.
	Signatur string `json:"signatur,omitempty"`
	// Mehrjahresband und JahrgangBis kommen vom Titel: Bleibt das Buch über
	// mehrere Schuljahre beim Kind, rechnet die Fristregel bis zum Stichtag des Schuljahres,
	// in dem das Kind JahrgangBis beendet (internal/service/frist.go).
	Mehrjahresband bool `json:"mehrjahresband"`
	JahrgangBis    int  `json:"jahrgang_bis"`
	// IstLernmittel: Schulbuch der Lernmittelfreiheit (buecher_titel.ist_lernmittel). Es trägt
	// die Frist des Schuljahres statt einer in Tagen und zählt nicht ins Ausleihlimit.
	IstLernmittel bool `json:"ist_lernmittel"`
	// ErweiterteEigenschaften speichert zusätzliche dynamische Metadaten als JSON-Map.
	ErweiterteEigenschaften map[string]any `json:"erweiterteEigenschaften,omitempty"`
}

// leseExemplar ist eine Hilfsfunktion zum Einlesen einer Zeile in ein Exemplar-Objekt.
func leseExemplar(row scanner) (*Exemplar, error) {
	var bc Exemplar
	err := row.Scan(
		&bc.ID, &bc.TitelID, &bc.BarcodeID, &bc.ZustandNotiz, &bc.ErworbenAm, &bc.IstAusleihbar, &bc.IstAusgesondert, &bc.ErstelltAm, &bc.AktualisiertAm,
		&bc.Titel, &bc.Autor, &bc.Verlag, &bc.ISBN, &bc.CoverURL, &bc.Medientyp, &bc.Signatur, &bc.Mehrjahresband, &bc.JahrgangBis, &bc.IstLernmittel, &bc.ErweiterteEigenschaften,
	)
	if err != nil {
		return nil, err
	}
	return &bc, nil
}

// FindeExemplar löst einen Buch-Barcode auf: das Exemplar mit den Angaben seines Titels; nil,
// wenn keines die Nummer trägt. Ein altes LMF-Etikett (lmf_barcode) führt zum selben Exemplar;
// als eigene Nummer trägt keines eine LMF-Nummer (chk_exemplar_barcode_keine_lmf_nummer), ein Scan
// trifft also höchstens eins. Wer weiterarbeitet, nimmt BarcodeID des Treffers, nicht den
// gescannten Text.
func (a *KatalogAbfragen) FindeExemplar(ctx context.Context, barcode string) (*Exemplar, error) {
	query := `
		SELECT 
			e.id, e.titel_id, coalesce(e.barcode_id, ''), coalesce(e.zustand_notiz, ''), e.erworben_am, coalesce(e.ist_ausleihbar, false), coalesce(e.ist_ausgesondert, false), e.erstellt_am, e.aktualisiert_am,
			coalesce(t.titel, ''), coalesce(t.autor, ''), coalesce(t.verlag, ''), coalesce(t.isbn, ''), coalesce(t.cover_url, ''), coalesce(t.medientyp, ''), coalesce(t.signatur, ''), t.mehrjahresband, coalesce(t.jahrgang_bis, 0), t.ist_lernmittel, coalesce(t.erweiterte_eigenschaften, '{}'::jsonb)
		FROM buecher_exemplare e
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE e.barcode_id = $1 OR e.lmf_barcode = $1
		LIMIT 1
	`
	bc, err := leseExemplar(a.db.QueryRow(ctx, query, barcode))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return bc, nil
}
