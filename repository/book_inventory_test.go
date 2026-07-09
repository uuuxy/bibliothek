package repository

import (
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// Regressionstest: Die Signatur klebt physisch auf dem Buchrücken — der
// Littera-Upsert darf eine befüllte Signatur NIE mit einem Leerwert
// überschreiben. Der Regex fixiert den COALESCE(NULLIF(...))-Schutz im SQL;
// würde jemand auf `signatur = EXCLUDED.signatur` zurückbauen, wird es rot.
func TestUpsertBookTitle_SignaturConflictClauseIsProtected(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)

	mock.ExpectExec(`ON CONFLICT \(isbn\) DO UPDATE SET[\s\S]*signatur = COALESCE\(NULLIF\(EXCLUDED\.signatur, ''\), buecher_titel\.signatur\)`).
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
		t.Errorf("Signatur-Schutzklausel fehlt im Upsert-SQL: %v", err)
	}
}
