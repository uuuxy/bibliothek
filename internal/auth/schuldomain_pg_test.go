package auth

import (
	"context"
	"net/http"
	"testing"
)

// Mit gesetzter Domain der Schule genügt der Name vor dem @: Die Anmeldung trifft das Konto mit
// der vollen Adresse, am Mailserver und in der Datenbank.
func TestAnmeldung_NameOhneDomainMeldetMitDerAdresseDerSchuleAn(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("IMAP_HOST", "mock")
	t.Setenv(selbstanmeldeDomainVariable, "namentest.invalid")
	pool := pgPoolFuerSelbstanmeldung(t)
	const mail = "nora.name@namentest.invalid"
	raeumeKontoAb(t, pool, mail)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Nora', 'Namenprobe', $1, 'kollegium', true)`, mail); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM leser WHERE nachname = 'Namenprobe'`); err != nil {
			t.Errorf("Aufräumen leser: %v", err)
		}
	})

	if code, text := anmelden(t, pool, "nora.name"); code != http.StatusOK {
		t.Fatalf("Anmeldung mit dem Namen: Status %d (%q), erwartet 200", code, text)
	}
	if code, text := anmelden(t, pool, " Nora.Name "); code != http.StatusOK {
		t.Fatalf("Anmeldung mit Leerzeichen und Großbuchstaben: Status %d (%q), erwartet 200", code, text)
	}
	if code, text := anmelden(t, pool, mail); code != http.StatusOK {
		t.Fatalf("Anmeldung mit der vollen Adresse: Status %d (%q), erwartet 200", code, text)
	}
	// Mit Domain ist das Feld ein Textfeld, und der Browser lässt Leerzeichen stehen.
	if code, text := anmelden(t, pool, " "+mail+" "); code != http.StatusOK {
		t.Fatalf("Anmeldung mit der vollen Adresse und Leerzeichen: Status %d (%q), erwartet 200", code, text)
	}
}

// Ohne Domain der Schule ergänzt der Server nichts: Der Name allein trifft kein Konto.
func TestAnmeldung_NameOhneDomainDerSchuleTrifftKeinKonto(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("IMAP_HOST", "mock")
	t.Setenv(selbstanmeldeDomainVariable, "")
	pool := pgPoolFuerSelbstanmeldung(t)
	const mail = "otto.ohne@namentest.invalid"
	raeumeKontoAb(t, pool, mail)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Otto', 'Ohnedomain', $1, 'kollegium', true)`, mail); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM leser WHERE nachname = 'Ohnedomain'`); err != nil {
			t.Errorf("Aufräumen leser: %v", err)
		}
	})

	if code, _ := anmelden(t, pool, "otto.ohne"); code == http.StatusOK {
		t.Fatal("Anmeldung mit dem Namen ohne Domain der Schule gelang")
	}
}
