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

	// Verify groupings and mappings
	// Sorting is descending by date
	// First two groups should have date1, last two should have date2
	for _, g := range groups {
		if g.Date == date1.Format("02.01.2006") {
			if g.SupplierName == "Lieferant A" {
				if len(g.Items) != 1 {
					t.Errorf("expected 1 item for Lieferant A group, got %d", len(g.Items))
				}
				if g.Items[0].Menge != 2 {
					t.Errorf("expected Menge 2, got %d", g.Items[0].Menge)
				}
			} else if g.SupplierName == "Automatische Nachbestellung" {
				if len(g.Items) != 1 {
					t.Errorf("expected 1 item for Automatische Nachbestellung, got %d", len(g.Items))
				}
			} else {
				t.Errorf("unexpected supplier for date1: %s", g.SupplierName)
			}
		} else if g.Date == date2.Format("02.01.2006") {
			if g.SupplierName == "Vorab-Barcode Bestellung" {
				if len(g.Items) != 1 {
					t.Errorf("expected 1 item, got %d", len(g.Items))
				}
			} else if g.SupplierName == "Unbekannter Lieferant" {
				if len(g.Items) != 1 {
					t.Errorf("expected 1 item, got %d", len(g.Items))
				}
			} else {
				t.Errorf("unexpected supplier for date2: %s", g.SupplierName)
			}
		} else {
			t.Errorf("unexpected date: %s", g.Date)
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
