package schaden_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/api"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/sse"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Verlust und Schaden gehören an das Buch, das die Ausleihe nennt. Eine Geräteausleihe hat
// keins; die Tür nahm dann das Exemplar aus der Anfrage. Mit der Kennung eines Regalbuchs
// sonderte „Verlust/Schaden melden" es als Verlust aus, das Kind mit dem Gerät bekam dafür eine
// Forderung, und die Geräteausleihe galt als beendet, obwohl das Gerät noch draußen war. Beide
// Türen, die Schadensmeldung und der Bescheid, lehnen eine Geräteausleihe jetzt ab und buchen
// nichts.

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

func TestSchadenMelden_GeraeteausleiheBuchtNichts(t *testing.T) {
	w := baueSchadensWelt(t, "schaden-geraet@example.org")
	kind := testdaten.LegeSchuelerAn(t, w.pool, "S-OHNEBUCH-A", "Gerda", "07A")
	ausleihe := w.geraeteAusleihe(t, kind, "G-OHNEBUCH-1")
	regalbuch := testdaten.LegeExemplarAn(t, w.pool, testdaten.LegeTitelAn(t, w.pool, "Regaltitel", 1), "EX-OHNEBUCH-1", true, "")

	rec := w.sende(t, "/api/damage/report", fmt.Sprintf(
		`{"loan_id":%q,"copy_id":%q,"beschreibung":"weg","art":"nicht_zurueckgegeben","betrag":1500}`, ausleihe, regalbuch))
	if rec.Code != http.StatusConflict {
		t.Fatalf("Status %d, erwartet 409 — %s", rec.Code, rec.Body.String())
	}
	if got := tuertest.Fehlermeldung(t, rec); got != meldungAusleiheOhneBuch {
		t.Errorf("Meldung %q, erwartet %q", got, meldungAusleiheOhneBuch)
	}
	if ausgesondert(t, w.pool, regalbuch) {
		t.Error("das Regalbuch aus der Anfrage ist ausgesondert, es war nie verliehen")
	}
	w.pruefeGeraetUnberuehrt(t, ausleihe)
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
	titel := testdaten.LegeTitelAn(t, w.pool, "Exemplartitel", 1)
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
