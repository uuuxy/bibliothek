package apierrors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// leseFehlermeldung liest die eine Form der Fehlerantwort {"error": "..."} und bricht ab, wenn
// der Rumpf abweicht (anderer Schlüssel, kein JSON).
func leseFehlermeldung(t *testing.T, rumpf string) string {
	t.Helper()
	var gelesen map[string]string
	if err := json.Unmarshal([]byte(rumpf), &gelesen); err != nil {
		t.Fatalf("Fehler-Body ist kein JSON: %q (%v)", rumpf, err)
	}
	if len(gelesen) != 1 {
		t.Fatalf("Fehler-Body muss genau {\"error\":...} sein, war: %q", rumpf)
	}
	meldung, ok := gelesen["error"]
	if !ok {
		t.Fatalf("Fehler-Body ohne \"error\"-Feld: %q", rumpf)
	}
	return meldung
}

// Unter 500 und ohne Text aus der Datenbank bleibt die Meldung, wie sie ist.
func TestAntworteFehler_BedienfehlerBehaeltSeineMeldung(t *testing.T) {
	rec := httptest.NewRecorder()
	AntworteFehler(rec, http.StatusConflict, errors.New("benutzer hat noch aktive ausleihen"))

	if rec.Code != http.StatusConflict {
		t.Errorf("Status: erwartet 409, war %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type: erwartet application/json, war %q", ct)
	}
	if meldung := leseFehlermeldung(t, rec.Body.String()); meldung != "benutzer hat noch aktive ausleihen" {
		t.Errorf("Meldung verfälscht: %q", meldung)
	}
}

// Ein 500 gibt keine Einzelheit des Servers heraus.
func TestAntworteFehler_500IstNeutral(t *testing.T) {
	rec := httptest.NewRecorder()
	AntworteFehler(rec, http.StatusInternalServerError,
		errors.New("pgx: SELECT * FROM geheim WHERE token = 'abc123'"))

	rumpf := rec.Body.String()
	meldung := leseFehlermeldung(t, rumpf)
	for _, leck := range []string{"pgx", "SELECT", "geheim", "abc123"} {
		if strings.Contains(meldung, leck) {
			t.Errorf("interner Detail-Leak im 500-Body (%q): %q", leck, rumpf)
		}
	}
}

// Auch unter 500 geht ein Text aus der Datenbank (SQL, Bedingung) nicht hinaus, sonst läse die
// Oberfläche den Aufbau der Tabellen.
func TestAntworteFehler_DatenbanktextAuchUnter500Gefiltert(t *testing.T) {
	rec := httptest.NewRecorder()
	AntworteFehler(rec, http.StatusBadRequest,
		errors.New("ERROR: duplicate key value violates unique constraint \"foo_key\""))

	meldung := leseFehlermeldung(t, rec.Body.String())
	if strings.Contains(meldung, "unique constraint") || strings.Contains(meldung, "foo_key") {
		t.Errorf("DB-Detail im 4xx-Body geleakt: %q", meldung)
	}
}

// Beide Wege einer Fehlerantwort schreiben für denselben Fehler dieselbe Form ({"error": ...})
// mit derselben Meldung, damit sich die Oberfläche auf eine verlassen kann.
func TestMitFehlerantwortUndAntworteFehler_GleicheForm(t *testing.T) {
	konflikt := Conflict("Konflikt X", errors.New("intern: darf nicht sichtbar sein"))
	tuer := MitFehlerantwort(func(_ http.ResponseWriter, _ *http.Request) error {
		return konflikt
	})
	recA := httptest.NewRecorder()
	tuer(recA, httptest.NewRequest(http.MethodGet, "/x", nil))

	recB := httptest.NewRecorder()
	AntworteFehler(recB, http.StatusConflict, konflikt)

	if recA.Code != recB.Code {
		t.Errorf("Status weicht ab: MitFehlerantwort=%d, AntworteFehler=%d", recA.Code, recB.Code)
	}
	meldungA := leseFehlermeldung(t, recA.Body.String())
	meldungB := leseFehlermeldung(t, recB.Body.String())
	if meldungA != "Konflikt X" || meldungB != "Konflikt X" {
		t.Errorf("Meldungen: MitFehlerantwort=%q, AntworteFehler=%q", meldungA, meldungB)
	}
	// Die Ursache steht auf keinem der beiden Wege im Rumpf.
	for weg, rumpf := range map[string]string{"MitFehlerantwort": recA.Body.String(), "AntworteFehler": recB.Body.String()} {
		if strings.Contains(rumpf, "darf nicht sichtbar sein") {
			t.Errorf("%s leakt internen Fehler: %q", weg, rumpf)
		}
	}
}

// Hat der Browser die Anfrage abgebrochen, etwa durch einen Seitenwechsel, ist das kein Fehler
// des Servers: Die Antwort ist 499, ob der Fehler context.Canceled einwickelt oder nur der
// Kontext der Anfrage abgebrochen ist (nächster Test).
func TestAntworteFehler_AbbruchIstKein500(t *testing.T) {
	rec := httptest.NewRecorder()
	AntworteFehler(rec, http.StatusInternalServerError, fmt.Errorf("query: %w", context.Canceled))
	if rec.Code != statusAbgebrochen {
		t.Fatalf("Status = %d, want 499", rec.Code)
	}
	if meldung := leseFehlermeldung(t, rec.Body.String()); !strings.Contains(meldung, "abgebrochen") {
		t.Errorf("Meldung = %q, want Hinweis auf Abbruch", meldung)
	}
}

func TestMitFehlerantwort_AbgebrochenerKontextIstKein500(t *testing.T) {
	ctx, abbrechen := context.WithCancel(context.Background())
	abbrechen()
	req := httptest.NewRequest(http.MethodGet, "/api/x", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	// Der Fehler der Tür wickelt context.Canceled nicht ein; der Kontext allein reicht.
	MitFehlerantwort(func(http.ResponseWriter, *http.Request) error { return errors.New("scan: broken pipe") })(rec, req)
	if rec.Code != statusAbgebrochen {
		t.Fatalf("Status = %d, want 499", rec.Code)
	}
}

// Gegenprobe: Ein echter Fehler bleibt ein 500.
func TestMitFehlerantwort_EchterFehlerBleibt500(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	rec := httptest.NewRecorder()
	MitFehlerantwort(func(http.ResponseWriter, *http.Request) error { return errors.New("kaputt") })(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("Status = %d, want 500", rec.Code)
	}
}

// Die Oberfläche liest den mitgegebenen Satz, die eingewickelte Ursache bleibt im Log: Der
// Filter kennt SQL und Bedingungen, aber keine Meldung der Verbindung wie „connection reset by
// peer".
func TestAntworteFehlerMitMeldung_UrsacheBleibtImLog(t *testing.T) {
	rec := httptest.NewRecorder()
	gestoert := errors.New("sitzung konnte nicht geprüft werden, bitte erneut versuchen")
	AntworteFehlerMitMeldung(rec, http.StatusServiceUnavailable, gestoert.Error(),
		fmt.Errorf("%w: sperrliste: %v", gestoert, errors.New("connection reset by peer")))

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Status: erwartet 503, war %d", rec.Code)
	}
	meldung := leseFehlermeldung(t, rec.Body.String())
	if meldung != gestoert.Error() {
		t.Errorf("Meldung %q, erwartet genau %q", meldung, gestoert.Error())
	}
	for _, leck := range []string{"sperrliste", "connection reset by peer"} {
		if strings.Contains(meldung, leck) {
			t.Errorf("Ursache im Client-Body: %q steht in %q", leck, meldung)
		}
	}
}

// Ohne Meldung antwortet die Funktion wie AntworteFehler; sonst hätten die beiden Wege zwei
// Regeln.
func TestAntworteFehlerMitMeldung_OhneMeldungWieAntworteFehler(t *testing.T) {
	rec := httptest.NewRecorder()
	AntworteFehlerMitMeldung(rec, http.StatusConflict, "", errors.New("benutzer hat noch aktive ausleihen"))

	if meldung := leseFehlermeldung(t, rec.Body.String()); meldung != "benutzer hat noch aktive ausleihen" {
		t.Errorf("Meldung verfälscht: %q", meldung)
	}
}

// Ein 500 bleibt neutral, auch wenn der Rufer eine Meldung mitgibt.
func TestAntworteFehlerMitMeldung_500BleibtNeutral(t *testing.T) {
	rec := httptest.NewRecorder()
	AntworteFehlerMitMeldung(rec, http.StatusInternalServerError, "Tabelle benutzer fehlt",
		errors.New("pgx: relation \"benutzer\" does not exist"))

	meldung := leseFehlermeldung(t, rec.Body.String())
	for _, leck := range []string{"Tabelle", "benutzer", "pgx"} {
		if strings.Contains(meldung, leck) {
			t.Errorf("500 gibt Detail heraus (%q): %q", leck, meldung)
		}
	}
}

// Der Filter liest den Text, der hinausgeht: Eine Meldung erlaubt einen Satz, aber keinen Text
// einer Abfrage, den jemand aus Versehen durchreicht.
func TestAntworteFehlerMitMeldung_DatenbanktextWirdTrotzdemGefiltert(t *testing.T) {
	rec := httptest.NewRecorder()
	AntworteFehlerMitMeldung(rec, http.StatusConflict,
		"select * from benutzer schlug fehl", errors.New("konflikt beim speichern"))

	meldung := leseFehlermeldung(t, rec.Body.String())
	if strings.Contains(strings.ToLower(meldung), "select") {
		t.Errorf("SQL-Wortlaut in der Meldung durchgereicht: %q", meldung)
	}
}

// Eine Meldung geht ohne Umweg über einen Fehlerwert hinaus. Antwort und Eintrag im Log sind
// dieselben wie mit errors.New(Meldung) als Ursache, für jeden Text und jeden Status; ein 500
// antwortet auf beiden Wegen mit der neutralen Meldung.
func TestAntworteFehlerMitMeldung_OhneUrsacheWieMitFehlerwert(t *testing.T) {
	const satz = "bitte einen grund für die korrektur angeben"
	for _, status := range []int{
		http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
		http.StatusConflict, http.StatusRequestEntityTooLarge, http.StatusInternalServerError,
		http.StatusBadGateway, http.StatusServiceUnavailable,
	} {
		mitFehlerwert := httptest.NewRecorder()
		AntworteFehler(mitFehlerwert, status, errors.New(satz))
		ohneUrsache := httptest.NewRecorder()
		AntworteFehlerMitMeldung(ohneUrsache, status, satz, nil)

		if ohneUrsache.Code != mitFehlerwert.Code || ohneUrsache.Body.String() != mitFehlerwert.Body.String() {
			t.Errorf("AntworteFehlerMitMeldung(%d) = %d %q, erwartet %d %q", status,
				ohneUrsache.Code, ohneUrsache.Body.String(), mitFehlerwert.Code, mitFehlerwert.Body.String())
		}
		zeileMitFehlerwert, ok := logZeile(status, "", errors.New(satz))
		if !ok {
			t.Fatalf("logZeile(%d) mit Ursache: kein Eintrag", status)
		}
		if zeile, ok := logZeile(status, satz, nil); !ok || zeile != zeileMitFehlerwert {
			t.Errorf("logZeile(%d) ohne Ursache = %q, %v, erwartet %q", status, zeile, ok, zeileMitFehlerwert)
		}
	}
	if zeile, ok := logZeile(http.StatusBadRequest, "", nil); ok {
		t.Errorf("logZeile ohne Meldung und Ursache = %q, erwartet keinen Eintrag", zeile)
	}
}

// Eine Prüfung neben der Tür liefert ihren Satz als Fehler mit Meldung. AntworteFehler schickt
// diese Meldung hinaus wie MitFehlerantwort, auch wenn der Fehler eine Ursache trägt oder
// eingewickelt ist; die Ursache steht nur im Log.
func TestAntworteFehler_FehlerMitMeldungSchicktDieMeldung(t *testing.T) {
	ursache := errors.New("leser.klasse leer")
	for name, err := range map[string]error{
		"ohne Ursache": BadRequest("Klasse fehlt.", nil),
		"mit Ursache":  BadRequest("Klasse fehlt.", ursache),
		"eingewickelt": fmt.Errorf("leser anlegen: %w", BadRequest("Klasse fehlt.", ursache)),
	} {
		rec := httptest.NewRecorder()
		AntworteFehler(rec, http.StatusBadRequest, err)
		if meldung := leseFehlermeldung(t, rec.Body.String()); meldung != "Klasse fehlt." {
			t.Errorf("%s: AntworteFehler = %q, erwartet %q", name, meldung, "Klasse fehlt.")
		}
		zeile, ok := logZeile(http.StatusBadRequest, "", err)
		if !ok || !strings.Contains(zeile, "Klasse fehlt.") {
			t.Errorf("%s: logZeile = %q, %v, erwartet einen Eintrag mit der Meldung", name, zeile, ok)
		}
	}
}
