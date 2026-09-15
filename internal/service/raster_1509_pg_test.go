//go:build raster

package service

// Nachstellung Rasterdurchgang 15.09.2026 abends (OFFEN.md 5.15). Build-Tag raster: Die Tests
// laufen nur mit -tags raster, damit die absichtlich roten Nachstellungen keine parallele
// Sitzung und keinen Hook stören. Jeder Test beschreibt den Schaden; rot heißt „bestätigt".
// Behobene Funde sind als dauerhafte Tests umgezogen (bekannter Schlüssel und wiederholte
// Portion: api/nachbuchen_schluessel_pg_test.go).

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"bibliothek/repository"

	"github.com/google/uuid"
)

// onlineFremdrueckgabe bucht wie der erste Online-Scan eines Buchs, das auf jemand anderem
// steht (handleForeignReturn): NUR die Rücknahme dort, eigene Transaktion — kein Umbuchen
// (Produktentscheidung 10.07.).
func (w *nbWelt) onlineFremdrueckgabe(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	loans := repository.NewLoanRepository(w.pool)
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	aktiv, err := loans.GetActiveLoanByCopyIDTx(ctx, tx, w.exemplarID)
	if err != nil || aktiv == nil {
		t.Fatalf("aktive Ausleihe für die Fremdrückgabe: %v %v", aktiv, err)
	}
	if err := loans.ReturnLoanTx(ctx, tx, aktiv.ID, w.staff, true); err != nil {
		t.Fatalf("Fremdrückgabe: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

// onlineAusleihe bucht wie die Online-Theke: Steht das Buch auf jemand anderem, nimmt der
// erste Scan es dort zurück (onlineFremdrueckgabe), der zweite leiht aus (handleNewLoan) —
// zwei Transaktionen, zwei Stempel.
func (w *nbWelt) onlineAusleihe(t *testing.T, schueler string) string {
	t.Helper()
	ctx := context.Background()
	loans := repository.NewLoanRepository(w.pool)
	if n, _ := w.offeneAusleihen(t); n > 0 {
		w.onlineFremdrueckgabe(t)
	}
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	loan, err := loans.CreateLoanTx(ctx, tx, w.exemplarID, schueler, w.staff, time.Now().AddDate(0, 0, 14))
	if err != nil {
		t.Fatalf("ausleihen: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	return loan.ID
}

func (w *nbWelt) meldungenDerSchluessel(t *testing.T, schluessel ...string) []string {
	t.Helper()
	rows, err := w.pool.Query(context.Background(),
		`SELECT ergebnis || ': ' || coalesce(grund, '') FROM nachbuch_meldungen WHERE idempotency_key = ANY($1::uuid[]) ORDER BY erstellt_am`, schluessel)
	if err != nil {
		t.Fatalf("Meldungen lesen: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			t.Fatalf("Meldung: %v", err)
		}
		out = append(out, m)
	}
	return out
}

// Verdacht A: Die Uhr von Theke 1 geht vor. Anna gibt dort offline zurück; danach leiht Ben
// dasselbe Buch online an Theke 2. Beim Nachbuchen liegt der Scan-Zeitpunkt (Theken-Uhr,
// gekappt auf Serverzeit) nach Bens Ausleihe — die Rückgabe trifft Bens Ausleihe.
func TestRaster_VorgehendeThekenUhrBeendetJuengereAusleihe(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()

	w.onlineAusleihe(t, w.anna)

	echteRueckgabe := time.Now()
	thekenUhr := echteRueckgabe.Add(15 * time.Minute) // Theke 1 geht 15 Minuten vor
	time.Sleep(50 * time.Millisecond)

	benLoan := w.onlineAusleihe(t, w.ben) // nach der echten Rückgabe, an Theke 2

	schluessel := uuid.NewString()
	erg, err := w.svc.Nachbuchen(ctx, NachbuchEintrag{
		Schluessel: schluessel, Absicht: NachbuchAbsichtRueckgabe, Barcode: w.code,
		GescanntAm: thekenUhr, StaffID: w.staff,
	})
	if err != nil {
		t.Fatalf("nachbuchen: %v", err)
	}
	var benOffen bool
	if err := w.pool.QueryRow(ctx, `SELECT rueckgabe_am IS NULL FROM ausleihen WHERE id = $1`, benLoan).Scan(&benOffen); err != nil {
		t.Fatalf("Bens Ausleihe: %v", err)
	}
	meldungen := w.meldungenDerSchluessel(t, schluessel)
	t.Logf("Ergebnis %q · Bens Ausleihe offen: %v · Meldungen: %v", erg.Ergebnis, benOffen, meldungen)
	if !benOffen {
		t.Errorf("NACHGESTELLT: Ben hat das Buch in der Hand, seine Ausleihe ist beendet (Ergebnis %q, Meldungen %v)", erg.Ergebnis, meldungen)
	}
}

// litteraEtikett baut den EAN-13 eines Littera-Etiketts (internal/littera.EtikettBarcode,
// Bibliotheksnummer 395) — die Umkehrung von dekodiereLitteraEtikett.
func litteraEtikett(nummer string) string {
	payload := nummer + strings.Repeat("0", 8-len(nummer)) + "395" + strconv.Itoa(len(nummer))
	summe := 0
	for i := 0; i < 12; i++ {
		z := int(payload[i] - '0')
		if i%2 == 1 {
			z *= 3
		}
		summe += z
	}
	return payload + strconv.Itoa((10-summe%10)%10)
}

// Verdacht B (zwei): Die Barcode-Liste der Theke gegen das, was der Server beim Nachbuchen
// als Buch erkennt — ein Littera-Etikett auf einem Exemplar mit nackter Mediennummer, und
// ein ausgesondertes Exemplar mit Ziffern-Barcode.
func TestRaster_BarcodeListeGegenNachbuchTuer(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	svc := w.svc.(*defaultLoanService)
	nanos := time.Now().UnixNano()

	nummer := fmt.Sprintf("%d", 1000000+nanos%8999999) // 7 Stellen, erste ≠ 0
	etikett := litteraEtikett(nummer)
	if n, ok := dekodiereLitteraEtikett(etikett); !ok || n != nummer {
		t.Fatalf("Etikett-Formel: %s → %q %v", etikett, n, ok)
	}
	var litteraID string
	if err := w.pool.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, einkaufspreis)
		VALUES ($1, $2, true, 9.00) RETURNING id`, w.titelID, nummer).Scan(&litteraID); err != nil {
		t.Fatalf("Exemplar mit Mediennummer: %v", err)
	}
	gefunden, err := svc.loeseExemplar(ctx, etikett)
	if err != nil || gefunden == nil || gefunden.ID != litteraID {
		t.Fatalf("der Server erkennt das Etikett nicht: %v %v", gefunden, err)
	}

	abgeschrieben := fmt.Sprintf("%d", 20000000+nanos%79999999) // 8 Stellen
	if _, err := w.pool.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, ist_ausgesondert, aussonderung_grund, einkaufspreis)
		VALUES ($1, $2, false, true, 'VERLUST', 9.00)`, w.titelID, abgeschrieben); err != nil {
		t.Fatalf("ausgesondertes Exemplar: %v", err)
	}

	liste, err := repository.ListeBuchbarcodes(ctx, w.pool, 0)
	if err != nil {
		t.Fatalf("Liste: %v", err)
	}
	auf := map[string]bool{}
	for _, b := range liste {
		auf[b] = true
	}
	erg, err := w.svc.Nachbuchen(ctx, NachbuchEintrag{
		Schluessel: uuid.NewString(), Absicht: NachbuchAbsichtRueckgabe, Barcode: abgeschrieben,
		GescanntAm: time.Now(), StaffID: w.staff,
	})
	if err != nil {
		t.Fatalf("nachbuchen: %v", err)
	}
	t.Logf("Etikett %s auf der Liste: %v (Mediennummer %s: %v) · ausgesondert %s auf der Liste: %v, Nachbuchen → %q",
		etikett, auf[etikett], nummer, auf[nummer], abgeschrieben, auf[abgeschrieben], erg.Ergebnis)
	if !auf[etikett] {
		t.Errorf("NACHGESTELLT: Das gescannte Etikett %s fehlt in der Liste (sie führt %s) — der Server nimmt es als Buch, die Theke hielte es offline für unklar", etikett, nummer)
	}
	if erg.Ergebnis == repository.NachbuchNurReaktiviert && !auf[abgeschrieben] {
		t.Errorf("NACHGESTELLT: Das Nachbuchen holt das ausgesonderte Exemplar %s zurück, die Liste kennt es nicht", abgeschrieben)
	}
}

// Gegenprobe zur Uhr: dieselbe Lage mit richtig gehender Theken-Uhr — der Wächter muss greifen.
// Bleibt diese Probe grün, ist der Uhrversatz (und nichts anderes) die Ursache.
func TestRaster_Gegenprobe_RichtigeUhr(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	w.onlineAusleihe(t, w.anna)
	echteRueckgabe := time.Now()
	time.Sleep(50 * time.Millisecond)
	benLoan := w.onlineAusleihe(t, w.ben)
	erg, err := w.svc.Nachbuchen(ctx, NachbuchEintrag{
		Schluessel: uuid.NewString(), Absicht: NachbuchAbsichtRueckgabe, Barcode: w.code,
		GescanntAm: echteRueckgabe, StaffID: w.staff,
	})
	if err != nil {
		t.Fatalf("nachbuchen: %v", err)
	}
	var benOffen bool
	if err := w.pool.QueryRow(ctx, `SELECT rueckgabe_am IS NULL FROM ausleihen WHERE id = $1`, benLoan).Scan(&benOffen); err != nil {
		t.Fatalf("Bens Ausleihe: %v", err)
	}
	t.Logf("richtige Uhr: Ergebnis %q · Bens Ausleihe offen: %v", erg.Ergebnis, benOffen)
	if erg.Ergebnis != repository.NachbuchVeraltet || !benOffen {
		t.Errorf("Gegenprobe: erwartet veraltet und Bens Ausleihe offen, bekam %q, offen=%v", erg.Ergebnis, benOffen)
	}
}
