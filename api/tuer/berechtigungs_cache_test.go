package tuer

import (
	"context"
	"strings"
	"testing"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

// invalidiereWaehrendQuery stellt das Rennen deterministisch nach: WÄHREND eine
// Berechtigungs-Entscheidung aus der DB unterwegs ist (nach dem Lesen, vor dem
// Cache-Schreiben), schlägt die Invalidierung zu — z. B. weil ein Admin genau jetzt
// die Rechte-Matrix geändert hat.
type invalidiereWaehrendQuery struct {
	pgxmock.PgxPoolIface
	imFlug func()
}

func (p *invalidiereWaehrendQuery) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	row := p.PgxPoolIface.QueryRow(ctx, sql, args...)
	if strings.Contains(sql, "role_permissions") {
		p.imFlug()
	}
	return row
}

// TestBerechtigungsCache_UeberholterLeserCachtNicht schließt das 60-Sekunden-Fenster
// aus dem Nebenläufigkeits-Audit (19.08.): Die Invalidierung nach einer Rechteänderung
// leert zwar den Cache — aber ein Request, der den ALTEN Stand kurz vorher aus der DB
// gelesen hatte, schrieb ihn DANACH wieder hinein. Das entzogene Recht wirkte dann bis
// zu 60 s weiter. Eine überholte Entscheidung darf den Cache nicht mehr erreichen.
func TestBerechtigungsCache_UeberholterLeserCachtNicht(t *testing.T) {
	InvalidatePermissionCache() // globaler Cache darf nicht zwischen Tests lecken
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock init: %v", err)
	}
	defer mock.Close()
	s := &Basis{DB: &db.Database{Pool: mock}}

	// Die DB liefert den alten Stand: Recht (noch) gewährt.
	mock.ExpectQuery("role_permissions").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"allowed"}).AddRow(true))
	s.DB.Pool = &invalidiereWaehrendQuery{PgxPoolIface: mock, imFlug: InvalidatePermissionCache}

	const cacheKey = "KOLLEGIUM:buch.ausleihen"
	if _, err := s.ermittleUndCacheBerechtigung(t.Context(), "KOLLEGIUM", "buch.ausleihen", cacheKey); err != nil {
		t.Fatalf("Berechtigung ermitteln: %v", err)
	}

	// Der laufende Request durfte mit dem gelesenen Stand antworten — aber der Cache
	// darf die überholte Entscheidung nicht halten: Der nächste Request muss frisch lesen.
	if _, found := leseBerechtigungCache(cacheKey); found {
		t.Fatal("überholte Entscheidung wurde NACH der Invalidierung gecacht — das entzogene Recht wirkt bis zu 60 s weiter")
	}
}

// Die Naht für Tests legt genau die Entscheidung ab, die BesitztRecht und die Rechteprüfung
// der Routen lesen, und die Invalidierung nimmt sie zurück.
func TestMerkeRechtFuerTest_LegtDieEntscheidungAb(t *testing.T) {
	InvalidatePermissionCache()
	t.Cleanup(InvalidatePermissionCache)
	MerkeRechtFuerTest("helfer", "view_students", false)
	MerkeRechtFuerTest("mitarbeiter", "view_students", true)
	for rolle, erwartet := range map[string]bool{"helfer": false, "mitarbeiter": true} {
		erlaubt, gefunden := leseBerechtigungCache(rolle + ":view_students")
		if !gefunden || erlaubt != erwartet {
			t.Errorf("%s: gefunden=%v erlaubt=%v, erwartet eine Entscheidung mit erlaubt=%v", rolle, gefunden, erlaubt, erwartet)
		}
	}
	if _, gefunden := leseBerechtigungCache("helfer:edit_students"); gefunden {
		t.Error("für ein anderes Recht derselben Rolle liegt eine Entscheidung vor")
	}
	InvalidatePermissionCache()
	if _, gefunden := leseBerechtigungCache("helfer:view_students"); gefunden {
		t.Error("nach der Invalidierung liegt die Entscheidung noch vor")
	}
}
