package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// seedSuchTitel legt einen Buchtitel mit Titel, Autor, ISBN und Signatur an.
func seedSuchTitel(t *testing.T, pool *pgxpool.Pool, titel, autor, isbn, signatur string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO buecher_titel (titel, autor, isbn, signatur)
		 VALUES ($1, $2, $3, $4) RETURNING id`, titel, autor, isbn, signatur).Scan(&id); err != nil {
		t.Fatalf("Titel %q anlegen: %v", titel, err)
	}
	return id
}

func titelDerTreffer(treffer []BookTitle) []string {
	namen := make([]string, 0, len(treffer))
	for _, b := range treffer {
		namen = append(namen, b.Titel)
	}
	return namen
}

// TestSearchTitlesFuzzy_TokenUndDiakritika deckt für die Titelsuche dieselben Fälle
// ab wie die Namenssuche: Die Eingabe wird tokenweise geprüft, sodass Autor und
// Titel gemischt getippt werden können, und Diakritika stehen nicht im Weg.
func TestSearchTitlesFuzzy_TokenUndDiakritika(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	seedSuchTitel(t, pool, "Der Vorleser", "Bernhard Schlink", "978-3-257-06065-3", "SIG-A1")
	seedSuchTitel(t, pool, "Harry Potter und der Stein der Weisen", "J. K. Rowling", "978-3-551-55741-1", "SIG-B2")
	seedSuchTitel(t, pool, "Öl auf Leinwand", "José Saramago", "978-3-499-22222-2", "SIG-C3")
	seedSuchTitel(t, pool, "Buddenbrooks in der Straße", "Thomas Mann", "978-3-596-29431-2", "SIG-D4")
	// Autor bereits in Ersatzschreibung erfasst — die Suche mit Umlaut muss ihn finden.
	seedSuchTitel(t, pool, "Die Blechtrommel", "Guenter Grass", "978-3-423-11821-5", "SIG-E5")

	repo := NewBookRepository(pool)

	faelle := []struct {
		name    string
		eingabe string
		erwarte string
	}{
		{"Titelwort", "Vorleser", "Der Vorleser"},
		{"Autor allein", "Schlink", "Der Vorleser"},
		{"Autor und Titel gemischt", "rowling stein", "Harry Potter und der Stein der Weisen"},
		{"Titel vor Autor", "stein rowling", "Harry Potter und der Stein der Weisen"},
		{"Akzent weggelassen", "Jose Saramago", "Öl auf Leinwand"},
		{"Umlaut weggelassen", "Ol Leinwand", "Öl auf Leinwand"},
		{"ss statt ß im Titel", "Buddenbrooks Strasse", "Buddenbrooks in der Straße"},
		{"ß getippt", "Buddenbrooks Straße", "Buddenbrooks in der Straße"},
		{"ue statt ü im Autor", "Guenter Grass", "Die Blechtrommel"},
		{"Umlaut getippt, Ersatzschreibung gespeichert", "Günter Grass", "Die Blechtrommel"},
		{"ISBN mit Bindestrichen", "978-3-257-06065-3", "Der Vorleser"},
		{"ISBN ohne Bindestriche", "9783257060653", "Der Vorleser"},
		{"Signatur", "SIG-B2", "Harry Potter und der Stein der Weisen"},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			treffer, _, err := repo.SearchTitlesFuzzy(ctx, f.eingabe, 10)
			if err != nil {
				t.Fatalf("Suche %q: %v", f.eingabe, err)
			}
			titel := titelDerTreffer(treffer)
			if !enthaelt(titel, f.erwarte) {
				t.Errorf("Suche %q fand %q nicht — geliefert: %v", f.eingabe, f.erwarte, titel)
			}
		})
	}
}

// TestSearchTitlesFuzzy_GesamtzahlTrotzLimit: wie bei der Schülersuche muss die
// Trefferzahl die volle Menge nennen, nicht die Länge der gekürzten Liste.
func TestSearchTitlesFuzzy_GesamtzahlTrotzLimit(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	for _, n := range []string{"A", "B", "C", "D"} {
		seedSuchTitel(t, pool, "Faust Band "+n, "Goethe", "978-0-00-00000"+n, "SIG-F"+n)
	}

	repo := NewBookRepository(pool)

	treffer, gesamt, err := repo.SearchTitlesFuzzy(ctx, "Faust", 2)
	if err != nil {
		t.Fatalf("Suche: %v", err)
	}
	if len(treffer) != 2 {
		t.Errorf("Limit 2 nicht eingehalten: %d Treffer", len(treffer))
	}
	if gesamt != 4 {
		t.Errorf("Gesamtzahl sollte 4 sein, war %d", gesamt)
	}
}

// TestSearchTitlesFuzzy_LeereEingabe: ohne Token kein Rundumschlag über den Katalog.
func TestSearchTitlesFuzzy_LeereEingabe(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	seedSuchTitel(t, pool, "Der Vorleser", "Bernhard Schlink", "978-3-257-06065-3", "SIG-A1")

	repo := NewBookRepository(pool)

	for _, eingabe := range []string{"", "   "} {
		treffer, gesamt, err := repo.SearchTitlesFuzzy(ctx, eingabe, 10)
		if err != nil {
			t.Fatalf("Suche %q: %v", eingabe, err)
		}
		if len(treffer) != 0 || gesamt != 0 {
			t.Errorf("leere Eingabe %q lieferte %d Treffer (gesamt %d)", eingabe, len(treffer), gesamt)
		}
	}
}
