package bestellung_test

import (
	"context"
	"testing"
	"time"

	"bibliothek/internal/mailservice"
	"bibliothek/internal/testdaten"
)

// postfachZeuge nimmt jede Mail an und meldet ihre Art; den Absender wählt der Postausgang danach.
type postfachZeuge struct {
	postfaecher chan<- mailservice.Mailart
}

func (z postfachZeuge) Sende(_ context.Context, m mailservice.Nachricht) error {
	z.postfaecher <- m.Art
	return nil
}

func (postfachZeuge) IstKonfiguriert(context.Context) bool { return true }

// Die Bestellmail an den Händler trägt die Art Bestellung: Über den zweiten Absender geht sie nur,
// wenn die Einstellungen ihn dafür anhaken (mailservice.SMTPKonfig.postfachFuer).
func TestBestellmail_TraegtDieArtBestellung(t *testing.T) {
	l := bestellmailLageAnlegen(t, "postfach")
	postfaecher := make(chan mailservice.Mailart, 4)
	l.srv.Mail = postfachZeuge{postfaecher: postfaecher}

	lieferant := testdaten.LegeLieferantAn(t, l.pool, "Buchhandlung-Postfach", false)
	l.bestelle(t, lieferant, testdaten.LegeTitelAn(t, l.pool, "LMF-Mathe-Postfach"))

	select {
	case p := <-postfaecher:
		if p != mailservice.MailartBestellung {
			t.Errorf("Bestellmail mit Art %q, erwartet %q", p, mailservice.MailartBestellung)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("keine Bestellmail verschickt")
	}
}
