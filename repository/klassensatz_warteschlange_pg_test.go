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
	ersteID, err := repo.CreateKlassensatzReservierung(ctx, titel, "8a", 3, nil, bearbeiter)
	if err != nil {
		t.Fatalf("Reservierung 8a: %v", err)
	}
	if _, err := repo.CreateKlassensatzReservierung(ctx, titel, "9b", 2, nil, bearbeiter); err != nil {
		t.Fatalf("Reservierung 9b: %v", err)
	}

	// (1) Admin-Liste: Warteschlangen-Reihenfolge (älteste zuerst), verfügbar = 3.
	liste, err := repo.GetKlassensatzReservierungen(ctx)
	if err != nil {
		t.Fatalf("Liste: %v", err)
	}
	meine := nurTitel(liste, titel)
	if len(meine) != 2 || meine[0].Klasse != "8a" || meine[1].Klasse != "9b" {
		t.Fatalf("Warteschlange: erwartet [8a, 9b], bekam %+v", meine)
	}
	if meine[0].Verfuegbar != 3 {
		t.Errorf("verfügbar = %d, erwartet 3 (alle im Regal)", meine[0].Verfuegbar)
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
	if len(offeneMeine) != 2 || offeneMeine[0].Klasse != "8a" || offeneMeine[1].Klasse != "9b" {
		t.Fatalf("Portal-Warteschlange: erwartet [8a, 9b], bekam %+v", offeneMeine)
	}

	// (4) 8a wird erledigt → die 9b rückt in der offenen Liste nach vorn, die
	// erledigte 8a bleibt als Historie dahinter.
	if _, err := repo.ErledigeKlassensatzReservierung(ctx, ersteID); err != nil {
		t.Fatalf("Erledigen: %v", err)
	}
	liste, err = repo.GetKlassensatzReservierungen(ctx)
	if err != nil {
		t.Fatalf("Liste nach Erledigen: %v", err)
	}
	meine = nurTitel(liste, titel)
	if len(meine) != 2 || meine[0].Klasse != "9b" || meine[0].Erledigt || !meine[1].Erledigt {
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
