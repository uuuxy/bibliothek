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

	faecher, err := repo.GetLernmittelFaecher(ctx, LernmittelFilter{})
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

	titel, err := repo.GetLernmittelTitel(ctx, "Mathematik", false, LernmittelFilter{})
	if err != nil {
		t.Fatal(err)
	}
	// Mathe 7: LM-1 verliehen, LM-2 frei, LM-3 ausgesondert (zählt nirgends).
	if len(titel) != 2 || titel[0].Title != "Mathe 7" || titel[0].Gesamt != 2 || titel[0].Verliehen != 1 || titel[0].Verfuegbar != 1 {
		t.Errorf("Titel eines Fachs: %+v", titel)
	}
	alle, err := repo.GetLernmittelTitel(ctx, "", true, LernmittelFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(alle) != 3 || alle[2].Title != "Lesebuch" {
		t.Errorf("Export-Liste: erwartet 3 Lernmittel, ohne Fach zuletzt: %+v", alle)
	}

	// Jahrgang-Filter (03.09.2026): Mathe 8 nur Jahrgang 8, die anderen tragen die
	// Vorgabe 5–10 und zählen bei jedem Jahrgang darin mit; Jahrgang 12 trifft nichts.
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET jahrgang_von = 8, jahrgang_bis = 8 WHERE id = '00000000-0000-0000-0000-00000000a002'`); err != nil {
		t.Fatal(err)
	}
	jg7, err := repo.GetLernmittelTitel(ctx, "", true, LernmittelFilter{Jahrgang: 7})
	if err != nil {
		t.Fatal(err)
	}
	if len(jg7) != 2 || jg7[0].Title != "Mathe 7" || jg7[1].Title != "Lesebuch" {
		t.Errorf("Jahrgang 7: erwartet Mathe 7 + Lesebuch, bekam %+v", jg7)
	}
	f12, err := repo.GetLernmittelFaecher(ctx, LernmittelFilter{Jahrgang: 12})
	if err != nil {
		t.Fatal(err)
	}
	if len(f12) != 0 {
		t.Errorf("Jahrgang 12: erwartet keine Fächer, bekam %+v", f12)
	}
	jg8, err := repo.GetLernmittelTitel(ctx, "Mathematik", false, LernmittelFilter{Jahrgang: 8})
	if err != nil {
		t.Fatal(err)
	}
	if len(jg8) != 2 || jg8[0].JahrgangVon != 5 || jg8[1].JahrgangVon != 8 {
		t.Errorf("Jahrgang 8 in Mathematik: erwartet Mathe 7 (5–10) und Mathe 8 (8), bekam %+v", jg8)
	}

	// Schulzweig-Filter (03.09.2026): Mathe 8 auf Gymnasium; "-" trifft die ohne Zweig.
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET track = 'Gymnasium' WHERE id = '00000000-0000-0000-0000-00000000a002'`); err != nil {
		t.Fatal(err)
	}
	gym, err := repo.GetLernmittelTitel(ctx, "", true, LernmittelFilter{Zweig: "Gymnasium"})
	if err != nil {
		t.Fatal(err)
	}
	if len(gym) != 1 || gym[0].Title != "Mathe 8" || gym[0].Track != "Gymnasium" {
		t.Errorf("Zweig Gymnasium: erwartet nur Mathe 8, bekam %+v", gym)
	}
	ohne, err := repo.GetLernmittelTitel(ctx, "", true, LernmittelFilter{Zweig: ZweigOhne})
	if err != nil {
		t.Fatal(err)
	}
	if len(ohne) != 2 {
		t.Errorf("Zweig ohne Angabe: erwartet 2 Titel, bekam %+v", ohne)
	}

	// Suche greift über Titel, ISBN, Autor UND Fach — „Mathematik" findet beide
	// Mathe-Bücher über das Fach, obwohl keines das Wort im Titel trägt.
	treffer, err := repo.GetLernmittelTitel(ctx, "", true, LernmittelFilter{Suche: "Mathematik"})
	if err != nil {
		t.Fatal(err)
	}
	if len(treffer) != 2 {
		t.Errorf("Suche Mathematik: erwartet 2 Titel über das Fach, bekam %+v", treffer)
	}
	fSuche, err := repo.GetLernmittelFaecher(ctx, LernmittelFilter{Suche: "Lesebuch"})
	if err != nil {
		t.Fatal(err)
	}
	if len(fSuche) != 1 || fSuche[0].Fach != "" || fSuche[0].Titel != 1 {
		t.Errorf("Suche wirkt auch auf die Fach-Zahlen: %+v", fSuche)
	}
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET track = NULL WHERE id = '00000000-0000-0000-0000-00000000a002'`); err != nil {
		t.Fatal(err)
	}

	// Excel: eine Zeile je Titel, Kopf + Zahlen lesbar, Formel-Schutz greift — frisch
	// gelesen, damit der Jahrgang 8 von oben in der Spalte steht.
	if alle, err = repo.GetLernmittelTitel(ctx, "", true, LernmittelFilter{}); err != nil {
		t.Fatal(err)
	}
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
	// Spalten: Fach, Titel, Autor, ISBN, Jahrgang, Gesamt, … — Jahrgang 5–10 (Vorgabe) bleibt leer, 8 steht drin.
	if len(rows) != 4 || rows[0][1] != "Titel" || rows[1][0] != "Mathematik" || rows[1][4] != "" || rows[2][4] != "8" || rows[1][6] != "2" || rows[3][0] != "ohne Fach" {
		t.Errorf("Excel-Inhalt: %v", rows)
	}
	if rows[1][1] == "=1+1" {
		t.Error("Formel-Injection: Titel darf nicht als Formel in der Zelle stehen")
	}
}
