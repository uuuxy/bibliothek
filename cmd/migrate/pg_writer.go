package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"bibliothek/internal/uebernahme"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const errorLogPath = "migration_errors.log"

// Die Protokoll-, Fehlerklassen-, ISBN- und Savepoint-Logik dieses Werkzeugs lebt seit
// dem Littera-Schreibpfad in internal/uebernahme. Sie war hier entstanden und gegen
// echtes PostgreSQL gehärtet; sie zweimal zu halten hätte bedeutet, dass die zweite
// Fassung dieselben Fehler noch einmal macht.
func newErrLogger() (*uebernahme.Protokoll, error) { return newErrLoggerAt(errorLogPath) }

// newErrLoggerAt trennt den Pfad vom Aufruf, damit Tests in ein temporäres Verzeichnis
// schreiben können, statt das Arbeitsverzeichnis zu verschmutzen.
func newErrLoggerAt(pfad string) (*uebernahme.Protokoll, error) {
	return uebernahme.NeuesProtokoll(pfad, "mysql_id")
}

// buildErweiterteEigenschaften serialises free-text fields into the JSONB column.
func buildErweiterteEigenschaften(m mysqlMedium) (string, error) {
	props := make(map[string]string)
	if m.Standort.Valid && m.Standort.String != "" {
		props["standort"] = m.Standort.String
	}
	if m.Regal.Valid && m.Regal.String != "" {
		props["regal"] = m.Regal.String
	}
	if m.Notizen.Valid && m.Notizen.String != "" {
		props["notizen"] = m.Notizen.String
	}
	// Add legacy source reference for traceability
	props["mysql_id"] = strconv.Itoa(m.ID)

	b, err := json.Marshal(props)
	if err != nil {
		return "{}", fmt.Errorf("json marshal: %w", err)
	}
	return string(b), nil
}

func nullableString(s sql.NullString) *string {
	if !s.Valid || s.String == "" {
		return nil
	}
	v := s.String
	return &v
}

func nullableInt(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

// nullStr converts an empty string to a typed nil suitable for pgx nullable columns.
func nullStr(s string) *string { return uebernahme.Nullbar(s) }

// isbnRoh liefert die ISBN der Quellzeile unverändert — der Wert, den jemand in der
// Quelldatei suchen würde.
func isbnRoh(m mysqlMedium) string {
	if m.ISBN.Valid {
		return m.ISBN.String
	}
	return ""
}

func quellID(m mysqlMedium) string { return strconv.Itoa(m.ID) }

// batchErgebnis hält, was ein Batch bewirkt hat. Uebersprungen zählt die Titel, die
// bewusst ausgelassen und als FEHLER protokolliert wurden.
type batchErgebnis struct {
	Titel         int
	Exemplare     int
	Uebersprungen int
}

// mediumErgebnis hält, was ein einzelner Titel bewirkt hat.
type mediumErgebnis struct {
	Titel     int // 0 oder 1
	Exemplare int
	ISBN      string // im Lauf reservierte ISBN; bei Rücknahme wieder freizugeben
}

// insertBatch schreibt einen Batch Titel in EINER Transaktion. Jeder einzelne Titel ist
// darin durch einen Savepoint abgesichert (siehe insertMediumAtomar), sodass ein
// fehlerhafter Datensatz nur sich selbst kostet.
//
// Ein Fehler-Rückgabewert bedeutet: die Übernahme ist zu Ende. Dann steht aus diesem
// Batch nichts in der Datenbank (der defer rollt zurück), und die Zahlen sagen das auch.
func insertBatch(
	ctx context.Context,
	pool *pgxpool.Pool,
	batch []mysqlMedium,
	seenISBNs map[string]string, // isbn → mysql source ID; updated in-place
	el *uebernahme.Protokoll,
	barcodeSeq *int,
) (batchErgebnis, error) {
	var res batchErgebnis

	tx, err := pool.Begin(ctx)
	if err != nil {
		return batchErgebnis{}, fmt.Errorf("konnte die Transaktion nicht öffnen: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	for _, m := range batch {
		erg, err := insertMediumAtomar(ctx, tx, m, seenISBNs, el, barcodeSeq)
		if err != nil {
			return batchErgebnis{}, err
		}
		res.Titel += erg.Titel
		res.Exemplare += erg.Exemplare
		if erg.Titel == 0 {
			res.Uebersprungen++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return batchErgebnis{}, fmt.Errorf(
			"der COMMIT des Batches schlug fehl – KEIN Datensatz dieses Batches wurde geschrieben, "+
				"auch nicht die zuvor als übernommen protokollierten: %w", err)
	}
	return res, nil
}

// insertMediumAtomar schreibt einen Titel samt allen seinen Exemplaren in einem SAVEPOINT.
//
// Die atomare Einheit ist bewusst der TITEL, nicht das einzelne Exemplar: buecher_titel.stock
// speichert die Stückzahl aus der Quelle. Ein Titel mit stock=5, dem nur 3 Exemplare folgen,
// wäre ein stiller Bestandsfehler, den niemand je bemerkt. Entweder der Titel kommt
// vollständig an — oder gar nicht, und dann steht er als FEHLER im Protokoll.
func insertMediumAtomar(
	ctx context.Context,
	tx pgx.Tx,
	m mysqlMedium,
	seenISBNs map[string]string,
	el *uebernahme.Protokoll,
	barcodeSeq *int,
) (mediumErgebnis, error) {
	var res mediumErgebnis
	erg, err := uebernahme.ImSavepoint(ctx, tx, "mysql_id="+quellID(m), func(sp pgx.Tx) error {
		var insErr error
		res, insErr = insertMedium(ctx, sp, m, seenISBNs, el, barcodeSeq)
		return insErr
	})
	if err != nil {
		return mediumErgebnis{}, err
	}
	if erg.Uebernommen {
		return res, nil
	}

	// Die ISBN-Reservierung lebte nur im Arbeitsspeicher und muss mit zurück. Sonst
	// verlöre der nächste echte Titel mit dieser ISBN sie an einen Datensatz, der gar
	// nicht in der Datenbank steht.
	if res.ISBN != "" {
		delete(seenISBNs, res.ISBN)
	}
	el.Fehler(quellID(m), isbnRoh(m), "Titel übersprungen – "+uebernahme.BeschreibeFehler(erg.Zurueckgerollt))
	return mediumErgebnis{}, nil
}

const sqlInsertTitel = `
	INSERT INTO buecher_titel
		(titel, untertitel, autor, isbn, verlag, erscheinungsjahr,
		 beschreibung, medientyp, erweiterte_eigenschaften,
		 stock, erstellt_am)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	RETURNING id`

// insertMedium schreibt Titel und Exemplare innerhalb des übergebenen Savepoints.
// Abwertungen (ungültige ISBN, Dublette, gekürzter Freitext) werden hier als Warnung
// protokolliert; harte Fehler werden unverändert nach oben gereicht, damit
// insertMediumAtomar sie einordnen kann.
func insertMedium(
	ctx context.Context,
	tx pgx.Tx,
	m mysqlMedium,
	seenISBNs map[string]string,
	el *uebernahme.Protokoll,
	barcodeSeq *int,
) (mediumErgebnis, error) {
	var res mediumErgebnis
	res.ISBN = uebernahme.KlaereISBN(el, quellID(m), isbnRoh(m), seenISBNs)

	jsonbProps, err := buildErweiterteEigenschaften(m)
	if err != nil {
		return res, fmt.Errorf("%w: JSONB konnte nicht erzeugt werden: %v", uebernahme.ErrZeile, err)
	}

	felder := kuerzeFelder(el, m)

	erstelltAm := time.Now()
	if m.ErstelltAm.Valid {
		erstelltAm = m.ErstelltAm.Time
	}

	var titelID string
	err = tx.QueryRow(ctx, sqlInsertTitel,
		felder.Titel,
		felder.Untertitel,
		felder.Autor,
		nullStr(res.ISBN),
		felder.Verlag,
		nullableInt(m.Erscheinungsjahr),
		nullableString(m.Beschreibung),
		felder.Medientyp,
		jsonbProps,
		m.Anzahl,
		erstelltAm,
	).Scan(&titelID)
	if err != nil {
		return res, fmt.Errorf("beim Titel %q: %w", felder.Titel, err)
	}
	res.Titel = 1

	res.Exemplare, err = insertExemplare(ctx, tx,
		exemplarInsert{TitelID: titelID, Medium: m, ErstelltAm: erstelltAm}, barcodeSeq)
	if err != nil {
		return res, err
	}
	return res, nil
}

// Spaltenbreiten: siehe uebernahme.MaxFreitext / MaxMedientyp.
const (
	maxTitelSpalte     = uebernahme.MaxFreitext
	maxMedientypSpalte = uebernahme.MaxMedientyp
)

// titelFelder sind die auf Spaltenbreite gebrachten Freitextfelder eines Titels.
type titelFelder struct {
	Titel      string
	Untertitel *string
	Autor      *string
	Verlag     *string
	Medientyp  string
}

// kuerzeFelder bringt die Freitextfelder auf die Spaltenbreite und meldet jede Kürzung.
func kuerzeFelder(el *uebernahme.Protokoll, m mysqlMedium) titelFelder {
	id, isbn := quellID(m), isbnRoh(m)
	f := titelFelder{
		Titel:      uebernahme.Kuerze(el, id, isbn, "titel", m.Titel, maxTitelSpalte),
		Untertitel: uebernahme.KuerzeNullbar(el, id, isbn, "untertitel", m.Untertitel.String, maxTitelSpalte),
		Autor:      uebernahme.KuerzeNullbar(el, id, isbn, "autor", m.Autor.String, maxTitelSpalte),
		Verlag:     uebernahme.KuerzeNullbar(el, id, isbn, "verlag", m.Verlag.String, maxTitelSpalte),
		Medientyp:  "Buch",
	}
	if m.Medientyp.Valid && m.Medientyp.String != "" {
		f.Medientyp = uebernahme.Kuerze(el, id, isbn, "medientyp", m.Medientyp.String, maxMedientypSpalte)
	}
	return f
}

// exemplarInsert bündelt die Daten eines Titels, dessen Exemplare geschrieben werden.
type exemplarInsert struct {
	TitelID    string
	Medium     mysqlMedium
	ErstelltAm time.Time
}

const sqlInsertExemplar = `
	INSERT INTO buecher_exemplare
		(titel_id, barcode_id, erworben_am, ist_ausleihbar,
		 erweiterte_eigenschaften, erstellt_am)
	VALUES ($1, $2, CURRENT_DATE, true, '{}', $3)`

// insertExemplare schreibt die Exemplare eines Titels. Es bricht beim ersten Fehler ab,
// statt weiterzuzählen: der Titel ist die atomare Einheit (siehe insertMediumAtomar),
// ein halb gefüllter Bestand wäre schlimmer als ein protokollierter Ausfall. Der Fehler
// trägt den Barcode im Text — das ist die Angabe, mit der jemand die Kollision in der
// Quelldatei tatsächlich findet.
//
// Der Zähler wird auch im Fehlerfall weitergestellt. Die verbrannten Nummern sind
// folgenlos — Barcodes sind undurchsichtige Etiketten, Lücken darin bedeuten nichts.
// Gäbe man sie zurück, liefe der nächste Titel in dieselbe Kollision und die Übernahme
// bliebe an einer einzigen belegten Nummer hängen, Datensatz für Datensatz.
func insertExemplare(ctx context.Context, tx pgx.Tx, data exemplarInsert, barcodeSeq *int) (int, error) {
	barcodes := nextBarcodes(*barcodeSeq, data.Medium.Anzahl)
	*barcodeSeq += data.Medium.Anzahl

	for _, bc := range barcodes {
		if !validateBarcode(bc) {
			return 0, fmt.Errorf(
				"%w: erzeugter Barcode %q entspricht nicht dem Muster B-<Ziffern>", uebernahme.ErrZeile, bc)
		}
	}

	batch := &pgx.Batch{}
	for _, bc := range barcodes {
		batch.Queue(sqlInsertExemplar, data.TitelID, bc, data.ErstelltAm)
	}

	br := tx.SendBatch(ctx, batch)
	defer func() { _ = br.Close() }() //nolint:errcheck

	for i, bc := range barcodes {
		if _, err := br.Exec(); err != nil {
			return i, fmt.Errorf("beim Exemplar mit Barcode %s: %w", bc, err)
		}
	}
	return len(barcodes), nil
}
