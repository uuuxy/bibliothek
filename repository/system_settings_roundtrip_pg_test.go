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

// Die vier Datenschutz-/Sitzungsfelder: nil lässt den gespeicherten Wert stehen, ein
// gesetzter Zeiger schreibt — auch die 0, denn 0 heißt hier „aus" und ist ein Wert.
// Ohne diese Unterscheidung hätte das Speichern einer anderen Sektion die Befristung
// der Lesehistorie still abgeschaltet (Upsert-Blanking).
func TestSettingsRoundtrip_DatenschutzZeigerSemantik(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewSystemSettingsRepository(pool)

	geladen, err := repo.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if TageOderStandard(geladen.LesehistorieTage, -1) != StandardLesehistorieTage ||
		TageOderStandard(geladen.LesehistorieLernmittelTage, -1) != StandardLesehistorieLernmittelTage ||
		TageOderStandard(geladen.SperreMinuten, -1) != StandardSperreMinuten {
		t.Fatalf("Vorgaben fehlen: %+v", geladen)
	}

	null, dreissig := 0, 30
	if err := repo.SaveSettings(ctx, &SystemEinstellungen{LmfStichtag: "07-31", LesehistorieTage: &null, SperreMinuten: &dreissig}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	if geladen, err = repo.GetSettings(ctx); err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if TageOderStandard(geladen.LesehistorieTage, -1) != 0 {
		t.Fatalf("0 (= aus) muss gespeichert werden, kam %v", geladen.LesehistorieTage)
	}
	if TageOderStandard(geladen.SperreMinuten, -1) != 30 {
		t.Fatalf("sperre_minuten kam nicht zurück: %v", geladen.SperreMinuten)
	}
	// Andere Sektion speichert ohne die Felder → nil → die 0 bleibt stehen.
	if err := repo.SaveSettings(ctx, &SystemEinstellungen{LmfStichtag: "07-31"}); err != nil {
		t.Fatalf("SaveSettings (andere Sektion): %v", err)
	}
	if geladen, err = repo.GetSettings(ctx); err != nil {
		t.Fatalf("GetSettings (andere Sektion): %v", err)
	}
	if TageOderStandard(geladen.LesehistorieTage, -1) != 0 || TageOderStandard(geladen.SperreMinuten, -1) != 30 {
		t.Fatalf("Speichern ohne die Felder hat sie überschrieben: %v / %v", geladen.LesehistorieTage, geladen.SperreMinuten)
	}
}
