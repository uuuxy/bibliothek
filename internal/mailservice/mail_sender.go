package mailservice

import (
	"bibliothek/internal/db"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
	"time"
)

// Anhang ist eine Datei, die an der Mail hängt.
type Anhang struct {
	Name        string
	ContentType string
	Inhalt      []byte
}

// Nachricht ist eine Mail, die verschickt werden soll: Empfänger, Betreff, Text und Anhänge.
type Nachricht struct {
	An       string
	Betreff  string
	Text     string
	Anhaenge []Anhang
	// Art sagt, welche Mail das ist. Über welchen Absender sie hinausgeht, legen die
	// Einstellungen fest (SMTPKonfig.postfachFuer); ohne Art geht sie über den Postausgang.
	Art Mailart
}

// Mailart ist eine Art von Mail, die sich über den zweiten Absender schicken lässt. Die Werte
// stehen so in mail_settings_config.portal_absender_fuer, dessen Bedingung sie aufzählt.
type Mailart string

const (
	// MailartPortal sind die Mails an Lehrkräfte aus Mein Portal: Anliegen erledigt, Klassensatz
	// bereit.
	MailartPortal Mailart = "portal"
	// MailartMahnliste sind die Mahnlisten an die Klassenleitungen.
	MailartMahnliste Mailart = "mahnliste"
	// MailartAbgaenger sind die Kontoauszüge der Abgänger an die Klassenleitungen.
	MailartAbgaenger Mailart = "abgaenger"
	// MailartBestellung ist die Bestellmail an den Händler.
	MailartBestellung Mailart = "bestellung"
)

// Mailarten sind alle Arten in der Reihenfolge, in der die Einstellungen sie zeigen.
var Mailarten = []Mailart{MailartPortal, MailartMahnliste, MailartAbgaenger, MailartBestellung}

// Postfach sagt, unter welchem Absender eine Mail hinausgeht.
type Postfach int

const (
	// PostfachAllgemein ist der Postausgang der Einstellungen. Ihn nimmt jede Mail, deren Art die
	// Einstellungen nicht dem zweiten Absender geben.
	PostfachAllgemein Postfach = iota
	// PostfachPortal ist der zweite Absender (Spalten portal_*). Ohne eigene Adresse in den
	// Einstellungen gehen auch seine Mails über den Postausgang.
	PostfachPortal
)

// Postausgang verschickt Mails über SMTP. Die Konfiguration liest er vor jedem Versand neu, damit
// eine Änderung in der Oberfläche ohne Neustart gilt. Der Server legt einen an und gibt ihn den
// Türen; ein Test legt einen gegen seinen Testserver an.
type Postausgang struct {
	konfig func(context.Context) (SMTPKonfig, error)
}

// NewPostausgang verschickt mit der Konfiguration, die konfig vor jedem Versand liefert: im
// Betrieb KonfigAusDatenbank, in einem Werkzeug ohne Datenbank die Umgebung.
func NewPostausgang(konfig func(context.Context) (SMTPKonfig, error)) *Postausgang {
	return &Postausgang{konfig: konfig}
}

// KonfigAusDatenbank liefert die in der Oberfläche gespeicherte Konfiguration (ladeSMTPKonfig),
// damit der Test-Knopf und jeder echte Versand dieselbe Quelle nutzen.
func KonfigAusDatenbank(pool db.Pool) func(context.Context) (SMTPKonfig, error) {
	return func(ctx context.Context) (SMTPKonfig, error) {
		// Kurze Frist: Ein hängendes Lesen der Konfiguration darf den Versand vieler Mails nicht
		// aufhalten. Es erbt die Werte der Anfrage, nicht ihr Ende: Ein Sammelversand läuft zu
		// Ende wie der Versand selbst, den SMTP mit eigenen Fristen begrenzt.
		ctx, abbruch := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer abbruch()
		return ladeSMTPKonfig(ctx, pool)
	}
}

// IstKonfiguriert meldet, ob ein Versand versucht werden kann. Die Türen fragen damit die
// Konfiguration ab, die wirklich benutzt wird, statt os.Getenv("SMTP_HOST"); sonst meldeten sie
// „SMTP nicht konfiguriert", während in der Oberfläche ein Server steht, und umgekehrt.
func (p *Postausgang) IstKonfiguriert(ctx context.Context) bool {
	konfig, err := p.konfig(ctx)
	if err != nil {
		// Der Fehler steht im Log: ladeSMTPKonfig erzeugt ihn, damit er auffällt („Ein
		// unlesbares Passwort heißt, dass der APP_ENCRYPTION_KEY nicht mehr derselbe ist").
		// Ohne die Zeile meldete der Mahnlauf nach einem Wechsel des Schlüssels nur „SMTP nicht
		// konfiguriert", während Host, Port und Benutzer stimmen, und der Admin suchte an der
		// falschen Stelle.
		slog.Error("SMTP-Konfiguration nicht lesbar — Versand wird als „nicht konfiguriert\" behandelt",
			"fehler", err)
		return false
	}
	return konfig.istKonfiguriert()
}

// Sende verschickt eine Mail mit Text und Anhängen über SMTP, mit der Konfiguration des
// Postausgangs.
func (p *Postausgang) Sende(ctx context.Context, nachricht Nachricht) error {
	konfig, err := p.konfig(ctx)
	if err != nil {
		return err
	}
	if !konfig.istKonfiguriert() {
		return fmt.Errorf("%w: kein SMTP-Server hinterlegt", ErrMailNichtKonfiguriert)
	}
	konfig, err = konfig.fuer(konfig.postfachFuer(nachricht.Art))
	if err != nil {
		return err
	}

	absender := konfig.Absender

	gelesenerAbsender, err := mail.ParseAddress(absender)
	if err != nil {
		return fmt.Errorf("invalid sender email address: %w", err)
	}
	absender = gelesenerAbsender.Address

	gelesenerEmpfaenger, err := mail.ParseAddress(nachricht.An)
	if err != nil {
		return fmt.Errorf("invalid recipient email address: %w", err)
	}
	nachricht.An = gelesenerEmpfaenger.Address

	// Ein Zeilenumbruch im Betreff könnte weitere Kopfzeilen anhängen.
	nachricht.Betreff = strings.ReplaceAll(nachricht.Betreff, "\r", "")
	nachricht.Betreff = strings.ReplaceAll(nachricht.Betreff, "\n", "")

	rohtext, err := baueMailNachricht(nachricht, absender)
	if err != nil {
		return err
	}

	return versendeUeberSMTP(konfig, absender, []string{nachricht.An}, rohtext)
}

// baueMailNachricht setzt die ganze Nachricht im Format MIME zusammen: Kopfzeilen, Text und
// Anhänge. Die Kopfzeilen prüft pruefeKopfzeilen hier, wo sie entstehen, für jeden Aufrufer.
func baueMailNachricht(nachricht Nachricht, absender string) ([]byte, error) {
	absender, an, betreff, err := pruefeKopfzeilen(absender, nachricht.An, nachricht.Betreff)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	teile := multipart.NewWriter(&buf)
	grenze := teile.Boundary()

	schreibeKopf(&buf, absender, an, betreff)
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%s\r\n\r\n", grenze)

	kopfDesTexts := make(textproto.MIMEHeader)
	kopfDesTexts.Set("Content-Type", "text/plain; charset=utf-8")
	kopfDesTexts.Set("Content-Transfer-Encoding", "quoted-printable")
	teil, err := teile.CreatePart(kopfDesTexts)
	if err != nil {
		return nil, fmt.Errorf("failed to create email body part: %w", err)
	}
	if err := schreibeQuotedPrintable(teil, nachricht.Text); err != nil {
		return nil, fmt.Errorf("failed to write email body: %w", err)
	}

	for _, anhang := range nachricht.Anhaenge {
		if err := schreibeAnhang(teile, anhang); err != nil {
			return nil, err
		}
	}

	// Close schreibt die schließende Grenze; scheitert es, ist die Nachricht unvollständig.
	if err := teile.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize multipart message: %w", err)
	}

	return buf.Bytes(), nil
}

// base64Zeilenlaenge ist die längste Zeile in Base64 nach RFC 2045 (6.8). Ohne Umbruch stünde ein
// Anhang als eine Zeile da, weit über den 998 Zeichen, die RFC 5321 einer Zeile erlaubt, und ein
// strenger Mailserver wiese die Mail ab oder bräche die Daten selbst um.
const base64Zeilenlaenge = 76

// zeilenUmbruch schreibt nach je base64Zeilenlaenge Zeichen ein CRLF.
type zeilenUmbruch struct {
	ziel   io.Writer
	spalte int
}

func (z *zeilenUmbruch) Write(p []byte) (int, error) {
	geschrieben := 0
	for len(p) > 0 {
		if z.spalte == base64Zeilenlaenge {
			if _, err := z.ziel.Write([]byte("\r\n")); err != nil {
				return geschrieben, err
			}
			z.spalte = 0
		}
		stueck := min(len(p), base64Zeilenlaenge-z.spalte)
		n, err := z.ziel.Write(p[:stueck])
		geschrieben += n
		z.spalte += n
		if err != nil {
			return geschrieben, err
		}
		p = p[stueck:]
	}
	return geschrieben, nil
}

// schreibeQuotedPrintable schreibt einen Text in quoted-printable: Umlaute als ASCII und Zeilen
// höchstens 76 Zeichen lang, auch wenn ein Absatz einer Vorlage keinen Umbruch hat.
func schreibeQuotedPrintable(ziel io.Writer, text string) error {
	kodierer := quotedprintable.NewWriter(ziel)
	if _, err := kodierer.Write([]byte(text)); err != nil {
		return err
	}
	return kodierer.Close()
}

// schreibeAnhang hängt eine Datei in Base64 als Teil der Nachricht an. Aus Dateiname und
// Content-Type entfernt sie Zeilenumbrüche und Anführungszeichen: CreatePart schreibt die Werte
// ungeprüft, ein eingeschleuster Umbruch erzeugte weitere Kopfzeilen. Einen Dateinamen mit Umlaut
// (ein Kontoauszug trägt den Namen des Kindes) kodiert mime.FormatMediaType nach RFC 2231.
func schreibeAnhang(teile *multipart.Writer, anhang Anhang) error {
	name := strings.NewReplacer("\r", "", "\n", "", `"`, "").Replace(anhang.Name)
	contentType := strings.NewReplacer("\r", "", "\n", "").Replace(anhang.ContentType)
	kopf := make(textproto.MIMEHeader)
	kopf.Set("Content-Type", contentType)
	kopf.Set("Content-Transfer-Encoding", "base64")
	kopf.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	teil, err := teile.CreatePart(kopf)
	if err != nil {
		return fmt.Errorf("failed to create attachment part for %s: %w", anhang.Name, err)
	}

	kodierer := base64.NewEncoder(base64.StdEncoding, &zeilenUmbruch{ziel: teil})
	if _, err := kodierer.Write(anhang.Inhalt); err != nil {
		return fmt.Errorf("failed to write attachment data for %s: %w", anhang.Name, err)
	}
	// Close schreibt die letzten Bytes in Base64; scheitert es, ist der Anhang verdorben.
	if err := kodierer.Close(); err != nil {
		return fmt.Errorf("failed to finalize attachment encoding for %s: %w", anhang.Name, err)
	}
	return nil
}
