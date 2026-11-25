package schaden_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/api"
	"bibliothek/internal/auth"
	"bibliothek/internal/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/sse"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Verlust und Schaden gehören an den Gegenstand, den die Ausleihe nennt, nie an das Exemplar aus
// der Anfrage. Bei einer Geräteausleihe sondert die Schadensmeldung das Gerät aus und beendet die
// Ausleihe; eine Forderung entsteht nur mit Betrag. Der Bescheid des Landes nimmt nur Bücher und
// weist eine Geräteausleihe ab, ohne etwas zu buchen.

const meldungAusleiheOhneBuch = "Verlust und Schaden lassen sich hier nur für ein ausgeliehenes Buch buchen, nicht für ein Gerät"

type schadensWelt struct {
	pool    *pgxpool.Pool
	router  http.Handler
	sitzung string
}

func baueSchadensWelt(t *testing.T, email string) schadensWelt {
	t.Helper()
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	anmeldung, err := auth.NewAnmeldung("schaden-ohne-buch-testgeheimnis-32-bytes!", pool, time.Hour)
	if err != nil {
		t.Fatalf("NewAnmeldung: %v", err)
	}
	kontoID := testdaten.LegeKontoAn(t, pool, "Olga", "Ohnebuch", email, "admin", true)
	sitzung, err := anmeldung.ErzeugeToken(kontoID, auth.RolleAdmin, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	router := api.NewServer(&db.Verbindung{Pool: pool}, anmeldung, sse.NewBroker(), api.ServerKonfig{}).Routes()
	return schadensWelt{pool: pool, router: router, sitzung: sitzung}
}

func (w schadensWelt) sende(t *testing.T, pfad, rumpf string) *httptest.ResponseRecorder {
	t.Helper()
	req := tuertest.JSONPost(pfad, rumpf)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: w.sitzung})
	rec := httptest.NewRecorder()
	w.router.ServeHTTP(rec, req)
	return rec
}

// lies holt pfad mit GET und liefert den Rumpf; ein anderer Status als 200 bricht ab.
func (w schadensWelt) lies(t *testing.T, pfad string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, pfad, nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: w.sitzung})
	rec := httptest.NewRecorder()
	w.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: %d %s", pfad, rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

// geraeteAusleihe legt ein Gerät an und leiht es dem Kind aus.
func (w schadensWelt) geraeteAusleihe(t *testing.T, kind, nummer string) string {
	t.Helper()
	ctx := context.Background()
	var geraet, ausleihe string
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO geraete (modellname, barcode_id) VALUES ('Tablet', $1) RETURNING id`, nummer).Scan(&geraet); err != nil {
		t.Fatalf("Gerät anlegen: %v", err)
	}
	if err := w.pool.QueryRow(ctx, `INSERT INTO ausleihen (geraet_id, schueler_id, rueckgabe_frist)
		VALUES ($1, $2, now() + interval '14 days') RETURNING id`, geraet, kind).Scan(&ausleihe); err != nil {
		t.Fatalf("Geräteausleihe anlegen: %v", err)
	}
	return ausleihe
}

// pruefeGeraetUnberuehrt: Die Geräteausleihe läuft weiter, und es gibt keine Forderung.
func (w schadensWelt) pruefeGeraetUnberuehrt(t *testing.T, ausleihe string) {
	t.Helper()
	var beendet bool
	var forderungen int
	if err := w.pool.QueryRow(context.Background(), `
		SELECT (SELECT rueckgabe_am IS NOT NULL FROM ausleihen WHERE id = $1),
		       (SELECT count(*) FROM schadensfaelle)`, ausleihe).Scan(&beendet, &forderungen); err != nil {
		t.Fatalf("Stand lesen: %v", err)
	}
	if beendet {
		t.Error("die Geräteausleihe ist beendet, das Gerät ist aber noch draußen")
	}
	if forderungen != 0 {
		t.Errorf("%d Forderung(en) gebucht, erwartet keine", forderungen)
	}
}

func ausgesondert(t *testing.T, pool *pgxpool.Pool, exemplar string) bool {
	t.Helper()
	var ja bool
	if err := pool.QueryRow(context.Background(),
		`SELECT ist_ausgesondert FROM buecher_exemplare WHERE id = $1`, exemplar).Scan(&ja); err != nil {
		t.Fatalf("Exemplar lesen: %v", err)
	}
	return ja
}

func TestSchadenMelden_GeraetVerlorenBleibtSichtbar(t *testing.T) {
	w := baueSchadensWelt(t, "schaden-geraet@example.org")
	ctx := context.Background()
	kind := testdaten.LegeSchuelerAn(t, w.pool, "S-OHNEBUCH-A", "Gerda", "07A")
	ausleihe := w.geraeteAusleihe(t, kind, "G-OHNEBUCH-1")
	regalbuch := testdaten.LegeExemplarAn(t, w.pool, testdaten.LegeTitelAn(t, w.pool, "Regaltitel"), "EX-OHNEBUCH-1", true, "")
	rec := w.sende(t, "/api/damage/report", fmt.Sprintf(
		`{"loan_id":%q,"copy_id":%q,"beschreibung":"weg","art":"nicht_zurueckgegeben","betrag":15}`, ausleihe, regalbuch))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200 — %s", rec.Code, rec.Body.String())
	}
	if ausgesondert(t, w.pool, regalbuch) {
		t.Error("das Regalbuch aus der Anfrage ist ausgesondert, es war nie verliehen")
	}
	var geraetWeg, ausleihbar, beendet bool
	var forderung, protokoll int
	if err := w.pool.QueryRow(ctx, `
		SELECT g.ist_ausgesondert, g.ist_ausleihbar, a.rueckgabe_am IS NOT NULL,
		       (SELECT count(*) FROM schadensfaelle f
		        WHERE f.ausleihe_id = a.id AND f.geraet_id = g.id AND f.exemplar_id IS NULL AND f.betrag = 15),
		       (SELECT count(*) FROM audit_log l
		        WHERE l.tabelle = 'geraete' AND l.datensatz_id = g.id AND l.details->>'action' = 'verloren'
		          AND l.details->>'grund' = 'VERLUST')
		FROM ausleihen a JOIN geraete g ON g.id = a.geraet_id WHERE a.id = $1`, ausleihe,
	).Scan(&geraetWeg, &ausleihbar, &beendet, &forderung, &protokoll); err != nil {
		t.Fatalf("Stand lesen: %v", err)
	}
	if !geraetWeg || ausleihbar || !beendet || forderung != 1 || protokoll != 1 {
		t.Errorf("verloren %v, ausleihbar %v, Ausleihe beendet %v, %d Forderung(en) zu 15 €, %d Logbuchzeile(n); erwartet true, false, true, 1, 1",
			geraetWeg, ausleihbar, beendet, forderung, protokoll)
	}
	// Ein verlorenes Gerät verschwindet nicht aus der Geräteverwaltung.
	if liste := w.lies(t, "/api/geraete"); !strings.Contains(liste, `"barcode_id":"G-OHNEBUCH-1"`) ||
		!strings.Contains(liste, `"ist_ausgesondert":true`) {
		t.Errorf("die Geräteverwaltung zeigt das verlorene Gerät nicht: %s", liste)
	}
}

// Beschädigt bleibt das Gerät im Bestand und steht als defekt da; die Forderung entsteht wie beim
// Buch. Ein zweiter Klick bucht nichts doppelt, und nach der Reparatur lässt es sich in der
// Geräteverwaltung wieder freigeben.
func TestSchadenMelden_GeraetBeschaedigtStehtAlsDefekt(t *testing.T) {
	w := baueSchadensWelt(t, "schaden-geraet-defekt@example.org")
	ctx := context.Background()
	kind := testdaten.LegeSchuelerAn(t, w.pool, "S-OHNEBUCH-D", "Dora", "07A")
	ausleihe := w.geraeteAusleihe(t, kind, "G-OHNEBUCH-3")
	rumpf := fmt.Sprintf(`{"loan_id":%q,"beschreibung":"Display gesprungen","art":"beschaedigt","betrag":30}`, ausleihe)
	var geraet string
	for i := range 2 {
		if rec := w.sende(t, "/api/damage/report", rumpf); rec.Code != http.StatusOK {
			t.Fatalf("Meldung %d: Status %d, erwartet 200 — %s", i+1, rec.Code, rec.Body.String())
		}
		var verloren, ausleihbar, beendet bool
		var notiz string
		var forderungen int
		if err := w.pool.QueryRow(ctx, `
			SELECT g.id, g.ist_ausgesondert, g.ist_ausleihbar, coalesce(g.zustand_notiz, ''),
			       a.rueckgabe_am IS NOT NULL, (SELECT count(*) FROM schadensfaelle WHERE geraet_id = g.id)
			FROM ausleihen a JOIN geraete g ON g.id = a.geraet_id WHERE a.id = $1`, ausleihe,
		).Scan(&geraet, &verloren, &ausleihbar, &notiz, &beendet, &forderungen); err != nil {
			t.Fatalf("Stand lesen: %v", err)
		}
		if verloren || ausleihbar || notiz != "Display gesprungen" || !beendet || forderungen != 1 {
			t.Errorf("Meldung %d: verloren %v, ausleihbar %v, Notiz %q, beendet %v, %d Forderung(en); erwartet defekt im Bestand mit einer Forderung",
				i+1, verloren, ausleihbar, notiz, beendet, forderungen)
		}
	}

	req := httptest.NewRequest(http.MethodPut, "/api/geraete/"+geraet, strings.NewReader(`{"ist_ausleihbar":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: w.sitzung})
	rec := httptest.NewRecorder()
	w.router.ServeHTTP(rec, tuertest.MitCSRF(req))
	if rec.Code != http.StatusOK {
		t.Fatalf("Wieder freigeben: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(w.lies(t, "/api/geraete"), `"ist_ausleihbar":true`) {
		t.Error("das reparierte Gerät ist nicht wieder ausleihbar")
	}
}

func TestBescheid_GeraeteausleiheBuchtNichts(t *testing.T) {
	w := baueSchadensWelt(t, "bescheid-geraet@example.org")
	testdaten.LeereBescheide(t, w.pool)
	testdaten.SetzeBescheidAngaben(t, w.pool)
	kind := testdaten.LegeSchuelerAn(t, w.pool, "S-OHNEBUCH-B", "Bruno", "07A")
	ausleihe := w.geraeteAusleihe(t, kind, "G-OHNEBUCH-2")
	frist := time.Now().AddDate(0, 0, 28).Format("2006-01-02")

	rec := w.sende(t, "/api/schueler/"+kind+"/bescheide", fmt.Sprintf(
		`{"mittel":"land","frist_bis":%q,"ausleihen":[{"ausleihe_id":%q,"betrag":100}]}`, frist, ausleihe))
	if rec.Code != http.StatusConflict {
		t.Fatalf("Status %d, erwartet 409 — %s", rec.Code, rec.Body.String())
	}
	if got := tuertest.Fehlermeldung(t, rec); got != meldungAusleiheOhneBuch {
		t.Errorf("Meldung %q, erwartet %q", got, meldungAusleiheOhneBuch)
	}
	var bescheide, nummern int
	if err := w.pool.QueryRow(context.Background(), `
		SELECT (SELECT count(*) FROM schadensersatz_bescheide),
		       (SELECT coalesce(max(letzte_nr), 0) FROM schadensersatz_nummern)`).Scan(&bescheide, &nummern); err != nil {
		t.Fatalf("Bescheide lesen: %v", err)
	}
	if bescheide != 0 || nummern != 0 {
		t.Errorf("%d Bescheid(e), letzte Nummer %d: Abgewiesen wird ohne Brief und ohne verbrauchte Nummer", bescheide, nummern)
	}
	w.pruefeGeraetUnberuehrt(t, ausleihe)
}

// Das Exemplar steht an der Ausleihe wie der Schuldner. Nennt die Anfrage ein anderes, trifft die
// Meldung trotzdem das geliehene Buch und lässt das genannte im Regal.
func TestSchadenMelden_ExemplarDerAnfrageZaehltNicht(t *testing.T) {
	w := baueSchadensWelt(t, "schaden-exemplar@example.org")
	ctx := context.Background()
	titel := testdaten.LegeTitelAn(t, w.pool, "Exemplartitel")
	geliehen := testdaten.LegeExemplarAn(t, w.pool, titel, "EX-GELIEHEN-1", true, "")
	imRegal := testdaten.LegeExemplarAn(t, w.pool, titel, "EX-IMREGAL-1", true, "")
	kind := testdaten.LegeSchuelerAn(t, w.pool, "S-OHNEBUCH-C", "Finn", "08A")
	var ausleihe string
	if err := w.pool.QueryRow(ctx, `INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		VALUES ($1, $2, now() + interval '14 days') RETURNING id`, geliehen, kind).Scan(&ausleihe); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}

	rec := w.sende(t, "/api/damage/report", fmt.Sprintf(
		`{"loan_id":%q,"copy_id":%q,"beschreibung":"Wasserschaden","art":"beschaedigt","betrag":600}`, ausleihe, imRegal))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200 — %s", rec.Code, rec.Body.String())
	}
	if ausgesondert(t, w.pool, imRegal) {
		t.Error("das Exemplar aus der Anfrage ist ausgesondert, es war nie verliehen")
	}
	if !ausgesondert(t, w.pool, geliehen) {
		t.Error("das geliehene Exemplar ist nicht ausgesondert")
	}
	var amFall string
	if err := w.pool.QueryRow(ctx,
		`SELECT exemplar_id::text FROM schadensfaelle WHERE ausleihe_id = $1`, ausleihe).Scan(&amFall); err != nil {
		t.Fatalf("Forderung lesen: %v", err)
	}
	if amFall != geliehen {
		t.Errorf("die Forderung nennt Exemplar %s, geliehen war %s", amFall, geliehen)
	}
}
