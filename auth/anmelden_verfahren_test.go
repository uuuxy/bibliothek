package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v5"
)

// Die Anmeldung sucht zuerst das Konto und fragt dann den Server, den es nennt. Die Tests halten
// die Reihenfolge fest und dass ein LDAP-Konto am Verzeichnisdienst geprüft wird.

var anmeldeKontoSpalten = []string{"id", "rolle", "vorname", "nachname", "aktiv", "email", "beantragt", "anmeldeverfahren", "ldap_kennung"}

// Eine Eingabe ohne Konto erreicht weder Mailserver noch Verzeichnisdienst: Ein Active Directory,
// das nach Fehlversuchen sperrt, sperrte sonst Konten, nur weil jemand an der Bibliothek Namen
// durchprobiert.
func TestAnmelden_OhneKontoFragtKeinenServer(t *testing.T) {
	a, mock := newTestAnmeldung(t, 12*time.Hour)
	befehle := mitMiniIMAP(t, a, "OK")
	v := &miniVerzeichnis{treffer: einEintrag, passwoerter: map[string]string{testKontoDN: testKontoPasswort}}
	starteMiniVerzeichnis(t, v)
	t.Setenv(selbstanmeldeDomainVariable, "")

	mock.ExpectQuery(benutzerSelect).
		WithArgs("mmueller", "mmueller").
		WillReturnRows(pgxmock.NewRows(anmeldeKontoSpalten))

	rec := rufeAnmelden(t, a, mock, `{"email":"mmueller","password":"richtig"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Anmelden: Status %d, erwartet 401: %s", rec.Code, rec.Body.String())
	}
	select {
	case zeile := <-befehle:
		t.Errorf("Mailserver bekam %q, erwartet nichts für eine Eingabe ohne Konto", zeile)
	default:
	}
	if binds, _, _ := v.gesehen(); len(binds) != 0 {
		t.Errorf("Verzeichnisdienst bekam Binds %q, erwartet keine", binds)
	}
}

// Ein LDAP-Konto meldet sich mit seiner Kennung an; das Passwort prüft der Verzeichnisdienst,
// nicht der Mailserver.
func TestAnmelden_LDAPKontoMeldetSichMitKennungAn(t *testing.T) {
	a, mock := newTestAnmeldung(t, 12*time.Hour)
	befehle := mitMiniIMAP(t, a, "NO nicht hier")
	v := &miniVerzeichnis{treffer: einEintrag, passwoerter: map[string]string{testKontoDN: testKontoPasswort}}
	starteMiniVerzeichnis(t, v)
	t.Setenv(selbstanmeldeDomainVariable, "")

	mock.ExpectQuery(benutzerSelect).
		WithArgs("mmueller", "mmueller").
		WillReturnRows(pgxmock.NewRows(anmeldeKontoSpalten).
			AddRow("u-ldap", "admin", "Max", "Muster", true, "max@schule.test", false, "ldap", "mmueller"))
	erwarteSitzungAngelegt(mock, "u-ldap")

	rec := rufeAnmelden(t, a, mock, `{"email":"mmueller","password":"`+testKontoPasswort+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Anmelden: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	if binds, _, _ := v.gesehen(); len(binds) != 2 || binds[1] != testKontoDN {
		t.Errorf("Binds = %q, erwartet Dienstkonto und dann %q", binds, testKontoDN)
	}
	select {
	case zeile := <-befehle:
		t.Errorf("Mailserver bekam %q, erwartet nichts für ein LDAP-Konto", zeile)
	default:
	}

	// Dasselbe Konto mit falschem Passwort: 401, und der Verzeichnisdienst hat entschieden.
	mock.ExpectQuery(benutzerSelect).
		WithArgs("mmueller", "mmueller").
		WillReturnRows(pgxmock.NewRows(anmeldeKontoSpalten).
			AddRow("u-ldap", "admin", "Max", "Muster", true, "max@schule.test", false, "ldap", "mmueller"))
	if rec := rufeAnmelden(t, a, mock, `{"email":"mmueller","password":"falsch"}`); rec.Code != http.StatusUnauthorized {
		t.Errorf("Anmelden mit falschem Passwort: Status %d, erwartet 401", rec.Code)
	}
}

// Trifft die Eingabe ein Konto über die E-Mail und ein anderes über die LDAP-Kennung, wird keines
// angemeldet, und kein Server gefragt.
func TestAnmelden_EingabeTrifftZweiKonten(t *testing.T) {
	aktiviereIMAPAttrappe(t)
	a, mock := newTestAnmeldung(t, 12*time.Hour)

	mock.ExpectQuery(benutzerSelect).
		WithArgs("doppelt@schule.test", "doppelt@schule.test").
		WillReturnRows(pgxmock.NewRows(anmeldeKontoSpalten).
			AddRow("u-1", "admin", "Eins", "Konto", true, "doppelt@schule.test", false, "imap", "").
			AddRow("u-2", "helfer", "Zwei", "Konto", true, "zwei@schule.test", false, "ldap", "doppelt@schule.test"))

	rec := rufeAnmelden(t, a, mock, `{"email":"doppelt@schule.test","password":"egal"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Anmelden: Status %d, erwartet 401: %s", rec.Code, rec.Body.String())
	}
}

// Die Absage kommt ohne Konto so spät wie nach der Antwort des Servers: Am Unterschied ließe sich
// sonst ablesen, welche Adressen ein Konto haben.
func TestAnmelden_AbsageWartetGleichLang(t *testing.T) {
	const frist = 200 * time.Millisecond
	a, mock := newTestAnmeldung(t, 12*time.Hour)
	a.absageFrist = frist
	mitMiniIMAP(t, a, "NO falsch")
	t.Setenv(selbstanmeldeDomainVariable, "")
	t.Setenv("LDAP_URL", "")

	mock.ExpectQuery(benutzerSelect).
		WithArgs("niemand@schule.test", "niemand@schule.test").
		WillReturnRows(pgxmock.NewRows(anmeldeKontoSpalten))
	beginn := time.Now()
	if rec := rufeAnmelden(t, a, mock, `{"email":"niemand@schule.test","password":"egal"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("Anmelden ohne Konto: Status %d, erwartet 401", rec.Code)
	}
	if dauer := time.Since(beginn); dauer < frist {
		t.Errorf("Absage ohne Konto nach %v, erwartet frühestens nach %v", dauer, frist)
	}

	mock.ExpectQuery(benutzerSelect).
		WithArgs("jemand@schule.test", "jemand@schule.test").
		WillReturnRows(pgxmock.NewRows(anmeldeKontoSpalten).
			AddRow("u-1", "helfer", "Je", "Mand", true, "jemand@schule.test", false, "imap", ""))
	beginn = time.Now()
	if rec := rufeAnmelden(t, a, mock, `{"email":"jemand@schule.test","password":"falsch"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("Anmelden mit falschem Passwort: Status %d, erwartet 401", rec.Code)
	}
	if dauer := time.Since(beginn); dauer < frist {
		t.Errorf("Absage nach falschem Passwort nach %v, erwartet frühestens nach %v", dauer, frist)
	}
}
