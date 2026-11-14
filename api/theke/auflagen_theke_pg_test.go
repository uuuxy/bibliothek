package theke

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/service"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
	"bibliothek/sse"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Gemischte Auflagen an der Theke: Bekommt ein Kind eine Auflage,
// während Kinder seiner Klasse eine andere desselben Buchs haben, trägt die Antwort des Scans
// den Hinweis. Geprüft über den Live-Pfad — POST /api/action mit dem echten Omnibox- und
// Ausleih-Dienst —, denn nur so ist belegt, dass der Hinweis von der Buchung bis in die JSON-
// Antwort reist. Die Regeln der Zählung prüft repository/auflagen_klasse_pg_test.go.
func TestTheke_AuflagenHinweisBeiGemischterKlasse(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE werke CASCADE`); err != nil {
		t.Fatal(err)
	}

	scan := thekeZumScannen(t, pool, "theke-auflage@schule.invalid")
	kind := func(barcode, klasse string) string {
		t.Helper()
		id := testdaten.LegeSchuelerMitNamenAn(t, pool, barcode, "Kind", barcode, klasse, 2031)
		return id
	}
	titel := func(auflage string, jahr int) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, auflage, erscheinungsjahr, ist_lernmittel)
			VALUES ('Mathe 7', $1, $2, true) RETURNING id`, auflage, jahr).Scan(&id); err != nil {
			t.Fatalf("Titel anlegen: %v", err)
		}
		return id
	}
	alt, neu := titel("3. Aufl.", 2019), titel("4. Aufl.", 2023)
	if _, err := repository.FasseAuflagenZusammen(ctx, pool, alt, neu); err != nil {
		t.Fatal(err)
	}
	for i, t2 := range []string{alt, alt, alt, neu, neu} {
		testdaten.LegeExemplarAn(t, pool, t2, fmt.Sprintf("B-518T-%d", i), true, "")
	}

	// Zwei Kinder der 7b bekommen die 3. Auflage — kein Hinweis, die Klasse ist einheitlich.
	if a := scan("B-518T-0", kind("S-518T-1", "7b")); a.Art != "ausleihe" || a.AuflagenHinweis != nil {
		t.Fatalf("erste Ausleihe: Typ %q, Hinweis %+v — erwartet Ausleihe ohne Hinweis", a.Art, a.AuflagenHinweis)
	}
	if a := scan("B-518T-1", kind("S-518T-2", "7b")); a.AuflagenHinweis != nil {
		t.Errorf("zweite Ausleihe derselben Auflage: Hinweis %+v — erwartet keiner", a.AuflagenHinweis)
	}

	// Das dritte Kind bekommt die 4. Auflage: Hinweis mit Klasse und den zwei anderen.
	a := scan("B-518T-3", kind("S-518T-3", "7b"))
	if a.Art != "ausleihe" {
		t.Fatalf("Typ %q — die Ausleihe muss gebucht sein, der Hinweis ändert daran nichts", a.Art)
	}
	h := a.AuflagenHinweis
	if h == nil || h.Klasse != "07B" || h.Auflage != "4. Aufl." || len(h.Andere) != 1 ||
		h.Andere[0].Auflage != "3. Aufl." || h.Andere[0].Kinder != 2 {
		t.Errorf("Hinweis %+v — erwartet 07B, 4. Aufl., andere: 3. Aufl. bei 2 Kindern", h)
	}

	// Ein Kind einer anderen Klasse bekommt die 4. Auflage: dort gibt es keine Mischung.
	if a := scan("B-518T-4", kind("S-518T-4", "7c")); a.AuflagenHinweis != nil {
		t.Errorf("7c: Hinweis %+v — erwartet keiner", a.AuflagenHinweis)
	}
}

// Die zwei Hinweise beim Bücherstapel: Ein Buch, das sonst niemand in der Klasse hat, und ein
// Buch, das das Kind schon hat. Geprüft wie der Auflagen-Hinweis über den Live-Pfad; die Regeln
// der Zählung prüft repository/buch_in_klasse_pg_test.go.
func TestTheke_BuchHinweiseBeimStapel(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	scan := thekeZumScannen(t, pool, "theke-stapel@schule.invalid")

	titel := func(name string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, ist_lernmittel) VALUES ($1, true) RETURNING id`,
			name).Scan(&id); err != nil {
			t.Fatalf("Titel anlegen: %v", err)
		}
		return id
	}
	mathe8, mathe7 := titel("Mathe 8"), titel("Mathe 7")
	nr := 0
	exemplar := func(titelID string) string {
		t.Helper()
		nr++
		barcode := fmt.Sprintf("B-STAPEL-%d", nr)
		testdaten.LegeExemplarAn(t, pool, titelID, barcode, true, "")
		return barcode
	}
	var kinder []string
	for i := range 8 {
		barcode := fmt.Sprintf("S-STAPEL-%d", i)
		kinder = append(kinder, testdaten.LegeSchuelerMitNamenAn(t, pool, barcode, "Kind", barcode, "8g1", 2031))
	}

	// Fünf Kinder der 8G1 bekommen Mathe 8: kein Hinweis.
	for i, k := range kinder[:5] {
		if a := scan(exemplar(mathe8), k); a.Art != "ausleihe" || a.KlassenHinweis != nil || a.DoppeltHinweis {
			t.Fatalf("Mathe 8 an Kind %d: Typ %q, Klasse %+v, doppelt %v — erwartet Ausleihe ohne Hinweis",
				i, a.Art, a.KlassenHinweis, a.DoppeltHinweis)
		}
	}

	// Das sechste bekommt Mathe 7, das sonst niemand hat: Mit ihm haben sechs von acht ihre Bücher.
	a := scan(exemplar(mathe7), kinder[5])
	soll := repository.BuchFremdInKlasse{Klasse: "08G1", MitBuechern: 6, Kinder: 8}
	if a.Art != "ausleihe" || a.KlassenHinweis == nil || *a.KlassenHinweis != soll || a.DoppeltHinweis {
		t.Errorf("Mathe 7: Typ %q, Klasse %+v, doppelt %v — erwartet Ausleihe mit %+v",
			a.Art, a.KlassenHinweis, a.DoppeltHinweis, soll)
	}

	// Das erste Kind bekommt ein zweites Mathe 8: doppelt, aber kein fremdes Buch.
	if a := scan(exemplar(mathe8), kinder[0]); a.Art != "ausleihe" || !a.DoppeltHinweis || a.KlassenHinweis != nil {
		t.Errorf("zweites Mathe 8: Typ %q, Klasse %+v, doppelt %v — erwartet Ausleihe, doppelt",
			a.Art, a.KlassenHinweis, a.DoppeltHinweis)
	}
}

// thekeZumScannen baut die Theke mit den echten Diensten und liefert den Scan über POST
// /api/action: Barcode und geladener Leser hinein, die Antwort heraus.
func thekeZumScannen(t *testing.T, pool *pgxpool.Pool, email string) func(barcode, leser string) eingabeAntwort {
	t.Helper()
	mitarbeiter := testdaten.LegeKontoAn(t, pool, "Theke", "Kraft", email, "mitarbeiter", true)
	studentRepo := repository.NewLeserAbfragen(pool)
	bookRepo := repository.NewKatalogAbfragen(pool)
	loanRepo := repository.NewAusleihAbfragen(pool)
	auditRepo := repository.NewProtokollAbfragen(pool)
	loanSvc := service.NewAusleihDienst(pool, studentRepo, bookRepo, loanRepo, auditRepo)
	deviceSvc := service.NewGeraeteDienst(pool, studentRepo, loanRepo, auditRepo)
	omnibox := service.NewOmniboxDienst(pool, studentRepo, bookRepo, loanRepo, loanSvc, deviceSvc)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}, Broker: sse.NewBroker()}}

	return func(barcode, leser string) eingabeAntwort {
		t.Helper()
		req := httptest.NewRequest("POST", "/api/action",
			strings.NewReader(fmt.Sprintf(`{"query":%q,"active_leser_id":%q}`, barcode, leser)))
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
			&auth.Claims{Rolle: auth.Rolle("mitarbeiter"), KontoID: mitarbeiter}))
		rec := httptest.NewRecorder()
		srv.EingabeVerarbeiten(omnibox)(rec, req)
		var antwort eingabeAntwort
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil || rec.Code != 200 {
			t.Fatalf("Scan %s: %d %s", barcode, rec.Code, rec.Body.String())
		}
		return antwort
	}
}
