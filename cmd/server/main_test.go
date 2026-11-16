package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func setzeVariable(name, wert string) {
	if err := os.Setenv(name, wert); err != nil {
		panic(err)
	}
}

func loescheVariable(name string) {
	if err := os.Unsetenv(name); err != nil {
		panic(err)
	}
}

func setzeGueltigeUmgebung() {
	os.Clearenv()
	setzeVariable("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	setzeVariable("JWT_SECRET", "this-is-a-very-secret-key-that-is-at-least-32-bytes")
	setzeVariable("APP_ENCRYPTION_KEY", "12345678901234567890123456789012")
	setzeVariable("PORT", "8080")
	setzeVariable("ENFORCE_PROD_SECRETS", "false")
	setzeVariable("IMAP_HOST", "imap.example.test")
}

func stelleUmgebungHer(vorher []string) {
	os.Clearenv()
	for _, eintrag := range vorher {
		teile := strings.SplitN(eintrag, "=", 2)
		if len(teile) == 2 {
			setzeVariable(teile[0], teile[1])
		}
	}
}

func TestLeseKonfig_Gueltig(t *testing.T) {
	vorher := os.Environ()
	defer stelleUmgebungHer(vorher)

	setzeGueltigeUmgebung()
	setzeVariable("TRUSTED_PROXIES", "172.16.0.0/12,10.0.0.0/8")
	dsn, jwtGeheimnis, port, server := leseKonfig()

	if dsn != "postgres://user:pass@localhost:5432/db" {
		t.Errorf("dsn = %s, erwartet ist der gesetzte Wert", dsn)
	}
	if jwtGeheimnis != "this-is-a-very-secret-key-that-is-at-least-32-bytes" {
		t.Errorf("jwtGeheimnis = %s, erwartet ist der gesetzte Wert", jwtGeheimnis)
	}
	if port != "8080" {
		t.Errorf("port = %s, erwartet ist der gesetzte Wert", port)
	}
	if !server.UmleitungAufHTTPS {
		t.Errorf("UmleitungAufHTTPS = %v, erwartet ist true", server.UmleitungAufHTTPS)
	}
	if got := strings.Join(server.VertrauteProxys, " "); got != "172.16.0.0/12 10.0.0.0/8" {
		t.Errorf("VertrauteProxys = %q, erwartet die Netze aus TRUSTED_PROXIES", got)
	}
}

// pruefeAbbruch startet den Test als eigenen Prozess und verlangt, dass leseKonfig ihn beendet;
// os.Exit ließe sich im selben Prozess nicht beobachten.
func pruefeAbbruch(t *testing.T, testName string, richteEin func()) {
	t.Helper()
	if os.Getenv("BE_CRASHER") == "1" {
		richteEin()
		leseKonfig()
		return
	}

	befehl := exec.Command(os.Args[0], "-test.run="+testName)
	befehl.Env = append(os.Environ(), "BE_CRASHER=1")
	err := befehl.Run()
	if abbruch, ok := err.(*exec.ExitError); ok && !abbruch.Success() {
		return
	}
	t.Fatalf("der Prozess endete mit %v, erwartet ist ein Exit-Status ungleich 0", err)
}

// Ein unlesbarer Eintrag in TRUSTED_PROXIES verhindert den Start nicht, steht aber im Log. Der
// Test läuft als eigener Prozess, damit er den Logger des Pakets slog nicht umstellen muss.
func TestLeseKonfig_UnlesbarerProxyStehtImLog(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		setzeGueltigeUmgebung()
		setzeVariable("TRUSTED_PROXIES", "172.16.0.0/12,caddy")
		leseKonfig()
		return
	}

	befehl := exec.Command(os.Args[0], "-test.run=TestLeseKonfig_UnlesbarerProxyStehtImLog")
	befehl.Env = append(os.Environ(), "BE_CRASHER=1")
	ausgabe, err := befehl.CombinedOutput()
	if err != nil {
		t.Fatalf("der Prozess endete mit %v, erwartet ist ein Start ohne Abbruch:\n%s", err, ausgabe)
	}
	if !strings.Contains(string(ausgabe), "weder Netz noch Adresse") || !strings.Contains(string(ausgabe), "caddy") {
		t.Errorf("Log = %q, erwartet eine Warnung, die den Eintrag caddy nennt", ausgabe)
	}
}

func TestLeseKonfig_OhneDatabaseURL(t *testing.T) {
	pruefeAbbruch(t, "TestLeseKonfig_OhneDatabaseURL", func() {
		setzeGueltigeUmgebung()
		loescheVariable("DATABASE_URL")
	})
}

func TestLeseKonfig_KurzesJWTGeheimnis(t *testing.T) {
	pruefeAbbruch(t, "TestLeseKonfig_KurzesJWTGeheimnis", func() {
		setzeGueltigeUmgebung()
		setzeVariable("JWT_SECRET", "short")
	})
}

func TestLeseKonfig_UngueltigerSchluessel(t *testing.T) {
	pruefeAbbruch(t, "TestLeseKonfig_UngueltigerSchluessel", func() {
		setzeGueltigeUmgebung()
		setzeVariable("APP_ENCRYPTION_KEY", "invalid")
	})
}

func TestLeseKonfig_OhnePort(t *testing.T) {
	pruefeAbbruch(t, "TestLeseKonfig_OhnePort", func() {
		setzeGueltigeUmgebung()
		loescheVariable("PORT")
	})
}

func TestLeseKonfig_GeheimnisSchutz_BeispielJWT(t *testing.T) {
	pruefeAbbruch(t, "TestLeseKonfig_GeheimnisSchutz_BeispielJWT", func() {
		setzeGueltigeUmgebung()
		setzeVariable("JWT_SECRET", "super-secret-default-key-at-least-32-bytes")
		setzeVariable("ENFORCE_PROD_SECRETS", "true")
	})
}

func TestLeseKonfig_GeheimnisSchutz_BeispielSchluessel(t *testing.T) {
	pruefeAbbruch(t, "TestLeseKonfig_GeheimnisSchutz_BeispielSchluessel", func() {
		setzeGueltigeUmgebung()
		setzeVariable("APP_ENCRYPTION_KEY", "super-secure-aes-key-32-chars-ok")
		setzeVariable("ENFORCE_PROD_SECRETS", "true")
	})
}

// Die Attrappe für IMAP nimmt jedes Passwort für jede in benutzer eingetragene E-Mail an. Stünde
// sie in einer Vorgabe für den Betrieb, wäre die Anmeldung offen, sobald jemand den Server ohne
// eigenes IMAP_HOST startet. Außerhalb von local/development/test muss der Start deshalb
// scheitern, nicht nur warnen.
func TestLeseKonfig_MockIMAPInProduktionVerweigertStart(t *testing.T) {
	pruefeAbbruch(t, "TestLeseKonfig_MockIMAPInProduktionVerweigertStart", func() {
		setzeGueltigeUmgebung()
		setzeVariable("APP_ENV", "production")
		setzeVariable("IMAP_HOST", "mock")
	})
}

// Ohne IMAP_HOST startet der Server nicht. Ein stiller Rückfall auf den Mailserver der Schule
// spräche aus jedem "go run" heraus den echten Dienst an.
func TestLeseKonfig_FehlenderIMAPHostVerweigertStart(t *testing.T) {
	pruefeAbbruch(t, "TestLeseKonfig_FehlenderIMAPHostVerweigertStart", func() {
		setzeGueltigeUmgebung()
		loescheVariable("IMAP_HOST")
	})
}

// Außerhalb von local/development/test verweigert der Server bekannte Beispiel-Geheimnisse auch
// ohne ENFORCE_PROD_SECRETS.
func TestLeseKonfig_ProduktionOhneSchalterVerweigertBeispielJWT(t *testing.T) {
	pruefeAbbruch(t, "TestLeseKonfig_ProduktionOhneSchalterVerweigertBeispielJWT", func() {
		setzeGueltigeUmgebung()
		loescheVariable("ENFORCE_PROD_SECRETS")
		setzeVariable("APP_ENV", "production")
		setzeVariable("JWT_SECRET", "super-secret-default-key-at-least-32-bytes")
	})
}

// Die Spielwiese bleibt bequem: lokal ohne Schalter startet der Server mit den
// Beispielwerten aus docker-compose.local.yml.
func TestLeseKonfig_LokalOhneSchalterStartetMitBeispielJWT(t *testing.T) {
	vorher := os.Environ()
	defer stelleUmgebungHer(vorher)

	setzeGueltigeUmgebung()
	loescheVariable("ENFORCE_PROD_SECRETS")
	setzeVariable("APP_ENV", "local")
	setzeVariable("JWT_SECRET", "super-secret-default-key-at-least-32-bytes")
	if _, jwtGeheimnis, _, _ := leseKonfig(); jwtGeheimnis != "super-secret-default-key-at-least-32-bytes" {
		t.Fatalf("leseKonfig hat den lokalen Beispielwert nicht durchgelassen: %q", jwtGeheimnis)
	}
}

// Ausdrückliches false bleibt der Weg für eine Testphase auf einem production-Server.
func TestLeseKonfig_AusdruecklichFalseLaesstBeispielJWTDurch(t *testing.T) {
	vorher := os.Environ()
	defer stelleUmgebungHer(vorher)

	setzeGueltigeUmgebung()
	setzeVariable("APP_ENV", "production")
	setzeVariable("ENFORCE_PROD_SECRETS", "false")
	setzeVariable("JWT_SECRET", "super-secret-default-key-at-least-32-bytes")
	if _, jwtGeheimnis, _, _ := leseKonfig(); jwtGeheimnis == "" {
		t.Fatal("leseKonfig ist ausgestiegen, obwohl ENFORCE_PROD_SECRETS=false gesetzt ist")
	}
}

// Das Log des Servers schreibt jede Zeile als JSON und zählt dabei die der Stufe ERROR in den
// Fehlerstand, den der Start an den Server gibt.
func TestBaueLog_SchreibtJSONUndZaehltFehler(t *testing.T) {
	var ausgabe bytes.Buffer
	log, fehlerstand := baueLog(&ausgabe)
	log.Debug("unter der Stufe INFO")
	log.Info("Anfrage")
	log.Error("Mail nicht zugestellt", "fehler", "dial tcp: i/o timeout")

	if n := strings.Count(ausgabe.String(), "\n"); n != 2 || !strings.Contains(ausgabe.String(), `"level":"ERROR"`) {
		t.Errorf("Log = %q, erwartet zwei Zeilen JSON ab INFO", ausgabe.String())
	}
	gruppen := fehlerstand.Gruppen(time.Now())
	if len(gruppen) != 1 || gruppen[0].Meldung != "Mail nicht zugestellt" || gruppen[0].Ursache != "dial tcp: i/o timeout" {
		t.Errorf("Fehlerstand = %+v, erwartet die eine Zeile der Stufe ERROR mit Ursache", gruppen)
	}
}
