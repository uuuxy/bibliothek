package api

import (
	"context"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Der Live-Pfad der Bescheid-Prüfung: Einstellung leeren → sammleLage → Pruefe. Ohne die
// Verdrahtung in sammleLage stünde der Bereich auf „nicht erhoben", und der Unit-Test mit
// fertiger Lage bliebe trotzdem grün.
func TestBescheidAngabenErreichenDieSelbstpruefung(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	settingsRepo := repository.NewSystemSettingsRepository(pool)
	mailRepo := repository.NewMailSettingsRepository(pool)
	zustandRepo := repository.NewBetriebszustandRepository(pool)

	einst, err := settingsRepo.GetSettings(ctx)
	if err != nil || einst == nil {
		t.Fatalf("Einstellungen lesen: %v", err)
	}
	vorher := einst.BescheidSchulnummer
	t.Cleanup(func() {
		patch := &repository.EinstellungenPatch{BescheidSchulnummer: &vorher}
		if err := settingsRepo.SaveSettings(context.Background(), patch); err != nil {
			t.Logf("Schulnummer zurücksetzen: %v", err)
		}
	})
	setze := func(wert string) {
		t.Helper()
		if err := settingsRepo.SaveSettings(ctx, &repository.EinstellungenPatch{BescheidSchulnummer: &wert}); err != nil {
			t.Fatalf("Schulnummer setzen: %v", err)
		}
	}
	befund := func() Befund {
		t.Helper()
		for _, b := range Pruefe(srv.sammleLage(ctx, settingsRepo, mailRepo, zustandRepo)) {
			if b.Bereich == "Schadensersatz-Bescheid" {
				return b
			}
		}
		t.Fatal("Die Selbstprüfung enthält keinen Bereich 'Schadensersatz-Bescheid'")
		return Befund{}
	}

	setze("")
	if b := befund(); b.Stufe != StufeWarnung || !strings.Contains(b.Befund, "Schulnummer") {
		t.Fatalf("leere Schulnummer erreicht die Selbstprüfung nicht: %+v", b)
	}
	setze("1234")
	if b := befund(); strings.Contains(b.Befund, "Schulnummer") {
		t.Errorf("eingetragene Schulnummer wird weiter als fehlend gemeldet: %s", b.Befund)
	}
}
