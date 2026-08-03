package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/repository"
)

// TestSignaturScope_PraefixMitGrenze ist der Kern-Regressionstest der Signatur-Umstellung.
//
// Vorher filterte der Inventur-Scope auf buecher_titel.signature_id — eine Spalte, die
// ausser Migration 021 nie jemand geschrieben hat. Eine Signatur-Inventur traf damit NULL
// Exemplare, und zwar unauffaellig: Sie startete, meldete "0 erwartet" und schloss ohne
// Befund ab. Der Test belegt jetzt am Verhalten, dass der Scope die richtigen Buecher
// trifft — und, genauso wichtig, die falschen NICHT.
//
// Die Signatur wirkt als Praefix mit Grenze am Leerzeichen: "BIB Deu" meint das Regal
// samt Unterfaechern, "BIB De" ist eine andere Adresse und darf nicht hineinreichen.
func TestSignaturScope_PraefixMitGrenze(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	resetBestandsdaten(t, pool)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	basis := "BIB Deu " + suffix

	// Im Regal: die Signatur selbst und ein Unterfach darunter.
	exGenau := seedSignaturMitExemplar(t, pool, basis, "SIG-GENAU-"+suffix)
	exUnter := seedSignaturMitExemplar(t, pool, basis+" 5 KRUE", "SIG-UNTER-"+suffix)
	// NICHT im Regal: gleiche Zeichenfolge, aber ohne Grenze am Leerzeichen.
	exNachbar := seedSignaturMitExemplar(t, pool, basis+"X", "SIG-NACHBAR-"+suffix)
	// NICHT im Regal: voellig andere Signatur.
	exFremd := seedSignaturMitExemplar(t, pool, "LMF Mat "+suffix, "SIG-FREMD-"+suffix)

	invRepo := repository.NewInventoryRepository(pool)
	scope := repository.InventurScope{Signatur: &basis}

	anzahl, err := invRepo.ZaehleScope(ctx, scope)
	if err != nil {
		t.Fatalf("ZaehleScope: %v", err)
	}
	if anzahl != 2 {
		t.Errorf("Scope %q: erwartet 2 Exemplare (genau + Unterfach), waren %d", basis, anzahl)
	}

	imScope := map[string]bool{exGenau: true, exUnter: true, exNachbar: false, exFremd: false}
	for exID, erwartet := range imScope {
		drin, err := invRepo.ExemplarImScope(ctx, exID, scope)
		if err != nil {
			t.Fatalf("ExemplarImScope %s: %v", exID, err)
		}
		if drin != erwartet {
			t.Errorf("Exemplar %s im Scope %q: erwartet %v, war %v", exID, basis, erwartet, drin)
		}
	}
}

// TestSignaturenEndpunkte belegt die beiden neuen Endpunkte am echten Bestand:
// die Signaturliste kommt aus buecher_titel.signatur (nicht aus einer Stammtabelle,
// die keine Buecher kannte), und die Regalansicht liest dieselbe Praefix-Grenze wie
// der Inventur-Scope. Liefen die beiden auseinander, wuerde man etwas anderes
// inventarisieren, als die Liste anzeigt.
func TestSignaturenEndpunkte(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	basis := "BIB Gesch " + suffix
	seedSignaturMitExemplar(t, pool, basis, "SIGE-A-"+suffix)
	seedSignaturMitExemplar(t, pool, basis+" 7", "SIGE-B-"+suffix)
	seedSignaturMitExemplar(t, pool, basis+"X", "SIGE-C-"+suffix)

	srv := &Server{DB: &db.Database{Pool: pool}}

	// 1) Liste: die drei angelegten Signaturen tauchen als eigene Gruppen auf.
	rec := httptest.NewRecorder()
	srv.GetSignaturenHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/signaturen", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Signaturliste: Status %d", rec.Code)
	}
	var gruppen []SignaturGruppe
	if err := json.Unmarshal(rec.Body.Bytes(), &gruppen); err != nil {
		t.Fatalf("Signaturliste unlesbar: %v", err)
	}
	gefunden := map[string]int{}
	for _, g := range gruppen {
		gefunden[g.Signatur] = g.Exemplare
	}
	for _, erwartet := range []string{basis, basis + " 7", basis + "X"} {
		if gefunden[erwartet] != 1 {
			t.Errorf("Signatur %q: erwartet 1 Exemplar in der Liste, war %d", erwartet, gefunden[erwartet])
		}
	}

	// 2) Regalansicht: Praefix trifft Basis + Unterfach, aber nicht den Nachbarn ohne Grenze.
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/signaturen/buecher?signatur="+url.QueryEscape(basis), nil)
	srv.GetSignaturBuecherHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Regalansicht: Status %d", rec.Code)
	}
	var regal SignaturBuecherResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &regal); err != nil {
		t.Fatalf("Regalansicht unlesbar: %v", err)
	}
	if len(regal.Buecher) != 2 {
		t.Fatalf("Regal %q: erwartet 2 Titel (Basis + Unterfach), waren %d", basis, len(regal.Buecher))
	}
	for _, b := range regal.Buecher {
		if b.Signatur == basis+"X" {
			t.Errorf("Nachbarsignatur %q darf nicht im Regal %q stehen", b.Signatur, basis)
		}
	}
	// Regalreihenfolge: danach laeuft man das Regal ab.
	if regal.Buecher[0].Signatur != basis {
		t.Errorf("Regal nicht nach Signatur sortiert: erste Zeile %q", regal.Buecher[0].Signatur)
	}

	// 3) Leere Signatur wird abgewiesen — sie waere Praefix von allem.
	rec = httptest.NewRecorder()
	srv.GetSignaturBuecherHandler().ServeHTTP(rec,
		httptest.NewRequest(http.MethodGet, "/api/signaturen/buecher?signatur=%20", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("leere Signatur: erwartet 400, war %d", rec.Code)
	}
}
