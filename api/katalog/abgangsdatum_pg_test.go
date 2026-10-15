package katalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Das Abgangsdatum (Migration 128) trägt das Abgangsbuch. Gemessen wird an den echten Türen,
// nicht am Trigger allein: Ein Stempel, den ein Schreiber überschreibt oder umgeht, wäre ein
// Abgangsbuch mit Lücken, und niemand sucht eine Zeile, von der er nicht weiß.

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
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	bookRepo := repository.NewKatalogAbfragen(pool)
	auditRepo := repository.NewProtokollAbfragen(pool)
	damageRepo := repository.NewSchadensAbfragen(pool)

	// Ein Bearbeiter für das Ausbuchen: Es schreibt einen Eintrag ins Protokoll und verlangt
	// deshalb eine echte Benutzer-ID.
	var bearbeiterID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Abgang', 'Buch', 'abgangsbuch@test.invalid', 'admin', true)
		ON CONFLICT (email) DO UPDATE SET vorname = EXCLUDED.vorname
		RETURNING id`).Scan(&bearbeiterID); err != nil {
		t.Fatalf("Bearbeiter anlegen: %v", err)
	}

	titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Abgangs-Titel", "Abg 1", 0)

	// Tür 1: die Abfrage des Status-Editors, direkt (KatalogAbfragen.AendereExemplarStatus).
	eins := testdaten.LegeExemplarAn(t, pool, titelID, "ABG-1", true, "")
	if err := bookRepo.AendereExemplarStatus(ctx, eins, false, true, "", nil, testdaten.LegeBearbeiterAn(t, pool)); err != nil {
		t.Fatalf("Aussondern: %v", err)
	}

	// Tür 2: dieselbe Abfrage über die Tür, PUT /status mit ist_ausgesondert=true
	zwei := testdaten.LegeExemplarAn(t, pool, titelID, "ABG-2", true, "")
	req := httptest.NewRequest(http.MethodPut, "/api/buecher/exemplare/"+zwei+"/status",
		strings.NewReader(`{"ist_ausleihbar":false,"ist_ausgesondert":true,"zustand_notiz":"verloren"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", zwei)
	rec := httptest.NewRecorder()
	// Türen mit Datenbank: Die Tür fragt nach dem Speichern die Preisquelle aus den
	// Einstellungen; ohne Pool stürzte sie dort ab.
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}
	srv.ExemplarStatusAendern(bookRepo, repository.NewBescheidAbfragen(pool))(rec, tuertest.AlsKonto(req, testdaten.LegeBearbeiterAn(t, pool)))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status-Tür: HTTP %d: %s", rec.Code, rec.Body.String())
	}

	// Tür 3: Ausbuchen (SondereExemplarAus) — sondert aus, statt zu löschen.
	drei := testdaten.LegeExemplarAn(t, pool, titelID, "ABG-3", true, "")
	if err := auditRepo.SondereExemplarAus(ctx, drei, bearbeiterID); err != nil {
		t.Fatalf("Ausbuchen: %v", err)
	}

	// Tür 4: Schaden melden. Der Schuldner steht an der Ausleihe, also braucht dieser Weg
	// eine offene: Es ist der Weg aus der Schülerakte.
	vier := testdaten.LegeExemplarAn(t, pool, titelID, "ABG-4", true, "")
	schuelerID := testdaten.LegeSchuelerAn(t, pool, "ABG-S-1", "Ada", "6b")
	var ausleiheID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
		VALUES ($1, $2, NOW() - interval '2 days', NOW() + interval '21 days') RETURNING id`,
		vier, schuelerID).Scan(&ausleiheID); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
	if _, err := damageRepo.Melde(ctx, vier, ausleiheID, bearbeiterID, "Wasserschaden",
		repository.SchadensArtBeschaedigt, 12.50); err != nil {
		t.Fatalf("Schaden melden: %v", err)
	}

	for _, fall := range []struct{ name, id string }{
		{"AendereExemplarStatus", eins},
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
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	bookRepo := repository.NewKatalogAbfragen(pool)

	titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Abgangs-Titel 2", "Abg 2", 0)
	id := testdaten.LegeExemplarAn(t, pool, titelID, "ABG-STABIL", true, "")
	if err := bookRepo.AendereExemplarStatus(ctx, id, false, true, "", nil, testdaten.LegeBearbeiterAn(t, pool)); err != nil {
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
	if err := bookRepo.AendereExemplarStatus(ctx, id, false, true, "", nil, testdaten.LegeBearbeiterAn(t, pool)); err == nil {
		// AendereExemplarStatus meldet bei einem bereits ausgesonderten Exemplar keinen Fehler;
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
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	bookRepo := repository.NewKatalogAbfragen(pool)

	titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Abgangs-Titel 3", "Abg 3", 0)
	id := testdaten.LegeExemplarAn(t, pool, titelID, "ABG-ZURUECK", true, "")
	if err := bookRepo.AendereExemplarStatus(ctx, id, false, true, "", nil, testdaten.LegeBearbeiterAn(t, pool)); err != nil {
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
// Bestandsnachweis. Dasselbe gilt für eine Zeile, die ausgesondert eingefügt wird: Ein
// Import historischer Abgänge trüge sonst das Datum seines Imports.
func TestAbgangsdatum_KeinErfundenesDatumBeimEinfuegen(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Abgangs-Titel 4", "Abg 4", 0)
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
