package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Ein manuell gesperrter Schüler muss aus dem Papierkorb zurückkommen — mit seinem Grund.
//
// Fund des Komplett-Durchgangs 31.08.2026: DeleteStudent überschrieb block_reason
// BEDINGUNGSLOS mit 'Systematisch gelöscht' — als einziger Schreiber im Baum; alle
// anderen (LUSD-Abgang, Versetzung, Sperr-Endpunkt) bewahren einen bestehenden Grund
// per COALESCE(NULLIF(...)). Folge: Der echte Sperrgrund („Ausweismissbrauch, Eltern
// informiert") war nach dem Löschen weg, und der Restore erkannte die Zeile an seinem
// Marker als Lösch-Sperre: ist_gesperrt=false, block_reason=NULL — während
// is_manually_blocked=true stehen blieb. Das verletzt chk_schueler_block_reason
// (gesperrt ⇒ Grund nicht leer) → 23514 → 500 „interner Datenbankfehler". Ein manuell
// gesperrter Schüler ließ sich nach dem Löschen NIE wiederherstellen.
func TestPapierkorb_ManuelleSperreUeberlebtLoeschenUndRestore(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	auditRepo := repository.NewAuditRepository(pool)
	bearbeiter := seedPortalLehrkraft(t, pool, "papierkorb@test.invalid")

	gesperrt := seedSchueler(t, pool, "S-SPERRE-1", "Mia", "5a")
	if _, err := pool.Exec(ctx, `
		UPDATE schueler SET is_manually_blocked = true, ist_gesperrt = true,
		       block_reason = 'Ausweismissbrauch, Eltern informiert' WHERE id = $1`, gesperrt); err != nil {
		t.Fatalf("manuelle Sperre setzen: %v", err)
	}
	frei := seedSchueler(t, pool, "S-SPERRE-2", "Ben", "5a")

	loeschen := func(id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodDelete, "/api/schueler/"+id, nil)
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: bearbeiter}))
		rec := httptest.NewRecorder()
		srv.DeleteStudentHandler(auditRepo)(rec, req)
		return rec
	}
	wiederherstellen := func(id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/schueler/"+id+"/restore", nil)
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		srv.RestoreStudentHandler()(rec, req)
		return rec
	}

	// 1. Manuell gesperrter Schüler: Der Grund überlebt das Löschen.
	if rec := loeschen(gesperrt); rec.Code != http.StatusOK {
		t.Fatalf("löschen (gesperrt): HTTP %d: %s", rec.Code, rec.Body.String())
	}
	if grund := sperrGrund(t, pool, gesperrt); grund != "Ausweismissbrauch, Eltern informiert" {
		t.Errorf("Sperrgrund nach dem Löschen: %q — der echte Grund ist unwiederbringlich weg", grund)
	}

	// 2. … und die Wiederherstellung gelingt, mit erhaltener Sperre.
	if rec := wiederherstellen(gesperrt); rec.Code != http.StatusOK {
		t.Fatalf("restore (gesperrt): HTTP %d: %s — ein manuell gesperrter Schüler kommt nie zurück",
			rec.Code, rec.Body.String())
	}
	sperre, manuell, grund := sperrZustand(t, pool, gesperrt)
	if !sperre || !manuell || grund != "Ausweismissbrauch, Eltern informiert" {
		t.Errorf("nach Restore: gesperrt=%v manuell=%v grund=%q — die manuelle Sperre muss bestehen bleiben",
			sperre, manuell, grund)
	}

	// 3. Ungesperrter Schüler: unverändertes Verhalten — Lösch-Sperre wird aufgehoben.
	if rec := loeschen(frei); rec.Code != http.StatusOK {
		t.Fatalf("löschen (frei): HTTP %d: %s", rec.Code, rec.Body.String())
	}
	if rec := wiederherstellen(frei); rec.Code != http.StatusOK {
		t.Fatalf("restore (frei): HTTP %d: %s", rec.Code, rec.Body.String())
	}
	if sperre, manuell, grund := sperrZustand(t, pool, frei); sperre || manuell || grund != "" {
		t.Errorf("nach Restore (frei): gesperrt=%v manuell=%v grund=%q — die Lösch-Sperre muss weg sein",
			sperre, manuell, grund)
	}
}

func sperrGrund(t *testing.T, pool *pgxpool.Pool, id string) string {
	t.Helper()
	_, _, grund := sperrZustand(t, pool, id)
	return grund
}

func sperrZustand(t *testing.T, pool *pgxpool.Pool, id string) (gesperrt, manuell bool, grund string) {
	t.Helper()
	if err := pool.QueryRow(context.Background(),
		`SELECT ist_gesperrt, COALESCE(is_manually_blocked, false), COALESCE(block_reason, '')
		 FROM schueler WHERE id = $1`, id).Scan(&gesperrt, &manuell, &grund); err != nil {
		t.Fatalf("Sperrzustand lesen: %v", err)
	}
	return gesperrt, manuell, grund
}

// Ein zweites Löschen darf die Lösch-Uhr nicht neu stellen.
//
// Fund aus dem Bestands-Durchgang 10.09.2026 (Register): DeleteStudent schrieb
// `deleted_at = CURRENT_TIMESTAMP` bedingungslos, und geprüft wurde davor nur, DASS es
// den Schüler gibt — nicht, ob er schon im Papierkorb liegt. Wer eine Zeile ein zweites
// Mal löschte, verschob damit die 180-Tage-Frist der Anonymisierung
// (repository.PredikatAnonymisierung liest deleted_at) um die ganze bereits abgelaufene
// Zeit nach hinten. Still: Die Antwort war beide Male „success", und im Papierkorb sieht
// man das Datum nur als „gelöscht am".
func TestPapierkorb_ZweitesLoeschenVerschiebtDieUhrNicht(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	auditRepo := repository.NewAuditRepository(pool)
	bearbeiter := seedPortalLehrkraft(t, pool, "papierkorb-uhr@test.invalid")
	schueler := seedSchueler(t, pool, "S-UHR-1", "Uri", "5a")

	loeschen := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodDelete, "/api/schueler/"+schueler, nil)
		req.SetPathValue("id", schueler)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: bearbeiter}))
		rec := httptest.NewRecorder()
		srv.DeleteStudentHandler(auditRepo)(rec, req)
		return rec
	}
	geloeschtAm := func() time.Time {
		t.Helper()
		var wann time.Time
		if err := pool.QueryRow(ctx, `SELECT deleted_at FROM schueler WHERE id = $1`, schueler).Scan(&wann); err != nil {
			t.Fatalf("deleted_at lesen: %v", err)
		}
		return wann
	}

	if rec := loeschen(); rec.Code != http.StatusOK {
		t.Fatalf("erstes Löschen: HTTP %d: %s", rec.Code, rec.Body.String())
	}
	// 179 Tage zurückdatieren: Die Zeile steht einen Tag vor der Anonymisierung.
	if _, err := pool.Exec(ctx, `UPDATE schueler SET deleted_at = now() - interval '179 days' WHERE id = $1`, schueler); err != nil {
		t.Fatalf("zurückdatieren: %v", err)
	}
	vorher := geloeschtAm()

	loeschen()

	nachher := geloeschtAm()
	if !nachher.Equal(vorher) {
		t.Errorf("die Lösch-Uhr wurde neu gestellt: %s → %s — die 180-Tage-Frist beginnt von vorn",
			vorher.Format(time.RFC3339), nachher.Format(time.RFC3339))
	}
}
