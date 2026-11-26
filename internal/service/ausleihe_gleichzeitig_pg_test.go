package service_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"bibliothek/internal/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/repository"
	"bibliothek/internal/service"
	"bibliothek/internal/testdaten"

	"github.com/jackc/pgx/v5/pgconn"
)

// Zwanzig Theken scannen dasselbe freie Exemplar im selben Augenblick, jede mit einem anderen
// Leser. Eine verleiht es, die übrigen bekommen eine Ablehnung oder nehmen es als fremde Rückgabe
// zurück, eine nach der anderen. Keine scheitert an der Datenbank: Alle Wege sperren in derselben
// Reihenfolge (ausleihe_zeilensperren.go), sonst warten zwei aufeinander, bis die Datenbank eine
// mit 40P01 abbricht, und die Theke zeigt einen Serverfehler.
//
// Jeder Leser scannt nur einmal. Scannte der Ausleiher selbst, während eine andere Theke sein Buch
// zurücknimmt, griffe der Rückgabe-Trigger in umgekehrter Folge auf seine Zeile zu; das ist
// entschieden und bleibt so (invarianten.md, Abschnitt 1, Sperrreihenfolge).
func TestBucheAmLeser_ZwanzigThekenEinExemplar(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	bearbeiter := testdaten.LegeBearbeiterAn(t, pool)
	titel := testdaten.LegeTitelAn(t, pool, "Gleichzeitig-Titel")
	testdaten.LegeExemplarAn(t, pool, titel, "B-GLEICH-1", true, "")
	const theken, runden = 20, 10
	leser := make([]string, theken*runden)
	for i := range leser {
		leser[i] = testdaten.LegeSchuelerAn(t, pool, fmt.Sprintf("S-GLEICH-%03d", i), "Kind", "7a")
	}
	// Der Dienst bekommt den Pool des Servers (db.Verbinde, 50 Verbindungen): Er liest unter der
	// Sperre auch über den Pool, und zwanzig Transaktionen erschöpften den kleinen Pool des Tests,
	// bevor die Datenbank etwas zu entscheiden hätte.
	verbindung, err := db.Verbinde(ctx, os.Getenv(pgtest.Variable))
	if err != nil {
		t.Fatalf("Pool wie im Betrieb: %v", err)
	}
	t.Cleanup(verbindung.Close)
	betrieb := verbindung.Pool
	katalog := repository.NewKatalogAbfragen(betrieb)
	dienst := service.NewAusleihDienst(betrieb, repository.NewLeserAbfragen(betrieb), katalog,
		repository.NewAusleihAbfragen(betrieb), repository.NewProtokollAbfragen(betrieb))
	exemplar, err := katalog.FindeExemplar(ctx, "B-GLEICH-1")
	if err != nil {
		t.Fatalf("Exemplar lesen: %v", err)
	}

	for runde := range runden {
		if _, err := pool.Exec(ctx, `UPDATE ausleihen SET rueckgabe_am = now()
			WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`, exemplar.ID); err != nil {
			t.Fatalf("Exemplar vor Runde %d freigeben: %v", runde, err)
		}
		start := make(chan struct{})
		fehler := make([]error, theken)
		var wg sync.WaitGroup
		for i := range theken {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				// Gestaffelt wie Anfragen über das Netz: Wer später kommt, findet die Ausleihe des
				// ersten schon vor und nimmt sie zurück, während frühere noch auf das Exemplar warten.
				time.Sleep(time.Duration(i) * 400 * time.Microsecond)
				_, fehler[i] = dienst.BucheAmLeser(ctx, exemplar, &leser[runde*theken+i], bearbeiter, false)
			}()
		}
		close(start)
		wg.Wait()

		for i, err := range fehler {
			var pgFehler *pgconn.PgError
			if errors.As(err, &pgFehler) {
				t.Fatalf("Runde %d, Theke %d: die Datenbank brach ab (%s): %v", runde, i, pgFehler.Code, err)
			}
		}
		var offen int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ausleihen a JOIN buecher_exemplare e ON e.id = a.exemplar_id
			WHERE e.barcode_id = 'B-GLEICH-1' AND a.rueckgabe_am IS NULL`).Scan(&offen); err != nil {
			t.Fatalf("offene Ausleihen zählen: %v", err)
		}
		if offen > 1 {
			t.Fatalf("Runde %d: %d offene Ausleihen eines Exemplars", runde, offen)
		}
	}
}
