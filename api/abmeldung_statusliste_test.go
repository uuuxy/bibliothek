package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"bibliothek/auth"

	"github.com/pashagolub/pgxmock/v5"
)

// Die Statusliste der Oberfläche ist genau die Menge der Antworten des Abmeldens mit
// Löschcookie.
//
// Zwei Dateien teilen sich eine Verabredung. Die Tür (abmeldung.go) setzt das Löschcookie und
// antwortet 200 oder 503; die Oberfläche (frontend/src/lib/stores/authStore.svelte.js,
// abmeldungZugestellt) hält genau diese beiden Codes für „das Löschcookie kam an" und räumt
// daraufhin den Merker „Abmeldung ausstehend". Ohne diesen Test könnte sich jede Seite ändern,
// ohne dass die andere rot wird.
//
// Die gefährliche Richtung: Nimmt jemand dem 503-Zweig das Löschcookie („wenn wir nicht
// widerrufen konnten, melden wir auch nicht ab"), hält die Oberfläche die Abmeldung weiter für
// zugestellt und löscht den Merker. Das HttpOnly-Cookie bleibt im Browser, und das nächste
// Neuladen am geteilten Theken-Rechner meldet die vorige Person wieder an.
//
// Deshalb wird die Seite des Servers hier gemessen, nicht aufgezählt: Die Tür läuft durch jeden
// Pfad, in dem ein Sitzungscookie vorliegt, und die Antworten sagen selbst, ob sie das
// Löschcookie tragen. Auf der Seite der Oberfläche gilt nur der Rumpf der Funktion: Der JSDoc
// darüber nennt 429 und 502/504 als Gegenbeispiele, ein Kommentar im Rumpf könnte es ebenso,
// und ein Test, der Kommentare mitliest, fände dort Codes, die nie gelten.
func TestAbmeldung_StatuslisteDesClientsIstDieMengeMitLoeschcookie(t *testing.T) {
	server := abmeldeAntwortenMitLoeschcookie(t)
	client := statuslisteDerOberflaeche(t)

	for _, code := range sortierteCodes(server) {
		if !client[code] {
			t.Errorf("Der Abmelde-Handler antwortet %d MIT Löschcookie, abmeldungZugestellt hält "+
				"das nicht für zugestellt: Der Merker bleibt stehen, der nächste Start meldet ohne "+
				"Not ab. Code in authStore.svelte.js ergänzen.", code)
		}
	}
	for _, code := range sortierteCodes(client) {
		if !server[code] {
			t.Errorf("abmeldungZugestellt hält %d für zugestellt, aber keine Antwort des Handlers "+
				"trägt bei %d das Löschcookie: Der Client löscht den Merker, das Sitzungscookie "+
				"bleibt — am geteilten Theken-Rechner meldet das nächste Neuladen die vorige Person "+
				"wieder an. Entweder das Löschcookie zurück in diesen Zweig oder den Code aus der "+
				"Liste im Client nehmen.", code, code)
		}
	}
}

// abmeldeAntwortenMitLoeschcookie fährt die Tür durch jeden Pfad mit vorliegendem
// Sitzungscookie und sammelt die Statuscodes, deren Antwort das Löschcookie trägt. Der Pfad ohne
// Sitzungscookie fehlt absichtlich: Dort gibt es nichts zu löschen, und die Oberfläche hat auch
// nichts, was sie fälschlich behalten könnte.
func abmeldeAntwortenMitLoeschcookie(t *testing.T) map[int]bool {
	t.Helper()
	pfade := []struct {
		name string
		lauf func(t *testing.T) *httptest.ResponseRecorder
	}{
		{"Widerruf gelingt", func(t *testing.T) *httptest.ResponseRecorder {
			s, mock := abmeldeServer(t)
			erwarteSperrlisteFrei(mock, auth.RolleMitarbeiter)
			mock.ExpectExec("INSERT INTO revoked_tokens").
				WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
				WillReturnResult(pgxmock.NewResult("INSERT", 1))
			return abmeldenMitToken(t, s)
		}},
		{"Widerruf scheitert", func(t *testing.T) *httptest.ResponseRecorder {
			s, mock := abmeldeServer(t)
			erwarteSperrlisteFrei(mock, auth.RolleMitarbeiter)
			mock.ExpectExec("INSERT INTO revoked_tokens").
				WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
				WillReturnError(errors.New("connection reset by peer"))
			return abmeldenMitToken(t, s)
		}},
		{"Prüfung gestört", func(t *testing.T) *httptest.ResponseRecorder {
			s, mock := abmeldeServer(t)
			mock.ExpectQuery("revoked_tokens").
				WithArgs(pgxmock.AnyArg()).
				WillReturnError(errors.New("context deadline exceeded"))
			return abmeldenMitToken(t, s)
		}},
		{"Token unbrauchbar", func(t *testing.T) *httptest.ResponseRecorder {
			s, mock := abmeldeServer(t)
			mock.ExpectQuery("revoked_tokens").
				WithArgs(pgxmock.AnyArg()).
				WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
			req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
			req.AddCookie(&http.Cookie{Name: "session_token", Value: "kein.gueltiges.token"})
			rec := httptest.NewRecorder()
			s.abmelden()(rec, req)
			return rec
		}},
	}

	mit, ohne := map[int]bool{}, map[int]bool{}
	for _, p := range pfade {
		rec := p.lauf(t)
		if loeschcookieGesetzt(rec) {
			mit[rec.Code] = true
		} else {
			ohne[rec.Code] = true
		}
	}
	// Ein Code, der mal mit und mal ohne Löschcookie kommt, wäre für die Oberfläche nicht
	// zu deuten: Sie sieht nur den Status, nie das HttpOnly-Cookie.
	for _, code := range sortierteCodes(mit) {
		if ohne[code] {
			t.Errorf("Status %d kommt vom Handler mal mit und mal ohne Löschcookie — der Client "+
				"kann ihn nicht deuten", code)
		}
	}
	return mit
}

var (
	jsZeilenkommentar = regexp.MustCompile(`//[^\n]*`)
	jsBlockkommentar  = regexp.MustCompile(`(?s)/\*.*?\*/`)
	jsStatusVergleich = regexp.MustCompile(`status\s*===\s*(\d{3})`)
)

// statuslisteDerOberflaeche liest die Codes aus dem Rumpf von abmeldungZugestellt. Fehlt die
// Funktion oder steht kein `status === NNN` mehr darin, ist das ein Fehler dieses Tests, kein
// grüner Lauf: Die Form ist die einzige, die er liest.
func statuslisteDerOberflaeche(t *testing.T) map[int]bool {
	t.Helper()
	pfad := filepath.Join("..", "frontend", "src", "lib", "stores", "authStore.svelte.js")
	roh, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatalf("Client-Datei nicht lesbar: %v", err)
	}
	quelle := string(roh)
	start := strings.Index(quelle, "function abmeldungZugestellt(")
	if start < 0 {
		t.Fatalf("%s: function abmeldungZugestellt nicht gefunden — wurde sie umbenannt, muss "+
			"dieses Gate mitziehen", pfad)
	}
	rest := quelle[start:]
	rumpfStart, rumpfEnde := strings.Index(rest, "{"), strings.Index(rest, "\n}")
	if rumpfStart < 0 || rumpfEnde < rumpfStart {
		t.Fatalf("%s: Rumpf von abmeldungZugestellt nicht abgrenzbar", pfad)
	}
	rumpf := rest[rumpfStart:rumpfEnde]
	rumpf = jsBlockkommentar.ReplaceAllString(rumpf, "")
	rumpf = jsZeilenkommentar.ReplaceAllString(rumpf, "")

	liste := map[int]bool{}
	for _, m := range jsStatusVergleich.FindAllStringSubmatch(rumpf, -1) {
		code, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("%s: %q ist keine Zahl: %v", pfad, m[1], err)
		}
		liste[code] = true
	}
	if len(liste) == 0 {
		t.Fatalf("%s: kein `status === NNN` im Rumpf von abmeldungZugestellt — die Funktion hat "+
			"eine Form, die dieses Gate nicht liest", pfad)
	}
	return liste
}

func sortierteCodes(m map[int]bool) []int {
	codes := make([]int, 0, len(m))
	for c := range m {
		codes = append(codes, c)
	}
	sort.Ints(codes)
	return codes
}
