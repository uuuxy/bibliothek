package api

// Die Nachbuch-Tür und der Idempotenz-Schlüssel am echten Postgres (Rasterdurchgang 15.09.2026,
// OFFEN.md 5.15). Der Theken-Rechner benutzt für einen Scan EINEN Schlüssel: zuerst beim
// Online-Versand, und wenn dessen Antwort nicht ankommt, beim Nachbuchen. Bis hierher fragte die
// Tür nur, OB unter dem Schlüssel etwas steht, und ließ den Eintrag dann an jeder späteren Bewegung
// vorbei. Die Online-Buchungen laufen hier über die echte Theken-Route (ActionHandler), damit unter
// dem Schlüssel genau das steht, was der Online-Weg speichert.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/repository"
	"bibliothek/sse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type nbTuer struct {
	pool                    *pgxpool.Pool
	srv                     *Server
	online                  service.OmniboxService
	nachbuch                service.NachbuchService
	staff, anna, ben, carla string
	code, exemplarID        string
}

func nbTuerAufbau(t *testing.T) *nbTuer {
	t.Helper()
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	w := &nbTuer{pool: pool}
	eins := func(was, sql string, args ...any) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %v", was, err)
		}
		return id
	}
	w.staff = eins("Mitarbeiter", `INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ($1, 'Tür', 'Prüfer', $2, 'mitarbeiter', true) RETURNING id`, "MA-"+suffix, "nbtuer-"+suffix+"@schule.invalid")
	w.anna = eins("Anna", `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr) VALUES ($1, 'Anna', 'Erste', '07B', 2031) RETURNING id`, "S-TA-"+suffix)
	w.ben = eins("Ben", `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr) VALUES ($1, 'Ben', 'Zweiter', '07B', 2031) RETURNING id`, "S-TB-"+suffix)
	w.carla = eins("Carla", `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt, block_reason)
		VALUES ($1, 'Carla', 'Gesperrt', '07B', 2031, true, 'Testsperre') RETURNING id`, "S-TC-"+suffix)
	titelID := seedMonitorTitel(t, pool, "Nachbuch-Tür-Band", "NBT", false, 0)
	w.code = "B-NBT-" + suffix
	w.exemplarID = exemplar(t, pool, titelID, w.code, true, "")

	studentRepo := repository.NewStudentRepository(pool)
	bookRepo := repository.NewBookRepository(pool)
	loanRepo := repository.NewLoanRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	loanSvc := service.NewLoanService(pool, studentRepo, bookRepo, loanRepo, auditRepo)
	deviceSvc := service.NewDeviceService(pool, studentRepo, loanRepo, auditRepo)
	w.online = service.NewOmniboxService(pool, studentRepo, bookRepo, userRepo, loanRepo, loanSvc, deviceSvc)
	w.nachbuch = service.NewNachbuchService(pool, studentRepo, bookRepo, userRepo, loanRepo, auditRepo)
	w.srv = &Server{DB: &db.Database{Pool: pool}, Broker: sse.NewBroker()}
	return w
}

func (w *nbTuer) sitzung(req *http.Request) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{Rolle: auth.Role("mitarbeiter"), UserID: w.staff}))
}

// onlineScan scannt das Buch an der Online-Theke; schueler "" heißt: niemand geladen.
func (w *nbTuer) onlineScan(t *testing.T, schluessel, schueler string) (int, ActionResponse) {
	t.Helper()
	body := fmt.Sprintf(`{"query":%q,"idempotency_key":%q`, w.code, schluessel)
	if schueler != "" {
		body += fmt.Sprintf(`,"active_student_id":%q`, schueler)
	}
	rec := httptest.NewRecorder()
	w.srv.ActionHandler(w.online)(rec, w.sitzung(httptest.NewRequest(http.MethodPost, "/api/action", strings.NewReader(body+"}"))))
	var resp ActionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Online-Scan: Status %d, keine JSON-Antwort: %v (%s)", rec.Code, err, rec.Body.String())
	}
	return rec.Code, resp
}

func (w *nbTuer) eintrag(schluessel, absicht string, schueler *string, gescannt time.Time) map[string]any {
	e := map[string]any{"schluessel": schluessel, "absicht": absicht, "barcode": w.code, "gescannt_am": gescannt}
	if schueler != nil {
		e["schueler_id"] = *schueler
	}
	return e
}

func (w *nbTuer) nachbuchenMit(t *testing.T, svc service.NachbuchService, eintraege ...map[string]any) []NachbuchenErgebnis {
	t.Helper()
	body, err := json.Marshal(map[string]any{"eintraege": eintraege})
	if err != nil {
		t.Fatalf("Rumpf: %v", err)
	}
	rec := httptest.NewRecorder()
	w.srv.NachbuchenHandler(svc)(rec, w.sitzung(httptest.NewRequest(http.MethodPost, "/api/action/nachbuchen", strings.NewReader(string(body)))))
	if rec.Code != http.StatusOK {
		t.Fatalf("Nachbuchen: Status %d, %s", rec.Code, rec.Body.String())
	}
	var resp NachbuchenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Antwort: %v", err)
	}
	if len(resp.Ergebnisse) != len(eintraege) {
		t.Fatalf("%d Ergebnisse für %d Einträge", len(resp.Ergebnisse), len(eintraege))
	}
	return resp.Ergebnisse
}

func (w *nbTuer) nachbuchen(t *testing.T, eintraege ...map[string]any) []NachbuchenErgebnis {
	t.Helper()
	return w.nachbuchenMit(t, w.nachbuch, eintraege...)
}

func (w *nbTuer) offen(t *testing.T) (n int, bei string) {
	t.Helper()
	if err := w.pool.QueryRow(context.Background(), `
		SELECT count(*), coalesce(max(schueler_id::text), '') FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`, w.exemplarID).Scan(&n, &bei); err != nil {
		t.Fatalf("offene Ausleihen: %v", err)
	}
	return
}

func (w *nbTuer) meldungen(t *testing.T, schluessel ...string) int {
	t.Helper()
	var n int
	if err := w.pool.QueryRow(context.Background(), `SELECT count(*) FROM nachbuch_meldungen WHERE idempotency_key = ANY($1::uuid[])`, schluessel).Scan(&n); err != nil {
		t.Fatalf("Meldungen: %v", err)
	}
	return n
}

// Der Online-Versand hat die Ausleihe vollständig gebucht, die Antwort erreichte die Theke nicht;
// Anna gibt das Buch danach an einer anderen Theke zurück. Das Nachbuchen darf es ihr nicht erneut
// ausleihen — sonst wird sie für ein abgegebenes Buch gemahnt.
func TestNachbuchen_Schluessel_OnlineGebuchtDanachZurueckgegeben(t *testing.T) {
	w := nbTuerAufbau(t)
	scan := time.Now()
	k := uuid.NewString()
	if code, r := w.onlineScan(t, k, w.anna); code != http.StatusOK || r.Type != "ausleihe" {
		t.Fatalf("Online-Ausleihe: %d %q", code, r.Type)
	}
	if code, r := w.onlineScan(t, uuid.NewString(), ""); code != http.StatusOK || r.Type != "rueckgabe" {
		t.Fatalf("Online-Rückgabe: %d %q", code, r.Type)
	}

	erg := w.nachbuchen(t, w.eintrag(k, "ausleihe", &w.anna, scan))
	if erg[0].Ergebnis != nachbuchBereitsGebucht {
		t.Errorf("Ergebnis %q, erwartet %q", erg[0].Ergebnis, nachbuchBereitsGebucht)
	}
	if n, _ := w.offen(t); n != 0 {
		t.Errorf("Anna hat das Buch zurückgegeben, das Nachbuchen hat es ihr erneut ausgeliehen (%d offen)", n)
	}
	if erg[0].Daten == nil || erg[0].Daten.Type != "ausleihe" {
		t.Errorf("die Antwort nennt die gebuchte Wirkung nicht: %+v", erg[0].Daten)
	}
	if m := w.meldungen(t, k); m != 0 {
		t.Errorf("%d Meldungen für einen vollständig gebuchten Scan", m)
	}
}

// Die Theke schickt eine Portion nach einem Timeout erneut, obwohl der erste Aufruf gebucht hat.
// Die Wiederholung bekommt dieselben Ergebnisse, und es entstehen keine Meldungen.
func TestNachbuchen_Schluessel_WiederholtePortion(t *testing.T) {
	w := nbTuerAufbau(t)
	jetzt := time.Now()
	k1, k2 := uuid.NewString(), uuid.NewString()
	portion := []map[string]any{
		w.eintrag(k1, "ausleihe", &w.anna, jetzt.Add(-10*time.Minute)),
		w.eintrag(k2, "rueckgabe", nil, jetzt.Add(-5*time.Minute)),
	}
	erste := w.nachbuchen(t, portion...)
	if erste[0].Ergebnis != repository.NachbuchAusgeliehen || erste[1].Ergebnis != repository.NachbuchZurueckgegeben {
		t.Fatalf("erster Aufruf: %q, %q", erste[0].Ergebnis, erste[1].Ergebnis)
	}
	zweite := w.nachbuchen(t, portion...)
	for i := range erste {
		if zweite[i].Ergebnis != erste[i].Ergebnis {
			t.Errorf("Eintrag %d: Wiederholung %q (%s), erster Aufruf %q", i, zweite[i].Ergebnis, zweite[i].Grund, erste[i].Ergebnis)
		}
	}
	if m := w.meldungen(t, k1, k2); m != 0 {
		t.Errorf("%d Meldungen für eine vollständig gebuchte Portion", m)
	}
	if n, _ := w.offen(t); n != 0 {
		t.Errorf("die Wiederholung hat gebucht: %d offen", n)
	}
}

// Der Zweck der Ausnahme: Annas Sitzung scannt online ein Buch, das auf Ben steht. Der Server
// bucht nur die Fremdrückgabe; die Antwort geht verloren, die Ausleihe an Anna liegt in der
// Warteschlange. Das Nachbuchen holt sie nach, obwohl der Scan vor der Rücknahme liegt.
func TestNachbuchen_Schluessel_NachFremdrueckgabeWirdAusgeliehen(t *testing.T) {
	w := nbTuerAufbau(t)
	if code, _ := w.onlineScan(t, uuid.NewString(), w.ben); code != http.StatusOK {
		t.Fatalf("Online-Ausleihe an Ben: %d", code)
	}
	scan := time.Now()
	k := uuid.NewString()
	if code, r := w.onlineScan(t, k, w.anna); code != http.StatusOK || r.Type != "rueckgabe" || !r.Fremdrueckgabe {
		t.Fatalf("Fremdrückgabe: %d %q fremd=%v", code, r.Type, r.Fremdrueckgabe)
	}
	erg := w.nachbuchen(t, w.eintrag(k, "ausleihe", &w.anna, scan))
	if erg[0].Ergebnis != repository.NachbuchAusgeliehen {
		t.Errorf("Ergebnis %q (%s), erwartet ausgeliehen", erg[0].Ergebnis, erg[0].Grund)
	}
	if n, bei := w.offen(t); n != 1 || bei != w.anna {
		t.Errorf("%d offen bei %s, erwartet eine bei Anna", n, bei)
	}
	var ausgeliehen, rueckgabeBen time.Time
	if err := w.pool.QueryRow(context.Background(), `
		SELECT (SELECT ausgeliehen_am FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL),
		       (SELECT max(rueckgabe_am) FROM ausleihen WHERE exemplar_id = $1 AND schueler_id = $2)`, w.exemplarID, w.ben).Scan(&ausgeliehen, &rueckgabeBen); err != nil {
		t.Fatalf("Zeitpunkte: %v", err)
	}
	if ausgeliehen.Before(rueckgabeBen) {
		t.Errorf("Annas Ausleihe (%v) beginnt vor Bens Rückgabe (%v)", ausgeliehen, rueckgabeBen)
	}
}

// Dieselbe Lage, aber das Buch wurde nach der Fremdrückgabe bewegt (Ben leiht es erneut und gibt es
// zurück). Dann gilt die Ausnahme nicht: Der Scan ist veraltet.
func TestNachbuchen_Schluessel_NachFremdrueckgabeBewegt(t *testing.T) {
	w := nbTuerAufbau(t)
	if code, _ := w.onlineScan(t, uuid.NewString(), w.ben); code != http.StatusOK {
		t.Fatalf("Online-Ausleihe an Ben: %d", code)
	}
	scan := time.Now()
	k := uuid.NewString()
	if code, r := w.onlineScan(t, k, w.anna); code != http.StatusOK || !r.Fremdrueckgabe {
		t.Fatalf("Fremdrückgabe: %d fremd=%v", code, r.Fremdrueckgabe)
	}
	if code, r := w.onlineScan(t, uuid.NewString(), w.ben); code != http.StatusOK || r.Type != "ausleihe" {
		t.Fatalf("erneute Ausleihe an Ben: %d %q", code, r.Type)
	}
	if code, r := w.onlineScan(t, uuid.NewString(), ""); code != http.StatusOK || r.Type != "rueckgabe" {
		t.Fatalf("Rückgabe von Ben: %d %q", code, r.Type)
	}
	erg := w.nachbuchen(t, w.eintrag(k, "ausleihe", &w.anna, scan))
	if erg[0].Ergebnis != repository.NachbuchVeraltet {
		t.Errorf("Ergebnis %q (%s), erwartet veraltet", erg[0].Ergebnis, erg[0].Grund)
	}
	if n, _ := w.offen(t); n != 0 {
		t.Errorf("das Buch liegt im Regal, das Nachbuchen hat es Anna ausgeliehen (%d offen)", n)
	}
}

// Läuft die Online-Buchung unter dem Schlüssel noch, darf die Tür nicht daneben buchen: Der Eintrag
// bleibt auf dem Rechner.
func TestNachbuchen_Schluessel_OnlineBuchungLaeuftNoch(t *testing.T) {
	w := nbTuerAufbau(t)
	k := uuid.NewString()
	if _, err := w.pool.Exec(context.Background(), `INSERT INTO idempotency_keys (idempotency_key, response_data, status_code) VALUES ($1, '{"in_arbeit": true}', 0)`, k); err != nil {
		t.Fatalf("Reservierung: %v", err)
	}
	erg := w.nachbuchen(t, w.eintrag(k, "ausleihe", &w.anna, time.Now()))
	if erg[0].Ergebnis != nachbuchWiederholen {
		t.Errorf("Ergebnis %q, erwartet %q", erg[0].Ergebnis, nachbuchWiederholen)
	}
	if n, _ := w.offen(t); n != 0 {
		t.Errorf("neben der laufenden Online-Buchung gebucht: %d offen", n)
	}
}

// Hat der Online-Versand nichts gebucht (Sperre, 403 gespeichert), ist der Eintrag ein gewöhnlicher
// Scan — der Wächter gilt. Das Buch wurde seitdem bewegt: veraltet.
func TestNachbuchen_Schluessel_OnlineAbgewiesenWaechterGilt(t *testing.T) {
	w := nbTuerAufbau(t)
	scan := time.Now()
	k := uuid.NewString()
	if code, _ := w.onlineScan(t, k, w.carla); code != http.StatusForbidden {
		t.Fatalf("Online-Scan für die gesperrte Carla: Status %d, erwartet 403", code)
	}
	if code, _ := w.onlineScan(t, uuid.NewString(), w.ben); code != http.StatusOK {
		t.Fatalf("Ausleihe an Ben: %d", code)
	}
	if code, _ := w.onlineScan(t, uuid.NewString(), ""); code != http.StatusOK {
		t.Fatalf("Rückgabe von Ben: %d", code)
	}
	erg := w.nachbuchen(t, w.eintrag(k, "ausleihe", &w.carla, scan))
	if erg[0].Ergebnis != repository.NachbuchVeraltet {
		t.Errorf("Ergebnis %q (%s), erwartet veraltet", erg[0].Ergebnis, erg[0].Grund)
	}
}

// Zwei Aufrufe mit derselben Portion gleichzeitig: genau eine Buchung, keine Meldung, und keiner
// bekommt ein anderes Ergebnis als „ausgeliehen" oder „wiederholen".
func TestNachbuchen_Schluessel_ZweiGleichzeitigeAufrufe(t *testing.T) {
	w := nbTuerAufbau(t)
	k := uuid.NewString()
	e := w.eintrag(k, "ausleihe", &w.anna, time.Now().Add(-time.Minute))
	var wg sync.WaitGroup
	ergebnisse := make([]string, 2)
	for i := range ergebnisse {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ergebnisse[i] = w.nachbuchen(t, e)[0].Ergebnis
		}(i)
	}
	wg.Wait()
	ausgeliehen := 0
	for _, r := range ergebnisse {
		switch r {
		case repository.NachbuchAusgeliehen:
			ausgeliehen++
		case nachbuchWiederholen:
		default:
			t.Errorf("Ergebnis %q — erwartet ausgeliehen oder wiederholen (%v)", r, ergebnisse)
		}
	}
	if ausgeliehen == 0 {
		t.Errorf("keiner der Aufrufe hat ausgeliehen: %v", ergebnisse)
	}
	if n, bei := w.offen(t); n != 1 || bei != w.anna {
		t.Errorf("%d offen bei %s, erwartet eine bei Anna", n, bei)
	}
	if m := w.meldungen(t, k); m != 0 {
		t.Errorf("%d Meldungen", m)
	}
}

type nachbuchFehler struct{}

func (nachbuchFehler) Nachbuchen(context.Context, service.NachbuchEintrag) (*service.NachbuchErgebnis, error) {
	return nil, errors.New("datenbank weg")
}

// Scheitert das Nachbuchen am Server, bleibt die Online-Antwort unter dem Schlüssel stehen — sonst
// verlöre die nächste Runde, dass der Online-Versand die Fremdrückgabe schon gebucht hat.
func TestNachbuchen_Schluessel_ServerfehlerLaesstOnlineAntwortStehen(t *testing.T) {
	w := nbTuerAufbau(t)
	if code, _ := w.onlineScan(t, uuid.NewString(), w.ben); code != http.StatusOK {
		t.Fatalf("Online-Ausleihe an Ben: %d", code)
	}
	k := uuid.NewString()
	if code, r := w.onlineScan(t, k, w.anna); code != http.StatusOK || !r.Fremdrueckgabe {
		t.Fatalf("Fremdrückgabe: %d", code)
	}
	vorher, err := repository.LiesIdempotenzAntwort(context.Background(), w.pool, k)
	if err != nil {
		t.Fatalf("Antwort vorher: %v", err)
	}
	erg := w.nachbuchenMit(t, nachbuchFehler{}, w.eintrag(k, "ausleihe", &w.anna, time.Now()))
	if erg[0].Ergebnis != nachbuchWiederholen {
		t.Errorf("Ergebnis %q, erwartet %q", erg[0].Ergebnis, nachbuchWiederholen)
	}
	nachher, err := repository.LiesIdempotenzAntwort(context.Background(), w.pool, k)
	if err != nil {
		t.Fatalf("Antwort nachher: %v", err)
	}
	if nachher.Status != vorher.Status || string(nachher.Daten) != string(vorher.Daten) {
		t.Errorf("Online-Antwort verändert: %d %s → %d %s", vorher.Status, vorher.Daten, nachher.Status, nachher.Daten)
	}
}
