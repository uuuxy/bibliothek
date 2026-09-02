package inventur

import (
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
