package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"

	"bibliothek/internal/tuertest"
)

// Der Leser erkennt das Null-Zeichen in jeder Schreibweise, die JSON kennt, und nur dort: Ein
// aufgehobener Backslash macht aus \u0000 Text, ein anderes Zeichen nach \u ist kein Null-Zeichen.
// Byte für Byte gelesen gilt dasselbe, denn ein Rumpf kommt in Stücken an.
func TestNullZeichenLeser(t *testing.T) {
	for _, fall := range []struct {
		rumpf string
		null  bool
	}{
		{`{"a":"\u0000"}`, true},
		{`{"a":"x\u0000y"}`, true},
		{`{"\u0000":1}`, true},
		{`{"a":"\\\u0000"}`, true},
		{`{"a":"\u00001"}`, true},
		{"{\"a\":\"\x00\"}", true},
		{`{"a":"\\u0000"}`, false},
		{`{"a":"\\\\u0000"}`, false},
		{`{"a":"\u0041\u000A\u0100"}`, false},
		{`{"a":"\u000"}`, false},
		{`{"a":"C:\\temp\\u0000"}`, false},
	} {
		for name, lies := range map[string]func(io.Reader) io.Reader{
			"am Stück":     func(r io.Reader) io.Reader { return r },
			"Byte je Byte": iotest.OneByteReader,
		} {
			leser := &nullZeichenLeser{quelle: lies(strings.NewReader(fall.rumpf))}
			gelesen, err := io.ReadAll(leser)
			if null := errors.Is(err, errNullZeichen); null != fall.null {
				t.Errorf("%s %s: Null-Zeichen erkannt = %v, erwartet %v (Fehler %v)", name, fall.rumpf, null, fall.null, err)
				continue
			}
			if !fall.null && string(gelesen) != fall.rumpf {
				t.Errorf("%s %s: gelesen %q, erwartet den Rumpf unverändert", name, fall.rumpf, gelesen)
			}
		}
	}
}

// Die Gegenprobe zu den Fällen ohne Null-Zeichen: encoding/json liest sie als Text ohne das
// Zeichen. Sonst hielte der Leser etwas für harmlos, das als Null-Zeichen ankommt.
func TestNullZeichenLeser_StimmtMitEncodingJSONUeberein(t *testing.T) {
	for _, rumpf := range []string{`{"a":"\\u0000"}`, `{"a":"\\\\u0000"}`, `{"a":"\u0041\u000A\u0100"}`, `{"a":"C:\\temp\\u0000"}`} {
		var ziel map[string]string
		if err := json.Unmarshal([]byte(rumpf), &ziel); err != nil {
			t.Fatalf("%s: %v", rumpf, err)
		}
		if strings.ContainsRune(ziel["a"], 0) {
			t.Errorf("%s: encoding/json liest ein Null-Zeichen, der Leser nicht", rumpf)
		}
	}
	var ziel map[string]string
	if err := json.Unmarshal([]byte(`{"a":"\\\u0000"}`), &ziel); err != nil || !strings.ContainsRune(ziel["a"], 0) {
		t.Errorf(`{"a":"\\\u0000"}: encoding/json liest %q (%v), erwartet ein Null-Zeichen`, ziel["a"], err)
	}
}

// Am Eingang: Pfad, Abfrage und ein JSON-Rumpf mit dem Zeichen bekommen 400, und die Tür läuft
// nicht. Ein Rumpf ohne das Zeichen kommt Byte für Byte bei der Tür an, auch über den Vorlauf
// hinaus; eine hochgeladene Datei bleibt ungeprüft.
func TestNullZeichenMiddleware(t *testing.T) {
	var gelesen []byte
	var lesefehler error
	tuerLief := false
	pruefung := nullZeichenMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tuerLief = true
		gelesen, lesefehler = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	rufe := func(methode, pfad, art, rumpf string) *httptest.ResponseRecorder {
		tuerLief, gelesen, lesefehler = false, nil, nil
		req := httptest.NewRequest(methode, pfad, strings.NewReader(rumpf))
		if art != "" {
			req.Header.Set("Content-Type", art)
		}
		rec := httptest.NewRecorder()
		pruefung.ServeHTTP(rec, req)
		return rec
	}

	for _, fall := range []struct{ name, methode, pfad, art, rumpf string }{
		{"Pfad", http.MethodGet, "/api/schueler/x%00y/photo", "", ""},
		{"Abfrage", http.MethodGet, "/api/schueler?q=%00", "", ""},
		{"Abfrage, Schlüssel", http.MethodGet, "/api/schueler?%00=1", "", ""},
		{"Rumpf JSON", http.MethodPost, "/api/action", "application/json", `{"query":"A\u0000B"}`},
		{"Rumpf JSON mit Zeichensatz", http.MethodPost, "/api/action", "application/json; charset=utf-8", `{"query":"\u0000"}`},
		{"Rumpf ohne Art", http.MethodPut, "/api/anliegen/1/erledigen", "", `{"notiz":"\u0000"}`},
	} {
		rec := rufe(fall.methode, fall.pfad, fall.art, fall.rumpf)
		if rec.Code != http.StatusBadRequest || tuertest.Fehlermeldung(t, rec) != meldungNullZeichen {
			t.Errorf("%s: Status %d, Meldung %q, erwartet 400 mit %q", fall.name, rec.Code, rec.Body.String(), meldungNullZeichen)
		}
		if tuerLief {
			t.Errorf("%s: die Tür lief", fall.name)
		}
	}

	grosserText := strings.Repeat("x", 2*nullZeichenVorlauf)
	for _, fall := range []struct{ name, art, rumpf string }{
		{"JSON ohne das Zeichen", "application/json", `{"a":"\\u0000 ist Text"}`},
		{"JSON größer als der Vorlauf", "application/json", `{"a":"` + grosserText + `"}`},
		{"Datei mit Null-Bytes", "multipart/form-data; boundary=x", "--x\r\n\x00\x00\\u0000\r\n--x--\r\n"},
		{"Text", "text/csv", `a;\u0000;b`},
	} {
		rec := rufe(http.MethodPost, "/api/probe", fall.art, fall.rumpf)
		if rec.Code != http.StatusOK || lesefehler != nil {
			t.Errorf("%s: Status %d, Lesefehler %v, erwartet 200 ohne Fehler", fall.name, rec.Code, lesefehler)
		}
		if !bytes.Equal(gelesen, []byte(fall.rumpf)) {
			t.Errorf("%s: die Tür las %d Bytes, erwartet den Rumpf mit %d Bytes unverändert", fall.name, len(gelesen), len(fall.rumpf))
		}
	}

	// Hinter dem Vorlauf scheitert das Lesen in der Tür; die Tür antwortet dann selbst, wie auf
	// jeden unlesbaren Rumpf.
	rec := rufe(http.MethodPost, "/api/probe", "application/json", `{"a":"`+grosserText+`\u0000"}`)
	if rec.Code != http.StatusOK || !errors.Is(lesefehler, errNullZeichen) {
		t.Errorf("Null-Zeichen hinter dem Vorlauf: Status %d, Lesefehler %v, erwartet errNullZeichen in der Tür", rec.Code, lesefehler)
	}
}
