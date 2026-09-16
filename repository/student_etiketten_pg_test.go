package repository

import (
	"context"
	"testing"
)

// Gegen echtes Postgres, nicht gegen den Mock — die Abfrage steht und fällt mit zwei
// Dingen, die ein pgxmock nie prüft:
//
//  1. `id = ANY($1)` mit einem []string gegen eine uuid-Spalte. Ob pgx daraus ein
//     uuid[] macht oder Postgres mit „operator does not exist: uuid = text" abbricht,
//     entscheidet der Treiber im Zusammenspiel mit dem echten Spaltentyp.
//  2. Die Sortierung nach Klasse, Nachname, Vorname. Der Mock liefert zurück, was man
//     ihm vorlegt — die Reihenfolge wäre damit eine Behauptung über sich selbst.
//
// Beides trägt direkt auf einen gedruckten Klebebogen: Die falsche Reihenfolge merkt
// man erst beim Verteilen, und der Typfehler legt den Druck ganz still lahm (500).
func TestEtikettenZeilen_ZweiKlassenBleibenBeieinander(t *testing.T) {
	// Der eigentliche Grund für die Sortierung. Wer "7" in die Schülerdatei tippt und
	// alle Treffer markiert, bekommt zwei Klassen auf einen Bogen. Sortiert allein nach
	// Nachname laufen sie ineinander (gemessen an echten Daten: 7A, 7A, 7A, 7B, 7B, 7B,
	// 7B, 7B, 7A, …) — und wer klassenweise austeilt, klaubt jedes Etikett einzeln
	// heraus. Die Klasse gehört deshalb an die erste Stelle der Sortierung.
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	var ids []string
	for _, f := range []struct{ barcode, vorname, nachname, klasse string }{
		{"SORT-1", "Anna", "Aal", "7B"},
		{"SORT-2", "Bert", "Bock", "7A"},
		{"SORT-3", "Cara", "Cent", "7B"},
		{"SORT-4", "Dora", "Dill", "7A"},
	} {
		var id string
		if err := pool.QueryRow(ctx,
			`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			 VALUES ($1, $2, $3, $4, 2031) RETURNING id`, f.barcode, f.vorname, f.nachname, f.klasse).Scan(&id); err != nil {
			t.Fatalf("Schüler %s anlegen: %v", f.barcode, err)
		}
		ids = append(ids, id)
	}

	zeilen, err := NewStudentRepository(pool).EtikettenZeilen(ctx, ids)
	if err != nil {
		t.Fatalf("EtikettenZeilen: %v", err)
	}

	var folge []string
	for _, z := range zeilen {
		folge = append(folge, z.Klasse+"/"+z.Nachname)
	}
	// Erst die ganze 7A, dann die ganze 7B — innerhalb der Klasse nach Nachname.
	erwartet := []string{"07A/Bock", "07A/Dill", "07B/Aal", "07B/Cent"}
	if len(folge) != len(erwartet) {
		t.Fatalf("erwartet %v, bekommen %v", erwartet, folge)
	}
	for i := range erwartet {
		if folge[i] != erwartet[i] {
			t.Fatalf("Reihenfolge falsch: erwartet %v, bekommen %v", erwartet, folge)
		}
	}
}

func TestEtikettenZeilen_SortiertUndNurLebende(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	// Bewusst in einer Reihenfolge angelegt, die NICHT der erwarteten entspricht:
	// Käme die Sortierung aus der Eingabe statt aus dem SQL, fiele es nicht auf.
	zimmer := seedSuchSchueler(t, pool, "ETI-1", "Tim", "Zimmermann")
	anders := seedSuchSchueler(t, pool, "ETI-2", "Bea", "Anders")
	mueller := seedSuchSchueler(t, pool, "ETI-3", "Ayse", "Müller")
	geloescht := seedSuchSchueler(t, pool, "ETI-4", "Erik", "Fort")

	if _, err := pool.Exec(ctx, `UPDATE schueler SET deleted_at = now() WHERE id = $1`, geloescht); err != nil {
		t.Fatalf("Schüler als gelöscht markieren: %v", err)
	}

	repo := NewStudentRepository(pool)

	zeilen, err := repo.EtikettenZeilen(ctx, []string{zimmer, anders, mueller, geloescht})
	if err != nil {
		t.Fatalf("EtikettenZeilen: %v", err)
	}

	var namen []string
	for _, z := range zeilen {
		namen = append(namen, z.Nachname+", "+z.Vorname)
	}
	erwartet := []string{"Anders, Bea", "Müller, Ayse", "Zimmermann, Tim"}
	if len(namen) != len(erwartet) {
		t.Fatalf("erwartet %v, bekommen %v", erwartet, namen)
	}
	for i := range erwartet {
		if namen[i] != erwartet[i] {
			t.Errorf("Platz %d: erwartet %q, bekommen %q (ganze Liste: %v)", i+1, erwartet[i], namen[i], namen)
		}
	}

	// Der gelöschte Schüler darf gar nicht auftauchen — ein Etikett mit seinem Namen
	// wäre ein gedrucktes Stück personenbezogener Daten zu einem Konto, das es
	// nicht mehr gibt.
	for _, z := range zeilen {
		if z.BarcodeID == "ETI-4" {
			t.Error("gelöschter Schüler steht auf dem Etikettenbogen")
		}
	}
}

func TestEtikettenZeilen_UnbekannteIDsSindKeinFehler(t *testing.T) {
	// Zwischen dem Markieren und dem Druck kann ein Schüler gelöscht worden sein
	// (Mehrplatzbetrieb). Der Bogen soll deshalb mit den übrigen Namen entstehen und
	// nicht abbrechen; der Aufrufer vergleicht die Anzahl.
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	da := seedSuchSchueler(t, pool, "ETI-5", "Lea", "Berger")
	repo := NewStudentRepository(pool)

	zeilen, err := repo.EtikettenZeilen(ctx, []string{da, "00000000-0000-0000-0000-000000000000"})
	if err != nil {
		t.Fatalf("EtikettenZeilen mit unbekannter ID: %v", err)
	}
	if len(zeilen) != 1 || zeilen[0].BarcodeID != "ETI-5" {
		t.Errorf("erwartet genau Lea Berger, bekommen %+v", zeilen)
	}
}

func TestEtikettenZeilen_LeereListeFragtDieDatenbankNichtAn(t *testing.T) {
	// `= ANY('{}')` wäre gültiges SQL, aber eine Abfrage ohne Zweck. Der Handler
	// weist leere Aufträge ohnehin ab — hier steht, dass auch die Schicht darunter
	// nicht in einen Fehler läuft.
	pool := pgTestPool(t)
	repo := NewStudentRepository(pool)

	zeilen, err := repo.EtikettenZeilen(context.Background(), nil)
	if err != nil {
		t.Fatalf("leere Liste: %v", err)
	}
	if len(zeilen) != 0 {
		t.Errorf("erwartet keine Zeilen, bekommen %d", len(zeilen))
	}
}

// TestEtikettenZeilen_LaesstKeinenKollegenWeg: Seit dem 16.09.2026 stehen Schüler und
// Kollegium in derselben Liste, mit demselben Häkchen davor. Wer „alle markieren" drückt
// und einen Etikettenbogen anfordert, meint auch die Kollegen.
//
// Die Abfrage las die Sicht `schueler`. Ein markierter Kollege fiel damit LAUTLOS vom
// Bogen — der Druck gelang, der Bogen war nur kürzer, und gemerkt hätte man es beim
// Verteilen. Solange die Kollegen in keiner Liste standen, war die Sicht richtig; mit der
// Leserdatei wurde sie zum stillen Verlust.
func TestEtikettenZeilen_LaesstKeinenKollegenWeg(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	var ids []string
	for _, f := range []struct{ barcode, vorname, nachname, klasse, art string }{
		{"ETIK-1", "Anna", "Aal", "7B", "schueler"},
		{"ETIK-2", "Katrin", "Wendland", "", "lehrkraft"},
		{"ETIK-3", "Bert", "Bock", "7A", "schueler"},
	} {
		var id string
		var klasse, jahr any
		if f.art == "schueler" {
			klasse, jahr = f.klasse, 2031
		}
		if err := pool.QueryRow(ctx,
			`INSERT INTO leser (barcode_id, vorname, nachname, klasse, abgaenger_jahr, art)
			 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			f.barcode, f.vorname, f.nachname, klasse, jahr, f.art).Scan(&id); err != nil {
			t.Fatalf("Leser %s anlegen: %v", f.barcode, err)
		}
		ids = append(ids, id)
	}

	zeilen, err := NewStudentRepository(pool).EtikettenZeilen(ctx, ids)
	if err != nil {
		t.Fatalf("EtikettenZeilen: %v", err)
	}
	if len(zeilen) != 3 {
		namen := make([]string, 0, len(zeilen))
		for _, z := range zeilen {
			namen = append(namen, z.Nachname)
		}
		t.Fatalf("%d Etiketten statt 3 — ein markierter Leser fiel lautlos vom Bogen: %v", len(zeilen), namen)
	}
	// Dieselbe Reihenfolge wie die Liste, aus der markiert wurde: Kollegium zuerst,
	// dann die Schüler klassenweise. Zwei Ausgaben derselben Auswahl sollen nicht
	// verschieden sortiert aus dem Drucker kommen.
	if zeilen[0].Nachname != "Wendland" || zeilen[1].Nachname != "Bock" || zeilen[2].Nachname != "Aal" {
		t.Errorf("Reihenfolge: %s, %s, %s", zeilen[0].Nachname, zeilen[1].Nachname, zeilen[2].Nachname)
	}
	if zeilen[0].Klasse != "" {
		t.Errorf("ein Kollege hat keine Klasse auf dem Etikett: %q", zeilen[0].Klasse)
	}
}
