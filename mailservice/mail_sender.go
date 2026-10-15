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

// Sende verschickt eine Mail. Sie ist eine Variable und keine Funktion, damit Tests den Versand
// ersetzen können, etwa um zu prüfen, dass ein Fehler beim Versand die Transaktion einer
// Bestellung zurückrollt und keine Bestellung ohne Mail stehen bleibt.
var Sende = sendeNachricht

// smtpKonfigLader liefert die Konfiguration für den nächsten Versand.
//
// Eine Variable, damit NewServer sie an die Datenbank binden kann: Sende(Nachricht) hat weder
// Kontext noch Pool zur Hand. Die Vorgabe ist die Umgebung, damit Tests und Werkzeuge ohne
// Datenbank versenden können.
var smtpKonfigLader = func() (SMTPKonfig, error) {
	return KonfigAusUmgebung(), nil
}

// BindeSMTPKonfigAnDatenbank lässt jeden Versand die in der Oberfläche gespeicherte
// Konfiguration benutzen. Der Start ruft sie einmal (NewServer), bevor die erste Anfrage läuft.
func BindeSMTPKonfigAnDatenbank(pool db.Pool) {
	smtpKonfigLader = func() (SMTPKonfig, error) {
		// Eigener Kontext mit kurzer Frist: Der Aufrufer hat keinen, und ein hängendes Lesen
		// der Konfiguration darf den Versand vieler Mails nicht aufhalten.
		ctx, abbruch := context.WithTimeout(context.Background(), 5*time.Second)
		defer abbruch()
		return LadeSMTPKonfig(ctx, pool)
	}
}

// VersandKonfig liefert die Konfiguration, mit der der nächste Versand ginge.
func VersandKonfig() (SMTPKonfig, error) {
	return smtpKonfigLader()
}

// TauscheKonfigLaderFuerTest lässt jeden Versand die Konfiguration aus lader nehmen: Ein Test
// stellt so einen Mailserver ein, einen fehlenden oder eine unlesbare Konfiguration. Der
// Rückgabewert setzt den vorigen Lader wieder ein.
func TauscheKonfigLaderFuerTest(lader func() (SMTPKonfig, error)) (zurueck func()) {
	alt := smtpKonfigLader
	smtpKonfigLader = lader
	return func() { smtpKonfigLader = alt }
}

// IstSMTPKonfiguriert meldet, ob ein Versand versucht werden kann. Die Türen fragen damit die
// Konfiguration ab, die wirklich benutzt wird, statt os.Getenv("SMTP_HOST"); sonst meldeten sie
// „SMTP nicht konfiguriert", während in der Oberfläche ein Server steht, und umgekehrt.
func IstSMTPKonfiguriert() bool {
	konfig, err := smtpKonfigLader()
	if err != nil {
		// Der Fehler steht im Log: LadeSMTPKonfig erzeugt ihn, damit er auffällt („Ein
		// unlesbares Passwort heißt, dass der APP_ENCRYPTION_KEY nicht mehr derselbe ist").
		// Ohne die Zeile meldete der Mahnlauf nach einem Wechsel des Schlüssels nur „SMTP nicht
		// konfiguriert", während Host, Port und Benutzer stimmen, und der Admin suchte an der
		// falschen Stelle.
		slog.Error("SMTP-Konfiguration nicht lesbar — Versand wird als „nicht konfiguriert\" behandelt",
			"fehler", err)
		return false
	}
	return konfig.IstKonfiguriert()
}

// sendeNachricht verschickt eine Mail mit Text und Anhängen über SMTP. Die Zugangsdaten kommen
// aus der gespeicherten Konfiguration (smtpKonfigLader).
func sendeNachricht(nachricht Nachricht) error {
	konfig, err := smtpKonfigLader()
	if err != nil {
		return err
	}
	if !konfig.IstKonfiguriert() {
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

	return VersendeUeberSMTP(konfig, absender, []string{nachricht.An}, rohtext)
}

// baueMailNachricht setzt die ganze Nachricht im Format MIME zusammen: Kopfzeilen, Text und
// Anhänge. Die Kopfzeilen prüft PruefeKopfzeilen hier, wo sie entstehen, für jeden Aufrufer.
func baueMailNachricht(nachricht Nachricht, absender string) ([]byte, error) {
	absender, an, betreff, err := PruefeKopfzeilen(absender, nachricht.An, nachricht.Betreff)
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
