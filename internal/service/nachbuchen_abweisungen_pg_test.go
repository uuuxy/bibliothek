package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"bibliothek/repository"
)

// Die Ausgänge der Nachbuch-Tür, die nichts oder nur einen Teil buchen, am echten Postgres:
// Rückgabe eines Buchs im Regal, unbekannte Absicht, Scan aus der Zukunft, offline gescannter
// Ausweis, Scan vor der Ausleihe, gleichzeitige Buchung an einem anderen Arbeitsplatz und der
// Grund, den die Rückkehr eines abgeschriebenen Buchs nennt.

func (w *nbWelt) meldungsGrund(t *testing.T, ergebnis string) string {
	t.Helper()
	var grund string
	if err := w.pool.QueryRow(context.Background(), `
		SELECT coalesce(grund, '') FROM nachbuch_meldungen WHERE barcode = $1 AND ergebnis = $2`, w.code, ergebnis).Scan(&grund); err != nil {
		t.Fatalf("Meldung %s lesen: %v", ergebnis, err)
	}
	return grund
}

func TestNachbuchen_RueckgabeEinesBuchsImRegal(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	erg, err := w.svc.Nachbuchen(ctx, w.eintrag(NachbuchAbsichtRueckgabe, nil, time.Now().Add(-time.Minute)))
	if err != nil || erg.Ergebnis != repository.NachbuchNichtGebucht || erg.Grund != "Buch war nicht ausgeliehen" {
		t.Fatalf("Rückgabe im Regal: %v %+v — erwartet nicht_gebucht mit dem Grund", err, erg)
	}
	if got := w.meldungsGrund(t, repository.NachbuchNichtGebucht); got != "Buch war nicht ausgeliehen" {
		t.Errorf("Meldung nennt %q", got)
	}
	var gestempelt bool
	if err := w.pool.QueryRow(ctx, `SELECT letzte_bewegung_am IS NOT NULL FROM buecher_exemplare WHERE id = $1`, w.exemplarID).Scan(&gestempelt); err != nil {
		t.Fatalf("Exemplar lesen: %v", err)
	}
	if gestempelt {
		t.Errorf("das Exemplar trägt einen Bewegungsstempel, obwohl nichts gebucht wurde")
	}
}

func TestNachbuchen_UnbekannteAbsicht(t *testing.T) {
	w := nbAufbau(t)
	erg, err := w.svc.Nachbuchen(context.Background(), w.eintrag("verlaengern", &w.anna, time.Now()))
	if !errors.Is(err, ErrInvalidState) || erg != nil {
		t.Fatalf("unbekannte Absicht: %v %+v — erwartet ErrInvalidState ohne Ergebnis", err, erg)
	}
	if n, _ := w.offeneAusleihen(t); n != 0 {
		t.Errorf("%d offene Ausleihen nach einer unbekannten Absicht", n)
	}
	var meldungen int
	if err := w.pool.QueryRow(context.Background(), `SELECT count(*) FROM nachbuch_meldungen WHERE barcode = $1`, w.code).Scan(&meldungen); err != nil {
		t.Fatalf("Meldungen zählen: %v", err)
	}
	if meldungen != 0 {
		t.Errorf("%d Meldungen nach einer unbekannten Absicht", meldungen)
	}
}

// Die Uhr des Theken-Rechners geht drei Stunden vor: Gebucht wird höchstens zur Serverzeit.
func TestNachbuchen_ScanAusDerZukunftZaehltAbServerzeit(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	vorher := time.Now()
	erg, err := w.svc.Nachbuchen(ctx, w.eintrag(NachbuchAbsichtAusleihe, &w.anna, time.Now().Add(3*time.Hour)))
	if err != nil || erg.Ergebnis != repository.NachbuchAusgeliehen {
		t.Fatalf("Scan aus der Zukunft: %v %+v", err, erg)
	}
	var ausgeliehen time.Time
	if err := w.pool.QueryRow(ctx, `SELECT ausgeliehen_am FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`, w.exemplarID).Scan(&ausgeliehen); err != nil {
		t.Fatalf("Ausleihe lesen: %v", err)
	}
	if ausgeliehen.Before(vorher.Add(-time.Second)) || ausgeliehen.After(time.Now().Add(time.Second)) {
		t.Errorf("ausgeliehen_am %v, erwartet die Serverzeit um %v", ausgeliehen, vorher)
	}
}

// Der Rechner konnte die Person beim Scan nicht auflösen und schickt den Ausweis: Der Server
// löst ihn auf und leiht an diese Person aus.
func TestNachbuchen_OfflineGescannterAusweisLoestDiePersonAuf(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	var ausweis string
	if err := w.pool.QueryRow(ctx, `SELECT barcode_id FROM schueler WHERE id = $1`, w.anna).Scan(&ausweis); err != nil {
		t.Fatalf("Ausweis lesen: %v", err)
	}
	e := w.eintrag(NachbuchAbsichtAusleihe, nil, time.Now().Add(-time.Minute))
	e.AusweisBarcode = &ausweis
	erg, err := w.svc.Nachbuchen(ctx, e)
	if err != nil || erg.Ergebnis != repository.NachbuchAusgeliehen {
		t.Fatalf("Ausleihe über den Ausweis: %v %+v", err, erg)
	}
	if n, bei := w.offeneAusleihen(t); n != 1 || bei != w.anna {
		t.Errorf("%d offene Ausleihen, bei %s — erwartet 1 bei Anna (%s)", n, bei, w.anna)
	}
}

// Ein Scan, der vor dem Beginn der laufenden Ausleihe liegt, beschreibt eine Wirklichkeit, die
// es nicht mehr gibt — auch wenn das Exemplar keinen Bewegungsstempel trägt. Die Ausleihe
// bleibt, der Eintrag ist veraltet und gemeldet.
func TestNachbuchen_ScanVorDerAusleiheIstVeraltet(t *testing.T) {
	for _, f := range []struct {
		name, absicht, grund string
		person               func(w *nbWelt) *string
	}{
		{"Rückgabe", NachbuchAbsichtRueckgabe, "Rückgabe liegt vor der Ausleihe", func(*nbWelt) *string { return nil }},
		{"Ausleihe an ein anderes Kind", NachbuchAbsichtAusleihe, "Scan liegt vor der Ausleihe des Vorbesitzers", func(w *nbWelt) *string { return &w.ben }},
	} {
		t.Run(f.name, func(t *testing.T) {
			w := nbAufbau(t)
			ctx := context.Background()
			jetzt := time.Now()
			if _, err := w.pool.Exec(ctx, `
				INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, ausgeliehen_am, erfasst_am, rueckgabe_frist)
				VALUES ($1, $2, $3, $4::timestamptz, $4::timestamptz, $4::timestamptz + interval '14 days')`,
				w.exemplarID, w.anna, w.staff, jetzt.Add(-5*time.Minute)); err != nil {
				t.Fatalf("Ausleihe an Anna: %v", err)
			}
			if _, err := w.pool.Exec(ctx, `UPDATE buecher_exemplare SET letzte_bewegung_am = NULL WHERE id = $1`, w.exemplarID); err != nil {
				t.Fatalf("Stempel leeren: %v", err)
			}
			erg, err := w.svc.Nachbuchen(ctx, w.eintrag(f.absicht, f.person(w), jetzt.Add(-10*time.Minute)))
			if err != nil || erg.Ergebnis != repository.NachbuchVeraltet || erg.Grund != f.grund {
				t.Fatalf("%v %+v — erwartet veraltet mit dem Grund %q", err, erg, f.grund)
			}
			if n, bei := w.offeneAusleihen(t); n != 1 || bei != w.anna {
				t.Errorf("%d offene Ausleihen, bei %s — Annas Ausleihe muss bleiben", n, bei)
			}
			if got := w.meldungsGrund(t, repository.NachbuchVeraltet); got != f.grund {
				t.Errorf("Meldung nennt %q", got)
			}
		})
	}
}

// Legt ein anderer Arbeitsplatz die Ausleihe im selben Augenblick an, verwirft die Datenbank die
// zweite (ON CONFLICT, keine Zeile). Der Eintrag ist dann nicht gebucht und gemeldet; die
// Rücknahme beim Vorbesitzer bleibt.
func TestNachbuchen_GleichzeitigeAusleiheAmAnderenArbeitsplatz(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	t0 := time.Now().Add(-30 * time.Minute)
	if erg, err := w.svc.Nachbuchen(ctx, w.eintrag(NachbuchAbsichtAusleihe, &w.ben, t0)); err != nil || erg.Ergebnis != repository.NachbuchAusgeliehen {
		t.Fatalf("Ben: %v %+v", err, erg)
	}

	// Die Datenbank verwirft jede neue Ausleihe an Anna — wie der Unique-Index, wenn ein
	// anderer Arbeitsplatz schneller war.
	if _, err := w.pool.Exec(ctx, fmt.Sprintf(`
		CREATE OR REPLACE FUNCTION test_verwirft_ausleihe() RETURNS trigger AS $$
		BEGIN
			IF NEW.schueler_id = '%s'::uuid THEN
				RETURN NULL;
			END IF;
			RETURN NEW;
		END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER test_verwirft_ausleihe BEFORE INSERT ON ausleihen
		FOR EACH ROW EXECUTE FUNCTION test_verwirft_ausleihe();`, w.anna)); err != nil {
		t.Fatalf("Trigger anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := w.pool.Exec(context.Background(), `
			DROP TRIGGER IF EXISTS test_verwirft_ausleihe ON ausleihen;
			DROP FUNCTION IF EXISTS test_verwirft_ausleihe();`); err != nil {
			t.Errorf("Trigger entfernen: %v", err)
		}
	})

	scan := t0.Add(5 * time.Minute)
	erg, err := w.svc.Nachbuchen(ctx, w.eintrag(NachbuchAbsichtAusleihe, &w.anna, scan))
	const grund = "Exemplar wurde soeben an einem anderen Arbeitsplatz verbucht"
	if err != nil || erg.Ergebnis != repository.NachbuchNichtGebucht || erg.Grund != grund {
		t.Fatalf("%v %+v — erwartet nicht_gebucht mit dem Grund %q", err, erg, grund)
	}
	if n, _ := w.offeneAusleihen(t); n != 0 {
		t.Errorf("%d offene Ausleihen — die Rücknahme bei Ben muss bleiben, die Ausleihe an Anna nicht entstehen", n)
	}
	var zurueck time.Time
	var fremd bool
	if err := w.pool.QueryRow(ctx, `
		SELECT rueckgabe_am, ist_fremdrueckgabe FROM ausleihen WHERE exemplar_id = $1 AND schueler_id = $2`, w.exemplarID, w.ben).Scan(&zurueck, &fremd); err != nil {
		t.Fatalf("Bens Ausleihe lesen: %v", err)
	}
	if d := zurueck.Sub(scan); d < -time.Second || d > time.Second || !fremd {
		t.Errorf("Bens Ausleihe: zurück am %v (fremd %v) — erwartet zum Scan %v als Fremdrückgabe", zurueck, fremd, scan)
	}
	if got := w.meldungsGrund(t, repository.NachbuchNichtGebucht); got != grund {
		t.Errorf("Meldung nennt %q", got)
	}
}

// Kommt ein abgeschriebenes Buch über eine Rückgabe zurück, nennt die Meldung, was mit ihm
// geschah: wieder im Umlauf, und mit Betrag, wenn dabei eine Forderung endete.
func TestNachbuchen_RueckkehrNenntDenGrund(t *testing.T) {
	ctx := context.Background()

	ohne := nbAufbau(t)
	if _, err := ohne.pool.Exec(ctx, `UPDATE buecher_exemplare SET ist_ausleihbar = false, ist_ausgesondert = true, aussonderung_grund = 'VERLUST' WHERE id = $1`, ohne.exemplarID); err != nil {
		t.Fatalf("aussondern: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	erg, err := ohne.svc.Nachbuchen(ctx, ohne.eintrag(NachbuchAbsichtRueckgabe, nil, time.Now()))
	const umlauf = "Buch war abgeschrieben und ist wieder im Umlauf"
	if err != nil || erg.Ergebnis != repository.NachbuchNurReaktiviert || erg.Grund != umlauf {
		t.Fatalf("ohne Forderung: %v %+v — erwartet nur_reaktiviert mit dem Grund %q", err, erg, umlauf)
	}
	if got := ohne.meldungsGrund(t, repository.NachbuchNurReaktiviert); got != umlauf {
		t.Errorf("ohne Forderung: Meldung nennt %q", got)
	}

	mit := nbAufbau(t)
	verlorenGemeldet(t, mit)
	time.Sleep(50 * time.Millisecond)
	erg, err = mit.svc.Nachbuchen(ctx, mit.eintrag(NachbuchAbsichtRueckgabe, nil, time.Now()))
	const storniert = "Buch war abgeschrieben; Forderung über 12.00 € storniert"
	if err != nil || erg.Ergebnis != repository.NachbuchNurReaktiviert || erg.Grund != storniert {
		t.Fatalf("mit Forderung: %v %+v — erwartet nur_reaktiviert mit dem Grund %q", err, erg, storniert)
	}
	if got := mit.meldungsGrund(t, repository.NachbuchNurReaktiviert); got != storniert {
		t.Errorf("mit Forderung: Meldung nennt %q", got)
	}
}

// Eine Ausleihe ohne Person wird abgewiesen, und der Grund nennt den gescannten Ausweis, wenn
// es einen gab.
func TestNachbuchen_AusleiheOhnePersonNenntDenAusweis(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	erg, err := w.svc.Nachbuchen(ctx, w.eintrag(NachbuchAbsichtAusleihe, nil, time.Now().Add(-time.Minute)))
	if err != nil || erg.Ergebnis != repository.NachbuchNichtGebucht || erg.Grund != "Ausweis unbekannt" {
		t.Fatalf("ohne Person und ohne Ausweis: %v %+v", err, erg)
	}
	ausweis := "S-FREMD-4711"
	e := w.eintrag(NachbuchAbsichtAusleihe, nil, time.Now().Add(-time.Minute))
	e.AusweisBarcode = &ausweis
	erg, err = w.svc.Nachbuchen(ctx, e)
	if err != nil || erg.Ergebnis != repository.NachbuchNichtGebucht || erg.Grund != "Ausweis unbekannt: S-FREMD-4711" {
		t.Fatalf("mit unbekanntem Ausweis: %v %+v", err, erg)
	}
	if n, _ := w.offeneAusleihen(t); n != 0 {
		t.Errorf("%d offene Ausleihen nach zwei Abweisungen", n)
	}
}

func (w *nbWelt) exemplarStand(t *testing.T) (ausleihbar, ausgesondert bool, bewegt *time.Time) {
	t.Helper()
	if err := w.pool.QueryRow(context.Background(), `
		SELECT ist_ausleihbar, ist_ausgesondert, letzte_bewegung_am FROM buecher_exemplare WHERE id = $1`, w.exemplarID).Scan(&ausleihbar, &ausgesondert, &bewegt); err != nil {
		t.Fatalf("Exemplar lesen: %v", err)
	}
	return ausleihbar, ausgesondert, bewegt
}

// Ein gesperrtes Exemplar (nicht ausleihbar, nicht ausgesondert), das im Regal steht, kommt mit
// dem Scan zurück in den Umlauf. Ist es verliehen, endet mit der Rückgabe nur die Ausleihe —
// die Sperre am Exemplar bleibt.
func TestNachbuchen_GesperrtesExemplar(t *testing.T) {
	ctx := context.Background()

	imRegal := nbAufbau(t)
	if _, err := imRegal.pool.Exec(ctx, `UPDATE buecher_exemplare SET ist_ausleihbar = false WHERE id = $1`, imRegal.exemplarID); err != nil {
		t.Fatalf("sperren: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	erg, err := imRegal.svc.Nachbuchen(ctx, imRegal.eintrag(NachbuchAbsichtRueckgabe, nil, time.Now()))
	if err != nil || erg.Ergebnis != repository.NachbuchNurReaktiviert {
		t.Fatalf("gesperrt im Regal: %v %+v — erwartet nur_reaktiviert", err, erg)
	}
	if ausleihbar, _, _ := imRegal.exemplarStand(t); !ausleihbar {
		t.Errorf("gesperrt im Regal: das Exemplar ist nach dem Scan nicht ausleihbar")
	}

	verliehen := nbAufbau(t)
	if erg, err := verliehen.svc.Nachbuchen(ctx, verliehen.eintrag(NachbuchAbsichtAusleihe, &verliehen.anna, time.Now().Add(-30*time.Minute))); err != nil || erg.Ergebnis != repository.NachbuchAusgeliehen {
		t.Fatalf("Anna: %v %+v", err, erg)
	}
	if _, err := verliehen.pool.Exec(ctx, `UPDATE buecher_exemplare SET ist_ausleihbar = false WHERE id = $1`, verliehen.exemplarID); err != nil {
		t.Fatalf("sperren: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	erg, err = verliehen.svc.Nachbuchen(ctx, verliehen.eintrag(NachbuchAbsichtRueckgabe, nil, time.Now()))
	if err != nil || erg.Ergebnis != repository.NachbuchZurueckgegeben {
		t.Fatalf("gesperrt und verliehen: %v %+v — erwartet zurueckgegeben", err, erg)
	}
	if n, _ := verliehen.offeneAusleihen(t); n != 0 {
		t.Errorf("gesperrt und verliehen: %d offene Ausleihen nach der Rückgabe", n)
	}
	if ausleihbar, _, _ := verliehen.exemplarStand(t); ausleihbar {
		t.Errorf("gesperrt und verliehen: die Rückgabe hat die Sperre am Exemplar aufgehoben")
	}
}

// Das Rückholen eines abgeschriebenen Buchs zählt ab dem Scan: Der Bewegungsstempel trägt den
// Scan-Zeitpunkt, nicht die Zeit des Nachbuchens. Kommt es über eine Ausleihe zurück, ist es
// in derselben Buchung ausgeliehen, und die Antwort zeigt es im Umlauf.
func TestNachbuchen_RueckholenZaehltAbDemScan(t *testing.T) {
	ctx := context.Background()

	w := nbAufbau(t)
	verlorenGemeldet(t, w)
	jetzt := time.Now()
	if _, err := w.pool.Exec(ctx, `UPDATE buecher_exemplare SET letzte_bewegung_am = $2 WHERE id = $1`, w.exemplarID, jetzt.Add(-10*time.Minute)); err != nil {
		t.Fatalf("Stempel setzen: %v", err)
	}
	scan := jetzt.Add(-5 * time.Minute)
	erg, err := w.svc.Nachbuchen(ctx, w.eintrag(NachbuchAbsichtRueckgabe, nil, scan))
	if err != nil || erg.Ergebnis != repository.NachbuchNurReaktiviert {
		t.Fatalf("Rückkehr: %v %+v", err, erg)
	}
	_, _, bewegt := w.exemplarStand(t)
	if bewegt == nil {
		t.Fatal("kein Bewegungsstempel nach dem Rückholen")
	}
	if d := bewegt.Sub(scan); d < -time.Second || d > time.Second {
		t.Errorf("Bewegungsstempel %v, erwartet den Scan-Zeitpunkt %v", *bewegt, scan)
	}

	a := nbAufbau(t)
	verlorenGemeldet(t, a)
	time.Sleep(50 * time.Millisecond)
	erg, err = a.svc.Nachbuchen(ctx, a.eintrag(NachbuchAbsichtAusleihe, &a.ben, time.Now()))
	if err != nil || erg.Ergebnis != repository.NachbuchAusgeliehen {
		t.Fatalf("Ausleihe eines abgeschriebenen Buchs: %v %+v", err, erg)
	}
	if n, bei := a.offeneAusleihen(t); n != 1 || bei != a.ben {
		t.Errorf("%d offene Ausleihen, bei %s — erwartet 1 bei Ben", n, bei)
	}
	if ausleihbar, ausgesondert, _ := a.exemplarStand(t); !ausleihbar || ausgesondert {
		t.Errorf("Exemplar nach dem Rückholen: ausleihbar %v, ausgesondert %v", ausleihbar, ausgesondert)
	}
	if b := erg.Result.Book; b == nil || !b.IstAusleihbar || b.IstAusgesondert || b.ZustandNotiz != "" {
		t.Errorf("die Antwort zeigt das Exemplar nicht im Umlauf: %+v", b)
	}
}

// Die Antwort einer gebuchten Ausleihe nennt die neue Ausleihe und ihre Frist — daran hängt an
// der Theke das Rückgängigmachen.
func TestNachbuchen_AntwortNenntAusleiheUndFrist(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	erg, err := w.svc.Nachbuchen(ctx, w.eintrag(NachbuchAbsichtAusleihe, &w.anna, time.Now().Add(-time.Minute)))
	if err != nil || erg.Ergebnis != repository.NachbuchAusgeliehen || erg.Result == nil {
		t.Fatalf("Ausleihe: %v %+v", err, erg)
	}
	var id string
	var frist time.Time
	if err := w.pool.QueryRow(ctx, `SELECT id::text, rueckgabe_frist FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`, w.exemplarID).Scan(&id, &frist); err != nil {
		t.Fatalf("Ausleihe lesen: %v", err)
	}
	if erg.Result.LoanID == nil || *erg.Result.LoanID != id {
		t.Errorf("die Antwort nennt die Ausleihe %v, erwartet %s", erg.Result.LoanID, id)
	}
	if erg.Result.DueDate == nil || !erg.Result.DueDate.Equal(frist) {
		t.Errorf("die Antwort nennt die Frist %v, erwartet %v", erg.Result.DueDate, frist)
	}
}

// Die Schranken des Online-Scans gelten auch beim Nachbuchen, und die Meldung nennt, woran es
// lag: an der Sperre des Lesers oder an der Vormerkung eines anderen. Beides ist eine
// Abweisung, keine Störung.
func TestNachbuchen_AbweisungNenntDieSchranke(t *testing.T) {
	ctx := context.Background()

	sperre := nbAufbau(t)
	erg, err := sperre.svc.Nachbuchen(ctx, sperre.eintrag(NachbuchAbsichtAusleihe, &sperre.carla, time.Now().Add(-time.Minute)))
	if err != nil || erg.Ergebnis != repository.NachbuchNichtGebucht {
		t.Fatalf("gesperrter Leser: %v %+v", err, erg)
	}
	if !strings.Contains(erg.Grund, "gesperrt") {
		t.Errorf("gesperrter Leser: der Grund %q nennt die Sperre nicht", erg.Grund)
	}
	if got := sperre.meldungsGrund(t, repository.NachbuchNichtGebucht); got != erg.Grund || got == "" {
		t.Errorf("Meldung nennt %q, die Antwort %q", got, erg.Grund)
	}

	vorgemerkt := nbAufbau(t)
	if _, err := vorgemerkt.pool.Exec(ctx, `
		INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis)
		VALUES ($1, $2, 'abholbereit', $3, CURRENT_TIMESTAMP + interval '2 days')`,
		vorgemerkt.titelID, vorgemerkt.ben, vorgemerkt.exemplarID); err != nil {
		t.Fatalf("Vormerkung für Ben: %v", err)
	}
	erg, err = vorgemerkt.svc.Nachbuchen(ctx, vorgemerkt.eintrag(NachbuchAbsichtAusleihe, &vorgemerkt.anna, time.Now().Add(-time.Minute)))
	const reserviert = "Achtung: dieses Exemplar ist noch für Ben Zweiter reserviert"
	if err != nil || erg.Ergebnis != repository.NachbuchNichtGebucht || erg.Grund != reserviert {
		t.Fatalf("fremde Vormerkung: %v %+v — erwartet nicht_gebucht mit dem Grund %q", err, erg, reserviert)
	}
	if n, _ := vorgemerkt.offeneAusleihen(t); n != 0 {
		t.Errorf("%d offene Ausleihen trotz fremder Vormerkung", n)
	}
	if got := vorgemerkt.meldungsGrund(t, repository.NachbuchNichtGebucht); got != reserviert {
		t.Errorf("Meldung nennt %q", got)
	}
}
