package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/pkg/ersatzwert"
	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5"
)

// Gate für die Größen des Betragsvorschlags im Melde-Dialog (OFFEN.md 9.3 a).
//
// Warum am echten Postgres und nicht mit einem Mock: Geprüft wird genau das, was ein Mock
// nicht kann — dass der JOIN auf buecher_titel trägt, dass ein nullbares Feld den Scan
// nicht sprengt, und vor allem, dass die Schuljahr-Zählung DIESELBE ist wie die des
// Bescheids. Zwei Auslegungen der Grenze zum 1. August verschöben den Betrag um eine
// ganze Stufe der Staffel, und die Zahl steht am Ende in einer Forderung.
func TestGroessenFuerExemplarZaehltSchuljahre(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	// Der konkrete Typ, weil ein Untertest gegen schuljahreMitAusleihe vergleicht — die
	// private Methode, die der Bescheid-Weg benutzt. Zweiwertig geprüft, damit ein
	// Umbau der Konstruktorsignatur hier einen Satz erzeugt und keine Panik.
	repo, ok := NewBescheidRepository(pool).(*pgBescheidRepository)
	if !ok {
		t.Fatalf("NewBescheidRepository liefert %T, erwartet *pgBescheidRepository",
			NewBescheidRepository(pool))
	}

	schuljahrBeginn := SchuljahrBeginn(schulzeit.Jetzt())

	var schuelerID, titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ($1, 'Ersatz', 'Wert', '07B', 2032) RETURNING id
	`, "SEW-"+suffix).Scan(&schuelerID); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, medientyp, ist_lernmittel)
		VALUES ('Ersatzwert-Testband', 'Prüfer', 'Buch', true) RETURNING id
	`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	t.Cleanup(func() {
		auf := context.Background()
		if _, err := pool.Exec(auf, `DELETE FROM ausleihen WHERE schueler_id = $1`, schuelerID); err != nil {
			t.Errorf("Aufräumen Ausleihen: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM buecher_exemplare WHERE titel_id = $1`, titelID); err != nil {
			t.Errorf("Aufräumen Exemplare: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM buecher_titel WHERE id = $1`, titelID); err != nil {
			t.Errorf("Aufräumen Titel: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM schueler WHERE id = $1`, schuelerID); err != nil {
			t.Errorf("Aufräumen Schüler: %v", err)
		}
	})

	// erworbenVorJahren legt ein Exemplar an, das vor n Schuljahren in den Bestand kam.
	// Die laufende Nummer hält die Barcodes auseinander: Zwei Fälle beschaffen im
	// SELBEN Schuljahr, und barcode_id ist über den ganzen Bestand eindeutig.
	lfd := 0
	erworbenVorJahren := func(t *testing.T, n int, preis float64) string {
		t.Helper()
		lfd++
		var id string
		err := pool.QueryRow(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, einkaufspreis, erworben_am)
			VALUES ($1, $2, true, $3, $4) RETURNING id
		`, titelID, fmt.Sprintf("B-EW%s-%d", suffix, lfd), preis,
			schuljahrBeginn.AddDate(-n, 0, 1)).Scan(&id)
		if err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
		return id
	}

	// ausgeliehenAm trägt eine abgeschlossene Ausleihe nach.
	ausgeliehenAm := func(t *testing.T, exemplarID string, am time.Time) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
			VALUES ($1, $2, $3, $3, $3)
		`, exemplarID, schuelerID, am); err != nil {
			t.Fatalf("Ausleihe anlegen: %v", err)
		}
	}

	t.Run("frisch beschafft, nie ausgeliehen — 1. Verleihjahr", func(t *testing.T) {
		id := erworbenVorJahren(t, 0, 20.00)

		g, err := repo.GroessenFuerExemplar(ctx, id)
		if err != nil {
			t.Fatalf("GroessenFuerExemplar: %v", err)
		}
		if g.Kaufpreis != 20.00 {
			t.Errorf("Kaufpreis = %.2f, want 20.00", g.Kaufpreis)
		}
		if !g.IstLernmittel {
			t.Error("IstLernmittel = false — der Titel ist als Lernmittel angelegt")
		}
		if jahr := ersatzwert.Verleihjahr(g.SchuljahreMitAusleihe, g.SchuljahreImBestand); jahr != 1 {
			t.Errorf("Verleihjahr = %d, want 1 (im Bestand: %d, mit Ausleihe: %d)",
				jahr, g.SchuljahreImBestand, g.SchuljahreMitAusleihe)
		}
	})

	t.Run("vier Schuljahre im Bestand — 5. Verleihjahr, 20 %", func(t *testing.T) {
		id := erworbenVorJahren(t, 4, 30.00)

		g, err := repo.GroessenFuerExemplar(ctx, id)
		if err != nil {
			t.Fatalf("GroessenFuerExemplar: %v", err)
		}
		if g.SchuljahreImBestand != 4 {
			t.Fatalf("SchuljahreImBestand = %d, want 4", g.SchuljahreImBestand)
		}
		v := ersatzwert.Rechne(ersatzwert.Verleihjahr(g.SchuljahreMitAusleihe, g.SchuljahreImBestand),
			g.Kaufpreis, g.Listenpreis, g.ZustandAbschlag)
		if v.Prozent != 20 {
			t.Errorf("Prozent = %d, want 20 — fünftes Verleihjahr", v.Prozent)
		}
	})

	t.Run("mehrfach im SELBEN Schuljahr ausgeliehen zählt EIN Jahr", func(t *testing.T) {
		// Der Fehler, den es bis zum 12.09.2026 gab: Gezählt wurden die AUSLEIHEN. Ein
		// Buch, das in einem Schuljahr sechsmal hinausging, stand damit im 6. Verleihjahr
		// — 10 % statt 100 %. Hier sind es drei Ausleihen in einem Schuljahr.
		id := erworbenVorJahren(t, 0, 40.00)
		for _, tage := range []int{10, 40, 70} {
			ausgeliehenAm(t, id, schuljahrBeginn.AddDate(0, 0, tage))
		}

		g, err := repo.GroessenFuerExemplar(ctx, id)
		if err != nil {
			t.Fatalf("GroessenFuerExemplar: %v", err)
		}
		if g.SchuljahreMitAusleihe != 1 {
			t.Errorf("SchuljahreMitAusleihe = %d, want 1 — drei Ausleihen, aber EIN Schuljahr",
				g.SchuljahreMitAusleihe)
		}
		v := ersatzwert.Rechne(ersatzwert.Verleihjahr(g.SchuljahreMitAusleihe, g.SchuljahreImBestand),
			g.Kaufpreis, g.Listenpreis, g.ZustandAbschlag)
		if v.Betrag != 40.00 {
			t.Errorf("Betrag = %.2f, want 40.00 (voller Kaufpreis im ersten Verleihjahr)", v.Betrag)
		}
	})

	t.Run("dieselbe Zählung wie der Bescheid-Weg", func(t *testing.T) {
		// Die eigentliche Zusicherung dieses Gates: Melde-Dialog und Bescheid dürfen für
		// dasselbe Exemplar nie verschiedene Verleihjahre nennen. Verglichen wird gegen
		// schuljahreMitAusleihe — den Zähler, den OffeneForderungen benutzt.
		id := erworbenVorJahren(t, 2, 25.00)
		ausgeliehenAm(t, id, schuljahrBeginn.AddDate(-2, 0, 5))
		ausgeliehenAm(t, id, schuljahrBeginn.AddDate(-1, 0, 5))

		g, err := repo.GroessenFuerExemplar(ctx, id)
		if err != nil {
			t.Fatalf("GroessenFuerExemplar: %v", err)
		}
		ueberDenBescheidWeg, err := repo.schuljahreMitAusleihe(ctx, []string{id})
		if err != nil {
			t.Fatalf("schuljahreMitAusleihe: %v", err)
		}
		if g.SchuljahreMitAusleihe != ueberDenBescheidWeg[id] {
			t.Errorf("Melde-Dialog zählt %d Schuljahre, der Bescheid-Weg %d — für dasselbe "+
				"Exemplar müssen beide dieselbe Zahl nennen",
				g.SchuljahreMitAusleihe, ueberDenBescheidWeg[id])
		}
	})

	t.Run("unbekanntes Exemplar meldet ErrNoRows, nicht 500", func(t *testing.T) {
		// Der Dialog fragt beim Öffnen; zwischen Anzeige und Klick kann ein Exemplar
		// verschwunden sein. Der Handler macht daraus ein 404 — dafür muss hier ein
		// erkennbares ErrNoRows ankommen und kein irgendwie gearteter Scan-Fehler.
		_, err := repo.GroessenFuerExemplar(ctx, "00000000-0000-0000-0000-000000000000")
		if err == nil {
			t.Fatal("kein Fehler für ein Exemplar, das es nicht gibt")
		}
		if err != pgx.ErrNoRows {
			t.Errorf("Fehler = %v, want pgx.ErrNoRows — sonst wird daraus ein 500 statt 404", err)
		}
	})
}
