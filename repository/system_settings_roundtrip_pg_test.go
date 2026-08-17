package repository

import (
	"context"
	"testing"
)

// Der Roundtrip Speichern→Laden am echten Postgres. Die pgxmock-Tests daneben
// sehen nur, WELCHE Keys ins Upsert gehen — ob das Laden den Wert auch wieder
// EINFÜLLT, entscheidet der case-Zweig in applyEinstellung. Ein Struct-Feld ohne
// diesen Zweig speichert scheinbar erfolgreich und liest für immer leer
// (Bugklasse „Feld ohne Nachfüller", Audit-Raster: blinde Hälfte). Anlass ist
// alarm_empfaenger (17.08.2026): Der Alarm-Verteiler MUSS aus der DB zurückkommen,
// sonst fällt der Kritisch-Alarm still auf alle aktiven Admins zurück.
func TestSettingsRoundtrip_AlarmEmpfaenger(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewSystemSettingsRepository(pool)

	verteiler := "pflasch@philipp-reis-schule.de, it@schule.example"
	if err := repo.SaveSettings(ctx, &SystemEinstellungen{
		LmfStichtag:     "07-31",
		AlarmEmpfaenger: &verteiler,
	}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	geladen, err := repo.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if geladen.AlarmEmpfaenger == nil || *geladen.AlarmEmpfaenger != verteiler {
		t.Fatalf("alarm_empfaenger kam nicht zurück: %v (gespeichert war %q)",
			geladen.AlarmEmpfaenger, verteiler)
	}

	// Leerer String ist ein GÜLTIGER Wert (= Rückfall auf alle Admins) und muss
	// einen vorhandenen Verteiler überschreiben können — sonst wird man eine
	// einmal gesetzte Liste nie wieder los.
	leer := ""
	if err := repo.SaveSettings(ctx, &SystemEinstellungen{
		LmfStichtag:     "07-31",
		AlarmEmpfaenger: &leer,
	}); err != nil {
		t.Fatalf("SaveSettings (leer): %v", err)
	}
	geladen, err = repo.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings (leer): %v", err)
	}
	if geladen.AlarmEmpfaenger != nil && *geladen.AlarmEmpfaenger != "" {
		t.Fatalf("leerer Verteiler überschreibt nicht: %q", *geladen.AlarmEmpfaenger)
	}
}
