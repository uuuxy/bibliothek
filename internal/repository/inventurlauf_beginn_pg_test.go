package repository

import (
	"context"
	"testing"

	"bibliothek/internal/testdaten"
)

// Ein Buch, das erst während einer Inventur ins Regal kommt, bucht ihr Abschluss nicht als
// Verlust: an der Theke zurückgegeben, entsperrt oder neu angelegt. Es stand beim Beginn nicht im
// Regal; dass niemand es gescannt hat, beweist keinen Verlust. Die Zählung vor dem Abschluss nennt
// dieselben Bücher. Gegenprobe: Das Buch, das beim Beginn im Regal stand und fehlt, bucht er.
func TestSchliesseLaufAb_WasWaehrendDesLaufsInsRegalKommtIstKeinVerlust(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)
	bearbeiter := legeBearbeiterAn(t, pool)

	chemie := "Chemie"
	// ex[0] steht beim Beginn im Regal und fehlt; ex[1] ist verliehen, ex[2] gesperrt.
	ex := legeSignaturMitExemplarenAn(t, pool, chemie, 3)
	leiheAus(t, pool, ex[1])
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET ist_ausleihbar = false WHERE id = $1`, ex[2]); err != nil {
		t.Fatalf("Exemplar sperren: %v", err)
	}
	lauf, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &chemie}, "Chemie", "")
	if err != nil {
		t.Fatalf("Lauf anlegen: %v", err)
	}

	// Während des Laufs: Rückgabe an der Theke, Entsperren, ein neues Exemplar.
	if _, err := pool.Exec(ctx, `UPDATE ausleihen SET rueckgabe_am = now()
		WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`, ex[1]); err != nil {
		t.Fatalf("Rückgabe buchen: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET ist_ausleihbar = true WHERE id = $1`, ex[2]); err != nil {
		t.Fatalf("Exemplar entsperren: %v", err)
	}
	var titelID string
	if err := pool.QueryRow(ctx, `SELECT titel_id FROM buecher_exemplare WHERE id = $1`, ex[0]).Scan(&titelID); err != nil {
		t.Fatalf("Titel lesen: %v", err)
	}
	neu := testdaten.LegeExemplarAn(t, pool, titelID, "BC-Chemie-neu", true, "")

	stand, err := repo.ZaehleStand(ctx, lauf.ID, lauf.Bereich())
	if err != nil {
		t.Fatalf("ZaehleStand: %v", err)
	}
	if stand != (LaufStand{Erwartet: 1, Fehlend: 1}) {
		t.Errorf("Stand = %+v, erwartet 1 erwartet und 1 fehlend: nur das Buch vom Beginn", stand)
	}
	verloren, err := repo.SchliesseLaufAb(ctx, lauf.ID, lauf.Bereich(), bearbeiter)
	if err != nil {
		t.Fatalf("Abschluss: %v", err)
	}
	if verloren != stand.Fehlend {
		t.Errorf("Abschluss buchte %d Verluste, der Stand nannte %d fehlend", verloren, stand.Fehlend)
	}
	if aus := ausgesonderteZahl(t, pool, ex[:1]); aus != 1 {
		t.Error("das Buch, das beim Beginn im Regal stand und fehlt, ist nicht als Verlust gebucht")
	}
	if aus := ausgesonderteZahl(t, pool, []string{ex[1], ex[2], neu}); aus != 0 {
		t.Errorf("%d Bücher, die erst während des Laufs ins Regal kamen, als Verlust gebucht", aus)
	}
}

// Ein Buch, das für eine Vormerkung im Abholfach liegt, steht nicht im Regal: Der Abschluss bucht
// es nicht als Verlust, auch wenn es vor dem Beginn dorthin kam und niemand es scannt. Als Verlust
// ausgesondert, teilte es der Verfall der Vormerkung keinem Wartenden mehr zu. Gegenprobe: Das
// Buch, das im Regal fehlt, bucht er.
func TestSchliesseLaufAb_BuchImAbholfachIstKeinVerlust(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)
	bearbeiter := legeBearbeiterAn(t, pool)

	physik := "Physik"
	// ex[0] liegt im Abholfach, ex[1] steht im Regal und fehlt.
	ex := legeSignaturMitExemplarenAn(t, pool, physik, 2)
	var schueler string
	if err := pool.QueryRow(ctx, `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('S-ABHOL', 'Test', 'Schueler', '7a', 2030) RETURNING id::text`).Scan(&schueler); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis)
		SELECT titel_id, $2, 'abholbereit', id, now() + interval '3 days' FROM buecher_exemplare WHERE id = $1`,
		ex[0], schueler); err != nil {
		t.Fatalf("Vormerkung anlegen: %v", err)
	}
	lauf, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &physik}, "Physik", "")
	if err != nil {
		t.Fatalf("Lauf anlegen: %v", err)
	}

	stand, err := repo.ZaehleStand(ctx, lauf.ID, lauf.Bereich())
	if err != nil {
		t.Fatalf("ZaehleStand: %v", err)
	}
	if stand != (LaufStand{Erwartet: 1, Fehlend: 1}) {
		t.Errorf("Stand = %+v, erwartet 1 erwartet und 1 fehlend: nur das Buch im Regal", stand)
	}
	verloren, err := repo.SchliesseLaufAb(ctx, lauf.ID, lauf.Bereich(), bearbeiter)
	if err != nil {
		t.Fatalf("Abschluss: %v", err)
	}
	if verloren != 1 {
		t.Errorf("Abschluss buchte %d Verluste, erwartet 1", verloren)
	}
	if aus := ausgesonderteZahl(t, pool, ex[:1]); aus != 0 {
		t.Error("das Buch im Abholfach ist als Verlust gebucht")
	}
	if aus := ausgesonderteZahl(t, pool, ex[1:]); aus != 1 {
		t.Error("das Buch, das im Regal fehlt, ist nicht als Verlust gebucht")
	}
}

// Verlässt ein Buch das Abholfach während des Laufs, ohne verliehen zu werden, kommt es ins Regal,
// vielleicht in ein schon gezähltes: Der Abschluss bucht es nicht als Verlust, gleich ob nur die
// Abholfrist abgelaufen ist, der stündliche Lauf die Vormerkung gelöscht hat oder jemand sie von
// Hand löscht. Gegenprobe: Das Buch, das im Regal fehlt, bucht er.
func TestSchliesseLaufAb_BuchVerlaesstDasAbholfachWaehrendDesLaufs(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)
	vormerkungen := NewVormerkungsAbfragen(pool)
	bearbeiter := legeBearbeiterAn(t, pool)

	bio := "Biologie"
	// ex[0] bis ex[2] liegen beim Beginn im Abholfach, ex[3] steht im Regal und fehlt.
	ex := legeSignaturMitExemplarenAn(t, pool, bio, 4)
	var vormerkung []string
	for i, ausweis := range []string{"S-ABHOL-LAUF-0", "S-ABHOL-LAUF-1", "S-ABHOL-LAUF-2"} {
		var schueler, id string
		if err := pool.QueryRow(ctx, `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			VALUES ($1, 'Test', 'Schueler', '7a', 2030) RETURNING id::text`, ausweis).Scan(&schueler); err != nil {
			t.Fatalf("Schüler anlegen: %v", err)
		}
		if err := pool.QueryRow(ctx, `INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis)
			SELECT titel_id, $2, 'abholbereit', id, now() + interval '3 days' FROM buecher_exemplare WHERE id = $1
			RETURNING id::text`, ex[i], schueler).Scan(&id); err != nil {
			t.Fatalf("Vormerkung anlegen: %v", err)
		}
		vormerkung = append(vormerkung, id)
	}
	lauf, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &bio}, "Biologie", "")
	if err != nil {
		t.Fatalf("Lauf anlegen: %v", err)
	}

	// Während des Laufs: Die Frist von ex[1] läuft ab, und der stündliche Lauf löscht die
	// Vormerkung; ex[2] gibt jemand von Hand frei; die Frist von ex[0] läuft ab, die Vormerkung
	// steht noch.
	laufAb := `UPDATE vormerkungen SET bereitgestellt_bis = clock_timestamp() WHERE id = $1`
	if _, err := pool.Exec(ctx, laufAb, vormerkung[1]); err != nil {
		t.Fatalf("Frist ablaufen lassen: %v", err)
	}
	if _, _, err := vormerkungen.LasseAbgelaufeneVerfallen(ctx); err != nil {
		t.Fatalf("LasseAbgelaufeneVerfallen: %v", err)
	}
	if err := vormerkungen.Loesche(ctx, vormerkung[2]); err != nil {
		t.Fatalf("Vormerkung löschen: %v", err)
	}
	if _, err := pool.Exec(ctx, laufAb, vormerkung[0]); err != nil {
		t.Fatalf("Frist ablaufen lassen: %v", err)
	}

	stand, err := repo.ZaehleStand(ctx, lauf.ID, lauf.Bereich())
	if err != nil {
		t.Fatalf("ZaehleStand: %v", err)
	}
	if stand != (LaufStand{Erwartet: 1, Fehlend: 1}) {
		t.Errorf("Stand = %+v, erwartet 1 erwartet und 1 fehlend: nur das Buch im Regal", stand)
	}
	verloren, err := repo.SchliesseLaufAb(ctx, lauf.ID, lauf.Bereich(), bearbeiter)
	if err != nil {
		t.Fatalf("Abschluss: %v", err)
	}
	if verloren != 1 {
		t.Errorf("Abschluss buchte %d Verluste, erwartet 1", verloren)
	}
	if aus := ausgesonderteZahl(t, pool, ex[:3]); aus != 0 {
		t.Errorf("%d Bücher, die während des Laufs das Abholfach verließen, als Verlust gebucht", aus)
	}
	if aus := ausgesonderteZahl(t, pool, ex[3:]); aus != 1 {
		t.Error("das Buch, das im Regal fehlt, ist nicht als Verlust gebucht")
	}
}
