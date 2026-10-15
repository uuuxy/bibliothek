package tuer

import (
	"context"
	"encoding/json"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"

	"github.com/google/go-cmp/cmp"
)

// Der Eintrag zur Konto-Anlage trägt jede Angabe unter ihrem Namen, mit der angemeldeten Person
// als Bearbeiter. An ziel_id findet die Auskunft den Eintrag, an Adresse und Namen die Tilgung;
// eine vertauschte Angabe bliebe dort unbemerkt stehen.
func TestProtokolliereKontoAnlage_JedeAngabeUnterIhremNamen(t *testing.T) {
	pool := pgtest.Pool(t)
	const zielID = "5f0c2a9e-7d41-4b6e-9c3a-2e8b1d4f6a70"
	raeume := func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM audit_logs WHERE aktion = 'USER_CREATE' AND details->>'ziel_id' = $1`, zielID)
	}
	raeume()
	t.Cleanup(raeume)
	adminID := testdaten.LegeAdminKontoAn(t, pool, "konto-anlage-protokoll@example.org", "Paula", "Protokoll")

	s := &Basis{DB: &db.Verbindung{Pool: pool}}
	ctx := context.WithValue(t.Context(), auth.ClaimsSchluessel, &auth.Claims{KontoID: adminID, Rolle: auth.RolleAdmin})
	s.ProtokolliereKontoAnlage(ctx, zielID, "neu@konto-anlage.example", "kollegium", "Vera", "Nachbar")

	var bearbeiter string
	var roh []byte
	if err := pool.QueryRow(t.Context(), `
		SELECT coalesce(admin_id::text, ''), details FROM audit_logs
		WHERE aktion = 'USER_CREATE' AND details->>'ziel_id' = $1`, zielID).Scan(&bearbeiter, &roh); err != nil {
		t.Fatalf("Eintrag der Anlage lesen: %v", err)
	}
	if bearbeiter != adminID {
		t.Errorf("Bearbeiter %q, erwartet die angemeldete Person %q", bearbeiter, adminID)
	}
	var details map[string]any
	if err := json.Unmarshal(roh, &details); err != nil {
		t.Fatalf("Details unlesbar (%s): %v", roh, err)
	}
	erwartet := map[string]any{
		"ziel_id": zielID, "email": "neu@konto-anlage.example", "rolle": "kollegium",
		"vorname": "Vera", "nachname": "Nachbar",
	}
	if !cmp.Equal(details, erwartet) {
		t.Errorf("Details %v, erwartet %v", details, erwartet)
	}
}
