package api

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Bug 5 (Permanent Ghost-Block): Ein Abgänger, der wegen offener Vorgänge nur GESPERRT (nicht
// anonymisiert) wurde, behielt seine "Automatisierte Abgänger-Sperre" beim LUSD-Wiedereintritt
// dauerhaft. aktualisiereBestandsschueler muss beim Rückkehrer prüfen, ob noch Vorgänge offen
// sind, und andernfalls automatisch entsperren. Der Kern ist SQL-CASE-Logik mit Sub-Selects —
// nur ein echter DB-Test (nicht pgxmock) prüft sie.

// insertGesperrterAbgaenger legt einen als Abgänger gesperrten Schüler an und liefert die ID.
func insertGesperrterAbgaenger(t *testing.T, pool *pgxpool.Pool, barcode, vorname, nachname, klasse, blockReason string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO schueler
		   (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger, ist_gesperrt, block_reason)
		 VALUES ($1, $2, $3, $4, 2025, true, true, $5)
		 RETURNING id`,
		barcode, vorname, nachname, klasse, blockReason).Scan(&id); err != nil {
		t.Fatalf("gesperrten Abgänger %q anlegen: %v", barcode, err)
	}
	return id
}

// seedOffeneAusleihe hängt dem Schüler eine noch nicht zurückgegebene Ausleihe an.
func seedOffeneAusleihe(t *testing.T, pool *pgxpool.Pool, schuelerID, barcodePraefix string) {
	t.Helper()
	titelID := titelMitMeldebestand(t, pool, "Titel-"+barcodePraefix, 1)
	exID := exemplar(t, pool, titelID, "EX-"+barcodePraefix, true, "")
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		 VALUES ($1, $2, now() + interval '14 days')`, exID, schuelerID); err != nil {
		t.Fatalf("offene Ausleihe anlegen: %v", err)
	}
}

// seedOffenerSchaden hängt dem Schüler einen unbezahlten Schadensfall an.
func seedOffenerSchaden(t *testing.T, pool *pgxpool.Pool, schuelerID, barcodePraefix string) {
	t.Helper()
	titelID := titelMitMeldebestand(t, pool, "SchadenTitel-"+barcodePraefix, 1)
	exID := exemplar(t, pool, titelID, "SEX-"+barcodePraefix, true, "")
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, ist_bezahlt)
		 VALUES ($1, $2, 'Wasserschaden', 12.50, false)`, exID, schuelerID); err != nil {
		t.Fatalf("offenen Schaden anlegen: %v", err)
	}
}

// runAktualisiere führt aktualisiereBestandsschueler in einer echten Transaktion aus.
func runAktualisiere(t *testing.T, pool *pgxpool.Pool, rec parsedStudentRow, id string) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // Rollback nach Commit ist no-op
	if err := aktualisiereBestandsschueler(ctx, tx, rec, id); err != nil {
		t.Fatalf("aktualisiereBestandsschueler: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Commit: %v", err)
	}
}

// leseSchuelerStatus liefert Sperr-/Abgänger-Status, Grund und Vorname nach dem Update.
func leseSchuelerStatus(t *testing.T, pool *pgxpool.Pool, id string) (gesperrt, abgaenger bool, reason *string, vorname string) {
	t.Helper()
	if err := pool.QueryRow(context.Background(),
		`SELECT ist_gesperrt, ist_abgaenger, block_reason, vorname FROM schueler WHERE id = $1`, id).
		Scan(&gesperrt, &abgaenger, &reason, &vorname); err != nil {
		t.Fatalf("Status lesen: %v", err)
	}
	return
}

func TestAktualisiereBestandsschueler_Rueckkehrer(t *testing.T) {
	pool := pgTestPool(t)

	t.Run("gesperrter Abgänger ohne offene Vorgänge wird entsperrt", func(t *testing.T) {
		resetBestandsdaten(t, pool)
		id := insertGesperrterAbgaenger(t, pool, "R-A", "Max", "Muster", "ABG",
			"Automatisierte Abgänger-Sperre (offene Vorgänge)")

		runAktualisiere(t, pool, parsedStudentRow{Vorname: "Max", Nachname: "Muster", Klasse: "9a"}, id)

		gesperrt, abgaenger, reason, _ := leseSchuelerStatus(t, pool, id)
		if gesperrt {
			t.Error("Rückkehrer ohne offene Vorgänge muss entsperrt sein (Permanent Ghost-Block)")
		}
		if abgaenger {
			t.Error("ist_abgaenger muss beim Rückkehrer zurückgesetzt sein")
		}
		if reason != nil {
			t.Errorf("block_reason muss geräumt sein, war %q", *reason)
		}
	})

	t.Run("gesperrter Abgänger mit offener Ausleihe bleibt gesperrt, Grund umbenannt", func(t *testing.T) {
		resetBestandsdaten(t, pool)
		id := insertGesperrterAbgaenger(t, pool, "R-B", "Lea", "Klein", "ABG",
			"Automatisierte Abgänger-Sperre (offene Vorgänge)")
		seedOffeneAusleihe(t, pool, id, "B")

		runAktualisiere(t, pool, parsedStudentRow{Vorname: "Lea", Nachname: "Klein", Klasse: "9a"}, id)

		gesperrt, abgaenger, reason, _ := leseSchuelerStatus(t, pool, id)
		if !gesperrt {
			t.Error("mit offener Ausleihe muss die Sperre bestehen bleiben")
		}
		if abgaenger {
			t.Error("ist_abgaenger muss trotzdem zurückgesetzt sein (er ist wieder aktiv)")
		}
		if reason == nil || *reason != "Sperre wegen offener Vorgänge" {
			t.Errorf("irreführender Abgänger-Grund muss umbenannt werden, war %v", reason)
		}
	})

	t.Run("gesperrter Abgänger mit unbezahltem Schaden bleibt gesperrt", func(t *testing.T) {
		resetBestandsdaten(t, pool)
		id := insertGesperrterAbgaenger(t, pool, "R-C", "Tom", "Groß", "ABG",
			"Automatisierte Abgänger-Sperre (offene Vorgänge)")
		seedOffenerSchaden(t, pool, id, "C")

		runAktualisiere(t, pool, parsedStudentRow{Vorname: "Tom", Nachname: "Groß", Klasse: "9a"}, id)

		gesperrt, _, reason, _ := leseSchuelerStatus(t, pool, id)
		if !gesperrt {
			t.Error("mit unbezahltem Schaden muss die Sperre bestehen bleiben")
		}
		if reason == nil || *reason != "Sperre wegen offener Vorgänge" {
			t.Errorf("Grund muss umbenannt werden, war %v", reason)
		}
	})

	t.Run("manuelle Sperre bleibt unangetastet", func(t *testing.T) {
		resetBestandsdaten(t, pool)
		const manuell = "Manuell gesperrt: wiederholter Vandalismus"
		id := insertGesperrterAbgaenger(t, pool, "R-D", "Nia", "Wolf", "ABG", manuell)

		runAktualisiere(t, pool, parsedStudentRow{Vorname: "Nia", Nachname: "Wolf", Klasse: "9a"}, id)

		gesperrt, abgaenger, reason, _ := leseSchuelerStatus(t, pool, id)
		if !gesperrt {
			t.Error("manuelle Sperre darf nicht automatisch aufgehoben werden")
		}
		if abgaenger {
			t.Error("ist_abgaenger muss dennoch zurückgesetzt sein")
		}
		if reason == nil || *reason != manuell {
			t.Errorf("manueller Grund muss erhalten bleiben, war %v", reason)
		}
	})

	t.Run("anonymisierter Abgänger wird entsperrt und umbenannt", func(t *testing.T) {
		resetBestandsdaten(t, pool)
		// Anonymisierte tragen 'Abgänger' / 'Anonymisiert-…' und den festen Grund.
		id := insertGesperrterAbgaenger(t, pool, "R-E", "Abgänger", "Anonymisiert-xyz", "ABG",
			"Abgänger anonymisiert")

		runAktualisiere(t, pool, parsedStudentRow{Vorname: "Sophie", Nachname: "Real", Klasse: "Q1"}, id)

		gesperrt, abgaenger, reason, vorname := leseSchuelerStatus(t, pool, id)
		if gesperrt {
			t.Error("anonymisierter Rückkehrer muss entsperrt werden")
		}
		if abgaenger {
			t.Error("ist_abgaenger muss zurückgesetzt sein")
		}
		if reason != nil {
			t.Errorf("block_reason muss geräumt sein, war %q", *reason)
		}
		if vorname != "Sophie" {
			t.Errorf("echter Name aus dem Export muss übernommen werden, war %q", vorname)
		}
	})
}
