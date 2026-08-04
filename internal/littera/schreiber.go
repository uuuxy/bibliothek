package littera

import (
	"context"
	"fmt"
	"time"

	"bibliothek/internal/uebernahme"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Barcodequelle bestimmt, was in buecher_exemplare.barcode_id landet.
type Barcodequelle string

const (
	// BarcodeLittera übernimmt Litteras Exemplarnummer. Die Etiketten, die heute auf den
	// Büchern kleben, tragen genau diese Nummer (belegt in BarcodeInhalt: 61.520 von
	// 61.520 rekonstruiert) — der Bestand bleibt damit ohne Neubeklebung scannbar,
	// vorausgesetzt die Lesegeräte liefern den Zifferninhalt.
	BarcodeLittera Barcodequelle = "littera"
	// BarcodeNeu vergibt B-XXXXX aus der Postgres-Sequenz barcode_seq — dieselbe Quelle,
	// aus der auch die Anwendung ihre Barcodes zieht (repository/book_inventory.go).
	// Damit sind Kollisionen ausgeschlossen, aber alle Bücher brauchen neue Etiketten.
	BarcodeNeu Barcodequelle = "neu"
)

// Optionen steuern den Lauf. Die Vorgaben stehen in StandardOptionen.
type Optionen struct {
	Barcodes      Barcodequelle
	BatchGroesse  int  // Titel je Transaktion
	SchuljahrEnde int  // Jahr, in dem das laufende Schuljahr endet (2026/27 → 2027)
	LehrerAktiv   bool // Lehrkräfte als aktive Benutzer anlegen
	Jetzt         time.Time
}

// StandardOptionen liefert die Vorgaben für einen Lauf zum Zeitpunkt jetzt.
//
// Das Schuljahr endet im Sommer: Ab August zählt bereits das nächste Kalenderjahr.
// Dieselbe Auslegung wie im Versetzungslauf.
func StandardOptionen(jetzt time.Time) Optionen {
	schuljahrEnde := jetzt.Year()
	if jetzt.Month() >= time.August {
		schuljahrEnde++
	}
	return Optionen{
		Barcodes:      BarcodeLittera,
		BatchGroesse:  200,
		SchuljahrEnde: schuljahrEnde,
		LehrerAktiv:   false,
		Jetzt:         jetzt,
	}
}

// Schreiber überträgt einen gelesenen Altbestand nach PostgreSQL.
//
// Die Härtung steckt in internal/uebernahme und ist dieselbe, die cmd/migrate benutzt:
// ein Savepoint je Datensatz, Postgres-Fehler nach SQLSTATE eingeordnet, Abwertungen
// getrennt von Ausfällen protokolliert.
type Schreiber struct {
	pool *pgxpool.Pool
	prot *uebernahme.Protokoll
	opt  Optionen
}

func NeuerSchreiber(pool *pgxpool.Pool, prot *uebernahme.Protokoll, opt Optionen) *Schreiber {
	if opt.BatchGroesse <= 0 {
		opt.BatchGroesse = 200
	}
	if opt.Barcodes == "" {
		opt.Barcodes = BarcodeLittera
	}
	if opt.Jetzt.IsZero() {
		opt.Jetzt = time.Now()
	}
	return &Schreiber{pool: pool, prot: prot, opt: opt}
}

// PruefeZielbestand bricht ab, wenn schon einmal ein Littera-Import gelaufen ist.
//
// Ein zweiter Lauf legt jeden Titel und jedes Exemplar ein weites Mal an — es gibt
// keinen natürlichen Schlüssel, an dem Postgres das verhindern könnte. Der Bestand wäre
// danach doppelt und ließe sich nur noch über die Reihenfolge der UUIDs auseinander­
// sortieren. Deshalb wird vorher gezählt, statt hinterher zu erklären.
func (s *Schreiber) PruefeZielbestand(ctx context.Context) error {
	var titel, exemplare, schueler int
	err := s.pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM buecher_titel      WHERE erweiterte_eigenschaften ? 'littera_id'),
		       (SELECT count(*) FROM buecher_exemplare  WHERE erweiterte_eigenschaften ? 'littera_id'),
		       (SELECT count(*) FROM schueler           WHERE lusd_id LIKE 'littera:%')
	`).Scan(&titel, &exemplare, &schueler)
	if err != nil {
		return fmt.Errorf("konnte den Zielbestand nicht prüfen: %w", err)
	}
	if titel+exemplare+schueler == 0 {
		return nil
	}
	return fmt.Errorf(
		"in der Zieldatenbank stehen bereits %d Titel, %d Exemplare und %d Schüler aus einem "+
			"früheren Littera-Import. Ein zweiter Lauf legt sie ein weiteres Mal an. "+
			"Erst aufräumen (siehe docs/SCRIPTS.md), dann erneut starten",
		titel, exemplare, schueler)
}

// Bericht ist die Bilanz eines vollständigen Laufs.
//
// Gemeldet und Ist werden getrennt geführt und am Ende gegeneinander gehalten: Eine Zahl,
// die niemand gegen die Datenbank geprüft hat, ist keine Bilanz, sondern eine Behauptung.
type Bericht struct {
	Bestand   BestandBericht
	Personen  PersonenBericht
	Ausleihen AusleihBericht

	Warnungen int
	Fehler    int
	Abbruch   error
}

// Vollstaendig sagt, ob jeder Quelldatensatz angekommen ist.
func (b Bericht) Vollstaendig() bool {
	return b.Abbruch == nil && b.Fehler == 0 &&
		b.Bestand.AbgleichOK && b.Personen.AbgleichOK && b.Ausleihen.AbgleichOK
}

// zaehle liest einen einzelnen Zählwert.
func (s *Schreiber) zaehle(ctx context.Context, sql string) (int, error) {
	var n int
	if err := s.pool.QueryRow(ctx, sql).Scan(&n); err != nil {
		return 0, fmt.Errorf("eine Zählung schlug fehl: %w", err)
	}
	return n, nil
}
