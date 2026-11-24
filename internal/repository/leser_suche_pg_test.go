package repository

import (
	"bibliothek/internal/testdaten"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestNameKompakt_GemeinsameFaelle: Die Datenbank lässt aus einem Namen dieselben Zeichen weg wie
// kompakt im Browser (frontend/src/lib/utils/suchnorm.js); beide lesen die Fälle „kompakt" in
// suchnorm.faelle.json. Sonst fände „dangelo" im Mahnwesen D´Angelo und an der Theke nicht.
func TestNameKompakt_GemeinsameFaelle(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	roh, err := os.ReadFile("../../frontend/src/lib/utils/suchnorm.faelle.json")
	if err != nil {
		t.Fatal(err)
	}
	var pruefung struct {
		Kompakt [][2]string `json:"kompakt"`
	}
	if err := json.Unmarshal(roh, &pruefung); err != nil {
		t.Fatal(err)
	}
	if len(pruefung.Kompakt) < 40 {
		t.Fatalf("%d Fälle „kompakt“, erwartet mindestens 40", len(pruefung.Kompakt))
	}
	trenner := map[string]bool{}
	for _, fall := range pruefung.Kompakt {
		var ist string
		if err := pool.QueryRow(ctx, "SELECT "+sqlNameKompakt("$1::text"), fall[0]).Scan(&ist); err != nil {
			t.Fatalf("%q: %v", fall[0], err)
		}
		if ist != fall[1] {
			t.Errorf("sqlNameKompakt(%q) = %q, erwartet %q", fall[0], ist, fall[1])
		}
		if z := []rune(fall[0]); len(z) == 8 && string(z[:2]) == "Al" && string(z[3:]) == "Kahil" && fall[1] == "alkahil" {
			trenner[string(z[2])] = true
		}
	}

	// Vollprobe über den Grundbereich: Jedes Zeichen, das die Datenbank aus einem Namen weglässt,
	// steht in den Fällen. Sonst wüchse die Liste am Server, ohne dass der Browser es merkte.
	zeilen, err := pool.Query(ctx, `SELECT chr(cp) FROM generate_series(1, 65533) AS cp
		WHERE (cp < 55296 OR cp > 57343)
		  AND suchnorm('x' || chr(cp) || 'z') <> 'xz'
		  AND `+sqlNameKompakt("'x' || chr(cp) || 'z'")+` = 'xz'`)
	if err != nil {
		t.Fatal(err)
	}
	defer zeilen.Close()
	anzahl := 0
	for zeilen.Next() {
		var z string
		if err := zeilen.Scan(&z); err != nil {
			t.Fatal(err)
		}
		anzahl++
		if !trenner[z] {
			t.Errorf("Die Datenbank lässt U+%04X aus Namen weg, die Fälle „kompakt“ kennen es nicht", []rune(z)[0])
		}
	}
	if err := zeilen.Err(); err != nil {
		t.Fatal(err)
	}
	if anzahl != len(trenner) {
		t.Errorf("Die Datenbank lässt %d Zeichen weg, die Fälle führen %d als Trenner", anzahl, len(trenner))
	}
}

// legeSuchSchuelerAn legt einen Schüler mit frei wählbarem Vor- und Nachnamen an.
// Der vorhandene legeSchuelerAn nagelt den Nachnamen auf "Test" fest und taugt für
// Namenssuchtests deshalb nicht.
func legeSuchSchuelerAn(t *testing.T, pool *pgxpool.Pool, barcode, vorname, nachname string) string {
	t.Helper()
	id := testdaten.LegeSchuelerMitNamenAn(t, pool, barcode, vorname, nachname, "7a", 2030)
	return id
}

// namenDerTreffer bildet die Ergebnisse auf "Vorname Nachname" ab — so lesen sich
// die Fehlermeldungen wie die Trefferliste in der Omnibox.
func namenDerTreffer(treffer []Leser) []string {
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

// TestFindeUnscharf_ReihenfolgeUndDiakritika ist der eigentliche Beleg für den
// Umbau der Namenssuche. Alles hier ist gegen echtes Postgres nötig: ILIKE-Semantik,
// unaccent() und der IMMUTABLE-Wrapper existieren im Mock nicht, ein Unit-Test würde
// grün laufen und trotzdem nichts über das Verhalten an der Theke aussagen.
func TestFindeUnscharf_ReihenfolgeUndDiakritika(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	legeSuchSchuelerAn(t, pool, "SUCH-1", "Lena", "Hoffmann")
	legeSuchSchuelerAn(t, pool, "SUCH-2", "Anna Maria", "Müller-Schmidt")
	legeSuchSchuelerAn(t, pool, "SUCH-3", "José", "García Rodríguez")
	legeSuchSchuelerAn(t, pool, "SUCH-4", "Mehmet", "Öztürk")
	legeSuchSchuelerAn(t, pool, "SUCH-5", "Lena", "Bauer")
	legeSuchSchuelerAn(t, pool, "SUCH-6", "Jörg", "Straßburger")
	// Umgekehrter Fall: Der Name steht bereits in Ersatzschreibung in der Datenbank
	// (so kommt er aus manchen Importen) und wird mit Umlaut gesucht.
	legeSuchSchuelerAn(t, pool, "SUCH-7", "Kaethe", "Schaefer")

	repo := NewLeserAbfragen(pool)

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

		// Deutsche Ersatzschreibung — in beide Richtungen, denn mal steht der Umlaut
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
			treffer, _, err := repo.FindeUnscharf(ctx, f.eingabe, 10)
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

// TestFindeUnscharf_TrenntNamensgleiche sichert die Kernaussage des Umbaus ab:
// Vor- plus Nachname engt auf eine Person ein. Ein Vergleich mit dem ganzen Text lieferte
// für "Lena Hoffmann" nichts, weil keine Spalte den kompletten String enthält.
func TestFindeUnscharf_TrenntNamensgleiche(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	legeSuchSchuelerAn(t, pool, "TRENN-1", "Lena", "Hoffmann")
	legeSuchSchuelerAn(t, pool, "TRENN-2", "Lena", "Bauer")
	legeSuchSchuelerAn(t, pool, "TRENN-3", "Lena", "Fischer")

	repo := NewLeserAbfragen(pool)

	alle, _, err := repo.FindeUnscharf(ctx, "Lena", 10)
	if err != nil {
		t.Fatalf("Suche 'Lena': %v", err)
	}
	if len(alle) != 3 {
		t.Errorf("'Lena' sollte 3 Schüler finden, fand %d: %v", len(alle), namenDerTreffer(alle))
	}

	eine, _, err := repo.FindeUnscharf(ctx, "Lena Hoffmann", 10)
	if err != nil {
		t.Fatalf("Suche 'Lena Hoffmann': %v", err)
	}
	if len(eine) != 1 || eine[0].Nachname != "Hoffmann" {
		t.Errorf("'Lena Hoffmann' sollte genau Lena Hoffmann finden, lieferte: %v", namenDerTreffer(eine))
	}
}

// TestFindeUnscharf_GesamtzahlTrotzLimit belegt, dass die Trefferzahl die
// vollständige Menge nennt und nicht die Länge der gekürzten Liste. Ohne das steht
// im Dropdown "Schüler (2)", obwohl es fünf sind — die Kraft an der Theke hätte
// keinen Anlass, weiterzutippen.
func TestFindeUnscharf_GesamtzahlTrotzLimit(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	for _, n := range []string{"Bauer", "Becker", "Braun", "Demir", "Fischer"} {
		legeSuchSchuelerAn(t, pool, "LIMIT-"+n, "Maximilian", n)
	}

	repo := NewLeserAbfragen(pool)

	treffer, gesamt, err := repo.FindeUnscharf(ctx, "Maximilian", 2)
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

// TestFindeUnscharf_UebergehtAnonymisierte: Eine anonymisierte Zeile trägt den Ersatznamen
// „Anonym" und einen Vornamen aus Hex-Ziffern. Fände die Omnibox sie, belegten nach einem
// Schuljahr lauter „Anonym" die zehn Plätze, sobald jemand „An" tippt, und die gesuchte Anna
// stünde nicht in der Liste.
func TestFindeUnscharf_UebergehtAnonymisierte(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	legeSuchSchuelerAn(t, pool, "ANONSUCH-1", "Anna", "Arndt")
	if _, err := pool.Exec(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger,
		                      ist_gesperrt, block_reason, anonymized_at)
		VALUES ('ANON-SUCH-2', 'a1b2c3d4', 'Anonym', '', 2026, true, true, 'Anonymisiert (DSGVO)', now())`); err != nil {
		t.Fatalf("Seed anonymisiert: %v", err)
	}

	repo := NewLeserAbfragen(pool)
	for _, eingabe := range []string{"An", "Anonym", "a1b2"} {
		treffer, gesamt, err := repo.FindeUnscharf(ctx, eingabe, 10)
		if err != nil {
			t.Fatalf("Suche %q: %v", eingabe, err)
		}
		namen := namenDerTreffer(treffer)
		if enthaelt(namen, "a1b2c3d4 Anonym") {
			t.Errorf("Suche %q findet die anonymisierte Zeile: %v (gesamt %d)", eingabe, namen, gesamt)
		}
	}
	treffer, _, err := repo.FindeUnscharf(ctx, "An", 10)
	if err != nil {
		t.Fatalf("Suche An: %v", err)
	}
	if !enthaelt(namenDerTreffer(treffer), "Anna Arndt") {
		t.Errorf("Suche An findet Anna Arndt nicht: %v", namenDerTreffer(treffer))
	}
}

// TestFindeUnscharf_RanktVollenNamenNachOben prüft die Sortierung: Wer beide
// Namensteile tippt, will die passende Person oben sehen und nicht alphabetisch
// irgendwo. Bauer stünde sonst vor Hoffmann.
func TestFindeUnscharf_RanktVollenNamenNachOben(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	legeSuchSchuelerAn(t, pool, "RANK-1", "Lena", "Bauer")
	legeSuchSchuelerAn(t, pool, "RANK-2", "Hoffmann", "Lenau") // Namensteile vertauscht als Störtreffer
	legeSuchSchuelerAn(t, pool, "RANK-3", "Lena", "Hoffmann")

	repo := NewLeserAbfragen(pool)

	treffer, _, err := repo.FindeUnscharf(ctx, "Lena Hoffmann", 10)
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

// TestFindeUnscharf_WildcardsWirkenNichtAlsMuster stellt sicher, dass die
// LIKE-Metazeichen escaped werden. Ohne das Escaping wäre eine getippte "%" eine
// Wildcard und lieferte die halbe Schülerschaft statt keiner Treffer.
func TestFindeUnscharf_WildcardsWirkenNichtAlsMuster(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	legeSuchSchuelerAn(t, pool, "WILD-1", "Lena", "Hoffmann")
	legeSuchSchuelerAn(t, pool, "WILD-2", "Tim", "Berger")

	repo := NewLeserAbfragen(pool)

	for _, eingabe := range []string{"%", "_", "%%"} {
		treffer, gesamt, err := repo.FindeUnscharf(ctx, eingabe, 10)
		if err != nil {
			t.Fatalf("Suche %q: %v", eingabe, err)
		}
		if len(treffer) != 0 || gesamt != 0 {
			t.Errorf("Eingabe %q wirkte als Wildcard: %d Treffer (gesamt %d)", eingabe, len(treffer), gesamt)
		}
	}
}

// TestFindeUnscharf_LeereEingabe: Ohne Token darf die Suche nicht die gesamte
// Schülerschaft ausschütten.
func TestFindeUnscharf_LeereEingabe(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	legeSuchSchuelerAn(t, pool, "LEER-1", "Lena", "Hoffmann")

	repo := NewLeserAbfragen(pool)

	for _, eingabe := range []string{"", "   "} {
		treffer, gesamt, err := repo.FindeUnscharf(ctx, eingabe, 10)
		if err != nil {
			t.Fatalf("Suche %q: %v", eingabe, err)
		}
		if len(treffer) != 0 || gesamt != 0 {
			t.Errorf("leere Eingabe %q lieferte %d Treffer (gesamt %d)", eingabe, len(treffer), gesamt)
		}
	}
}

// TestFindeUnscharf_TrennerZaehlenNicht: Einen Namen hört man ohne Bindestrich und ohne
// Leerzeichen. „Al Kahil", „Al-Kahil" und „Alkahil" finden sich in jeder Richtung, ebenso
// „Anna-Lena" und „Annalena" und „O'Brien" und „OBrien", an der Theke wie in der Leserdatei;
// ´ und ` stehen für den Apostroph, das geschützte Leerzeichen aus einem Import für das Leerzeichen.
func TestFindeUnscharf_TrennerZaehlenNicht(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	legeSuchSchuelerAn(t, pool, "TRENN-1", "Ahmad", "Al-Kahil")
	legeSuchSchuelerAn(t, pool, "TRENN-2", "Samir", "Al Kahil")
	legeSuchSchuelerAn(t, pool, "TRENN-3", "Yusuf", "Alkahil")
	legeSuchSchuelerAn(t, pool, "TRENN-4", "Anna-Lena", "Brandt")
	legeSuchSchuelerAn(t, pool, "TRENN-5", "Annalena", "Kurz")
	legeSuchSchuelerAn(t, pool, "TRENN-6", "Sean", "O'Brien")
	legeSuchSchuelerAn(t, pool, "TRENN-7", "Luca", "D´Angelo")
	legeSuchSchuelerAn(t, pool, "TRENN-8", "Maria\u00a0Sofia", "Weber")

	repo := NewLeserAbfragen(pool)
	kahil := []string{"Ahmad Al-Kahil", "Samir Al Kahil", "Yusuf Alkahil"}
	annalena := []string{"Anna-Lena Brandt", "Annalena Kurz"}
	brien := []string{"Sean O'Brien"}
	angelo := []string{"Luca D´Angelo"}
	sofia := []string{"Maria\u00a0Sofia Weber"}
	faelle := []struct {
		eingabe string
		erwarte []string
	}{
		{"al kahil", kahil},
		{"al-kahil", kahil},
		{"alkahil", kahil},
		{"AL-KAHIL", kahil},
		{"kahil ahmad", []string{"Ahmad Al-Kahil"}},
		{"annalena", annalena},
		{"anna-lena", annalena},
		{"anna lena", annalena},
		{"obrien", brien},
		{"o'brien", brien},
		{"o’brien", brien},
		{"o´brien", brien},
		{"o`brien", brien},
		{"dangelo", angelo},
		{"d'angelo", angelo},
		{"mariasofia", sofia},
		{"maria-sofia", sofia},
	}

	for _, f := range faelle {
		t.Run(f.eingabe, func(t *testing.T) {
			treffer, _, err := repo.FindeUnscharf(ctx, f.eingabe, 10)
			if err != nil {
				t.Fatalf("Theke %q: %v", f.eingabe, err)
			}
			liste, err := repo.LadeLeserListe(ctx, nil, f.eingabe, LeserSortierung{})
			if err != nil {
				t.Fatalf("Leserdatei %q: %v", f.eingabe, err)
			}
			theke := namenDerTreffer(treffer)
			datei := make([]string, 0, len(liste))
			for _, z := range liste {
				datei = append(datei, z.Vorname+" "+z.Nachname)
			}
			for _, name := range f.erwarte {
				if !enthaelt(theke, name) {
					t.Errorf("Theke %q fand %q nicht, geliefert: %v", f.eingabe, name, theke)
				}
				if !enthaelt(datei, name) {
					t.Errorf("Leserdatei %q fand %q nicht, geliefert: %v", f.eingabe, name, datei)
				}
			}
		})
	}
}
