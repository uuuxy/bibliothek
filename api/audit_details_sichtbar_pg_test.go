package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/db"
)

// Der Rückweg muss am ERGEBNIS begehbar sein, nicht nur im Datensatz existieren.
//
// Am 23.08.2026 hat dieses Projekt eine Protokollspur bekommen: Wird ein Titel gelöscht,
// während ein Exemplar verliehen ist, hält eine audit_log-Zeile fest, WER das Buch hatte
// (inventur/db_books_delete_spur.go). Der Commit versprach, die Rückgabe bleibe damit
// „nachschlagbar".
//
// Sie war es nicht: Der Endpunkt /api/audit lieferte die Spalte `details` überhaupt nicht
// aus. Übrig blieb „DELETE auf ausleihen, Datensatz <UUID>" — und diese UUID gehört zu
// einem Exemplar, das im selben Vorgang gelöscht wurde. Wer das zurückgebrachte Buch auf
// dem Tresen liegen hat, findet damit niemanden.
//
// Der Test geht deshalb durch den HANDLER und sucht die Sachangabe in der ANTWORT.
func TestAuditLogLiefertDieSachangabe(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	marke := fmt.Sprintf("e2e_details_%d", time.Now().UnixNano())
	t.Cleanup(func() { aufraeumen(t, pool, `DELETE FROM audit_log WHERE tabelle = $1`, marke) })

	// Eine Zeile in der Form, die db_books_delete_spur.go schreibt.
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, kontext, details)
		VALUES ($1, 'DELETE', gen_random_uuid(), 'SYSTEM',
		        'Titel gelöscht, Buch war zu diesem Zeitpunkt verliehen',
		        jsonb_build_object('barcode_id','ZB-PROBE','titel','Der Zauberberg',
		                           'entleiher','Hans Castorp'))`, marke); err != nil {
		t.Fatalf("Spur anlegen: %v", err)
	}

	rr := httptest.NewRecorder()
	srv.GetAuditLogsHandler()(rr, httptest.NewRequest(http.MethodGet, "/api/audit", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("Status %d: %s", rr.Code, rr.Body.String())
	}

	var eintraege []AuditLogEntry
	if err := json.Unmarshal(rr.Body.Bytes(), &eintraege); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}

	var gefunden *AuditLogEntry
	for i := range eintraege {
		if eintraege[i].Tabelle == marke {
			gefunden = &eintraege[i]
			break
		}
	}
	if gefunden == nil {
		t.Fatal("die Spur steht nicht in der Antwort")
	}
	// Genau die drei Angaben, mit denen jemand das Buch zuordnen kann.
	for _, muss := range []string{"ZB-PROBE", "Zauberberg", "Hans Castorp"} {
		if !strings.Contains(gefunden.Details, muss) {
			t.Errorf("die Antwort nennt %q nicht — der Eintrag ist nicht nachschlagbar: %q",
				muss, gefunden.Details)
		}
	}
}
