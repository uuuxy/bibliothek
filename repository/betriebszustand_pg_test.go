package repository

import (
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
		if _, err := pool.Exec(ctx, `
			INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
			VALUES ($1, 'T', 'T', $2, $3, $4)`, b.barcode, b.mail, b.rolle, b.aktiv); err != nil {
			t.Fatalf("Seed %s: %v", b.barcode, err)
		}
	}

	mails, err := NewBetriebszustandRepository(pool).AktiveAdminMails(ctx)
	if err != nil {
		t.Fatalf("AktiveAdminMails: %v", err)
	}
	gefunden := map[string]bool{}
	for _, m := range mails {
		gefunden[m] = true
	}
	if !gefunden["admin-aktiv@example.org"] {
		t.Error("aktiver Admin fehlt in der Empfängerliste")
	}
	if gefunden["admin-weg@example.org"] || gefunden["mitarbeiter@example.org"] {
		t.Errorf("Empfängerliste enthält Unbefugte: %v", mails)
	}
}
