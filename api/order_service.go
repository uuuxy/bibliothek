package api

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// OrderService verarbeitet die Geschäftslogik zum Erstellen und Verarbeiten von Bestellungen.
type OrderService struct {
	db           *db.Database
	bookRepo     repository.BookRepository
	supplierRepo repository.SupplierRepository
}

// NewOrderService erstellt eine neue OrderService-Instanz.
func NewOrderService(database *db.Database, bookRepo repository.BookRepository) *OrderService {
	return &OrderService{
		db:           database,
		bookRepo:     bookRepo,
		supplierRepo: repository.NewSupplierRepository(database.Pool),
	}
}

// OrderResult enthält das Ergebnis einer verarbeiteten Bestellung, einschließlich der generierten Barcodes.
type OrderResult struct {
	SupplierName   string
	SupplierEmail  string
	CustomerNumber string
	Labels         []BarcodeLabelDetail
	SummaryItems   []OrderedItem
	TotalAllocated int
	// IstHauptlieferant: siehe repository.Supplier — steuert, ob DispatchOrderEmail
	// zusätzlich das große Lernmittel-Etikett anhängt.
	IstHauptlieferant bool
	// BestellungID der soeben geschriebenen Bestellung.
	BestellungID string
	// BestaetigungsToken ist der KLARTEXT-Token für den Link in der Bestellmail. In der
	// Datenbank liegt nur sein Hash; wer ihn hier nicht mitnimmt, kann ihn nie wieder
	// erfahren. Leer, wenn der Lieferant keinen Bestätigungsschritt hat.
	BestaetigungsToken string
}

type bestellungPosition struct {
	titelID   string
	titelName string
	isbn      string
	menge     int
	preis     float64
	// mitVorabBarcode hält fest, ob diese Position auf dem Barcodebogen der Bestellmail
	// stand. Ohne diese Angabe könnte die Etikettenseite des Lieferanten-Links nicht
	// dieselbe Auswahl drucken wie der Mailanhang — sie würde auch Exemplare mitdrucken,
	// die bewusst ohne Vorab-Etikett bestellt wurden.
	mitVorabBarcode bool
}

// bestellItemResult bündelt die aus einer einzelnen Bestellposition erzeugten Daten.
type bestellItemResult struct {
	summary  OrderedItem
	position bestellungPosition
	copies   []repository.BookCopyInsert
	labels   []BarcodeLabelDetail
	betrag   float64
}

// ProcessOrder verarbeitet eine eingehende SubmitOrderRequest innerhalb einer Transaktion, generiert Barcodes und gibt das OrderResult zurück.
func (s *OrderService) ProcessOrder(ctx context.Context, req SubmitOrderRequest) (*OrderResult, error) {
	// 1. Lieferantendetails abrufen
	supplier, err := s.supplierRepo.GetSupplierByID(ctx, req.SupplierID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("supplier not found")
		}
		return nil, err
	}

	// Der Bestätigungs-Link entsteht NUR für den Hauptlieferanten, der selbst etikettiert
	// und bestätigt. Alle anderen bekämen eine Seite, auf der es nichts zu tun gibt —
	// für sie bleibt es bei der reinen Bestellmail.
	var token, tokenHash string
	if supplier.IstHauptlieferant {
		token, tokenHash, err = neuerBestaetigungsToken()
		if err != nil {
			return nil, fmt.Errorf("bestaetigungs-token: %w", err)
		}
	}

	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	labels := make([]BarcodeLabelDetail, 0)
	orderSummaryItems := make([]OrderedItem, 0)
	var totalAllocated int
	var gesamtbetrag float64

	var copyInserts []repository.BookCopyInsert
	var positionen []bestellungPosition

	for _, item := range req.Items {
		res, err := s.verarbeiteBestellItem(ctx, tx, item, supplier)
		if err != nil {
			return nil, err
		}
		orderSummaryItems = append(orderSummaryItems, res.summary)
		positionen = append(positionen, res.position)
		copyInserts = append(copyInserts, res.copies...)
		labels = append(labels, res.labels...)
		gesamtbetrag += res.betrag
		totalAllocated += res.position.menge
	}

	// REIHENFOLGE: Bestellkopf VOR den Exemplaren. Die Exemplare tragen seit Migration 063
	// ihre bestellung_id, und die gibt es erst, wenn der Kopf geschrieben ist. Der Tausch
	// ist unbedenklich, weil keine der beiden Einfügungen die andere liest — die Barcodes
	// sind oben in der Schleife bereits reserviert, und der Kopf zählt nur die dort
	// errechneten Summen.
	bestellungID, err := s.insertBestellverlauf(ctx, tx, req, supplier, gesamtbetrag, totalAllocated, tokenHash)
	if err != nil {
		return nil, err
	}
	for i := range copyInserts {
		copyInserts[i].BestellungID = bestellungID
	}

	if err := s.bookRepo.BulkInsertCopiesTx(ctx, tx, copyInserts); err != nil {
		return nil, fmt.Errorf("bulk insert error: %w", err)
	}

	if err := s.insertBestellpositionen(ctx, tx, bestellungID, positionen); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &OrderResult{
		SupplierName:       supplier.Name,
		SupplierEmail:      supplier.Email,
		CustomerNumber:     supplier.Kundennummer,
		Labels:             labels,
		SummaryItems:       orderSummaryItems,
		TotalAllocated:     totalAllocated,
		IstHauptlieferant:  supplier.IstHauptlieferant,
		BestellungID:       bestellungID,
		BestaetigungsToken: token,
	}, nil
}

// verarbeiteBestellItem validiert eine Bestellposition, lädt den Titel, reserviert die
// Barcodes und erzeugt die Exemplar-Datensätze samt (optionalen) Etiketten.
func (s *OrderService) verarbeiteBestellItem(ctx context.Context, tx pgx.Tx, item OrderItemRequest, supplier *repository.Supplier) (*bestellItemResult, error) {
	supplierName := supplier.Name

	if item.Menge <= 0 || item.Menge > 200 {
		return nil, fmt.Errorf("invalid quantity %d for title %s", item.Menge, item.TitelID)
	}

	title, err := s.bookRepo.GetTitleByIDTx(ctx, tx, item.TitelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("book title %s not found", item.TitelID)
		}
		return nil, err
	}

	res := &bestellItemResult{
		summary: OrderedItem{
			Titel:  title.Titel,
			Autor:  title.Autor,
			ISBN:   title.ISBN,
			Verlag: title.Verlag,
			Menge:  item.Menge,
		},
		position: bestellungPosition{
			titelID:         item.TitelID,
			titelName:       title.Titel,
			isbn:            title.ISBN,
			menge:           item.Menge,
			preis:           item.Preis,
			mitVorabBarcode: item.GenerateBarcodes,
		},
		betrag: float64(item.Menge) * item.Preis,
	}

	// ALWAYS pre-allocate barcodes in the database
	barcodes, err := s.bookRepo.GenerateBarcodes(ctx, item.Menge)
	if err != nil {
		return nil, fmt.Errorf("sequence error: %w", err)
	}

	statusText := fmt.Sprintf("Im Zulauf - %s", supplierName)
	if !item.GenerateBarcodes {
		statusText = fmt.Sprintf("Bestellt (ohne Vorab-Barcode) - %s", supplierName)
	}

	// Beklebt der Händler selbst, gilt das Exemplar sofort als etikettiert — sonst stünde
	// ein fertig beklebt geliefertes Buch dauerhaft auf der Nachdruck-Liste, und der
	// Hinweis im Bestellwesen nennte eine Zahl, die niemand mehr abarbeiten kann.
	//
	// Beide Bedingungen müssen zutreffen: Ohne erzeugten Barcode steht für diese Position
	// nichts auf dem Barcodebogen, der Händler kann also nichts aufkleben — dann bleibt das
	// Etikett unsere Aufgabe, auch wenn er sonst beklebt liefert.
	beklebtGeliefert := supplier.IstHauptlieferant && item.GenerateBarcodes

	for i := 0; i < item.Menge; i++ {
		barcodeID := barcodes[i]
		res.copies = append(res.copies, repository.BookCopyInsert{
			TitelID:         item.TitelID,
			BarcodeID:       barcodeID,
			ZustandNotiz:    statusText,
			IstAusleihbar:   false,
			EtikettGedruckt: beklebtGeliefert,
			Einkaufspreis:   item.Preis,
		})

		// Only add to labels for the supplier PDF if requested
		if item.GenerateBarcodes {
			res.labels = append(res.labels, BarcodeLabelDetail{
				BarcodeID: barcodeID,
				Titel:     title.Titel,
				Autor:     title.Autor,
				ISBN:      title.ISBN,
				Signatur:  title.Signatur,
				// „Ansch.J." steht auf der physischen Etikettenvorlage der Schule. Die
				// Exemplare entstehen in dieser Transaktion mit erworben_am = heute, das Jahr
				// ist also schon bekannt — und stimmt damit mit dem überein, was die
				// Lieferantenseite später aus der Datenbank liest.
				AnschaffungsJahr: strconv.Itoa(time.Now().Year()),
			})
		}
	}

	return res, nil
}

// insertBestellverlauf schreibt den Bestellkopf und liefert die erzeugte Bestell-ID.
//
// tokenHash ist leer, wenn dieser Lieferant keinen Bestätigungs-Link bekommt; NULLIF
// macht daraus ein SQL-NULL, damit der Teil-Index (Migration 063) nicht zwei Bestellungen
// ohne Link als Dublette ablehnt.
func (s *OrderService) insertBestellverlauf(ctx context.Context, tx pgx.Tx, req SubmitOrderRequest, supplier *repository.Supplier, gesamtbetrag float64, totalAllocated int, tokenHash string) (string, error) {
	var bestellungID string
	err := tx.QueryRow(ctx, `
		INSERT INTO bestellungen_verlauf
			(lieferant_id, lieferant_name, lieferant_email, kundennummer, gesamtbetrag, anzahl_exemplare,
			 bestaetigungs_token_hash, token_gueltig_bis)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''),
		        CASE WHEN $7 = '' THEN NULL ELSE now() + make_interval(days => $8) END)
		RETURNING id`,
		req.SupplierID, supplier.Name, supplier.Email, supplier.Kundennummer,
		gesamtbetrag, totalAllocated, tokenHash, TokenGueltigkeitTage,
	).Scan(&bestellungID)
	if err != nil {
		return "", fmt.Errorf("bestellverlauf insert: %w", err)
	}
	return bestellungID, nil
}

// insertBestellpositionen schreibt alle Positionen des Bestellkopfs.
func (s *OrderService) insertBestellpositionen(ctx context.Context, tx pgx.Tx, bestellungID string, positionen []bestellungPosition) error {
	if len(positionen) == 0 {
		return nil
	}

	copyRows := make([][]any, 0, len(positionen))
	for _, pos := range positionen {
		copyRows = append(copyRows, []any{
			bestellungID, pos.titelID, pos.titelName, pos.isbn, pos.menge, pos.preis, pos.mitVorabBarcode,
		})
	}

	// Use pgx.CopyFromRows to resolve N+1 queries when inserting multiple order positions
	if _, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"bestellungen_positionen"},
		[]string{"bestellung_id", "titel_id", "titel_name", "isbn", "menge", "einzelpreis", "mit_vorab_barcode"},
		pgx.CopyFromRows(copyRows),
	); err != nil {
		return fmt.Errorf("position bulk insert: %w", err)
	}

	return nil
}
