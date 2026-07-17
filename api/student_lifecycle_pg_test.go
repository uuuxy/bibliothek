package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// TestRestoreStudent_HebtLoeschSperreAuf sichert #2 ab: Nach dem Wiederherstellen aus
// dem Papierkorb darf die beim Löschen gesetzte Sperre nicht bestehen bleiben (Zombie-
// Sperre) — sonst kann der Schüler nichts ausleihen.
func TestRestoreStudent_HebtLoeschSperreAuf(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	var id string
	if err := pool.QueryRow(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr,
		                       deleted_at, ist_gesperrt, block_reason)
		 VALUES ('R-1', 'Max', 'Muster', '7a', 2030, now(), true, 'Systematisch gelöscht')
		 RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("gelöschten Schüler anlegen: %v", err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	req := httptest.NewRequest(http.MethodPost, "/api/schueler/deleted/"+id+"/restore", nil)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.RestoreStudentHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}

	var deletedAt *string
	var gesperrt bool
	var grund *string
	if err := pool.QueryRow(ctx,
		`SELECT deleted_at::text, ist_gesperrt, block_reason FROM schueler WHERE id = $1`, id).
		Scan(&deletedAt, &gesperrt, &grund); err != nil {
		t.Fatal(err)
	}
	if deletedAt != nil {
		t.Error("deleted_at wurde nicht zurückgenommen")
	}
	if gesperrt {
		t.Error("Schüler bleibt nach Restore gesperrt (Zombie-Sperre)")
	}
	if grund != nil {
		t.Errorf("block_reason nicht geräumt: %q", *grund)
	}
}

// TestRestoreStudent_FremdeSperreBleibt: Eine Sperre aus ANDEREM Grund (nicht das
// Löschen) muss ein Restore überstehen.
func TestRestoreStudent_FremdeSperreBleibt(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	var id string
	if err := pool.QueryRow(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr,
		                       deleted_at, ist_gesperrt, block_reason)
		 VALUES ('R-2', 'Eva', 'Sperr', '8b', 2030, now(), true, 'Zu viele Überfällige')
		 RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.RestoreStudentHandler().ServeHTTP(rec, req)

	var gesperrt bool
	if err := pool.QueryRow(ctx, `SELECT ist_gesperrt FROM schueler WHERE id = $1`, id).Scan(&gesperrt); err != nil {
		t.Fatal(err)
	}
	if !gesperrt {
		t.Error("eine fremde Sperre wurde beim Restore fälschlich aufgehoben")
	}
}

// TestLusdRueckkehrer_NameUndStatusZurueckgesetzt sichert #3 ab: Kehrt ein zuvor
// anonymisierter Abgänger im LUSD-Export zurück, muss sein echter Name übernommen und
// der Abgänger-/Anonymisierungs-Status aufgehoben werden.
func TestLusdRueckkehrer_NameUndStatusZurueckgesetzt(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	var id string
	if err := pool.QueryRow(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id,
		                       ist_abgaenger, ist_gesperrt)
		 VALUES ('L-1', 'Abgänger', 'Anonymisiert-alt', 'ABG', 2030, 'LUSD-42', true, true)
		 RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SafeRollback(ctx, tx)

	rec := parsedStudentRow{LusdID: "LUSD-42", Vorname: "Lena", Nachname: "Zurück", Klasse: "EF"}
	if err := aktualisiereBestandsschueler(ctx, tx, rec, id); err != nil {
		t.Fatalf("aktualisiereBestandsschueler: %v", err)
	}

	var vorname, nachname, klasse string
	var abgaenger, gesperrt bool
	if err := tx.QueryRow(ctx,
		`SELECT vorname, nachname, klasse, ist_abgaenger, ist_gesperrt FROM schueler WHERE id = $1`, id).
		Scan(&vorname, &nachname, &klasse, &abgaenger, &gesperrt); err != nil {
		t.Fatal(err)
	}
	if vorname != "Lena" || nachname != "Zurück" {
		t.Errorf("Name nicht übernommen: %q %q (blieb anonymisiert)", vorname, nachname)
	}
	if klasse != "EF" {
		t.Errorf("Klasse nicht aktualisiert: %q", klasse)
	}
	if abgaenger {
		t.Error("ist_abgaenger nicht zurückgesetzt")
	}
	if gesperrt {
		t.Error("Anonymisierungs-Sperre nicht aufgehoben")
	}
}

// TestPurgeStudent_AnonymisiertUndLoescht sichert #1 ab: Das endgültige Löschen aus dem
// Papierkorb entfernt den Schüler-Datensatz UND anonymisiert die Ausleihhistorie
// (schueler_id = NULL) — die PII verschwindet wirklich, nicht nur der Papierkorb-Eintrag.
func TestPurgeStudent_AnonymisiertUndLoescht(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// Schüler im Papierkorb + ein Titel/Exemplar + eine ZURÜCKGEGEBENE (historische) Ausleihe.
	var sid, tid, eid string
	if err := pool.QueryRow(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, deleted_at)
		 VALUES ('P-1', 'Paul', 'Weg', '9a', 2030, now()) RETURNING id`).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ('X') RETURNING id`).Scan(&tid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'PB-1') RETURNING id`, tid).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	var loanID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist, rueckgabe_am)
		 VALUES ($1, $2, CURRENT_DATE, now()) RETURNING id`, eid, sid).Scan(&loanID); err != nil {
		t.Fatal(err)
	}

	auditRepo := repository.NewAuditRepository(pool)
	if err := auditRepo.PurgeStudent(ctx, sid, ""); err != nil {
		t.Fatalf("PurgeStudent: %v", err)
	}

	// Schüler-Datensatz weg.
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schueler WHERE id = $1)`, sid).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("Schüler-Datensatz nicht gelöscht")
	}
	// Historische Ausleihe anonymisiert (bleibt für Statistik, aber ohne Personenbezug).
	var schuelerRef *string
	if err := pool.QueryRow(ctx, `SELECT schueler_id FROM ausleihen WHERE id = $1`, loanID).Scan(&schuelerRef); err != nil {
		t.Fatalf("Ausleihe sollte erhalten bleiben: %v", err)
	}
	if schuelerRef != nil {
		t.Error("Ausleihhistorie nicht anonymisiert (schueler_id noch gesetzt)")
	}
}

// TestPurgeStudent_OffeneAusleiheBlockiert: Ein Schüler mit offener Ausleihe darf nicht
// endgültig gelöscht werden (das Buch ist noch draußen).
func TestPurgeStudent_OffeneAusleiheBlockiert(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	var sid, tid, eid string
	if err := pool.QueryRow(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, deleted_at)
		 VALUES ('P-2', 'Offen', 'Leihe', '9b', 2030, now()) RETURNING id`).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ('Y') RETURNING id`).Scan(&tid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'PB-2') RETURNING id`, tid).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist) VALUES ($1, $2, CURRENT_DATE)`, eid, sid); err != nil {
		t.Fatal(err)
	}

	auditRepo := repository.NewAuditRepository(pool)
	if err := auditRepo.PurgeStudent(ctx, sid, ""); err == nil {
		t.Error("Purge trotz offener Ausleihe durchgelaufen — Buch wäre verwaist")
	}

	// Schüler muss noch da sein (Transaktion zurückgerollt).
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schueler WHERE id = $1)`, sid).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("Schüler wurde trotz Blockade gelöscht")
	}
}
