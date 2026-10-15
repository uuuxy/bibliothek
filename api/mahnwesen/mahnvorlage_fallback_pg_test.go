package mahnwesen

// Eine leer gespeicherte Vorlage darf keinen leeren Brief ergeben: ladeMahnVorlageText fällt
// auch dann auf den Vorgabetext zurück, nicht nur bei einem Fehler der Datenbank. Der Editor
// lässt '' durch (die Spalten sind NOT NULL, aber '' ist erlaubt), und es entstünden
// Mahnbriefe ohne Betreff und ohne Anschreiben. Die Vorlage der Bestellung
// (service.LadeBestellMailVorlage) prüft ebenso auf leer.
// Echtes Postgres, weil genau das Zusammenspiel aus gespeicherter Zeile und
// Fallback-Entscheidung geprüft wird.

import (
	"context"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
)

func TestLeereMahnvorlageFaelltAufStandardtextZurueck(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	// Bestehenden Seed-Text sichern und die Vorlage leer speichern — genau der
	// Zustand, den der Editor heute zulässt.
	var altBetreff, altText string
	if err := pool.QueryRow(ctx,
		`SELECT betreff, text_body FROM mail_vorlagen WHERE typ = 'MAHNUNG_ELTERN'`).
		Scan(&altBetreff, &altText); err != nil {
		t.Fatalf("Seed-Vorlage lesen: %v", err)
	}
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `UPDATE mail_vorlagen SET betreff = $1, text_body = $2 WHERE typ = 'MAHNUNG_ELTERN'`, altBetreff, altText)
	})
	if _, err := pool.Exec(ctx,
		`UPDATE mail_vorlagen SET betreff = '', text_body = '   ' WHERE typ = 'MAHNUNG_ELTERN'`); err != nil {
		t.Fatalf("Vorlage leeren: %v", err)
	}

	betreff, textBody := srv.ladeMahnVorlageText(ctx)
	if strings.TrimSpace(betreff) == "" || strings.TrimSpace(textBody) == "" {
		t.Errorf("leere Vorlage muss den Standardtext liefern, kam: betreff=%q text=%q", betreff, textBody)
	}
}
