package repository

import (
	"context"
	"testing"
)

// Gemischte Auflagen in einer Klasse (docs/OFFEN.md 4.18, Stufe 5) gegen echtes Postgres: Wer
// zählt zur Klasse (klassen_normkey, nur aktive Kinder), welche Ausleihe zählt (nur offene,
// nur ANDERE Auflagen desselben Werks) — das steht im SQL und an der Sicht schueler.
func TestAuflagenMischungInKlasse(t *testing.T) {
	pool := pgTestPool(t)
	resetAuflagen(t, pool)
	ctx := context.Background()

	alt := seedAuflage(t, pool, "Mathe 7", 2019, true)
	neu := seedAuflage(t, pool, "Mathe 7", 2023, true)
	einzeln := seedAuflage(t, pool, "Physik 8", 2020, true)
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET auflage = CASE WHEN id = $1::uuid THEN '3. Aufl.' ELSE '4. Aufl.' END
		WHERE id IN ($1::uuid, $2::uuid)`, alt, neu); err != nil {
		t.Fatal(err)
	}
	if _, err := FasseAuflagenZusammen(ctx, pool, alt, neu); err != nil {
		t.Fatal(err)
	}

	nr := 0
	leihe := func(titelID, schuelerID string, zurueck bool) {
		t.Helper()
		nr++
		var exemplar string
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`,
			titelID, "B-5180"+string(rune('0'+nr/10))+string(rune('0'+nr%10))).Scan(&exemplar); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
		rueckgabe := "NULL"
		if zurueck {
			rueckgabe = "now()"
		}
		if _, err := pool.Exec(ctx, `INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist, rueckgabe_am)
			VALUES ($1, $2, CURRENT_DATE + 300, `+rueckgabe+`)`, exemplar, schuelerID); err != nil {
			t.Fatalf("Ausleihe anlegen: %v", err)
		}
	}

	ida := seedSchueler(t, pool, "S-518001", "Ida", "7b")
	ben := seedSchueler(t, pool, "S-518002", "Ben", "7b")
	cem := seedSchueler(t, pool, "S-518003", "Cem", "07B") // dieselbe Klasse, andere Schreibweise
	dora := seedSchueler(t, pool, "S-518004", "Dora", "7b")
	emil := seedSchueler(t, pool, "S-518005", "Emil", "7c")
	neuling := seedSchueler(t, pool, "S-518006", "Nele", "7b")
	leihe(alt, ida, false)
	leihe(alt, ben, false)
	leihe(alt, cem, false)
	leihe(alt, dora, true) // zurückgegeben — zählt nicht
	leihe(alt, emil, false)
	leihe(neu, neuling, false) // die Ausleihe, zu der der Hinweis gehört

	// Wer nicht mehr zur Klasse gehört, zählt nicht — auch mit der alten Auflage in der Hand:
	// ein Abgänger und ein gelöschtes Kind. Die Sicht schueler filtert beides nicht.
	gustav := seedSchueler(t, pool, "S-518009", "Gustav", "7b")
	hanna := seedSchueler(t, pool, "S-518010", "Hanna", "7b")
	leihe(alt, gustav, false)
	leihe(alt, hanna, false)
	if _, err := pool.Exec(ctx, `UPDATE leser SET ist_abgaenger = true WHERE id = $1`, gustav); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE leser SET deleted_at = now() WHERE id = $1`, hanna); err != nil {
		t.Fatal(err)
	}

	m, err := AuflagenMischungInKlasse(ctx, pool, neu, neuling)
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("kein Hinweis — in der 7b haben drei Kinder die 3. Auflage")
	}
	// Die Klasse in der Schreibweise des Vokabulars (Migration 087): aus „7b" wird „07B".
	if m.Klasse != "07B" || m.Auflage != "4. Aufl." || m.Erscheinungsjahr != 2023 {
		t.Errorf("Kopf %+v — erwartet 07B, 4. Aufl., 2023", m)
	}
	if len(m.Andere) != 1 || m.Andere[0].ID != alt || m.Andere[0].Auflage != "3. Aufl." || m.Andere[0].Kinder != 3 {
		t.Errorf("andere Auflagen %+v — erwartet 3. Aufl. bei 3 Kindern (Ida, Ben, Cem; nicht Dora, Gustav, Hanna, nicht die 7c)", m.Andere)
	}

	// Ein Kollege gehört zu keiner Klasse, auch wenn im Feld eine steht: kein Hinweis.
	var kollege string
	if err := pool.QueryRow(ctx, `INSERT INTO leser (barcode_id, vorname, nachname, art, klasse)
		VALUES ('L-518011', 'Lena', 'Test', 'lehrkraft', '7b') RETURNING id`).Scan(&kollege); err != nil {
		t.Fatal(err)
	}
	if m, err := AuflagenMischungInKlasse(ctx, pool, neu, kollege); err != nil || m != nil {
		t.Errorf("Kollege: %+v, %v — erwartet kein Hinweis", m, err)
	}

	// Die 7c bekommt die neue Auflage: dort hat nur Emil die alte.
	emma := seedSchueler(t, pool, "S-518007", "Emma", "7c")
	if m, err := AuflagenMischungInKlasse(ctx, pool, neu, emma); err != nil || m == nil || m.Andere[0].Kinder != 1 {
		t.Errorf("7c: %+v, %v — erwartet 1 Kind mit der 3. Auflage", m, err)
	}

	// Ohne Werk kein Hinweis; ebenso ohne andere Auflage in der Klasse.
	if m, err := AuflagenMischungInKlasse(ctx, pool, einzeln, neuling); err != nil || m != nil {
		t.Errorf("Titel ohne Werk: %+v, %v — erwartet kein Hinweis", m, err)
	}
	fremd := seedSchueler(t, pool, "S-518008", "Finn", "9a")
	if m, err := AuflagenMischungInKlasse(ctx, pool, neu, fremd); err != nil || m != nil {
		t.Errorf("9a ohne andere Auflage: %+v, %v — erwartet kein Hinweis", m, err)
	}
}
