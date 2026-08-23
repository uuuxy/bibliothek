package repository

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Das endgültige Löschen eines abgeschriebenen Verlust-Exemplars — an dem Zustand, den
// es in einer Schulbibliothek WIRKLICH vorfindet.
//
// Der bestehende Test legt frische Exemplare an und leiht sie nie aus. Er war grün,
// während der Knopf an jedem Exemplar scheiterte, das je unterwegs war: ON DELETE
// RESTRICT auf ausleihen.exemplar_id, gemeldet als 500 → „interner Datenbankfehler".
// Ein Buch verschwindet aber typischerweise, NACHDEM es ausgeliehen war — der grüne
// Test prüfte den einen Fall, den es kaum gibt.
func TestEndgueltigLoescheVerlust_MitAusleihhistorie(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventoryRepository(pool)
	bearbeiter := seedBearbeiter(t, pool)

	exemplarID := seedVerlorenesExemplar(t, ctx, repo, "Erdkundebuch 7", "VL-HIST")
	schuelerID := seedSchuelerFuerVerlust(t, ctx, pool, "VL-S1")
	ausleiheID := seedZurueckgegebeneAusleihe(t, ctx, pool, exemplarID, schuelerID)

	// Eine BEZAHLTE Gebühr an derselben Ausleihe: erledigte Geschichte, darf nicht
	// blockieren — und muss mit verschwinden, sonst hält ihr RESTRICT das Exemplar fest.
	if _, err := pool.Exec(ctx, `
		INSERT INTO schadensfaelle (exemplar_id, ausleihe_id, schueler_id, beschreibung, betrag, ist_bezahlt)
		VALUES ($1, $2, $3, 'Eselsohr', 2.00, true)`, exemplarID, ausleiheID, schuelerID); err != nil {
		t.Fatalf("bezahlte Gebühr: %v", err)
	}

	anzahl, err := repo.EndgueltigLoescheVerlustExemplare(ctx, []string{exemplarID}, bearbeiter)
	if err != nil {
		t.Fatalf("Exemplar mit Ausleihhistorie ließ sich nicht löschen: %v", err)
	}
	if anzahl != 1 {
		t.Fatalf("%d gelöscht, erwartet 1", anzahl)
	}

	// Der Barcode muss wieder frei sein — buecher_exemplare.barcode_id ist UNIQUE ohne
	// Teilindex. Bliebe die Zeile stehen, wäre der Aufkleber für immer verbraucht und das
	// Buch könnte nie mit derselben Nummer neu eingepflegt werden.
	var titelID string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ('Erdkundebuch 7 (neu)') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'VL-HIST')`, titelID); err != nil {
		t.Errorf("Barcode wurde nicht freigegeben, Wiedereinpflegen unmöglich: %v", err)
	}

	// Der Fehlbestandsbericht muss die Abschrift behalten (ON DELETE SET NULL) — sonst
	// wäre nach dem Aufräumen nicht mehr nachvollziehbar, WAS abgeschrieben wurde.
	var abschrift int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM inventur_verluste WHERE barcode_id = 'VL-HIST'`).Scan(&abschrift); err != nil {
		t.Fatalf("Abschrift: %v", err)
	}
	if abschrift != 1 {
		t.Errorf("Fehlbestandsbericht hat die Abschrift verloren (%d Zeilen)", abschrift)
	}
}

// TestEndgueltigLoescheVerlust_OffeneGebuehrSperrt: Eine unbezahlte Gebühr ist eine
// Forderung der Schule. Sie mitzulöschen hieße, sie unwiederbringlich aufzugeben —
// und zwar lautlos, als Nebenwirkung eines Aufräumknopfs.
func TestEndgueltigLoescheVerlust_OffeneGebuehrSperrt(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventoryRepository(pool)
	bearbeiter := seedBearbeiter(t, pool)

	frei := seedVerlorenesExemplar(t, ctx, repo, "Mathebuch 5", "VL-FREI")
	gesperrt := seedVerlorenesExemplar(t, ctx, repo, "Mathebuch 6", "VL-GEBUEHR")
	schuelerID := seedSchuelerFuerVerlust(t, ctx, pool, "VL-S2")
	ausleiheID := seedZurueckgegebeneAusleihe(t, ctx, pool, gesperrt, schuelerID)
	if _, err := pool.Exec(ctx, `
		INSERT INTO schadensfaelle (exemplar_id, ausleihe_id, schueler_id, beschreibung, betrag, ist_bezahlt)
		VALUES ($1, $2, $3, 'Buch verloren', 24.90, false)`, gesperrt, ausleiheID, schuelerID); err != nil {
		t.Fatalf("offene Gebühr: %v", err)
	}

	_, err := repo.EndgueltigLoescheVerlustExemplare(ctx, []string{frei, gesperrt}, bearbeiter)
	if err == nil {
		t.Fatal("Stapel mit offener Gebühr durchgelaufen — die Forderung wäre weg")
	}
	// Der Handler macht daraus eine 409 mit Klartext; ohne den Sentinel würde daraus
	// ein 500 und der Sanitizer ersetzte die Begründung.
	if !errors.Is(err, ErrVerlustNochGebunden) {
		t.Fatalf("falscher Fehlertyp, wird zur 500 sanitisiert: %v", err)
	}
	// Der Barcode muss in der Meldung stehen — sonst weiß niemand, welches Exemplar
	// den Stapel aufhält.
	if !strings.Contains(err.Error(), "VL-GEBUEHR") {
		t.Errorf("Meldung nennt das blockierende Exemplar nicht: %v", err)
	}

	// Alles-oder-nichts: Auch das unbelastete Exemplar bleibt stehen.
	for _, fall := range []struct{ id, name string }{{frei, "VL-FREI"}, {gesperrt, "VL-GEBUEHR"}} {
		var da bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM buecher_exemplare WHERE id = $1)`, fall.id).Scan(&da); err != nil {
			t.Fatal(err)
		}
		if !da {
			t.Errorf("%s wurde trotz abgewiesenem Stapel gelöscht", fall.name)
		}
	}
	var gebuehrDa bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM schadensfaelle WHERE exemplar_id = $1 AND ist_bezahlt = false)`, gesperrt).Scan(&gebuehrDa); err != nil {
		t.Fatal(err)
	}
	if !gebuehrDa {
		t.Error("die offene Forderung wurde gelöscht")
	}
}

func seedSchuelerFuerVerlust(t *testing.T, ctx context.Context, pool *pgxpool.Pool, barcode string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ($1, 'Vera', 'Lust', '7a', 2030) RETURNING id`, barcode).Scan(&id); err != nil {
		t.Fatalf("Schüler: %v", err)
	}
	return id
}

// seedZurueckgegebeneAusleihe: eine abgeschlossene Ausleihe — genau die Historie, an der
// das endgültige Löschen bisher scheiterte.
func seedZurueckgegebeneAusleihe(t *testing.T, ctx context.Context, pool *pgxpool.Pool, exemplarID, schuelerID string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
		VALUES ($1, $2, now() - interval '40 days', now() - interval '26 days', now() - interval '20 days')
		RETURNING id`, exemplarID, schuelerID).Scan(&id); err != nil {
		t.Fatalf("Ausleihe: %v", err)
	}
	return id
}
