package inventurlauf

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

// verliehenImRegal ist ein Regal mit zwei Büchern eines Titels, eines davon an ein Kind verliehen,
// und ein Lauf darüber.
type verliehenImRegal struct {
	lauf                     *repository.Inventurlauf
	titelID, verliehen, kind string
}

func legeVerliehenesImRegalAn(t *testing.T, pool *pgxpool.Pool) verliehenImRegal {
	t.Helper()
	leereInventur(t, pool)
	// Die Rechte der Rollen wie in einer neuen Anlage (main.go): Ob zurückgebucht wird, hängt am
	// Recht perform_actions der Rolle.
	if err := (&db.Verbindung{Pool: pool}).SchreibeRechteVorgabe(context.Background()); err != nil {
		t.Fatalf("Rechte vorbelegen: %v", err)
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	signatur := "RB " + suffix
	titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Rückbuch", signatur, 0)
	testdaten.LegeExemplarAn(t, pool, titelID, "RB-DA-"+suffix, true, "")
	verliehen := "RB-VERL-" + suffix
	exemplarID := testdaten.LegeExemplarAn(t, pool, titelID, verliehen, true, "")
	kind := testdaten.LegeSchuelerAn(t, pool, "RB-S-"+suffix, "Leihkind", "07A")
	if _, err := pool.Exec(context.Background(), `INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		VALUES ($1, $2, now() + interval '14 days')`, exemplarID, kind); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
	lauf, err := repository.NewInventurAbfragen(pool).
		LegeLaufAn(context.Background(), "signature", repository.InventurBereich{Signatur: &signatur}, signatur, "")
	if err != nil {
		t.Fatalf("Inventur anlegen: %v", err)
	}
	return verliehenImRegal{lauf: lauf, titelID: titelID, verliehen: verliehen, kind: kind}
}

// scanneAls scannt mit der Sitzung eines Kontos der Rolle rolle; zurückgebucht wird über den
// Dienst der Theke.
func scanneAls(t *testing.T, srv *Tueren, pool *pgxpool.Pool, kontoID string, rolle auth.Rolle, laufID, barcode string) scanAntwort {
	t.Helper()
	dienst := service.NewAusleihDienst(pool, repository.NewLeserAbfragen(pool), repository.NewKatalogAbfragen(pool),
		repository.NewAusleihAbfragen(pool), repository.NewProtokollAbfragen(pool))
	body := fmt.Sprintf(`{"session_id":%q,"barcode_id":%q}`, laufID, barcode)
	req := httptest.NewRequest(http.MethodPost, "/api/inventur/scan", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
		&auth.Claims{KontoID: kontoID, Rolle: rolle}))
	rec := httptest.NewRecorder()
	srv.ScanVerbuchen(dienst).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Scan: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	var antwort scanAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	return antwort
}

func offeneAusleihen(t *testing.T, pool *pgxpool.Pool, barcode string) int {
	t.Helper()
	return pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM ausleihen a
		JOIN buecher_exemplare e ON e.id = a.exemplar_id
		WHERE e.barcode_id = $1 AND a.rueckgabe_am IS NULL`, barcode)
}

// Ein verliehenes Buch steht im Regal und wird in der Inventur gescannt: Der Scan bucht es zurück
// wie die Theke. Die Ausleihe endet, die Vormerkung eines anderen Kindes ist bedient, die Karte
// sagt „ins Abholfach", die übrigen Arbeitsplätze bekommen die Meldung „action", und der Stand
// zählt das Buch als gefunden. Die Meldung wird am Strom gelesen: Was nach „connected" ankommt,
// hat der Scan gesendet.
func TestScanVerbuchen_VerliehenesBuchWirdZurueckgebucht(t *testing.T) {
	pool := pgtest.Pool(t)
	regal := legeVerliehenesImRegalAn(t, pool)
	wartend := testdaten.LegeSchuelerAn(t, pool, "RB-W-"+regal.verliehen, "Wartekind", "05B")
	if _, err := pool.Exec(context.Background(), `INSERT INTO vormerkungen (titel_id, schueler_id) VALUES ($1, $2)`,
		regal.titelID, wartend); err != nil {
		t.Fatalf("Vormerkung anlegen: %v", err)
	}
	bearbeiter := testdaten.LegeBearbeiterAn(t, pool)

	broker := sse.NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	go broker.FuehreAus(ctx)
	strom := httptest.NewServer(broker.Tuer(func(*http.Request) bool { return true }))
	resp, err := http.Get(strom.URL)
	if err != nil {
		cancel()
		strom.Close()
		t.Fatalf("SSE-Strom nicht erreichbar: %v", err)
	}
	// Erst den Strom des Clients schließen, dann den Broker beenden: strom.Close wartet auf die
	// offene Verbindung.
	t.Cleanup(func() {
		if cerr := resp.Body.Close(); cerr != nil {
			t.Logf("SSE-Strom schließen: %v", cerr)
		}
		cancel()
		strom.CloseClientConnections()
		strom.Close()
	})
	zeilen := make(chan string, 32)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			if z := sc.Text(); strings.HasPrefix(z, "event: ") || strings.HasPrefix(z, "data: ") {
				zeilen <- z
			}
		}
	}()
	naechste := func() string {
		t.Helper()
		select {
		case z := <-zeilen:
			return z
		case <-time.After(3 * time.Second):
			t.Fatal("keine Zeile vom SSE-Strom innerhalb von 3 s")
			return ""
		}
	}
	if z := naechste(); z != "event: connected" {
		t.Fatalf("erste Zeile %q, erwartet event: connected", z)
	}
	naechste() // Daten zu „connected"

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}, Broker: broker}}
	antwort := scanneAls(t, srv, pool, bearbeiter, auth.RolleMitarbeiter, regal.lauf.ID, regal.verliehen)

	if n := offeneAusleihen(t, pool, regal.verliehen); n != 0 {
		t.Errorf("Ausleihe nach dem Scan noch offen (%d)", n)
	}
	if len(antwort.Warnungen) != 2 || antwort.Warnungen[0] != hinweisZurueckgebucht ||
		antwort.Warnungen[1] != "Vorgemerkt für Wartekind Test, 05B: nicht ins Regal stellen, ins Abholfach legen." {
		t.Errorf("Hinweise %q, erwartet zurückgebucht und Abholfach für Wartekind", antwort.Warnungen)
	}
	if n := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM vormerkungen
		WHERE schueler_id = $1 AND status = 'abholbereit'`, wartend); n != 1 {
		t.Errorf("Vormerkung des wartenden Kindes nicht bereitgestellt (%d abholbereit)", n)
	}
	if antwort.LaufStand == nil || *antwort.LaufStand != (repository.LaufStand{Erwartet: 2, Fehlend: 1}) {
		t.Errorf("Stand %+v, erwartet 2 erwartet und 1 fehlend: das zurückgebuchte Buch zählt als gefunden", antwort.LaufStand)
	}
	if z := naechste(); z != "event: action" {
		t.Fatalf("Meldung %q, erwartet event: action", z)
	}
	var ereignis struct {
		Art     string `json:"event"`
		LeserID string `json:"student_id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(naechste(), "data: ")), &ereignis); err != nil {
		t.Fatalf("Daten der Meldung lesen: %v", err)
	}
	if ereignis.Art != "rueckgabe" || ereignis.LeserID != regal.kind {
		t.Errorf("Meldung %+v, erwartet rueckgabe für das Kind %s", ereignis, regal.kind)
	}
}

// Wer an der Theke nicht zurücknehmen darf, bucht auch in der Inventur nichts zurück: Die Karte
// nennt das Buch als verliehen, die Ausleihe bleibt offen, und gefunden ist es nicht.
func TestScanVerbuchen_VerliehenesBuchOhneThekenrechtBleibtVerliehen(t *testing.T) {
	pool := pgtest.Pool(t)
	regal := legeVerliehenesImRegalAn(t, pool)
	bearbeiter := testdaten.LegeBearbeiterAn(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}, Broker: sse.NewBroker()}}

	antwort := scanneAls(t, srv, pool, bearbeiter, auth.RolleKollegium, regal.lauf.ID, regal.verliehen)

	if n := offeneAusleihen(t, pool, regal.verliehen); n != 1 {
		t.Errorf("offene Ausleihen nach dem Scan: %d, erwartet 1", n)
	}
	if len(antwort.Warnungen) != 1 || antwort.Warnungen[0] != hinweisVerliehen {
		t.Errorf("Hinweise %q, erwartet nur %q", antwort.Warnungen, hinweisVerliehen)
	}
	if antwort.LaufStand == nil || *antwort.LaufStand != (repository.LaufStand{Erwartet: 1, Fehlend: 1}) {
		t.Errorf("Stand %+v, erwartet 1 erwartet und 1 fehlend", antwort.LaufStand)
	}
}
