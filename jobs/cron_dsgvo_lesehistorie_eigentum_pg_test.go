package jobs

import (
	"context"
	"os"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Frist der Ausleihhistorie folgt dem Eigentum des Exemplars, nicht dem Merkmal am Titel
// (docs/OFFEN.md 4.26, entschieden am 29.09.2026) — dieselbe Regel wie Etikett, Bestandsbücher
// und Bescheid (repository.ExemplarTopfSQL). Jede Ausleihe ist vor 100 Tagen zurückgekommen:
// Die Frist der Schülerbücherei ist damit abgelaufen, die des Landes (730 Tage) nicht.
//
//	LAND-EX     Roman, Eigentum am Exemplar Land              → bleibt
//	LAND-BEST   Roman, Bestellung aus dem Topf des Landes     → bleibt
//	TRAEGER-EX  Schulbuch, Eigentum am Exemplar Schulträger   → getrennt
//	ROMAN       Roman ohne Eigentum und Bestellung            → getrennt
//	SCHULBUCH   Schulbuch ohne Eigentum und Bestellung        → bleibt
//
// Das Ausleih-Protokoll (audit_log) verliert den Schüler genau dort, wo die Ausleihe ihn
// verliert: Bis zum 29.09.2026 las die Frist ist_lernmittel am Titel, während der Bescheid
// schon dem Eigentum folgte — die Lektüre des Landes ging an die Schulaufsicht und verlor
// ihren Ausleiher nach 90 Tagen.
func TestLesehistorieBefristung_KlasseFolgtDemEigentum(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "lesehisteig")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	zeile := func(ziel any, sql string, args ...any) {
		t.Helper()
		if err := pool.QueryRow(ctx, sql, args...).Scan(ziel); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	var schuelerID, roman, schulbuch, bestellung string
	zeile(&schuelerID, `SELECT id FROM schueler WHERE barcode_id = 'S-DRILL-1'`)
	zeile(&roman, `INSERT INTO buecher_titel (titel, signatur) VALUES ('Tintenherz', 'JF Fun') RETURNING id`)
	zeile(&schulbuch, `INSERT INTO buecher_titel (titel, signatur, ist_lernmittel)
		VALUES ('Deutschbuch 8', 'LMF-Deutsch 8', true) RETURNING id`)
	zeile(&bestellung, `INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email, mittel)
		VALUES ('Buchhandlung', 'bestellung@example.org', 'land') RETURNING id`)

	// exemplar legt ein Exemplar an; eigentum und bestellungID leer = nicht gesetzt.
	exemplar := func(titel, barcode, eigentum, bestellungID string) string {
		t.Helper()
		var id string
		zeile(&id, `INSERT INTO buecher_exemplare (titel_id, barcode_id, eigentum, eigentum_quelle, bestellung_id)
			VALUES ($1, $2, NULLIF($3, ''), CASE WHEN $3 = '' THEN NULL ELSE 'hand' END, NULLIF($4, '')::uuid)
			RETURNING id`, titel, barcode, eigentum, bestellungID)
		return id
	}
	faelle := map[string]string{
		"LAND-EX":    exemplar(roman, "E-LAND-1", "land", ""),
		"LAND-BEST":  exemplar(roman, "E-LAND-2", "", bestellung),
		"TRAEGER-EX": exemplar(schulbuch, "E-TRAEGER-1", "schultraeger", ""),
		"ROMAN":      exemplar(roman, "E-ROMAN-1", "", ""),
		"SCHULBUCH":  exemplar(schulbuch, "E-SCHUL-1", "", ""),
	}
	ausleihe := map[string]string{}
	for name, exemplarID := range faelle {
		var id string
		zeile(&id, `INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
			VALUES ($1, $2, NOW() - interval '130 days', NOW() - interval '109 days', NOW() - interval '100 days')
			RETURNING id`, exemplarID, schuelerID)
		ausleihe[name] = id
		if _, err := pool.Exec(ctx, `INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, details, timestamp)
			VALUES ('ausleihen', 'RETURN', $1::uuid, 'USER',
			        jsonb_build_object('exemplar_id', $1::text, 'schueler_id', $2::text),
			        NOW() - interval '100 days')`, exemplarID, schuelerID); err != nil {
			t.Fatalf("Protokoll %s: %v", name, err)
		}
	}

	NewScheduler(pool, repository.NewAuditRepository(pool)).RunLesehistorieBefristung()

	erwartung := map[string]bool{ // true = der Schüler bleibt zugeordnet
		"LAND-EX": true, "LAND-BEST": true, "TRAEGER-EX": false, "ROMAN": false, "SCHULBUCH": true,
	}
	for name, bleibt := range erwartung {
		var ausleiheHat, protokollHat bool
		zeile(&ausleiheHat, `SELECT schueler_id IS NOT NULL FROM ausleihen WHERE id = $1`, ausleihe[name])
		zeile(&protokollHat, `SELECT count(*) > 0 FROM audit_log
			WHERE tabelle = 'ausleihen' AND datensatz_id = $1 AND details ? 'schueler_id'`, faelle[name])
		if ausleiheHat != bleibt {
			t.Errorf("%s: Ausleihe trägt den Schüler = %v, erwartet %v", name, ausleiheHat, bleibt)
		}
		if protokollHat != bleibt {
			t.Errorf("%s: Protokoll trägt den Schüler = %v, erwartet %v", name, protokollHat, bleibt)
		}
	}
}
