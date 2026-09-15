//go:build raster

package service

// Nachstellung Rasterdurchgang 15.09.2026 abends (OFFEN.md 5.15). Build-Tag raster: Die Tests
// laufen nur mit -tags raster, damit die absichtlich roten Nachstellungen keine parallele
// Sitzung und keinen Hook stören. Jeder Test beschreibt den Schaden; rot heißt „bestätigt".
// Behobene Funde sind als dauerhafte Tests umgezogen: bekannter Schlüssel und wiederholte
// Portion nach api/nachbuchen_schluessel_pg_test.go, vorgehende Theken-Uhr nach
// api/nachbuchen_uhr_pg_test.go.

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
