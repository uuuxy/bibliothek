package service

import (
	"testing"

	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v4"
)

// Der UPDATE…RETURNING-Kern von BulkReceiveOrder (Regex für pgxmock).
const bulkReceiveQuery = `UPDATE buecher_exemplare e\s+SET ist_ausleihbar = true, zustand_notiz = ''\s+FROM buecher_titel t\s+WHERE e\.titel_id = t\.id\s+AND e\.ist_ausleihbar = false\s+AND e\.id = ANY\(\$1\)\s+RETURNING e\.barcode_id, t\.titel, coalesce\(t\.autor, ''\) AS autor, e\.etikett_gedruckt`

func TestBulkReceiveOrder_ReturnsReceivedItemsWithEtikettStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	ids := []string{"ex-1", "ex-2"}

	mock.ExpectQuery(bulkReceiveQuery).
		WithArgs(ids).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "titel", "autor", "etikett_gedruckt"}).
			AddRow("B-100", "Faust", "Goethe", true).
			AddRow("B-101", "Die Verwandlung", "Kafka", false))

	// Audit-Log-Insert (Anzahl = 2)
	mock.ExpectExec(`INSERT INTO audit_logs`).
		WithArgs(pgxmock.AnyArg(), "BULK_RECEIVE_ITEMS", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	items, err := BulkReceiveOrder(t.Context(), mock, repository.NewAuditRepository(mock), ids, "admin-1", "127.0.0.1")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("erwartet 2 Exemplare, bekam %d", len(items))
	}
	// Das Frontend baut aus etikett_gedruckt=false die Druckempfehlung —
	// dieses Feld muss den DB-Wert exakt durchreichen.
	if items[0].EtikettGedruckt != true || items[1].EtikettGedruckt != false {
		t.Errorf("etikett_gedruckt falsch durchgereicht: %+v", items)
	}
	if items[1].Titel != "Die Verwandlung" || items[1].Autor != "Kafka" || items[1].BarcodeID != "B-101" {
		t.Errorf("Titeldaten falsch gemappt: %+v", items[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestBulkReceiveOrder_NoMatchingCopiesReturnsError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	ids := []string{"bereits-freigegeben"}

	mock.ExpectQuery(bulkReceiveQuery).
		WithArgs(ids).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "titel", "autor", "etikett_gedruckt"}))

	items, err := BulkReceiveOrder(t.Context(), mock, repository.NewAuditRepository(mock), ids, "admin-1", "127.0.0.1")
	if err == nil {
		t.Fatalf("erwartet Fehler bei leerem Ergebnis, bekam Items: %+v", items)
	}
	// Der Handler mappt exakt diese Meldung auf 404 — Wortlaut ist Vertrag.
	if err.Error() != "keine zu aktualisierenden Exemplare gefunden (bereits freigegeben?)" {
		t.Errorf("unerwartete Fehlermeldung: %q", err.Error())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}
