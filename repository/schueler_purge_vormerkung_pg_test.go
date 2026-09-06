package repository

import (
	"context"
	"testing"
)

// TestPurgeStudent_BedientNaechstenWartenden befragt den Fremdschlüssel
// `vormerkungen.schueler_id -> schueler ON DELETE CASCADE` (Frage 12, Gegenrichtung
// Schema): Die DSGVO-Löschung eines Schülers nimmt seine Vormerkungen lautlos mit.
// War eine davon 'abholbereit', lag ein Exemplar für ihn auf dem Abholregal — und der
// CASCADE überspringt den Schritt, den der Verfall-Lauf ausdrücklich tut: das
// freigewordene Exemplar dem NÄCHSTEN Wartenden zuteilen. Ohne diesen Schritt bleibt
// der nächste Schüler für immer 'wartend', während das Buch verfügbar herumsteht.
func TestPurgeStudent_BedientNaechstenWartenden(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := seedSignaturMitExemplaren(t, pool, "PurgeVorm", 1)
	titelID := titelIDVonExemplar(t, pool, ex[0])
	gehend := seedSchueler(t, pool, "PV-1", "Mia", "9a")
	wartend := seedSchueler(t, pool, "PV-2", "Ben", "9a")
	bearbeiter := seedBearbeiter(t, pool)

	// Der gehende Schüler hat das Exemplar abholbereit liegen.
	if _, err := pool.Exec(ctx,
		`INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis, erstellt_am)
		 VALUES ($1, $2, 'abholbereit', $3, now() + interval '3 days', now() - interval '2 days')`,
		titelID, gehend, ex[0]); err != nil {
		t.Fatalf("abholbereite Vormerkung anlegen: %v", err)
	}
	// Hinter ihm wartet jemand auf denselben Titel.
	var wartendeID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO vormerkungen (titel_id, schueler_id, status, erstellt_am)
		 VALUES ($1, $2, 'wartend', now() - interval '1 day') RETURNING id`,
		titelID, wartend).Scan(&wartendeID); err != nil {
		t.Fatalf("wartende Vormerkung anlegen: %v", err)
	}

	// In den Papierkorb legen, damit PurgeStudent überhaupt greift.
	if _, err := pool.Exec(ctx, `UPDATE schueler SET deleted_at = now() WHERE id = $1`, gehend); err != nil {
		t.Fatalf("weichlöschen: %v", err)
	}

	if err := NewAuditRepository(pool).PurgeStudent(ctx, gehend, bearbeiter); err != nil {
		t.Fatalf("PurgeStudent: %v", err)
	}

	var status string
	var bereitgestellt *string
	if err := pool.QueryRow(ctx,
		`SELECT status, bereitgestellt_exemplar_id::text FROM vormerkungen WHERE id = $1`, wartendeID).
		Scan(&status, &bereitgestellt); err != nil {
		t.Fatalf("wartende Vormerkung lesen: %v", err)
	}
	if status != "abholbereit" {
		t.Errorf("nächste Vormerkung = %q, erwartet 'abholbereit' — das freigewordene Exemplar wurde niemandem zugeteilt", status)
	}
	if bereitgestellt == nil || *bereitgestellt != ex[0] {
		t.Errorf("bereitgestellt_exemplar_id = %v, erwartet %s", bereitgestellt, ex[0])
	}
}
