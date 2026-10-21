package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"bibliothek/db"
	"bibliothek/pkg/bestelllink"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Das Anlegen einer Bestellung aus dem Warenkorb: Bestellkopf, Positionen und die Exemplare
// im Zulauf entstehen in einer Transaktion, die Barcodes werden dabei reserviert. Die Tür prüft
// die Anfrage, füllt den Auftrag und verschickt danach die Mail an den Lieferanten.

// BestellDienst legt Bestellungen an.
type BestellDienst struct {
	datenbank           *db.Verbindung
	katalogAbfragen     *repository.KatalogAbfragen
	lieferantenAbfragen *repository.LieferantenAbfragen
}

// NewBestellDienst baut den Dienst für Bestellungen.
func NewBestellDienst(datenbank *db.Verbindung, katalogAbfragen *repository.KatalogAbfragen) *BestellDienst {
	return &BestellDienst{
		datenbank:           datenbank,
		katalogAbfragen:     katalogAbfragen,
		lieferantenAbfragen: repository.NewLieferantenAbfragen(datenbank.Pool),
	}
}

// BestellAuftragPosition ist eine Position des Warenkorbs.
type BestellAuftragPosition struct {
	TitelID string
	Menge   int
	Preis   float64
	// MitVorabBarcode sagt, ob die Exemplare dieser Position auf dem Barcodebogen für den
	// Lieferanten stehen.
	MitVorabBarcode bool
}

// BestellAuftrag ist ein Warenkorb an einen Lieferanten, wie die Tür ihn aus der Anfrage füllt.
type BestellAuftrag struct {
	LieferantID string
	Positionen  []BestellAuftragPosition
	// IdempotenzSchluessel vergibt die Oberfläche je Absende-Vorgang; ein Doppelklick schickt
	// denselben. Leer heißt: ohne diesen Schutz.
	IdempotenzSchluessel string
	// Mittel ist der Topf der Bestellung (mitteltopf.Land / mitteltopf.Schultraeger), Pflicht.
	Mittel string
}

// BestellterTitel ist ein bestellter Titel mit seiner Menge, wie Anschreiben und Mail ihn
// nennen.
type BestellterTitel struct {
	Titel  string
	Autor  string
	ISBN   string
	Verlag string
	Menge  int
}

// BestellEtikett ist das Etikett eines bestellten Exemplars für den Barcodebogen des
// Lieferanten. Die Felder sind die des Druckauftrags der Tür (buchetikett.Etikett), sie
// wandelt per Typumwandlung.
type BestellEtikett struct {
	BarcodeID string
	Titel     string
	Autor     string
	ISBN      string
	// AnschaffungsJahr steht als „Ansch.J." auf dem Etikett.
	AnschaffungsJahr string
	Signatur         string
	// Topf ist der Topf der Bestellung und entscheidet über den Eigentumsvermerk.
	Topf string
}

// BestellErgebnis ist das Ergebnis einer angelegten Bestellung.
type BestellErgebnis struct {
	LieferantName  string
	LieferantEmail string
	Kundennummer   string
	Etiketten      []BestellEtikett
	BestellteTitel []BestellterTitel
	Menge          int
	// IstHauptlieferant steuert, ob die Bestellmail zusätzlich das große Lernmittel-Etikett
	// anhängt (repository.Lieferant).
	IstHauptlieferant bool
	// BestellungID ist die Kennung der eben geschriebenen Bestellung.
	BestellungID string
	// BestaetigungsToken ist der Klartext für den Link in der Bestellmail. In der Datenbank
	// liegt nur sein Hash; wer ihn hier nicht mitnimmt, erfährt ihn nicht mehr. Leer, wenn der
	// Lieferant keinen Bestätigungsschritt hat.
	BestaetigungsToken string
	// LinkGueltigBis ist der Ablauf des Bestätigungs-Links für die Mail, nil ohne Link.
	LinkGueltigBis *time.Time
	// BereitsVorhanden sagt, dass derselbe Idempotenz-Schlüssel schon eine Bestellung erzeugt
	// hat (Doppelklick); die Tür verschickt dann keine zweite Mail.
	BereitsVorhanden bool
	// Mittel ist der Topf dieser Bestellung (mitteltopf.Land / mitteltopf.Schultraeger). Er
	// steuert den Vermerk in Anschreiben und Mail; Kundennummer ist schon die Nummer dieses
	// Topfs (Lieferant.KundennummerFuer).
	Mittel string
}

// Ablehnungen eines Warenkorbs. Die Tür erkennt sie am Wert, nicht am Wortlaut, und setzt den
// Satz für die Person im Bestellwesen. Der Warenkorb steht nur im Browser, ein anderer Platz kann
// Lieferant oder Titel inzwischen gelöscht haben.
var (
	ErrLieferantUnbekannt = errors.New("der Lieferant ist nicht mehr angelegt, bitte einen anderen wählen")
	ErrTitelUnbekannt     = errors.New("der Titel steht nicht mehr im Katalog, bitte aus dem Warenkorb nehmen")
	ErrMengeUngueltig     = errors.New("die Menge muss zwischen 1 und 200 liegen")
)

// BestellpositionFehler nennt die Position des Warenkorbs, an der das Anlegen scheiterte. Die
// Nummer zählt wie die Liste im Warenkorb, ab 1; ohne sie ließe sich der Warenkorb nicht
// berichtigen.
type BestellpositionFehler struct {
	Position int
	Grund    error
}

func (e *BestellpositionFehler) Error() string {
	return fmt.Sprintf("%v (Position %d)", e.Grund, e.Position)
}

func (e *BestellpositionFehler) Unwrap() error { return e.Grund }

// positionsErgebnis bündelt die aus einer einzelnen Bestellposition erzeugten Daten.
type positionsErgebnis struct {
	bestellterTitel BestellterTitel
	position        repository.BestellpositionNeu
	exemplare       []repository.ExemplarNeu
	etiketten       []BestellEtikett
	betrag          float64
}

// LegeAn legt die Bestellung in einer Transaktion an und reserviert ihre Barcodes.
func (d *BestellDienst) LegeAn(ctx context.Context, auftrag BestellAuftrag) (*BestellErgebnis, error) {
	// Der Topf ist Pflicht und wird vor jedem Datenbankzugriff geprüft, damit keine Barcodes
	// für eine Bestellung reserviert werden, die es nicht geben wird.
	if !mitteltopf.IstGueltig(auftrag.Mittel) {
		return nil, mitteltopf.ErrUngueltig
	}

	lieferant, err := d.lieferantenAbfragen.Lade(ctx, auftrag.LieferantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLieferantUnbekannt
		}
		return nil, err
	}

	// Der Bestätigungs-Link entsteht nur für den Hauptlieferanten, der selbst etikettiert und
	// bestätigt. Alle anderen bekämen eine Seite, auf der es nichts zu tun gibt; für sie bleibt
	// es bei der Bestellmail.
	var token, tokenHash string
	linkTage := bestelllink.TageVorgabe
	if lieferant.IstHauptlieferant {
		token, tokenHash, err = bestelllink.ErzeugeToken()
		if err != nil {
			return nil, fmt.Errorf("bestaetigungs-token: %w", err)
		}
		linkTage = LadeBestelllinkTage(ctx, d.datenbank.Pool)
	}

	tx, err := d.datenbank.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Verwerfe(ctx, tx)

	posten, err := d.verarbeitePositionen(ctx, tx, auftrag, lieferant)
	if err != nil {
		return nil, err
	}

	// Der Bestellkopf kommt vor den Exemplaren: Sie tragen ihre bestellung_id, und die gibt es
	// erst, wenn der Kopf geschrieben ist. Keine der beiden Einfügungen liest die andere; die
	// Barcodes sind in verarbeitePositionen schon reserviert, und der Kopf zählt nur die dort
	// errechneten Summen.
	bestellungID, linkGueltigBis, err := d.schreibeBestellkopf(ctx, tx, auftrag, lieferant, posten, tokenHash, linkTage)
	if errors.Is(err, errBestellungDuplikat) {
		// Doppelklick: Dieselbe Bestellung lief schon durch. Die Transaktion wird zurückgerollt,
		// die hier reservierten Exemplare verschwinden wieder, und gemeldet wird die bestehende
		// Bestellung.
		return d.ladeBestehendeBestellung(ctx, auftrag.IdempotenzSchluessel)
	}
	if err != nil {
		return nil, err
	}
	if err := d.schreibeExemplareUndPositionen(ctx, tx, bestellungID, posten); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &BestellErgebnis{
		LieferantName:      lieferant.Name,
		LieferantEmail:     lieferant.Email,
		Kundennummer:       lieferant.KundennummerFuer(auftrag.Mittel),
		Mittel:             auftrag.Mittel,
		Etiketten:          posten.etiketten,
		BestellteTitel:     posten.bestellteTitel,
		Menge:              posten.menge,
		IstHauptlieferant:  lieferant.IstHauptlieferant,
		BestellungID:       bestellungID,
		BestaetigungsToken: token,
		LinkGueltigBis:     linkGueltigBis,
	}, nil
}

// bestellPosten ist, was die Positionen eines Warenkorbs zusammen ergeben.
type bestellPosten struct {
	etiketten      []BestellEtikett
	bestellteTitel []BestellterTitel
	positionen     []repository.BestellpositionNeu
	exemplare      []repository.ExemplarNeu
	gesamtbetrag   float64
	menge          int
}

// verarbeitePositionen prüft jede Position, reserviert ihre Barcodes und sammelt Exemplare,
// Etiketten und Summen der Bestellung.
func (d *BestellDienst) verarbeitePositionen(ctx context.Context, tx pgx.Tx, auftrag BestellAuftrag, lieferant *repository.Lieferant) (bestellPosten, error) {
	posten := bestellPosten{etiketten: make([]BestellEtikett, 0), bestellteTitel: make([]BestellterTitel, 0)}
	for i, position := range auftrag.Positionen {
		erg, err := d.verarbeitePosition(ctx, tx, position, lieferant)
		if err != nil {
			// Der Warenkorb schickt die Positionen in der Reihenfolge seiner Liste.
			return bestellPosten{}, &BestellpositionFehler{Position: i + 1, Grund: err}
		}
		posten.bestellteTitel = append(posten.bestellteTitel, erg.bestellterTitel)
		posten.positionen = append(posten.positionen, erg.position)
		posten.exemplare = append(posten.exemplare, erg.exemplare...)
		// Der Topf jedes Etiketts ist der Topf dieser Bestellung, derselbe Wert, den
		// repository.ExemplarTopfSQL später aus bestellungen_verlauf.mittel liest. Die Exemplare
		// entstehen erst in dieser Transaktion und lassen sich noch nicht abfragen.
		// TestEtikettenWegeDruckenDasselbe hält die Wege am PDF zusammen.
		for i := range erg.etiketten {
			erg.etiketten[i].Topf = auftrag.Mittel
		}
		posten.etiketten = append(posten.etiketten, erg.etiketten...)
		posten.gesamtbetrag += erg.betrag
		posten.menge += erg.position.Menge
	}
	return posten, nil
}

// schreibeExemplareUndPositionen hängt die Exemplare an den geschriebenen Bestellkopf und
// schreibt sie und die Positionen.
func (d *BestellDienst) schreibeExemplareUndPositionen(ctx context.Context, tx pgx.Tx, bestellungID string, posten bestellPosten) error {
	for i := range posten.exemplare {
		posten.exemplare[i].BestellungID = bestellungID
	}
	if err := d.katalogAbfragen.LegeExemplareAn(ctx, tx, posten.exemplare); err != nil {
		return fmt.Errorf("bulk insert error: %w", err)
	}
	if err := repository.SchreibeBestellpositionen(ctx, tx, bestellungID, posten.positionen); err != nil {
		return fmt.Errorf("position bulk insert: %w", err)
	}
	return nil
}

// verarbeitePosition prüft eine Bestellposition, lädt den Titel, reserviert die Barcodes und
// erzeugt die Exemplare samt ihren Etiketten.
func (d *BestellDienst) verarbeitePosition(ctx context.Context, tx pgx.Tx, position BestellAuftragPosition, lieferant *repository.Lieferant) (*positionsErgebnis, error) {
	lieferantName := lieferant.Name

	if position.Menge <= 0 || position.Menge > 200 {
		return nil, fmt.Errorf("%w, angegeben ist %d", ErrMengeUngueltig, position.Menge)
	}

	titel, err := d.katalogAbfragen.LadeTitel(ctx, tx, position.TitelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTitelUnbekannt
		}
		return nil, err
	}

	erg := &positionsErgebnis{
		bestellterTitel: BestellterTitel{
			Titel:  titel.Titel,
			Autor:  titel.Autor,
			ISBN:   titel.ISBN,
			Verlag: titel.Verlag,
			Menge:  position.Menge,
		},
		position: repository.BestellpositionNeu{
			TitelID:         position.TitelID,
			TitelName:       titel.Titel,
			ISBN:            titel.ISBN,
			Menge:           position.Menge,
			Einzelpreis:     position.Preis,
			MitVorabBarcode: position.MitVorabBarcode,
		},
		betrag: float64(position.Menge) * position.Preis,
	}

	// Jedes Exemplar bekommt seinen Barcode schon mit der Bestellung, auch ohne Barcodebogen.
	barcodes, err := d.katalogAbfragen.ErzeugeBarcodes(ctx, position.Menge)
	if err != nil {
		return nil, fmt.Errorf("sequence error: %w", err)
	}

	// Text für Menschen und Status für das Programm sind getrennt: Die Notiz nennt den
	// Lieferanten, die Spalte steuert den Weg des Exemplars.
	statusText := fmt.Sprintf("Im Zulauf - %s", lieferantName)
	bestellstatus := "im_zulauf"
	if !position.MitVorabBarcode {
		statusText = fmt.Sprintf("Bestellt (ohne Vorab-Barcode) - %s", lieferantName)
		bestellstatus = "bestellt"
	}

	// Beklebt der Händler selbst, gilt das Exemplar sofort als etikettiert. Sonst stünde ein
	// fertig beklebt geliefertes Buch dauerhaft auf der Nachdruck-Liste. Beide Bedingungen
	// müssen zutreffen: Ohne erzeugten Barcode steht für diese Position nichts auf dem
	// Barcodebogen, der Händler kann nichts aufkleben, und das Etikett bleibt Aufgabe der
	// Schule.
	beklebtGeliefert := lieferant.IstHauptlieferant && position.MitVorabBarcode

	for i := 0; i < position.Menge; i++ {
		barcodeID := barcodes[i]
		erg.exemplare = append(erg.exemplare, repository.ExemplarNeu{
			TitelID:         position.TitelID,
			BarcodeID:       barcodeID,
			ZustandNotiz:    statusText,
			Bestellstatus:   bestellstatus,
			IstAusleihbar:   false,
			EtikettGedruckt: beklebtGeliefert,
			Einkaufspreis:   position.Preis,
		})

		// Auf den Barcodebogen für den Lieferanten kommt das Exemplar nur auf Wunsch.
		if position.MitVorabBarcode {
			erg.etiketten = append(erg.etiketten, BestellEtikett{
				BarcodeID: barcodeID,
				Titel:     titel.Titel,
				Autor:     titel.Autor,
				ISBN:      titel.ISBN,
				Signatur:  titel.Signatur,
				// Die Exemplare entstehen in dieser Transaktion mit erworben_am = heute in der Zone
				// der Schule (Vorgabe der Spalte). Das Jahr dieses Tages liest später auch die Seite
				// des Lieferanten aus der Datenbank.
				AnschaffungsJahr: strconv.Itoa(schulzeit.Jetzt().Year()),
			})
		}
	}

	return erg, nil
}

// schreibeBestellkopf schreibt den Bestellkopf und liefert die erzeugte Bestell-ID. Vom
// Warenkorb stehen dort nur die Summen: Gesamtbetrag und Zahl der Exemplare. tokenHash ist
// leer, wenn dieser Lieferant keinen Bestätigungs-Link bekommt.
func (d *BestellDienst) schreibeBestellkopf(ctx context.Context, tx pgx.Tx, auftrag BestellAuftrag, lieferant *repository.Lieferant, posten bestellPosten, tokenHash string, linkTage int) (string, *time.Time, error) {
	// Die Kundennummer ist die des Topfs (Lieferant.KundennummerFuer): Händler führen Lernmittel
	// und Bibliothek oft als getrennte Kundenkonten. Sie steht als Abschrift auf der
	// Bestellung, wie Name und Adresse, damit der Beleg auch nach einer Änderung am Lieferanten
	// stimmt.
	bestellungID, linkGueltigBis, err := repository.LegeBestellkopfAn(ctx, tx, repository.BestellkopfNeu{
		LieferantID:          auftrag.LieferantID,
		LieferantName:        lieferant.Name,
		LieferantEmail:       lieferant.Email,
		Kundennummer:         lieferant.KundennummerFuer(auftrag.Mittel),
		Gesamtbetrag:         posten.gesamtbetrag,
		Menge:                posten.menge,
		TokenHash:            tokenHash,
		LinkTage:             linkTage,
		IdempotenzSchluessel: leerAlsNil(auftrag.IdempotenzSchluessel),
		Mittel:               auftrag.Mittel,
	})
	// Keine Zeile: Ein Doppelklick mit demselben Schlüssel lief am Unique-Index auf.
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, errBestellungDuplikat
	}
	if err != nil {
		return "", nil, fmt.Errorf("bestellverlauf insert: %w", err)
	}
	return bestellungID, linkGueltigBis, nil
}

// leerAlsNil macht aus einem leeren Text einen fehlenden Wert: Der Idempotenz-Schlüssel ist
// in der Spalte NULL, wenn die Oberfläche keinen schickt.
func leerAlsNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// errBestellungDuplikat meldet, dass es zu diesem Idempotenz-Schlüssel schon eine Bestellung
// gibt (Doppelklick). LegeAn liefert dann die bestehende Bestellung mit
// BereitsVorhanden.
var errBestellungDuplikat = errors.New("bestellung mit diesem idempotenz-schlüssel existiert bereits")

// ladeBestehendeBestellung liest die Bestellung, die es zum Idempotenz-Schlüssel schon gibt,
// am Pool und außerhalb der zurückgerollten Transaktion. Das Ergebnis trägt nur die Meldung;
// Etiketten und Mail gab es beim ersten Lauf.
func (d *BestellDienst) ladeBestehendeBestellung(ctx context.Context, idempotenzSchluessel string) (*BestellErgebnis, error) {
	var erg BestellErgebnis
	erg.BereitsVorhanden = true
	var err error
	erg.BestellungID, erg.LieferantName, erg.Menge, err = repository.LadeBestellungZuIdempotenzSchluessel(ctx, d.datenbank.Pool, idempotenzSchluessel)
	if err != nil {
		return nil, fmt.Errorf("bestehende bestellung laden: %w", err)
	}
	return &erg, nil
}
