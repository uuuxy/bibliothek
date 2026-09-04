package inventur

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

// erwarteFachBekannt spielt den Nachschlag der Fach-Registrierung nach: Das Fach ist
// bereits in systematik_kategorien registriert, der Schreibpfad bekommt die kanonische
// Bezeichnung zurück. Jeder pgxmock-Test eines subject-Schreibers braucht diese
// Erwartung VOR seinem eigentlichen Statement (siehe StelleFaecherSicher).
func erwarteFachBekannt(mock pgxmock.PgxPoolIface, fach string) {
	mock.ExpectQuery(`SELECT bezeichnung FROM systematik_kategorien WHERE lower\(bezeichnung\) = lower\(\$1\)`).
		WithArgs(fach).
		WillReturnRows(pgxmock.NewRows([]string{"bezeichnung"}).AddRow(fach))
}

func TestStelleFaecherSicher_Bekannt(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT bezeichnung FROM systematik_kategorien WHERE lower\(bezeichnung\) = lower\(\$1\)`).
		WithArgs("Deutsch").
		WillReturnRows(pgxmock.NewRows([]string{"bezeichnung"}).AddRow("Deutsch"))

	res, err := StelleFaecherSicher(context.Background(), mock, []string{"Deutsch"})
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if res["Deutsch"] != "Deutsch" {
		t.Errorf("erwartet Deutsch, bekommen %q", res["Deutsch"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestStelleFaecherSicher_UnbekanntNeu(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	// first read fails
	mock.ExpectQuery(`SELECT bezeichnung FROM systematik_kategorien WHERE lower\(bezeichnung\) = lower\(\$1\)`).
		WithArgs("Neues Fach").
		WillReturnError(pgx.ErrNoRows)

	// insert succeeds
	mock.ExpectExec(`INSERT INTO systematik_kategorien \(kuerzel, bezeichnung\)`).
		WithArgs("NeuesFach", "Neues Fach").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// second read succeeds
	mock.ExpectQuery(`SELECT bezeichnung FROM systematik_kategorien WHERE lower\(bezeichnung\) = lower\(\$1\)`).
		WithArgs("Neues Fach").
		WillReturnRows(pgxmock.NewRows([]string{"bezeichnung"}).AddRow("Neues Fach"))

	res, err := StelleFaecherSicher(context.Background(), mock, []string{"Neues Fach"})
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if res["Neues Fach"] != "Neues Fach" {
		t.Errorf("erwartet Neues Fach, bekommen %q", res["Neues Fach"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestStelleFaecherSicher_KuerzelKollision(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	// first read fails
	mock.ExpectQuery(`SELECT bezeichnung FROM systematik_kategorien WHERE lower\(bezeichnung\) = lower\(\$1\)`).
		WithArgs("Mathe").
		WillReturnError(pgx.ErrNoRows)

	// first insert fails with unique violation on kuerzel
	pgErr := &pgconn.PgError{
		Code:           "23505",
		ConstraintName: "systematik_kategorien_kuerzel_key",
	}
	mock.ExpectExec(`INSERT INTO systematik_kategorien \(kuerzel, bezeichnung\)`).
		WithArgs("Mathe", "Mathe").
		WillReturnError(pgErr)

	// second insert succeeds with hash suffix
	mock.ExpectExec(`INSERT INTO systematik_kategorien \(kuerzel, bezeichnung\)`).
		WithArgs(pgxmock.AnyArg(), "Mathe").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// second read succeeds
	mock.ExpectQuery(`SELECT bezeichnung FROM systematik_kategorien WHERE lower\(bezeichnung\) = lower\(\$1\)`).
		WithArgs("Mathe").
		WillReturnRows(pgxmock.NewRows([]string{"bezeichnung"}).AddRow("Mathe"))

	res, err := StelleFaecherSicher(context.Background(), mock, []string{"Mathe"})
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if res["Mathe"] != "Mathe" {
		t.Errorf("erwartet Mathe, bekommen %q", res["Mathe"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestStelleFaecherSicher_Mehrere(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	// "Deutsch"
	mock.ExpectQuery(`SELECT bezeichnung FROM systematik_kategorien WHERE lower\(bezeichnung\) = lower\(\$1\)`).
		WithArgs("Deutsch").
		WillReturnRows(pgxmock.NewRows([]string{"bezeichnung"}).AddRow("Deutsch"))

	// "Neues Fach"
	mock.ExpectQuery(`SELECT bezeichnung FROM systematik_kategorien WHERE lower\(bezeichnung\) = lower\(\$1\)`).
		WithArgs("Neues Fach").
		WillReturnError(pgx.ErrNoRows)
	mock.ExpectExec(`INSERT INTO systematik_kategorien \(kuerzel, bezeichnung\)`).
		WithArgs("NeuesFach", "Neues Fach").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectQuery(`SELECT bezeichnung FROM systematik_kategorien WHERE lower\(bezeichnung\) = lower\(\$1\)`).
		WithArgs("Neues Fach").
		WillReturnRows(pgxmock.NewRows([]string{"bezeichnung"}).AddRow("Neues Fach"))

	res, err := StelleFaecherSicher(context.Background(), mock, []string{"Deutsch", "Neues Fach", " ", "Deutsch"})
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(res) != 2 {
		t.Errorf("erwartet 2 Einträge, bekommen %d", len(res))
	}
	if res["Deutsch"] != "Deutsch" || res["Neues Fach"] != "Neues Fach" {
		t.Errorf("falsche Werte in map: %v", res)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}
