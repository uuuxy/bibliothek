package bestellung_test

import (
	"context"
	"testing"
	"time"

	"bibliothek/internal/testdaten"
	"bibliothek/mailservice"
)

// postfachZeuge nimmt jede Mail an und meldet, unter welchem Postfach sie hinausgehen sollte.
type postfachZeuge struct {
	postfaecher chan<- mailservice.Postfach
}

func (z postfachZeuge) Sende(_ context.Context, m mailservice.Nachricht) error {
	z.postfaecher <- m.Postfach
	return nil
}

func (postfachZeuge) IstKonfiguriert(context.Context) bool { return true }

// Die Bestellmail an den Händler geht über den Postausgang der Einstellungen. Den Absender für
// Mein Portal nehmen nur die Mails an Lehrkräfte; eine Bestellung darf nie unter ihm hinausgehen.
func TestBestellmail_GehtUeberDenPostausgang(t *testing.T) {
	l := bestellmailLageAnlegen(t, "postfach")
	postfaecher := make(chan mailservice.Postfach, 4)
	l.srv.Mail = postfachZeuge{postfaecher: postfaecher}

	lieferant := testdaten.LegeLieferantAn(t, l.pool, "Buchhandlung-Postfach", false)
	l.bestelle(t, lieferant, testdaten.LegeTitelAn(t, l.pool, "LMF-Mathe-Postfach"))

	select {
	case p := <-postfaecher:
		if p != mailservice.PostfachAllgemein {
			t.Errorf("Bestellmail unter Postfach %d, erwartet den Postausgang (PostfachAllgemein)", p)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("keine Bestellmail verschickt")
	}
}
