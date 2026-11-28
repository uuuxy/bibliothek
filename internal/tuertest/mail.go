package tuertest

import (
	"context"
	"testing"

	"bibliothek/internal/mailservice"
	"bibliothek/internal/smtptest"
)

// FangeMailAb startet einen Testserver und liefert einen Postausgang, der dorthin verschickt,
// mit den abgefangenen Sitzungen. Der Test gibt den Postausgang der Basis seiner Tür (Mail); der
// Kanal nimmt genau eine Sitzung an.
func FangeMailAb(t *testing.T) (*mailservice.Postausgang, <-chan smtptest.Sitzung) {
	t.Helper()
	return FangeMailAbMit(t, smtptest.Normal)
}

// FangeMailAbMit startet wie FangeMailAb einen Testserver, mit dem gewählten Verhalten: für die
// Abläufe, in denen der Server eine Mail abweist.
func FangeMailAbMit(t *testing.T, verhalten smtptest.Verhalten) (*mailservice.Postausgang, <-chan smtptest.Sitzung) {
	t.Helper()
	// Der Testserver kündigt kein STARTTLS an; sichereVerbindung bräche den Versand
	// sonst ab. Dass die Erzwingung wirkt, prüft mailservice.
	t.Setenv("SMTP_ALLOW_PLAINTEXT", "true")
	host, port, sitzungen := smtptest.Starte(t, verhalten)
	post := mailservice.NewPostausgang(func(context.Context) (mailservice.SMTPKonfig, error) {
		return mailservice.SMTPKonfig{Host: host, Port: port, Absender: "bibliothek@schule.invalid"}, nil
	})
	return post, sitzungen
}
