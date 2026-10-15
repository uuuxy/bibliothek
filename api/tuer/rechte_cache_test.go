package tuer

import (
	"context"
	"strings"
	"testing"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

// leereWaehrendAbfrage stellt das Rennen fest nach: Während eine Entscheidung aus der Datenbank
// unterwegs ist (nach dem Lesen, vor dem Schreiben in den Cache), wird der Cache geleert, etwa
// weil ein Admin gerade die Rechte geändert hat.
type leereWaehrendAbfrage struct {
	pgxmock.PgxPoolIface
	imFlug func()
}

func (p *leereWaehrendAbfrage) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	row := p.PgxPoolIface.QueryRow(ctx, sql, args...)
	if strings.Contains(sql, "role_permissions") {
		p.imFlug()
	}
	return row
}

// TestRechteCache_UeberholteEntscheidungBleibtDraussen: Eine Anfrage, die den alten Stand kurz
// vor einer Rechteänderung gelesen hat, schreibt ihn nach dem Leeren nicht zurück in den Cache.
// Sonst wirkte das entzogene Recht bis zu 60 s weiter.
func TestRechteCache_UeberholteEntscheidungBleibtDraussen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock init: %v", err)
	}
	defer mock.Close()
	s := &Basis{DB: &db.Verbindung{Pool: mock}}

	// Die DB liefert den alten Stand: Recht (noch) gewährt.
	mock.ExpectQuery("role_permissions").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"allowed"}).AddRow(true))
	s.DB.Pool = &leereWaehrendAbfrage{PgxPoolIface: mock, imFlug: s.Rechte.Leere}

	const schluessel = "KOLLEGIUM:buch.ausleihen"
	if _, err := s.ladeUndMerkeRecht(t.Context(), "KOLLEGIUM", "buch.ausleihen", schluessel); err != nil {
		t.Fatalf("Berechtigung ermitteln: %v", err)
	}

	// Die laufende Anfrage durfte mit dem gelesenen Stand antworten, der Cache hält ihn aber
	// nicht: Die nächste Anfrage liest neu.
	if _, found := s.Rechte.recht(schluessel); found {
		t.Fatal("überholte Entscheidung wurde NACH der Invalidierung gecacht — das entzogene Recht wirkt bis zu 60 s weiter")
	}
}

// Die Naht für Tests legt genau die Entscheidung ab, die HatRecht und die Prüfung der Routen
// lesen, und das Leeren des Caches nimmt sie zurück.
func TestRechteCache_MerkeFuerTestLegtDieEntscheidungAb(t *testing.T) {
	var c RechteCache
	c.MerkeFuerTest("helfer", "view_students", false)
	c.MerkeFuerTest("mitarbeiter", "view_students", true)
	for rolle, erwartet := range map[string]bool{"helfer": false, "mitarbeiter": true} {
		erlaubt, gefunden := c.recht(rolle + ":view_students")
		if !gefunden || erlaubt != erwartet {
			t.Errorf("%s: gefunden=%v erlaubt=%v, erwartet eine Entscheidung mit erlaubt=%v", rolle, gefunden, erlaubt, erwartet)
		}
	}
	if _, gefunden := c.recht("helfer:edit_students"); gefunden {
		t.Error("für ein anderes Recht derselben Rolle liegt eine Entscheidung vor")
	}
	c.Leere()
	if _, gefunden := c.recht("helfer:view_students"); gefunden {
		t.Error("nach der Invalidierung liegt die Entscheidung noch vor")
	}
}
