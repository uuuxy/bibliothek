package api

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/pkg/closeutil"
)

// Diese Tests laufen gegen einen ECHTEN httptest-Server, nicht gegen einen
// ResponseRecorder — und zwar zwingend: http.ResponseController.SetReadDeadline
// braucht die darunterliegende Verbindung. Ein Recorder kennt sie nicht und meldet
// ErrNotSupported; die Middleware liefe grün durch, ohne je eine Frist gesetzt zu
// haben. Genau die Sorte Gate, die man sonst für wirksam hält.

// langsamerRumpf schickt Kopfzeilen mit angekündigter Länge, dann ein erstes Byte,
// wartet und schickt den Rest. Das ist der Slowloris-Fall: formal korrekt, beliebig
// langsam. Geliefert wird der Statuscode des Servers — oder ein Fehler, wenn er die
// Verbindung vorher gekappt hat.
func langsamerRumpf(t *testing.T, adresse, pfad string, pause time.Duration) (int, error) {
	t.Helper()

	c, err := net.Dial("tcp", adresse)
	if err != nil {
		t.Fatalf("Verbindung zum Testserver fehlgeschlagen: %v", err)
	}
	defer closeutil.LogClose(c, "lesefrist-test verbindung")

	kopf := fmt.Sprintf("POST %s HTTP/1.1\r\nHost: x\r\nContent-Length: 4\r\n\r\n", pfad)
	if _, err := io.WriteString(c, kopf+"a"); err != nil {
		return 0, err
	}

	time.Sleep(pause)

	if _, err := io.WriteString(c, "bcd"); err != nil {
		return 0, err
	}
	if err := c.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return 0, err
	}

	antwort := make([]byte, 15)
	if _, err := io.ReadFull(c, antwort); err != nil {
		return 0, err
	}
	var status int
	if _, err := fmt.Sscanf(string(antwort), "HTTP/1.1 %d", &status); err != nil {
		return 0, err
	}
	return status, nil
}

// baueLesefristServer startet einen Server mit absichtlich winziger ReadTimeout und
// derselben Middleware wie die Anwendung.
func baueLesefristServer(t *testing.T, readTimeout time.Duration) *httptest.Server {
	t.Helper()

	handler := ErweitereLesefristFuerLangeUploads(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			w.WriteHeader(http.StatusRequestTimeout)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	srv := httptest.NewUnstartedServer(handler)
	srv.Config.ReadTimeout = readTimeout
	srv.Config.ReadHeaderTimeout = readTimeout
	srv.Start()
	t.Cleanup(srv.Close)
	return srv
}

// Der Kern des Ganzen: Ein gewöhnlicher Endpunkt lässt sich nicht beliebig lange
// hinhalten. Vor der Änderung stand am Server nur ReadHeaderTimeout — die Kopfzeilen
// waren begrenzt, der Rumpf nicht, und diese Verbindung wäre offen geblieben.
func TestLesefrist_LangsamerRumpfWirdAbgeschnitten(t *testing.T) {
	srv := baueLesefristServer(t, 300*time.Millisecond)

	status, err := langsamerRumpf(t, srv.Listener.Addr().String(), "/api/schueler", 900*time.Millisecond)
	if err == nil && status == http.StatusOK {
		t.Fatal("langsamer Rumpf wurde vollständig angenommen — die Lesefrist greift nicht")
	}
}

// Die Gegenprobe, ohne die die erste Prüfung wertlos wäre: Für die Import-Endpunkte
// MUSS derselbe langsame Rumpf durchgehen. Sonst hätte man die Slowloris-Bremse mit
// einem Abbruch mitten im Littera-Import (100 MB über eine Schulleitung) bezahlt.
func TestLesefrist_ImportpfadDarfLangsamSein(t *testing.T) {
	srv := baueLesefristServer(t, 300*time.Millisecond)

	status, err := langsamerRumpf(t, srv.Listener.Addr().String(), "/api/import/littera", 900*time.Millisecond)
	if err != nil {
		t.Fatalf("Import-Upload wurde abgebrochen, obwohl die Frist verlängert sein sollte: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("Status = %d, want 200 — der Import-Pfad bekam die lange Frist nicht", status)
	}
}

// Der Fall, der bei dieser Änderung am teuersten schiefgehen konnte: Eine Antwort, die
// länger dauert als ReadTimeout, darf NICHT abgeschnitten werden.
//
// Genau daran hängt der SSE-Stream (/events) — eine Antwort, die definitionsgemäß nie
// endet. Wäre die Lesefrist auch eine Schreibfrist, verlöre die Oberfläche reihum ihre
// Live-Aktualisierung, und zwar ohne Fehler: Der Browser baut die Verbindung neu auf,
// es fällt nur als "manchmal aktualisiert es nicht" auf. Deshalb steht am Server
// bewusst KEIN WriteTimeout (siehe main.go) — dieser Test hält fest, dass ReadTimeout
// allein das nicht heimlich nachholt.
func TestLesefrist_LangeAntwortWirdNichtAbgeschnitten(t *testing.T) {
	const readTimeout = 300 * time.Millisecond

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("ResponseWriter kann nicht flushen — Testaufbau stimmt nicht")
			return
		}
		if _, err := io.WriteString(w, "data: erstes\n\n"); err != nil {
			return
		}
		flusher.Flush()

		// Deutlich länger als die Lesefrist warten und danach weiterschreiben.
		time.Sleep(3 * readTimeout)

		if _, err := io.WriteString(w, "data: zweites\n\n"); err != nil {
			t.Errorf("Schreiben nach Ablauf der Lesefrist schlug fehl: %v", err)
			return
		}
		flusher.Flush()
	}))
	srv.Config.ReadTimeout = readTimeout
	srv.Config.ReadHeaderTimeout = readTimeout
	srv.Start()
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/events")
	if err != nil {
		t.Fatalf("Verbindung fehlgeschlagen: %v", err)
	}
	defer closeutil.LogClose(resp.Body, "lesefrist-test response body")

	koerper, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Stream wurde abgebrochen: %v", err)
	}
	if !strings.Contains(string(koerper), "data: zweites") {
		t.Fatalf("zweiter Abschnitt fehlt — die Antwort wurde abgeschnitten. Empfangen: %q", koerper)
	}
}

// Hält die Zuordnung fest: Die Lesefrist benutzt DIESELBE Pfadliste wie die
// Bearbeitungsfrist. Zwei Listen wären zwei Wahrheiten darüber, was ein langer
// Vorgang ist.
func TestLesefrist_NutztDieselbePfadliste(t *testing.T) {
	for _, pfad := range langLaufendePfade {
		if RequestFrist(pfad, StandardLesefrist) != LangLaufendeFrist {
			t.Errorf("%s gilt nicht als langlaufend", pfad)
		}
	}
	if RequestFrist("/api/schueler", StandardLesefrist) != StandardLesefrist {
		t.Error("ein gewöhnlicher Pfad bekam die lange Frist")
	}
}
