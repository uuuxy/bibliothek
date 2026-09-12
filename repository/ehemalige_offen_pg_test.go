package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
)

// „Wie lange ist der weg?" — eine Frage, eine Formulierung (#593, Paket 5).
//
// Der Wächter „Ehemalige mit offenen Vorgängen" meldet Weggegangene, deren Name und
// Anschrift auf Dauer stehen bleiben, weil ein offener Vorgang die Anonymisierung
// blockiert. Er verlangte `abgaenger_seit IS NOT NULL`; das Löschprädikat rechnet seit je
// mit `COALESCE(abgaenger_seit, aktualisiert_am)` — für Altzeilen ohne Stempel.
//
// Der Unterschied trifft genau den Fall, für den es den Wächter gibt: Eine Altzeile ohne
// `abgaenger_seit` war für ihn unsichtbar, egal wie lange der Schüler schon weg ist —
// gemeldet wurde sie nie, gelöscht auch nicht (der offene Vorgang schützt sie).
// Migration 094 hat die Spalte nachgetragen und alle drei Schreiber stempeln sie, der
// Fall ist also heute nicht erreichbar; erreichbar bleibt er über jeden Import, der
// morgen eine Zeile ohne Stempel anlegt.
func TestWaechterSiehtAuchAbgaengerOhneStempel(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBetriebszustandRepository(pool)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	vorher, err := repo.ZaehleEhemaligeMitOffenenVorgaengen(ctx, 365)
	if err != nil {
		t.Fatalf("Wächter lesen: %v", err)
	}

	var titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, medientyp) VALUES ('Waechter-Testband', 'P', 'Buch')
		RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	t.Cleanup(func() {
		auf := context.Background()
		if _, err := pool.Exec(auf, `DELETE FROM ausleihen WHERE exemplar_id IN
			(SELECT id FROM buecher_exemplare WHERE titel_id = $1)`, titelID); err != nil {
			t.Errorf("Aufräumen Ausleihen: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM buecher_exemplare WHERE titel_id = $1`, titelID); err != nil {
			t.Errorf("Aufräumen Exemplare: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM buecher_titel WHERE id = $1`, titelID); err != nil {
			t.Errorf("Aufräumen Titel: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM schueler WHERE barcode_id LIKE 'W-%' || $1`, suffix); err != nil {
			t.Errorf("Aufräumen Schüler: %v", err)
		}
	})

	// abgaenger legt einen Weggegangenen an. abgaengerSeit nil = Altzeile ohne Stempel;
	// aktualisiert_am wird beim INSERT gesetzt (der Trigger greift nur bei UPDATE).
	abgaenger := func(t *testing.T, name string, abgaengerSeit any, aktualisiertVorTagen int) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr,
			                      ist_abgaenger, abgaenger_seit, aktualisiert_am)
			VALUES ($1, $2, 'Weg', '10A', 2024, true, $3, now() - make_interval(days => $4))
			RETURNING id`,
			"W-"+name+"-"+suffix, name, abgaengerSeit, aktualisiertVorTagen).Scan(&id); err != nil {
			t.Fatalf("Abgänger %s anlegen: %v", name, err)
		}
		return id
	}
	offeneAusleihe := func(t *testing.T, schuelerID, barcode string) {
		t.Helper()
		var exemplarID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			VALUES ($1, $2, true) RETURNING id`, titelID, barcode+"-"+suffix).Scan(&exemplarID); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
			VALUES ($1, $2, now() - interval '400 days', now() - interval '379 days')
		`, exemplarID, schuelerID); err != nil {
			t.Fatalf("Ausleihe anlegen: %v", err)
		}
	}

	vorZweiJahren := time.Now().AddDate(-2, 0, 0)

	// 1. Mit Stempel, lange weg, offenes Buch — der Regelfall, den der Wächter meldet.
	offeneAusleihe(t, abgaenger(t, "MitStempel", vorZweiJahren, 700), "B-W1")
	// 2. OHNE Stempel (Altzeile), lange nicht angefasst, offenes Buch — der Fund.
	offeneAusleihe(t, abgaenger(t, "OhneStempel", nil, 700), "B-W2")
	// 3. Mit Stempel, aber erst seit zehn Tagen weg — noch kein Fall für den Wächter.
	offeneAusleihe(t, abgaenger(t, "Frisch", time.Now().AddDate(0, 0, -10), 10), "B-W3")
	// 4. Lange weg, aber nichts offen — nichts blockiert, also keine Meldung.
	abgaenger(t, "Sauber", vorZweiJahren, 700)

	nachher, err := repo.ZaehleEhemaligeMitOffenenVorgaengen(ctx, 365)
	if err != nil {
		t.Fatalf("Wächter lesen: %v", err)
	}
	if neu := nachher - vorher; neu != 2 {
		t.Errorf("der Wächter meldet %d neue Fälle, erwartet 2 (mit und ohne Stempel) — "+
			"eine Altzeile ohne abgaenger_seit bleibt unsichtbar, obwohl ihr Name auf Dauer stehen bleibt", neu)
	}
}
