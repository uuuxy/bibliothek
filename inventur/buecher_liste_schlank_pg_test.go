package inventur

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// testDBLockKey ist DER paket-übergreifende Advisory-Lock, mit dem sich db/, repository/
// und api/ die eine geteilte Test-DB teilen (`go test ./...` startet ihre Binaries
// parallel). Ohne ihn kollidiert das DROP SCHEMA hier mit den anderen Paketen — der Wert
// muss identisch zu ihrem sein.
const testDBLockKey int64 = 0x42DB0001

// ladeSchemaPool baut auf TEST_DATABASE_URL eine frische Schema-Instanz auf und liefert
// den Pool. Sicherheitsbremse: nur auf einer DB, deren Name "test" enthält (wie in den
// db-/repository-PG-Tests), weil hier DROP SCHEMA läuft. Serialisiert über den geteilten
// Advisory-Lock, damit es die parallel laufenden Pakete nicht unter den Füßen wegzieht.
func ladeSchemaPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL nicht gesetzt — DB-Integrationstest übersprungen")
	}
	ctx := context.Background()

	// Dedizierte Sperr-Verbindung, gehalten bis Testende (Close gibt den Lock frei).
	lockConn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("Sperr-Verbindung: %v", err)
	}
	if _, err := lockConn.Exec(ctx, "SELECT pg_advisory_lock($1)", testDBLockKey); err != nil {
		t.Fatalf("Advisory-Lock: %v", err)
	}
	t.Cleanup(func() { _ = lockConn.Close(context.Background()) }) //nolint:errcheck // Cleanup: Close-Fehler irrelevant

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	var name string
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&name); err != nil {
		t.Fatalf("DB-Name: %v", err)
	}
	if !strings.Contains(strings.ToLower(name), "test") {
		t.Fatalf("Sicherheitsabbruch: Datenbank %q enthält nicht \"test\"", name)
	}
	if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatalf("Schema zurücksetzen: %v", err)
	}
	sql, err := os.ReadFile(filepath.Join("..", "schema.sql"))
	if err != nil {
		t.Fatalf("schema.sql lesen: %v", err)
	}
	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		t.Fatalf("schema.sql laden: %v", err)
	}
	return pool
}

// TestListBooksSchlank_KeineSchwerenFelder belegt die Payload-Verschlankung von
// GET /api/books (Listen-Limits-Sweep 20.08.2026): Die Katalogliste liefert die zwei
// schwersten Spalten (beschreibung, erweiterte_eigenschaften) NICHT mehr mit — die
// Listenansicht und ihre clientseitige Suche brauchen sie nicht. Der Einzel-Read
// (ListBooksByIDs, für Detail/Bearbeiten) liefert sie weiterhin vollständig.
func TestListBooksSchlank_KeineSchwerenFelder(t *testing.T) {
	pool := ladeSchemaPool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)

	const grosseBeschreibung = "Sehr langer Beschreibungstext, der die Payload aufblähen würde. " +
		"Wiederholt sich, damit der Unterschied messbar ist."
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, beschreibung, erweiterte_eigenschaften)
		VALUES ('Schlank-Test', 'Autor', $1, '{"regal":"A1","notiz":"vertraulich"}'::jsonb)
		RETURNING id`, grosseBeschreibung).Scan(&id); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'SCHLANK-1')`, id); err != nil {
		t.Fatalf("Exemplar: %v", err)
	}

	// Listenansicht: Titel da, aber die schweren Felder leer.
	liste, err := repo.ListBooks(ctx, "", nil, "")
	if err != nil {
		t.Fatalf("ListBooks: %v", err)
	}
	var gefunden *Book
	for i := range liste {
		if liste[i].ID == id {
			gefunden = &liste[i]
		}
	}
	if gefunden == nil {
		t.Fatal("Titel fehlt in der Katalogliste")
	}
	if gefunden.Beschreibung != "" {
		t.Errorf("Listenansicht schleppt beschreibung mit (%d Zeichen) — Payload nicht verschlankt", len(gefunden.Beschreibung))
	}
	if len(gefunden.ErweiterteEigenschaften) != 0 {
		t.Errorf("Listenansicht schleppt erweiterte_eigenschaften mit: %v", gefunden.ErweiterteEigenschaften)
	}
	// Stammdaten müssen aber vollständig da sein (Suche/Karte hängen daran).
	if gefunden.Title != "Schlank-Test" || gefunden.Author != "Autor" {
		t.Errorf("Stammdaten der Listenansicht unvollständig: %+v", gefunden)
	}

	// Detail-Read: schwere Felder vollständig.
	detail, err := repo.ListBooksByIDs(ctx, []string{id})
	if err != nil {
		t.Fatalf("ListBooksByIDs: %v", err)
	}
	if len(detail) != 1 {
		t.Fatalf("Detail-Read lieferte %d statt 1", len(detail))
	}
	if detail[0].Beschreibung != grosseBeschreibung {
		t.Errorf("Detail-Read muss die volle beschreibung liefern, war %q", detail[0].Beschreibung)
	}
	if detail[0].ErweiterteEigenschaften["regal"] != "A1" {
		t.Errorf("Detail-Read muss erweiterte_eigenschaften liefern, war %v", detail[0].ErweiterteEigenschaften)
	}
}
