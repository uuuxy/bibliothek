package katalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/littera"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// TestSignaturBereich_PraefixMitGrenze prüft am Verhalten, dass der Bereich einer Inventur nach
// Signatur die richtigen Bücher trifft und die falschen nicht. Ein Bereich, der nichts trifft,
// fällt sonst nicht auf: Die Inventur startet, meldet „0 erwartet" und schließt ohne Befund ab.
//
// Die Signatur wirkt als Anfang mit Grenze am Leerzeichen: „BIB Deu" meint das Regal samt
// Unterfächern, „BIB De" ist eine andere Adresse und darf nicht hineinreichen.
func TestSignaturBereich_PraefixMitGrenze(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	testdaten.LeereBestandsdaten(t, pool)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	basis := "BIB Deu " + suffix

	// Im Regal: die Signatur selbst und ein Unterfach darunter.
	exGenau := testdaten.LegeExemplarMitSignaturAn(t, pool, basis, "SIG-GENAU-"+suffix)
	exUnter := testdaten.LegeExemplarMitSignaturAn(t, pool, basis+" 5 KRUE", "SIG-UNTER-"+suffix)
	// Nicht im Regal: gleiche Zeichenfolge, aber ohne Grenze am Leerzeichen.
	exNachbar := testdaten.LegeExemplarMitSignaturAn(t, pool, basis+"X", "SIG-NACHBAR-"+suffix)
	// Nicht im Regal: eine völlig andere Signatur.
	exFremd := testdaten.LegeExemplarMitSignaturAn(t, pool, "LMF Mat "+suffix, "SIG-FREMD-"+suffix)

	invRepo := repository.NewInventurAbfragen(pool)
	scope := repository.InventurBereich{Signatur: &basis}

	anzahl, err := invRepo.ZaehleBereich(ctx, scope)
	if err != nil {
		t.Fatalf("ZaehleBereich: %v", err)
	}
	if anzahl != 2 {
		t.Errorf("Scope %q: erwartet 2 Exemplare (genau + Unterfach), waren %d", basis, anzahl)
	}

	imScope := map[string]bool{exGenau: true, exUnter: true, exNachbar: false, exFremd: false}
	for exID, erwartet := range imScope {
		drin, err := invRepo.IstExemplarImBereich(ctx, exID, scope)
		if err != nil {
			t.Fatalf("IstExemplarImBereich %s: %v", exID, err)
		}
		if drin != erwartet {
			t.Errorf("Exemplar %s im Scope %q: erwartet %v, war %v", exID, basis, erwartet, drin)
		}
	}
}

// TestSignaturenEndpunkte belegt die beiden Türen am echten Bestand: Die Liste der Signaturen
// kommt aus buecher_titel.signatur, und die Regalansicht liest dieselbe Grenze am Anfang wie der
// Bereich einer Inventur. Liefen die beiden auseinander, zählte die Inventur etwas anderes, als
// die Liste anzeigt.
func TestSignaturenEndpunkte(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	basis := "BIB Gesch " + suffix
	testdaten.LegeExemplarMitSignaturAn(t, pool, basis, "SIGE-A-"+suffix)
	testdaten.LegeExemplarMitSignaturAn(t, pool, basis+" 7", "SIGE-B-"+suffix)
	testdaten.LegeExemplarMitSignaturAn(t, pool, basis+"X", "SIGE-C-"+suffix)

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}

	// 1) Liste: die drei angelegten Signaturen tauchen als eigene Gruppen auf.
	rec := httptest.NewRecorder()
	srv.Signaturen().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/signaturen", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Signaturliste: Status %d", rec.Code)
	}
	var gruppen []signaturGruppe
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

	// 2) Regalansicht: Der Anfang trifft Basis und Unterfach, aber nicht den Nachbarn ohne Grenze.
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/signaturen/buecher?signatur="+url.QueryEscape(basis), nil)
	srv.TitelUnterSignatur().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Regalansicht: Status %d", rec.Code)
	}
	var regal titelUnterSignaturAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &regal); err != nil {
		t.Fatalf("Regalansicht unlesbar: %v", err)
	}
	if len(regal.Titel) != 2 {
		t.Fatalf("Regal %q: erwartet 2 Titel (Basis + Unterfach), waren %d", basis, len(regal.Titel))
	}
	for _, b := range regal.Titel {
		if b.Signatur == basis+"X" {
			t.Errorf("Nachbarsignatur %q darf nicht im Regal %q stehen", b.Signatur, basis)
		}
	}
	// Reihenfolge des Regals: danach läuft man das Regal ab.
	if regal.Titel[0].Signatur != basis {
		t.Errorf("Regal nicht nach Signatur sortiert: erste Zeile %q", regal.Titel[0].Signatur)
	}

	// 3) Eine leere Signatur wird abgewiesen; sie wäre der Anfang von allem.
	rec = httptest.NewRecorder()
	srv.TitelUnterSignatur().ServeHTTP(rec,
		httptest.NewRequest(http.MethodGet, "/api/signaturen/buecher?signatur=%20", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("leere Signatur: erwartet 400, war %d", rec.Code)
	}
}

// Die Liste fasst nach der Regaladresse zusammen, dem Teil vor dem Trenner. Die Littera-
// Übernahme schreibt an jeden Titel die ganze Aufschrift des Buchrückens („LMF Deu 7 / Bie"):
// Je Aufschrift eine Zeile wären in der Sicherung von 2010 6.251 Einträge, 4.840 davon mit
// einem einzigen Titel. Schreiber (littera.SignaturAus) und Leser stehen im selben Test.
func TestSignaturenListe_FasstNachRegaladresseZusammen(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	regal := "Sk " + suffix
	nachbar := regal + "X"
	aufschriften := []string{
		regal,
		littera.SignaturAus(regal, "Bie"),
		littera.SignaturAus(regal, "Gri"),
		// Die zweite Zeile trägt in der Sicherung 235-mal selbst den Trenner.
		littera.SignaturAus(regal, "PoWi / Ich"),
		littera.SignaturAus(nachbar, "Abc"),
	}
	for i, aufschrift := range aufschriften {
		testdaten.LegeExemplarMitSignaturAn(t, pool, aufschrift, fmt.Sprintf("SIGR-%d-%s", i, suffix))
	}

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}
	rec := httptest.NewRecorder()
	srv.Signaturen().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/signaturen", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Signaturliste: Status %d", rec.Code)
	}
	var gruppen []signaturGruppe
	if err := json.Unmarshal(rec.Body.Bytes(), &gruppen); err != nil {
		t.Fatalf("Signaturliste unlesbar: %v", err)
	}
	gefunden := map[string]signaturGruppe{}
	for _, g := range gruppen {
		if strings.Contains(g.Signatur, repository.SignaturTrenner) {
			t.Errorf("Die Liste nennt eine ganze Aufschrift: %q", g.Signatur)
		}
		gefunden[g.Signatur] = g
	}
	if g := gefunden[regal]; g.Titel != 4 || g.Exemplare != 4 {
		t.Errorf("Regal %q: %d Titel und %d Exemplare, erwartet je 4", regal, g.Titel, g.Exemplare)
	}
	if g := gefunden[nachbar]; g.Titel != 1 || g.Exemplare != 1 {
		t.Errorf("Regal %q: %d Titel und %d Exemplare, erwartet je 1", nachbar, g.Titel, g.Exemplare)
	}

	// Die Regalansicht zu einer Zeile der Liste zeigt deren Titel mit der ganzen Aufschrift.
	rec = httptest.NewRecorder()
	srv.TitelUnterSignatur().ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/signaturen/buecher?signatur="+url.QueryEscape(regal), nil))
	var ansicht titelUnterSignaturAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &ansicht); err != nil {
		t.Fatalf("Regalansicht unlesbar: %v", err)
	}
	if len(ansicht.Titel) != 4 {
		t.Fatalf("Regal %q: %d Titel in der Ansicht, erwartet 4", regal, len(ansicht.Titel))
	}
	ganze := map[string]bool{}
	for _, b := range ansicht.Titel {
		ganze[b.Signatur] = true
	}
	for _, aufschrift := range aufschriften[:4] {
		if !ganze[aufschrift] {
			t.Errorf("Aufschrift %q fehlt in der Ansicht des Regals %q", aufschrift, regal)
		}
	}
}
