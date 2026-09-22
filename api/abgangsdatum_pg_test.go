package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Das Abgangsdatum (Migration 128) ist der fehlende Teil des Abgangsbuchs — Punkt 1 des
// Protokolls vom 16.09.2026. Gemessen wird an den echten Türen, nicht am Trigger allein:
// Ein Stempel, den ein Schreiber überschreibt oder umgeht, wäre ein Abgangsbuch mit
// Lücken, und niemand sucht eine Zeile, von der er nicht weiß.

func abgangsdatum(t *testing.T, pool *pgxpool.Pool, id string) *time.Time {
	t.Helper()
	var ts *time.Time
	if err := pool.QueryRow(context.Background(),
		`SELECT ausgesondert_am FROM buecher_exemplare WHERE id = $1`, id).Scan(&ts); err != nil {
		t.Fatalf("ausgesondert_am lesen: %v", err)
	}
	return ts
}

func TestAbgangsdatum_JedeTuerStempelt(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	bookRepo := repository.NewBookRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)
	damageRepo := repository.NewDamageRepository(pool)

	// Ein Bearbeiter für das Ausbuchen: Es schreibt einen Audit-Eintrag und verlangt
	// deshalb eine echte Benutzer-ID.
	var bearbeiterID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Abgang', 'Buch', 'abgangsbuch@test.invalid', 'admin', true)
		ON CONFLICT (email) DO UPDATE SET vorname = EXCLUDED.vorname
		RETURNING id`).Scan(&bearbeiterID); err != nil {
		t.Fatalf("Bearbeiter anlegen: %v", err)
	}

	titelID := titelMitSignatur(t, pool, "Abgangs-Titel", "Abg 1", 0)

	// Tür 1: die Repository-Methode des Status-Editors, direkt (POST /aussondern ist am
	// 22.09.2026 gestrichen, OFFEN.md 4.16)
	eins := exemplar(t, pool, titelID, "ABG-1", true, "")
	if err := bookRepo.UpdateCopyStatus(ctx, eins, false, true, "", nil); err != nil {
		t.Fatalf("Aussondern: %v", err)
	}

	// Tür 2: dieselbe Methode über den Handler, PUT /status mit ist_ausgesondert=true
	zwei := exemplar(t, pool, titelID, "ABG-2", true, "")
	req := httptest.NewRequest(http.MethodPut, "/api/buecher/exemplare/"+zwei+"/status",
		strings.NewReader(`{"ist_ausleihbar":false,"ist_ausgesondert":true,"zustand_notiz":"verloren"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", zwei)
	rec := httptest.NewRecorder()
	// Server MIT Pool: Der Handler fragt nach dem Speichern die Preisquelle aus den
	// Einstellungen — ohne Pool stürzt er dort ab (dieselbe Falle wie bei der Leserdatei
	// am 17.09.2026).
	srv := &Server{DB: &db.Database{Pool: pool}}
	srv.UpdateCopyStatusHandler(bookRepo, repository.NewBescheidRepository(pool))(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status-Tür: HTTP %d: %s", rec.Code, rec.Body.String())
	}

	// Tür 3: Ausbuchen (DeleteCopy) — sondert aus, statt zu löschen.
	drei := exemplar(t, pool, titelID, "ABG-3", true, "")
	if err := auditRepo.DeleteCopy(ctx, drei, bearbeiterID); err != nil {
		t.Fatalf("Ausbuchen: %v", err)
	}

	// Tür 4: Schaden melden. Der Schuldner steht an der Ausleihe, also braucht dieser Weg
	// eine offene: Es ist der Weg aus der Schülerakte.
	vier := exemplar(t, pool, titelID, "ABG-4", true, "")
	schuelerID := seedSchueler(t, pool, "ABG-S-1", "Ada", "6b")
	var ausleiheID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
		VALUES ($1, $2, NOW() - interval '2 days', NOW() + interval '21 days') RETURNING id`,
		vier, schuelerID).Scan(&ausleiheID); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
	if _, err := damageRepo.ReportDamage(ctx, vier, ausleiheID, schuelerID, bearbeiterID, "Wasserschaden",
		repository.SchadensArtBeschaedigt, 12.50); err != nil {
		t.Fatalf("Schaden melden: %v", err)
	}

	for _, fall := range []struct{ name, id string }{
		{"UpdateCopyStatus", eins},
		{"PUT /status", zwei},
		{"Ausbuchen", drei},
		{"Schaden melden", vier},
	} {
		if ts := abgangsdatum(t, pool, fall.id); ts == nil {
			t.Errorf("%s hat kein Abgangsdatum gesetzt — diese Zeile fehlt im Abgangsbuch", fall.name)
		}
	}
}

// Ein zweites UPDATE auf ein bereits ausgesondertes Exemplar darf den Abgang nicht auf
// heute schieben: Das Datum ist ein Nachweis, kein „zuletzt angefasst".
func TestAbgangsdatum_ZweitesUpdateVerschiebtNichts(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	bookRepo := repository.NewBookRepository(pool)

	titelID := titelMitSignatur(t, pool, "Abgangs-Titel 2", "Abg 2", 0)
	id := exemplar(t, pool, titelID, "ABG-STABIL", true, "")
	if err := bookRepo.UpdateCopyStatus(ctx, id, false, true, "", nil); err != nil {
		t.Fatalf("Aussondern: %v", err)
	}
	erst := abgangsdatum(t, pool, id)
	if erst == nil {
		t.Fatal("kein Abgangsdatum nach dem Aussondern")
	}

	// Rückdatieren, damit eine Verschiebung sichtbar wäre.
	if _, err := pool.Exec(ctx,
		`UPDATE buecher_exemplare SET ausgesondert_am = $1 WHERE id = $2`,
		erst.AddDate(-1, 0, 0), id); err != nil {
		t.Fatalf("zurückdatieren: %v", err)
	}
	alt := abgangsdatum(t, pool, id)

	// Nochmal dieselbe Tür, und ein UPDATE, das die Spalte gar nicht nennt.
	if err := bookRepo.UpdateCopyStatus(ctx, id, false, true, "", nil); err == nil {
		// UpdateCopyStatus meldet bei einem bereits ausgesonderten Exemplar keinen Fehler;
		// entscheidend ist der Stempel darunter.
		_ = err
	}
	if _, err := pool.Exec(ctx,
		`UPDATE buecher_exemplare SET zustand_notiz = 'später ergänzt' WHERE id = $1`, id); err != nil {
		t.Fatalf("Notiz ändern: %v", err)
	}

	if jetzt := abgangsdatum(t, pool, id); jetzt == nil || !jetzt.Equal(*alt) {
		t.Errorf("Abgangsdatum verschoben: war %v, ist %v", alt, jetzt)
	}
}

// Zurückgeholt heißt: Der Abgang war ein Irrtum. Bliebe das Datum stehen, führte das
// Abgangsbuch Bücher, die im Regal stehen.
func TestAbgangsdatum_RueckholenLoeschtDenStempel(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	bookRepo := repository.NewBookRepository(pool)

	titelID := titelMitSignatur(t, pool, "Abgangs-Titel 3", "Abg 3", 0)
	id := exemplar(t, pool, titelID, "ABG-ZURUECK", true, "")
	if err := bookRepo.UpdateCopyStatus(ctx, id, false, true, "", nil); err != nil {
		t.Fatalf("Aussondern: %v", err)
	}
	if abgangsdatum(t, pool, id) == nil {
		t.Fatal("kein Abgangsdatum nach dem Aussondern")
	}

	if _, err := repository.HoleExemplarZurueck(ctx, pool, id, "", nil); err != nil {
		t.Fatalf("zurückholen: %v", err)
	}
	if ts := abgangsdatum(t, pool, id); ts != nil {
		t.Errorf("zurückgeholtes Exemplar trägt weiter ein Abgangsdatum (%v)", ts)
	}
}

// Altbestand: Was vor der Migration ausgesondert wurde, hat kein bekanntes Datum. NULL ist
// die Wahrheit über diese Zeilen — ein erfundenes Datum stünde als Tatsache in einem
// Bestandsnachweis. Dasselbe gilt für eine Zeile, die AUSGESONDERT eingefügt wird: Ein
// Import historischer Abgänge trüge sonst das Datum seines Imports.
func TestAbgangsdatum_KeinErfundenesDatumBeimEinfuegen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	titelID := titelMitSignatur(t, pool, "Abgangs-Titel 4", "Abg 4", 0)
	var id string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, ist_ausgesondert, aussonderung_grund)
		 VALUES ($1, 'ABG-ALT', false, true, 'AUSSORTIERT') RETURNING id`, titelID).Scan(&id); err != nil {
		t.Fatalf("Altbestand anlegen: %v", err)
	}
	if ts := abgangsdatum(t, pool, id); ts != nil {
		t.Errorf("eingefügter Altbestand bekam ein erfundenes Abgangsdatum: %v", ts)
	}
}

// Quelltext-Ratsche: Das Datum setzt der Trigger, sonst niemand. Schriebe ein Schreibpfad
// es selbst, hätte das Abgangsbuch wieder so viele Wahrheiten wie Türen — und die nächste
// Tür vergäße es. Erlaubt ist nur, was ein Test zum Zurückdatieren tut.
func TestAbgangsdatum_KeinSchreiberSetztEsSelbst(t *testing.T) {
	wurzel := ".."
	var fundstellen []string
	err := filepath.Walk(wurzel, func(pfad string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		roh, err := os.ReadFile(pfad) //nolint:gosec // Testlauf über den eigenen Quellbaum
		if err != nil {
			return err
		}
		for _, zeile := range strings.Split(string(roh), "\n") {
			if strings.Contains(zeile, "ausgesondert_am") &&
				(strings.Contains(zeile, "SET") || strings.Contains(zeile, "ausgesondert_am =")) {
				fundstellen = append(fundstellen, pfad+": "+strings.TrimSpace(zeile))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Quellbaum lesen: %v", err)
	}
	if len(fundstellen) > 0 {
		t.Errorf("ausgesondert_am wird im Code geschrieben — das Datum gehört dem Trigger "+
			"(Migration 128), sonst vergisst es die nächste Tür:\n%s", strings.Join(fundstellen, "\n"))
	}
}
