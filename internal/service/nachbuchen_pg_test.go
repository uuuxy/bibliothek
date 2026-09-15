package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Nachbuch-Tür am echten Postgres — die Server-Hälfte der Nachweisliste (OFFEN.md 2.3):
// Umbuchung, Doppelscan, gesperrter Ausweis (Rücknahme bleibt), Rückgabe vor älterer
// Bewegung (veraltet), gleicher Schlüssel nach abgebrochenem Online-Versand (nachgeholt),
// verloren gemeldetes Buch (Forderung endet, Hinweis in der Meldung), zwei parallele
// Aufrufe auf ein Exemplar, unbekanntes Buch, unbekannter Ausweis. Rot am alten Code: Die
// Tür existierte nicht.

type nbWelt struct {
	pool                      *pgxpool.Pool
	svc                       NachbuchService
	staff, anna, ben, carla   string
	titelID, exemplarID, code string
}

func nbAufbau(t *testing.T) *nbWelt {
	t.Helper()
	pool := pgtest.Pool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	w := &nbWelt{pool: pool}
	eins := func(was, sql string, args ...any) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %v", was, err)
		}
		return id
	}
	w.staff = eins("Mitarbeiter", `INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ($1, 'Nach', 'Bucher', $2, 'mitarbeiter', true) RETURNING id`, "MA-"+suffix, "nb-"+suffix+"@schule.invalid")
	w.anna = eins("Anna", `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr) VALUES ($1, 'Anna', 'Erste', '07B', 2031) RETURNING id`, "S-A-"+suffix)
	w.ben = eins("Ben", `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr) VALUES ($1, 'Ben', 'Zweiter', '07B', 2031) RETURNING id`, "S-B-"+suffix)
	w.carla = eins("Carla", `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt, block_reason)
		VALUES ($1, 'Carla', 'Gesperrt', '07B', 2031, true, 'Testsperre') RETURNING id`, "S-C-"+suffix)
	w.titelID = eins("Titel", `INSERT INTO buecher_titel (titel, autor, medientyp, ist_lernmittel) VALUES ('Nachbuch-Testband', 'Prüfer', 'Buch', false) RETURNING id`)
	w.code = "B-NB-" + suffix
	w.exemplarID = eins("Exemplar", `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, einkaufspreis) VALUES ($1, $2, true, 12.00) RETURNING id`, w.titelID, w.code)

	studentRepo := repository.NewStudentRepository(pool)
	bookRepo := repository.NewBookRepository(pool)
	loanRepo := repository.NewLoanRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)
	w.svc = NewNachbuchService(pool, studentRepo, bookRepo, repository.NewUserRepository(pool), loanRepo, auditRepo)
	return w
}

func (w *nbWelt) eintrag(absicht string, schueler *string, gescannt time.Time) NachbuchEintrag {
	return NachbuchEintrag{Schluessel: uuid.NewString(), Absicht: absicht, Barcode: w.code, GescanntAm: gescannt, SchuelerID: schueler, StaffID: w.staff}
}

func (w *nbWelt) offeneAusleihen(t *testing.T) (n int, schueler string) {
	t.Helper()
	err := w.pool.QueryRow(context.Background(), `
		SELECT count(*), coalesce(max(schueler_id::text), '') FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`, w.exemplarID).Scan(&n, &schueler)
	if err != nil {
		t.Fatalf("offene Ausleihen: %v", err)
	}
	return
}

func (w *nbWelt) meldungen(t *testing.T, ergebnis string) int {
	t.Helper()
	var n int
	if err := w.pool.QueryRow(context.Background(), `SELECT count(*) FROM nachbuch_meldungen WHERE barcode = $1 AND ergebnis = $2`, w.code, ergebnis).Scan(&n); err != nil {
		t.Fatalf("Meldungen zählen: %v", err)
	}
	return n
}

func TestNachbuchen_UmbuchungDoppelscanUndSperre(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	t0 := time.Now().Add(-30 * time.Minute)

	// Anna leiht offline (t0): ausgeliehen, erfasst_am = Scan.
	erg, err := w.svc.Nachbuchen(ctx, w.eintrag(NachbuchAbsichtAusleihe, &w.anna, t0))
	if err != nil || erg.Ergebnis != repository.NachbuchAusgeliehen {
		t.Fatalf("Anna: %v %+v", err, erg)
	}
	var ausgeliehen, erfasst time.Time
	if err := w.pool.QueryRow(ctx, `SELECT ausgeliehen_am, erfasst_am FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`, w.exemplarID).Scan(&ausgeliehen, &erfasst); err != nil {
		t.Fatalf("Ausleihe lesen: %v", err)
	}
	if d := ausgeliehen.Sub(t0); d < -time.Second || d > time.Second {
		t.Errorf("ausgeliehen_am %v, erwartet Scan-Zeit %v", ausgeliehen, t0)
	}
	if !erfasst.Equal(ausgeliehen) {
		t.Errorf("erfasst_am %v ≠ ausgeliehen_am %v", erfasst, ausgeliehen)
	}

	// Doppelscan: Anna noch einmal — nichts umgekehrt.
	erg, err = w.svc.Nachbuchen(ctx, w.eintrag(NachbuchAbsichtAusleihe, &w.anna, t0.Add(time.Minute)))
	if err != nil || erg.Ergebnis != repository.NachbuchBereitsAusgeliehen {
		t.Fatalf("Doppelscan: %v %+v", err, erg)
	}
	if n, s := w.offeneAusleihen(t); n != 1 || s != w.anna {
		t.Fatalf("nach Doppelscan: %d offene, bei %s", n, s)
	}

	// Umbuchung: Ben scannt dasselbe Buch später — bei Anna zurück, an Ben ausgeliehen, Meldung.
	erg, err = w.svc.Nachbuchen(ctx, w.eintrag(NachbuchAbsichtAusleihe, &w.ben, t0.Add(5*time.Minute)))
	if err != nil || erg.Ergebnis != repository.NachbuchUmgebucht {
		t.Fatalf("Umbuchung: %v %+v", err, erg)
	}
	if n, s := w.offeneAusleihen(t); n != 1 || s != w.ben {
		t.Fatalf("nach Umbuchung: %d offene, bei %s (Ben=%s)", n, s, w.ben)
	}
	if erg.Result == nil || erg.Result.Vorbesitzer == nil || erg.Result.Vorbesitzer.ID != w.anna {
		t.Errorf("Umbuchung nennt den Vorbesitzer nicht: %+v", erg.Result)
	}
	if w.meldungen(t, repository.NachbuchUmgebucht) != 1 {
		t.Errorf("Umbuchung ohne Meldung")
	}

	// Gesperrter Ausweis: Carla scannt — bei Ben zurückgenommen, Carla abgewiesen, Meldung.
	erg, err = w.svc.Nachbuchen(ctx, w.eintrag(NachbuchAbsichtAusleihe, &w.carla, t0.Add(10*time.Minute)))
	if err != nil || erg.Ergebnis != repository.NachbuchNichtGebucht {
		t.Fatalf("Sperre: %v %+v", err, erg)
	}
	if n, _ := w.offeneAusleihen(t); n != 0 {
		t.Fatalf("nach abgewiesener Ausleihe: %d offene — die Rücknahme bei Ben muss bleiben, die Ausleihe an Carla nicht entstehen", n)
	}
	if w.meldungen(t, repository.NachbuchNichtGebucht) != 1 {
		t.Errorf("abgewiesene Ausleihe ohne Meldung")
	}
}

func TestNachbuchen_WaechterUndBekannterSchluessel(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	jetzt := time.Now()
	// Online: Anna lieh vor 20 Minuten, gab vor 5 Minuten zurück — die letzte Bewegung ist -5 min.
	if _, err := w.pool.Exec(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, ausgeliehen_am, erfasst_am, rueckgabe_frist, rueckgabe_am)
		VALUES ($1, $2, $3, $4::timestamptz, $4::timestamptz, $4::timestamptz + interval '14 days', $5::timestamptz)`,
		w.exemplarID, w.anna, w.staff, jetzt.Add(-20*time.Minute), jetzt.Add(-5*time.Minute)); err != nil {
		t.Fatalf("Online-Ausleihe: %v", err)
	}
	if _, err := w.pool.Exec(ctx, `UPDATE buecher_exemplare SET letzte_bewegung_am = $2 WHERE id = $1`, w.exemplarID, jetzt.Add(-5*time.Minute)); err != nil {
		t.Fatalf("Stempel: %v", err)
	}

	// Ein Scan von -10 min ist älter als die letzte Bewegung: veraltet, Meldung, nichts gebucht.
	erg, err := w.svc.Nachbuchen(ctx, w.eintrag(NachbuchAbsichtRueckgabe, nil, jetzt.Add(-10*time.Minute)))
	if err != nil || erg.Ergebnis != repository.NachbuchVeraltet {
		t.Fatalf("veraltet: %v %+v", err, erg)
	}
	if w.meldungen(t, repository.NachbuchVeraltet) != 1 {
		t.Errorf("veralteter Scan ohne Meldung")
	}

	// Derselbe Scan mit bekanntem Schlüssel (abgebrochener Online-Versand): Die fehlende
	// Hälfte wird nachgeholt — Anna bekommt das Buch, obwohl der Scan älter ist.
	e := w.eintrag(NachbuchAbsichtAusleihe, &w.anna, jetzt.Add(-10*time.Minute))
	e.SchluesselBekannt = true
	erg, err = w.svc.Nachbuchen(ctx, e)
	if err != nil || erg.Ergebnis != repository.NachbuchAusgeliehen {
		t.Fatalf("bekannter Schlüssel: %v %+v", err, erg)
	}
	if n, s := w.offeneAusleihen(t); n != 1 || s != w.anna {
		t.Fatalf("nachgeholt: %d offene, bei %s", n, s)
	}
}

func TestNachbuchen_VerlorenGemeldetesBuchKommtZurueck(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	var schadensfallID string
	if _, err := w.pool.Exec(ctx, `UPDATE buecher_exemplare SET ist_ausleihbar = false, ist_ausgesondert = true, aussonderung_grund = 'VERLUST' WHERE id = $1`, w.exemplarID); err != nil {
		t.Fatalf("aussondern: %v", err)
	}
	if err := w.pool.QueryRow(ctx, `INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, art)
		VALUES ($1, $2, 'Nicht zurückgegeben', 12.00, 'nicht_zurueckgegeben') RETURNING id`, w.exemplarID, w.anna).Scan(&schadensfallID); err != nil {
		t.Fatalf("Forderung: %v", err)
	}
	erg, err := w.svc.Nachbuchen(ctx, w.eintrag(NachbuchAbsichtRueckgabe, nil, time.Now().Add(-time.Minute)))
	if err != nil || erg.Ergebnis != repository.NachbuchNurReaktiviert {
		t.Fatalf("Rückkehr: %v %+v", err, erg)
	}
	var storniert, ausleihbar bool
	if err := w.pool.QueryRow(ctx, `SELECT f.storniert_am IS NOT NULL, e.ist_ausleihbar FROM schadensfaelle f JOIN buecher_exemplare e ON e.id = f.exemplar_id WHERE f.id = $1`, schadensfallID).Scan(&storniert, &ausleihbar); err != nil {
		t.Fatalf("Lage: %v", err)
	}
	if !storniert || !ausleihbar {
		t.Errorf("Forderung storniert=%v, ausleihbar=%v — erwartet beides true", storniert, ausleihbar)
	}
	if w.meldungen(t, repository.NachbuchNurReaktiviert) != 1 {
		t.Errorf("Rückkehr ohne Meldung")
	}
}

func TestNachbuchen_ZweiParalleleAufrufeEinExemplar(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	gescannt := time.Now().Add(-time.Minute)
	var wg sync.WaitGroup
	ergebnisse := make([]string, 2)
	fehler := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			erg, err := w.svc.Nachbuchen(ctx, w.eintrag(NachbuchAbsichtAusleihe, &w.anna, gescannt))
			fehler[i] = err
			if erg != nil {
				ergebnisse[i] = erg.Ergebnis
			}
		}(i)
	}
	wg.Wait()
	for i := range fehler {
		if fehler[i] != nil {
			t.Fatalf("Aufruf %d: %v", i, fehler[i])
		}
	}
	if n, _ := w.offeneAusleihen(t); n != 1 {
		t.Fatalf("zwei parallele Aufrufe: %d offene Ausleihen, erwartet genau 1 (%v)", n, ergebnisse)
	}
	einmalJe := (ergebnisse[0] == repository.NachbuchAusgeliehen && ergebnisse[1] == repository.NachbuchBereitsAusgeliehen) ||
		(ergebnisse[1] == repository.NachbuchAusgeliehen && ergebnisse[0] == repository.NachbuchBereitsAusgeliehen)
	if !einmalJe {
		t.Errorf("Ergebnisse %v — erwartet einmal ausgeliehen, einmal bereits_ausgeliehen", ergebnisse)
	}
}

func TestNachbuchen_UnbekanntesBuchUndUnbekannterAusweis(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	e := w.eintrag(NachbuchAbsichtAusleihe, &w.anna, time.Now())
	e.Barcode = "B-GIBT-ES-NICHT"
	erg, err := w.svc.Nachbuchen(ctx, e)
	if err != nil || erg.Ergebnis != repository.NachbuchNichtGebucht {
		t.Fatalf("unbekanntes Buch: %v %+v", err, erg)
	}
	ausweis := "S-UNBEKANNT-" + uuid.NewString()[:8]
	e = w.eintrag(NachbuchAbsichtAusleihe, nil, time.Now())
	e.AusweisBarcode = &ausweis
	erg, err = w.svc.Nachbuchen(ctx, e)
	if err != nil || erg.Ergebnis != repository.NachbuchNichtGebucht {
		t.Fatalf("unbekannter Ausweis: %v %+v", err, erg)
	}
	var text string
	if err := w.pool.QueryRow(ctx, `SELECT coalesce(ausweis_text, '') FROM nachbuch_meldungen WHERE barcode = $1 AND ergebnis = 'nicht_gebucht'`, w.code).Scan(&text); err != nil {
		t.Fatalf("Meldung lesen: %v", err)
	}
	if text != ausweis {
		t.Errorf("Meldung trägt den Ausweis-Text %q, erwartet %q", text, ausweis)
	}
}
