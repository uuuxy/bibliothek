package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// vormerkTheke ist der Aufbau für die Tests von Ausleihe und Rückgabe mit Vormerkungen an der
// Datenbank: ein Mitarbeiter, der Dienst und Helfer, die unter dem Namen des Tests anlegen.
type vormerkTheke struct {
	pool        *pgxpool.Pool
	suffix      string
	mitarbeiter string
	buecher     repository.BookRepository
	svc         LoanService
}

func neueVormerkTheke(t *testing.T) *vormerkTheke {
	t.Helper()
	pool := pgtest.Pool(t)
	ctx := context.Background()
	v := &vormerkTheke{pool: pool, suffix: fmt.Sprintf("%d", time.Now().UnixNano()), buecher: repository.NewBookRepository(pool)}
	if err := pool.QueryRow(ctx, `INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Vormerktheke', 'Ausgabe', $1, 'mitarbeiter', true) RETURNING id`, "vormerktheke-"+v.suffix+"@schule.invalid").Scan(&v.mitarbeiter); err != nil {
		t.Fatalf("Mitarbeiter anlegen: %v", err)
	}
	t.Cleanup(func() {
		for _, sql := range []string{
			`DELETE FROM ausleihen WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE barcode_id LIKE 'VT-%-' || $1)`,
			`DELETE FROM buecher_titel WHERE titel LIKE 'Vormerktheke %' || $1`,
			`DELETE FROM benutzer WHERE email = 'vormerktheke-' || $1 || '@schule.invalid'`,
			`DELETE FROM leser WHERE vorname = 'Vormerktheke' AND $1 = $1`,
		} {
			if _, err := pool.Exec(ctx, sql, v.suffix); err != nil {
				t.Errorf("aufräumen (%s): %v", sql, err)
			}
		}
	})
	v.svc = NewLoanService(pool, repository.NewStudentRepository(pool), v.buecher, repository.NewLoanRepository(pool),
		repository.NewAuditRepository(pool))
	return v
}

// titel legt einen Titel mit so vielen Exemplaren an, wie Namen genannt sind, und liefert seine
// Kennung und je Namen die Nummer des Exemplars.
func (v *vormerkTheke) titel(t *testing.T, name string, exemplare ...string) (titelID string, nummern map[string]string) {
	t.Helper()
	ctx := context.Background()
	if err := v.pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ($1) RETURNING id`, "Vormerktheke "+name+" "+v.suffix).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	nummern = map[string]string{}
	for _, e := range exemplare {
		nummer := "VT-" + name + e + "-" + v.suffix
		if _, err := v.pool.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2)`, titelID, nummer); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
		nummern[e] = nummer
	}
	return titelID, nummern
}

// leser legt einen Leser der genannten Art an; ein Schüler bekommt die Klasse 06C.
func (v *vormerkTheke) leser(t *testing.T, nachname, art string) string {
	t.Helper()
	var id string
	sql := `INSERT INTO leser (vorname, nachname, art) VALUES ('Vormerktheke', $1, $2) RETURNING id`
	args := []any{nachname, art}
	if art == "schueler" {
		sql = `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr) VALUES ($2, 'Vormerktheke', $1, '06C', 2032) RETURNING id`
		args = []any{nachname, "VT-L-" + nachname + "-" + v.suffix}
	}
	if err := v.pool.QueryRow(context.Background(), sql, args...).Scan(&id); err != nil {
		t.Fatalf("Leser %s anlegen: %v", nachname, err)
	}
	return id
}

// legeBereit trägt eine Vormerkung des Lesers ein, für die das Exemplar im Abholfach liegt.
func (v *vormerkTheke) legeBereit(t *testing.T, titelID, leserID, nummer string) {
	t.Helper()
	if _, err := v.pool.Exec(context.Background(), `INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis)
		SELECT $1, $2, 'abholbereit', id, now() + interval '2 days' FROM buecher_exemplare WHERE barcode_id = $3`, titelID, leserID, nummer); err != nil {
		t.Fatalf("Vormerkung anlegen: %v", err)
	}
}

// leihe scannt das Exemplar für den Leser.
func (v *vormerkTheke) leihe(t *testing.T, nummer, leserID string) (*LoanResult, error) {
	t.Helper()
	ctx := context.Background()
	exemplar, err := v.buecher.GetCopyByBarcode(ctx, nummer)
	if err != nil || exemplar == nil {
		t.Fatalf("Exemplar %s laden: %v", nummer, err)
	}
	return v.svc.HandleUnifiedCheckout(ctx, exemplar, &leserID, v.mitarbeiter, false)
}

// vormerkungen zählt die Vormerkungen des Titels.
func (v *vormerkTheke) vormerkungen(t *testing.T, titelID string) int {
	t.Helper()
	var n int
	if err := v.pool.QueryRow(context.Background(), `SELECT count(*) FROM vormerkungen WHERE titel_id = $1`, titelID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Ein Exemplar, das für einen Schüler im Abholfach liegt, bekommt nur er: kein anderer Schüler
// und kein Kollege. Nimmt er es, ist seine Vormerkung erfüllt, und die Antwort nennt kein
// Exemplar, das zurück ins Regal müsste.
func TestAusleihe_ReserviertesExemplarNurFuerDenVorgemerkten(t *testing.T) {
	v := neueVormerkTheke(t)
	titelID, nummern := v.titel(t, "Reserviert", "A")
	vorgemerkt := v.leser(t, "Vorgemerkt", "schueler")
	v.legeBereit(t, titelID, vorgemerkt, nummern["A"])

	for name, leserID := range map[string]string{
		"ein anderer Schüler": v.leser(t, "Anderer", "schueler"),
		"ein Kollege":         v.leser(t, "Kollegin", "lehrkraft"),
	} {
		_, err := v.leihe(t, nummern["A"], leserID)
		if !errors.Is(err, ErrConflict) || !strings.Contains(fmt.Sprint(err), "für Vormerktheke Vorgemerkt reserviert") {
			t.Errorf("%s: Fehler %v, erwartet den Konflikt mit dem Namen des Vorgemerkten", name, err)
		}
	}
	if n := v.vormerkungen(t, titelID); n != 1 {
		t.Fatalf("%d Vormerkungen nach den abgewiesenen Scans, erwartet die eine", n)
	}

	res, err := v.leihe(t, nummern["A"], vorgemerkt)
	if err != nil || res.Type != "ausleihe" {
		t.Fatalf("der Vorgemerkte: %+v, Fehler %v", res, err)
	}
	if res.RegalfreigabeBarcode != "" {
		t.Errorf("Regal-Hinweis %q, erwartet keinen: Er hat das bereitgelegte Exemplar genommen", res.RegalfreigabeBarcode)
	}
	if n := v.vormerkungen(t, titelID); n != 0 {
		t.Errorf("%d Vormerkungen nach der Ausleihe, erwartet keine", n)
	}
}

// Nimmt der Vorgemerkte ein anderes Exemplar des Titels als das bereitgelegte, ist seine
// Vormerkung ebenfalls erfüllt, und die Antwort nennt das Exemplar im Abholfach: Es muss
// zurück ins Regal.
func TestAusleihe_AnderesExemplarNenntDasBereitgelegte(t *testing.T) {
	v := neueVormerkTheke(t)
	titelID, nummern := v.titel(t, "Freihand", "Fach", "Regal")
	vorgemerkt := v.leser(t, "Freihand", "schueler")
	v.legeBereit(t, titelID, vorgemerkt, nummern["Fach"])

	res, err := v.leihe(t, nummern["Regal"], vorgemerkt)
	if err != nil || res.Type != "ausleihe" {
		t.Fatalf("Ausleihe: %+v, Fehler %v", res, err)
	}
	if res.RegalfreigabeBarcode != nummern["Fach"] {
		t.Errorf("Regal-Hinweis %q, erwartet das bereitgelegte Exemplar %s", res.RegalfreigabeBarcode, nummern["Fach"])
	}
	if n := v.vormerkungen(t, titelID); n != 0 {
		t.Errorf("%d Vormerkungen nach der Ausleihe, erwartet keine", n)
	}
}
