package api

import (
	"context"
	"fmt"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/inventur"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Ein neues Exemplar erbt den Standort, den alle Exemplare seines Titels im Bestand tragen
// (repository.SQLGeerbterStandort, docs/OFFEN.md 5.53). Ein Exemplar kommt auf vier Wegen zu
// einem vorhandenen Titel; jede Ausgangslage läuft deshalb über jeden Weg.

// erbTitel ist ein Titel der Probe: Die Wege finden ihn über Kennung, Namen oder ISBN.
type erbTitel struct{ id, name, isbn string }

// erbWeg legt zwei neue Exemplare an einem vorhandenen Titel an.
type erbWeg struct {
	name       string
	legeZweiAn func(t *testing.T, pool *pgxpool.Pool, titel erbTitel)
}

func erbWege() []erbWeg {
	ctx := context.Background()
	return []erbWeg{
		{"Bestand in der Titelmaske erhöht", func(t *testing.T, pool *pgxpool.Pool, titel erbTitel) {
			repo := inventur.NewBookRepository(pool)
			buecher, err := repo.ListBooksByIDs(ctx, []string{titel.id})
			if err != nil || len(buecher) != 1 {
				t.Fatalf("Titel lesen: %v, %d Titel", err, len(buecher))
			}
			gesehen := buecher[0].Stock
			if err := repo.UpdateBook(ctx, titel.id, buecher[0], nil,
				&inventur.Bestandsangabe{Soll: gesehen + 2, Gesehen: &gesehen}); err != nil {
				t.Fatalf("Bestand erhöhen: %v", err)
			}
		}},
		{"Listenimport mit Stückzahl", func(t *testing.T, pool *pgxpool.Pool, titel erbTitel) {
			if _, err := inventur.NewBookRepository(pool).
				UpsertBooksBatch(ctx, []inventur.Book{{ISBN: titel.isbn, Stock: 2}}); err != nil {
				t.Fatalf("Listenimport: %v", err)
			}
		}},
		{"Listenimport mit Exemplarnummern", func(t *testing.T, pool *pgxpool.Pool, titel erbTitel) {
			rows := [][]string{{"Titel", "Barcode"}, {titel.name, titel.isbn + "-L1"}, {titel.name, titel.isbn + "-L2"}}
			neueTitel, neueExemplare, err := service.NewImportService(nil, pool).
				ImportDynamic(ctx, rows, map[string]int{"titel": 0, "barcode": 1})
			if err != nil || neueTitel != 0 || neueExemplare != 2 {
				t.Fatalf("Import: %v, %d neue Titel, %d neue Exemplare — erwartet 0 und 2", err, neueTitel, neueExemplare)
			}
		}},
		{"Bestellung", func(t *testing.T, pool *pgxpool.Pool, titel erbTitel) {
			svc := NewOrderService(&db.Database{Pool: pool}, repository.NewBookRepository(pool))
			if _, err := svc.ProcessOrder(ctx, SubmitOrderRequest{
				Mittel:     mitteltopf.Land,
				SupplierID: haendler(t, pool, "Händler "+titel.isbn, false),
				Items:      []OrderItemRequest{{TitelID: titel.id, Menge: 2, Preis: 10, GenerateBarcodes: true}},
			}); err != nil {
				t.Fatalf("Bestellung: %v", err)
			}
		}},
	}
}

// erbLage ist der Titel vor dem neuen Exemplar: die Standorte seiner Exemplare im Bestand
// („" = keiner), mitRand dazu ein ausgesondertes im „Keller" und ein bestelltes ohne Standort.
type erbLage struct {
	name      string
	imBestand []string
	mitRand   bool
	erbe      string
}

func TestExemplarStandort_NeuesExemplarErbt(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// Ein Nachbar mit eigenem Standort: Die Regel liest nur die Exemplare des eigenen Titels.
	nachbar := titelMitSignatur(t, pool, "Nachbar im Keller", "Ju Nac", 0)
	for i := range 2 {
		id := exemplar(t, pool, nachbar, fmt.Sprintf("ERB-NACHBAR-%d", i), true, "")
		if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET standort = 'Keller' WHERE id = $1`, id); err != nil {
			t.Fatalf("Nachbar: %v", err)
		}
	}

	lagen := []erbLage{
		{"alle tragen denselben", []string{"Regal 11", "Regal 11"}, false, "Regal 11"},
		{"eines trägt keinen", []string{"Regal 11", ""}, false, "(NULL)"},
		{"zwei verschiedene", []string{"Regal 11", "Lehrerschrank"}, false, "(NULL)"},
		{"kein Exemplar im Bestand", nil, false, "(NULL)"},
		{"ausgesonderte und bestellte zählen nicht", []string{"Regal 11"}, true, "Regal 11"},
	}
	nr := 0
	for _, weg := range erbWege() {
		for _, lage := range lagen {
			nr++
			titel := erbTitel{name: fmt.Sprintf("Erbprobe %d", nr), isbn: fmt.Sprintf("978999400%04d", nr)}
			t.Run(weg.name+"/"+lage.name, func(t *testing.T) {
				if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, autor, isbn) VALUES ($1, 'Probe', $2)
					RETURNING id::text`, titel.name, titel.isbn).Scan(&titel.id); err != nil {
					t.Fatalf("Titel anlegen: %v", err)
				}
				stelleErbLageHer(t, pool, titel, lage)

				weg.legeZweiAn(t, pool, titel)

				rows, err := pool.Query(ctx, `SELECT coalesce(standort, '(NULL)') FROM buecher_exemplare
					WHERE titel_id = $1 AND barcode_id NOT LIKE 'ERB-ALT-%'`, titel.id)
				if err != nil {
					t.Fatalf("neue Exemplare lesen: %v", err)
				}
				defer rows.Close()
				neue := 0
				for rows.Next() {
					var standort string
					if err := rows.Scan(&standort); err != nil {
						t.Fatalf("neue Exemplare lesen: %v", err)
					}
					neue++
					if standort != lage.erbe {
						t.Errorf("neues Exemplar steht in %q, erwartet %q", standort, lage.erbe)
					}
				}
				if err := rows.Err(); err != nil {
					t.Fatalf("neue Exemplare lesen: %v", err)
				}
				if neue != 2 {
					t.Fatalf("%d neue Exemplare, erwartet 2 — der Test misst nicht, was er soll", neue)
				}
			})
		}
	}
}

// stelleErbLageHer legt die Exemplare an, die der Titel vor dem neuen trägt. Ihre Nummern
// beginnen mit ERB-ALT, daran trennt der Test sie von den neuen.
func stelleErbLageHer(t *testing.T, pool *pgxpool.Pool, titel erbTitel, lage erbLage) {
	t.Helper()
	ctx := context.Background()
	setze := func(id, zusatz string, werte ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET `+zusatz+` WHERE id = $1`, append([]any{id}, werte...)...); err != nil {
			t.Fatalf("Ausgangslage: %v", err)
		}
	}
	for i, standort := range lage.imBestand {
		id := exemplar(t, pool, titel.id, fmt.Sprintf("ERB-ALT-%s-%d", titel.isbn, i), true, "")
		if standort != "" {
			setze(id, `standort = $2`, standort)
		}
	}
	if !lage.mitRand {
		return
	}
	weg := exemplar(t, pool, titel.id, "ERB-ALT-"+titel.isbn+"-WEG", true, "")
	setze(weg, `standort = 'Keller', ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'AUSSORTIERT'`)
	bestellt := exemplar(t, pool, titel.id, "ERB-ALT-"+titel.isbn+"-BESTELLT", false, "")
	setze(bestellt, `bestellstatus = 'bestellt'`)
}
