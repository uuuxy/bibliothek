package littera

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bibliothek/internal/uebernahme"

	"github.com/jackc/pgx/v5"
)

// BestandBericht ist die Bilanz des Bestandsteils.
type BestandBericht struct {
	QuellTitel     int
	QuellExemplare int
	Titel          int // gemeldet: geschrieben
	Exemplare      int
	Uebersprungen  int

	IstTitel     int // tatsächlicher Zeilenzuwachs in PostgreSQL
	IstExemplare int
	AbgleichOK   bool

	// TitelIDs bildet Littera-Titel → UUID ab, ExemplarIDs Littera-Exemplar → UUID.
	// Der Ausleihteil braucht die zweite Karte.
	TitelIDs    map[string]string
	ExemplarIDs map[string]string
}

const sqlTitelEinfuegen = `
	INSERT INTO buecher_titel
		(titel, untertitel, autor, isbn, verlag, erscheinungsjahr, beschreibung,
		 medientyp, signatur, erweiterte_eigenschaften, erstellt_am)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	RETURNING id`

// etikett_gedruckt = true: Altbestand traegt seine Littera-Etiketten physisch —
// siehe gleiche Begruendung am Sammelimport (import_dynamic.go).
const sqlExemplarEinfuegen = `
	INSERT INTO buecher_exemplare
		(titel_id, barcode_id, erworben_am, ist_ausleihbar, einkaufspreis,
		 erweiterte_eigenschaften, erstellt_am, etikett_gedruckt)
	VALUES ($1,$2,$3,true,$4,$5,$6,true)
	RETURNING id`

// SchreibeBestand überträgt Titel und Exemplare.
//
// Die atomare Einheit ist der TITEL, nicht das Exemplar: Die Stückzahl der Quelle wird
// vollständig zu Exemplar-Zeilen. Ein Titel mit 5 Stück, dem nur drei folgen, wäre ein stiller
// Bestandsfehler, den niemand je bemerkt. Entweder ein Titel kommt vollständig an — oder
// gar nicht, und dann steht er als FEHLER im Protokoll.
func (s *Schreiber) SchreibeBestand(ctx context.Context, ab *Altbestand) (BestandBericht, error) {
	bericht := BestandBericht{
		QuellTitel:     len(ab.Titel),
		QuellExemplare: len(ab.Exemplare),
		TitelIDs:       make(map[string]string, len(ab.Titel)),
		ExemplarIDs:    make(map[string]string, len(ab.Exemplare)),
	}

	vorherT, vorherE, err := s.zaehleBestand(ctx)
	if err != nil {
		return bericht, err
	}

	barcodes, err := s.klaereBarcodes(ctx, ab.Exemplare, ab.Fremdbarcodes)
	if err != nil {
		return bericht, err
	}
	isbns, err := s.vorhandeneISBNs(ctx)
	if err != nil {
		return bericht, err
	}

	lauf := &bestandslauf{s: s, ab: ab, barcodes: barcodes, isbns: isbns,
		exemplareJeTitel: ExemplareJeTitel(ab.Exemplare), bericht: &bericht}
	if err := lauf.alleBatches(ctx); err != nil {
		return bericht, err
	}

	nachherT, nachherE, err := s.zaehleBestand(ctx)
	if err != nil {
		return bericht, fmt.Errorf("der Abgleich nach der Übernahme schlug fehl: %w", err)
	}
	bericht.IstTitel, bericht.IstExemplare = nachherT-vorherT, nachherE-vorherE
	bericht.AbgleichOK = bericht.IstTitel == bericht.Titel && bericht.IstExemplare == bericht.Exemplare
	return bericht, nil
}

func (s *Schreiber) zaehleBestand(ctx context.Context) (titel, exemplare int, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM buecher_titel), (SELECT count(*) FROM buecher_exemplare)
	`).Scan(&titel, &exemplare)
	if err != nil {
		return 0, 0, fmt.Errorf("konnte den Bestand nicht zählen: %w", err)
	}
	return titel, exemplare, nil
}

// bestandslauf hält den Zustand einer Bestandsübernahme zusammen. Ohne dieses Bündel
// müsste jede Ebene sieben Parameter durchreichen.
type bestandslauf struct {
	s                *Schreiber
	ab               *Altbestand
	barcodes         map[string]string // Littera-Exemplar-ID → barcode_id
	isbns            map[string]string // normalisierte ISBN → Littera-Titel-ID
	exemplareJeTitel map[string][]Exemplar
	bericht          *BestandBericht
}

// alleBatches schreibt die Titel in Transaktionen von BatchGroesse Stück.
//
// Meldet ein Batch einen Fehler, ist die Übernahme zu Ende: dann betrifft er nicht mehr
// einen einzelnen Datensatz, sondern die Verbindung oder die Transaktion selbst.
// Weiterzulaufen hieße, Tausende Zeilen „übersprungen" zu protokollieren, die in
// Wahrheit nie versucht wurden.
func (l *bestandslauf) alleBatches(ctx context.Context) error {
	titel := l.ab.Titel
	for i := 0; i < len(titel); i += l.s.opt.BatchGroesse {
		ende := min(i+l.s.opt.BatchGroesse, len(titel))
		if err := l.einBatch(ctx, titel[i:ende]); err != nil {
			return fmt.Errorf("abgebrochen ab Titel %s: %w", titel[i].ID, err)
		}
	}
	return nil
}

func (l *bestandslauf) einBatch(ctx context.Context, batch []Titel) error {
	tx, err := l.s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("konnte die Transaktion nicht öffnen: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	for _, t := range batch {
		if err := l.einTitel(ctx, tx, t); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"der COMMIT schlug fehl – KEIN Titel dieses Batches wurde geschrieben, auch nicht "+
				"die zuvor als übernommen protokollierten: %w", err)
	}
	return nil
}

// einTitel schreibt einen Titel samt Exemplaren in einem Savepoint und bucht das Ergebnis.
func (l *bestandslauf) einTitel(ctx context.Context, tx pgx.Tx, t Titel) error {
	var titelID string
	var exemplarIDs map[string]string
	var reservierteISBN string

	erg, err := uebernahme.ImSavepoint(ctx, tx, "littera_id="+t.ID, func(sp pgx.Tx) error {
		var innerErr error
		titelID, exemplarIDs, reservierteISBN, innerErr = l.schreibeTitel(ctx, sp, t)
		return innerErr
	})
	if err != nil {
		return err
	}

	if !erg.Uebernommen {
		// Die ISBN-Vormerkung lebte nur im Arbeitsspeicher und muss mit zurück, sonst
		// verlöre der nächste echte Titel sie an einen, der gar nicht in der Datenbank steht.
		if reservierteISBN != "" {
			delete(l.isbns, reservierteISBN)
		}
		l.bericht.Uebersprungen++
		l.s.prot.Fehler(t.ID, t.ISBN,
			"Titel samt Exemplaren übersprungen – "+uebernahme.BeschreibeFehler(erg.Zurueckgerollt))
		return nil
	}

	l.bericht.Titel++
	l.bericht.Exemplare += len(exemplarIDs)
	l.bericht.TitelIDs[t.ID] = titelID
	for littera, uuid := range exemplarIDs {
		l.bericht.ExemplarIDs[littera] = uuid
	}
	return nil
}

// schreibeTitel führt die INSERTs aus. Abwertungen werden hier protokolliert, harte
// Fehler unverändert nach oben gereicht, damit ImSavepoint sie einordnen kann.
func (l *bestandslauf) schreibeTitel(
	ctx context.Context, tx pgx.Tx, t Titel,
) (titelID string, exemplarIDs map[string]string, reservierteISBN string, err error) {
	exemplare := l.exemplareJeTitel[t.ID]
	f := l.felder(t)
	reservierteISBN = uebernahme.KlaereISBN(l.s.prot, t.ID, t.ISBN, l.isbns)

	eigenschaften, err := jsonEigenschaften(map[string]string{
		"quelle":     "littera",
		"littera_id": t.ID,
	})
	if err != nil {
		return "", nil, reservierteISBN, err
	}

	err = tx.QueryRow(ctx, sqlTitelEinfuegen,
		f.titel, f.untertitel, f.autor, uebernahme.Nullbar(reservierteISBN), f.verlag,
		jahrOderNil(t.Erscheinungsjahr), uebernahme.Nullbar(t.Beschreibung),
		f.medientyp, f.signatur, eigenschaften, l.s.opt.Jetzt,
	).Scan(&titelID)
	if err != nil {
		return "", nil, reservierteISBN, fmt.Errorf("beim Titel %q: %w", f.titel, err)
	}

	exemplarIDs, err = l.schreibeExemplare(ctx, tx, titelID, exemplare)
	return titelID, exemplarIDs, reservierteISBN, err
}

// schreibeExemplare schreibt alle Exemplare eines Titels und bricht beim ersten Fehler ab
// — der Titel ist die atomare Einheit, ein halb gefüllter Bestand wäre schlimmer als ein
// protokollierter Ausfall.
func (l *bestandslauf) schreibeExemplare(
	ctx context.Context, tx pgx.Tx, titelID string, exemplare []Exemplar,
) (map[string]string, error) {
	ids := make(map[string]string, len(exemplare))
	if len(exemplare) == 0 {
		return ids, nil
	}

	batch := &pgx.Batch{}
	for _, e := range exemplare {
		barcode, ok := l.barcodes[e.ID]
		if !ok {
			return nil, fmt.Errorf("%w: Exemplar %s hat keinen Barcode zugewiesen bekommen",
				uebernahme.ErrZeile, e.ID)
		}
		eigenschaften, err := jsonEigenschaften(map[string]string{
			"quelle":             "littera",
			"littera_id":         e.ID,
			"littera_etikett":    e.Barcode,
			"littera_signatur":   e.Signatur,
			"littera_exemplarnr": e.Exemplarnummer,
		})
		if err != nil {
			return nil, err
		}
		batch.Queue(sqlExemplarEinfuegen, titelID, barcode,
			erworbenAm(e, l.s.opt.Jetzt), e.Preis, eigenschaften, l.s.opt.Jetzt)
	}

	br := tx.SendBatch(ctx, batch)
	defer func() { _ = br.Close() }() //nolint:errcheck

	for _, e := range exemplare {
		var id string
		if err := br.QueryRow().Scan(&id); err != nil {
			return nil, fmt.Errorf("beim Exemplar mit Barcode %s: %w", l.barcodes[e.ID], err)
		}
		ids[e.ID] = id
	}
	return ids, nil
}

// titelfelder sind die auf Spaltenbreite gebrachten Textfelder eines Titels.
type titelfelder struct {
	titel      string
	untertitel *string
	autor      *string
	verlag     *string
	signatur   *string
	medientyp  string
}

func (l *bestandslauf) felder(t Titel) titelfelder {
	k := func(feld, wert string, max int) string {
		return uebernahme.Kuerze(l.s.prot, t.ID, t.ISBN, feld, wert, max)
	}
	kn := func(feld, wert string, max int) *string {
		return uebernahme.KuerzeNullbar(l.s.prot, t.ID, t.ISBN, feld, wert, max)
	}

	f := titelfelder{
		titel:      k("titel", t.Haupttitel, uebernahme.MaxFreitext),
		untertitel: kn("untertitel", t.Untertitel, uebernahme.MaxFreitext),
		autor:      kn("autor", t.Autor, uebernahme.MaxFreitext),
		verlag:     kn("verlag", l.ab.Verlage[t.VerlagID], uebernahme.MaxFreitext),
		signatur:   kn("signatur", l.ab.Signaturen[t.ID], uebernahme.MaxFreitext),
		medientyp:  "Buch",
	}
	// buecher_titel.titel ist NOT NULL; ein Katalogeintrag ohne Aufschrift wäre in der
	// Oberfläche eine leere Zeile, die niemand zuordnen kann.
	if f.titel == "" {
		f.titel = "[ohne Titel, Littera " + t.ID + "]"
		l.s.prot.Warnung(t.ID, t.ISBN, "Haupttitel ist leer – Platzhalter eingesetzt")
	}
	if name := l.ab.Medienarten[t.MedienartID]; name != "" {
		f.medientyp = k("medientyp", name, uebernahme.MaxMedientyp)
	}
	return f
}

// erworbenAm liefert das Zugangsdatum; ohne lesbaren Wert das Datum des Laufs.
// buecher_exemplare.erworben_am ist NOT NULL.
func erworbenAm(e Exemplar, ersatz time.Time) time.Time {
	if t, ok := DatumAus(e.Zugangsdatum); ok {
		return t
	}
	return ersatz
}

func jahrOderNil(jahr int) *int {
	if jahr == 0 {
		return nil
	}
	return &jahr
}

// jsonEigenschaften serialisiert die Herkunftsangaben für die JSONB-Spalte. Leere Werte
// bleiben draußen, damit der Datensatz nicht mit Leerstrings zugestellt wird.
func jsonEigenschaften(werte map[string]string) (string, error) {
	gefiltert := make(map[string]string, len(werte))
	for k, v := range werte {
		if v != "" {
			gefiltert[k] = v
		}
	}
	b, err := json.Marshal(gefiltert)
	if err != nil {
		return "", fmt.Errorf("%w: erweiterte_eigenschaften nicht serialisierbar: %v",
			uebernahme.ErrZeile, err)
	}
	return string(b), nil
}
