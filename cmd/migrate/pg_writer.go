package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const errorLogPath = "migration_errors.log"

// Schweregrad einer Protokollzeile.
//
// Die Trennung ist der Kern der ehrlichen Meldung: eine WARNUNG heißt „der Datensatz
// steht in der neuen Datenbank, aber abgewertet" (ISBN verworfen, Freitext gekürzt),
// ein FEHLER heißt „dieser Datensatz steht NICHT drin". Bisher liefen beide in denselben
// Zähler — niemand konnte unterscheiden, ob 3.000 Meldungen 3.000 verlorene Titel oder
// 3.000 kosmetische Abwertungen bedeuteten.
type severity string

const (
	sevWarn severity = "WARNUNG"
	sevFail severity = "FEHLER"
)

// errZeile markiert einen Fehler, der eindeutig nur diesen einen Datensatz betrifft und
// nicht von Postgres stammt (etwa ein nicht serialisierbares Freitextfeld). Postgres-
// Fehler werden stattdessen über ihren SQLSTATE eingeordnet, siehe istZeilenfehler.
var errZeile = errors.New("Datensatzfehler")

type errLogger struct {
	f        *os.File
	w        *bufio.Writer
	warnings int // Datensatz übernommen, aber abgewertet
	failures int // Datensatz NICHT übernommen
}

func newErrLogger() (*errLogger, error) { return newErrLoggerAt(errorLogPath) }

// newErrLoggerAt trennt den Pfad vom Aufruf, damit Tests in ein temporäres Verzeichnis
// schreiben können, statt das Arbeitsverzeichnis zu verschmutzen.
func newErrLoggerAt(pfad string) (*errLogger, error) {
	// #nosec G304 - der Pfad stammt aus errorLogPath bzw. aus dem Test
	f, err := os.OpenFile(pfad, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open error log: %w", err)
	}
	return &errLogger{f: f, w: bufio.NewWriterSize(f, 64*1024)}, nil
}

func (el *errLogger) log(sev severity, mysqlID int, isbn, reason string) {
	switch sev {
	case sevWarn:
		el.warnings++
	case sevFail:
		el.failures++
	}
	ts := time.Now().Format("2006-01-02 15:04:05")
	_, _ = fmt.Fprintf(el.w, "[%s] %s mysql_id=%d isbn=%q reason=%s\n", ts, sev, mysqlID, isbn, reason) //nolint:errcheck
}

// warn protokolliert eine Abwertung: der Datensatz kommt an, aber nicht unverändert.
func (el *errLogger) warn(mysqlID int, isbn, reason string) { el.log(sevWarn, mysqlID, isbn, reason) }

// fail protokolliert einen Ausfall: dieser Datensatz fehlt in der neuen Datenbank.
func (el *errLogger) fail(mysqlID int, isbn, reason string) { el.log(sevFail, mysqlID, isbn, reason) }

func (el *errLogger) close() {
	_ = el.w.Flush() //nolint:errcheck
	_ = el.f.Close() //nolint:errcheck
}

// istZeilenfehler entscheidet, ob ein Fehler nur diesen einen Datensatz betrifft — dann
// wird auf den Savepoint zurückgerollt und mit dem nächsten Titel weitergemacht — oder
// die gesamte Übernahme, dann wird sofort abgebrochen.
//
// Diese Unterscheidung ist die Voraussetzung dafür, dass „übersprungen" überhaupt wahr
// sein kann. Ohne sie erschiene ein Verbindungsabbruch bei Titel 300 als 79.700 einzelne
// Übersprungen-Zeilen im Protokoll: formal vollständig, inhaltlich eine Lüge.
func istZeilenfehler(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errZeile) {
		return true
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || len(pgErr.Code) < 2 {
		return false // Netzwerk- oder Treiberfehler: nicht zeilenbezogen
	}
	switch pgErr.Code[:2] {
	case "22", // Datenausnahme – z. B. 22001 Wert zu lang, 22003 Zahl außerhalb des Bereichs
		"23": // Integritätsverletzung – 23505 doppelte ISBN/Barcode, 23502 NOT NULL, 23514 CHECK
		return true
	default:
		// 08 Verbindung, 25P02 vergiftete Transaktion, 40 Deadlock, 53 Ressourcen,
		// 57 Administrator-Eingriff, XX intern — nichts davon repariert der nächste Datensatz.
		return false
	}
}

// beschreibeFehler übersetzt einen Postgres-Fehler in eine Zeile, mit der jemand die
// Quelldatei reparieren kann: SQLSTATE, Constraint und betroffene Spalte statt nur der
// nackten Meldung. Bei einer Übernahme von 80.000 Zeilen ist genau das die Arbeit —
// nicht die Laufzeit.
func beschreibeFehler(err error) string {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err.Error()
	}
	teile := []string{fmt.Sprintf("SQLSTATE %s: %s", pgErr.Code, pgErr.Message)}
	if pgErr.ConstraintName != "" {
		teile = append(teile, "Constraint "+pgErr.ConstraintName)
	}
	if pgErr.ColumnName != "" {
		teile = append(teile, "Spalte "+pgErr.ColumnName)
	}
	if pgErr.Detail != "" {
		teile = append(teile, pgErr.Detail)
	}
	// Der umschließende Kontext (z. B. der Barcode) geht sonst verloren, weil errors.As
	// bis zum PgError durchgreift.
	return err.Error() + " [" + strings.Join(teile, " | ") + "]"
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
func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// isbnRoh liefert die ISBN der Quellzeile unverändert — der Wert, den jemand in der
// Quelldatei suchen würde.
func isbnRoh(m mysqlMedium) string {
	if m.ISBN.Valid {
		return m.ISBN.String
	}
	return ""
}

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
	seenISBNs map[string]int, // isbn → mysql source ID; updated in-place
	el *errLogger,
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
// Der Savepoint ist der Kern dieser Datei. Ohne ihn war jedes `continue` in der Schleife
// wirkungslos: Postgres versetzt die Transaktion beim ersten Fehler in den Abbruchzustand
// (SQLSTATE 25P02), jedes weitere Statement scheitert nur noch daran, und der abschließende
// COMMIT wird zum ROLLBACK. Ein einziger zu langer Titel riss damit den gesamten Batch mit
// (Standard: 200 Titel) — protokolliert als 200 harmlos klingende Einzelmeldungen.
//
// Die atomare Einheit ist bewusst der TITEL, nicht das einzelne Exemplar: buecher_titel.stock
// speichert die Stückzahl aus der Quelle. Ein Titel mit stock=5, dem nur 3 Exemplare folgen,
// wäre ein stiller Bestandsfehler, den niemand je bemerkt. Entweder der Titel kommt
// vollständig an — oder gar nicht, und dann steht er als FEHLER im Protokoll.
//
// Kosten: bis zu `--batch` Subtransaktionen je Transaktion. Jenseits von 64 verliert
// Postgres den Subtransaktions-Cache und muss pg_subtrans lesen. Für ein Werkzeug, das
// einmal pro Schuljahr auf einer ansonsten ruhenden Datenbank läuft, ist das folgenlos.
func insertMediumAtomar(
	ctx context.Context,
	tx pgx.Tx,
	m mysqlMedium,
	seenISBNs map[string]int,
	el *errLogger,
	barcodeSeq *int,
) (mediumErgebnis, error) {
	sp, err := tx.Begin(ctx) // pgx bildet die verschachtelte Transaktion als SAVEPOINT ab
	if err != nil {
		return mediumErgebnis{}, fmt.Errorf("konnte den Savepoint für mysql_id=%d nicht setzen: %w", m.ID, err)
	}

	res, insErr := insertMedium(ctx, sp, m, seenISBNs, el, barcodeSeq)
	if insErr == nil {
		if err := sp.Commit(ctx); err != nil { // RELEASE SAVEPOINT
			return mediumErgebnis{}, fmt.Errorf("konnte den Savepoint für mysql_id=%d nicht freigeben: %w", m.ID, err)
		}
		return res, nil
	}

	if !istZeilenfehler(insErr) {
		return mediumErgebnis{}, fmt.Errorf(
			"abgebrochen bei mysql_id=%d (kein Datensatzfehler): %w", m.ID, insErr)
	}

	if err := sp.Rollback(ctx); err != nil { // ROLLBACK TO SAVEPOINT – hebt 25P02 wieder auf
		return mediumErgebnis{}, fmt.Errorf("konnte den Savepoint für mysql_id=%d nicht zurückrollen: %w", m.ID, err)
	}
	// Die ISBN-Reservierung lebte nur im Arbeitsspeicher und muss mit zurück. Sonst
	// verlöre der nächste echte Titel mit dieser ISBN sie an einen Datensatz, der gar
	// nicht in der Datenbank steht.
	if res.ISBN != "" {
		delete(seenISBNs, res.ISBN)
	}
	el.fail(m.ID, isbnRoh(m), "Titel übersprungen – "+beschreibeFehler(insErr))
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
	seenISBNs map[string]int,
	el *errLogger,
	barcodeSeq *int,
) (mediumErgebnis, error) {
	var res mediumErgebnis
	res.ISBN = klaerISBN(m, seenISBNs, el)

	jsonbProps, err := buildErweiterteEigenschaften(m)
	if err != nil {
		return res, fmt.Errorf("%w: JSONB konnte nicht erzeugt werden: %v", errZeile, err)
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

// klaerISBN ermittelt die ISBN, mit der der Titel geschrieben wird, und meldet jede
// Abwertung als Warnung. Rückgabe "" heißt: Titel ohne ISBN übernehmen — das ist besser,
// als ihn wegen einer kaputten Prüfziffer ganz zu verlieren.
func klaerISBN(m mysqlMedium, seenISBNs map[string]int, el *errLogger) string {
	isbnRaw := isbnRoh(m)
	normISBN, ok := validateISBN(isbnRaw)
	if isbnRaw != "" && !ok {
		el.warn(m.ID, isbnRaw, "ungültige ISBN-Prüfziffer – Titel wird ohne ISBN übernommen")
		return ""
	}
	if normISBN == "" {
		return ""
	}
	if prevID, exists := seenISBNs[normISBN]; exists {
		el.warn(m.ID, normISBN, fmt.Sprintf(
			"doppelte ISBN – bereits übernommen als mysql_id=%d; Titel wird ohne ISBN übernommen", prevID))
		return ""
	}
	seenISBNs[normISBN] = m.ID
	return normISBN
}

// Spaltenbreiten aus schema.sql. Läuft ein Freitextfeld darüber, lehnt Postgres den
// ganzen Datensatz mit SQLSTATE 22001 ab — und nennt dabei nicht einmal die Spalte, weil
// der Fehler bei der Typumwandlung entsteht. Bei einer Altbestandsübernahme aus Littera
// sind überlange Titel- und Verlagsangaben Alltag.
const (
	maxTitelSpalte     = 255 // titel, untertitel, autor, verlag
	maxMedientypSpalte = 100
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
//
// Gekürzt statt abgelehnt, im selben Geist wie die ISBN-Behandlung: der Titel kommt an,
// die Kürzung steht mit vollem Originalwert als WARNUNG im Protokoll und lässt sich
// gezielt nacharbeiten. Ein Buch wegen zwölf überzähliger Zeichen im Untertitel gar
// nicht zu übernehmen wäre der schlechtere Tausch.
func kuerzeFelder(el *errLogger, m mysqlMedium) titelFelder {
	f := titelFelder{
		Titel:      kuerzeAufSpalte(el, m, "titel", m.Titel, maxTitelSpalte),
		Untertitel: kuerzeNullable(el, m, "untertitel", m.Untertitel, maxTitelSpalte),
		Autor:      kuerzeNullable(el, m, "autor", m.Autor, maxTitelSpalte),
		Verlag:     kuerzeNullable(el, m, "verlag", m.Verlag, maxTitelSpalte),
		Medientyp:  "Buch",
	}
	if m.Medientyp.Valid && m.Medientyp.String != "" {
		f.Medientyp = kuerzeAufSpalte(el, m, "medientyp", m.Medientyp.String, maxMedientypSpalte)
	}
	return f
}

// kuerzeAufSpalte kürzt auf max ZEICHEN, nicht auf max Bytes: Postgres zählt bei VARCHAR
// Zeichen, und ein byteweiser Schnitt zerlegte deutsche Umlaute in ungültiges UTF-8.
func kuerzeAufSpalte(el *errLogger, m mysqlMedium, feld, wert string, max int) string {
	r := []rune(wert)
	if len(r) <= max {
		return wert
	}
	el.warn(m.ID, isbnRoh(m), fmt.Sprintf(
		"%s war %d Zeichen lang (Spaltenbreite %d) und wurde gekürzt; Original: %q",
		feld, len(r), max, wert))
	return string(r[:max])
}

func kuerzeNullable(el *errLogger, m mysqlMedium, feld string, s sql.NullString, max int) *string {
	v := nullableString(s)
	if v == nil {
		return nil
	}
	gekuerzt := kuerzeAufSpalte(el, m, feld, *v, max)
	return &gekuerzt
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

// insertExemplare schreibt die Exemplare eines Titels. Anders als früher bricht es beim
// ersten Fehler ab, statt weiterzuzählen: der Titel ist die atomare Einheit (siehe
// insertMediumAtomar), ein halb gefüllter Bestand wäre schlimmer als ein protokollierter
// Ausfall. Der Fehler trägt den Barcode im Text — das ist die Angabe, mit der jemand die
// Kollision in der Quelldatei tatsächlich findet.
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
				"%w: erzeugter Barcode %q entspricht nicht dem Muster B-<Ziffern>", errZeile, bc)
		}
	}

	batch := &pgx.Batch{}
	for _, bc := range barcodes {
		batch.Queue(sqlInsertExemplar, data.TitelID, bc, data.ErstelltAm)
	}

	br := tx.SendBatch(ctx, batch)
	defer br.Close()

	for i, bc := range barcodes {
		if _, err := br.Exec(); err != nil {
			return i, fmt.Errorf("beim Exemplar mit Barcode %s: %w", bc, err)
		}
	}
	return len(barcodes), nil
}
