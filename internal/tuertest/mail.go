package tuertest

import (
	"context"
	"testing"

	"bibliothek/internal/smtptest"
	"bibliothek/mailservice"
)

// FangeMailAb startet einen Testserver und liefert einen Postausgang, der dorthin verschickt,
// mit den abgefangenen Sitzungen. Der Test gibt den Postausgang der Basis seiner Tür (Mail); der
// Kanal nimmt genau eine Sitzung an.
func FangeMailAb(t *testing.T) (*mailservice.Postausgang, <-chan smtptest.Sitzung) {
	t.Helper()
	// Der Testserver kündigt kein STARTTLS an; sichereVerbindung bräche den Versand
	// sonst ab. Dass die Erzwingung wirkt, prüft mailservice.
	t.Setenv("SMTP_ALLOW_PLAINTEXT", "true")
	host, port, sitzungen := smtptest.Starte(t, smtptest.Normal)
	post := mailservice.NewPostausgang(func(context.Context) (mailservice.SMTPKonfig, error) {
		return mailservice.SMTPKonfig{Host: host, Port: port, Absender: "bibliothek@schule.invalid"}, nil
	})
	return post, sitzungen
}
