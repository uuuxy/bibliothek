package repository

import (
	"context"
	"testing"
)

// Speichern und wieder Laden am Postgres. Die Tests mit pgxmock daneben sehen nur, welche
// Schlüssel ins Upsert gehen; ob das Laden den Wert wieder einfüllt, entscheidet der case-Zweig
// in setzeEinstellung. Ein Feld ohne diesen Zweig speichert scheinbar und liest leer. Geprüft
// wird am Verteiler der Alarme: Kommt er nicht aus der Datenbank zurück, fällt der Alarm auf
// alle aktiven Admins zurück.
func TestEinstellungenRoundtrip_AlarmEmpfaenger(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewEinstellungsAbfragen(pool)

	verteiler := "nberger@philipp-reis-schule.de, it@schule.example"
	if err := repo.Schreibe(ctx, &EinstellungenPatch{AlarmEmpfaenger: &verteiler}); err != nil {
		t.Fatalf("Schreibe: %v", err)
	}

	geladen, err := repo.Lade(ctx)
	if err != nil {
		t.Fatalf("Lade: %v", err)
	}
	if geladen.AlarmEmpfaenger == nil || *geladen.AlarmEmpfaenger != verteiler {
		t.Fatalf("alarm_empfaenger kam nicht zurück: %v (gespeichert war %q)",
			geladen.AlarmEmpfaenger, verteiler)
	}

	// Der leere Text ist ein Wert (Rückfall auf alle Admins) und muss einen vorhandenen
	// Verteiler überschreiben können; sonst wird man eine gesetzte Liste nicht mehr los.
	leer := ""
	if err := repo.Schreibe(ctx, &EinstellungenPatch{AlarmEmpfaenger: &leer}); err != nil {
		t.Fatalf("Schreibe (leer): %v", err)
	}
	geladen, err = repo.Lade(ctx)
	if err != nil {
		t.Fatalf("Lade (leer): %v", err)
	}
	if geladen.AlarmEmpfaenger != nil && *geladen.AlarmEmpfaenger != "" {
		t.Fatalf("leerer Verteiler überschreibt nicht: %q", *geladen.AlarmEmpfaenger)
	}
}

// Die Fristen des Datenschutzes und der Sitzung: nil lässt den gespeicherten Wert stehen, ein
// gesetzter Zeiger schreibt, auch die 0, denn 0 heißt hier „aus" und ist ein Wert. Ohne diese
// Unterscheidung schaltete das Speichern einer anderen Kategorie die Befristung der
// Lesehistorie ab.
func TestEinstellungenRoundtrip_DatenschutzZeigerSemantik(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewEinstellungsAbfragen(pool)

	// Die Prüfung beginnt bei den Vorgaben; was andere Tests hier gespeichert haben, verdeckte sie.
	entferne := func() {
		if _, err := pool.Exec(ctx, `DELETE FROM system_einstellungen WHERE schluessel = ANY($1)`,
			[]string{"lesehistorie_tage", "lesehistorie_lernmittel_tage", "sperre_minuten", "lmf_stichtag"}); err != nil {
			t.Fatalf("Einstellungen entfernen: %v", err)
		}
	}
	entferne()
	t.Cleanup(entferne)

	geladen, err := repo.Lade(ctx)
	if err != nil {
		t.Fatalf("Lade: %v", err)
	}
	if TageOderVorgabe(geladen.LesehistorieTage, -1) != LesehistorieTageVorgabe ||
		TageOderVorgabe(geladen.LesehistorieLernmittelTage, -1) != LesehistorieLernmittelTageVorgabe ||
		TageOderVorgabe(geladen.SperreMinuten, -1) != SperreMinutenVorgabe {
		t.Fatalf("Vorgaben fehlen: %+v", geladen)
	}

	null, dreissig := 0, 30
	if err := repo.Schreibe(ctx, &EinstellungenPatch{LesehistorieTage: &null, SperreMinuten: &dreissig}); err != nil {
		t.Fatalf("Schreibe: %v", err)
	}
	if geladen, err = repo.Lade(ctx); err != nil {
		t.Fatalf("Lade: %v", err)
	}
	if TageOderVorgabe(geladen.LesehistorieTage, -1) != 0 {
		t.Fatalf("0 (= aus) muss gespeichert werden, kam %v", geladen.LesehistorieTage)
	}
	if TageOderVorgabe(geladen.SperreMinuten, -1) != 30 {
		t.Fatalf("sperre_minuten kam nicht zurück: %v", geladen.SperreMinuten)
	}
	// Eine andere Kategorie speichert ohne die Felder: Sie sind nil, die 0 bleibt stehen.
	if err := repo.Schreibe(ctx, &EinstellungenPatch{LMFStichtag: ptr("07-31")}); err != nil {
		t.Fatalf("Schreibe (andere Kategorie): %v", err)
	}
	if geladen, err = repo.Lade(ctx); err != nil {
		t.Fatalf("Lade (andere Kategorie): %v", err)
	}
	if TageOderVorgabe(geladen.LesehistorieTage, -1) != 0 || TageOderVorgabe(geladen.SperreMinuten, -1) != 30 {
		t.Fatalf("Speichern ohne die Felder hat sie überschrieben: %v / %v", geladen.LesehistorieTage, geladen.SperreMinuten)
	}
}

// Das Speichern je Kategorie am Postgres: Die Tests mit pgxmock sehen nur, welche Schlüssel ins
// Upsert gehen, nicht was danach in der Datenbank steht.
//
// Der Test stellt einen vollen, von der Vorgabe abweichenden Stand her, speichert dann eine
// Kategorie (wie „Datenschutz & Sitzung speichern") und sieht nach, ob die anderen noch stehen.
// Mit einem Speichern, das jedes Feld aus einem Typ voller Nullwerte schreibt, wären Leseclub,
// Bestellbedarf-Warnung und Preiserfassung danach ausgeschaltet.
func TestEinstellungenRoundtrip_KategorieSpeichernLaesstDenRestInRuhe(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewEinstellungsAbfragen(pool)

	if err := repo.Schreibe(ctx, &EinstellungenPatch{
		FerienLeseclubAktiv:        ptr(true),
		FerienLeseclubZieldatum:    ptr("2027-01-06"),
		LMFStichtag:                ptr("08-15"),
		MaxAusleihenSchueler:       ptr(9),
		FristBuchTage:              ptr(28),
		FristMedienTage:            ptr(10),
		UeberfaelligKarenzTage:     ptr(20),
		UeberfaelligSperreAbMedien: ptr(3),
		BestellbedarfWarnungAktiv:  ptr(true),
		BestellbedarfSchwelle:      ptr(7),
		BestelllinkGueltigkeitTage: ptr(45),
		PreiseErfassen:             ptr(true),
		SchuleName:                 ptr("Philipp-Reis-Schule"),
		SchuleOrt:                  ptr("Friedrichsdorf"),
	}); err != nil {
		t.Fatalf("Ausgangsstand: %v", err)
	}

	// Ein Speichern in einer einzigen Kategorie.
	if err := repo.Schreibe(ctx, &EinstellungenPatch{
		LesehistorieTage:           ptr(120),
		LesehistorieLernmittelTage: ptr(700),
		ThekeLeerenMinuten:         ptr(4),
		SperreMinuten:              ptr(12),
	}); err != nil {
		t.Fatalf("Kategorie speichern: %v", err)
	}

	nachher, err := repo.Lade(ctx)
	if err != nil {
		t.Fatalf("Lade: %v", err)
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
		{"max_overdue_days", nachher.UeberfaelligKarenzTage, 20},
		{"max_overdue_items", nachher.UeberfaelligSperreAbMedien, 3},
		{"bestellbedarf_schwelle", nachher.BestellbedarfSchwelle, 7},
		{"bestelllink_gueltigkeit_tage", nachher.BestelllinkGueltigkeitTage, 45},
	} {
		if f.ist != f.soll {
			t.Errorf("%s wurde von einer fremden Kategorie zurückgesetzt: %d statt %d", f.name, f.ist, f.soll)
		}
	}
	if nachher.LMFStichtag != "08-15" || nachher.SchuleName != "Philipp-Reis-Schule" {
		t.Errorf("Text-Einstellungen überschrieben: Stichtag=%q Schule=%q", nachher.LMFStichtag, nachher.SchuleName)
	}
	if nachher.FerienLeseclubZieldatum == nil || *nachher.FerienLeseclubZieldatum != "2027-01-06" {
		t.Errorf("Zieldatum des Leseclubs verloren: %v", nachher.FerienLeseclubZieldatum)
	}

	// Die gespeicherte Kategorie ist angekommen.
	if TageOderVorgabe(nachher.LesehistorieTage, -1) != 120 || TageOderVorgabe(nachher.SperreMinuten, -1) != 12 {
		t.Errorf("die gespeicherte Kategorie fehlt: %v / %v", nachher.LesehistorieTage, nachher.SperreMinuten)
	}
}
