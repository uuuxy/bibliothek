package api

import (
	"context"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Löschsperre: Solange ein Buch draußen ist oder eine Gebühr offen, verschwindet
// niemand — weder in den Papierkorb noch endgültig, weder von Hand noch nachts.
//
// Es gibt FÜNF Türen zu diesem Zustand, und jede hat ihren eigenen Wächter:
//
//	Soft-Delete (Papierkorb, Oberfläche) → pruefeSchuelerLoeschbar
//	PurgeStudent (endgültig, von Hand)   → blockiereBeiOffenenVorgaengen
//	PurgeAbgaenger (Cronjob, nachts)     → blockiereBeiOffenenVorgaengen
//	Anonymisierungs-Cronjob              → PredikatAnonymisierung (jobs-Paket getestet)
//	LUSD-Import                          → sperreAbgaenger statt anonymisieren
//
// Belegt war davon bisher genau eine (PurgeStudent bei offener Ausleihe). Für die
// übrigen stand der Schutz teils NUR IM KOMMENTAR — „unbezahlte würden die Löschung
// blockieren" in jobs/cron_dsgvo_abgaenger_pg_test.go. Genau diese Sorte Zusage hat in
// diesem Projekt schon einmal nicht gehalten.
//
// Der Preis eines Fehlers ist hier asymmetrisch: Ein zu früh gelöschter Schüler nimmt
// die Forderung und die Spur zum Buch mit — beides unwiederbringlich.
func TestLoeschsperreBeiOffenenVorgaengen(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	auditRepo := repository.NewAuditRepository(pool)

	// baueFall legt einen Schüler mit genau EINEM offenen Vorgang an.
	// vorgang: "ausleihe" = Buch noch draußen, "schaden" = Gebühr unbezahlt.
	baueFall := func(t *testing.T, marke, vorgang string, imPapierkorb bool) string {
		t.Helper()
		resetBestandsdaten(t, pool)

		papierkorb := "NULL"
		if imPapierkorb {
			papierkorb = "now()"
		}
		var sid, tid, eid string
		if err := pool.QueryRow(ctx, `
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger, deleted_at)
			VALUES ($1, 'Offen', 'Vorgang', '9b', 2020, true, `+papierkorb+`) RETURNING id`, marke).Scan(&sid); err != nil {
			t.Fatalf("Schüler: %v", err)
		}
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_titel (titel) VALUES ('Sperrbuch') RETURNING id`).Scan(&tid); err != nil {
			t.Fatalf("Titel: %v", err)
		}
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`,
			tid, "EX-"+marke).Scan(&eid); err != nil {
			t.Fatalf("Exemplar: %v", err)
		}

		switch vorgang {
		case "ausleihe":
			// rueckgabe_am IS NULL — das Buch ist draußen.
			if _, err := pool.Exec(ctx,
				`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist) VALUES ($1, $2, CURRENT_DATE)`,
				eid, sid); err != nil {
				t.Fatalf("Ausleihe: %v", err)
			}
		case "schaden":
			// Zurückgegeben, aber die Gebühr steht offen — die Forderung lebt weiter.
			var aid string
			if err := pool.QueryRow(ctx, `
				INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist, rueckgabe_am)
				VALUES ($1, $2, CURRENT_DATE, now()) RETURNING id`, eid, sid).Scan(&aid); err != nil {
				t.Fatalf("Ausleihe: %v", err)
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO schadensfaelle (exemplar_id, ausleihe_id, schueler_id, beschreibung, betrag, ist_bezahlt)
				VALUES ($1, $2, $3, 'Einband gerissen', 12.50, false)`, eid, aid, sid); err != nil {
				t.Fatalf("Schadensfall: %v", err)
			}
		}
		return sid
	}

	nochDa := func(t *testing.T, sid string) {
		t.Helper()
		var da bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schueler WHERE id = $1 AND deleted_at IS NULL)`, sid).Scan(&da); err != nil {
			t.Fatalf("Nachschau: %v", err)
		}
		if !da {
			t.Error("Schüler ist trotz offenem Vorgang weg oder im Papierkorb")
		}
	}

	for _, vorgang := range []string{"ausleihe", "schaden"} {
		t.Run("Papierkorb blockiert bei "+vorgang, func(t *testing.T) {
			sid := baueFall(t, "SP-SOFT", vorgang, false)
			status, err := srv.pruefeSchuelerLoeschbar(ctx, sid)
			if err == nil {
				t.Fatalf("Soft-Delete trotz offenem Vorgang (%s) erlaubt", vorgang)
			}
			// 400, nicht 500: Das ist eine Regel, kein Störfall — sonst ersetzt der
			// Sanitizer die Meldung durch „interner Datenbankfehler" und das Personal
			// erfährt nie, WARUM es nicht geht.
			if status != 400 {
				t.Errorf("Status %d, erwartet 400 — die Begründung erreicht sonst das Formular nicht", status)
			}
			nochDa(t, sid)
		})

		t.Run("endgültiges Löschen von Hand blockiert bei "+vorgang, func(t *testing.T) {
			sid := baueFall(t, "SP-PURGE", vorgang, true)
			if err := auditRepo.PurgeStudent(ctx, sid, ""); err == nil {
				t.Fatalf("PurgeStudent trotz offenem Vorgang (%s) durchgelaufen", vorgang)
			}
			var da bool
			if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schueler WHERE id = $1)`, sid).Scan(&da); err != nil {
				t.Fatalf("Nachschau: %v", err)
			}
			if !da {
				t.Error("Schüler wurde trotz Blockade endgültig gelöscht")
			}
		})

		t.Run("Cronjob blockiert bei "+vorgang, func(t *testing.T) {
			sid := baueFall(t, "SP-CRON", vorgang, false)
			if err := auditRepo.PurgeAbgaenger(ctx, sid, ""); err == nil {
				t.Fatalf("PurgeAbgaenger trotz offenem Vorgang (%s) durchgelaufen", vorgang)
			}
			nochDa(t, sid)
		})
	}
}

// TestStornoHebtDieLoeschsperreAuf: Der Storno setzt ist_bezahlt = true — deshalb genügt
// allen Wächtern die Frage „ist_bezahlt = false", ohne storniert_am mitzuprüfen.
//
// Diese Zusage trägt das ganze Gebäude: Wäre sie falsch, bliebe eine stornierte Gebühr
// für immer „offen" und der Schüler damit dauerhaft unlöschbar — eine DSGVO-Frist, die
// nie ablaufen kann, und zwar lautlos. Bisher stand sie nur als Kommentar in
// api/lusd_apply.go.
func TestStornoHebtDieLoeschsperreAuf(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	var sid, tid, eid, aid, fallID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger)
		VALUES ('SP-STORNO', 'Stor', 'No', '9b', 2020, true) RETURNING id`).Scan(&sid); err != nil {
		t.Fatalf("Schüler: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ('Stornobuch') RETURNING id`).Scan(&tid); err != nil {
		t.Fatalf("Titel: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'EX-STORNO') RETURNING id`, tid).Scan(&eid); err != nil {
		t.Fatalf("Exemplar: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist, rueckgabe_am)
		VALUES ($1, $2, CURRENT_DATE, now()) RETURNING id`, eid, sid).Scan(&aid); err != nil {
		t.Fatalf("Ausleihe: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO schadensfaelle (exemplar_id, ausleihe_id, schueler_id, beschreibung, betrag, ist_bezahlt)
		VALUES ($1, $2, $3, 'Wasserschaden', 20.00, false) RETURNING id`, eid, aid, sid).Scan(&fallID); err != nil {
		t.Fatalf("Schadensfall: %v", err)
	}

	auditRepo := repository.NewAuditRepository(pool)
	if err := auditRepo.PurgeAbgaenger(ctx, sid, ""); err == nil {
		t.Fatal("Vorbedingung verfehlt: die unbezahlte Gebühr sperrt nicht")
	} else if !strings.Contains(err.Error(), "blockiert") {
		t.Fatalf("unerwarteter Fehler statt Blockade: %v", err)
	}

	// Über den echten Storno-Weg, nicht per UPDATE von Hand — sonst prüfte der Test
	// seine eigene Annahme statt des Codes.
	if err := auditRepo.StornierungGebuehr(ctx, fallID, adminFuerStorno(t, pool), "Kulanz"); err != nil {
		t.Fatalf("Storno: %v", err)
	}

	if err := auditRepo.PurgeAbgaenger(ctx, sid, ""); err != nil {
		t.Fatalf("nach dem Storno muss die Löschung durchgehen, sonst läuft die Frist nie ab: %v", err)
	}
}

// adminFuerStorno legt den Bearbeiter an, den der Storno protokollieren muss.
func adminFuerStorno(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ('ADM-STORNO', 'Ada', 'Admin', 'storno@example.org', 'admin', true) RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Bearbeiter: %v", err)
	}
	return id
}
