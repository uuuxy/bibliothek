package sse

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"bibliothek/pkg/closeutil"
)

// Fährt der Server herunter, während ein Arbeitsplatz einen Strom offen hat, endet erst
// FuehreAus, danach räumt httpServer.Shutdown die offenen Verbindungen ab, und jede Tür meldet
// ihr Fenster ab. Blockiert das Abmelden, wartet Shutdown bis zu seiner Frist, und main endet
// mit os.Exit(1). Arbeitsplätze sind den ganzen Tag verbunden; es träfe jedes Update.
func TestMeldeAbNachDemHerunterfahrenBlockiertNicht(t *testing.T) {
	broker := NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	go broker.FuehreAus(ctx)

	fenster := angemeldet(broker.meldeAn(""))
	if fenster == nil {
		t.Fatal("Anmeldung am laufenden Broker muss gelingen")
	}

	// Der Server fährt herunter.
	cancel()
	warteAufHerunterfahren(t, broker)

	// Jetzt läuft die Tür in ihr defer. Das darf nicht hängen.
	fertig := make(chan struct{})
	go func() {
		broker.meldeAb(fenster)
		close(fertig)
	}()

	select {
	case <-fertig:
	case <-time.After(2 * time.Second):
		t.Fatal("meldeAb nach dem Herunterfahren blockiert — genau hier hängt die " +
			"Tür, und httpServer.Shutdown läuft in seine Frist")
	}
}

// Trifft nach dem Signal noch eine Verbindung ein, wird sie abgewiesen. Eine wartende Tür hielte
// das Herunterfahren genauso auf.
func TestAnmeldungNachDemHerunterfahrenWirdAbgelehnt(t *testing.T) {
	broker := NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	go broker.FuehreAus(ctx)
	cancel()
	warteAufHerunterfahren(t, broker)

	ergebnis := make(chan chan string, 1)
	go func() { ergebnis <- angemeldet(broker.meldeAn("")) }()

	select {
	case ch := <-ergebnis:
		if ch != nil {
			t.Error("nach dem Herunterfahren darf sich kein Client mehr anmelden")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("meldeAn nach dem Herunterfahren blockiert statt abzulehnen")
	}
}

// Die Türen warten an ihrem Kanal. Bleiben die Kanäle beim Herunterfahren offen, wartet jede
// Tür bis zu 15 Sekunden auf das nächste Lebenszeichen, länger als die zehn Sekunden, die
// main.go dem Herunterfahren gibt.
func TestHerunterfahrenSchliesstOffeneStroeme(t *testing.T) {
	broker := NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	go broker.FuehreAus(ctx)

	fenster := angemeldet(broker.meldeAn(""))
	if fenster == nil {
		t.Fatal("Anmeldung muss gelingen")
	}

	cancel()

	select {
	case _, ok := <-fenster:
		if ok {
			t.Error("Kanal nach cancel() offen, erwartet geschlossen als Abbruchsignal")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("der Client-Kanal wurde beim Herunterfahren nicht geschlossen — " +
			"der Handler bliebe bis zum nächsten Heartbeat stehen")
	}
}

// warteAufHerunterfahren wartet, bis FuehreAus das Herunterfahren verarbeitet hat.
func warteAufHerunterfahren(t *testing.T, b *Broker) {
	t.Helper()
	frist := time.Now().Add(2 * time.Second)
	for time.Now().Before(frist) {
		if b.istGeschlossen() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("Broker hat das Herunterfahren nicht verarbeitet")
}

// ── Gewöhnlicher Betrieb ───────────────────────────────────────────────────────

// Der Zweck des Pakets: Mehrere Arbeitsplätze sind verbunden, eine Buchung an einem kommt an
// allen anderen an.
func TestVerteileErreichtAlleFenster(t *testing.T) {
	broker := NewBroker()

	fenster := make([]chan string, 3)
	for i := range fenster {
		fenster[i] = angemeldet(broker.meldeAn(""))
		if fenster[i] == nil {
			t.Fatalf("Client %d konnte sich nicht anmelden", i)
		}
	}

	broker.Verteile("ausleihe", `{"barcode":"B-00042"}`)

	const erwartet = "event: ausleihe\ndata: {\"barcode\":\"B-00042\"}\n\n"
	for i, c := range fenster {
		select {
		case got := <-c:
			if got != erwartet {
				t.Errorf("Client %d bekam %q, erwartet %q", i, got, erwartet)
			}
		case <-time.After(time.Second):
			t.Errorf("Client %d hat die Nachricht nicht bekommen", i)
		}
	}
}

// Ein Senden auf einen geschlossenen Kanal beendet den Prozess. meldeAb schließt den Kanal;
// Verteile darf ihn danach nicht mehr kennen.
func TestAbgemeldetesFensterBekommtNichtsMehr(t *testing.T) {
	broker := NewBroker()
	bleibt := angemeldet(broker.meldeAn(""))
	geht := angemeldet(broker.meldeAn(""))

	broker.meldeAb(geht)
	broker.Verteile("test", "{}") // sendete sonst auf einem geschlossenen Kanal

	select {
	case <-bleibt:
	case <-time.After(time.Second):
		t.Error("der verbliebene Client hätte die Nachricht bekommen müssen")
	}
	if _, ok := <-geht; ok {
		t.Error("der abgemeldete Kanal muss geschlossen sein")
	}
}

// Ein Arbeitsplatz, dessen Browser hängt, bremst die Theke nicht mit aus: Verteile sendet, ohne
// zu warten, und überspringt volle Puffer.
func TestLangsamesFensterBremstDieAnderenNicht(t *testing.T) {
	broker := NewBroker()
	langsam := angemeldet(broker.meldeAn(""))
	schnell := angemeldet(broker.meldeAn(""))

	// Den Puffer des langsamen Fensters randvoll laufen lassen.
	for i := 0; i < pufferJeFenster+5; i++ {
		broker.Verteile("fuellen", "{}")
		<-schnell // das schnelle Fenster liest mit
	}

	fertig := make(chan struct{})
	go func() { broker.Verteile("wichtig", "{}"); close(fertig) }()

	select {
	case <-fertig:
	case <-time.After(2 * time.Second):
		t.Fatal("Verteile blockiert an einem vollen Client-Puffer")
	}

	select {
	case got := <-schnell:
		if got != "event: wichtig\ndata: {}\n\n" {
			t.Errorf("der schnelle Client bekam %q", got)
		}
	case <-time.After(time.Second):
		t.Error("der schnelle Client wurde vom langsamen ausgebremst")
	}
	if len(langsam) != pufferJeFenster {
		t.Errorf("Puffer des langsamen Clients: %d, erwartet %d (voll)", len(langsam), pufferJeFenster)
	}
}

// Unter -race: Arbeitsplätze kommen und gehen, während Ereignisse verteilt werden, wie im
// Betrieb.
func TestNebenlaeufigkeit(t *testing.T) {
	broker := NewBroker()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					broker.Verteile("tick", "{}")
				}
			}
		}()
	}

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				if c := angemeldet(broker.meldeAn("")); c != nil {
					broker.meldeAb(c)
				}
			}
		}()
	}

	// Die Sender laufen, bis alle An- und Abmeldungen durch sind.
	go func() { time.Sleep(300 * time.Millisecond); close(stop) }()
	wg.Wait()
}

// ── Die Tür über echtes HTTP ───────────────────────────────────────────────────

// Geprüft wird der Weg über HTTP statt nur der Broker: die Kopfzeilen, die Begrüßung, und ein
// verteiltes Ereignis kommt beim Browser an.
func TestTuerLiefertStromUndEreignisse(t *testing.T) {
	broker := NewBroker()
	srv := httptest.NewServer(broker.Tuer(immerGueltig, ohneKonto, immerGueltig))
	defer srv.Close()

	ctx, abbrechen := context.WithCancel(context.Background())
	defer abbrechen()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("Anfrage: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Verbindung: %v", err)
	}
	defer closeutil.Schliesse(resp.Body, "sse-test response body")

	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Errorf("Content-Type = %q, erwartet text/event-stream", got)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, erwartet no-cache", got)
	}
	// HTTP/2 verbietet das Kopffeld Connection; es darf hier nicht auftauchen.
	if got := resp.Header.Get("Connection"); got == "keep-alive" {
		t.Error("Connection: keep-alive gesetzt — bricht SSE über HTTP/2")
	}

	leser := bufio.NewReader(resp.Body)
	if zeile := leseZeile(t, leser); zeile != "event: connected" {
		t.Fatalf("leseZeile(t, leser) = %q, erwartet Handshake", zeile)
	}

	// Warten, bis die Tür ihr Fenster angemeldet hat, sonst geht das Ereignis ins Leere.
	warteAufFensterzahl(t, broker, 1)
	broker.Verteile("rueckgabe", `{"barcode":"B-1"}`)

	var sah string
	for i := 0; i < 6; i++ {
		if zeile := leseZeile(t, leser); zeile == "event: rueckgabe" {
			sah = zeile
			break
		}
	}
	if sah == "" {
		t.Error("das gesendete Ereignis kam nicht im Strom an")
	}
}

// Vor jedem Lebenszeichen fragt die Tür, ob die Anmeldung noch gilt, mit der Anfrage, die den
// Strom geöffnet hat. Solange sie gilt, kommen Lebenszeichen, danach endet der Strom. Sonst läse
// ein Mithörer mit einem kopierten Token nach Abmelden oder Sperre weiter mit.
func TestTuerBeendetDenStromOhneGueltigeAnmeldung(t *testing.T) {
	broker := NewBroker()
	broker.takt = 20 * time.Millisecond
	var gueltig, fremdeAnfrage atomic.Bool
	gueltig.Store(true)
	var gefragt atomic.Int32
	gilt := func(r *http.Request) bool {
		if cookie, err := r.Cookie("session_token"); err != nil || cookie.Value != "token-vom-aufbau" {
			fremdeAnfrage.Store(true)
		}
		gefragt.Add(1)
		return gueltig.Load()
	}
	srv := httptest.NewServer(broker.Tuer(gilt, ohneKonto, immerGueltig))
	defer srv.Close()

	ctx, abbrechen := context.WithCancel(context.Background())
	defer abbrechen()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("Anfrage: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "token-vom-aufbau"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Verbindung: %v", err)
	}
	defer closeutil.Schliesse(resp.Body, "sse-test response body")

	leser := bufio.NewReader(resp.Body)
	if zeile := leseZeile(t, leser); zeile != "event: connected" {
		t.Fatalf("leseZeile(t, leser) = %q, erwartet Handshake", zeile)
	}
	for pings := 0; pings < 2; {
		if leseZeile(t, leser) == "event: ping" {
			pings++
		}
	}
	if gefragt.Load() < 2 {
		t.Errorf("zwei Lebenszeichen nach %d Prüfungen, erwartet eine vor jedem", gefragt.Load())
	}

	gueltig.Store(false)
	ende := make(chan error, 1)
	go func() {
		_, err := io.Copy(io.Discard, leser)
		ende <- err
	}()
	select {
	case err := <-ende:
		if err != nil {
			t.Errorf("Strom endete mit %v, erwartet ein geordnetes Ende", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("der Strom läuft weiter, obwohl die Anmeldung nicht mehr gilt")
	}
	if fremdeAnfrage.Load() {
		t.Error("die Prüfung bekam nicht die Anfrage mit dem Cookie vom Aufbau")
	}
}

// Kommt nach dem Signal noch eine Verbindung, antwortet die Tür mit 503, statt das
// Herunterfahren aufzuhalten.
func TestTuerWeistNachDemHerunterfahrenAb(t *testing.T) {
	broker := NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	go broker.FuehreAus(ctx)
	cancel()
	warteAufHerunterfahren(t, broker)

	srv := httptest.NewServer(broker.Tuer(immerGueltig, ohneKonto, immerGueltig))
	defer srv.Close()

	resp, err := http.Get(srv.URL) //nolint:noctx // Testaufruf gegen den lokalen Testserver
	if err != nil {
		t.Fatalf("Anfrage: %v", err)
	}
	defer closeutil.Schliesse(resp.Body, "sse-test response body")

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("HTTP %d, erwartet 503 — sonst hängt der Handler beim Herunterfahren",
			resp.StatusCode)
	}
}

// immerGueltig ist die Prüfung der Anmeldung für die Tests, die nur den Strom ansehen.
func immerGueltig(*http.Request) bool { return true }

// ohneKonto ist das Konto für die Tests, die die Grenzen nicht ansehen.
func ohneKonto(*http.Request) string { return "" }

// angemeldet ist der Kanal einer Anmeldung, ohne ihren Fehler: nil, wenn der Broker abwies.
func angemeldet(kanal chan string, _ error) chan string { return kanal }

// Ein Konto hält höchstens grenzeJeKonto Ströme offen, alle Konten zusammen höchstens
// grenzeGesamt; ein abgemeldeter Strom macht wieder Platz.
func TestMeldeAn_GrenzenJeKontoUndGesamt(t *testing.T) {
	broker := NewBroker()
	broker.grenzeJeKonto, broker.grenzeGesamt = 2, 3
	erster, err1 := broker.meldeAn("a")
	_, err2 := broker.meldeAn("a")
	if err1 != nil || err2 != nil {
		t.Fatalf("meldeAn(a) zweimal = %v, %v, erwartet angenommen", err1, err2)
	}
	if _, err := broker.meldeAn("a"); !errors.Is(err, errZuVieleJeKonto) {
		t.Errorf("meldeAn(a) zum dritten = %v, erwartet errZuVieleJeKonto", err)
	}
	if _, err := broker.meldeAn("b"); err != nil {
		t.Errorf("meldeAn(b) = %v, erwartet angenommen", err)
	}
	if _, err := broker.meldeAn("c"); !errors.Is(err, errZuVieleGesamt) {
		t.Errorf("meldeAn(c) als vierter Strom = %v, erwartet errZuVieleGesamt", err)
	}
	broker.meldeAb(erster)
	if _, err := broker.meldeAn("a"); err != nil {
		t.Errorf("meldeAn(a) nach dem Abmelden eines Stroms von a = %v, erwartet angenommen", err)
	}
}

// Über der Grenze antwortet die Tür sofort mit 429, statt einen Strom zu öffnen.
func TestTuerWeistUeberDerGrenzeAb(t *testing.T) {
	broker := NewBroker()
	broker.grenzeJeKonto = 1
	srv := httptest.NewServer(broker.Tuer(immerGueltig, func(*http.Request) string { return "konto-1" }, immerGueltig))
	defer srv.Close()
	ctx, abbrechen := context.WithCancel(context.Background())
	defer abbrechen()

	oeffne := func() *http.Response {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		if err != nil {
			t.Fatalf("Anfrage: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Verbindung: %v", err)
		}
		return resp
	}
	erster := oeffne()
	defer closeutil.Schliesse(erster.Body, "sse-test response body")
	zweiter := oeffne()
	defer closeutil.Schliesse(zweiter.Body, "sse-test response body")
	if erster.StatusCode != http.StatusOK || zweiter.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Status = %d und %d, erwartet 200 und 429", erster.StatusCode, zweiter.StatusCode)
	}
}

func leseZeile(t *testing.T, r *bufio.Reader) string {
	t.Helper()
	zeile, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("Strom lesen: %v", err)
	}
	return strings.TrimRight(zeile, "\r\n")
}

func warteAufFensterzahl(t *testing.T, b *Broker, n int) {
	t.Helper()
	frist := time.Now().Add(2 * time.Second)
	for time.Now().Before(frist) {
		b.mu.RLock()
		da := len(b.fenster)
		b.mu.RUnlock()
		if da == n {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("es waren nicht %d Clients angemeldet", n)
}

// istGeschlossen meldet, ob der Broker heruntergefahren ist. Nur die Tests fragen danach;
// deshalb steht es hier und nicht im Programm.
func (b *Broker) istGeschlossen() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.geschlossen
}

// ── Zwei Fassungen eines Ereignisses ───────────────────────────────────────────

// oeffneStrom öffnet einen Strom an der Tür und liefert einen Leser hinter der Begrüßung.
func oeffneStrom(t *testing.T, ctx context.Context, adresse string) *bufio.Reader {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, adresse, nil)
	if err != nil {
		t.Fatalf("Anfrage: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Verbindung: %v", err)
	}
	t.Cleanup(func() { closeutil.Schliesse(resp.Body, "sse-test response body") })
	leser := bufio.NewReader(resp.Body)
	if zeile := leseZeile(t, leser); zeile != "event: connected" {
		t.Fatalf("erste Zeile %q, erwartet die Begrüßung", zeile)
	}
	return leser
}

// datenNach liest bis zum Ereignis mit diesem Namen und liefert seine Datenzeile.
func datenNach(t *testing.T, leser *bufio.Reader, ereignis string) string {
	t.Helper()
	for range 40 {
		if leseZeile(t, leser) == "event: "+ereignis {
			return leseZeile(t, leser)
		}
	}
	t.Fatalf("das Ereignis %q kam nicht im Strom an", ereignis)
	return ""
}

// Die Meldung über eine Ausleihe geht an jedes angemeldete Konto, die Kennung des Lesers darin
// nur an Konten, die sie sehen dürfen. Die Tür fragt beim Aufbau und nach jedem Stück des Stroms
// neu: Wird das Recht entzogen, während der Strom offen ist, wirkt es spätestens mit dem
// nächsten Lebenszeichen.
func TestVerteileNachVertrauen_KennungNurFuerVertrauteStroeme(t *testing.T) {
	broker := NewBroker()
	broker.takt = 20 * time.Millisecond
	var thekeDarf atomic.Bool
	thekeDarf.Store(true)
	theke := httptest.NewServer(broker.Tuer(immerGueltig, ohneKonto, func(*http.Request) bool { return thekeDarf.Load() }))
	defer theke.Close()
	portal := httptest.NewServer(broker.Tuer(immerGueltig, ohneKonto, func(*http.Request) bool { return false }))
	defer portal.Close()

	ctx, abbrechen := context.WithCancel(context.Background())
	defer abbrechen()
	anDerTheke := oeffneStrom(t, ctx, theke.URL)
	imPortal := oeffneStrom(t, ctx, portal.URL)
	warteAufFensterzahl(t, broker, 2)

	const mit, ohne = `{"event":"ausleihe","student_id":"l1"}`, `{"event":"ausleihe"}`
	broker.VerteileNachVertrauen("action", mit, ohne)
	if got := datenNach(t, anDerTheke, "action"); got != "data: "+mit {
		t.Errorf("Theke bekam %q, erwartet die Fassung mit Kennung", got)
	}
	if got := datenNach(t, imPortal, "action"); got != "data: "+ohne {
		t.Errorf("Portal bekam %q, erwartet die Fassung ohne Kennung", got)
	}

	// Das Recht wird entzogen, der Strom bleibt offen. Nach dem nächsten Lebenszeichen kommt
	// auch dort nur noch die Fassung ohne Kennung.
	thekeDarf.Store(false)
	datenNach(t, anDerTheke, "ping")
	datenNach(t, anDerTheke, "ping")
	broker.VerteileNachVertrauen("action", mit, ohne)
	if got := datenNach(t, anDerTheke, "action"); got != "data: "+ohne {
		t.Errorf("nach dem Entzug bekam die Theke %q, erwartet die Fassung ohne Kennung", got)
	}
}
