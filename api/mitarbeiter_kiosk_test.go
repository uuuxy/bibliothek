package api

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/service"
	"bibliothek/repository"
)

// Die Theken-Antwort trägt von einer Lehrkraft nur Identität — kein Konto. Gemessen an
// der SERIALISIERTEN Antwort, nicht am Typ: Genau das JSON bekommt der Helfer.
func TestActionResponse_LehrkraftOhneKontodaten(t *testing.T) {
	antrag := time.Now()
	voll := &repository.User{
		ID: "u-1", BarcodeID: "L-0001", Vorname: "Frieda", Nachname: "Fachlehrerin",
		Rolle: "KOLLEGIUM", Email: "frieda.fachlehrerin@philipp-reis-schule.de", Aktiv: true,
		ErstelltAm: time.Now(), ZugangBeantragtAm: &antrag,
	}
	resp := mapOmniboxResultToActionResponse(&service.OmniboxResult{Type: "teacher", Teacher: voll, VorbesitzerUser: voll})
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, verboten := range []string{"email", "philipp-reis-schule.de", "rolle", "KOLLEGIUM", "aktiv", "erstellt_am", "zugang_beantragt_am"} {
		if strings.Contains(body, verboten) {
			t.Errorf("Theken-Antwort trägt Kontodaten (%q): %s", verboten, body)
		}
	}
	for _, pflicht := range []string{`"vorname":"Frieda"`, `"nachname":"Fachlehrerin"`, `"barcode_id":"L-0001"`, `"id":"u-1"`} {
		if !strings.Contains(body, pflicht) {
			t.Errorf("Theken-Antwort verliert die Identität (%s): %s", pflicht, body)
		}
	}
}
