package repository

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Die Klassensätze sprechen eine Klasse über ihren Normalschlüssel an (klassen_normkey) wie der
// Rest des Programms: „05a“ aus einer Anfrage und „05A“, wie der Auslöser class_name_kanonisieren
// die Klasse in class_books schreibt, meinen dieselbe Klasse. Geprüft an der echten Tabelle mit
// ihrem Auslöser, über die Funktionen, die die Türen rufen.
func TestKlassensaetze_SprechenDieKlasseUeberDenNormalschluesselAn(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewTitelAbfragen(pool)

	titel := func(name string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ($1) RETURNING id`, name).Scan(&id); err != nil {
			t.Fatalf("Titel %s anlegen: %v", name, err)
		}
		return id
	}
	mathe, deutsch := titel("Mathe 5"), titel("Deutsch 5")
	zuordnung := func() map[string][]string {
		t.Helper()
		zeilen, err := pool.Query(ctx, `SELECT cb.class_name, t.titel FROM class_books cb
			JOIN buecher_titel t ON t.id = cb.book_id ORDER BY cb.class_name, t.titel`)
		if err != nil {
			t.Fatalf("Zuordnung lesen: %v", err)
		}
		defer zeilen.Close()
		ist := map[string][]string{}
		for zeilen.Next() {
			var klasse, name string
			if err := zeilen.Scan(&klasse, &name); err != nil {
				t.Fatalf("Zuordnung lesen: %v", err)
			}
			ist[klasse] = append(ist[klasse], name)
		}
		if err := zeilen.Err(); err != nil {
			t.Fatalf("Zuordnung lesen: %v", err)
		}
		return ist
	}

	if err := repo.OrdneTitelKlassenZu(ctx, []string{"5a"}, []string{mathe}); err != nil {
		t.Fatalf("Zuordnen: %v", err)
	}
	if diff := cmp.Diff(map[string][]string{"05A": {"Mathe 5"}}, zuordnung()); diff != "" {
		t.Fatalf("Zuordnung nach dem Anlegen (-soll +ist):\n%s", diff)
	}

	// Neu schreiben mit „05a“, wie die Tür den Namen aus „5a“ macht: Der vorhandene Satz wird
	// ersetzt, nicht ergänzt, und der schon zugeordnete Titel stößt nicht an den Primärschlüssel.
	if err := repo.AendereKlassensaetze(ctx, "05a", []string{"05a"}, []string{mathe, deutsch}); err != nil {
		t.Fatalf("Neu schreiben: %v", err)
	}
	if diff := cmp.Diff(map[string][]string{"05A": {"Deutsch 5", "Mathe 5"}}, zuordnung()); diff != "" {
		t.Errorf("Zuordnung nach dem Neuschreiben (-soll +ist):\n%s", diff)
	}

	if err := repo.LoescheKlassensatzGruppe(ctx, []string{"05a"}); err != nil {
		t.Fatalf("Löschen: %v", err)
	}
	if ist := zuordnung(); len(ist) != 0 {
		t.Errorf("nach dem Löschen stehen noch Zuordnungen: %v", ist)
	}
}
