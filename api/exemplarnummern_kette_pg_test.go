package api

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Exemplarnummern — EINE Nummer über die ganze Kette.
//
// Die Frage aus dem Betrieb: "Gibt es die Nummern dann auch wirklich?" Sie entstehen beim
// Bestellen aus einer Sequenz, wandern in die Mail, auf den Aufkleber des Händlers, und
// müssen Wochen später am gelieferten Buch gescannt werden. Jedes Glied für sich ist
// geprüft; brechen kann die Kette an den ÜBERGÄNGEN — eine Nummer im Anhang, die in der
// Datenbank fehlt, wäre ein Aufkleber auf einem Buch, das niemand kennt.
//
// Deshalb hier kein Einzelglied, sondern der Durchlauf: bestellen → Nummern aus dem
// CSV-Anhang der echten Mail → dieselben Nummern in der Datenbank → dieselben Nummern
// gedruckt auf dem Bogen hinter dem Bestätigungs-Link → Wareneingang → Scan findet das Buch.

// dateiAusMail schält einen Anhang aus der MIME-Nachricht und dekodiert ihn.
func dateiAusMail(t *testing.T, nachricht, namensteil string) string {
	t.Helper()
	teile := strings.Split(nachricht, "--"+mimeGrenze(t, nachricht))
	for _, teil := range teile {
		if !strings.Contains(teil, namensteil) {
			continue
		}
		leer := strings.Index(teil, "\r\n\r\n")
		if leer < 0 {
			t.Fatalf("Anhang %q hat keinen Rumpf", namensteil)
		}
		roh := strings.NewReplacer("\r", "", "\n", "").Replace(teil[leer:])
		daten, err := base64.StdEncoding.DecodeString(strings.TrimSpace(roh))
		if err != nil {
			t.Fatalf("Anhang %q nicht dekodierbar: %v", namensteil, err)
		}
		return string(daten)
	}
	t.Fatalf("Anhang %q nicht in der Mail", namensteil)
	return ""
}

var grenzeMuster = regexp.MustCompile(`boundary=([^\s;"]+)`)

func mimeGrenze(t *testing.T, nachricht string) string {
	t.Helper()
	treffer := grenzeMuster.FindStringSubmatch(nachricht)
	if treffer == nil {
		t.Fatal("keine MIME-Grenze in der Nachricht — die Mail ist kaputt")
	}
	return treffer[1]
}

var barcodeMuster = regexp.MustCompile(`B-\d{5,}`)

func barcodesAus(text string) []string {
	gefunden := barcodeMuster.FindAllString(text, -1)
	einmalig := map[string]bool{}
	var liste []string
	for _, b := range gefunden {
		if !einmalig[b] {
			einmalig[b] = true
			liste = append(liste, b)
		}
	}
	sort.Strings(liste)
	return liste
}

var linkMuster = regexp.MustCompile(`/bestellung/([A-Za-z0-9_-]{20,})`)

func TestExemplarnummern_VonDerBestellungBisZumScan(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	setzeOeffentlicheAdresse(t, pool, "https://bib.example.invalid")
	sitzungen := mailAbfangen(t)

	srv := &Server{DB: &db.Database{Pool: pool}}
	lieferant := haendler(t, pool, "Naacher-Kette", true)
	titel := titelMitMeldebestand(t, pool, "LMF-Kette", 0)

	rec := bestelleUeberHandler(t, srv, lieferant, titel)
	if rec.Code != http.StatusOK {
		t.Fatalf("Bestellung: Status %d — %s", rec.Code, rec.Body.String())
	}
	nachricht := warteAufMail(t, sitzungen)

	// 1. Die Nummern, die der Händler bekommt.
	ausMail := barcodesAus(dateiAusMail(t, nachricht, "barcode_mapping"))
	if len(ausMail) != 2 {
		t.Fatalf("Barcodes im CSV-Anhang = %v, erwartet 2 (Bestellmenge)", ausMail)
	}

	// 2. Dieselben Nummern müssen als Exemplare existieren — sonst klebt der Händler
	//    Aufkleber auf Bücher, die das System nicht kennt.
	ausDB := barcodesDerBestellung(t, pool)
	if strings.Join(ausDB, ",") != strings.Join(ausMail, ",") {
		t.Fatalf("Datenbank hat %v, die Mail nennt %v — die Nummern laufen auseinander", ausDB, ausMail)
	}

	// 3. Und dieselben Nummern stehen GEDRUCKT auf dem Bogen hinter dem Link. Der Bogen
	//    ist der zweite Weg zu denselben Daten (er liest neu aus der Datenbank); genau an
	//    solchen Nahtstellen sind hier schon zweimal Felder verlorengegangen.
	treffer := linkMuster.FindStringSubmatch(nachricht)
	if treffer == nil {
		t.Fatal("kein Bestätigungs-Link in der Mail")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/public/bestellung/"+treffer[1]+"/etiketten/gross", nil)
	req.SetPathValue("token", treffer[1])
	req.SetPathValue("groesse", "gross")
	bogen := httptest.NewRecorder()
	srv.OeffentlicheEtikettenHandler()(bogen, req)
	if bogen.Code != http.StatusOK {
		t.Fatalf("Etikettenbogen über den Link: Status %d", bogen.Code)
	}
	gedruckt := barcodesAus(strings.Join(pdfTexte(t, bogen.Body.Bytes()), "\n"))
	if strings.Join(gedruckt, ",") != strings.Join(ausMail, ",") {
		t.Fatalf("auf dem Bogen stehen %v, bestellt wurden %v", gedruckt, ausMail)
	}

	// 4. Bis zur Lieferung steht das Exemplar im Zulauf und ist nicht ausleihbar.
	zulauf, err := service.GetIncomingShipments(ctx, pool)
	if err != nil {
		t.Fatalf("Zulauf laden: %v", err)
	}
	ids := exemplarIDsDerBestellung(t, pool)
	if fehlend := nichtImZulauf(zulauf, ids); len(fehlend) > 0 {
		t.Fatalf("%d Exemplare fehlen im Wareneingang — sie wären nur noch über die Datenbank auffindbar", len(fehlend))
	}

	// 5. Wareneingang: dieselben Exemplare werden ausleihbar, die Nummer bleibt.
	if _, err := service.BulkReceiveOrder(ctx, pool, repository.NewAuditRepository(pool), ids, "", "127.0.0.1"); err != nil {
		t.Fatalf("Wareneingang: %v", err)
	}

	// 6. Der Scan am Buch. Code39 druckt den Inhalt in GROSSBUCHSTABEN — die Nummer muss
	//    also genau so gefunden werden, wie sie auf dem Aufkleber steht.
	bookRepo := repository.NewBookRepository(pool)
	for _, barcode := range ausMail {
		exemplar, err := bookRepo.GetCopyByBarcode(ctx, strings.ToUpper(barcode))
		if err != nil {
			t.Fatalf("Scan von %q findet kein Exemplar: %v", barcode, err)
		}
		if !exemplar.IstAusleihbar {
			t.Errorf("%s ist nach dem Wareneingang nicht ausleihbar", barcode)
		}
	}
}

// barcodesDerBestellung liest die tatsächlich angelegten Exemplarnummern.
func barcodesDerBestellung(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		`SELECT barcode_id FROM buecher_exemplare WHERE ist_ausleihbar = false ORDER BY barcode_id`)
	if err != nil {
		t.Fatalf("Barcodes lesen: %v", err)
	}
	defer rows.Close()

	var liste []string
	for rows.Next() {
		var b string
		if err := rows.Scan(&b); err != nil {
			t.Fatalf("Barcode lesen: %v", err)
		}
		liste = append(liste, b)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Barcodes lesen: %v", err)
	}
	return liste
}

// exemplarIDsDerBestellung liefert die IDs, mit denen der Wareneingang arbeitet.
func exemplarIDsDerBestellung(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		`SELECT id FROM buecher_exemplare WHERE ist_ausleihbar = false ORDER BY barcode_id`)
	if err != nil {
		t.Fatalf("Exemplar-IDs lesen: %v", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("Exemplar-ID lesen: %v", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Exemplar-IDs lesen: %v", err)
	}
	return ids
}

// nichtImZulauf meldet die Exemplare, die der Wareneingang NICHT anbietet.
func nichtImZulauf(gruppen []*service.ShipmentGroup, ids []string) []string {
	angeboten := map[string]bool{}
	for _, g := range gruppen {
		for _, item := range g.Items {
			for _, id := range item.ExemplarIDs {
				angeboten[id] = true
			}
		}
	}
	var fehlend []string
	for _, id := range ids {
		if !angeboten[id] {
			fehlend = append(fehlend, id)
		}
	}
	return fehlend
}
