package api

// Die Uhr des Theken-Rechners an der Nachbuch-Tür (Rasterdurchgang 15.09.2026, OFFEN.md 5.15).
//
// Der Scan-Zeitpunkt kommt vom Rechner, der Bewegungsstempel vom Server. Bis hierher wurde der
// Scan nur nach oben auf die Serverzeit gekappt: Ging die Uhr des Rechners vor, lag eine offline
// gebuchte Rückgabe nach der jüngeren Online-Ausleihe eines anderen Kindes und beendete sie. Jetzt
// schickt der Rechner mit jeder Portion seine Uhrzeit beim Versand; der Server misst daran den
// Versatz und rechnet die Scan-Zeitpunkte der Portion auf seine Uhr um.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type nachbuchUhrAntwort struct {
	Ergebnisse         []NachbuchenErgebnis `json:"ergebnisse"`
	UhrVersatzSekunden *int                 `json:"uhr_versatz_sekunden"`
}

// nachbuchenMitUhr schickt eine Portion mit der Sendezeit des Rechners; gesendet nil lässt das Feld weg.
func (w *nbTuer) nachbuchenMitUhr(t *testing.T, gesendet *time.Time, eintraege ...map[string]any) (int, nachbuchUhrAntwort) {
	t.Helper()
	rumpf := map[string]any{"eintraege": eintraege}
	if gesendet != nil {
		rumpf["gesendet_am"] = *gesendet
	}
	body, err := json.Marshal(rumpf)
	if err != nil {
		t.Fatalf("Rumpf: %v", err)
	}
	rec := httptest.NewRecorder()
	w.srv.NachbuchenHandler(w.nachbuch)(rec, w.sitzung(httptest.NewRequest(http.MethodPost, "/api/action/nachbuchen", strings.NewReader(string(body)))))
	var antwort nachbuchUhrAntwort
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Antwort: %v", err)
		}
	}
	return rec.Code, antwort
}

func (w *nbTuer) ausleiheOffen(t *testing.T, schueler string) bool {
	t.Helper()
	var n int
	if err := w.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM ausleihen WHERE exemplar_id = $1 AND schueler_id = $2 AND rueckgabe_am IS NULL`, w.exemplarID, schueler).Scan(&n); err != nil {
		t.Fatalf("Ausleihe lesen: %v", err)
	}
	return n == 1
}

// Theke 1 geht 15 Minuten vor. Anna gibt dort offline zurück; danach leiht Theke 2 dasselbe Buch
// online an Ben. Beim Nachbuchen muss die Rückgabe VOR Bens Ausleihe liegen: veraltet, Bens
// Ausleihe bleibt.
func TestNachbuchen_Uhr_VorgehendeUhrBeendetKeineJuengereAusleihe(t *testing.T) {
	w := nbTuerAufbau(t)
	if code, r := w.onlineScan(t, uuid.NewString(), w.anna); code != http.StatusOK || r.Type != "ausleihe" {
		t.Fatalf("Online-Ausleihe an Anna: %d %q", code, r.Type)
	}
	vorgehen := 15 * time.Minute
	echteRueckgabe := time.Now()
	time.Sleep(200 * time.Millisecond)

	// Theke 2: Bens Sitzung scannt das Buch, das noch auf Anna steht — Fremdrückgabe, dann Ausleihe.
	if code, r := w.onlineScan(t, uuid.NewString(), w.ben); code != http.StatusOK || !r.Fremdrueckgabe {
		t.Fatalf("Fremdrückgabe an Theke 2: %d fremd=%v", code, r.Fremdrueckgabe)
	}
	if code, r := w.onlineScan(t, uuid.NewString(), w.ben); code != http.StatusOK || r.Type != "ausleihe" {
		t.Fatalf("Online-Ausleihe an Ben: %d %q", code, r.Type)
	}

	gesendet := time.Now().Add(vorgehen)
	code, antwort := w.nachbuchenMitUhr(t, &gesendet, w.eintrag(uuid.NewString(), "rueckgabe", nil, echteRueckgabe.Add(vorgehen)))
	if code != http.StatusOK {
		t.Fatalf("Nachbuchen: Status %d", code)
	}
	if antwort.Ergebnisse[0].Ergebnis != "veraltet" {
		t.Errorf("Ergebnis %q (%s), erwartet veraltet", antwort.Ergebnisse[0].Ergebnis, antwort.Ergebnisse[0].Grund)
	}
	if !w.ausleiheOffen(t, w.ben) {
		t.Error("Ben hat das Buch in der Hand, seine Ausleihe ist beendet")
	}
	if antwort.UhrVersatzSekunden == nil || *antwort.UhrVersatzSekunden > -895 || *antwort.UhrVersatzSekunden < -905 {
		t.Errorf("uhr_versatz_sekunden %v, erwartet etwa -900", antwort.UhrVersatzSekunden)
	}
}

// Theke 1 geht 15 Minuten nach. Anna gibt dort offline zurück, lange nach ihrer Online-Ausleihe.
// Ohne Umrechnung läge die Rückgabe vor der Ausleihe und würde als veraltet abgewiesen — Anna
// behielte ein Buch, das im Regal steht.
func TestNachbuchen_Uhr_NachgehendeUhrBuchtDieRueckgabe(t *testing.T) {
	w := nbTuerAufbau(t)
	if code, r := w.onlineScan(t, uuid.NewString(), w.anna); code != http.StatusOK || r.Type != "ausleihe" {
		t.Fatalf("Online-Ausleihe an Anna: %d %q", code, r.Type)
	}
	nachgehen := 15 * time.Minute
	time.Sleep(200 * time.Millisecond)
	echteRueckgabe := time.Now()

	gesendet := time.Now().Add(-nachgehen)
	code, antwort := w.nachbuchenMitUhr(t, &gesendet, w.eintrag(uuid.NewString(), "rueckgabe", nil, echteRueckgabe.Add(-nachgehen)))
	if code != http.StatusOK {
		t.Fatalf("Nachbuchen: Status %d", code)
	}
	if antwort.Ergebnisse[0].Ergebnis != "zurueckgegeben" {
		t.Errorf("Ergebnis %q (%s), erwartet zurueckgegeben", antwort.Ergebnisse[0].Ergebnis, antwort.Ergebnisse[0].Grund)
	}
	if w.ausleiheOffen(t, w.anna) {
		t.Error("das Buch steht im Regal, Annas Ausleihe ist noch offen")
	}
	if antwort.UhrVersatzSekunden == nil || *antwort.UhrVersatzSekunden < 895 || *antwort.UhrVersatzSekunden > 905 {
		t.Errorf("uhr_versatz_sekunden %v, erwartet etwa 900", antwort.UhrVersatzSekunden)
	}
}

// Ohne Sendezeit lässt sich der Versatz nicht messen: Die Portion wird abgewiesen, nicht mit
// ungeprüfter Uhr gebucht.
func TestNachbuchen_Uhr_OhneSendezeitAbgewiesen(t *testing.T) {
	w := nbTuerAufbau(t)
	code, _ := w.nachbuchenMitUhr(t, nil, w.eintrag(uuid.NewString(), "ausleihe", &w.anna, time.Now()))
	if code != http.StatusBadRequest {
		t.Errorf("Status %d, erwartet 400", code)
	}
	if n, _ := w.offen(t); n != 0 {
		t.Errorf("ohne Sendezeit gebucht: %d offen", n)
	}
}
