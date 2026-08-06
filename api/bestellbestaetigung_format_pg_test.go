package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Formatwahl des Lieferanten — am echten Endpunkt, nicht am Generator.
//
// Warum es diese Datei zusätzlich zu etiketten_layout_test.go gibt: Der dortige Test ruft
// GenerateLabelsPDF direkt auf und belegt, dass der Generator das Raster beachtet. Als der
// Lieferanten-Weg beim Rückbau probeweise wieder auf das feste "zweckform_l4760" gesetzt
// wurde, blieb er GRÜN — er sieht die Durchreichung gar nicht. Genau diese Lücke ist die
// Bugklasse dieses Projekts: ein Gate, das eine Stelle prüft, die niemand kaputtmacht.
//
// Hier läuft deshalb der echte Weg: Bestellung → Token → HTTP-Aufruf mit ?format= →
// fertiges PDF.

// etikettenBogenHolen ruft den öffentlichen Etiketten-Endpunkt und liefert die Antwort.
func etikettenBogenHolen(t *testing.T, srv *Server, token, groesse, format string) *httptest.ResponseRecorder {
	t.Helper()
	pfad := "/api/public/bestellung/" + token + "/etiketten/" + groesse
	if format != "" {
		pfad += "?format=" + format
	}
	req := httptest.NewRequest(http.MethodGet, pfad, nil)
	req.SetPathValue("token", token)
	req.SetPathValue("groesse", groesse)
	rec := httptest.NewRecorder()
	srv.OeffentlicheEtikettenHandler().ServeHTTP(rec, req)
	return rec
}

// bestellungMitEtiketten legt eine Bestellung mit `menge` Vorab-Barcodes an.
func bestellungMitEtiketten(t *testing.T, srv *Server, pool *pgxpool.Pool, menge int) string {
	t.Helper()
	ctx := context.Background()
	svc := NewOrderService(srv.DB, repository.NewBookRepository(pool))

	lieferant := haendler(t, pool, "Naacher", true)
	titel := titelMitMeldebestand(t, pool, "LMF-Formatprobe", 0)

	res, err := svc.ProcessOrder(ctx, SubmitOrderRequest{
		SupplierID: lieferant,
		Items:      []OrderItemRequest{{TitelID: titel, Menge: menge, Preis: 10, GenerateBarcodes: true}},
	})
	if err != nil {
		t.Fatalf("Bestellung: %v", err)
	}
	if res.BestaetigungsToken == "" {
		t.Fatal("kein Bestätigungs-Token — der Lieferant bekäme eine Mail ohne Link")
	}
	return res.BestaetigungsToken
}

// Das Gate: Dasselbe Bestellvolumen muss je nach gewähltem Raster auf unterschiedlich
// viele Blätter kommen — über den ÖFFENTLICHEN Endpunkt.
//
// 25 Exemplare: 3×7 = 21 je Bogen → 2 Blätter, 4×13 = 52 je Bogen → 1 Blatt.
// Bliebe das Format irgendwo auf dem Weg liegen, wären beide Antworten gleich.
func TestLieferantenEtiketten_FormatWirktBisInsPDF(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	srv := &Server{DB: &db.Database{Pool: pool}}
	token := bestellungMitEtiketten(t, srv, pool, 25)

	faelle := []struct {
		format     string
		wantSeiten int
	}{
		{"zweckform_l4760", 2},
		{"standard_52", 1},
	}

	gesehen := map[int]string{}
	for _, f := range faelle {
		rec := etikettenBogenHolen(t, srv, token, "klein", f.format)
		if rec.Code != http.StatusOK {
			t.Fatalf("Format %s: Status = %d, body: %s", f.format, rec.Code, rec.Body.String())
		}
		if !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF")) {
			t.Fatalf("Format %s: Antwort ist kein PDF", f.format)
		}
		seiten, breite, hoehe := pdfSeiten(t, rec.Body.Bytes())
		if seiten != f.wantSeiten {
			t.Errorf("Format %s: %d Seiten, erwartet %d — das Raster kam nicht am Generator an",
				f.format, seiten, f.wantSeiten)
		}
		if breite != 210 || hoehe != 297 {
			t.Errorf("Format %s: Seitengröße %d×%d mm, erwartet A4", f.format, breite, hoehe)
		}
		if vorher, doppelt := gesehen[seiten]; doppelt {
			t.Errorf("Format %s liefert dieselbe Seitenzahl wie %s — die Auswahl wirkt nicht",
				f.format, vorher)
		}
		gesehen[seiten] = f.format
	}
}

// Ohne Angabe gilt die Vorgabe. Der Lieferant, der nichts auswählt, bekommt weiterhin
// einen brauchbaren Bogen — die Formatwahl darf keine neue Pflicht sein.
func TestLieferantenEtiketten_OhneFormatGiltDieVorgabe(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	srv := &Server{DB: &db.Database{Pool: pool}}
	token := bestellungMitEtiketten(t, srv, pool, 25)

	ohne := etikettenBogenHolen(t, srv, token, "klein", "")
	mitVorgabe := etikettenBogenHolen(t, srv, token, "klein", StandardLabelFormat)

	if ohne.Code != http.StatusOK || mitVorgabe.Code != http.StatusOK {
		t.Fatalf("Status ohne=%d mitVorgabe=%d", ohne.Code, mitVorgabe.Code)
	}
	seitenOhne, _, _ := pdfSeiten(t, ohne.Body.Bytes())
	seitenMit, _, _ := pdfSeiten(t, mitVorgabe.Body.Bytes())
	if seitenOhne != seitenMit {
		t.Errorf("ohne Format %d Seiten, mit Vorgabe %d — die Vorgabe ist nicht dieselbe",
			seitenOhne, seitenMit)
	}
}

// Ein unbekanntes Raster wird ABGEWIESEN und nicht still auf die Vorgabe gedreht.
//
// GetLabelFormat liefert bei Unbekanntem stillschweigend zweckform_l4760 zurück. Würde
// der Handler das übernehmen, druckte der Lieferant nach einem Tippfehler im falschen
// Raster — und merkte es erst am verschnittenen Bogen.
func TestLieferantenEtiketten_UnbekanntesFormatWirdAbgewiesen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	srv := &Server{DB: &db.Database{Pool: pool}}
	token := bestellungMitEtiketten(t, srv, pool, 3)

	rec := etikettenBogenHolen(t, srv, token, "klein", "gibtesnicht")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Status = %d, want 400 — unbekanntes Format wurde angenommen (body: %s)",
			rec.Code, rec.Body.String())
	}
	if bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF")) {
		t.Fatal("trotz unbekanntem Format kam ein PDF zurück")
	}
}

// Die große Größe hat ein festes Raster — ein mitgeschicktes Format darf daran nichts
// ändern, sonst behauptete die Oberfläche eine Wahl, die es nicht gibt.
func TestLieferantenEtiketten_GrossIgnoriertDasRaster(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	srv := &Server{DB: &db.Database{Pool: pool}}
	token := bestellungMitEtiketten(t, srv, pool, 5)

	a := etikettenBogenHolen(t, srv, token, "gross", "zweckform_l4760")
	b := etikettenBogenHolen(t, srv, token, "gross", "standard_52")
	if a.Code != http.StatusOK || b.Code != http.StatusOK {
		t.Fatalf("Status a=%d b=%d", a.Code, b.Code)
	}

	seitenA, breiteA, hoeheA := pdfSeiten(t, a.Body.Bytes())
	seitenB, _, _ := pdfSeiten(t, b.Body.Bytes())
	if seitenA != seitenB {
		t.Errorf("große Etiketten reagieren auf das Raster (%d vs. %d Seiten)", seitenA, seitenB)
	}
	// 5 Exemplare, vier je Blatt → zwei Blätter, A4.
	if seitenA != 2 {
		t.Errorf("5 große Etiketten ergaben %d Seiten, erwartet 2 (4 je A4-Blatt)", seitenA)
	}
	if breiteA != 210 || hoeheA != 297 {
		t.Errorf("Seitengröße %d×%d mm, erwartet A4", breiteA, hoeheA)
	}
}

// Das gewählte Raster landet in der Historie der Schule. Ohne diese Notiz weiß die
// Bibliothek nicht, wie die Aufkleber aussehen, die gleich ankommen.
func TestLieferantenBestaetigung_MerktSichDasRaster(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	srv := &Server{DB: &db.Database{Pool: pool}}
	token := bestellungMitEtiketten(t, srv, pool, 3)

	koerper, err := json.Marshal(OeffentlichBestaetigenRequest{
		EtikettenGroesse: "klein",
		EtikettenFormat:  "standard_52",
	})
	if err != nil {
		t.Fatalf("Body bauen: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost,
		"/api/public/bestellung/"+token+"/bestaetigen", bytes.NewReader(koerper))
	req.SetPathValue("token", token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.OeffentlichBestaetigenHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Bestätigen: Status = %d, body: %s", rec.Code, rec.Body.String())
	}

	var groesse, format *string
	if err := pool.QueryRow(ctx, `
		SELECT etiketten_groesse, etiketten_format
		FROM bestellungen_verlauf
		WHERE bestaetigungs_token_hash = $1
	`, hashBestaetigungsToken(token)).Scan(&groesse, &format); err != nil {
		t.Fatalf("Bestellung lesen: %v", err)
	}
	if groesse == nil || *groesse != "klein" {
		t.Errorf("etiketten_groesse = %v, want 'klein'", groesse)
	}
	if format == nil || *format != "standard_52" {
		t.Errorf("etiketten_format = %v, want 'standard_52' — das Raster ging verloren", format)
	}
}

// Die Formatliste muss auch beim Lieferanten ankommen — sonst zeigt seine Seite keine
// Auswahl und er druckt weiter im Zweckform-Raster.
func TestOeffentlicheBestellung_LiefertDieFormatauswahl(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	srv := &Server{DB: &db.Database{Pool: pool}}
	token := bestellungMitEtiketten(t, srv, pool, 3)

	bestellungID, err := srv.bestellungPerToken(ctx, token)
	if err != nil {
		t.Fatalf("Token auflösen: %v", err)
	}
	ansicht, err := srv.ladeOeffentlicheBestellung(ctx, bestellungID)
	if err != nil {
		t.Fatalf("Ansicht laden: %v", err)
	}

	if len(ansicht.EtikettenFormate) != len(labelFormats) {
		t.Fatalf("Seite bekommt %d Formate, es gibt %d", len(ansicht.EtikettenFormate), len(labelFormats))
	}
	if ansicht.EtikettenFormatVorgabe != StandardLabelFormat {
		t.Errorf("Vorgabe = %q, want %q", ansicht.EtikettenFormatVorgabe, StandardLabelFormat)
	}
	// Jedes angebotene Format muss über den Endpunkt auch wirklich druckbar sein.
	for _, f := range ansicht.EtikettenFormate {
		rec := etikettenBogenHolen(t, srv, token, "klein", f.ID)
		if rec.Code != http.StatusOK {
			t.Errorf("angebotenes Format %s liefert Status %d", f.ID, rec.Code)
		}
	}
}
