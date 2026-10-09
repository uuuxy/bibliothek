package service

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Die Bestellung bleibt versandfähig, was auch immer mit der gespeicherten Vorlage ist: Gilt sie
// nicht ganz, gilt der Vorgabetext ganz. Am echten Postgres, weil das Zusammenspiel aus
// gespeicherter Zeile und Rückfall geprüft wird; der Editor lässt ein leeres Feld durch.
func TestBestellVorlage_RueckfallAufDenVorgabetext(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	var urBetreff, urText string
	hatte := true
	if err := pool.QueryRow(ctx,
		`SELECT betreff, text_body FROM mail_vorlagen WHERE typ = 'BESTELLUNG_HAENDLER'`).Scan(&urBetreff, &urText); err != nil {
		hatte = false
	}
	setze := func(betreff, text string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO mail_vorlagen (typ, betreff, text_body) VALUES ('BESTELLUNG_HAENDLER', $1, $2)
			ON CONFLICT (typ) DO UPDATE SET betreff = EXCLUDED.betreff, text_body = EXCLUDED.text_body`,
			betreff, text); err != nil {
			t.Fatalf("Vorlage schreiben: %v", err)
		}
	}
	entferne := func() {
		t.Helper()
		if _, err := pool.Exec(ctx, `DELETE FROM mail_vorlagen WHERE typ = 'BESTELLUNG_HAENDLER'`); err != nil {
			t.Fatalf("Vorlage entfernen: %v", err)
		}
	}
	t.Cleanup(func() {
		if hatte {
			setze(urBetreff, urText)
			return
		}
		entferne()
	})
	pruefe := func(fall, willBetreff, willText string) {
		t.Helper()
		betreff, text := BestellVorlage(ctx, pool)
		if betreff != willBetreff || text != willText {
			t.Errorf("%s: Betreff %q und Text %q, erwartet %q und %q", fall, betreff, text, willBetreff, willText)
		}
	}

	setze("Eigener Betreff", "Eigener Text")
	pruefe("gespeicherte Vorlage", "Eigener Betreff", "Eigener Text")

	setze("", "Eigener Text")
	pruefe("leerer Betreff", bestellMailVorgabeBetreff, bestellMailVorgabeText)

	setze("Eigener Betreff", "")
	pruefe("leerer Text", bestellMailVorgabeBetreff, bestellMailVorgabeText)

	entferne()
	pruefe("keine Vorlage", bestellMailVorgabeBetreff, bestellMailVorgabeText)

	setze("Eigener Betreff", "Eigener Text")
	abgebrochen, abbruch := context.WithCancel(ctx)
	abbruch()
	if betreff, text := BestellVorlage(abgebrochen, pool); betreff != bestellMailVorgabeBetreff || text != bestellMailVorgabeText {
		t.Errorf("Lesefehler: Betreff %q und Text %q, erwartet den Vorgabetext", betreff, text)
	}
}
