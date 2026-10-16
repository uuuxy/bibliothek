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

func TestProtokolliereVerwaltung_Erfolgreich(t *testing.T) {
	pool := pgtest.Pool(t)
	const lieferantName = "Test-Lieferant GmbH"

	raeume := func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM audit_logs WHERE aktion = $1 AND details->>'name' = $2`, AktionLieferantAngelegt, lieferantName)
	}
	raeume()
	t.Cleanup(raeume)

	adminID := testdaten.LegeAdminKontoAn(t, pool, "verwaltung-protokoll@example.org", "Paula", "Protokoll")

	s := &Basis{DB: &db.Verbindung{Pool: pool}}
	ctx := context.WithValue(t.Context(), auth.ClaimsSchluessel, &auth.Claims{KontoID: adminID, Rolle: auth.RolleAdmin})

	detailsInput := map[string]any{
		"name": lieferantName,
		"id":   "12345",
	}

	s.ProtokolliereVerwaltung(ctx, AktionLieferantAngelegt, detailsInput)

	var bearbeiter string
	var roh []byte
	if err := pool.QueryRow(t.Context(), `
		SELECT coalesce(admin_id::text, ''), details FROM audit_logs
		WHERE aktion = $1 AND details->>'name' = $2`, AktionLieferantAngelegt, lieferantName).Scan(&bearbeiter, &roh); err != nil {
		t.Fatalf("Eintrag lesen fehlgeschlagen: %v", err)
	}
	if bearbeiter != adminID {
		t.Errorf("Bearbeiter %q, erwartet %q", bearbeiter, adminID)
	}

	var details map[string]any
	if err := json.Unmarshal(roh, &details); err != nil {
		t.Fatalf("Details unlesbar: %v", err)
	}
	erwartet := map[string]any{
		"name": lieferantName,
		"id":   "12345",
	}
	if !cmp.Equal(details, erwartet) {
		t.Errorf("Details %v, erwartet %v", details, erwartet)
	}
}

func TestProtokolliereVerwaltung_OhneClaims(t *testing.T) {
	pool := pgtest.Pool(t)

	s := &Basis{DB: &db.Verbindung{Pool: pool}}
	ctx := t.Context() // Kein Claim

	s.ProtokolliereVerwaltung(ctx, AktionLieferantAngelegt, map[string]any{"name": "Niemand"})

	var count int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_logs WHERE aktion = $1 AND details->>'name' = 'Niemand'`, AktionLieferantAngelegt).Scan(&count); err != nil {
		t.Fatalf("Zählen fehlgeschlagen: %v", err)
	}
	if count > 0 {
		t.Errorf("Eintrag geschrieben, obwohl keine Claims im Kontext")
	}
}

func TestProtokolliereVerwaltung_OhneDatenbank(t *testing.T) {
	s := &Basis{DB: nil}
	ctx := context.WithValue(t.Context(), auth.ClaimsSchluessel, &auth.Claims{KontoID: "123", Rolle: auth.RolleAdmin})

	// Sollte nicht abbrechen
	s.ProtokolliereVerwaltung(ctx, AktionLieferantAngelegt, map[string]any{"name": "Niemand"})
}

func TestProtokolliereVerwaltung_DatenbankOhnePool(t *testing.T) {
	s := &Basis{DB: &db.Verbindung{Pool: nil}}
	ctx := context.WithValue(t.Context(), auth.ClaimsSchluessel, &auth.Claims{KontoID: "123", Rolle: auth.RolleAdmin})

	// Sollte nicht abbrechen
	s.ProtokolliereVerwaltung(ctx, AktionLieferantAngelegt, map[string]any{"name": "Niemand"})
}
