package repository_test

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// Beim Löschen eines Titels hinterlässt nur eine offene Forderung eine Spur. Eine bezahlte oder
// stornierte ist erledigt; ein Eintrag „es stand noch eine unbezahlte Forderung offen" behauptete
// sonst eine Schuld, die es nicht gibt, mit Name und Freitext des Lesers. Geprüft über beide
// Löschwege: den einzelnen Titel und die Sammellöschung.
func TestLoescheTitel_NurOffeneForderungHinterlaesstSpur(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	bearbeiter := testdaten.LegeBearbeiterAn(t, pool)
	schaeden := repository.NewSchadensAbfragen(pool)

	schuelerID := testdaten.LegeSchuelerMitNamenAn(t, pool, "TFS-L", "Forderung", "Spurprobe", "08B", 2030)

	// Je Löschweg ein Titel mit drei Exemplaren und je einer Forderung: offen, bezahlt und
	// storniert. Erledigt werden sie über die Türen des Schadensersatzes, damit die Zeilen so
	// aussehen wie im Betrieb (ein Storno setzt auch ist_bezahlt).
	type exemplare struct{ offen, bezahlt, storniert string }
	titelMitForderungen := func(titel, barcode string) (string, exemplare) {
		t.Helper()
		var titelID string
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_titel (titel, autor) VALUES ($1, 'Test') RETURNING id`, titel).Scan(&titelID); err != nil {
			t.Fatalf("Titel anlegen: %v", err)
		}
		forderung := func(suffix string) (exemplarID, schadensfallID string) {
			t.Helper()
			exemplarID = testdaten.LegeExemplarAn(t, pool, titelID, barcode+suffix, true, "")
			if err := pool.QueryRow(ctx,
				`INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag)
				 VALUES ($1, $2, 'Wasserschaden', 12.50) RETURNING id`,
				exemplarID, schuelerID).Scan(&schadensfallID); err != nil {
				t.Fatalf("Forderung anlegen: %v", err)
			}
			return exemplarID, schadensfallID
		}
		var ex exemplare
		ex.offen, _ = forderung("-O")
		var bezahlt, storniert string
		ex.bezahlt, bezahlt = forderung("-B")
		ex.storniert, storniert = forderung("-S")
		if err := schaeden.VerbucheZahlung(ctx, bezahlt, bearbeiter); err != nil {
			t.Fatalf("Zahlung verbuchen: %v", err)
		}
		if err := schaeden.StorniereForderung(ctx, storniert, bearbeiter, "Buch wieder aufgetaucht"); err != nil {
			t.Fatalf("Forderung stornieren: %v", err)
		}
		return titelID, ex
	}
	einzeln, exEinzeln := titelMitForderungen("Forderungsspur einzeln", "TFS-1")
	gesammelt, exGesammelt := titelMitForderungen("Forderungsspur gesammelt", "TFS-2")

	if err := repository.NewKatalogAbfragen(pool).LoescheTitel(ctx, einzeln, bearbeiter); err != nil {
		t.Fatalf("einzelnen Titel löschen: %v", err)
	}
	if err := repository.NewTitelAbfragen(pool).LoescheTitelSamtAusleihen(ctx, []string{gesammelt}, bearbeiter); err != nil {
		t.Fatalf("Sammellöschung: %v", err)
	}

	for _, weg := range []struct {
		name string
		ex   exemplare
	}{{"einzeln", exEinzeln}, {"gesammelt", exGesammelt}} {
		spuren := func(exemplarID string) int {
			t.Helper()
			return pgtest.ZaehleZeilen(t, pool,
				`SELECT count(*) FROM audit_log
				 WHERE tabelle = 'schadensfaelle' AND aktion = 'DELETE' AND datensatz_id = $1`, exemplarID)
		}
		// Positivkontrolle: Ohne die Spur der offenen Forderung wären die Prüfungen danach grün,
		// ohne etwas zu prüfen.
		if n := spuren(weg.ex.offen); n != 1 {
			t.Errorf("%s: %d Forderungsspuren für die offene Forderung, erwartet 1", weg.name, n)
		}
		// Am Merker erkennt das Auskunftsblatt den Vorgang; ohne ihn stünde dort „DELETE (schadensfaelle)".
		if n := pgtest.ZaehleZeilen(t, pool, `
			SELECT count(*) FROM audit_log
			WHERE tabelle = 'schadensfaelle' AND aktion = 'DELETE' AND datensatz_id = $1
			  AND details->>'action' = $2`, weg.ex.offen, repository.ProtokollAktionTitelMitForderung); n != 1 {
			t.Errorf("%s: Die Forderungsspur trägt den Merker %q nicht", weg.name, repository.ProtokollAktionTitelMitForderung)
		}
		if n := spuren(weg.ex.bezahlt); n != 0 {
			t.Errorf("%s: %d Forderungsspuren für eine bezahlte Forderung, erwartet 0", weg.name, n)
		}
		if n := spuren(weg.ex.storniert); n != 0 {
			t.Errorf("%s: %d Forderungsspuren für eine stornierte Forderung, erwartet 0", weg.name, n)
		}
	}
}
