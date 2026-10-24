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

const fehlerprotokollPfad = "migration_errors.log"

// Protokoll, Fehlerklassen, ISBN und Savepoints dieses Werkzeugs stehen in internal/uebernahme,
// gemeinsam mit der Übernahme aus Littera. Sie zweimal zu halten hieße, dass die zweite Fassung
// dieselben Fehler noch einmal macht.
func newFehlerprotokoll() (*uebernahme.Protokoll, error) {
	return newFehlerprotokollIn(fehlerprotokollPfad)
}

// newFehlerprotokollIn trennt den Pfad vom Aufruf, damit Tests in ein vorübergehendes
// Verzeichnis schreiben können, statt das Arbeitsverzeichnis zu verschmutzen.
func newFehlerprotokollIn(pfad string) (*uebernahme.Protokoll, error) {
	return uebernahme.NewProtokoll(pfad, "mysql_id")
}

// baueErweiterteEigenschaften schreibt die Freitextfelder als JSON für die Spalte JSONB.
func baueErweiterteEigenschaften(m mysqlMedium) (string, error) {
	eigenschaften := make(map[string]string)
	if m.Standort.Valid && m.Standort.String != "" {
		eigenschaften["standort"] = m.Standort.String
	}
	if m.Regal.Valid && m.Regal.String != "" {
		eigenschaften["regal"] = m.Regal.String
	}
	if m.Notizen.Valid && m.Notizen.String != "" {
		eigenschaften["notizen"] = m.Notizen.String
	}
	// Die Kennung der Quelle, damit sich jede Zeile zurückverfolgen lässt.
	eigenschaften["mysql_id"] = strconv.Itoa(m.ID)

	b, err := json.Marshal(eigenschaften)
	if err != nil {
		return "{}", fmt.Errorf("json marshal: %w", err)
	}
	return string(b), nil
}

// nullbareZahl macht aus einer NULL von MySQL ein nil für pgx.
func nullbareZahl(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

// nullbarerText macht aus einem leeren Text ein typisiertes nil für Spalten, die NULL erlauben.
func nullbarerText(s string) *string { return uebernahme.Nullbar(s) }

// isbnRoh liefert die ISBN der Quellzeile unverändert — der Wert, den jemand in der
// Quelldatei suchen würde.
func isbnRoh(m mysqlMedium) string {
	if m.ISBN.Valid {
		return m.ISBN.String
	}
	return ""
}

// quellID ist die Kennung der Quellzeile als Text, wie sie im Protokoll steht.
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

// legeBatchAn schreibt einen Batch Titel in einer Transaktion. Jeder einzelne Titel ist darin
// durch einen Savepoint abgesichert (legeMediumAtomarAn), sodass ein fehlerhafter Datensatz nur
// sich selbst kostet.
//
// Ein zurückgegebener Fehler heißt: Die Übernahme ist zu Ende. Dann steht aus diesem Batch
// nichts in der Datenbank (der defer rollt zurück), und die Zahlen sagen das auch.
func legeBatchAn(
	ctx context.Context,
	pool *pgxpool.Pool,
	teil []mysqlMedium,
	bekannteISBNs map[string]string, // ISBN → Kennung der Quelle; wird hier fortgeschrieben
	fehlerprotokoll *uebernahme.Protokoll,
	barcodeNummer *int,
) (batchErgebnis, error) {
	var ergebnis batchErgebnis

	tx, err := pool.Begin(ctx)
	if err != nil {
		return batchErgebnis{}, fmt.Errorf("konnte die Transaktion nicht öffnen: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	for _, m := range teil {
		medium, err := legeMediumAtomarAn(ctx, tx, m, bekannteISBNs, fehlerprotokoll, barcodeNummer)
		if err != nil {
			return batchErgebnis{}, err
		}
		ergebnis.Titel += medium.Titel
		ergebnis.Exemplare += medium.Exemplare
		if medium.Titel == 0 {
			ergebnis.Uebersprungen++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return batchErgebnis{}, fmt.Errorf(
			"der COMMIT des Batches schlug fehl – KEIN Datensatz dieses Batches wurde geschrieben, "+
				"auch nicht die zuvor als übernommen protokollierten: %w", err)
	}
	return ergebnis, nil
}

// legeMediumAtomarAn schreibt einen Titel samt allen seinen Exemplaren in einem SAVEPOINT.
//
// Die atomare Einheit ist der Titel, nicht das einzelne Exemplar: Die Stückzahl der Quelle wird
// vollständig zu Zeilen in buecher_exemplare. Ein Titel mit 5 Stück, dem nur 3 Exemplare folgen,
// wäre ein unbemerkter Fehler im Bestand. Entweder der Titel kommt vollständig an oder gar nicht,
// und dann steht er als FEHLER im Protokoll.
func legeMediumAtomarAn(
	ctx context.Context,
	tx pgx.Tx,
	m mysqlMedium,
	bekannteISBNs map[string]string,
	fehlerprotokoll *uebernahme.Protokoll,
	barcodeNummer *int,
) (mediumErgebnis, error) {
	var ergebnis mediumErgebnis
	savepoint, err := uebernahme.ImSavepoint(ctx, tx, "mysql_id="+quellID(m), func(sp pgx.Tx) error {
		var anlegeFehler error
		ergebnis, anlegeFehler = legeMediumAn(ctx, sp, m, bekannteISBNs, fehlerprotokoll, barcodeNummer)
		return anlegeFehler
	})
	if err != nil {
		return mediumErgebnis{}, err
	}
	if savepoint.Uebernommen {
		return ergebnis, nil
	}

	// Die ISBN-Reservierung lebte nur im Arbeitsspeicher und muss mit zurück. Sonst
	// verlöre der nächste echte Titel mit dieser ISBN sie an einen Datensatz, der gar
	// nicht in der Datenbank steht.
	if ergebnis.ISBN != "" {
		delete(bekannteISBNs, ergebnis.ISBN)
	}
	fehlerprotokoll.Fehler(quellID(m), isbnRoh(m), "Titel übersprungen – "+uebernahme.BeschreibeFehler(savepoint.Zurueckgerollt))
	return mediumErgebnis{}, nil
}

const sqlLegeTitelAn = `
	INSERT INTO buecher_titel
		(titel, untertitel, autor, isbn, verlag, erscheinungsjahr,
		 medientyp, erweiterte_eigenschaften,
		 erstellt_am)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	RETURNING id`

// legeMediumAn schreibt Titel und Exemplare innerhalb des übergebenen Savepoints. Abwertungen
// (ungültige ISBN, Dublette, gekürzter Freitext) stehen hier als Warnung im Protokoll; harte
// Fehler gehen unverändert nach oben, damit legeMediumAtomarAn sie einordnen kann.
func legeMediumAn(
	ctx context.Context,
	tx pgx.Tx,
	m mysqlMedium,
	bekannteISBNs map[string]string,
	fehlerprotokoll *uebernahme.Protokoll,
	barcodeNummer *int,
) (mediumErgebnis, error) {
	var ergebnis mediumErgebnis
	ergebnis.ISBN = uebernahme.KlaereISBN(fehlerprotokoll, quellID(m), isbnRoh(m), bekannteISBNs)

	eigenschaften, err := baueErweiterteEigenschaften(m)
	if err != nil {
		return ergebnis, fmt.Errorf("%w: JSONB konnte nicht erzeugt werden: %w", uebernahme.ErrZeile, err)
	}

	felder := kuerzeFelder(fehlerprotokoll, m)

	erstelltAm := time.Now()
	if m.ErstelltAm.Valid {
		erstelltAm = m.ErstelltAm.Time
	}

	var titelID string
	err = tx.QueryRow(ctx, sqlLegeTitelAn,
		felder.Titel,
		felder.Untertitel,
		felder.Autor,
		nullbarerText(ergebnis.ISBN),
		felder.Verlag,
		nullbareZahl(m.Erscheinungsjahr),
		felder.Medientyp,
		eigenschaften,
		erstelltAm,
	).Scan(&titelID)
	if err != nil {
		return ergebnis, fmt.Errorf("beim Titel %q: %w", felder.Titel, err)
	}
	ergebnis.Titel = 1

	ergebnis.Exemplare, err = legeExemplareAn(ctx, tx,
		exemplarAnlage{TitelID: titelID, Medium: m, ErstelltAm: erstelltAm}, barcodeNummer)
	if err != nil {
		return ergebnis, err
	}
	return ergebnis, nil
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
func kuerzeFelder(fehlerprotokoll *uebernahme.Protokoll, m mysqlMedium) titelFelder {
	id, isbn := quellID(m), isbnRoh(m)
	f := titelFelder{
		Titel:      uebernahme.Kuerze(uebernahme.Kuerzung{Protokoll: fehlerprotokoll, QuellID: id, Kennung: isbn, Feld: "titel", Wert: m.Titel, Max: maxTitelSpalte}),
		Untertitel: uebernahme.KuerzeNullbar(uebernahme.Kuerzung{Protokoll: fehlerprotokoll, QuellID: id, Kennung: isbn, Feld: "untertitel", Wert: m.Untertitel.String, Max: maxTitelSpalte}),
		Autor:      uebernahme.KuerzeNullbar(uebernahme.Kuerzung{Protokoll: fehlerprotokoll, QuellID: id, Kennung: isbn, Feld: "autor", Wert: m.Autor.String, Max: maxTitelSpalte}),
		Verlag:     uebernahme.KuerzeNullbar(uebernahme.Kuerzung{Protokoll: fehlerprotokoll, QuellID: id, Kennung: isbn, Feld: "verlag", Wert: m.Verlag.String, Max: maxTitelSpalte}),
		Medientyp:  "Buch",
	}
	if m.Medientyp.Valid && m.Medientyp.String != "" {
		f.Medientyp = uebernahme.Kuerze(uebernahme.Kuerzung{Protokoll: fehlerprotokoll, QuellID: id, Kennung: isbn, Feld: "medientyp", Wert: m.Medientyp.String, Max: maxMedientypSpalte})
	}
	return f
}

// exemplarInsert bündelt die Daten eines Titels, dessen Exemplare geschrieben werden.
type exemplarAnlage struct {
	TitelID    string
	Medium     mysqlMedium
	ErstelltAm time.Time
}

const sqlLegeExemplarAn = `
	INSERT INTO buecher_exemplare
		(titel_id, barcode_id, erworben_am, ist_ausleihbar,
		 erweiterte_eigenschaften, erstellt_am)
	VALUES ($1, $2, CURRENT_DATE, true, '{}', $3)`

// legeExemplareAn schreibt die Exemplare eines Titels. Es bricht beim ersten Fehler ab, statt
// weiterzuzählen: Der Titel ist die atomare Einheit (legeMediumAtomarAn), ein halb gefüllter
// Bestand wäre schlimmer als ein protokollierter Ausfall. Der Fehler trägt den Barcode im Text;
// das ist die Angabe, mit der jemand die Kollision in der Quelldatei wirklich findet.
//
// Der Zähler wird auch im Fehlerfall weitergestellt. Die verbrannten Nummern sind folgenlos:
// Barcodes sind undurchsichtige Etiketten, Lücken darin bedeuten nichts.
// Gäbe man sie zurück, liefe der nächste Titel in dieselbe Kollision und die Übernahme
// bliebe an einer einzigen belegten Nummer hängen, Datensatz für Datensatz.
func legeExemplareAn(ctx context.Context, tx pgx.Tx, anlage exemplarAnlage, barcodeNummer *int) (int, error) {
	barcodes := naechsteBarcodes(*barcodeNummer, anlage.Medium.Anzahl)
	*barcodeNummer += anlage.Medium.Anzahl

	for _, barcode := range barcodes {
		if !istBarcodeGueltig(barcode) {
			return 0, fmt.Errorf(
				"%w: erzeugter Barcode %q entspricht nicht dem Muster B-<Ziffern>", uebernahme.ErrZeile, barcode)
		}
	}

	stapel := &pgx.Batch{}
	for _, barcode := range barcodes {
		stapel.Queue(sqlLegeExemplarAn, anlage.TitelID, barcode, anlage.ErstelltAm)
	}

	ergebnisse := tx.SendBatch(ctx, stapel)
	defer func() { _ = ergebnisse.Close() }() //nolint:errcheck

	for i, barcode := range barcodes {
		if _, err := ergebnisse.Exec(); err != nil {
			return i, fmt.Errorf("beim Exemplar mit Barcode %s: %w", barcode, err)
		}
	}
	return len(barcodes), nil
}
