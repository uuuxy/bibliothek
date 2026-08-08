package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func TestGetIncomingShipments_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	// Two days
	date1 := now
	date2 := now.Add(-24 * time.Hour)

	// Rows: exemplarID, titelID, erstelltAm, zustandNotiz, titel, isbn, coverURL
	rows := pgxmock.NewRows([]string{"id", "titel_id", "erstellt_am", "zustand_notiz", "titel", "isbn", "cover_url"}).
		AddRow("ex1", "t1", date1, "Im Zulauf - Lieferant A", "Titel 1", "123", "cover1").
		AddRow("ex2", "t1", date1, "Im Zulauf - Lieferant A", "Titel 1", "123", "cover1").     // same group and item
		AddRow("ex3", "t2", date1, "bestellt", "Titel 2", "456", "cover2").                    // different group (supplier)
		AddRow("ex4", "t3", date2, "Bestellt (Lieferanten-Vorab-Barcode)", "Titel 3", "", ""). // different group (date and supplier)
		AddRow("ex5", "t4", date2, "andere notiz", "Titel 4", "", "")                          // fallback supplier

	query := `SELECT e\.id, e\.titel_id, e\.erstellt_am, e\.zustand_notiz, t\.titel, COALESCE\(t\.isbn, ''\), \s*COALESCE\(NULLIF\(t\.cover_url, ''\), CASE WHEN t\.isbn IS NOT NULL AND t\.isbn != '' THEN 'https://portal\.dnb\.de/opac/mvb/cover\?isbn=' \|\| replace\(t\.isbn, '-', ''\) ELSE '' END\)\s*FROM buecher_exemplare e\s*JOIN buecher_titel t ON e\.titel_id = t\.id\s*WHERE e\.ist_ausleihbar = false \s*AND \(e\.zustand_notiz LIKE 'Im Zulauf%' OR e\.zustand_notiz = 'bestellt' OR e\.zustand_notiz LIKE 'Bestellt%'\)\s*ORDER BY e\.erstellt_am DESC`

	mock.ExpectQuery(query).WillReturnRows(rows)

	groups, err := GetIncomingShipments(context.Background(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(groups) != 4 {
		t.Fatalf("expected 4 groups, got %d", len(groups))
	}

	// Gruppiert wird nach Datum UND dem aus zustand_notiz abgeleiteten Lieferanten.
	// Die Erwartung steht als Tabelle da, damit auch eine FEHLENDE Gruppe auffällt —
	// eine Schleife mit if/else prüft nur die Gruppen, die tatsächlich kamen.
	type gruppe struct{ datum, lieferant string }
	erwartet := map[gruppe]struct {
		posten int // Titel-Positionen in der Gruppe
		menge  int // Exemplare der ersten Position
	}{
		{date1.Format("02.01.2006"), "Lieferant A"}:                 {posten: 1, menge: 2},
		{date1.Format("02.01.2006"), "Automatische Nachbestellung"}: {posten: 1, menge: 1},
		{date2.Format("02.01.2006"), "Vorab-Barcode Bestellung"}:    {posten: 1, menge: 1},
		{date2.Format("02.01.2006"), "Unbekannter Lieferant"}:       {posten: 1, menge: 1},
	}

	gesehen := make(map[gruppe]bool, len(erwartet))
	for _, g := range groups {
		schluessel := gruppe{g.Date, g.SupplierName}
		soll, bekannt := erwartet[schluessel]
		if !bekannt {
			t.Errorf("unerwartete Gruppe: %s / %s", g.Date, g.SupplierName)
			continue
		}
		gesehen[schluessel] = true
		if len(g.Items) != soll.posten {
			t.Errorf("%s / %s: %d Positionen erwartet, bekam %d", g.Date, g.SupplierName, soll.posten, len(g.Items))
			continue
		}
		if g.Items[0].Menge != soll.menge {
			t.Errorf("%s / %s: Menge %d erwartet, bekam %d", g.Date, g.SupplierName, soll.menge, g.Items[0].Menge)
		}
	}
	for schluessel := range erwartet {
		if !gesehen[schluessel] {
			t.Errorf("Gruppe fehlt vollstaendig: %s / %s", schluessel.datum, schluessel.lieferant)
		}
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

	query := `SELECT e\.id, e\.titel_id, e\.erstellt_am, e\.zustand_notiz, t\.titel, COALESCE\(t\.isbn, ''\), \s*COALESCE\(NULLIF\(t\.cover_url, ''\), CASE WHEN t\.isbn IS NOT NULL AND t\.isbn != '' THEN 'https://portal\.dnb\.de/opac/mvb/cover\?isbn=' \|\| replace\(t\.isbn, '-', ''\) ELSE '' END\)\s*FROM buecher_exemplare e\s*JOIN buecher_titel t ON e\.titel_id = t\.id\s*WHERE e\.ist_ausleihbar = false \s*AND \(e\.zustand_notiz LIKE 'Im Zulauf%' OR e\.zustand_notiz = 'bestellt' OR e\.zustand_notiz LIKE 'Bestellt%'\)\s*ORDER BY e\.erstellt_am DESC`

	expectedErr := errors.New("db error")
	mock.ExpectQuery(query).WillReturnError(expectedErr)

	_, err = GetIncomingShipments(context.Background(), mock)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != expectedErr {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
