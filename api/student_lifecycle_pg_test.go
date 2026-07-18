package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	// Anonymisierter Abgänger, wie ihn anonymisiereAbgaenger hinterlässt: gesperrt MIT
	// Grund (chk_schueler_block_reason verlangt ihn).
	if err := pool.QueryRow(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id,
		                       ist_abgaenger, ist_gesperrt, block_reason)
		 VALUES ('L-1', 'Abgänger', 'Anonymisiert-alt', 'ABG', 2030, 'LUSD-42', true, true, 'Abgänger anonymisiert')
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
	var grund *string
	if err := tx.QueryRow(ctx,
		`SELECT vorname, nachname, klasse, ist_abgaenger, ist_gesperrt, block_reason FROM schueler WHERE id = $1`, id).
		Scan(&vorname, &nachname, &klasse, &abgaenger, &gesperrt, &grund); err != nil {
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
	if grund != nil {
		t.Errorf("block_reason nicht geräumt: %q (stehen gebliebener Anonymisierungs-Grund)", *grund)
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

// TestAbgaengerRetentionKette sichert die DSGVO-Kette ab, die sonst nie schliesst:
// Ein Abgänger mit offenem Buch wird gesperrt (Name bleibt) UND bekommt sein
// abgaenger_jahr aufs Abgangsjahr — nur so erfasst ihn der Cronjob später. Nach der
// Buchrückgabe entfernt PurgeAbgaenger ihn endgültig (PII weg).
func TestAbgaengerRetentionKette(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// Schüler mit lusd_id + abgaenger_jahr in der ZUKUNFT (Default vom Anlegen),
	// plus ein offenes Buch.
	zukunft := time.Now().Year() + 5
	var sid, tid, eid string
	if err := pool.QueryRow(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id)
		 VALUES ('A-1', 'Alt', 'Schüler', '10a', $1, 'LUSD-99') RETURNING id`, zukunft).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ('Z') RETURNING id`).Scan(&tid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'AB-1') RETURNING id`, tid).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	var loanID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist) VALUES ($1, $2, CURRENT_DATE) RETURNING id`,
		eid, sid).Scan(&loanID); err != nil {
		t.Fatal(err)
	}

	// Abgang mit offenem Buch -> sperreAbgaenger (über eine Tx, wie im Sync).
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := sperreAbgaenger(ctx, tx, sid); err != nil {
		t.Fatalf("sperreAbgaenger: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	// abgaenger_jahr muss jetzt das AKTUELLE Jahr sein (nicht mehr die Zukunft) —
	// sonst würde der Cronjob-Filter (abgaenger_jahr < cutoff) nie greifen.
	var jahr int
	var gesperrt bool
	var vorname string
	if err := pool.QueryRow(ctx, `SELECT abgaenger_jahr, ist_gesperrt, vorname FROM schueler WHERE id = $1`, sid).
		Scan(&jahr, &gesperrt, &vorname); err != nil {
		t.Fatal(err)
	}
	if jahr != time.Now().Year() {
		t.Errorf("abgaenger_jahr: erwartet %d (Abgangsjahr), war %d — Cronjob würde ihn nie erfassen", time.Now().Year(), jahr)
	}
	if !gesperrt || vorname != "Alt" {
		t.Errorf("Abgänger mit offenem Buch soll gesperrt sein, Name behalten: gesperrt=%v vorname=%q", gesperrt, vorname)
	}

	// PurgeAbgaenger ist blockiert, solange das Buch offen ist.
	auditRepo := repository.NewAuditRepository(pool)
	if err := auditRepo.PurgeAbgaenger(ctx, sid, ""); err == nil {
		t.Error("PurgeAbgaenger trotz offenem Buch durchgelaufen")
	}

	// Buch zurückgeben -> jetzt entfernt PurgeAbgaenger den Datensatz endgültig.
	if _, err := pool.Exec(ctx, `UPDATE ausleihen SET rueckgabe_am = now() WHERE id = $1`, loanID); err != nil {
		t.Fatal(err)
	}
	if err := auditRepo.PurgeAbgaenger(ctx, sid, ""); err != nil {
		t.Fatalf("PurgeAbgaenger nach Rückgabe: %v", err)
	}
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schueler WHERE id = $1)`, sid).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("Abgänger nach Purge nicht gelöscht (PII bliebe für immer)")
	}
}
