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
		Art: "wunsch", TitelText: "Markl Biologie 2", ISBN: "978-3-12-150010-9",
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

	// Jedes Feld in seiner eigenen Spalte. Sieben gleichartige Strings gingen bis
	// 23.08. als Positionsparameter in Create; ein Dreher zwischen ISBN und Klasse
	// hätte hier keinen Compilerfehler ausgelöst, sondern eine ISBN in der
	// Klassenspalte der LMF-Liste.
	if a := eigene[0]; a.Art != "wunsch" || a.ISBN != "978-3-12-150010-9" ||
		a.Klasse != "8G3" || a.Kommentar != "bitte zum Halbjahr" {
		t.Errorf("Felder vertauscht: %+v", a)
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
