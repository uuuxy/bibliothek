package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Schreibpfad des Schadensersatz-Bescheids am echten Postgres.
//
// Die zentrale Zusicherung ist die Referenznummer: Sie wird je Brief vergeben und NIE
// zweimal — sonst lässt sich eine eingehende Zahlung nicht zuordnen. Alles andere hängt
// daran: Wird ein Brief abgewiesen, darf auch keine Nummer verbraucht sein.

// bescheidReset leert Briefe UND Nummernkreis.
//
// resetBestandsdaten nimmt sie NICHT mit: schadensersatz_bescheide hängt am Schüler mit
// ON DELETE SET NULL, wird beim TRUNCATE der Bestandstabellen also nur entkoppelt und
// nicht geleert — und der Zähler in schadensersatz_nummern lebt ohnehin für sich. Ohne
// diesen Reset zählen die Tests dieser Datei ihre Nummern gemeinsam weiter, und jede
// Erwartung „letzte_nr = 1" wäre von der Ausführungsreihenfolge abhängig.
func bescheidReset(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`TRUNCATE schadensersatz_bescheide, schadensersatz_nummern CASCADE`); err != nil {
		t.Fatalf("Bescheid-Reset: %v", err)
	}
}

// bescheidAngabenSetzen hinterlegt die Pflichtangaben, ohne die kein Bescheid entsteht.
func bescheidAngabenSetzen(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	werte := map[string]string{
		"bescheid_bereich_nr":   "5830",
		"bescheid_schulnummer":  "1234",
		"bescheid_aufsicht":     "Staatliches Schulamt, Musterstraße 1, 12345 Musterstadt",
		"bescheid_schulleitung": "Dr. Beispiel, Schulleitung",
		"schule_name":           "Testschule",
		"schule_strasse":        "Schulweg 1",
		"schule_plz":            "12345",
		"schule_ort":            "Musterstadt",
	}
	for k, v := range werte {
		if _, err := pool.Exec(ctx, `
			INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)
			ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, k, v); err != nil {
			t.Fatalf("Einstellung %s: %v", k, err)
		}
	}
	t.Cleanup(func() {
		for k := range werte {
			if _, err := pool.Exec(context.Background(),
				`DELETE FROM system_einstellungen WHERE schluessel = $1`, k); err != nil {
				t.Logf("Aufräumen %s: %v", k, err)
			}
		}
	})
}

// bescheidForderung legt eine offene Forderung an und liefert ihre ID.
func bescheidForderung(t *testing.T, pool *pgxpool.Pool, schuelerID, exemplarID, art, beschreibung string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, art)
		VALUES ($1, $2, $3, 0, $4) RETURNING id`,
		exemplarID, schuelerID, beschreibung, art).Scan(&id); err != nil {
		t.Fatalf("Forderung anlegen: %v", err)
	}
	return id
}

// bescheidErstellenUeberHandler ruft POST /api/schueler/{id}/bescheide.
func bescheidErstellenUeberHandler(t *testing.T, srv *Server, pool *pgxpool.Pool, schuelerID, rumpf string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/schueler/"+schuelerID+"/bescheide", strings.NewReader(rumpf))
	req.SetPathValue("id", schuelerID)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: adminFuerAudit(t, pool), Rolle: auth.RoleAdmin}))
	rec := httptest.NewRecorder()
	srv.BescheidErstellenHandler(repository.NewBescheidRepository(pool), repository.NewAuditRepository(pool))(rec, req)
	return rec
}

// bescheidRumpf baut den Anfragerumpf für die genannten Forderungen.
func bescheidRumpf(frist string, betraege map[string]float64) string {
	teile := make([]string, 0, len(betraege))
	for id, betrag := range betraege {
		teile = append(teile, fmt.Sprintf(`{"schadensfall_id":%q,"betrag":%.2f}`, id, betrag))
	}
	return fmt.Sprintf(`{"mittel":"land","frist_bis":%q,"positionen":[%s]}`, frist, strings.Join(teile, ","))
}

func in28Tagen() string { return time.Now().AddDate(0, 0, 28).Format("2006-01-02") }

// bescheidLernmittel legt einen Lernmittel-Titel an — nur deren Forderungen gehören auf
// einen Bescheid des Landes.
func bescheidLernmittel(t *testing.T, pool *pgxpool.Pool, titel string) string {
	t.Helper()
	id := seedMonitorTitel(t, pool, titel, "Autor", true, 0)
	if _, err := pool.Exec(context.Background(), `UPDATE buecher_titel SET ist_lernmittel = true WHERE id = $1`, id); err != nil {
		t.Fatalf("Lernmittel setzen: %v", err)
	}
	return id
}

// Ein Bescheid trägt den Wortlaut der Lernmittelfreiheit („Eigentum des Landes …") und
// die Bankverbindung des Landes. Der Topf „schultraeger" hat noch keinen eigenen Brief
// (Konzept 4.7, Etappe 3) — bis dahin wies der Server ihn nicht ab, und der Dialog
// schickte ihn, sobald keine gewählte Forderung ein Lernmittel war: Die Eltern bekamen
// für ein Buch der Schülerbücherei einen Landes-Bescheid mit Landeskonto (Bestands-
// Durchgang 10.09.2026). Ebenso wenig darf ein Nicht-Lernmittel auf einen Landes-
// Bescheid. Beides wird abgewiesen, ohne eine Nummer zu verbrauchen.
func TestBescheidErstellen_NurLernmittelAufDemLandesBescheid(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	bescheidReset(t, pool)
	bescheidAngabenSetzen(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	sid := seedSchueler(t, pool, "S-BESCHEID-TOPF", "Topfkind", "08G2")
	roman := seedMonitorTitel(t, pool, "Roman der Buecherei", "Autor", true, 0)
	f := bescheidForderung(t, pool, sid, exemplar(t, pool, roman, "BESCH-TOPF-1", true, ""), "beschaedigt", "Roman beschädigt")

	for _, mittel := range []string{"land", "schultraeger"} {
		rumpf := fmt.Sprintf(`{"mittel":%q,"frist_bis":%q,"positionen":[{"schadensfall_id":%q,"betrag":12.00}]}`,
			mittel, in28Tagen(), f)
		rec := bescheidErstellenUeberHandler(t, srv, pool, sid, rumpf)
		if rec.Code != http.StatusConflict {
			t.Errorf("mittel=%s für ein Nicht-Lernmittel: Status %d, want 409: %s", mittel, rec.Code, rec.Body.String())
		}
	}
	var briefe, nummern int
	if err := pool.QueryRow(context.Background(), `SELECT (SELECT count(*) FROM schadensersatz_bescheide),
		(SELECT count(*) FROM schadensersatz_nummern)`).Scan(&briefe, &nummern); err != nil {
		t.Fatal(err)
	}
	if briefe != 0 || nummern != 0 {
		t.Errorf("abgewiesen, aber %d Brief(e) und %d Nummernzeile(n) angelegt", briefe, nummern)
	}
}

func TestBescheidErstellen_SchreibtBriefUndOrdnetForderungenZu(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	bescheidReset(t, pool)
	ctx := context.Background()
	bescheidAngabenSetzen(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	sid := seedSchueler(t, pool, "S-BESCHEID-1", "Bescheidkind", "08G2")
	titelID := bescheidLernmittel(t, pool, "Mathematik 7")
	ex1 := exemplar(t, pool, titelID, "BESCH-EX-1", true, "")
	ex2 := exemplar(t, pool, titelID, "BESCH-EX-2", true, "")
	f1 := bescheidForderung(t, pool, sid, ex1, "nicht_zurueckgegeben", "Mathematik 7 nicht zurück")
	f2 := bescheidForderung(t, pool, sid, ex2, "beschaedigt", "Mathematik 7 beschädigt")

	rec := bescheidErstellenUeberHandler(t, srv, pool, sid,
		bescheidRumpf(in28Tagen(), map[string]float64{f1: 24.90, f2: 12.00}))
	if rec.Code != http.StatusCreated {
		t.Fatalf("Status %d, want 201: %s", rec.Code, rec.Body.String())
	}
	var b repository.Bescheid
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}

	// 1. Die Referenznummer hat die Form des Musterschreibens: vier vierstellige Blöcke.
	if !strings.HasPrefix(b.Referenznummer, "5830 ") || !strings.HasSuffix(b.Referenznummer, " 1234 0001") {
		t.Errorf("Referenznummer = %q, erwartet „5830 <Jahr> 1234 0001\"", b.Referenznummer)
	}
	// 2. Die Summe ist die der Positionen, nicht die der Forderungsbeträge von vorher (0).
	if b.Gesamtbetrag != 36.90 {
		t.Errorf("Gesamtbetrag = %.2f, want 36.90", b.Gesamtbetrag)
	}
	if b.AnzahlPositionen != 2 {
		t.Errorf("AnzahlPositionen = %d, want 2", b.AnzahlPositionen)
	}
	// 3. Die Forderungen tragen den festgesetzten Betrag UND den Brief.
	for id, want := range map[string]float64{f1: 24.90, f2: 12.00} {
		var betrag float64
		var bescheidID *string
		if err := pool.QueryRow(ctx,
			`SELECT betrag::float8, bescheid_id::text FROM schadensfaelle WHERE id = $1`, id).Scan(&betrag, &bescheidID); err != nil {
			t.Fatal(err)
		}
		if betrag != want {
			t.Errorf("Forderung %s: Betrag %.2f, want %.2f", id, betrag, want)
		}
		if bescheidID == nil || *bescheidID != b.ID {
			t.Errorf("Forderung %s hängt nicht am Bescheid", id)
		}
	}
	// 4. Der Empfänger-Snapshot trägt Anrede und Name (Grundlage des Nachdrucks).
	var snapshot string
	if err := pool.QueryRow(ctx,
		`SELECT empfaenger_snapshot::text FROM schadensersatz_bescheide WHERE id = $1`, b.ID).Scan(&snapshot); err != nil {
		t.Fatal(err)
	}
	for _, erwartet := range []string{"Erziehungsberechtigte", "Bescheidkind"} {
		if !strings.Contains(snapshot, erwartet) {
			t.Errorf("Snapshot ohne %q: %s", erwartet, snapshot)
		}
	}
	// 5. Das Audit kennt Nummer und Betrag.
	var details string
	if err := pool.QueryRow(ctx,
		`SELECT details::text FROM audit_logs WHERE aktion = $1 ORDER BY zeitstempel DESC LIMIT 1`,
		auditBescheidErstellt).Scan(&details); err != nil {
		t.Fatalf("Audit-Eintrag fehlt: %v", err)
	}
	if !strings.Contains(details, b.Referenznummer) {
		t.Errorf("Audit ohne Referenznummer: %s", details)
	}
}

// Eine Forderung darf nie auf zwei Briefen stehen — sonst gäbe es zwei Nummern für
// dieselbe Forderung, und die Zahlung wäre nicht zuordenbar.
func TestBescheidErstellen_ForderungNurEinmal(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	bescheidReset(t, pool)
	bescheidAngabenSetzen(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	sid := seedSchueler(t, pool, "S-BESCHEID-2", "Zweitkind", "08G2")
	titelID := bescheidLernmittel(t, pool, "Deutsch 8")
	ex := exemplar(t, pool, titelID, "BESCH-EX-3", true, "")
	f := bescheidForderung(t, pool, sid, ex, "nicht_zurueckgegeben", "Deutsch 8 nicht zurück")

	if rec := bescheidErstellenUeberHandler(t, srv, pool, sid,
		bescheidRumpf(in28Tagen(), map[string]float64{f: 18.50})); rec.Code != http.StatusCreated {
		t.Fatalf("erster Bescheid: %d %s", rec.Code, rec.Body.String())
	}
	rec := bescheidErstellenUeberHandler(t, srv, pool, sid,
		bescheidRumpf(in28Tagen(), map[string]float64{f: 18.50}))
	if rec.Code != http.StatusConflict {
		t.Errorf("zweiter Bescheid auf dieselbe Forderung: %d, want 409: %s", rec.Code, rec.Body.String())
	}

	// Und die Nummer des abgewiesenen Briefs ist NICHT verbraucht: Die Transaktion ist
	// zurückgerollt, der Zähler steht auf 1.
	var letzte int
	if err := pool.QueryRow(context.Background(),
		`SELECT letzte_nr FROM schadensersatz_nummern WHERE mittel = 'land' AND kassenjahr = $1`,
		time.Now().AddDate(0, 0, 28).Year()).Scan(&letzte); err != nil {
		t.Fatal(err)
	}
	if letzte != 1 {
		t.Errorf("letzte_nr = %d, want 1 — der abgewiesene Brief hat eine Nummer verbraucht", letzte)
	}
}

// Ohne die Pflichtangaben entsteht kein Bescheid — und keine Nummer.
func TestBescheidErstellen_OhneAngabenKeinBriefUndKeineNummer(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	bescheidReset(t, pool)
	ctx := context.Background()
	// Bewusst OHNE bescheidAngabenSetzen; Reste aus anderen Tests räumen.
	for _, k := range []string{"bescheid_bereich_nr", "bescheid_schulnummer", "bescheid_aufsicht", "bescheid_schulleitung"} {
		if _, err := pool.Exec(ctx, `DELETE FROM system_einstellungen WHERE schluessel = $1`, k); err != nil {
			t.Fatal(err)
		}
	}
	srv := &Server{DB: &db.Database{Pool: pool}}

	sid := seedSchueler(t, pool, "S-BESCHEID-3", "Ohneangaben", "08G2")
	titelID := bescheidLernmittel(t, pool, "Englisch 9")
	ex := exemplar(t, pool, titelID, "BESCH-EX-4", true, "")
	f := bescheidForderung(t, pool, sid, ex, "nicht_zurueckgegeben", "Englisch 9 nicht zurück")

	rec := bescheidErstellenUeberHandler(t, srv, pool, sid,
		bescheidRumpf(in28Tagen(), map[string]float64{f: 20.00}))
	if rec.Code != http.StatusConflict {
		t.Fatalf("Status %d, want 409: %s", rec.Code, rec.Body.String())
	}
	// Die Meldung nennt, WAS fehlt — sonst sucht das Sekretariat.
	for _, erwartet := range []string{"Schulamtsbereichs", "Schulnummer", "Aufsichtsbehörde", "Einstellungen"} {
		if !strings.Contains(rec.Body.String(), erwartet) {
			t.Errorf("die Meldung nennt %q nicht: %s", erwartet, rec.Body.String())
		}
	}
	var briefe, nummern int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schadensersatz_bescheide WHERE schueler_id = $1`, sid).Scan(&briefe); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schadensersatz_nummern`).Scan(&nummern); err != nil {
		t.Fatal(err)
	}
	if briefe != 0 {
		t.Errorf("%d Bescheid(e) trotz fehlender Angaben", briefe)
	}
	if nummern != 0 {
		t.Errorf("%d Nummernkreis-Zeile(n) angelegt, obwohl kein Brief entstand", nummern)
	}
}

// Der Nummernkreis unter gleichzeitigem Zugriff: Jede Nummer genau einmal, keine Lücke.
//
// Der Grund für diesen Test ist die Bugklasse „Zwei Generatoren, ein Nummernkreis":
// Ein MAX+1 hätte hier zwei gleichen Briefen dieselbe Nummer gegeben. Das UPDATE …
// RETURNING sperrt die Zeile des Generators, die Nebenläufigen warten.
func TestBescheidNummernkreis_ParallelKeineDoppelte(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	bescheidReset(t, pool)
	ctx := context.Background()
	bescheidAngabenSetzen(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	const anzahl = 8
	titelID := bescheidLernmittel(t, pool, "Parallelbuch")
	type auftrag struct{ sid, forderung string }
	auftraege := make([]auftrag, 0, anzahl)
	for i := 0; i < anzahl; i++ {
		sid := seedSchueler(t, pool, fmt.Sprintf("S-PARALLEL-%d", i), fmt.Sprintf("Parallelkind%d", i), "08G2")
		ex := exemplar(t, pool, titelID, fmt.Sprintf("PARALLEL-EX-%d", i), true, "")
		auftraege = append(auftraege, auftrag{sid, bescheidForderung(t, pool, sid, ex, "nicht_zurueckgegeben", "Parallelbuch")})
	}

	frist := in28Tagen()
	var wg sync.WaitGroup
	nummern := make([]string, anzahl)
	codes := make([]int, anzahl)
	for i, a := range auftraege {
		wg.Add(1)
		go func(i int, a auftrag) {
			defer wg.Done()
			rec := bescheidErstellenUeberHandler(t, srv, pool, a.sid,
				bescheidRumpf(frist, map[string]float64{a.forderung: 10.00}))
			codes[i] = rec.Code
			var b repository.Bescheid
			if err := json.Unmarshal(rec.Body.Bytes(), &b); err == nil {
				nummern[i] = b.Referenznummer
			}
		}(i, a)
	}
	wg.Wait()

	gesehen := map[string]int{}
	for i, nr := range nummern {
		if codes[i] != http.StatusCreated {
			t.Errorf("Brief %d: Status %d", i, codes[i])
			continue
		}
		gesehen[nr]++
	}
	for nr, n := range gesehen {
		if n > 1 {
			t.Errorf("Referenznummer %q wurde %d mal vergeben — eine Zahlung darauf ist nicht zuordenbar", nr, n)
		}
	}
	if len(gesehen) != anzahl {
		t.Errorf("%d verschiedene Nummern bei %d Briefen", len(gesehen), anzahl)
	}
	var letzte int
	if err := pool.QueryRow(ctx,
		`SELECT letzte_nr FROM schadensersatz_nummern WHERE mittel = 'land' AND kassenjahr = $1`,
		time.Now().AddDate(0, 0, 28).Year()).Scan(&letzte); err != nil {
		t.Fatal(err)
	}
	if letzte != anzahl {
		t.Errorf("letzte_nr = %d, want %d — der Zähler hat Lücken oder Doppelungen", letzte, anzahl)
	}
}

// Die Übergabe geht erst nach Fristablauf und nur einmal.
func TestBescheidUebergabe_ErstNachFristUndNurEinmal(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	bescheidReset(t, pool)
	ctx := context.Background()
	bescheidAngabenSetzen(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	repo := repository.NewBescheidRepository(pool)

	sid := seedSchueler(t, pool, "S-BESCHEID-4", "Fristkind", "08G2")
	titelID := bescheidLernmittel(t, pool, "Physik 9")
	ex := exemplar(t, pool, titelID, "BESCH-EX-5", true, "")
	f := bescheidForderung(t, pool, sid, ex, "nicht_zurueckgegeben", "Physik 9 nicht zurück")

	rec := bescheidErstellenUeberHandler(t, srv, pool, sid,
		bescheidRumpf(in28Tagen(), map[string]float64{f: 22.00}))
	if rec.Code != http.StatusCreated {
		t.Fatalf("Bescheid: %d %s", rec.Code, rec.Body.String())
	}
	var b repository.Bescheid
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}

	// 1. Die Frist läuft noch: Übergabe abgewiesen.
	uebergeben := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/bescheide/"+b.ID+"/uebergeben", nil)
		req.SetPathValue("id", b.ID)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: adminFuerAudit(t, pool), Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		srv.BescheidUebergebenHandler(repo, repository.NewAuditRepository(pool))(rec, req)
		return rec
	}
	if rec := uebergeben(); rec.Code != http.StatusConflict {
		t.Errorf("Übergabe vor Fristablauf: %d, want 409: %s", rec.Code, rec.Body.String())
	}

	// 2. Frist vorbei: Übergabe geht.
	if _, err := pool.Exec(ctx,
		`UPDATE schadensersatz_bescheide SET frist_bis = CURRENT_DATE - 1 WHERE id = $1`, b.ID); err != nil {
		t.Fatal(err)
	}
	if rec := uebergeben(); rec.Code != http.StatusOK {
		t.Fatalf("Übergabe nach Fristablauf: %d %s", rec.Code, rec.Body.String())
	}

	// 3. Ein zweites Mal nicht — der Zeitpunkt der Übergabe darf nicht überschrieben werden.
	if rec := uebergeben(); rec.Code != http.StatusConflict {
		t.Errorf("zweite Übergabe: %d, want 409", rec.Code)
	}

	var status string
	var uebergebenAm *time.Time
	if err := pool.QueryRow(ctx,
		`SELECT status, uebergeben_am FROM schadensersatz_bescheide WHERE id = $1`, b.ID).Scan(&status, &uebergebenAm); err != nil {
		t.Fatal(err)
	}
	if status != "uebergeben" || uebergebenAm == nil {
		t.Errorf("Status %q, uebergeben_am %v", status, uebergebenAm)
	}
}

// Der Nachdruck ist derselbe Brief: gleiche Nummer, gleicher Betrag, gleiche Anschrift —
// auch wenn der Schüler inzwischen umgezogen ist. Dafür ist der Snapshot da.
func TestBescheidNachdruck_BleibtDerselbeBrief(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	bescheidReset(t, pool)
	ctx := context.Background()
	bescheidAngabenSetzen(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	repo := repository.NewBescheidRepository(pool)

	sid := seedSchueler(t, pool, "S-BESCHEID-5", "Umzugskind", "08G2")
	if _, err := pool.Exec(ctx,
		`UPDATE schueler SET strasse = 'Altweg', hausnummer = '1', plz = '11111', ort = 'Altstadt' WHERE id = $1`, sid); err != nil {
		t.Fatal(err)
	}
	titelID := bescheidLernmittel(t, pool, "Chemie 9")
	ex := exemplar(t, pool, titelID, "BESCH-EX-6", true, "")
	f := bescheidForderung(t, pool, sid, ex, "nicht_zurueckgegeben", "Chemie 9 nicht zurück")

	rec := bescheidErstellenUeberHandler(t, srv, pool, sid,
		bescheidRumpf(in28Tagen(), map[string]float64{f: 19.90}))
	if rec.Code != http.StatusCreated {
		t.Fatalf("Bescheid: %d %s", rec.Code, rec.Body.String())
	}
	var b repository.Bescheid
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}

	// Umzug NACH dem Brief.
	if _, err := pool.Exec(ctx,
		`UPDATE schueler SET strasse = 'Neuweg', ort = 'Neustadt' WHERE id = $1`, sid); err != nil {
		t.Fatal(err)
	}

	pdfAbrufen := func() string {
		req := httptest.NewRequest(http.MethodGet, "/api/bescheide/"+b.ID+"/pdf", nil)
		req.SetPathValue("id", b.ID)
		rec := httptest.NewRecorder()
		srv.BescheidPDFHandler(repo)(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("PDF: %d %s", rec.Code, rec.Body.String())
		}
		return pdfText(t, rec.Body.Bytes())
	}

	text := pdfAbrufen()
	for _, erwartet := range []string{b.Referenznummer, "19,90", "Altweg", "Altstadt", "Umzugskind"} {
		if !strings.Contains(text, erwartet) {
			t.Errorf("der Brief nennt %q nicht", erwartet)
		}
	}
	if strings.Contains(text, "Neuweg") {
		t.Error("der Nachdruck nimmt die NEUE Anschrift — dann ist es nicht mehr derselbe Bescheid")
	}

	// Zweiter Abruf: unverändert, und die Nummer wurde nicht neu gezogen.
	if zweiter := pdfAbrufen(); !strings.Contains(zweiter, b.Referenznummer) {
		t.Error("der Nachdruck trägt eine andere Nummer")
	}
	var letzte int
	if err := pool.QueryRow(ctx,
		`SELECT letzte_nr FROM schadensersatz_nummern WHERE mittel = 'land' AND kassenjahr = $1`,
		time.Now().AddDate(0, 0, 28).Year()).Scan(&letzte); err != nil {
		t.Fatal(err)
	}
	if letzte != 1 {
		t.Errorf("letzte_nr = %d, want 1 — ein Nachdruck hat eine Nummer verbraucht", letzte)
	}
}
