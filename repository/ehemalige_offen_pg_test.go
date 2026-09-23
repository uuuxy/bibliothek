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

// Der Wächter und die Löschuhr rechnen mit DERSELBEN Uhr (KarenzUhr): dem spätesten von
// Abgang, letzter Rückgabe und letztem Schadensabschluss. Bis zum 22.09.2026 nahm der
// Wächter allein den Abgang (OFFEN.md 5.12, „dritte Formulierung derselben Frage"): Wer vor
// zwei Jahren wegging, vor zehn Tagen an der Theke ein Buch zurückgab und ein zweites noch
// hat, stand als „Karenz abgelaufen, offener Vorgang" auf der Liste — während die Löschuhr
// für ihn erst seit zehn Tagen läuft. Zwei Antworten auf „ist die Karenz vorbei?".
func TestWaechterRechnetMitDerUhrDerLoeschfrist(t *testing.T) {
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
		INSERT INTO buecher_titel (titel, autor, medientyp) VALUES ('Karenzuhr-Testband', 'P', 'Buch')
		RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	t.Cleanup(func() {
		auf := context.Background()
		for _, sql := range []string{
			`DELETE FROM ausleihen WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = $1)`,
			`DELETE FROM buecher_exemplare WHERE titel_id = $1`,
			`DELETE FROM buecher_titel WHERE id = $1`,
		} {
			if _, err := pool.Exec(auf, sql, titelID); err != nil {
				t.Errorf("aufräumen (%s): %v", sql, err)
			}
		}
		if _, err := pool.Exec(auf, `DELETE FROM schueler WHERE barcode_id LIKE 'W-Uhr-%' || $1`, suffix); err != nil {
			t.Errorf("aufräumen Schüler: %v", err)
		}
	})
	abgaenger := func(t *testing.T, name string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger, abgaenger_seit)
			VALUES ($1, $2, 'Zurueck', '10A', 2024, true, now() - interval '2 years') RETURNING id`,
			"W-Uhr-"+name+"-"+suffix, name).Scan(&id); err != nil {
			t.Fatalf("Abgänger %s anlegen: %v", name, err)
		}
		return id
	}
	ausleihe := func(t *testing.T, schuelerID, barcode string, rueckgabe any) {
		t.Helper()
		var exemplarID string
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`,
			titelID, barcode+"-"+suffix).Scan(&exemplarID); err != nil {
			t.Fatalf("Exemplar: %v", err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
			VALUES ($1, $2, now() - interval '400 days', now() - interval '379 days', $3)`,
			exemplarID, schuelerID, rueckgabe); err != nil {
			t.Fatalf("Ausleihe: %v", err)
		}
	}
	// KURZ: vor zehn Tagen ein Buch zurückgegeben (die Uhr der Löschfrist läuft ab hier) …
	kurz := abgaenger(t, "Kurz")
	ausleihe(t, kurz, "B-U1", time.Now().AddDate(0, 0, -10))
	// … und ein zweites noch offen (der Vorgang, der die Löschung blockiert).
	ausleihe(t, kurz, "B-U2", nil)

	nachher, err := repo.ZaehleEhemaligeMitOffenenVorgaengen(ctx, 365)
	if err != nil {
		t.Fatalf("Wächter lesen: %v", err)
	}
	if neu := nachher - vorher; neu != 0 {
		t.Errorf("der Wächter meldet %d neue Fälle, erwartet 0 — die Karenz läuft seit der Rückgabe vor zehn Tagen, "+
			"die Löschuhr (PredikatAnonymisierung) rechnet genau so", neu)
	}
	// Gegenprobe: Ist die letzte Rückgabe älter als die Karenz, meldet der Wächter — wie die
	// Löschuhr, die ihn dann nur wegen des offenen Buchs stehen lässt.
	//
	// LANG bekommt seine alte Rückgabe beim Anlegen. Bis zum 23.09.2026 datierte diese
	// Gegenprobe stattdessen die Rückgabe von KURZ per UPDATE um 390 Tage zurück. Seit
	// Migration 137 steht die Uhr als Stempel am Leser und folgt einer Rückdatierung nicht
	// mehr — sie geht nur nach vorne. Nachgesehen am 23.09.2026: Kein Schreibpfad der
	// Anwendung stellt den Zustand her, den das UPDATE herstellte. Alle drei Rückgabewege
	// fassen nur offene Ausleihen an (repository/loan.go ReturnLoanZumTx,
	// repository/schaden_melden.go, internal/service/device_service.go über activeLoan), und
	// die Littera-Übernahme fügt nur ein (internal/littera/schreiber_ausleihen.go).
	// rueckgabe_am geht also von NULL auf einen Wert und nie von einem Wert auf einen
	// anderen; für jeden erreichbaren Zustand ist der Stempel dasselbe wie das frühere
	// max(rueckgabe_am). Dass er nach hinten nicht folgt, hält
	// repository/leser_stempel_pg_test.go fest.
	lang := abgaenger(t, "Lang")
	ausleihe(t, lang, "B-U3", time.Now().AddDate(0, 0, -400))
	ausleihe(t, lang, "B-U4", nil)

	spaeter, err := repo.ZaehleEhemaligeMitOffenenVorgaengen(ctx, 365)
	if err != nil {
		t.Fatalf("Wächter lesen: %v", err)
	}
	if neu := spaeter - vorher; neu != 1 {
		t.Errorf("mit alter Rückgabe meldet der Wächter %d neue Fälle, erwartet 1", neu)
	}
}
