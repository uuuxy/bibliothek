package api

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// Die Rechnung an die Eltern darf keine Forderung verlieren, weil ein Bezug fehlt
// (Bestands-Durchgang 06.09.2026). `schadensfaelle.exemplar_id` und `.ausleihe_id` sind
// nullbar — Letzteres steht auf ON DELETE SET NULL —, und `geraet_id` wartet als dritte
// Bezugsspalte. Mit den früheren INNER JOINs fiel so eine Forderung lautlos aus dem
// Brief: Der Betrag war zu klein, oder es hieß „keine offenen Schadensfälle", während
// die Akte offene Beträge zeigte und der Schüler gesperrt blieb.
func TestQueryRechnungItems_ForderungOhneExemplarUndAusleihe(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	var schuelerID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 VALUES ('R-RECH', 'Rita', 'Rechnung', '07H1', 2030) RETURNING id`).Scan(&schuelerID); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	// Ein Geräteschaden: exemplar_id NULL, ausleihe_id NULL — genau die Form, die
	// check_damage_item ausdrücklich erlaubt und die den alten INNER JOINs entging.
	var geraetID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO geraete (modellname, barcode_id) VALUES ('iPad 9. Gen.', 'G-RECH-1') RETURNING id`).
		Scan(&geraetID); err != nil {
		t.Fatalf("Gerät anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO schadensfaelle (schueler_id, geraet_id, beschreibung, betrag)
		 VALUES ($1, $2, 'Kopfhörerbuchse abgebrochen', 24.90)`, schuelerID, geraetID); err != nil {
		t.Fatalf("Forderung ohne Exemplar/Ausleihe anlegen: %v", err)
	}

	items, err := queryRechnungItems(ctx, pool, schuelerID)
	if err != nil {
		t.Fatalf("Rechnungspositionen lesen: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("erwartet 1 Position, waren %d — eine Forderung ohne Exemplar/Ausleihe fällt "+
			"aus der Rechnung, und bei „alle betroffen\" antwortet der Weg mit 404", len(items))
	}
	if items[0].Titel != "iPad 9. Gen." {
		t.Errorf("Bezeichnung: erwartet den Modellnamen des Geräts, war %q", items[0].Titel)
	}
	if items[0].Barcode != "G-RECH-1" {
		t.Errorf("Barcode: erwartet den Geräte-Barcode, war %q", items[0].Barcode)
	}
	if items[0].Ersatzpreis != 24.90 {
		t.Errorf("Betrag: erwartet 24.90, war %v", items[0].Ersatzpreis)
	}
	if items[0].Ausleihdatum.IsZero() {
		t.Error("Ausleihdatum leer — ersatzweise gilt das Datum der Forderung")
	}
}
