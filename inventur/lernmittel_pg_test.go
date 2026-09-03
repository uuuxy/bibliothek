package inventur

import (
	"bytes"
	"context"
	"testing"

	"bibliothek/internal/pgtest"

	"github.com/xuri/excelize/v2"
)

// Schulbücher je Fach (Lehrerportal, 03.09.2026) am echten Postgres: Nur Lernmittel
// zählen, das Fach gruppiert, „ohne Fach" ist eine eigene Kachel, und die Zahlen folgen
// derselben Zählweise wie die Klassensätze (ausgesondert und nur bestellt zählen nicht,
// verliehen = offene Ausleihe).
func TestLernmittelJeFach_ZaehltNurSchulbuecher(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	for _, sql := range []string{
		`DELETE FROM ausleihen`, `DELETE FROM buecher_exemplare`, `DELETE FROM buecher_titel`,
		`INSERT INTO systematik_kategorien (kuerzel, bezeichnung) VALUES ('LMTEST-MA', 'Mathematik') ON CONFLICT DO NOTHING`,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr) VALUES ('LM-S1', 'Lern', 'Mittel', '07A', 2031) ON CONFLICT DO NOTHING`,
		`INSERT INTO buecher_titel (id, titel, isbn, subject, ist_lernmittel) VALUES
			('00000000-0000-0000-0000-00000000a001', 'Mathe 7', '978-1', 'Mathematik', true),
			('00000000-0000-0000-0000-00000000a002', 'Mathe 8', '978-2', 'Mathematik', true),
			('00000000-0000-0000-0000-00000000a003', 'Lesebuch', '978-3', NULL, true),
			('00000000-0000-0000-0000-00000000a004', 'Roman', '978-4', 'Mathematik', false)`,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, ist_ausgesondert, aussonderung_grund) VALUES
			('00000000-0000-0000-0000-00000000a001', 'LM-1', true, false, NULL),
			('00000000-0000-0000-0000-00000000a001', 'LM-2', true, false, NULL),
			('00000000-0000-0000-0000-00000000a001', 'LM-3', true, true, 'AUSSORTIERT'),
			('00000000-0000-0000-0000-00000000a002', 'LM-4', true, false, NULL),
			('00000000-0000-0000-0000-00000000a003', 'LM-5', true, false, NULL),
			('00000000-0000-0000-0000-00000000a004', 'LM-6', true, false, NULL)`,
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
			SELECT e.id, s.id, now() + interval '14 days' FROM buecher_exemplare e, schueler s WHERE e.barcode_id = 'LM-1' AND s.barcode_id = 'LM-S1'`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatalf("%s: %v", sql[:30], err)
		}
	}
	repo := NewBookRepository(pool)

	faecher, err := repo.GetLernmittelFaecher(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(faecher) != 2 || faecher[0].Fach != "Mathematik" || faecher[1].Fach != "" {
		t.Fatalf("erwartet Mathematik + ohne Fach, bekam %+v", faecher)
	}
	ma := faecher[0]
	// Mathe 7: 2 aktive (1 verliehen), 1 ausgesondert zählt nicht; Mathe 8: 1. Roman zählt nicht (kein Lernmittel).
	if ma.Titel != 2 || ma.Gesamt != 3 || ma.Verliehen != 1 || ma.Verfuegbar != 2 {
		t.Errorf("Mathematik: %+v (erwartet 2 Titel, 3 gesamt, 1 verliehen, 2 verfügbar)", ma)
	}
	if faecher[1].Titel != 1 || faecher[1].Gesamt != 1 {
		t.Errorf("ohne Fach: %+v", faecher[1])
	}

	titel, err := repo.GetLernmittelTitel(ctx, "Mathematik", false)
	if err != nil {
		t.Fatal(err)
	}
	// Mathe 7: LM-1 verliehen, LM-2 frei, LM-3 ausgesondert (zählt nirgends).
	if len(titel) != 2 || titel[0].Title != "Mathe 7" || titel[0].Gesamt != 2 || titel[0].Verliehen != 1 || titel[0].Verfuegbar != 1 {
		t.Errorf("Titel eines Fachs: %+v", titel)
	}
	alle, err := repo.GetLernmittelTitel(ctx, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(alle) != 3 || alle[2].Title != "Lesebuch" {
		t.Errorf("Export-Liste: erwartet 3 Lernmittel, ohne Fach zuletzt: %+v", alle)
	}

	// Excel: eine Zeile je Titel, Kopf + Zahlen lesbar, Formel-Schutz greift.
	alle[0].Title = "=1+1"
	f, err := SchulbuecherAlsExcel(alle, "https://schule.test")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	gelesen, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := gelesen.GetRows("Schulbücher")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 || rows[0][1] != "Titel" || rows[1][0] != "Mathematik" || rows[1][4] != "2" || rows[3][0] != "ohne Fach" {
		t.Errorf("Excel-Inhalt: %v", rows)
	}
	if rows[1][1] == "=1+1" {
		t.Error("Formel-Injection: Titel darf nicht als Formel in der Zelle stehen")
	}
}
