package repository

import (
	"bibliothek/internal/testdaten"
	"context"
	"testing"
)

// Die Empfängerliste des Bereitschafts-Alarms: nur aktive Admins mit Adresse.
// Ein inaktiver Admin (ausgeschieden) oder ein Mitarbeiter darf keine
// Angriffsflächen-Mails bekommen.
func TestAktiveAdminMails_NurAktiveAdmins(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	for _, b := range []struct {
		barcode, mail, rolle string
		aktiv                bool
	}{
		{"ADM-AKTIV", "admin-aktiv@example.org", "admin", true},
		{"ADM-WEG", "admin-weg@example.org", "admin", false},
		{"MIT-1", "mitarbeiter@example.org", "mitarbeiter", true},
	} {
		testdaten.LegeKontoAn(t, pool, "T", "T", b.mail, b.rolle, b.aktiv)
	}

	admins, err := NewBetriebszustandAbfragen(pool).LadeAktiveAdmins(ctx)
	if err != nil {
		t.Fatalf("LadeAktiveAdmins: %v", err)
	}
	gefunden := map[string]bool{}
	for _, a := range admins {
		gefunden[a.Email] = true
	}
	if !gefunden["admin-aktiv@example.org"] {
		t.Error("aktiver Admin fehlt in der Empfängerliste")
	}
	if gefunden["admin-weg@example.org"] || gefunden["mitarbeiter@example.org"] {
		t.Errorf("Empfängerliste enthält Unbefugte: %v", admins)
	}
}
