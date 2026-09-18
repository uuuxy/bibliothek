package api

import (
	"context"
	"testing"
	"time"

	"bibliothek/internal/service"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Das Zugangsdatum (Migration 129) ist der Tag, an dem ein Exemplar in den Bestand kommt —
// nicht der Tag, an dem seine Zeile entsteht. Im Bestellweg entsteht die Zeile beim
// BESTELLEN; ein Zugangsbuch, das daraus rechnet, führt Bücher, die beim Händler liegen,
// und datiert gelieferte auf den Bestelltag.
//
// Gemessen wird an den echten Türen, nicht am Trigger allein.

func zugangsdatum(t *testing.T, pool *pgxpool.Pool, id string) *time.Time {
	t.Helper()
	var tag *time.Time
	if err := pool.QueryRow(context.Background(),
		`SELECT zugang_am FROM buecher_exemplare WHERE id = $1`, id).Scan(&tag); err != nil {
		t.Fatalf("zugang_am lesen: %v", err)
	}
	return tag
}

// zulaufExemplar legt ein Exemplar an, wie es das Bestellwesen anlegt: nicht ausleihbar,
// mit Bestellstatus, mit dem heutigen Tag als erworben_am (Vorgabewert der Spalte).
func zulaufExemplar(t *testing.T, pool *pgxpool.Pool, titelID, barcode string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus)
		VALUES ($1, $2, false, 'im_zulauf') RETURNING id`, titelID, barcode).Scan(&id); err != nil {
		t.Fatalf("Zulauf-Exemplar %q: %v", barcode, err)
	}
	return id
}

func TestZugangsdatum_ErstBeimEintreffen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	// Der Kalendertag der SCHULE, nicht der des Testprozesses: zugang_am ist ein DATE,
	// und der Trigger bildet es seit Migration 130 in Europe/Berlin. Mit time.Now() hing
	// die Erwartung an der Zone, in der der Test zufällig lief — lokal (MESZ) und in der
	// CI (UTC) wichen die beiden zwischen Mitternacht und 2 Uhr um einen Tag ab, und der
	// Lauf vom 18.09.2026 war in der CI genau deshalb rot. Gleiche Lesart wie im
	// Nachbartest (zugangsbuch_pg_test.go).
	heute := time.Now().In(schulzeit.Zone())
	titelID := titelMitSignatur(t, pool, "Zugangsdatum-Titel", "Zdt 1", 0)

	// 1. Ein Exemplar im Zulauf hat KEIN Zugangsdatum. Es ist bestellt, nicht da.
	imZulauf := zulaufExemplar(t, pool, titelID, "ZDT-ZULAUF")
	if tag := zugangsdatum(t, pool, imZulauf); tag != nil {
		t.Fatalf("Exemplar im Zulauf trägt ein Zugangsdatum (%s) — bestellt ist nicht zugegangen", tag)
	}

	// 2. Tür „Wareneingang": Der Zugang ist der Tag der Lieferung.
	geliefert := zulaufExemplar(t, pool, titelID, "ZDT-WARENEINGANG")
	if _, err := service.BulkReceiveOrder(ctx, pool, repository.NewAuditRepository(pool), service.BulkReceiveParams{
		ExemplarIDs: []string{geliefert},
	}); err != nil {
		t.Fatalf("Wareneingang: %v", err)
	}
	tag := zugangsdatum(t, pool, geliefert)
	if tag == nil {
		t.Fatal("nach dem Wareneingang steht kein Zugangsdatum — das Buch fehlt im Zugangsbuch")
	}
	if !gleicherTag(*tag, heute) {
		t.Fatalf("Zugang nach Wareneingang: %s, erwartet %s", tag.Format("2006-01-02"), heute.Format("2006-01-02"))
	}

	// 3. Tür „Freigeben" im Status-Editor — dieselbe Wirkung, anderer Schreiber.
	freigegeben := zulaufExemplar(t, pool, titelID, "ZDT-FREIGABE")
	bookRepo := repository.NewBookRepository(pool)
	if err := bookRepo.UpdateCopyStatus(ctx, freigegeben, true, false, "", nil); err != nil {
		t.Fatalf("Freigeben: %v", err)
	}
	if tag := zugangsdatum(t, pool, freigegeben); tag == nil || !gleicherTag(*tag, heute) {
		t.Fatalf("Zugang nach Freigabe: %v", tag)
	}

	// 4. Ein bestelltes Exemplar, das nie ankommt und ausgebucht wird, ist KEIN Zugang.
	nieGekommen := zulaufExemplar(t, pool, titelID, "ZDT-NIE")
	if err := bookRepo.DecommissionCopy(ctx, nieGekommen); err != nil {
		t.Fatalf("Aussondern aus dem Zulauf: %v", err)
	}
	if tag := zugangsdatum(t, pool, nieGekommen); tag != nil {
		t.Fatalf("nie geliefertes Exemplar trägt einen Zugang (%s)", tag)
	}
}

// Außerhalb des Bestellwegs ist der Zugang das Datum, das die Zeile mitbringt: Die
// Littera-Übernahme trägt das echte Zugangsdatum der Altanwendung ein. CURRENT_DATE machte
// aus einem 2019 übernommenen Buch einen Zugang von heute — und das Zugangsbuch eines alten
// Halbjahres wäre leer.
func TestZugangsdatum_AnlageAusserhalbDerBestellung(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	titelID := titelMitSignatur(t, pool, "Altbestands-Titel", "Zdt 2", 0)

	var alt string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, erworben_am)
		VALUES ($1, 'ZDT-ALT', DATE '2019-04-11') RETURNING id`, titelID).Scan(&alt); err != nil {
		t.Fatalf("Altbestand anlegen: %v", err)
	}
	tag := zugangsdatum(t, pool, alt)
	if tag == nil || tag.Format("2006-01-02") != "2019-04-11" {
		t.Fatalf("Zugang des Altbestands: %v, erwartet 2019-04-11", tag)
	}

	// Ein zweites Update verschiebt den Zugang nicht — sonst wanderte ein Buch bei jeder
	// Notiz in das laufende Halbjahr, und ein abgehefteter Ausdruck stimmte nicht mehr.
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET zustand_notiz = 'Eselsohr' WHERE id = $1`, alt); err != nil {
		t.Fatalf("Notiz schreiben: %v", err)
	}
	if tag := zugangsdatum(t, pool, alt); tag == nil || tag.Format("2006-01-02") != "2019-04-11" {
		t.Fatalf("Zugang nach zweitem Update verschoben: %v", tag)
	}
}

func gleicherTag(a, b time.Time) bool {
	return a.Format("2006-01-02") == b.Format("2006-01-02")
}
