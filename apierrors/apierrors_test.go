package apierrors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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

// Der Text eines Fehlers ohne Meldung geht nicht hinaus, unter 500 so wenig wie über 500: Er ist
// für das Log geschrieben. Der Status bleibt, die Oberfläche liest meldungOhneSatz.
func TestAntworteFehler_FehlerOhneMeldungGehtNichtHinaus(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusConflict, http.StatusBadGateway, http.StatusServiceUnavailable} {
		rec := httptest.NewRecorder()
		AntworteFehler(rec, status, errors.New("benutzer hat noch aktive ausleihen"))

		if rec.Code != status {
			t.Errorf("Status = %d, erwartet %d", rec.Code, status)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("%d: Content-Type = %q, erwartet application/json", status, ct)
		}
		if meldung := leseFehlermeldung(t, rec.Body.String()); meldung != meldungOhneSatz {
			t.Errorf("%d: Meldung %q, erwartet %q", status, meldung, meldungOhneSatz)
		}
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

// Ein 500 sagt auf beiden Wegen denselben Satz: über AntworteFehler, mit und ohne Ursache, und als
// Vorgabe von Internal hinter MitFehlerantwort.
func TestNeutraleMeldung_EinSatzAufAllenWegen(t *testing.T) {
	mitUrsache := httptest.NewRecorder()
	AntworteFehler(mitUrsache, http.StatusInternalServerError, errors.New("cover: datei nicht geschrieben"))
	ohneUrsache := httptest.NewRecorder()
	AntworteFehlerMitMeldung(ohneUrsache, http.StatusInternalServerError, "Tabelle fehlt", nil)
	ueberTuer := httptest.NewRecorder()
	MitFehlerantwort(func(http.ResponseWriter, *http.Request) error { return Internal("", errors.New("kaputt")) })(
		ueberTuer, httptest.NewRequest(http.MethodGet, "/api/x", nil))

	for weg, rec := range map[string]*httptest.ResponseRecorder{
		"AntworteFehler mit Ursache": mitUrsache, "ohne Ursache": ohneUrsache, "Internal": ueberTuer,
	} {
		if meldung := leseFehlermeldung(t, rec.Body.String()); meldung != "Ein interner Serverfehler ist aufgetreten" {
			t.Errorf("%s: Meldung %q, erwartet „Ein interner Serverfehler ist aufgetreten“", weg, meldung)
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
		t.Fatalf("Status = %d, erwartet 499", rec.Code)
	}
	if meldung := leseFehlermeldung(t, rec.Body.String()); !strings.Contains(meldung, "abgebrochen") {
		t.Errorf("Meldung = %q, erwartet Hinweis auf Abbruch", meldung)
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
		t.Fatalf("Status = %d, erwartet 499", rec.Code)
	}
}

// Gegenprobe: Ein echter Fehler bleibt ein 500.
func TestMitFehlerantwort_EchterFehlerBleibt500(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	rec := httptest.NewRecorder()
	MitFehlerantwort(func(http.ResponseWriter, *http.Request) error { return errors.New("kaputt") })(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("Status = %d, erwartet 500", rec.Code)
	}
}

// Die Oberfläche liest den mitgegebenen Satz, die eingewickelte Ursache bleibt im Log: Der
// Filter kennt SQL und Bedingungen, aber keine Meldung der Verbindung wie „connection reset by
// peer".
func TestAntworteFehlerMitMeldung_UrsacheBleibtImLog(t *testing.T) {
	rec := httptest.NewRecorder()
	gestoert := errors.New("sitzung konnte nicht geprüft werden, bitte erneut versuchen")
	AntworteFehlerMitMeldung(rec, http.StatusServiceUnavailable, gestoert.Error(),
		fmt.Errorf("%w: sperrliste: %w", gestoert, errors.New("connection reset by peer")))

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Status = %d, erwartet 503", rec.Code)
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
	ursache := errors.New("benutzer hat noch aktive ausleihen")
	ohneMeldung := httptest.NewRecorder()
	AntworteFehlerMitMeldung(ohneMeldung, http.StatusConflict, "", ursache)
	ueberAntworteFehler := httptest.NewRecorder()
	AntworteFehler(ueberAntworteFehler, http.StatusConflict, ursache)

	if ohneMeldung.Body.String() != ueberAntworteFehler.Body.String() {
		t.Errorf("AntworteFehlerMitMeldung ohne Meldung = %q, AntworteFehler = %q",
			ohneMeldung.Body.String(), ueberAntworteFehler.Body.String())
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

// Eine Meldung geht ohne Umweg über einen Fehlerwert hinaus, unter und über 500; ein 500
// antwortet mit der neutralen Meldung. Der Eintrag im Log nennt die Meldung, wie er sonst die
// Ursache nennt.
func TestAntworteFehlerMitMeldung_OhneUrsacheSchicktDieMeldung(t *testing.T) {
	const satz = "Bitte einen Grund für die Korrektur angeben"
	const text = "bitte einen grund für die korrektur angeben"
	for _, status := range []int{
		http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
		http.StatusConflict, http.StatusRequestEntityTooLarge, http.StatusInternalServerError,
		http.StatusBadGateway, http.StatusServiceUnavailable,
	} {
		rec := httptest.NewRecorder()
		AntworteFehlerMitMeldung(rec, status, satz, nil)

		erwartet := satz
		if status == http.StatusInternalServerError {
			erwartet = meldungInternerFehler
		}
		if rec.Code != status {
			t.Errorf("AntworteFehlerMitMeldung(%d): Status %d", status, rec.Code)
		}
		if meldung := leseFehlermeldung(t, rec.Body.String()); meldung != erwartet {
			t.Errorf("AntworteFehlerMitMeldung(%d) = %q, erwartet %q", status, meldung, erwartet)
		}
		zeileMitFehlerwert, ok := logZeile("", errors.New(text))
		if !ok {
			t.Fatalf("logZeile(%d) mit Ursache: kein Eintrag", status)
		}
		if zeile, ok := logZeile(text, nil); !ok || zeile != zeileMitFehlerwert {
			t.Errorf("logZeile(%d) ohne Ursache = %q, %v, erwartet %q", status, zeile, ok, zeileMitFehlerwert)
		}
	}
	if zeile, ok := logZeile("", nil); ok {
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
		zeile, ok := logZeile("", err)
		if !ok || !strings.Contains(zeile, "Klasse fehlt.") {
			t.Errorf("%s: logZeile = %q, %v, erwartet einen Eintrag mit der Meldung", name, zeile, ok)
		}
	}
}

// Eine Fehlerantwort steht nur mit Ursache und 5xx als ERROR im Log. Ohne Ursache zählt die Zeile
// zum Status aus logMiddleware den Fehler, die einen Abbruch des Browsers nicht mitzählt.
func TestStufeDerFehlerantwort(t *testing.T) {
	faelle := []struct {
		status  int
		ursache error
		will    slog.Level
	}{
		{http.StatusInternalServerError, errors.New("kaputt"), slog.LevelError},
		{http.StatusServiceUnavailable, errors.New("kaputt"), slog.LevelError},
		{http.StatusInternalServerError, nil, slog.LevelInfo},
		{http.StatusBadRequest, errors.New("unlesbar"), slog.LevelInfo},
	}
	for _, f := range faelle {
		if got := stufeDerFehlerantwort(f.status, f.ursache); got != f.will {
			t.Errorf("stufeDerFehlerantwort(%d, %v) = %v, erwartet %v", f.status, f.ursache, got, f.will)
		}
	}
}
