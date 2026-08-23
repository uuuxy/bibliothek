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
	if err := repo.SaveSettings(ctx, &EinstellungenPatch{AlarmEmpfaenger: &verteiler}); err != nil {
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
	if err := repo.SaveSettings(ctx, &EinstellungenPatch{AlarmEmpfaenger: &leer}); err != nil {
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
	if err := repo.SaveSettings(ctx, &EinstellungenPatch{LesehistorieTage: &null, SperreMinuten: &dreissig}); err != nil {
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
	if err := repo.SaveSettings(ctx, &EinstellungenPatch{LmfStichtag: ptr("07-31")}); err != nil {
		t.Fatalf("SaveSettings (andere Sektion): %v", err)
	}
	if geladen, err = repo.GetSettings(ctx); err != nil {
		t.Fatalf("GetSettings (andere Sektion): %v", err)
	}
	if TageOderStandard(geladen.LesehistorieTage, -1) != 0 || TageOderStandard(geladen.SperreMinuten, -1) != 30 {
		t.Fatalf("Speichern ohne die Felder hat sie überschrieben: %v / %v", geladen.LesehistorieTage, geladen.SperreMinuten)
	}
}

// Das Gate für das Speichern JE KATEGORIE (23.08.2026) — am echten Postgres, weil
// die pgxmock-Tests nur sehen, welche Schlüssel ins Upsert gehen, nicht was danach
// in der Datenbank steht.
//
// Ablauf: einen vollen, von der Vorgabe abweichenden Stand herstellen, dann EINE
// Kategorie speichern (wie ein Klick auf „Datenschutz & Sitzung speichern"), dann
// nachsehen, ob die anderen sechs Kategorien noch stehen. Gegen den Stand vor dem
// Patch ist dieser Test rot: Dort schrieb jedes Speichern elf Schlüssel aus einem
// Struct voller Nullwerte, und der Klick hätte Leseclub, Bestellbedarf-Warnung und
// Preiserfassung ausgeschaltet.
func TestSettingsRoundtrip_KategorieSpeichernLaesstDenRestInRuhe(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewSystemSettingsRepository(pool)

	if err := repo.SaveSettings(ctx, &EinstellungenPatch{
		FerienLeseclubAktiv:       ptr(true),
		FerienLeseclubZieldatum:   ptr("2027-01-06"),
		LmfStichtag:               ptr("08-15"),
		MaxAusleihenSchueler:      ptr(9),
		FristBuchTage:             ptr(28),
		FristMedienTage:           ptr(10),
		MaxOverdueDays:            ptr(20),
		MaxOverdueItems:           ptr(3),
		BestellbedarfWarnungAktiv: ptr(true),
		BestellbedarfSchwelle:     ptr(7),
		PreiseErfassen:            ptr(true),
		SchuleName:                ptr("Philipp-Reis-Schule"),
		SchuleOrt:                 ptr("Friedrichsdorf"),
	}); err != nil {
		t.Fatalf("Ausgangsstand: %v", err)
	}

	// Ein Klick in einer einzigen Kategorie.
	if err := repo.SaveSettings(ctx, &EinstellungenPatch{
		LesehistorieTage:           ptr(120),
		LesehistorieLernmittelTage: ptr(700),
		ThekeLeerenMinuten:         ptr(4),
		SperreMinuten:              ptr(12),
	}); err != nil {
		t.Fatalf("Kategorie speichern: %v", err)
	}

	nachher, err := repo.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}

	if !nachher.FerienLeseclubAktiv || !nachher.BestellbedarfWarnungAktiv || !nachher.PreiseErfassen {
		t.Errorf("ein Speichern in einer fremden Kategorie hat Schalter umgelegt: Leseclub=%v Bestellwarnung=%v Preise=%v",
			nachher.FerienLeseclubAktiv, nachher.BestellbedarfWarnungAktiv, nachher.PreiseErfassen)
	}
	for _, f := range []struct {
		name string
		ist  int
		soll int
	}{
		{"max_ausleihen_schueler", nachher.MaxAusleihenSchueler, 9},
		{"frist_buch_tage", nachher.FristBuchTage, 28},
		{"frist_medien_tage", nachher.FristMedienTage, 10},
		{"max_overdue_days", nachher.MaxOverdueDays, 20},
		{"max_overdue_items", nachher.MaxOverdueItems, 3},
		{"bestellbedarf_schwelle", nachher.BestellbedarfSchwelle, 7},
	} {
		if f.ist != f.soll {
			t.Errorf("%s wurde von einer fremden Kategorie zurückgesetzt: %d statt %d", f.name, f.ist, f.soll)
		}
	}
	if nachher.LmfStichtag != "08-15" || nachher.SchuleName != "Philipp-Reis-Schule" {
		t.Errorf("Text-Einstellungen überschrieben: Stichtag=%q Schule=%q", nachher.LmfStichtag, nachher.SchuleName)
	}
	if nachher.FerienLeseclubZieldatum == nil || *nachher.FerienLeseclubZieldatum != "2027-01-06" {
		t.Errorf("Zieldatum des Leseclubs verloren: %v", nachher.FerienLeseclubZieldatum)
	}

	// Und die gespeicherte Kategorie ist tatsächlich angekommen.
	if TageOderStandard(nachher.LesehistorieTage, -1) != 120 || TageOderStandard(nachher.SperreMinuten, -1) != 12 {
		t.Errorf("die gespeicherte Kategorie fehlt: %v / %v", nachher.LesehistorieTage, nachher.SperreMinuten)
	}
}
