package mailservice

import (
	"bibliothek/db"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"mime/multipart"
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
}

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

// Konfig liefert die Konfiguration, mit der der nächste Versand ginge.
func (p *Postausgang) Konfig(ctx context.Context) (SMTPKonfig, error) {
	return p.konfig(ctx)
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

	fmt.Fprintf(&buf, "From: %s\r\n", absender)
	fmt.Fprintf(&buf, "To: %s\r\n", an)
	fmt.Fprintf(&buf, "Subject: %s\r\n", betreff)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%s\r\n\r\n", grenze)

	kopfDesTexts := make(textproto.MIMEHeader)
	kopfDesTexts.Set("Content-Type", "text/plain; charset=utf-8")
	teil, err := teile.CreatePart(kopfDesTexts)
	if err != nil {
		return nil, fmt.Errorf("failed to create email body part: %w", err)
	}
	if _, err := teil.Write([]byte(nachricht.Text)); err != nil {
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

// schreibeAnhang hängt eine Datei in Base64 als Teil der Nachricht an. Aus Dateiname und
// Content-Type entfernt sie Zeilenumbrüche und Anführungszeichen: CreatePart schreibt die Werte
// ungeprüft, ein eingeschleuster Umbruch erzeugte weitere Kopfzeilen.
func schreibeAnhang(teile *multipart.Writer, anhang Anhang) error {
	name := strings.NewReplacer("\r", "", "\n", "", `"`, "").Replace(anhang.Name)
	contentType := strings.NewReplacer("\r", "", "\n", "").Replace(anhang.ContentType)
	kopf := make(textproto.MIMEHeader)
	kopf.Set("Content-Type", contentType)
	kopf.Set("Content-Transfer-Encoding", "base64")
	kopf.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	teil, err := teile.CreatePart(kopf)
	if err != nil {
		return fmt.Errorf("failed to create attachment part for %s: %w", anhang.Name, err)
	}

	kodierer := base64.NewEncoder(base64.StdEncoding, teil)
	if _, err := kodierer.Write(anhang.Inhalt); err != nil {
		return fmt.Errorf("failed to write attachment data for %s: %w", anhang.Name, err)
	}
	// Close schreibt die letzten Bytes in Base64; scheitert es, ist der Anhang verdorben.
	if err := kodierer.Close(); err != nil {
		return fmt.Errorf("failed to finalize attachment encoding for %s: %w", anhang.Name, err)
	}
	return nil
}
