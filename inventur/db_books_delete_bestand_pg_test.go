package inventur

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Die Massenaktion „Titel löschen" vermerkt an der Spur jedes Exemplars, ob es im Bestand
// war. Das Abgangsbuch zählt danach; ein bestelltes, nie eingetroffenes Exemplar ist kein
// Abgang.
func TestDeleteBooks_VermerktObDasExemplarImBestandWar(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	var titelID, imBestand, imZulauf string
	t.Cleanup(func() {
		ctx := context.Background()
		if _, err := pool.Exec(ctx, `DELETE FROM audit_log WHERE datensatz_id = ANY($1)`,
			[]string{titelID, imBestand, imZulauf}); err != nil {
			t.Errorf("Protokoll aufräumen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE titel = 'Bestand und Zulauf'`); err != nil {
			t.Errorf("Titel aufräumen: %v", err)
		}
	})
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel) VALUES ('Bestand und Zulauf') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'DB-BESTAND') RETURNING id`,
		titelID).Scan(&imBestand); err != nil {
		t.Fatalf("Exemplar im Bestand: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus)
		VALUES ($1, 'DB-ZULAUF', false, 'im_zulauf') RETURNING id`, titelID).Scan(&imZulauf); err != nil {
		t.Fatalf("Exemplar im Zulauf: %v", err)
	}

	if err := NewBookRepository(pool).DeleteBooks(ctx, []string{titelID}); err != nil {
		t.Fatalf("Titel löschen: %v", err)
	}

	for _, fall := range []struct {
		id, barcode string
		erwartet    bool
	}{
		{imBestand, "DB-BESTAND", true},
		{imZulauf, "DB-ZULAUF", false},
	} {
		var vermerk *bool
		if err := pool.QueryRow(ctx, `
			SELECT (details->>$2)::boolean FROM audit_log
			WHERE tabelle = 'buecher_exemplare' AND aktion = 'DELETE' AND datensatz_id = $1`,
			fall.id, repository.AuditDetailWarImBestand).Scan(&vermerk); err != nil {
			t.Fatalf("Spur von %s: %v", fall.barcode, err)
		}
		if vermerk == nil || *vermerk != fall.erwartet {
			t.Errorf("%s: Vermerk %v, erwartet %v", fall.barcode, vermerk, fall.erwartet)
		}
	}
}
