package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Das Warteschlangen-Modell der Klassensatz-Reservierungen am echten Postgres:
// Reservieren sperrt nichts — die Reihenfolge (älteste zuerst) und die je Zeile
// mitgelieferte Verfügbarkeit sind die ganze Steuerung. Beides ist SQL-Verhalten
// (ORDER BY über zwei CASE-Zweige, Verfügbar-Subquery mit der OPAC-Definition),
// das pgxmock nur nachspielen könnte.
func TestKlassensatzReservierungen_WarteschlangeUndVerfuegbarkeit(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewReservationRepository(pool)

	// Ein Titel mit DREI Exemplaren — seedSignaturMitExemplaren taugt hier nicht,
	// es legt pro Exemplar einen eigenen Titel an (der Verfügbar-Blick wäre immer 1).
	var titel string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel, signatur) VALUES ('KSQ-Klassensatz', 'KSQ') RETURNING id`).Scan(&titel); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	exemplare := make([]string, 3)
	for i := range exemplare {
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`,
			titel, "BC-KSQ-Q-"+string(rune('0'+i))).Scan(&exemplare[i]); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
	}
	bearbeiter := seedEigenerBearbeiter(t, pool, "KSQ-B")

	// Zwei Reservierungen nacheinander: 8a zuerst, 9b stellt sich an.
	ersteID, _, err := repo.CreateKlassensatzReservierung(ctx, titel, "8a", 3, nil, bearbeiter, nil)
	if err != nil {
		t.Fatalf("Reservierung 8a: %v", err)
	}
	if _, _, err := repo.CreateKlassensatzReservierung(ctx, titel, "9b", 2, nil, bearbeiter, nil); err != nil {
		t.Fatalf("Reservierung 9b: %v", err)
	}

	// (1) Admin-Liste: Warteschlangen-Reihenfolge (älteste zuerst), verfügbar = 3.
	liste, err := repo.GetKlassensatzReservierungen(ctx)
	if err != nil {
		t.Fatalf("Liste: %v", err)
	}
	meine := nurTitel(liste, titel)
	if len(meine) != 2 || meine[0].Klasse != "08A" || meine[1].Klasse != "09B" {
		t.Fatalf("Warteschlange: erwartet [08A, 09B], bekam %+v", meine)
	}
	if meine[0].Verfuegbar != 3 {
		t.Errorf("verfügbar = %d, erwartet 3 (alle im Regal)", meine[0].Verfuegbar)
	}
	// Der Anfragende steht MIT NAMEN in der Liste — das Feld existierte von Anfang
	// an in Struct und UI, wurde aber nie befüllt (Feld ohne Nachfüller): Die
	// Bibliothek wusste nicht, welche Lehrkraft sie anrufen soll.
	if meine[0].AngefordertVon == nil || *meine[0].AngefordertVon != "Klassensatz Kraft" {
		t.Errorf("angefordert_von = %v, erwartet den Klarnamen der Lehrkraft", meine[0].AngefordertVon)
	}

	// (2) Ein Exemplar wird verliehen → verfügbar sinkt auf 2. Genau dieser Blick
	// erspart der Bibliothek das Zählen am Regal.
	schueler := seedSchueler(t, pool, "S-KSQ-1", "Kim", "8a")
	seedAusleihe(t, pool, exemplare[0], schueler, bearbeiter)

	liste, err = repo.GetKlassensatzReservierungen(ctx)
	if err != nil {
		t.Fatalf("Liste nach Ausleihe: %v", err)
	}
	if v := nurTitel(liste, titel)[0].Verfuegbar; v != 2 {
		t.Errorf("verfügbar nach Ausleihe = %d, erwartet 2", v)
	}

	// (3) Die Portal-Sicht führt beide in derselben Reihenfolge — ohne Personendaten.
	offene, err := repo.OffeneKlassensatzReservierungen(ctx)
	if err != nil {
		t.Fatalf("Offene: %v", err)
	}
	offeneMeine := []KlassensatzOffen{}
	for _, o := range offene {
		if o.TitelID == titel {
			offeneMeine = append(offeneMeine, o)
		}
	}
	if len(offeneMeine) != 2 || offeneMeine[0].Klasse != "08A" || offeneMeine[1].Klasse != "09B" {
		t.Fatalf("Portal-Warteschlange: erwartet [08A, 09B], bekam %+v", offeneMeine)
	}
	// Der TITEL muss mitkommen: Auf der Startfläche des Portals steht die Warteschlange
	// ohne ein Buch daneben, und "Klasse 8a · 3 Stück" allein sagt niemandem, worum es
	// geht (bis zum 23.08.2026 lieferte die Abfrage nur die titel_id — der Kommentar am
	// Struct nannte den Titel trotzdem).
	if offeneMeine[0].Titel == "" || offeneMeine[1].Titel == "" {
		t.Errorf("Titel fehlt in der Portal-Warteschlange: %+v", offeneMeine)
	}

	// (4) 8a wird erledigt → die 9b rückt in der offenen Liste nach vorn, die
	// erledigte 8a bleibt als Historie dahinter. Das Abschliessen liefert die
	// Angaben für die Bereit-Mail — inklusive der Konto-Adresse der Lehrkraft.
	erledigt, err := repo.ErledigeKlassensatzReservierung(ctx, ersteID, "")
	if err != nil {
		t.Fatalf("Erledigen: %v", err)
	}
	if erledigt == nil || erledigt.Klasse != "08A" || erledigt.Anzahl != 3 ||
		erledigt.AnfragendeMail == nil || *erledigt.AnfragendeMail != "KSQ-B@example.org" {
		t.Fatalf("Bereit-Mail-Angaben falsch: %+v", erledigt)
	}
	// Der zweite Klick (zweiter Admin) schliesst nichts erneut ab — und löst damit
	// auch keine zweite Bereit-Mail aus.
	if nochmal, err := repo.ErledigeKlassensatzReservierung(ctx, ersteID, ""); err != nil || nochmal != nil {
		t.Fatalf("Doppel-Erledigen: erwartet nil ohne Fehler, bekam %+v / %v", nochmal, err)
	}
	liste, err = repo.GetKlassensatzReservierungen(ctx)
	if err != nil {
		t.Fatalf("Liste nach Erledigen: %v", err)
	}
	meine = nurTitel(liste, titel)
	if len(meine) != 2 || meine[0].Klasse != "09B" || meine[0].Erledigt || !meine[1].Erledigt {
		t.Fatalf("nach Erledigen: erwartet [9b offen, 8a erledigt], bekam %+v", meine)
	}
}

// nurTitel filtert die Gesamtliste auf einen Titel — andere Tests des Pakets
// hinterlassen eigene Reservierungen.
func nurTitel(liste []KlassensatzReservierung, titelID string) []KlassensatzReservierung {
	out := []KlassensatzReservierung{}
	for _, r := range liste {
		if r.TitelID == titelID {
			out = append(out, r)
		}
	}
	return out
}

// seedEigenerBearbeiter legt einen Benutzer mit eigenem Barcode an — der feste
// Barcode von seedBearbeiter gehört dem Race-Test (Reihenfolge wäre Schicksal).
func seedEigenerBearbeiter(t *testing.T, pool *pgxpool.Pool, barcode string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		 VALUES ($1, 'Klassensatz', 'Kraft', $2, 'mitarbeiter', true) RETURNING id`,
		barcode, barcode+"@example.org").Scan(&id); err != nil {
		t.Fatalf("Bearbeiter anlegen: %v", err)
	}
	return id
}

// TestKlassensatzIdempotenz belegt den Doppelklick-Schutz (Migration 076): Zwei
// Anfragen mit DEMSELBEN Idempotenz-Schlüssel erzeugen nur EINE Reservierung (die
// zweite gibt dieselbe ID zurück, neu=false). Ein anderer Schlüssel legt regulär an.
func TestKlassensatzIdempotenz(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewReservationRepository(pool)

	ex := seedSignaturMitExemplaren(t, pool, "KsIdem", 3)
	titel := titelIDVonExemplar(t, pool, ex[0])
	bearbeiter := seedBearbeiter(t, pool)
	key := "11111111-1111-1111-1111-111111111111"

	id1, neu1, err := repo.CreateKlassensatzReservierung(ctx, titel, "8a", 2, nil, bearbeiter, &key)
	if err != nil || !neu1 {
		t.Fatalf("erste Reservierung: id=%q neu=%v err=%v", id1, neu1, err)
	}
	// Doppelklick: gleicher Schlüssel → dieselbe ID, neu=false, KEINE zweite Zeile.
	id2, neu2, err := repo.CreateKlassensatzReservierung(ctx, titel, "8a", 2, nil, bearbeiter, &key)
	if err != nil {
		t.Fatalf("zweiter Klick: %v", err)
	}
	if neu2 {
		t.Error("zweiter Klick mit gleichem Schlüssel muss No-op sein (neu=false)")
	}
	if id2 != id1 {
		t.Errorf("zweiter Klick muss dieselbe ID liefern: %q vs %q", id2, id1)
	}

	var anzahl int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM klassensatz_reservierungen WHERE idempotenz_schluessel = $1`, key).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	if anzahl != 1 {
		t.Fatalf("genau 1 Reservierung erwartet, waren %d", anzahl)
	}

	// Anderer Schlüssel (bewusste zweite Reservierung) → legt regulär an.
	key2 := "22222222-2222-2222-2222-222222222222"
	if _, neu3, err := repo.CreateKlassensatzReservierung(ctx, titel, "8a", 1, nil, bearbeiter, &key2); err != nil || !neu3 {
		t.Fatalf("bewusste zweite Reservierung muss durchgehen: neu=%v err=%v", neu3, err)
	}
}
