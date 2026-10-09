package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/auskunft"
	"bibliothek/repository"
)

// Die Pflichtangaben der Auskunft nennen die eingestellten Fristen, geprüft über die Tür und
// nicht an der Funktion, die den Text formuliert: Deren Test bleibt grün, wenn im Text eine
// feste Zahl steht oder die Tür eine Einstellung nicht mitnimmt. Eine falsche Frist ist hier
// schlimmer als eine fehlende, weil sie wie eine geprüfte aussieht.
//
// Der Test setzt jede Frist auf einen Wert, der von der Vorgabe abweicht, und verlangt ihn an
// seiner Stelle im Satz: Eine vergessene Einstellung behauptete sonst still die Werksvorgabe,
// und zwei vertauschte Fristen stünden beide im Text. Die Frist erledigter Anliegen nennt nur
// die Auskunft eines Kollegen, deshalb steht eine Kollegin daneben.
func TestDsgvoAuskunft_FristenKommenAusDenEinstellungen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// Krumme Werte: Die Vorgaben (1 und 730 Tage Lesehistorie, 90 Tage Karenz, 24 Monate
	// Protokolle, 365 Tage Anliegen) stünden auch ohne gelesene Einstellung im Text.
	einstellungen := map[string]string{
		repository.AuditAufbewahrungSchluessel: "7",
		"lesehistorie_tage":                    "111",
		"lesehistorie_lernmittel_tage":         "222",
		"abgaenger_karenz_tage":                "33",
		"anliegen_tage":                        "44",
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
	var kollege string
	if err := pool.QueryRow(ctx, `INSERT INTO leser (vorname, nachname, art)
		VALUES ('Frieda', 'Fristenkollegin', 'lehrkraft') RETURNING id`).Scan(&kollege); err != nil {
		t.Fatalf("Kollegin anlegen: %v", err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	speicherdauer := func(leserID string) string {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/schueler/"+leserID+"/dsgvo-auskunft", nil)
		req.SetPathValue("id", leserID)
		rec := httptest.NewRecorder()
		srv.DsgvoAuskunftHandler()(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		var antwort auskunft.DsgvoAuskunftResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Antwort lesen: %v", err)
		}
		return antwort.Verarbeitungsangaben.Speicherdauer
	}

	for _, fall := range []struct {
		name, leserID string
		erwartet      []string
	}{
		{"Schüler", sid, []string{
			"Schülerbücherei 111 Tage nach Rückgabe", "Lernmittel 222 Tage nach Rückgabe",
			"Karenzzeit von 33 Tagen", "Protokolle 7 Monate"}},
		{"Kollegin", kollege, []string{
			"Schülerbücherei 111 Tage nach Rückgabe", "Lernmittel 222 Tage nach Rückgabe",
			"Reservierungen: 44 Tage nach der Erledigung", "Protokolle 7 Monate"}},
	} {
		dauer := speicherdauer(fall.leserID)
		for _, erwartet := range fall.erwartet {
			if !strings.Contains(dauer, erwartet) {
				t.Errorf("%s: Speicherdauer nennt %q nicht — die Auskunft ignoriert die Einstellung:\n%s", fall.name, erwartet, dauer)
			}
		}
		// Die Werksvorgaben stehen nicht im Text, wenn etwas anderes eingestellt ist.
		for _, verboten := range []string{"24 Monate", "360", "365 Tage"} {
			if strings.Contains(dauer, verboten) {
				t.Errorf("%s: Speicherdauer behauptet die Werksvorgabe %q:\n%s", fall.name, verboten, dauer)
			}
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
