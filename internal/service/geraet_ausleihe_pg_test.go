package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// geraeteTheke ist der Aufbau für die Tests der Geräte-Ausleihe an der Datenbank: zwei
// Mitarbeiter und der Dienst. Geräte und Schüler legt jeder Test unter seinem Namen an.
type geraeteTheke struct {
	pool             *pgxpool.Pool
	suffix           string
	ausgabe, annahme string
	svc              DeviceService
}

func neueGeraeteTheke(t *testing.T) *geraeteTheke {
	t.Helper()
	pool := pgtest.Pool(t)
	ctx := context.Background()
	g := &geraeteTheke{pool: pool, suffix: fmt.Sprintf("%d", time.Now().UnixNano())}
	for platz, ziel := range map[string]*string{"ausgabe": &g.ausgabe, "annahme": &g.annahme} {
		if err := pool.QueryRow(ctx, `INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
			VALUES ('Geraetetheke', $1, $2, 'mitarbeiter', true) RETURNING id`,
			platz, "geraetetheke-"+platz+"-"+g.suffix+"@schule.invalid").Scan(ziel); err != nil {
			t.Fatalf("Mitarbeiter anlegen: %v", err)
		}
	}
	t.Cleanup(func() {
		for _, sql := range []string{
			`DELETE FROM ausleihen WHERE geraet_id IN (SELECT id FROM geraete WHERE barcode_id LIKE 'G-TH-%-' || $1)`,
			`DELETE FROM geraete WHERE barcode_id LIKE 'G-TH-%-' || $1`,
			`DELETE FROM benutzer WHERE email LIKE 'geraetetheke-%-' || $1 || '@schule.invalid'`,
			`DELETE FROM leser WHERE vorname = 'Geraetetheke' AND $1 = $1`,
		} {
			if _, err := pool.Exec(ctx, sql, g.suffix); err != nil {
				t.Errorf("aufräumen (%s): %v", sql, err)
			}
		}
	})
	g.svc = NewDeviceService(pool, repository.NewStudentRepository(pool), repository.NewLoanRepository(pool),
		repository.NewAuditRepository(pool))
	return g
}

// geraet legt ein freies Gerät an und liefert seine Nummer.
func (g *geraeteTheke) geraet(t *testing.T, name string) string {
	t.Helper()
	nummer := "G-TH-" + name + "-" + g.suffix
	if _, err := g.pool.Exec(context.Background(), `INSERT INTO geraete (modellname, barcode_id) VALUES ('Tablet', $1)`, nummer); err != nil {
		t.Fatalf("Gerät anlegen: %v", err)
	}
	return nummer
}

// schueler legt einen Schüler der Klasse 05A an und liefert seine Kennung.
func (g *geraeteTheke) schueler(t *testing.T, nachname string) string {
	t.Helper()
	var id string
	if err := g.pool.QueryRow(context.Background(), `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ($1, 'Geraetetheke', $2, '05A', 2031) RETURNING id`, "GT-"+nachname+"-"+g.suffix, nachname).Scan(&id); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	return id
}

// ausleiheZeile ist, was die Ausleihe eines Geräts in der Datenbank trägt.
type ausleiheZeile struct {
	id, leser, ausgabe, annahme string
	frist                       time.Time
	offen, fremd                bool
}

// letzteAusleihe liest die jüngste Ausleihe des Geräts.
func (g *geraeteTheke) letzteAusleihe(t *testing.T, nummer string) ausleiheZeile {
	t.Helper()
	var z ausleiheZeile
	if err := g.pool.QueryRow(context.Background(), `
		SELECT a.id::text, a.schueler_id::text, coalesce(a.bearbeiter_id::text, ''), coalesce(a.rueckgabe_bearbeiter_id::text, ''),
		       a.rueckgabe_frist, a.rueckgabe_am IS NULL, a.ist_fremdrueckgabe
		FROM ausleihen a JOIN geraete ge ON ge.id = a.geraet_id
		WHERE ge.barcode_id = $1 ORDER BY a.erfasst_am DESC, a.id LIMIT 1`, nummer).
		Scan(&z.id, &z.leser, &z.ausgabe, &z.annahme, &z.frist, &z.offen, &z.fremd); err != nil {
		t.Fatalf("Ausleihe von %s lesen: %v", nummer, err)
	}
	return z
}

// offeneAusleihen zählt die offenen Ausleihen des Geräts.
func (g *geraeteTheke) offeneAusleihen(t *testing.T, nummer string) int {
	t.Helper()
	var n int
	if err := g.pool.QueryRow(context.Background(), `SELECT count(*) FROM ausleihen a JOIN geraete ge ON ge.id = a.geraet_id
		WHERE ge.barcode_id = $1 AND a.rueckgabe_am IS NULL`, nummer).Scan(&n); err != nil {
		t.Fatalf("offene Ausleihen zählen: %v", err)
	}
	return n
}

// warteAufWartende wartet, bis anzahl Sitzungen auf die Transaktion sperrer warten: unmittelbar
// oder hinter einer Sitzung, die auf sie wartet. Wer an derselben Zeile als Zweiter ansteht,
// wartet nach Auskunft der Datenbank auf den Ersten in der Schlange.
func warteAufWartende(t *testing.T, pool *pgxpool.Pool, sperrer pgx.Tx, anzahl int) {
	t.Helper()
	ctx := context.Background()
	var pid int
	if err := sperrer.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	for range 100 {
		var n int
		if err := pool.QueryRow(ctx, `
			WITH RECURSIVE wartende(pid) AS (
				SELECT pid FROM pg_stat_activity WHERE $1 = ANY (pg_blocking_pids(pid))
				UNION
				SELECT a.pid FROM pg_stat_activity a JOIN wartende w ON w.pid = ANY (pg_blocking_pids(a.pid))
			)
			SELECT count(*)::int FROM wartende`, pid).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n >= anzahl {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("es warten keine %d Sitzungen auf die Transaktion des Tests", anzahl)
}

// Die Antwort der Theke nennt die Ausleihe, die in der Datenbank steht: ihre Kennung und ihre
// Frist. Die Zeile trägt den Leser und den Mitarbeiter, der ausgegeben hat.
func TestGeraetAusleihe_AntwortNenntDieGeschriebeneAusleihe(t *testing.T) {
	g := neueGeraeteTheke(t)
	nummer := g.geraet(t, "antwort")
	leser := g.schueler(t, "Antwort")

	res, err := g.svc.HandleDeviceAction(context.Background(), nummer, &leser, true, false, g.ausgabe)
	if err != nil {
		t.Fatalf("Ausleihe: %v", err)
	}
	z := g.letzteAusleihe(t, nummer)
	if res.Type != "ausleihe" || res.LoanID == nil || *res.LoanID != z.id || res.DueDate == nil || !res.DueDate.Equal(z.frist) {
		t.Errorf("Antwort %+v nennt nicht die Ausleihe %s mit Frist %s", res, z.id, z.frist)
	}
	if res.Geraet == nil || res.Geraet.BarcodeID != nummer || res.Student == nil || res.Student.ID != leser {
		t.Errorf("Antwort nennt Gerät %+v und Leser %+v", res.Geraet, res.Student)
	}
	if !z.offen || z.leser != leser || z.ausgabe != g.ausgabe || z.annahme != "" {
		t.Errorf("Zeile der Ausleihe: %+v, erwartet offen, an %s, ausgegeben von %s", z, leser, g.ausgabe)
	}
}

// Eine Nummer, die kein Gerät trägt, ist an der Theke ein Bedienfehler (nicht gefunden), kein
// Serverfehler.
func TestGeraet_UnbekannteNummerIstNichtGefunden(t *testing.T) {
	g := neueGeraeteTheke(t)
	_, err := g.svc.HandleDeviceAction(context.Background(), "G-TH-unbekannt-"+g.suffix, nil, true, false, g.ausgabe)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("unbekannte Nummer: Fehler %v, erwartet ErrNotFound", err)
	}
}

// Bringt ein anderer als der Ausleiher das Gerät zurück, ist es eine Fremdrückgabe: Die
// Antwort nennt den Vorbesitzer mit Name und Klasse, die Zeile trägt das Merkmal und den
// Mitarbeiter, der angenommen hat. Bringt es der Ausleiher selbst, steht keines von beidem da.
func TestGeraetRueckgabe_FremdrueckgabeNenntDenVorbesitzer(t *testing.T) {
	g := neueGeraeteTheke(t)
	ctx := context.Background()
	ausleiher := g.schueler(t, "Ausleiher")
	bringer := g.schueler(t, "Bringer")

	fremd := g.geraet(t, "fremd")
	if _, err := g.svc.HandleDeviceAction(ctx, fremd, &ausleiher, true, false, g.ausgabe); err != nil {
		t.Fatalf("Ausleihe: %v", err)
	}
	res, err := g.svc.HandleDeviceAction(ctx, fremd, &bringer, true, false, g.annahme)
	if err != nil {
		t.Fatalf("Rückgabe durch einen anderen: %v", err)
	}
	z := g.letzteAusleihe(t, fremd)
	if res.Type != "rueckgabe" || !res.Fremdrueckgabe || res.LoanID == nil || *res.LoanID != z.id {
		t.Errorf("Antwort %+v, erwartet die Fremdrückgabe der Ausleihe %s", res, z.id)
	}
	if v := res.Vorbesitzer; v == nil || v.Vorname != "Geraetetheke" || v.Nachname != "Ausleiher" || v.Klasse != "05A" {
		t.Errorf("Vorbesitzer %+v, erwartet Geraetetheke Ausleiher, 05A", res.Vorbesitzer)
	}
	if z.offen || !z.fremd || z.leser != ausleiher || z.ausgabe != g.ausgabe || z.annahme != g.annahme {
		t.Errorf("Zeile nach der Fremdrückgabe: %+v", z)
	}

	selbst := g.geraet(t, "selbst")
	if _, err := g.svc.HandleDeviceAction(ctx, selbst, &ausleiher, true, false, g.ausgabe); err != nil {
		t.Fatalf("Ausleihe: %v", err)
	}
	res, err = g.svc.HandleDeviceAction(ctx, selbst, &ausleiher, true, false, g.annahme)
	if err != nil {
		t.Fatalf("Rückgabe durch den Ausleiher: %v", err)
	}
	z = g.letzteAusleihe(t, selbst)
	if res.Fremdrueckgabe || res.Vorbesitzer != nil || z.offen || z.fremd || z.annahme != g.annahme {
		t.Errorf("eigene Rückgabe: Antwort %+v, Zeile %+v; erwartet ohne Fremdrückgabe und Vorbesitzer", res, z)
	}
}

// Ist der Vorbesitzer inzwischen gelöscht, bleibt er in der Antwort ohne Namen, und die
// Rückgabe geht durch.
func TestGeraetRueckgabe_GeloeschterVorbesitzerHaeltNichtAuf(t *testing.T) {
	g := neueGeraeteTheke(t)
	ctx := context.Background()
	nummer := g.geraet(t, "geloescht")
	ausleiher := g.schueler(t, "Geloescht")
	if _, err := g.svc.HandleDeviceAction(ctx, nummer, &ausleiher, true, false, g.ausgabe); err != nil {
		t.Fatalf("Ausleihe: %v", err)
	}
	if _, err := g.pool.Exec(ctx, `UPDATE leser SET deleted_at = now() WHERE id = $1`, ausleiher); err != nil {
		t.Fatalf("Leser löschen: %v", err)
	}

	res, err := g.svc.HandleDeviceAction(ctx, nummer, nil, true, false, g.annahme)
	if err != nil {
		t.Fatalf("Rückgabe: %v", err)
	}
	if !res.Fremdrueckgabe || res.Vorbesitzer == nil || res.Vorbesitzer.Vorname != "" || res.Vorbesitzer.Nachname != "" {
		t.Errorf("Antwort %+v mit Vorbesitzer %+v, erwartet eine Fremdrückgabe ohne Namen", res, res.Vorbesitzer)
	}
	if g.offeneAusleihen(t, nummer) != 0 {
		t.Error("die Ausleihe ist nach der Rückgabe noch offen")
	}
}

// Zwei Plätze leihen dasselbe freie Gerät zugleich aus: Der zweite prallt an der Eindeutigkeit
// der offenen Ausleihe ab und bekommt die Meldung, dass das Gerät woanders verbucht ist. Der
// erste Platz ist hier eine Transaktion des Tests, die ihre Ausleihe geschrieben, aber noch
// nicht abgeschlossen hat; der Dienst sieht sie nicht und wartet beim Schreiben auf sie.
func TestGeraetAusleihe_ZweiterPlatzBekommtDenKonflikt(t *testing.T) {
	g := neueGeraeteTheke(t)
	ctx := context.Background()
	nummer := g.geraet(t, "konflikt")
	erster := g.schueler(t, "Erster")
	zweiter := g.schueler(t, "Zweiter")

	platzEins, err := g.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SafeRollback(ctx, platzEins)
	if _, err := platzEins.Exec(ctx, `INSERT INTO ausleihen (geraet_id, schueler_id, rueckgabe_frist, bearbeiter_id)
		SELECT id, $2, now() + interval '14 days', $3 FROM geraete WHERE barcode_id = $1`, nummer, erster, g.ausgabe); err != nil {
		t.Fatalf("Ausleihe am ersten Platz: %v", err)
	}

	fertig := make(chan error, 1)
	go func() {
		_, err := g.svc.HandleDeviceAction(ctx, nummer, &zweiter, true, false, g.annahme)
		fertig <- err
	}()
	warteAufWartende(t, g.pool, platzEins, 1)
	if err := platzEins.Commit(ctx); err != nil {
		t.Fatalf("ersten Platz abschließen: %v", err)
	}

	err = <-fertig
	if !errors.Is(err, ErrConflict) || !strings.Contains(fmt.Sprint(err), "woanders verbucht") {
		t.Errorf("zweiter Platz: Fehler %v, erwartet den Konflikt „woanders verbucht\"", err)
	}
	if z := g.letzteAusleihe(t, nummer); g.offeneAusleihen(t, nummer) != 1 || z.leser != erster {
		t.Errorf("nach dem Konflikt: %d offene Ausleihen, die jüngste an %s; erwartet eine an %s", g.offeneAusleihen(t, nummer), z.leser, erster)
	}
}

// Zwei Plätze nehmen dasselbe Gerät zugleich zurück: Einer bucht die Rückgabe, der andere
// wartet an der gesperrten Ausleihe und findet danach ein freies Gerät, zu dem ihm der Ausweis
// fehlt. Beide warten zuerst auf eine Transaktion des Tests, damit sie sicher zugleich ankommen.
func TestGeraetRueckgabe_ZweiPlaetzeBuchenNurEinmal(t *testing.T) {
	g := neueGeraeteTheke(t)
	ctx := context.Background()
	nummer := g.geraet(t, "doppelt")
	leser := g.schueler(t, "Doppelt")
	if _, err := g.svc.HandleDeviceAction(ctx, nummer, &leser, true, false, g.ausgabe); err != nil {
		t.Fatalf("Ausleihe: %v", err)
	}

	sperrer, err := g.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SafeRollback(ctx, sperrer)
	if _, err := sperrer.Exec(ctx, `SELECT a.id FROM ausleihen a JOIN geraete ge ON ge.id = a.geraet_id
		WHERE ge.barcode_id = $1 AND a.rueckgabe_am IS NULL FOR UPDATE OF a`, nummer); err != nil {
		t.Fatalf("Ausleihe sperren: %v", err)
	}

	type antwort struct {
		res *DeviceResult
		err error
	}
	antworten := make(chan antwort, 2)
	for range 2 {
		go func() {
			res, err := g.svc.HandleDeviceAction(ctx, nummer, nil, true, false, g.annahme)
			antworten <- antwort{res, err}
		}()
	}
	warteAufWartende(t, g.pool, sperrer, 2)
	if err := sperrer.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	gebucht, abgewiesen := 0, 0
	for range 2 {
		a := <-antworten
		switch {
		case a.err == nil && a.res != nil && a.res.Type == "rueckgabe":
			gebucht++
		case errors.Is(a.err, ErrInvalidState):
			abgewiesen++
		default:
			t.Errorf("unerwartete Antwort: %+v, Fehler %v", a.res, a.err)
		}
	}
	if gebucht != 1 || abgewiesen != 1 {
		t.Errorf("%d Rückgaben gebucht, %d abgewiesen; erwartet je eine", gebucht, abgewiesen)
	}
	if g.offeneAusleihen(t, nummer) != 0 {
		t.Error("die Ausleihe ist nach der Rückgabe noch offen")
	}
}

// protokollScheitert ist ein Protokoll, das Ausleihe und Rückgabe nicht schreiben kann.
type protokollScheitert struct{ repository.AuditRepository }

var errProtokollProbe = errors.New("protokoll nicht schreibbar")

func (protokollScheitert) LogAusleihe(context.Context, pgx.Tx, string, string, string, string) error {
	return errProtokollProbe
}

func (protokollScheitert) LogRueckgabe(context.Context, pgx.Tx, string, string, string, string) error {
	return errProtokollProbe
}

// Lässt sich der Protokolleintrag nicht schreiben, gibt es die Ausleihe nicht und die Rückgabe
// nicht: Beide stehen in derselben Transaktion wie ihr Eintrag.
func TestGeraet_OhneProtokollWederAusleiheNochRueckgabe(t *testing.T) {
	g := neueGeraeteTheke(t)
	ctx := context.Background()
	nummer := g.geraet(t, "protokoll")
	leser := g.schueler(t, "Protokoll")
	ohneProtokoll := NewDeviceService(g.pool, repository.NewStudentRepository(g.pool), repository.NewLoanRepository(g.pool),
		protokollScheitert{repository.NewAuditRepository(g.pool)})

	if _, err := ohneProtokoll.HandleDeviceAction(ctx, nummer, &leser, true, false, g.ausgabe); !errors.Is(err, errProtokollProbe) {
		t.Fatalf("Ausleihe ohne Protokoll: Fehler %v, erwartet den des Protokolls", err)
	}
	if n := g.offeneAusleihen(t, nummer); n != 0 {
		t.Fatalf("%d offene Ausleihen, obwohl das Protokoll scheiterte", n)
	}

	if _, err := g.svc.HandleDeviceAction(ctx, nummer, &leser, true, false, g.ausgabe); err != nil {
		t.Fatalf("Ausleihe: %v", err)
	}
	if _, err := ohneProtokoll.HandleDeviceAction(ctx, nummer, &leser, true, false, g.annahme); !errors.Is(err, errProtokollProbe) {
		t.Fatalf("Rückgabe ohne Protokoll: Fehler %v, erwartet den des Protokolls", err)
	}
	if z := g.letzteAusleihe(t, nummer); !z.offen || z.annahme != "" {
		t.Errorf("die Rückgabe steht an der Ausleihe, obwohl das Protokoll scheiterte: %+v", z)
	}
}
