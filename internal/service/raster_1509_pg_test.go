//go:build raster

package service

// Nachstellung Rasterdurchgang 15.09.2026 abends (OFFEN.md 5.15). Build-Tag raster: Die Tests
// laufen nur mit -tags raster, damit die absichtlich roten Nachstellungen keine parallele
// Sitzung und keinen Hook stören. Jeder Test beschreibt den Schaden; rot heißt „bestätigt".

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"bibliothek/repository"

	"github.com/google/uuid"
)

// onlineFremdrueckgabe bucht wie der erste Online-Scan eines Buchs, das auf jemand anderem
// steht (handleForeignReturn): NUR die Rücknahme dort, eigene Transaktion — kein Umbuchen
// (Produktentscheidung 10.07.).
func (w *nbWelt) onlineFremdrueckgabe(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	loans := repository.NewLoanRepository(w.pool)
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	aktiv, err := loans.GetActiveLoanByCopyIDTx(ctx, tx, w.exemplarID)
	if err != nil || aktiv == nil {
		t.Fatalf("aktive Ausleihe für die Fremdrückgabe: %v %v", aktiv, err)
	}
	if err := loans.ReturnLoanTx(ctx, tx, aktiv.ID, w.staff, true); err != nil {
		t.Fatalf("Fremdrückgabe: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

// onlineAusleihe bucht wie die Online-Theke: Steht das Buch auf jemand anderem, nimmt der
// erste Scan es dort zurück (onlineFremdrueckgabe), der zweite leiht aus (handleNewLoan) —
// zwei Transaktionen, zwei Stempel.
func (w *nbWelt) onlineAusleihe(t *testing.T, schueler string) string {
	t.Helper()
	ctx := context.Background()
	loans := repository.NewLoanRepository(w.pool)
	if n, _ := w.offeneAusleihen(t); n > 0 {
		w.onlineFremdrueckgabe(t)
	}
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	loan, err := loans.CreateLoanTx(ctx, tx, w.exemplarID, schueler, w.staff, time.Now().AddDate(0, 0, 14))
	if err != nil {
		t.Fatalf("ausleihen: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	return loan.ID
}

// onlineRueckgabe gibt wie der Online-Scan zurück (loan_return.go).
func (w *nbWelt) onlineRueckgabe(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	loans := repository.NewLoanRepository(w.pool)
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	aktiv, err := loans.GetActiveLoanByCopyIDTx(ctx, tx, w.exemplarID)
	if err != nil || aktiv == nil {
		t.Fatalf("aktive Ausleihe für die Rückgabe: %v %v", aktiv, err)
	}
	if err := loans.ReturnLoanTx(ctx, tx, aktiv.ID, w.staff, false); err != nil {
		t.Fatalf("zurückgeben: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func (w *nbWelt) stempel(t *testing.T) *time.Time {
	t.Helper()
	var s *time.Time
	if err := w.pool.QueryRow(context.Background(), `SELECT letzte_bewegung_am FROM buecher_exemplare WHERE id = $1`, w.exemplarID).Scan(&s); err != nil {
		t.Fatalf("Stempel lesen: %v", err)
	}
	return s
}

func (w *nbWelt) meldungenDerSchluessel(t *testing.T, schluessel ...string) []string {
	t.Helper()
	rows, err := w.pool.Query(context.Background(),
		`SELECT ergebnis || ': ' || coalesce(grund, '') FROM nachbuch_meldungen WHERE idempotency_key = ANY($1::uuid[]) ORDER BY erstellt_am`, schluessel)
	if err != nil {
		t.Fatalf("Meldungen lesen: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			t.Fatalf("Meldung: %v", err)
		}
		out = append(out, m)
	}
	return out
}

// Verdacht A: Die Uhr von Theke 1 geht vor. Anna gibt dort offline zurück; danach leiht Ben
// dasselbe Buch online an Theke 2. Beim Nachbuchen liegt der Scan-Zeitpunkt (Theken-Uhr,
// gekappt auf Serverzeit) nach Bens Ausleihe — die Rückgabe trifft Bens Ausleihe.
func TestRaster_VorgehendeThekenUhrBeendetJuengereAusleihe(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()

	w.onlineAusleihe(t, w.anna)

	echteRueckgabe := time.Now()
	thekenUhr := echteRueckgabe.Add(15 * time.Minute) // Theke 1 geht 15 Minuten vor
	time.Sleep(50 * time.Millisecond)

	benLoan := w.onlineAusleihe(t, w.ben) // nach der echten Rückgabe, an Theke 2

	schluessel := uuid.NewString()
	erg, err := w.svc.Nachbuchen(ctx, NachbuchEintrag{
		Schluessel: schluessel, Absicht: NachbuchAbsichtRueckgabe, Barcode: w.code,
		GescanntAm: thekenUhr, StaffID: w.staff,
	})
	if err != nil {
		t.Fatalf("nachbuchen: %v", err)
	}
	var benOffen bool
	if err := w.pool.QueryRow(ctx, `SELECT rueckgabe_am IS NULL FROM ausleihen WHERE id = $1`, benLoan).Scan(&benOffen); err != nil {
		t.Fatalf("Bens Ausleihe: %v", err)
	}
	meldungen := w.meldungenDerSchluessel(t, schluessel)
	t.Logf("Ergebnis %q · Bens Ausleihe offen: %v · Meldungen: %v", erg.Ergebnis, benOffen, meldungen)
	if !benOffen {
		t.Errorf("NACHGESTELLT: Ben hat das Buch in der Hand, seine Ausleihe ist beendet (Ergebnis %q, Meldungen %v)", erg.Ergebnis, meldungen)
	}
}

// Neuer Verdacht A: „Schlüssel bekannt" heißt, der Online-Versand ist vollständig gebucht
// (saveToCache speichert jede Antwort < 500). Hat Anna das Buch danach online
// zurückgegeben, darf das Nachbuchen desselben Eintrags es ihr nicht erneut ausleihen.
func TestRaster_BekannterSchluesselNachVollstaendigerOnlineBuchung(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	schluessel := uuid.NewString()
	scan := time.Now()

	// Online-Versand mit Schlüssel: gebucht, Antwort gespeichert — die Antwort erreichte die
	// Theke nicht (502), der Eintrag ging in die Warteschlange.
	w.onlineAusleihe(t, w.anna)
	if _, err := w.pool.Exec(ctx, `INSERT INTO idempotency_keys (idempotency_key, response_data, status_code) VALUES ($1, '{"success": true}', 200)`, schluessel); err != nil {
		t.Fatalf("Idempotenz-Antwort: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	w.onlineRueckgabe(t) // später, an einer Theke mit Netz
	vorher := w.stempel(t)

	// Der Handler fragt wie schluesselSchonGesehen: Zeile da, nicht in Arbeit.
	antwort, err := repository.LiesIdempotenzAntwort(ctx, w.pool, schluessel)
	bekannt := err == nil && antwort != nil && !antwort.InArbeit()
	erg, err := w.svc.Nachbuchen(ctx, NachbuchEintrag{
		Schluessel: schluessel, Absicht: NachbuchAbsichtAusleihe, Barcode: w.code, GescanntAm: scan,
		SchuelerID: &w.anna, SchluesselBekannt: bekannt, StaffID: w.staff,
	})
	if err != nil {
		t.Fatalf("nachbuchen: %v", err)
	}
	n, bei := w.offeneAusleihen(t)
	nachher := w.stempel(t)
	t.Logf("bekannt=%v · Ergebnis %q · offene Ausleihen %d (bei %s) · Stempel %v → %v", bekannt, erg.Ergebnis, n, bei, vorher, nachher)
	if n != 0 {
		t.Errorf("NACHGESTELLT: Anna hat das Buch zurückgegeben, das Nachbuchen leiht es ihr erneut aus (Ergebnis %q)", erg.Ergebnis)
	}
	if vorher != nil && nachher != nil && nachher.Before(*vorher) {
		t.Errorf("NACHGESTELLT: Der Bewegungsstempel läuft rückwärts: %v → %v", vorher, nachher)
	}
}

// Verdacht B: Die Theke schickt eine Portion nach einem Timeout erneut, obwohl der erste
// Aufruf gebucht hat. Die Tür schreibt ihre Schlüssel nicht in idempotency_keys.
func TestRaster_WiederholtePortionMeldetGebuchtesAlsAbweichung(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	jetzt := time.Now()
	portion := []NachbuchEintrag{
		w.eintrag(NachbuchAbsichtAusleihe, &w.anna, jetzt.Add(-10*time.Minute)),
		w.eintrag(NachbuchAbsichtRueckgabe, nil, jetzt.Add(-5*time.Minute)),
	}
	for _, e := range portion {
		erg, err := w.svc.Nachbuchen(ctx, e)
		if err != nil {
			t.Fatalf("erster Aufruf: %v", err)
		}
		t.Logf("erster Aufruf: %s → %q", e.Absicht, erg.Ergebnis)
	}
	var schluessel []string
	for i := range portion {
		schluessel = append(schluessel, portion[i].Schluessel)
		var drin int
		if err := w.pool.QueryRow(ctx, `SELECT count(*) FROM idempotency_keys WHERE idempotency_key = $1`, portion[i].Schluessel).Scan(&drin); err != nil {
			t.Fatalf("idempotency_keys: %v", err)
		}
		portion[i].SchluesselBekannt = drin > 0 // wie der Handler
		erg, err := w.svc.Nachbuchen(ctx, portion[i])
		if err != nil {
			t.Fatalf("Wiederholung: %v", err)
		}
		t.Logf("Wiederholung: %s → %q (Schlüssel in idempotency_keys: %d)", portion[i].Absicht, erg.Ergebnis, drin)
	}
	meldungen := w.meldungenDerSchluessel(t, schluessel...)
	if n, _ := w.offeneAusleihen(t); n != 0 {
		t.Errorf("Wiederholung hat gebucht: %d offene Ausleihen", n)
	}
	if len(meldungen) > 0 {
		t.Errorf("NACHGESTELLT: %d Meldungen für eine vollständig gebuchte Portion: %v", len(meldungen), meldungen)
	}
}

// litteraEtikett baut den EAN-13 eines Littera-Etiketts (internal/littera.EtikettBarcode,
// Bibliotheksnummer 395) — die Umkehrung von dekodiereLitteraEtikett.
func litteraEtikett(nummer string) string {
	payload := nummer + strings.Repeat("0", 8-len(nummer)) + "395" + strconv.Itoa(len(nummer))
	summe := 0
	for i := 0; i < 12; i++ {
		z := int(payload[i] - '0')
		if i%2 == 1 {
			z *= 3
		}
		summe += z
	}
	return payload + strconv.Itoa((10-summe%10)%10)
}

// Verdacht B (zwei): Die Barcode-Liste der Theke gegen das, was der Server beim Nachbuchen
// als Buch erkennt — ein Littera-Etikett auf einem Exemplar mit nackter Mediennummer, und
// ein ausgesondertes Exemplar mit Ziffern-Barcode.
func TestRaster_BarcodeListeGegenNachbuchTuer(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	svc := w.svc.(*defaultLoanService)
	nanos := time.Now().UnixNano()

	nummer := fmt.Sprintf("%d", 1000000+nanos%8999999) // 7 Stellen, erste ≠ 0
	etikett := litteraEtikett(nummer)
	if n, ok := dekodiereLitteraEtikett(etikett); !ok || n != nummer {
		t.Fatalf("Etikett-Formel: %s → %q %v", etikett, n, ok)
	}
	var litteraID string
	if err := w.pool.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, einkaufspreis)
		VALUES ($1, $2, true, 9.00) RETURNING id`, w.titelID, nummer).Scan(&litteraID); err != nil {
		t.Fatalf("Exemplar mit Mediennummer: %v", err)
	}
	gefunden, err := svc.loeseExemplar(ctx, etikett)
	if err != nil || gefunden == nil || gefunden.ID != litteraID {
		t.Fatalf("der Server erkennt das Etikett nicht: %v %v", gefunden, err)
	}

	abgeschrieben := fmt.Sprintf("%d", 20000000+nanos%79999999) // 8 Stellen
	if _, err := w.pool.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, ist_ausgesondert, aussonderung_grund, einkaufspreis)
		VALUES ($1, $2, false, true, 'VERLUST', 9.00)`, w.titelID, abgeschrieben); err != nil {
		t.Fatalf("ausgesondertes Exemplar: %v", err)
	}

	liste, err := repository.ListeBuchbarcodes(ctx, w.pool, 0)
	if err != nil {
		t.Fatalf("Liste: %v", err)
	}
	auf := map[string]bool{}
	for _, b := range liste {
		auf[b] = true
	}
	erg, err := w.svc.Nachbuchen(ctx, NachbuchEintrag{
		Schluessel: uuid.NewString(), Absicht: NachbuchAbsichtRueckgabe, Barcode: abgeschrieben,
		GescanntAm: time.Now(), StaffID: w.staff,
	})
	if err != nil {
		t.Fatalf("nachbuchen: %v", err)
	}
	t.Logf("Etikett %s auf der Liste: %v (Mediennummer %s: %v) · ausgesondert %s auf der Liste: %v, Nachbuchen → %q",
		etikett, auf[etikett], nummer, auf[nummer], abgeschrieben, auf[abgeschrieben], erg.Ergebnis)
	if !auf[etikett] {
		t.Errorf("NACHGESTELLT: Das gescannte Etikett %s fehlt in der Liste (sie führt %s) — der Server nimmt es als Buch, die Theke hielte es offline für unklar", etikett, nummer)
	}
	if erg.Ergebnis == repository.NachbuchNurReaktiviert && !auf[abgeschrieben] {
		t.Errorf("NACHGESTELLT: Das Nachbuchen holt das ausgesonderte Exemplar %s zurück, die Liste kennt es nicht", abgeschrieben)
	}
}

// Gegenprobe zur Uhr: dieselbe Lage mit richtig gehender Theken-Uhr — der Wächter muss greifen.
// Bleibt diese Probe grün, ist der Uhrversatz (und nichts anderes) die Ursache.
func TestRaster_Gegenprobe_RichtigeUhr(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	w.onlineAusleihe(t, w.anna)
	echteRueckgabe := time.Now()
	time.Sleep(50 * time.Millisecond)
	benLoan := w.onlineAusleihe(t, w.ben)
	erg, err := w.svc.Nachbuchen(ctx, NachbuchEintrag{
		Schluessel: uuid.NewString(), Absicht: NachbuchAbsichtRueckgabe, Barcode: w.code,
		GescanntAm: echteRueckgabe, StaffID: w.staff,
	})
	if err != nil {
		t.Fatalf("nachbuchen: %v", err)
	}
	var benOffen bool
	if err := w.pool.QueryRow(ctx, `SELECT rueckgabe_am IS NULL FROM ausleihen WHERE id = $1`, benLoan).Scan(&benOffen); err != nil {
		t.Fatalf("Bens Ausleihe: %v", err)
	}
	t.Logf("richtige Uhr: Ergebnis %q · Bens Ausleihe offen: %v", erg.Ergebnis, benOffen)
	if erg.Ergebnis != repository.NachbuchVeraltet || !benOffen {
		t.Errorf("Gegenprobe: erwartet veraltet und Bens Ausleihe offen, bekam %q, offen=%v", erg.Ergebnis, benOffen)
	}
}

// Gegenprobe zum bekannten Schlüssel: derselbe Eintrag, Schlüssel unbekannt — der Wächter greift.
func TestRaster_Gegenprobe_UnbekannterSchluessel(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	scan := time.Now()
	w.onlineAusleihe(t, w.anna)
	time.Sleep(50 * time.Millisecond)
	w.onlineRueckgabe(t)
	erg, err := w.svc.Nachbuchen(ctx, NachbuchEintrag{
		Schluessel: uuid.NewString(), Absicht: NachbuchAbsichtAusleihe, Barcode: w.code, GescanntAm: scan,
		SchuelerID: &w.anna, StaffID: w.staff,
	})
	if err != nil {
		t.Fatalf("nachbuchen: %v", err)
	}
	n, _ := w.offeneAusleihen(t)
	t.Logf("unbekannter Schlüssel: Ergebnis %q · offene Ausleihen %d", erg.Ergebnis, n)
	if erg.Ergebnis != repository.NachbuchVeraltet || n != 0 {
		t.Errorf("Gegenprobe: erwartet veraltet ohne offene Ausleihe, bekam %q, %d offen", erg.Ergebnis, n)
	}
}

// Gegenprobe: bekannter Schlüssel, aber das Buch liegt inzwischen bei Ben — dann fängt
// check_return_date die Rücknahme ab (Rückgabe vor Bens Ausleihe). Grenzt Verdacht A auf den
// Fall „Buch inzwischen zurückgegeben" ein.
func TestRaster_Gegenprobe_BekannterSchluesselBuchBeiBen(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	schluessel := uuid.NewString()
	scan := time.Now()
	w.onlineAusleihe(t, w.anna)
	if _, err := w.pool.Exec(ctx, `INSERT INTO idempotency_keys (idempotency_key, response_data, status_code) VALUES ($1, '{"success": true}', 200)`, schluessel); err != nil {
		t.Fatalf("Idempotenz-Antwort: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	w.onlineAusleihe(t, w.ben)
	erg, err := w.svc.Nachbuchen(ctx, NachbuchEintrag{
		Schluessel: schluessel, Absicht: NachbuchAbsichtAusleihe, Barcode: w.code, GescanntAm: scan,
		SchuelerID: &w.anna, SchluesselBekannt: true, StaffID: w.staff,
	})
	if err != nil {
		t.Fatalf("nachbuchen: %v", err)
	}
	n, bei := w.offeneAusleihen(t)
	t.Logf("bekannt, Buch bei Ben: Ergebnis %q · %d offen bei %s (Ben %s)", erg.Ergebnis, n, bei, w.ben)
	if erg.Ergebnis != repository.NachbuchVeraltet || n != 1 || bei != w.ben {
		t.Errorf("Gegenprobe: erwartet veraltet und das Buch bei Ben, bekam %q, %d offen bei %s", erg.Ergebnis, n, bei)
	}
}

// Gegenprobe: der Fall, für den die Ausnahme gebaut ist. Annas Sitzung scannt online ein Buch,
// das auf Ben steht — der Server bucht NUR die Fremdrückgabe (handleForeignReturn) und
// speichert die Antwort; sie erreicht die Theke nicht, der Eintrag „Ausleihe an Anna" geht in
// die Warteschlange. Das Nachbuchen soll die Ausleihe nachholen, obwohl der Scan vor der
// Rücknahme liegt. Bleibt diese Probe grün, hat die Ausnahme einen echten Zweck — eine
// Korrektur muss diesen Fall vom Fall „vollständig gebucht, danach zurückgegeben" trennen.
func TestRaster_Gegenprobe_BekannterSchluesselNachFremdrueckgabe(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	w.onlineAusleihe(t, w.ben)
	schluessel := uuid.NewString()
	scan := time.Now()
	time.Sleep(50 * time.Millisecond)
	w.onlineFremdrueckgabe(t)
	if _, err := w.pool.Exec(ctx, `INSERT INTO idempotency_keys (idempotency_key, response_data, status_code) VALUES ($1, '{"type": "rueckgabe", "fremdrueckgabe": true}', 200)`, schluessel); err != nil {
		t.Fatalf("Idempotenz-Antwort: %v", err)
	}
	erg, err := w.svc.Nachbuchen(ctx, NachbuchEintrag{
		Schluessel: schluessel, Absicht: NachbuchAbsichtAusleihe, Barcode: w.code, GescanntAm: scan,
		SchuelerID: &w.anna, SchluesselBekannt: true, StaffID: w.staff,
	})
	if err != nil {
		t.Fatalf("nachbuchen: %v", err)
	}
	n, bei := w.offeneAusleihen(t)
	t.Logf("bekannt nach Fremdrückgabe: Ergebnis %q · %d offen bei %s (Anna %s)", erg.Ergebnis, n, bei, w.anna)
	if erg.Ergebnis != repository.NachbuchAusgeliehen || n != 1 || bei != w.anna {
		t.Errorf("Gegenprobe: erwartet ausgeliehen an Anna, bekam %q, %d offen bei %s", erg.Ergebnis, n, bei)
	}
}
