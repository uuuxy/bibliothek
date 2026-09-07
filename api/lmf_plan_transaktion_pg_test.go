package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Plan-Schreibung und Frist-Kopplung sind EINE Transaktion.
//
// Register 06.09.2026 (B): Bis heute committete SaveLmfPlan selbst, und die Kopplung
// lief danach am Pool — eine Schleife über die Klassen, jede ein eigenes UPDATE auf
// ausleihen. Brach sie mittendrin ab (Verbindung weg, Sperre, Timeout), war der Plan
// geschrieben, die Antwort ein 500, und die Fristen standen zur Hälfte auf dem neuen
// Termin. Ein zweiter Anlauf heilte das nicht: Die Verlierer-Klassen rechnet der Handler
// aus dem ALTEN Plan, und der war schon überschrieben. Die Oberfläche lud neu und zeigte
// den echten Plan — die halb umgeschriebenen Fristen sah niemand.
//
// Der Beweis braucht einen Fehler an genau der richtigen Stelle. Kein Mock des Brokers,
// kein pgxmock: der ECHTE Pool, eingewickelt, und seine Transaktion lässt genau das
// Fristen-UPDATE scheitern. Alles andere — Plan lesen, Plan schreiben — läuft echt.
type poolMitFehlerBei struct {
	db.PgxPoolIface
	muster string
}

// Exec am Pool scheitert ebenfalls: So fällt auch die ALTE Form (Plan committet, Kopplung
// am Pool) an der richtigen Stelle — und nicht schon daran, dass kein Fehler ankommt.
func (p poolMitFehlerBei) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, p.muster) {
		return pgconn.CommandTag{}, errors.New("Testfehler mitten in der Frist-Kopplung: " + p.muster)
	}
	return p.PgxPoolIface.Exec(ctx, sql, args...)
}

func (p poolMitFehlerBei) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := p.PgxPoolIface.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return txMitFehlerBei{Tx: tx, muster: p.muster}, nil
}

type txMitFehlerBei struct {
	pgx.Tx
	muster string
}

func (t txMitFehlerBei) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, t.muster) {
		return pgconn.CommandTag{}, errors.New("Testfehler mitten in der Frist-Kopplung: " + t.muster)
	}
	return t.Tx.Exec(ctx, sql, args...)
}

func TestLmfPlan_ScheitertDieFristKopplung_BleibtAuchDerPlanUnangetastet(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM lmf_plaene`); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})
	echt := &Server{DB: &db.Database{Pool: pool}}
	repo := repository.NewLmfTerminRepository(pool)

	stichtag := time.Date(2027, time.July, 31, 23, 59, 0, 0, schulzeit.Zone())
	anna := seedSchueler(t, pool, "TX-1", "Anna", "9H1")
	annaLmf := seedAusleihe(t, pool, anna, "LMF Mathe 9 Anna", stichtag)

	// Ein veröffentlichter Plan mit 9H1 — Annas Frist folgt seinem Termin.
	if rec := lmfPlanAufruf(t, echt, http.MethodPut, "rueckgabe",
		`{"letzter_tag":"2027-06-28","letzte_stunde":3,"stunden_je_tag":6,"zeilen":[{"klassen":["9H1"]}]}`); rec.Code != http.StatusOK {
		t.Fatalf("Plan anlegen: %d %s", rec.Code, rec.Body.String())
	}
	if rec := lmfPlanAufruf(t, echt, http.MethodPost, "rueckgabe", ""); rec.Code != http.StatusOK {
		t.Fatalf("veröffentlichen: %d %s", rec.Code, rec.Body.String())
	}
	fristVorher := fristVon(t, pool, annaLmf)
	if fristVorher.Equal(stichtag) {
		t.Fatal("Aufbau: Annas Frist folgt dem Plan nicht — der Test prüfte dann ins Leere")
	}
	planVorher, err := repo.NeuesterLmfPlan(ctx, "rueckgabe")
	if err != nil {
		t.Fatal(err)
	}

	// Jetzt derselbe Server, nur dass seine Transaktion beim Fristen-UPDATE scheitert.
	kaputt := &Server{DB: &db.Database{Pool: poolMitFehlerBei{PgxPoolIface: pool, muster: "UPDATE ausleihen"}}}
	rec := lmfPlanAufruf(t, kaputt, http.MethodPut, "rueckgabe",
		`{"letzter_tag":"2027-06-28","letzte_stunde":4,"stunden_je_tag":6,"zeilen":[{"klassen":["8G1"]},{"klassen":["9H1"]}]}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("der eingebaute Fehler kam nicht an: Status %d %s", rec.Code, rec.Body.String())
	}

	// Der Kern: NICHTS ist geschehen — weder am Plan noch an den Fristen.
	planNachher, err := repo.NeuesterLmfPlan(ctx, "rueckgabe")
	if err != nil {
		t.Fatal(err)
	}
	if len(planNachher.Zeilen) != len(planVorher.Zeilen) {
		t.Errorf("der Plan wurde trotz gescheiterter Kopplung geschrieben: %d Zeilen, vorher %d — "+
			"Plan und Fristen liefen in zwei Transaktionen", len(planNachher.Zeilen), len(planVorher.Zeilen))
	} else if planNachher.Zeilen[0].Datum != planVorher.Zeilen[0].Datum || planNachher.Zeilen[0].Stunde != planVorher.Zeilen[0].Stunde {
		t.Errorf("der Plan wurde verändert: %+v, vorher %+v", planNachher.Zeilen[0], planVorher.Zeilen[0])
	}
	if planNachher.Plan.LetzteStunde != planVorher.Plan.LetzteStunde {
		t.Errorf("der Rahmen wurde verändert: letzte Stunde %d, vorher %d", planNachher.Plan.LetzteStunde, planVorher.Plan.LetzteStunde)
	}
	if ist := fristVon(t, pool, annaLmf); !ist.Equal(fristVorher) {
		t.Errorf("Annas Frist hat sich bewegt: %v, vorher %v", ist.In(schulzeit.Zone()), fristVorher.In(schulzeit.Zone()))
	}

	// Gegenprobe: Ohne den eingebauten Fehler geht dieselbe Änderung durch — sonst wäre
	// nicht zu unterscheiden, ob die Klammer hält oder der Schreibweg kaputt ist.
	if rec := lmfPlanAufruf(t, echt, http.MethodPut, "rueckgabe",
		`{"letzter_tag":"2027-06-28","letzte_stunde":4,"stunden_je_tag":6,"zeilen":[{"klassen":["8G1"]},{"klassen":["9H1"]}]}`); rec.Code != http.StatusOK {
		t.Fatalf("Gegenprobe: dieselbe Änderung ohne Fehler: %d %s", rec.Code, rec.Body.String())
	}
	if st, err := repo.NeuesterLmfPlan(ctx, "rueckgabe"); err != nil || len(st.Zeilen) != 2 {
		t.Errorf("Gegenprobe: Plan nach dem echten Speichern: %d Zeilen (%v), erwartet 2", len(st.Zeilen), err)
	}
}
