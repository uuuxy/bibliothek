// Package adressabgleich holt das globale Adressbuch aus dem Webmail der Schule (Roundcube) und
// berechnet, welches Konto mit Platzhalter-Adresse welche Adresse daraus bekommt.
//
// Das Webmail hat für das Adressbuch keine Schnittstelle; der Abruf meldet sich deshalb an wie ein
// Browser und lädt den Export, den Roundcube unter Kontakte → Exportieren anbietet. Im Webmail
// wird nur gelesen.
package adressabgleich

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Zugang sind die Angaben zur Anmeldung am Webmail.
type Zugang struct {
	Adresse  string
	Benutzer string
	Passwort string
}

// Fehler des Abrufs, die die Oberfläche beim Namen nennt.
var (
	// ErrAnmeldungAbgelehnt: Das Webmail zeigt nach der Anmeldung wieder das Formular, etwa weil
	// das Passwort geändert wurde.
	ErrAnmeldungAbgelehnt = errors.New("das Webmail hat die Anmeldung abgelehnt")
	// ErrKeinGlobalesAdressbuch: Das Konto sieht kein Adressbuch mit der Kennung „global".
	ErrKeinGlobalesAdressbuch = errors.New("das Webmail zeigt kein globales Adressbuch")
	// ErrAdresseOhneHTTPS: Unverschlüsselt ginge das Passwort im Klartext durchs Netz.
	ErrAdresseOhneHTTPS = errors.New("die Adresse des Webmails muss mit https:// beginnen")
)

const (
	// quelleGlobal ist die Kennung des globalen Adressbuchs in Roundcube.
	quelleGlobal = "global"
	// hoechstGroesse begrenzt jede Antwort des Webmails; das Adressbuch einer Schule mit einigen
	// hundert Einträgen hat wenige hundert Kilobyte.
	hoechstGroesse = 5 << 20
	wartezeit      = 30 * time.Second
	// gesamtfrist begrenzt den ganzen Abruf aus fünf Anfragen. Die Maske wartet etwas länger
	// (frontend/src/lib/components/admin/AdressabgleichEinstellungen.svelte), damit sie die
	// Antwort noch sieht.
	gesamtfrist = 2 * time.Minute
)

var (
	formularToken = regexp.MustCompile(`<input[^>]*name="_token"[^>]*>`)
	wertAttribut  = regexp.MustCompile(`value="([^"]*)"`)
	anfrageToken  = regexp.MustCompile(`"request_token":"([^"]*)"`)
	// Die Liste der Adressbücher in den Daten der Seite: in Roundcube 1.7 ein Array, davor ein
	// Objekt mit der Kennung als Schlüssel. Jedes Adressbuch ist ein Objekt ohne Verschachtelung.
	adressbuecher = regexp.MustCompile(`"address_sources":(\[(\{[^{}]*\},?)*\]|\{("[^"]*":\{[^{}]*\},?)*\})`)
	adressbuchID  = regexp.MustCompile(`\{[^{}]*"id":"([^"]*)"[^{}]*\}`)
)

// HoleGlobalesAdressbuch meldet sich am Webmail an, lädt das globale Adressbuch als vCard und
// meldet sich wieder ab.
func HoleGlobalesAdressbuch(ctx context.Context, zugang Zugang) ([]byte, error) {
	return holeAdressbuch(ctx, zugang, quelleGlobal)
}

// holeAdressbuch lädt das Adressbuch mit der Kennung quelle; der Test am echten Roundcube nimmt das
// persönliche („0"), weil ein frisches Roundcube kein globales hat.
func holeAdressbuch(ctx context.Context, zugang Zugang, quelle string) ([]byte, error) {
	ctx, abbrechen := context.WithTimeout(ctx, gesamtfrist)
	defer abbrechen()
	basis, err := Grundadresse(zugang.Adresse)
	if err != nil {
		return nil, err
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	s := &sitzung{client: &http.Client{Jar: jar, Timeout: wartezeit}, basis: basis, quelle: quelle}

	if err := s.meldeAn(ctx, zugang.Benutzer, zugang.Passwort); err != nil {
		return nil, err
	}
	token, err := s.anfrageTokenMitAdressbuch(ctx)
	if err != nil {
		return nil, err
	}
	defer s.meldeAb(ctx, token)
	return s.exportiere(ctx, token)
}

// Grundadresse macht aus der Adresse, wie sie in der Adresszeile des Browsers steht, die
// Grundadresse des Webmails: ohne „?_task=…" und „index.php", ohne Schrägstrich am Ende, ohne
// Angabe mit https. Unverschlüsselt geht das Passwort nur an den eigenen Rechner.
func Grundadresse(adresse string) (string, error) {
	a := strings.TrimSpace(adresse)
	if a == "" {
		return "", errors.New("die Adresse des Webmails fehlt")
	}
	if !strings.Contains(a, "://") {
		a = "https://" + a
	}
	u, err := url.Parse(a)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("die Adresse des Webmails %q ist keine gültige Adresse", adresse)
	}
	if u.Scheme != "https" && (u.Scheme != "http" || !istEigenerRechner(u.Hostname())) {
		return "", ErrAdresseOhneHTTPS
	}
	u.RawQuery, u.Fragment = "", ""
	u.Path = strings.TrimSuffix(strings.TrimSuffix(u.Path, "/"), "index.php")
	return strings.TrimSuffix(u.String(), "/"), nil
}

func istEigenerRechner(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

type sitzung struct {
	client *http.Client
	basis  string
	quelle string
}

// meldeAn holt das Formular samt Token und schickt die Anmeldung. Ob sie gelang, zeigt erst die
// nächste Seite: Roundcube antwortet auf eine abgelehnte Anmeldung wieder mit dem Formular.
func (s *sitzung) meldeAn(ctx context.Context, benutzer, passwort string) error {
	seite, ziel, err := s.hole(ctx, http.MethodGet, s.basis+"/?_task=login", nil)
	if err != nil {
		return fmt.Errorf("die Anmeldeseite des Webmails lädt nicht: %w", err)
	}
	// Eine Weiterleitung (auf https oder auf /roundcube/) gilt für alles Weitere.
	if basis, err := Grundadresse(ziel); err == nil {
		s.basis = basis
	}
	feld := formularToken.Find(seite)
	wert := wertAttribut.FindSubmatch(feld)
	if wert == nil {
		return fmt.Errorf("unter %s/?_task=login steht keine Anmeldeseite von Roundcube", s.basis)
	}
	formular := url.Values{
		"_token": {string(wert[1])}, "_task": {"login"}, "_action": {"login"},
		"_timezone": {"Europe/Berlin"}, "_url": {""}, "_user": {benutzer}, "_pass": {passwort},
	}
	if _, _, err := s.hole(ctx, http.MethodPost, s.basis+"/?_task=login", formular); err != nil {
		return fmt.Errorf("die Anmeldung am Webmail scheiterte: %w", err)
	}
	return nil
}

// anfrageTokenMitAdressbuch liest auf der Seite der Kontakte das Token, das Roundcube für den
// Export verlangt, und prüft, dass das Adressbuch dabei ist.
func (s *sitzung) anfrageTokenMitAdressbuch(ctx context.Context) (string, error) {
	seite, _, err := s.hole(ctx, http.MethodGet, s.basis+"/?_task=addressbook", nil)
	if err != nil {
		return "", fmt.Errorf("die Kontakte im Webmail laden nicht: %w", err)
	}
	if bytes.Contains(seite, []byte(`name="_pass"`)) {
		return "", ErrAnmeldungAbgelehnt
	}
	token := anfrageToken.FindSubmatch(seite)
	if token == nil {
		return "", errors.New("die Kontakte im Webmail tragen kein Anfrage-Token")
	}
	for _, buch := range adressbuchID.FindAllSubmatch(adressbuecher.Find(seite), -1) {
		if string(buch[1]) == s.quelle {
			return string(token[1]), nil
		}
	}
	return "", ErrKeinGlobalesAdressbuch
}

func (s *sitzung) exportiere(ctx context.Context, token string) ([]byte, error) {
	adresse := s.basis + "/?" + url.Values{
		"_task": {"addressbook"}, "_action": {"export"}, "_source": {s.quelle}, "_token": {token},
	}.Encode()
	vcf, _, err := s.hole(ctx, http.MethodGet, adresse, nil)
	if err != nil {
		return nil, fmt.Errorf("der Export des Adressbuchs scheiterte: %w", err)
	}
	if !bytes.Contains(vcf, []byte("BEGIN:VCARD")) && len(bytes.TrimSpace(vcf)) > 0 {
		return nil, errors.New("der Export des Adressbuchs enthält keine vCards")
	}
	return vcf, nil
}

// meldeAb beendet die Sitzung; scheitert das, läuft sie im Webmail von selbst ab.
func (s *sitzung) meldeAb(ctx context.Context, token string) {
	if _, _, err := s.hole(ctx, http.MethodGet, s.basis+"/?_task=logout&_token="+url.QueryEscape(token), nil); err != nil {
		slog.Warn("Adressabgleich: Abmelden am Webmail gescheitert, die Sitzung läuft dort ab", "fehler", err)
	}
}

// hole schickt eine Anfrage und liefert den Inhalt der Antwort und die Adresse nach allen
// Weiterleitungen. Roundcube meldet eine abgelehnte Anmeldung mit 401 und dem Formular; das
// wertet der Rufer am Inhalt aus, deshalb zählt 401 hier nicht als Fehler.
func (s *sitzung) hole(ctx context.Context, methode, adresse string, formular url.Values) ([]byte, string, error) {
	var inhalt io.Reader
	if formular != nil {
		inhalt = strings.NewReader(formular.Encode())
	}
	anfrage, err := http.NewRequestWithContext(ctx, methode, adresse, inhalt)
	if err != nil {
		return nil, "", err
	}
	anfrage.Header.Set("User-Agent", "Bibliothek-Adressabgleich")
	anfrage.Header.Set("Referer", s.basis+"/?_task=addressbook")
	if formular != nil {
		anfrage.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	antwort, err := s.client.Do(anfrage)
	if err != nil {
		// Die Adresse trägt das Anfrage-Token; im Fehler, der ins Log geht, steht sie ohne Anfrage.
		var adressFehler *url.Error
		if errors.As(err, &adressFehler) {
			return nil, "", fmt.Errorf("%s %s: %w", methode, s.basis, adressFehler.Err)
		}
		return nil, "", err
	}
	defer func() {
		if err := antwort.Body.Close(); err != nil {
			slog.Warn("Adressabgleich: Antwort des Webmails nicht geschlossen", "fehler", err)
		}
	}()
	if antwort.StatusCode >= 400 && antwort.StatusCode != http.StatusUnauthorized {
		return nil, "", fmt.Errorf("das Webmail antwortet mit HTTP %d", antwort.StatusCode)
	}
	daten, err := io.ReadAll(io.LimitReader(antwort.Body, hoechstGroesse+1))
	if err != nil {
		return nil, "", err
	}
	if len(daten) > hoechstGroesse {
		return nil, "", fmt.Errorf("die Antwort des Webmails ist größer als %d MB", hoechstGroesse>>20)
	}
	return daten, antwort.Request.URL.String(), nil
}
