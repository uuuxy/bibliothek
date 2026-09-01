package api

// Leere Vorlage ⇒ leerer Brief (Sweep 01.09.2026, Fund F7): loadMahnungTemplate
// fiel nur bei DB-FEHLER auf den Standardtext zurück — eine versehentlich leer
// gespeicherte Vorlage (der Editor lässt '' durch, die Spalten sind NOT NULL,
// aber '' ist erlaubt) erzeugte Mahnbriefe ohne Betreff und ohne Anschreiben.
// Die Bestell-Schwester (loadBestellTemplate) prüfte von Anfang an auf leer.
// Echtes Postgres, weil genau das Zusammenspiel aus gespeicherter Zeile und
// Fallback-Entscheidung geprüft wird.

import (
	"context"
	"strings"
	"testing"

	"bibliothek/db"
)

func TestLeereMahnvorlageFaelltAufStandardtextZurueck(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	// Bestehenden Seed-Text sichern und die Vorlage leer speichern — genau der
	// Zustand, den der Editor heute zulässt.
	var altBetreff, altText string
	if err := pool.QueryRow(ctx,
		`SELECT betreff, text_body FROM mail_vorlagen WHERE typ = 'MAHNUNG_ELTERN'`).
		Scan(&altBetreff, &altText); err != nil {
		t.Fatalf("Seed-Vorlage lesen: %v", err)
	}
	t.Cleanup(func() {
		aufraeumen(t, pool, `UPDATE mail_vorlagen SET betreff = $1, text_body = $2 WHERE typ = 'MAHNUNG_ELTERN'`, altBetreff, altText)
	})
	if _, err := pool.Exec(ctx,
		`UPDATE mail_vorlagen SET betreff = '', text_body = '   ' WHERE typ = 'MAHNUNG_ELTERN'`); err != nil {
		t.Fatalf("Vorlage leeren: %v", err)
	}

	betreff, textBody := srv.loadMahnungTemplate(ctx)
	if strings.TrimSpace(betreff) == "" || strings.TrimSpace(textBody) == "" {
		t.Errorf("leere Vorlage muss den Standardtext liefern, kam: betreff=%q text=%q", betreff, textBody)
	}
}
