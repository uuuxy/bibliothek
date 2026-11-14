package auth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bibliothek/pkg/closeutil"

	"github.com/go-ldap/ldap/v3"
)

// Die Umgebungsvariablen des Verzeichnisdienstes. Gelesen wird bei jeder Prüfung wie bei
// IMAP_HOST; der Start prüft sie vorab (PruefeAnmeldeKonfig).
const (
	ldapURLVariable          = "LDAP_URL"
	ldapCADateiVariable      = "LDAP_CA_DATEI"
	ldapBindDNVariable       = "LDAP_BIND_DN"
	ldapBindPasswortVariable = "LDAP_BIND_PASSWORT"
	ldapBasisDNVariable      = "LDAP_BASIS_DN"
	ldapFilterVariable       = "LDAP_FILTER"
)

// ldapPlatzhalter steht in LDAP_FILTER für die LDAP-Kennung des Kontos.
const ldapPlatzhalter = "%s"

// ldapFrist ist die eine Frist für Verbinden, Dienstkonto, Suche und Anmeldung am
// Verzeichnisdienst. Sie ist so lang wie die des Mailservers, damit anmeldeFrist beiden Luft lässt.
const ldapFrist = imapFrist

// ldapBefehlsfrist gilt je Anfrage an den Verzeichnisdienst; go-ldap wartet sonst ohne Ende auf
// eine Antwort.
const ldapBefehlsfrist = 5 * time.Second

// ldapKonfig ist, was der Start über den Verzeichnisdienst liest.
type ldapKonfig struct {
	url, caDatei, bindDN, bindPasswort, basisDN, filter string
}

// leseLDAPKonfig liest die Einstellungen des Verzeichnisdienstes aus der Umgebung. Das Passwort
// des Dienstkontos bleibt, wie es ist: Leerzeichen können dazugehören.
func leseLDAPKonfig() ldapKonfig {
	return ldapKonfig{
		url:          strings.TrimSpace(os.Getenv(ldapURLVariable)),
		caDatei:      strings.TrimSpace(os.Getenv(ldapCADateiVariable)),
		bindDN:       strings.TrimSpace(os.Getenv(ldapBindDNVariable)),
		bindPasswort: os.Getenv(ldapBindPasswortVariable),
		basisDN:      strings.TrimSpace(os.Getenv(ldapBasisDNVariable)),
		filter:       strings.TrimSpace(os.Getenv(ldapFilterVariable)),
	}
}

// istEingerichtet sagt, ob es einen Verzeichnisdienst gibt; LDAP_URL entscheidet.
func (k ldapKonfig) istEingerichtet() bool {
	return k.url != ""
}

// pruefe prüft die Einstellungen, ohne den Verzeichnisdienst zu fragen.
func (k ldapKonfig) pruefe() error {
	u, err := url.Parse(k.url)
	if err != nil || u.Hostname() == "" {
		return fmt.Errorf("LDAP_URL %q ist keine Adresse eines Verzeichnisdienstes (ldaps://host:636 oder ldap://host:389)", k.url)
	}
	if u.Scheme != "ldaps" && u.Scheme != "ldap" {
		return fmt.Errorf("LDAP_URL beginnt mit ldaps:// oder ldap://, nicht mit %q", u.Scheme+"://")
	}
	for _, pflicht := range []struct{ name, wert string }{
		{ldapBindDNVariable, k.bindDN},
		{ldapBindPasswortVariable, k.bindPasswort},
		{ldapBasisDNVariable, k.basisDN},
		{ldapFilterVariable, k.filter},
	} {
		if pflicht.wert == "" {
			return fmt.Errorf("%s ist nicht gesetzt — LDAP_URL verlangt Dienstkonto, Suchbasis und Filter", pflicht.name)
		}
	}
	if len(k.suchbasen()) == 0 {
		return fmt.Errorf("LDAP_BASIS_DN %q nennt keinen Suchordner", k.basisDN)
	}
	if !strings.Contains(k.filter, ldapPlatzhalter) {
		return fmt.Errorf("LDAP_FILTER enthält kein %%s für die LDAP-Kennung, etwa (sAMAccountName=%%s)")
	}
	if _, err := ldap.CompileFilter(k.baueFilter("kennung")); err != nil {
		return fmt.Errorf("LDAP_FILTER ist kein gültiger Filter: %w", err)
	}
	if _, err := k.tlsKonfig(u.Hostname()); err != nil {
		return err
	}
	return nil
}

// baueFilter setzt die Kennung in LDAP_FILTER ein. Maskiert, sonst änderte eine Kennung wie
// „*)(uid=*" den Filter selbst und träfe einen fremden Eintrag (RFC 4515, 3).
func (k ldapKonfig) baueFilter(kennung string) string {
	return strings.ReplaceAll(k.filter, ldapPlatzhalter, ldap.EscapeFilter(kennung))
}

// tlsKonfig baut die Einstellungen für TLS zum Verzeichnisdienst. LDAP_CA_DATEI ergänzt die
// Zertifikate des Systems um die Zertifizierungsstelle der Schule; ein Active Directory weist sich
// oft mit einem Zertifikat aus, das nur sie bestätigt. Eine Prüfung abzuschalten gibt es nicht.
func (k ldapKonfig) tlsKonfig(host string) (*tls.Config, error) {
	konfig := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	if k.caDatei == "" {
		return konfig, nil
	}
	pem, err := os.ReadFile(filepath.Clean(k.caDatei))
	if err != nil {
		return nil, fmt.Errorf("LDAP_CA_DATEI lässt sich nicht lesen: %w", err)
	}
	zertifikate, err := x509.SystemCertPool()
	if err != nil {
		zertifikate = x509.NewCertPool()
	}
	if !zertifikate.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("LDAP_CA_DATEI %q enthält kein Zertifikat im PEM-Format", k.caDatei)
	}
	konfig.RootCAs = zertifikate
	return konfig, nil
}

// pruefeLDAPAnmeldung prüft das Passwort des Eintrags, den die LDAP-Kennung des Kontos im
// Verzeichnisdienst trifft: Das Dienstkonto sucht ihn mit LDAP_FILTER, dann meldet sich der
// Eintrag selbst mit dem Passwort an.
//
// Wie bei IMAP heißt nur die Antwort „Zugangsdaten falsch" (Code 49) falsches Passwort, dazu eine
// Kennung ohne Eintrag. Alles andere ist ein Ausfall und zählt keinen Fehlversuch.
func pruefeLDAPAnmeldung(ctx context.Context, kontoID, kennung, passwort string) error {
	// Ein leeres Passwort wäre eine anonyme Anmeldung, und die gelingt am Verzeichnisdienst
	// (RFC 4513, 5.1.2).
	if passwort == "" {
		return errAnmeldungAbgelehnt
	}
	k := leseLDAPKonfig()
	if !k.istEingerichtet() {
		slog.Error("LDAP-Konto, aber LDAP_URL ist nicht gesetzt", "konto", kontoID)
		return errAnmeldeserverNichtErreichbar
	}
	if err := k.pruefe(); err != nil {
		slog.Error("LDAP-Konfiguration ungültig", "fehler", err)
		return errAnmeldeserverNichtErreichbar
	}

	ctx, cancel := context.WithTimeout(ctx, ldapFrist)
	defer cancel()

	verbindung, err := verbindeLDAP(ctx, k)
	if err != nil {
		slog.Warn("LDAP: keine Verbindung zum Verzeichnisdienst", "fehler", err)
		return errAnmeldeserverNichtErreichbar
	}
	defer closeutil.Schliesse(verbindung, ldapVerbindung)
	// Endet die Frist, bricht das Schließen die wartende Anfrage ab.
	stopp := context.AfterFunc(ctx, func() { closeutil.Schliesse(verbindung, ldapVerbindung) })
	defer stopp()

	if err := verbindung.Bind(k.bindDN, k.bindPasswort); err != nil {
		slog.Error("LDAP: Dienstkonto abgelehnt", "fehler", err)
		return errAnmeldeserverNichtErreichbar
	}
	dn, err := findeLDAPEintrag(verbindung, k, kontoID, kennung)
	if err != nil {
		return err
	}
	if dn == "" {
		slog.Warn("LDAP: Die Kennung trifft keinen Eintrag", "konto", kontoID)
		return errAnmeldungAbgelehnt
	}
	if err := verbindung.Bind(dn, passwort); err != nil {
		if ldap.IsErrorWithCode(err, ldap.LDAPResultInvalidCredentials) {
			return errAnmeldungAbgelehnt
		}
		slog.Warn("LDAP: Anmeldung ohne Entscheidung", "konto", kontoID, "fehler", err)
		return errAnmeldeserverNichtErreichbar
	}
	return nil
}

// verbindeLDAP öffnet die Verbindung zum Verzeichnisdienst über TLS. Bei ldap:// ist StartTLS
// Pflicht; ohne TLS ginge das Passwort offen durchs Schulnetz.
func verbindeLDAP(ctx context.Context, k ldapKonfig) (*ldap.Conn, error) {
	u, err := url.Parse(k.url)
	if err != nil {
		return nil, err
	}
	tlsKonfig, err := k.tlsKonfig(u.Hostname())
	if err != nil {
		return nil, err
	}
	waehler := &net.Dialer{Timeout: ldapBefehlsfrist}
	if frist, ok := ctx.Deadline(); ok {
		waehler.Deadline = frist
	}
	verbindung, err := ldap.DialURL(k.url, ldap.DialWithDialer(waehler), ldap.DialWithTLSConfig(tlsKonfig))
	if err != nil {
		return nil, err
	}
	verbindung.SetTimeout(ldapBefehlsfrist)
	if u.Scheme == "ldap" {
		if err := verbindung.StartTLS(tlsKonfig); err != nil {
			closeutil.Schliesse(verbindung, ldapVerbindung)
			return nil, fmt.Errorf("starttls: %w", err)
		}
	}
	return verbindung, nil
}

// suchbasen zerlegt LDAP_BASIS_DN in die Ordner, in denen gesucht wird, an jedem Semikolon wie
// die Authentication containers der OPNsense. Ein mit \ maskiertes Semikolon gehört zum DN
// (RFC 4514, 2.4).
func (k ldapKonfig) suchbasen() []string {
	var basen []string
	var teil strings.Builder
	nimm := func() {
		if basis := strings.TrimSpace(teil.String()); basis != "" {
			basen = append(basen, basis)
		}
		teil.Reset()
	}
	maskiert := false
	for _, z := range k.basisDN {
		switch {
		case maskiert:
			maskiert = false
		case z == '\\':
			maskiert = true
		case z == ';':
			nimm()
			continue
		}
		teil.WriteRune(z)
	}
	nimm()
	return basen
}

// findeLDAPEintrag sucht den Eintrag zur Kennung in jedem Suchordner und liefert seinen DN, ""
// ohne Treffer. Liegen die Ordner ineinander, zählt ein Eintrag, den zwei finden, einmal. Trifft
// die Kennung mehr als einen Eintrag, ist das ein Fehler der Einrichtung: Welcher gemeint ist,
// lässt sich nicht entscheiden, und angemeldet wird keiner.
func findeLDAPEintrag(verbindung *ldap.Conn, k ldapKonfig, kontoID, kennung string) (string, error) {
	gefunden := map[string]string{}
	for _, basis := range k.suchbasen() {
		dns, err := sucheInBasis(verbindung, basis, k.baueFilter(kennung))
		if err != nil && !errors.Is(err, errMehrereEintraege) {
			return "", err
		}
		for _, dn := range dns {
			gefunden[strings.ToLower(dn)] = dn
		}
		if err != nil || len(gefunden) > 1 {
			slog.Error("LDAP: Die Kennung trifft mehrere Einträge", "konto", kontoID)
			return "", errAnmeldungAbgelehnt
		}
	}
	for _, dn := range gefunden {
		return dn, nil
	}
	return "", nil
}

// errMehrereEintraege heißt: Ein Suchordner hält mehr Treffer, als die Suche abholt.
var errMehrereEintraege = errors.New("ldap: mehr als ein eintrag im suchordner")

// sucheInBasis sucht in einem Ordner. Gibt es den Ordner nicht (Code 32), steht das im Log, und
// gesucht wird in den übrigen weiter: Ein vertippter Ordner soll nicht alle anderen aussperren.
func sucheInBasis(verbindung *ldap.Conn, basis, filter string) ([]string, error) {
	anfrage := ldap.NewSearchRequest(basis, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases,
		2, int(ldapBefehlsfrist/time.Second), false, filter,
		// "1.1" verlangt keine Attribute (RFC 4511, 4.5.1.8): Gebraucht wird nur der DN.
		[]string{"1.1"}, nil)
	ergebnis, err := verbindung.Search(anfrage)
	switch {
	case ldap.IsErrorWithCode(err, ldap.LDAPResultSizeLimitExceeded):
		return nil, errMehrereEintraege
	case ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject):
		slog.Error("LDAP: Einen Suchordner aus LDAP_BASIS_DN gibt es nicht", "fehler", err)
		return nil, nil
	case err != nil:
		slog.Warn("LDAP: Suche gescheitert", "fehler", err)
		return nil, errAnmeldeserverNichtErreichbar
	}
	dns := make([]string, 0, len(ergebnis.Entries))
	for _, eintrag := range ergebnis.Entries {
		dns = append(dns, eintrag.DN)
	}
	return dns, nil
}

// errKeinAnmeldeserver heißt: Weder Mailserver noch Verzeichnisdienst ist eingetragen.
var errKeinAnmeldeserver = errors.New("weder IMAP_HOST noch LDAP_URL ist gesetzt — ohne Mailserver oder Verzeichnisdienst ist keine Anmeldung möglich (lokal: IMAP_HOST=mock mit APP_ENV=local)")
