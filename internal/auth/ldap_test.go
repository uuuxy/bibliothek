package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	ber "github.com/go-asn1-ber/asn1-ber"
	"github.com/go-ldap/ldap/v3"
)

// Die Tests sprechen über TLS mit einem kleinen Verzeichnisdienst, der LDAP über die Leitung
// spricht (Bind, Suche, StartTLS), nicht mit einer Attrappe der eigenen Logik. Geprüft wird, was
// beim Server ankommt: welcher Filter, welche Anmeldung, und ob vor TLS schon etwas kam.

const (
	testDienstDN       = "cn=leser,dc=schule,dc=test"
	testDienstPasswort = "dienst-geheim"
	testKontoDN        = "cn=Muster,ou=lehrkraefte,dc=schule,dc=test"
	testKontoPasswort  = "richtig"
)

// miniVerzeichnis ist ein Verzeichnisdienst mit einem Dienstkonto und festen Einträgen.
type miniVerzeichnis struct {
	// treffer liefert zu Suchordner und Filter, wie sie ankamen, die DNs der Einträge.
	treffer func(basis, filter string) []string
	// fehlendeBasen sind Suchordner, die es nicht gibt: Der Server antwortet mit Code 32.
	fehlendeBasen map[string]bool
	// passwoerter ordnet einem DN sein Passwort zu.
	passwoerter map[string]string
	// mitStartTLS: Der Server spricht zuerst ohne TLS (ldap://) und nimmt StartTLS an;
	// ohneStartTLS: ohne TLS, und StartTLS lehnt er ab. schweigt: Er beantwortet kein Bind.
	mitStartTLS, ohneStartTLS, schweigt bool

	mu       sync.Mutex
	binds    []string
	filter   []string
	basen    []string
	klartext bool
}

func (v *miniVerzeichnis) merkeBind(dn string, istTLS bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.binds = append(v.binds, dn)
	if !istTLS {
		v.klartext = true
	}
}

func (v *miniVerzeichnis) merkeSuche(basis, filter string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.filter = append(v.filter, filter)
	v.basen = append(v.basen, basis)
}

func (v *miniVerzeichnis) gesehen() (binds, filter []string, klartext bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return append([]string(nil), v.binds...), append([]string(nil), v.filter...), v.klartext
}

// starteMiniVerzeichnis startet den Server und stellt die Umgebung darauf ein, mit dem
// Zertifikat des Servers als LDAP_CA_DATEI und dem Filter (sAMAccountName=%s).
func starteMiniVerzeichnis(t *testing.T, v *miniVerzeichnis) {
	t.Helper()
	zertifikat, pemDaten := testZertifikat(t)
	caDatei := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caDatei, pemDaten, 0o600); err != nil {
		t.Fatal(err)
	}
	tlsKonfig := &tls.Config{Certificates: []tls.Certificate{zertifikat}, MinVersion: tls.VersionTLS12}

	schema := "ldaps"
	var l net.Listener
	var err error
	if v.mitStartTLS || v.ohneStartTLS {
		schema = "ldap"
		l, err = net.Listen("tcp", "127.0.0.1:0")
	} else {
		l, err = tls.Listen("tcp", "127.0.0.1:0", tlsKonfig)
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := l.Close(); err != nil {
			t.Logf("Listener schließen: %v", err)
		}
	})
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go v.bediene(c, tlsKonfig, schema == "ldaps")
		}
	}()

	t.Setenv("LDAP_URL", schema+"://"+l.Addr().String())
	t.Setenv("LDAP_CA_DATEI", caDatei)
	t.Setenv("LDAP_BIND_DN", testDienstDN)
	t.Setenv("LDAP_BIND_PASSWORT", testDienstPasswort)
	t.Setenv("LDAP_BASIS_DN", "dc=schule,dc=test")
	t.Setenv("LDAP_FILTER", "(sAMAccountName=%s)")
}

// bediene beantwortet die Anfragen einer Verbindung, bis der Client geht.
func (v *miniVerzeichnis) bediene(c net.Conn, tlsKonfig *tls.Config, istTLS bool) {
	defer func() { c.Close() }() //nolint:errcheck // Testserver: der Client ist dann schon fort
	for {
		paket, err := ber.ReadPacket(c)
		if err != nil || len(paket.Children) < 2 {
			return
		}
		id, ok := paket.Children[0].Value.(int64)
		if !ok {
			return
		}
		op := paket.Children[1]
		switch op.Tag {
		case ldap.ApplicationBindRequest:
			dn, passwort := op.Children[1].Data.String(), op.Children[2].Data.String()
			v.merkeBind(dn, istTLS)
			if v.schweigt {
				continue
			}
			code := ldap.LDAPResultInvalidCredentials
			if (dn == testDienstDN && passwort == testDienstPasswort) ||
				(v.passwoerter[dn] != "" && v.passwoerter[dn] == passwort) {
				code = ldap.LDAPResultSuccess
			}
			schreibeLDAP(c, ldapErgebnis(id, ldap.ApplicationBindResponse, code))
		case ldap.ApplicationSearchRequest:
			basis := op.Children[0].Data.String()
			filter, err := ldap.DecompileFilter(op.Children[6])
			if err != nil {
				return
			}
			v.merkeSuche(basis, filter)
			if v.fehlendeBasen[basis] {
				schreibeLDAP(c, ldapErgebnis(id, ldap.ApplicationSearchResultDone, ldap.LDAPResultNoSuchObject))
				continue
			}
			for _, dn := range v.treffer(basis, filter) {
				schreibeLDAP(c, ldapEintrag(id, dn))
			}
			schreibeLDAP(c, ldapErgebnis(id, ldap.ApplicationSearchResultDone, ldap.LDAPResultSuccess))
		case ldap.ApplicationExtendedRequest:
			if istTLS || v.ohneStartTLS {
				schreibeLDAP(c, ldapErgebnis(id, ldap.ApplicationExtendedResponse, ldap.LDAPResultProtocolError))
				continue
			}
			schreibeLDAP(c, ldapErgebnis(id, ldap.ApplicationExtendedResponse, ldap.LDAPResultSuccess))
			sicher := tls.Server(c, tlsKonfig)
			if err := sicher.Handshake(); err != nil {
				return
			}
			c, istTLS = sicher, true
		default:
			return
		}
	}
}

// schreibeLDAP schickt eine Antwort; ist der Client fort, endet die Verbindung beim nächsten
// Lesen ohnehin.
func schreibeLDAP(c net.Conn, p *ber.Packet) {
	c.Write(p.Bytes()) //nolint:errcheck // Testserver, siehe oben
}

func ldapHuelle(id int64) *ber.Packet {
	p := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "LDAP Response")
	p.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, id, "MessageID"))
	return p
}

func ldapErgebnis(id int64, tag ber.Tag, code int) *ber.Packet {
	op := ber.Encode(ber.ClassApplication, ber.TypeConstructed, tag, nil, "Response")
	op.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagEnumerated, int64(code), "resultCode"))
	op.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, "", "matchedDN"))
	op.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, "", "diagnosticMessage"))
	p := ldapHuelle(id)
	p.AppendChild(op)
	return p
}

func ldapEintrag(id int64, dn string) *ber.Packet {
	op := ber.Encode(ber.ClassApplication, ber.TypeConstructed, ldap.ApplicationSearchResultEntry, nil, "Entry")
	op.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, dn, "objectName"))
	op.AppendChild(ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "attributes"))
	p := ldapHuelle(id)
	p.AppendChild(op)
	return p
}

// testZertifikat erzeugt ein selbst signiertes Zertifikat für 127.0.0.1 und liefert es samt PEM.
func testZertifikat(t *testing.T) (tls.Certificate, []byte) {
	t.Helper()
	schluessel, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	vorlage := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "127.0.0.1"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		IsCA:        true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, vorlage, vorlage, &schluessel.PublicKey, schluessel)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: schluessel},
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// einEintrag trifft in jedem Suchordner genau den Eintrag testKontoDN, wenn der Filter die
// Kennung mmueller nennt.
func einEintrag(_, filter string) []string {
	if filter == "(sAMAccountName=mmueller)" {
		return []string{testKontoDN}
	}
	return nil
}

func TestPruefeLDAPAnmeldung_RichtigesPasswort(t *testing.T) {
	v := &miniVerzeichnis{treffer: einEintrag, passwoerter: map[string]string{testKontoDN: testKontoPasswort}}
	starteMiniVerzeichnis(t, v)

	if err := pruefeLDAPAnmeldung(context.Background(), "konto-1", "mmueller", testKontoPasswort); err != nil {
		t.Fatalf("pruefeLDAPAnmeldung(…) = %v, erwartet nil", err)
	}
	binds, filter, klartext := v.gesehen()
	if want := []string{testDienstDN, testKontoDN}; strings.Join(binds, "|") != strings.Join(want, "|") {
		t.Errorf("Binds = %q, erwartet erst das Dienstkonto, dann den Eintrag %q", binds, want)
	}
	if len(filter) != 1 || filter[0] != "(sAMAccountName=mmueller)" {
		t.Errorf("Filter = %q, erwartet [(sAMAccountName=mmueller)]", filter)
	}
	if klartext {
		t.Error("ein Bind kam ohne TLS an")
	}
}

// Nur Code 49 und eine Kennung ohne Eintrag heißen falsches Passwort; mehrere Einträge sind ein
// Fehler der Einrichtung und melden niemanden an.
func TestPruefeLDAPAnmeldung_Abgelehnt(t *testing.T) {
	faelle := []struct {
		name, kennung, passwort string
		treffer                 func(string, string) []string
	}{
		{name: "falsches Passwort", kennung: "mmueller", passwort: "falsch", treffer: einEintrag},
		{name: "Kennung ohne Eintrag", kennung: "unbekannt", passwort: testKontoPasswort, treffer: einEintrag},
		{name: "Kennung trifft zwei Einträge", kennung: "mmueller", passwort: testKontoPasswort,
			treffer: func(_, _ string) []string { return []string{testKontoDN, "cn=Zweiter,dc=schule,dc=test"} }},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			v := &miniVerzeichnis{treffer: f.treffer, passwoerter: map[string]string{testKontoDN: testKontoPasswort}}
			starteMiniVerzeichnis(t, v)
			err := pruefeLDAPAnmeldung(context.Background(), "konto-1", f.kennung, f.passwort)
			if !errors.Is(err, errAnmeldungAbgelehnt) {
				t.Fatalf("pruefeLDAPAnmeldung(…) = %v, erwartet errAnmeldungAbgelehnt", err)
			}
		})
	}
}

// Eine Kennung mit Zeichen des Filters ändert den Filter nicht: Ohne Maskierung träfe
// „*)(sAMAccountName=*" jeden Eintrag (RFC 4515, 3).
func TestPruefeLDAPAnmeldung_KennungAendertDenFilterNicht(t *testing.T) {
	v := &miniVerzeichnis{
		treffer:     func(_, _ string) []string { return nil },
		passwoerter: map[string]string{testKontoDN: testKontoPasswort},
	}
	starteMiniVerzeichnis(t, v)

	err := pruefeLDAPAnmeldung(context.Background(), "konto-1", "*)(sAMAccountName=*", testKontoPasswort)
	if !errors.Is(err, errAnmeldungAbgelehnt) {
		t.Fatalf("pruefeLDAPAnmeldung(…) = %v, erwartet errAnmeldungAbgelehnt", err)
	}
	_, filter, _ := v.gesehen()
	const want = `(sAMAccountName=\2a\29\28sAMAccountName=\2a)`
	if len(filter) != 1 || filter[0] != want {
		t.Errorf("Filter = %q, erwartet [%s]", filter, want)
	}
}

// Ein leeres Passwort wäre am Verzeichnisdienst eine anonyme Anmeldung, die gelingt; es kommt
// gar nicht erst dort an.
func TestPruefeLDAPAnmeldung_LeeresPasswortErreichtDenServerNicht(t *testing.T) {
	v := &miniVerzeichnis{treffer: einEintrag, passwoerter: map[string]string{testKontoDN: ""}}
	starteMiniVerzeichnis(t, v)

	if err := pruefeLDAPAnmeldung(context.Background(), "konto-1", "mmueller", ""); !errors.Is(err, errAnmeldungAbgelehnt) {
		t.Fatalf("pruefeLDAPAnmeldung(…) = %v, erwartet errAnmeldungAbgelehnt", err)
	}
	if binds, _, _ := v.gesehen(); len(binds) != 0 {
		t.Errorf("Binds = %q, erwartet keine", binds)
	}
}

// Ein Ausfall ist kein falsches Passwort: Die Tür antwortet 503 und zählt keinen Fehlversuch.
func TestPruefeLDAPAnmeldung_AusfallIstKeinFalschesPasswort(t *testing.T) {
	t.Run("Server tot", func(t *testing.T) {
		starteMiniVerzeichnis(t, &miniVerzeichnis{treffer: einEintrag})
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		adresse := l.Addr().String()
		if err := l.Close(); err != nil {
			t.Fatal(err)
		}
		t.Setenv("LDAP_URL", "ldaps://"+adresse)
		pruefeAusfall(t, context.Background())
	})
	t.Run("Dienstkonto abgelehnt", func(t *testing.T) {
		starteMiniVerzeichnis(t, &miniVerzeichnis{treffer: einEintrag})
		t.Setenv("LDAP_BIND_PASSWORT", "veraltet")
		pruefeAusfall(t, context.Background())
	})
	t.Run("Zertifikat ohne LDAP_CA_DATEI nicht vertraut", func(t *testing.T) {
		v := &miniVerzeichnis{treffer: einEintrag}
		starteMiniVerzeichnis(t, v)
		t.Setenv("LDAP_CA_DATEI", "")
		pruefeAusfall(t, context.Background())
		if binds, _, _ := v.gesehen(); len(binds) != 0 {
			t.Errorf("Binds = %q, erwartet keine über eine Verbindung ohne geprüftes Zertifikat", binds)
		}
	})
	t.Run("Server schweigt", func(t *testing.T) {
		starteMiniVerzeichnis(t, &miniVerzeichnis{treffer: einEintrag, schweigt: true})
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		beginn := time.Now()
		pruefeAusfall(t, ctx)
		if dauer := time.Since(beginn); dauer > 3*time.Second {
			t.Errorf("pruefeLDAPAnmeldung kam nach %v zurück, erwartet kurz nach dem Ende des Kontexts", dauer)
		}
	})
	t.Run("LDAP nicht eingerichtet", func(t *testing.T) {
		t.Setenv("LDAP_URL", "")
		pruefeAusfall(t, context.Background())
	})
}

func pruefeAusfall(t *testing.T, ctx context.Context) {
	t.Helper()
	err := pruefeLDAPAnmeldung(ctx, "konto-1", "mmueller", testKontoPasswort)
	if !errors.Is(err, errAnmeldeserverNichtErreichbar) {
		t.Fatalf("pruefeLDAPAnmeldung(…) = %v, erwartet errAnmeldeserverNichtErreichbar", err)
	}
}

// Mit ldap:// geht nichts ohne TLS über die Leitung: StartTLS ist Pflicht, und lehnt der Server
// es ab, wird nicht im Klartext weitergemacht.
func TestPruefeLDAPAnmeldung_StartTLS(t *testing.T) {
	t.Run("angenommen", func(t *testing.T) {
		v := &miniVerzeichnis{treffer: einEintrag, passwoerter: map[string]string{testKontoDN: testKontoPasswort}, mitStartTLS: true}
		starteMiniVerzeichnis(t, v)
		if err := pruefeLDAPAnmeldung(context.Background(), "konto-1", "mmueller", testKontoPasswort); err != nil {
			t.Fatalf("pruefeLDAPAnmeldung(…) = %v, erwartet nil", err)
		}
		if _, _, klartext := v.gesehen(); klartext {
			t.Error("ein Bind kam vor StartTLS an")
		}
	})
	t.Run("abgelehnt", func(t *testing.T) {
		v := &miniVerzeichnis{treffer: einEintrag, passwoerter: map[string]string{testKontoDN: testKontoPasswort}, ohneStartTLS: true}
		starteMiniVerzeichnis(t, v)
		pruefeAusfall(t, context.Background())
		if binds, _, _ := v.gesehen(); len(binds) != 0 {
			t.Errorf("Binds = %q, erwartet keine ohne TLS", binds)
		}
	})
}

// Der Start prüft die Einstellungen, ohne den Server zu fragen: Ein unvollständiges LDAP fällt
// beim Start auf, nicht bei der ersten Anmeldung.
func TestPruefeAnmeldeKonfig_LDAP(t *testing.T) {
	gueltig := map[string]string{
		"APP_ENV": "production", "IMAP_HOST": "", "LDAP_URL": "ldaps://dc.schule.test",
		"LDAP_CA_DATEI": "", "LDAP_BIND_DN": testDienstDN, "LDAP_BIND_PASSWORT": testDienstPasswort,
		"LDAP_BASIS_DN": "dc=schule,dc=test", "LDAP_FILTER": "(|(sAMAccountName=%s)(userPrincipalName=%s))",
	}
	faelle := []struct {
		name       string
		aenderung  map[string]string
		fehlerEnth string
	}{
		{name: "vollständig, ohne Mailserver", aenderung: nil},
		{name: "StartTLS-Adresse", aenderung: map[string]string{"LDAP_URL": "ldap://dc.schule.test:389"}},
		{name: "fremdes Schema", aenderung: map[string]string{"LDAP_URL": "http://dc.schule.test"}, fehlerEnth: "ldaps://"},
		{name: "ohne Dienstkonto", aenderung: map[string]string{"LDAP_BIND_DN": ""}, fehlerEnth: "LDAP_BIND_DN"},
		{name: "ohne Passwort des Dienstkontos", aenderung: map[string]string{"LDAP_BIND_PASSWORT": ""}, fehlerEnth: "LDAP_BIND_PASSWORT"},
		{name: "ohne Suchbasis", aenderung: map[string]string{"LDAP_BASIS_DN": ""}, fehlerEnth: "LDAP_BASIS_DN"},
		{name: "Filter ohne Platzhalter", aenderung: map[string]string{"LDAP_FILTER": "(sAMAccountName=x)"}, fehlerEnth: "%s"},
		{name: "kaputter Filter", aenderung: map[string]string{"LDAP_FILTER": "(sAMAccountName=%s"}, fehlerEnth: "kein gültiger Filter"},
		{name: "CA-Datei fehlt", aenderung: map[string]string{"LDAP_CA_DATEI": "/gibt/es/nicht.pem"}, fehlerEnth: "LDAP_CA_DATEI"},
		{name: "mehrere Suchordner", aenderung: map[string]string{"LDAP_BASIS_DN": "OU=A,DC=schule,DC=test;OU=B,DC=test"}},
		{name: "Suchordner nur Trennzeichen", aenderung: map[string]string{"LDAP_BASIS_DN": " ; "}, fehlerEnth: "keinen Suchordner"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			for name, wert := range gueltig {
				t.Setenv(name, wert)
			}
			for name, wert := range f.aenderung {
				t.Setenv(name, wert)
			}
			err := PruefeAnmeldeKonfig()
			if f.fehlerEnth == "" {
				if err != nil {
					t.Fatalf("PruefeAnmeldeKonfig() = %v, erwartet nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), f.fehlerEnth) {
				t.Fatalf("PruefeAnmeldeKonfig() = %v, erwartet einen Fehler mit %q", err, f.fehlerEnth)
			}
		})
	}
}

// LDAP_BASIS_DN nennt mehrere Suchordner wie die Authentication containers der OPNsense: Gesucht
// wird in jedem, ein Eintrag zählt einmal, auch wenn zwei Ordner ineinander liegen, und ein
// Ordner, den es nicht gibt, sperrt die übrigen nicht aus.
func TestPruefeLDAPAnmeldung_MehrereSuchordner(t *testing.T) {
	const lehrkraefte, extern, fehlt = "ou=lehrkraefte,dc=schule,dc=test", "ou=extern,dc=test", "ou=vertippt,dc=schule,dc=test"
	nurIn := func(ordner string) func(string, string) []string {
		return func(basis, filter string) []string {
			if basis == ordner {
				return einEintrag(basis, filter)
			}
			return nil
		}
	}
	faelle := []struct {
		name    string
		basen   string
		treffer func(string, string) []string
		fehlend map[string]bool
		erfolg  bool
		gesucht []string
	}{
		{name: "Eintrag im zweiten Ordner", basen: lehrkraefte + "; " + extern, treffer: nurIn(extern),
			erfolg: true, gesucht: []string{lehrkraefte, extern}},
		{name: "Ordner ineinander, derselbe Eintrag zweimal", basen: "dc=schule,dc=test;" + lehrkraefte, treffer: einEintrag,
			erfolg: true, gesucht: []string{"dc=schule,dc=test", lehrkraefte}},
		{name: "je Ordner ein anderer Eintrag", basen: lehrkraefte + ";" + extern,
			treffer: func(basis, _ string) []string { return []string{"cn=Muster," + basis} },
			gesucht: []string{lehrkraefte, extern}},
		{name: "erster Ordner fehlt", basen: fehlt + ";" + extern, treffer: nurIn(extern), fehlend: map[string]bool{fehlt: true},
			erfolg: true, gesucht: []string{fehlt, extern}},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			v := &miniVerzeichnis{treffer: f.treffer, fehlendeBasen: f.fehlend,
				passwoerter: map[string]string{testKontoDN: testKontoPasswort}}
			starteMiniVerzeichnis(t, v)
			t.Setenv("LDAP_BASIS_DN", f.basen)

			err := pruefeLDAPAnmeldung(context.Background(), "konto-1", "mmueller", testKontoPasswort)
			if f.erfolg && err != nil {
				t.Fatalf("pruefeLDAPAnmeldung(…) = %v, erwartet nil", err)
			}
			if !f.erfolg && !errors.Is(err, errAnmeldungAbgelehnt) {
				t.Fatalf("pruefeLDAPAnmeldung(…) = %v, erwartet errAnmeldungAbgelehnt", err)
			}
			v.mu.Lock()
			gesucht := append([]string(nil), v.basen...)
			v.mu.Unlock()
			if strings.Join(gesucht, "|") != strings.Join(f.gesucht, "|") {
				t.Errorf("gesucht in %q, erwartet %q", gesucht, f.gesucht)
			}
		})
	}
}

func TestSuchbasen(t *testing.T) {
	faelle := []struct {
		basisDN string
		want    []string
	}{
		{basisDN: "OU=A,DC=x", want: []string{"OU=A,DC=x"}},
		{basisDN: " OU=A,DC=x ; OU=B,DC=x;", want: []string{"OU=A,DC=x", "OU=B,DC=x"}},
		{basisDN: `OU=A\;B,DC=x;OU=C,DC=x`, want: []string{`OU=A\;B,DC=x`, "OU=C,DC=x"}},
		{basisDN: " ; ", want: nil},
	}
	for _, f := range faelle {
		if got := (ldapKonfig{basisDN: f.basisDN}).suchbasen(); strings.Join(got, "|") != strings.Join(f.want, "|") || len(got) != len(f.want) {
			t.Errorf("suchbasen(%q) = %q, erwartet %q", f.basisDN, got, f.want)
		}
	}
}
