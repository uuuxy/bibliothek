package repository

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/testdaten"

	"github.com/jackc/pgx/v5"
)

// Die Anmeldung findet ein Konto über die E-Mail, wie eingegeben oder um die Domain ergänzt, und
// ein LDAP-Konto auch über seine Kennung; ein IMAP-Konto nie über eine Kennung. Trifft die Eingabe
// zwei Konten, meldet die Abfrage das, statt eines zu wählen.
func TestLadeAnmeldeKonto_FindetUeberEmailUndKennung(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE email LIKE '%@anmeldeverfahren.test'`); err != nil {
		t.Fatal(err)
	}
	ldapID := testdaten.LegeKontoAn(t, pool, "Lena", "Verzeichnis", "lena@anmeldeverfahren.test", "helfer", true)
	imapID := testdaten.LegeKontoAn(t, pool, "Ina", "Post", "ina@anmeldeverfahren.test", "helfer", true)
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM benutzer WHERE id = ANY($1)`, []string{ldapID, imapID}); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})
	if _, err := pool.Exec(ctx, `UPDATE benutzer SET anmeldeverfahren = 'ldap', ldap_kennung = 'LVerzeichnis' WHERE id = $1`, ldapID); err != nil {
		t.Fatal(err)
	}

	faelle := []struct {
		name, eingabe, email, want string
	}{
		{name: "Kennung, andere Schreibweise", eingabe: "lverzeichnis", email: "lverzeichnis@anmeldeverfahren.test", want: ldapID},
		{name: "E-Mail des LDAP-Kontos", eingabe: "LENA@anmeldeverfahren.test", email: "LENA@anmeldeverfahren.test", want: ldapID},
		{name: "um die Domain ergänzt", eingabe: "ina", email: "ina@anmeldeverfahren.test", want: imapID},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			konto, err := LadeAnmeldeKonto(ctx, pool, f.eingabe, f.email)
			if err != nil {
				t.Fatalf("LadeAnmeldeKonto(%q, %q) = %v, erwartet Konto %s", f.eingabe, f.email, err, f.want)
			}
			if konto.ID != f.want {
				t.Errorf("LadeAnmeldeKonto(%q, %q).ID = %s, erwartet %s", f.eingabe, f.email, konto.ID, f.want)
			}
		})
	}

	// Ein IMAP-Konto hat keine Kennung, die Anmeldung findet es nur über die E-Mail.
	if _, err := LadeAnmeldeKonto(ctx, pool, "Post", "Post"); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("LadeAnmeldeKonto(%q) = %v, erwartet pgx.ErrNoRows", "Post", err)
	}

	// Kennung des einen, E-Mail des anderen: Angemeldet wird keines.
	if _, err := LadeAnmeldeKonto(ctx, pool, "lverzeichnis", "ina@anmeldeverfahren.test"); !errors.Is(err, ErrAnmeldungMehrdeutig) {
		t.Errorf("LadeAnmeldeKonto(…) = %v, erwartet ErrAnmeldungMehrdeutig", err)
	}

	verfahren, kennung, err := LadeAnmeldeverfahren(ctx, pool, ldapID)
	if err != nil || verfahren != AnmeldeverfahrenLDAP || kennung != "LVerzeichnis" {
		t.Errorf("LadeAnmeldeverfahren(ldap) = %q, %q, %v, erwartet ldap, LVerzeichnis", verfahren, kennung, err)
	}
}

// Wechselt ein Konto zu IMAP, fällt seine Kennung weg; ein LDAP-Konto ohne Kennung und eine
// Kennung zweier Konten lehnt die Datenbank ab, und die Tür erkennt beides.
func TestAendere_AnmeldeverfahrenUndKennung(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE email LIKE '%@verfahrenwechsel.test'`); err != nil {
		t.Fatal(err)
	}
	erstes := testdaten.LegeKontoAn(t, pool, "Erst", "Konto", "erst@verfahrenwechsel.test", "helfer", true)
	zweites := testdaten.LegeKontoAn(t, pool, "Zweit", "Konto", "zweit@verfahrenwechsel.test", "helfer", true)
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM benutzer WHERE id = ANY($1)`, []string{erstes, zweites}); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})
	repo := NewKontoAbfragen(pool)
	text := func(s string) *string { return &s }

	stand, err := repo.Aendere(ctx, KontoAenderung{ID: erstes, Anmeldeverfahren: text("ldap"), LDAPKennung: text("erstk")})
	if err != nil || stand.Anmeldeverfahren != "ldap" || stand.LDAPKennung != "erstk" {
		t.Fatalf("Aendere(zu LDAP) = %+v, %v, erwartet ldap mit Kennung erstk", stand, err)
	}

	_, err = repo.Aendere(ctx, KontoAenderung{ID: zweites, Anmeldeverfahren: text("ldap"), LDAPKennung: text("ERSTK")})
	if !IstLDAPKennungKollision(err) {
		t.Errorf("Aendere(Kennung eines anderen) = %v, erwartet die Kollision der Kennung", err)
	}
	_, err = repo.Aendere(ctx, KontoAenderung{ID: zweites, Anmeldeverfahren: text("ldap")})
	if !IstLDAPKennungFehlend(err) {
		t.Errorf("Aendere(LDAP ohne Kennung) = %v, erwartet die fehlende Kennung", err)
	}
	if vergeben, err := repo.IstLDAPKennungVergeben(ctx, "Zweit@verfahrenwechsel.test", erstes); err != nil || !vergeben {
		t.Errorf("IstLDAPKennungVergeben(E-Mail eines anderen) = %v, %v, erwartet true", vergeben, err)
	}

	stand, err = repo.Aendere(ctx, KontoAenderung{ID: erstes, Anmeldeverfahren: text("imap")})
	if err != nil || stand.Anmeldeverfahren != "imap" || stand.LDAPKennung != "" {
		t.Errorf("Aendere(zurück zu IMAP) = %+v, %v, erwartet imap ohne Kennung", stand, err)
	}
}
