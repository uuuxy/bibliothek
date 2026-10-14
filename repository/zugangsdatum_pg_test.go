package repository_test

import (
	"context"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/service"
	"bibliothek/internal/testdaten"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Das Zugangsdatum ist der Tag, an dem ein Exemplar in den Bestand kommt —
// nicht der Tag, an dem seine Zeile entsteht. Im Bestellweg entsteht die Zeile beim
// Bestellen; ein Zugangsbuch, das daraus rechnet, führt Bücher, die beim Händler liegen,
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

func TestZugangsdatum_ErstBeimEintreffen(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	// Der Kalendertag der Schule, nicht der des Testprozesses: zugang_am ist ein DATE,
	// und der Trigger bildet es seit Migration 130 in Europe/Berlin. Mit time.Now() hinge
	// die Erwartung an der Zone, in der der Test zufällig läuft — lokal (MESZ) und in der
	// CI (UTC) weichen die beiden zwischen Mitternacht und 2 Uhr um einen Tag ab, und der
	// Lauf in der CI wäre genau dann rot. Gleiche Lesart wie im
	// Nachbartest (api/katalog/zugangsbuch_pg_test.go).
	heute := time.Now().In(schulzeit.Zone())
	titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Zugangsdatum-Titel", "Zdt 1", 0)

	// 1. Ein Exemplar im Zulauf hat kein Zugangsdatum. Es ist bestellt, nicht da.
	imZulauf := testdaten.LegeExemplarImZulaufAn(t, pool, titelID, "ZDT-ZULAUF")
	if tag := zugangsdatum(t, pool, imZulauf); tag != nil {
		t.Fatalf("Exemplar im Zulauf trägt ein Zugangsdatum (%s) — bestellt ist nicht zugegangen", tag)
	}

	// 2. Tür „Wareneingang": Der Zugang ist der Tag der Lieferung.
	geliefert := testdaten.LegeExemplarImZulaufAn(t, pool, titelID, "ZDT-WARENEINGANG")
	if _, err := service.BucheWareneingang(ctx, pool, repository.NewProtokollAbfragen(pool), service.WareneingangAngaben{
		ExemplarIDs: []string{geliefert}, BearbeiterID: "", IPAdresse: "",
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
	freigegeben := testdaten.LegeExemplarImZulaufAn(t, pool, titelID, "ZDT-FREIGABE")
	bookRepo := repository.NewKatalogAbfragen(pool)
	if err := bookRepo.AendereExemplarStatus(ctx, freigegeben, true, false, "", nil, ""); err != nil {
		t.Fatalf("Freigeben: %v", err)
	}
	if tag := zugangsdatum(t, pool, freigegeben); tag == nil || !gleicherTag(*tag, heute) {
		t.Fatalf("Zugang nach Freigabe: %v", tag)
	}

	// 4. Ein bestelltes Exemplar, das nie ankommt und ausgebucht wird, ist kein Zugang.
	nieGekommen := testdaten.LegeExemplarImZulaufAn(t, pool, titelID, "ZDT-NIE")
	if err := bookRepo.AendereExemplarStatus(ctx, nieGekommen, false, true, "", nil, testdaten.LegeBearbeiterAn(t, pool)); err != nil {
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
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Altbestands-Titel", "Zdt 2", 0)

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
