package tuertest

import (
	"testing"

	"bibliothek/internal/smtptest"
	"bibliothek/mailservice"
)

// FangeMailAb biegt den Versand auf einen Testserver um und liefert die abgefangene
// Nachricht. Der Kanal nimmt genau eine Sitzung an.
func FangeMailAb(t *testing.T) <-chan smtptest.Sitzung {
	t.Helper()
	// Der Testserver kündigt kein STARTTLS an; sichereVerbindung bräche den Versand
	// sonst ab. Dass die Erzwingung wirkt, prüft mailservice.
	t.Setenv("SMTP_ALLOW_PLAINTEXT", "true")
	host, port, sitzungen := smtptest.Starte(t, smtptest.Normal)

	t.Cleanup(mailservice.TauscheKonfigLaderFuerTest(func() (mailservice.SMTPKonfig, error) {
		return mailservice.SMTPKonfig{Host: host, Port: port, Absender: "bibliothek@schule.invalid"}, nil
	}))

	return sitzungen
}
