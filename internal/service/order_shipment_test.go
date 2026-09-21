package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v5"
)

// Spalten der Zulauf-Abfrage: die sieben alten plus bestellung_id, lieferant_name,
// bestelldatum (NULL bei Altbestand ohne Bestellung).
var zulaufSpalten = []string{"id", "titel_id", "erstellt_am", "zustand_notiz", "titel", "isbn", "cover_url",
	"bestellung_id", "lieferant_name", "bestelldatum"}

const zulaufAbfrage = `LEFT JOIN bestellungen_verlauf b ON b\.id = e\.bestellung_id\s*WHERE e\.ist_ausleihbar = false\s*AND e\.bestellstatus IS NOT NULL\s*AND e\.ist_ausgesondert = false\s*ORDER BY e\.erstellt_am DESC`

func TestGetIncomingShipments_GruppiertNachBestellung(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	// 22:30 UTC am 29.09. ist in Friedrichsdorf schon der 30.09. — der Kalendertag der
	// Schule steht in der Gruppe, nicht der des Servers.
	bestellt := time.Date(2026, 9, 29, 22, 30, 0, 0, time.UTC)
	alt := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	haendler := "Buchhandlung Probe"

	rows := pgxmock.NewRows(zulaufSpalten).
		// Zwei Exemplare derselben Bestellung und desselben Titels: eine Gruppe, eine Position, Menge 2.
		AddRow("ex1", "t1", bestellt, "Im Zulauf - "+haendler, "Titel 1", "123", "cover1", ptr("B1"), &haendler, &bestellt).
		AddRow("ex2", "t1", bestellt, "Im Zulauf - "+haendler, "Titel 1", "123", "cover1", ptr("B1"), &haendler, &bestellt).
		// Zweiter Topf am selben Tag beim selben Händler: eigene Gruppe (bis 21.09.2026 dieselbe).
		AddRow("ex3", "t2", bestellt, "Bestellt (ohne Vorab-Barcode) - "+haendler, "Titel 2", "456", "cover2", ptr("B2"), &haendler, &bestellt).
		// Altbestand ohne Bestellung (vor Migration 063): der Name kommt aus der Notiz.
		AddRow("ex4", "t3", alt, "Im Zulauf - Lieferant A", "Titel 3", "", "", nil, nil, nil).
		AddRow("ex5", "t4", alt, "andere notiz", "Titel 4", "", "", nil, nil, nil)

	mock.ExpectQuery(zulaufAbfrage).WillReturnRows(rows)

	groups, err := GetIncomingShipments(context.Background(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Die Erwartung steht als Tabelle da, damit auch eine FEHLENDE Gruppe auffällt —
	// eine Schleife mit if/else prüft nur die Gruppen, die tatsächlich kamen.
	erwartet := map[string]struct {
		datum, lieferant string
		posten           int // Titel-Positionen in der Gruppe
		menge            int // Exemplare der ersten Position
	}{
		"B1":                               {"30.09.2026", haendler, 1, 2},
		"B2":                               {"30.09.2026", haendler, 1, 1},
		"01.01.2024|Lieferant A":           {"01.01.2024", "Lieferant A", 1, 1},
		"01.01.2024|Unbekannter Lieferant": {"01.01.2024", "Unbekannter Lieferant", 1, 1},
	}
	if len(groups) != len(erwartet) {
		t.Fatalf("expected %d groups, got %d", len(erwartet), len(groups))
	}
	gesehen := map[string]bool{}
	for _, g := range groups {
		soll, bekannt := erwartet[g.ID]
		if !bekannt {
			t.Errorf("unerwartete Gruppe %q (%s / %s)", g.ID, g.Date, g.SupplierName)
			continue
		}
		gesehen[g.ID] = true
		if g.Date != soll.datum || g.SupplierName != soll.lieferant {
			t.Errorf("%s: %s / %s, erwartet %s / %s", g.ID, g.Date, g.SupplierName, soll.datum, soll.lieferant)
		}
		if len(g.Items) != soll.posten {
			t.Errorf("%s: %d Positionen erwartet, bekam %d", g.ID, soll.posten, len(g.Items))
			continue
		}
		if g.Items[0].Menge != soll.menge {
			t.Errorf("%s: Menge %d erwartet, bekam %d", g.ID, soll.menge, g.Items[0].Menge)
		}
	}
	for id := range erwartet {
		if !gesehen[id] {
			t.Errorf("Gruppe fehlt vollständig: %s", id)
		}
	}
	// Neueste Bestellung zuerst.
	if groups[0].ID != "B1" && groups[0].ID != "B2" {
		t.Errorf("neueste Gruppe zuerst erwartet, oben steht %q", groups[0].ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetIncomingShipments_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	expectedErr := errors.New("db error")
	mock.ExpectQuery(zulaufAbfrage).WillReturnError(expectedErr)

	_, err = GetIncomingShipments(context.Background(), mock)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
