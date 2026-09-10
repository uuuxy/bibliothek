package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Die Pflichtangaben der Art.-15-Auskunft nennen die EINGESTELLTEN Fristen — geprüft am
// LIVE-PFAD, nicht an der Formulierungsfunktion.
//
// Warum am Live-Pfad: Die Funktion dsgvoVerarbeitungsangaben hat einen Unit-Test, und der
// war zweimal grün, während die Auskunft trotzdem eine falsche Frist nannte — beim ersten
// Mal, weil im Text eine feste Zahl stand (360 Tage, 02.09.2026), beim zweiten Mal, weil
// die Aufbewahrung der Protokolle inzwischen eine Einstellung war und der Leser sie nicht
// mitnahm (10.09.2026). Beides sind Pflichtangaben an die betroffene Person; falsch ist
// hier schlimmer als fehlend, weil eine falsche Frist wie eine geprüfte aussieht.
//
// Der Test setzt jede Frist auf einen Wert, der von der Vorgabe abweicht, und verlangt
// GENAU diesen Wert im Text — eine vergessene Einstellung fällt damit auf, statt still
// die Werksvorgabe zu behaupten.
func TestDsgvoAuskunft_FristenKommenAusDenEinstellungen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// Bewusst krumme Werte: 24 (Vorgabe Protokolle), 90/730 (Vorgabe Lesehistorie) und
	// 90 (Vorgabe Karenz) würden auch ohne Leser im Text stehen.
	einstellungen := map[string]string{
		repository.AuditAufbewahrungSchluessel: "7",
		"lesehistorie_tage":                    "111",
		"lesehistorie_lernmittel_tage":         "222",
		"abgaenger_karenz_tage":                "33",
	}
	for schluessel, wert := range einstellungen {
		if _, err := pool.Exec(ctx, `
			INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)
			ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, schluessel, wert); err != nil {
			t.Fatalf("Einstellung %s setzen: %v", schluessel, err)
		}
	}
	t.Cleanup(func() {
		for schluessel := range einstellungen {
			if _, err := pool.Exec(context.Background(),
				`DELETE FROM system_einstellungen WHERE schluessel = $1`, schluessel); err != nil {
				t.Logf("Aufräumen %s: %v", schluessel, err)
			}
		}
	})

	sid := seedSchueler(t, pool, "S-FRISTEN-PROBE", "Fristenkind", "5F1")

	srv := &Server{DB: &db.Database{Pool: pool}}
	req := httptest.NewRequest(http.MethodGet, "/api/schueler/"+sid+"/dsgvo-auskunft", nil)
	req.SetPathValue("id", sid)
	rec := httptest.NewRecorder()
	srv.DsgvoAuskunftHandler()(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var antwort DsgvoAuskunftResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	dauer := antwort.Verarbeitungsangaben.Speicherdauer

	for _, erwartet := range []string{"7 Monate", "111 Tage", "222 Tage", "Karenzzeit von 33 Tagen"} {
		if !strings.Contains(dauer, erwartet) {
			t.Errorf("Speicherdauer nennt %q nicht — die Auskunft ignoriert die Einstellung:\n%s", erwartet, dauer)
		}
	}
	// Die Werksvorgaben dürfen NICHT im Text stehen, wenn etwas anderes eingestellt ist.
	for _, verboten := range []string{"24 Monate", "360"} {
		if strings.Contains(dauer, verboten) {
			t.Errorf("Speicherdauer behauptet die Werksvorgabe %q:\n%s", verboten, dauer)
		}
	}
}

// Gegenprobe zur Untergrenze: Eine unplausible 0 in den Einstellungen darf der
// betroffenen Person nicht als „0 Monate" gemeldet werden — dieselbe Untergrenze wie beim
// Löschjob (repository.AufbewahrungMonateOderStandard).
func TestDsgvoAuskunft_ProtokollfristUnterUntergrenzeMeldetVorgabe(t *testing.T) {
	if got := repository.AufbewahrungMonateOderStandard(nil); got != repository.StandardAuditAufbewahrungMonate {
		t.Errorf("ohne Einstellung: %d, want %d", got, repository.StandardAuditAufbewahrungMonate)
	}
	null := 0
	if got := repository.AufbewahrungMonateOderStandard(&null); got != repository.StandardAuditAufbewahrungMonate {
		t.Errorf("0 Monate: %d, want %d (Untergrenze)", got, repository.StandardAuditAufbewahrungMonate)
	}
}
