package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// seedSuchSchueler legt einen Schüler mit frei wählbarem Vor- UND Nachnamen an.
// Der vorhandene seedSchueler nagelt den Nachnamen auf "Test" fest und taugt für
// Namenssuchtests deshalb nicht.
func seedSuchSchueler(t *testing.T, pool *pgxpool.Pool, barcode, vorname, nachname string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 VALUES ($1, $2, $3, '7a', 2030) RETURNING id`, barcode, vorname, nachname).Scan(&id); err != nil {
		t.Fatalf("Schüler %q %q anlegen: %v", vorname, nachname, err)
	}
	return id
}

// namenDerTreffer bildet die Ergebnisse auf "Vorname Nachname" ab — so lesen sich
// die Fehlermeldungen wie die Trefferliste in der Omnibox.
func namenDerTreffer(treffer []Student) []string {
	namen := make([]string, 0, len(treffer))
	for _, s := range treffer {
		namen = append(namen, s.Vorname+" "+s.Nachname)
	}
	return namen
}

func enthaelt(namen []string, gesucht string) bool {
	for _, n := range namen {
		if n == gesucht {
			return true
		}
	}
	return false
}

// TestSearchStudentsFuzzy_ReihenfolgeUndDiakritika ist der eigentliche Beleg für den
// Umbau der Namenssuche. Alles hier ist gegen echtes Postgres nötig: ILIKE-Semantik,
// unaccent() und der IMMUTABLE-Wrapper existieren im Mock nicht, ein Unit-Test würde
// grün laufen und trotzdem nichts über das Verhalten an der Theke aussagen.
func TestSearchStudentsFuzzy_ReihenfolgeUndDiakritika(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	seedSuchSchueler(t, pool, "SUCH-1", "Lena", "Hoffmann")
	seedSuchSchueler(t, pool, "SUCH-2", "Anna Maria", "Müller-Schmidt")
	seedSuchSchueler(t, pool, "SUCH-3", "José", "García Rodríguez")
	seedSuchSchueler(t, pool, "SUCH-4", "Mehmet", "Öztürk")
	seedSuchSchueler(t, pool, "SUCH-5", "Lena", "Bauer")
	seedSuchSchueler(t, pool, "SUCH-6", "Jörg", "Straßburger")
	// Umgekehrter Fall: Der Name steht bereits in Ersatzschreibung in der Datenbank
	// (so kommt er aus manchen Importen) und wird mit Umlaut gesucht.
	seedSuchSchueler(t, pool, "SUCH-7", "Kaethe", "Schaefer")

	repo := NewStudentRepository(pool)

	faelle := []struct {
		name    string
		eingabe string
		erwarte string
	}{
		// Kernforderung: Die Reihenfolge der Namensteile darf keine Rolle spielen.
		{"nur Vorname", "Lena", "Lena Hoffmann"},
		{"nur Nachname", "Hoffmann", "Lena Hoffmann"},
		{"Vorname zuerst", "Lena Hoffmann", "Lena Hoffmann"},
		{"Nachname zuerst", "Hoffmann Lena", "Lena Hoffmann"},
		{"Groß-/Kleinschreibung egal", "hOfFmAnN lEnA", "Lena Hoffmann"},
		{"überflüssige Leerzeichen", "  Hoffmann    Lena  ", "Lena Hoffmann"},

		// Mehrteilige Namen: jedes Teilstück muss einzeln und kombiniert greifen.
		{"zweiter Vorname allein", "Maria", "Anna Maria Müller-Schmidt"},
		{"Teil des Bindestrichnamens", "Schmidt", "Anna Maria Müller-Schmidt"},
		{"dreiteilig gemischt", "Schmidt Anna Maria", "Anna Maria Müller-Schmidt"},
		{"spanischer Zweitnachname", "Rodriguez Jose", "José García Rodríguez"},

		// Diakritika: an der Theke wird ohne Sonderzeichen getippt.
		{"Umlaut ohne Punkte", "Ozturk", "Mehmet Öztürk"},
		{"Umlaut ohne Punkte, Vorname", "Muller-Schmidt", "Anna Maria Müller-Schmidt"},
		{"Akzent weggelassen", "Garcia", "José García Rodríguez"},
		{"Akzent im Vornamen weggelassen", "Jose Garcia", "José García Rodríguez"},
		{"mit Sonderzeichen getippt", "Öztürk", "Mehmet Öztürk"},

		// Deutsche Ersatzschreibung — in BEIDE Richtungen, denn mal steht der Umlaut
		// in der Datenbank und wird ohne getippt, mal ist es genau andersherum.
		{"ue statt ü", "Mueller-Schmidt", "Anna Maria Müller-Schmidt"},
		{"oe und ue statt ö/ü", "Oeztuerk", "Mehmet Öztürk"},
		{"oe/ue mit Vornamen kombiniert", "Mehmet Oeztuerk", "Mehmet Öztürk"},
		{"ss statt ß", "Strassburger", "Jörg Straßburger"},
		{"ss statt ß mit oe-Vorname", "Joerg Strassburger", "Jörg Straßburger"},
		{"ß getippt, ss egal", "Straßburger", "Jörg Straßburger"},
		{"ae statt ä", "Schaefer", "Kaethe Schaefer"},
		{"Umlaut getippt, Ersatzschreibung gespeichert", "Schäfer", "Kaethe Schaefer"},
		{"Umlaut getippt, beide Namensteile ersetzt", "Käthe Schäfer", "Kaethe Schaefer"},

		// Der Ausweis bleibt als eigener Treffweg erhalten.
		{"Barcode", "SUCH-4", "Mehmet Öztürk"},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			treffer, _, err := repo.SearchStudentsFuzzy(ctx, f.eingabe, 10)
			if err != nil {
				t.Fatalf("Suche %q: %v", f.eingabe, err)
			}
			namen := namenDerTreffer(treffer)
			if !enthaelt(namen, f.erwarte) {
				t.Errorf("Suche %q fand %q nicht — geliefert: %v", f.eingabe, f.erwarte, namen)
			}
		})
	}
}

// TestSearchStudentsFuzzy_TrenntNamensgleiche sichert die Kernaussage des Umbaus ab:
// Vor- plus Nachname engt auf eine Person ein. Vorher lieferte "Lena Hoffmann" gar
// nichts, weil keine Spalte den kompletten String enthielt.
func TestSearchStudentsFuzzy_TrenntNamensgleiche(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	seedSuchSchueler(t, pool, "TRENN-1", "Lena", "Hoffmann")
	seedSuchSchueler(t, pool, "TRENN-2", "Lena", "Bauer")
	seedSuchSchueler(t, pool, "TRENN-3", "Lena", "Fischer")

	repo := NewStudentRepository(pool)

	alle, _, err := repo.SearchStudentsFuzzy(ctx, "Lena", 10)
	if err != nil {
		t.Fatalf("Suche 'Lena': %v", err)
	}
	if len(alle) != 3 {
		t.Errorf("'Lena' sollte 3 Schüler finden, fand %d: %v", len(alle), namenDerTreffer(alle))
	}

	eine, _, err := repo.SearchStudentsFuzzy(ctx, "Lena Hoffmann", 10)
	if err != nil {
		t.Fatalf("Suche 'Lena Hoffmann': %v", err)
	}
	if len(eine) != 1 || eine[0].Nachname != "Hoffmann" {
		t.Errorf("'Lena Hoffmann' sollte genau Lena Hoffmann finden, lieferte: %v", namenDerTreffer(eine))
	}
}

// TestSearchStudentsFuzzy_GesamtzahlTrotzLimit belegt, dass die Trefferzahl die
// vollständige Menge nennt und nicht die Länge der gekürzten Liste. Ohne das steht
// im Dropdown "Schüler (2)", obwohl es fünf sind — die Kraft an der Theke hätte
// keinen Anlass, weiterzutippen.
func TestSearchStudentsFuzzy_GesamtzahlTrotzLimit(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	for _, n := range []string{"Bauer", "Becker", "Braun", "Demir", "Fischer"} {
		seedSuchSchueler(t, pool, "LIMIT-"+n, "Maximilian", n)
	}

	repo := NewStudentRepository(pool)

	treffer, gesamt, err := repo.SearchStudentsFuzzy(ctx, "Maximilian", 2)
	if err != nil {
		t.Fatalf("Suche: %v", err)
	}
	if len(treffer) != 2 {
		t.Errorf("Limit 2 nicht eingehalten: %d Treffer", len(treffer))
	}
	if gesamt != 5 {
		t.Errorf("Gesamtzahl sollte 5 sein (alle Maximilians), war %d", gesamt)
	}
}

// TestSearchStudentsFuzzy_RanktVollenNamenNachOben prüft die Sortierung: Wer beide
// Namensteile tippt, will die passende Person oben sehen und nicht alphabetisch
// irgendwo. Bauer stünde sonst vor Hoffmann.
func TestSearchStudentsFuzzy_RanktVollenNamenNachOben(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	seedSuchSchueler(t, pool, "RANK-1", "Lena", "Bauer")
	seedSuchSchueler(t, pool, "RANK-2", "Hoffmann", "Lenau") // Namensteile vertauscht als Störtreffer
	seedSuchSchueler(t, pool, "RANK-3", "Lena", "Hoffmann")

	repo := NewStudentRepository(pool)

	treffer, _, err := repo.SearchStudentsFuzzy(ctx, "Lena Hoffmann", 10)
	if err != nil {
		t.Fatalf("Suche: %v", err)
	}
	if len(treffer) == 0 {
		t.Fatal("keine Treffer")
	}
	if got := treffer[0].Vorname + " " + treffer[0].Nachname; got != "Lena Hoffmann" {
		t.Errorf("erster Treffer sollte 'Lena Hoffmann' sein, war %q — Reihenfolge: %v", got, namenDerTreffer(treffer))
	}
}

// TestSearchStudentsFuzzy_WildcardsWirkenNichtAlsMuster stellt sicher, dass die
// LIKE-Metazeichen escaped werden. Ohne das Escaping wäre eine getippte "%" eine
// Wildcard und lieferte die halbe Schülerschaft statt keiner Treffer.
func TestSearchStudentsFuzzy_WildcardsWirkenNichtAlsMuster(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	seedSuchSchueler(t, pool, "WILD-1", "Lena", "Hoffmann")
	seedSuchSchueler(t, pool, "WILD-2", "Tim", "Berger")

	repo := NewStudentRepository(pool)

	for _, eingabe := range []string{"%", "_", "%%"} {
		treffer, gesamt, err := repo.SearchStudentsFuzzy(ctx, eingabe, 10)
		if err != nil {
			t.Fatalf("Suche %q: %v", eingabe, err)
		}
		if len(treffer) != 0 || gesamt != 0 {
			t.Errorf("Eingabe %q wirkte als Wildcard: %d Treffer (gesamt %d)", eingabe, len(treffer), gesamt)
		}
	}
}

// TestSearchStudentsFuzzy_LeereEingabe: Ohne Token darf die Suche nicht die gesamte
// Schülerschaft ausschütten.
func TestSearchStudentsFuzzy_LeereEingabe(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	seedSuchSchueler(t, pool, "LEER-1", "Lena", "Hoffmann")

	repo := NewStudentRepository(pool)

	for _, eingabe := range []string{"", "   "} {
		treffer, gesamt, err := repo.SearchStudentsFuzzy(ctx, eingabe, 10)
		if err != nil {
			t.Fatalf("Suche %q: %v", eingabe, err)
		}
		if len(treffer) != 0 || gesamt != 0 {
			t.Errorf("leere Eingabe %q lieferte %d Treffer (gesamt %d)", eingabe, len(treffer), gesamt)
		}
	}
}
