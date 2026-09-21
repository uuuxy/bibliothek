package repository

import (
	"context"
	"testing"
)

// Der Anliegen-Kern am echten Postgres: Anlegen, die offene Liste in
// Warteschlangen-Reihenfolge, und das Abhaken liefert die Mail-Daten GENAU
// EINMAL — der Doppelklick zweier Arbeitsplätze (Klassensatz-Lehre) darf
// keine zweite Mail auslösen.
func TestAnliegenLebenszyklus(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	var lehrkraftID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Wanda', 'Wunsch', 'wanda.wunsch@test.invalid', 'kollegium', true)
		ON CONFLICT (email) DO UPDATE SET vorname = EXCLUDED.vorname
		RETURNING id`).Scan(&lehrkraftID); err != nil {
		t.Fatalf("Lehrkraft: %v", err)
	}

	repo := NewAnliegenRepository(pool)
	id1, err := repo.Create(ctx, NeuesAnliegen{
		Art: "wunsch", TitelText: "Markl Biologie 2",
		Klasse: "8G3", Kommentar: "bitte zum Halbjahr", AngefordertVon: lehrkraftID,
	})
	if err != nil {
		t.Fatalf("Create Wunsch: %v", err)
	}
	if _, err := repo.Create(ctx, NeuesAnliegen{
		Art: "meldung", TitelText: "8G3 hat falsche Bücher bekommen",
		Klasse: "8G3", AngefordertVon: lehrkraftID,
	}); err != nil {
		t.Fatalf("Create Meldung: %v", err)
	}

	offene, err := repo.ListOffene(ctx)
	if err != nil {
		t.Fatalf("ListOffene: %v", err)
	}
	var eigene []Anliegen
	for _, a := range offene {
		if a.Von == "Wanda Wunsch" {
			eigene = append(eigene, a)
		}
	}
	if len(eigene) != 2 || eigene[0].TitelText != "Markl Biologie 2" {
		t.Fatalf("offene Liste falsch (älteste zuerst erwartet): %+v", eigene)
	}

	// Jedes Feld in seiner eigenen Spalte. Gleichartige Strings gingen bis 23.08. als
	// Positionsparameter in Create; ein Dreher zwischen Klasse und Kommentar hätte hier
	// keinen Compilerfehler ausgelöst, sondern eine Anmerkung in der Klassenspalte der
	// LMF-Liste.
	if a := eigene[0]; a.Art != "wunsch" || a.Klasse != "8G3" || a.Kommentar != "bitte zum Halbjahr" {
		t.Errorf("Felder vertauscht: %+v", a)
	}

	// titel_id und isbn schreibt seit dem 21.09.2026 niemand mehr; die Spalten bleiben.
	// Der INSERT nennt sie nicht — hier steht, dass die Tabelle das trägt (isbn ist
	// NOT NULL mit Vorgabe) und beide auf ihrer Vorgabe landen.
	var ohneTitel bool
	var isbn string
	if err := pool.QueryRow(ctx, `SELECT titel_id IS NULL, isbn FROM lehrer_anliegen WHERE id = $1`,
		id1).Scan(&ohneTitel, &isbn); err != nil {
		t.Fatalf("Vorgaben lesen: %v", err)
	}
	if !ohneTitel || isbn != "" {
		t.Errorf("titel_id leer = %v, isbn = %q; erwartet NULL und leer", ohneTitel, isbn)
	}

	// Abhaken: Mail-Daten kommen genau einmal.
	erledigt, err := repo.Erledige(ctx, id1, "bestellt, kommt Anfang September")
	if err != nil {
		t.Fatalf("Erledige: %v", err)
	}
	if erledigt == nil || erledigt.AnfragendeMail == nil || *erledigt.AnfragendeMail != "wanda.wunsch@test.invalid" {
		t.Fatalf("Mail-Daten fehlen: %+v", erledigt)
	}
	if erledigt.ErledigtNotiz != "bestellt, kommt Anfang September" {
		t.Errorf("Notiz fehlt in den Mail-Daten: %+v", erledigt)
	}

	// Der zweite Arbeitsplatz drückt denselben Haken: nichts, keine zweite Mail.
	nochmal, err := repo.Erledige(ctx, id1, "doppelt")
	if err != nil || nochmal != nil {
		t.Fatalf("Doppel-Abhaken muss leer ausgehen, got %+v (err=%v)", nochmal, err)
	}

	// Die Lehrkraft sieht ihren Status: 1 erledigt (mit Notiz), 1 offen.
	meine, err := repo.ListEigene(ctx, lehrkraftID)
	if err != nil {
		t.Fatalf("ListEigene: %v", err)
	}
	if len(meine) != 2 {
		t.Fatalf("eigene Liste: %d statt 2", len(meine))
	}
	var erledigte int
	for _, a := range meine {
		if a.ErledigtAm != nil {
			erledigte++
			if a.ErledigtNotiz == "" {
				t.Error("erledigtes Anliegen ohne Notiz im Portal-Status")
			}
		}
	}
	if erledigte != 1 {
		t.Errorf("genau 1 erledigtes Anliegen erwartet, got %d", erledigte)
	}
}
