package repository

import (
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// Regressionstest: Die Signatur klebt physisch auf dem Buchrücken — der
// Littera-Upsert darf eine befüllte Signatur NIE mit einem Leerwert
// überschreiben. Gleiches gilt für Autor, Verlag und Erscheinungsjahr.
// Der Regex fixiert den COALESCE(NULLIF(...))-Schutz im SQL;
// würde jemand auf `feld = EXCLUDED.feld` zurückbauen, wird es rot.
func TestUpsertBookTitle_ConflictClauseIsProtected(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)

	mock.ExpectExec(`ON CONFLICT \(isbn\) DO UPDATE SET[\s\S]*autor = COALESCE\(NULLIF\(EXCLUDED\.autor, ''\), buecher_titel\.autor\),[\s\S]*verlag = COALESCE\(NULLIF\(EXCLUDED\.verlag, ''\), buecher_titel\.verlag\),[\s\S]*erscheinungsjahr = COALESCE\(NULLIF\(EXCLUDED\.erscheinungsjahr, 0\), buecher_titel\.erscheinungsjahr\),[\s\S]*signatur = COALESCE\(NULLIF\(EXCLUDED\.signatur, ''\), buecher_titel\.signatur\)`).
		WithArgs("Faust", "Goethe", "978-1", "Reclam", 1999, "", 0).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.UpsertBookTitle(t.Context(), BookTitle{
		Titel: "Faust", Autor: "Goethe", ISBN: "978-1", Verlag: "Reclam",
		Erscheinungsjahr: 1999, Signatur: "", // leer — darf Bestand nicht anfassen
	})
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Schutzklausel fehlt im Upsert-SQL: %v", err)
	}
}
