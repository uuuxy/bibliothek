package api

// Der eine Prüfweg für Buch und Gerät am Live-Pfad der Theke (service.pruefeAusleihSperren,
// entschieden am 24.09.2026): das Buch über HandleUnifiedCheckout, das Gerät über die
// Omnibox wie beim Scan (ProcessQuery mit „G-"). Jeder Test nennt die Probe, an der er rot
// wurde.

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/service"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// sperrTheke ist die Theke eines Tests: Buch- und Geräte-Pfad über echte Dienste.
type sperrTheke struct {
	pool       *pgxpool.Pool
	books      repository.BookRepository
	loanSvc    service.LoanService
	omnibox    service.OmniboxService
	bearbeiter string
}

func neueSperrTheke(t *testing.T) *sperrTheke {
	t.Helper()
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	// Geräte räumt resetBestandsdaten nicht weg.
	aufraeumen := func() {
		for _, sql := range []string{
			`DELETE FROM ausleihen WHERE geraet_id IN (SELECT id FROM geraete WHERE barcode_id LIKE 'G-SPERRWEG-%')`,
			`DELETE FROM geraete WHERE barcode_id LIKE 'G-SPERRWEG-%'`,
		} {
			if _, err := pool.Exec(context.Background(), sql); err != nil {
				t.Errorf("Geräte aufräumen: %v", err)
			}
		}
	}
	aufraeumen()
	t.Cleanup(aufraeumen)

	students := repository.NewStudentRepository(pool)
	books := repository.NewBookRepository(pool)
	loans := repository.NewLoanRepository(pool)
	audit := repository.NewAuditRepository(pool)
	loanSvc := service.NewLoanService(pool, students, books, loans, audit)
	deviceSvc := service.NewDeviceService(pool, students, loans, audit)
	return &sperrTheke{
		pool:       pool,
		books:      books,
		loanSvc:    loanSvc,
		omnibox:    service.NewOmniboxService(pool, students, books, repository.NewUserRepository(pool), loans, loanSvc, deviceSvc),
		bearbeiter: adminFuerAudit(t, pool),
	}
}

// buch leiht das Exemplar barcode an leserID aus, wie der Scan an der Theke.
func (th *sperrTheke) buch(t *testing.T, barcode, leserID string, uebergehen bool) error {
	t.Helper()
	ex, err := th.books.GetCopyByBarcode(context.Background(), barcode)
	if err != nil {
		t.Fatalf("Exemplar %s laden: %v", barcode, err)
	}
	_, err = th.loanSvc.HandleUnifiedCheckout(context.Background(), ex, &leserID, th.bearbeiter, uebergehen)
	return err
}

// geraet scannt das Gerät an der Theke — über die Omnibox, denn dort kommt override_block an.
func (th *sperrTheke) geraet(t *testing.T, barcode, leserID string, bestaetigt, uebergehen bool) (*service.OmniboxResult, error) {
	t.Helper()
	return th.omnibox.ProcessQuery(context.Background(), service.OmniboxQuery{
		Query: barcode, ActiveLeserID: &leserID, ConfirmedChecklist: bestaetigt,
		StaffID: th.bearbeiter, OverrideBlock: uebergehen,
	})
}

func (th *sperrTheke) neuesGeraet(t *testing.T, barcode, zubehoer string) {
	t.Helper()
	if _, err := th.pool.Exec(context.Background(),
		`INSERT INTO geraete (modellname, barcode_id, zubehoer) VALUES ('Tablet', $1, $2)`, barcode, zubehoer); err != nil {
		t.Fatalf("Gerät anlegen: %v", err)
	}
}

// offen zählt die laufenden Ausleihen eines Exemplars oder Geräts.
func (th *sperrTheke) offen(t *testing.T, barcode string) int {
	t.Helper()
	var n int
	if err := th.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM ausleihen a
		LEFT JOIN buecher_exemplare e ON e.id = a.exemplar_id
		LEFT JOIN geraete g ON g.id = a.geraet_id
		WHERE a.rueckgabe_am IS NULL AND (e.barcode_id = $1 OR g.barcode_id = $1)`, barcode).Scan(&n); err != nil {
		t.Fatalf("offene Ausleihen zählen: %v", err)
	}
	return n
}

// uebergangen zählt die Protokolleinträge „Sperre übergangen" eines Lesers.
func (th *sperrTheke) uebergangen(t *testing.T, leserID string) int {
	t.Helper()
	var n int
	if err := th.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_logs WHERE aktion = 'OVERRIDE_BLOCK' AND details->>'schueler_id' = $1`,
		leserID).Scan(&n); err != nil {
		t.Fatalf("Protokoll zählen: %v", err)
	}
	return n
}

// bestand legt je ein Buch der Schülerbücherei und ein Schulbuch an, beide ausleihbar.
func (th *sperrTheke) bestand(t *testing.T, praefix string) (buecherei, schulbuch string) {
	t.Helper()
	roman := titelMitMeldebestand(t, th.pool, "Roman "+praefix, 1)
	var lmf string
	if err := th.pool.QueryRow(context.Background(),
		`INSERT INTO buecher_titel (titel, ist_lernmittel) VALUES ($1, true) RETURNING id`, "Mathe "+praefix).Scan(&lmf); err != nil {
		t.Fatalf("Lernmittel-Titel anlegen: %v", err)
	}
	buecherei, schulbuch = "B-"+praefix+"-R", "B-"+praefix+"-L"
	exemplar(t, th.pool, roman, buecherei, true, "")
	exemplar(t, th.pool, lmf, schulbuch, true, "")
	return buecherei, schulbuch
}

func istSperreAmLeser(err error) bool {
	return errors.Is(err, service.ErrBlocked) && service.IstSperreAmLeser(err) && !service.IstUebergehbareSperre(err)
}

// Eine Sperre von Hand lässt an der Theke nur die Rückgabe zu — beim Buch, beim Schulbuch
// und beim Gerät, auch mit override_block; kein Protokolleintrag, denn nichts wurde
// übergangen. Wie Littera: aufheben in den Leserdaten.
// Rot gesehen am Rückbau: pruefeAusleihSperren lässt die Sperre am Leser mit override_block
// durch, wie bis zum 24.09.2026 am Buch — das Buch geht raus.
func TestTheke_SperreVonHandLaesstNurRueckgabeZu(t *testing.T) {
	th := neueSperrTheke(t)
	buecherei, schulbuch := th.bestand(t, "SVH")
	kind := schuelerAnlegen(t, th.pool, "Handsperre", "07A", "S-SPERRWEG-1")
	if _, err := th.pool.Exec(context.Background(),
		`UPDATE schueler SET is_manually_blocked = true, block_reason = 'Ausweis verloren' WHERE id = $1`, kind); err != nil {
		t.Fatalf("sperren: %v", err)
	}
	th.neuesGeraet(t, "G-SPERRWEG-1", "")

	if err := th.buch(t, buecherei, kind, true); !istSperreAmLeser(err) {
		t.Errorf("Buch der Bücherei mit override_block: erwartet Sperre am Leser, bekam %v", err)
	}
	if err := th.buch(t, schulbuch, kind, true); !istSperreAmLeser(err) {
		t.Errorf("Schulbuch: Die Sperre von Hand gilt auch dort, bekam %v", err)
	}
	if _, err := th.geraet(t, "G-SPERRWEG-1", kind, true, true); !istSperreAmLeser(err) {
		t.Errorf("Gerät mit override_block: erwartet Sperre am Leser, bekam %v", err)
	}
	for _, b := range []string{buecherei, schulbuch, "G-SPERRWEG-1"} {
		if n := th.offen(t, b); n != 0 {
			t.Errorf("%s ging trotz Sperre raus (%d offen)", b, n)
		}
	}
	if n := th.uebergangen(t, kind); n != 0 {
		t.Errorf("nichts wurde übergangen, trotzdem %d Protokolleinträge", n)
	}
}

// Die Sperre, die das Programm den Ehemaligen setzt, hält Bücherei und Gerät auf wie die von
// Hand. Beim Schulbuch zählt sie nicht: Die Schule schließt jede automatische Sperre am
// Lernmittel aus (22.09.2026) — das Kind der 10R, das in die E-Phase geht, bekommt seine
// Schulbücher, ohne dass jemand etwas übergehen muss.
// Rot gesehen am Rückbau: `&& !lernmittel` aus pruefeSperreAmLeser entfernt — das Schulbuch
// wird abgewiesen.
func TestTheke_EhemaligeSperreNichtAmSchulbuch(t *testing.T) {
	th := neueSperrTheke(t)
	buecherei, schulbuch := th.bestand(t, "EHM")
	kind := schuelerAnlegen(t, th.pool, "Ehemalig", "10R1", "S-SPERRWEG-2")
	if _, err := th.pool.Exec(context.Background(), `UPDATE schueler SET ist_abgaenger = true, ist_gesperrt = true,
		block_reason = 'Automatisierte Abgänger-Sperre (Schuljahreswechsel)' WHERE id = $1`, kind); err != nil {
		t.Fatalf("Ehemaligen sperren: %v", err)
	}
	th.neuesGeraet(t, "G-SPERRWEG-2", "")

	if err := th.buch(t, schulbuch, kind, false); err != nil {
		t.Fatalf("Schulbuch an den Ehemaligen abgewiesen: %v", err)
	}
	if th.offen(t, schulbuch) != 1 {
		t.Error("das Schulbuch steht nicht als Ausleihe da")
	}
	if err := th.buch(t, buecherei, kind, true); !istSperreAmLeser(err) {
		t.Errorf("Buch der Bücherei mit override_block: erwartet Sperre am Leser, bekam %v", err)
	}
	if _, err := th.geraet(t, "G-SPERRWEG-2", kind, true, true); !istSperreAmLeser(err) {
		t.Errorf("Gerät mit override_block: erwartet Sperre am Leser, bekam %v", err)
	}
	if n := th.uebergangen(t, kind); n != 0 {
		t.Errorf("nichts wurde übergangen, trotzdem %d Protokolleinträge", n)
	}
}

// Ein anonymisierter Datensatz bekommt nichts, auch kein Schulbuch — die Namenssuche der
// Theke findet ihn noch („Abgänger Anonymisiert-…"). Die Sperre trägt kein Merkmal: Aufheben
// kann man sie nicht (student_lock_pg_test.go).
// Rot gesehen am Rückbau: die Prüfung auf IstAnonymisiert entfernt — das Schulbuch geht an
// den anonymisierten Datensatz.
func TestTheke_AnonymisiertBekommtKeinSchulbuch(t *testing.T) {
	th := neueSperrTheke(t)
	_, schulbuch := th.bestand(t, "ANO")
	anonym := schuelerAnlegen(t, th.pool, "Anonym", "10R1", "S-SPERRWEG-3")
	if _, err := th.pool.Exec(context.Background(), `UPDATE schueler SET ist_abgaenger = true, ist_gesperrt = true,
		block_reason = 'Abgänger anonymisiert', anonymized_at = NOW() WHERE id = $1`, anonym); err != nil {
		t.Fatalf("anonymisieren: %v", err)
	}

	err := th.buch(t, schulbuch, anonym, true)
	if !errors.Is(err, service.ErrBlocked) || service.IstSperreAmLeser(err) || service.IstUebergehbareSperre(err) {
		t.Errorf("erwartet Sperre ohne Merkmal, bekam %v", err)
	}
	if th.offen(t, schulbuch) != 0 {
		t.Error("das Schulbuch ging an einen anonymisierten Datensatz")
	}
}

// Ein Kollege wird nie gesperrt (16.09. und 24.09.2026) — auch nicht mit einer Sperre von
// Hand, die der Knopf bis zum 24.09.2026 setzen konnte, und nicht mit einer Forderung von vor
// dem 16.09.2026. Am Buch galt das schon, am Gerät hielt ihn bis dahin die eigene Prüfung auf.
// Rot gesehen am Rückbau: die Prüfung auf istKollegium entfernt — Buch und Gerät abgewiesen.
func TestTheke_KollegeWirdNieGesperrt(t *testing.T) {
	th := neueSperrTheke(t)
	buecherei, _ := th.bestand(t, "KOL")
	var kollege string
	if err := th.pool.QueryRow(context.Background(), `
		INSERT INTO leser (barcode_id, vorname, nachname, art, is_manually_blocked, block_reason)
		VALUES ('A-SPERRWEG-K', 'Kollege', 'Gesperrt', 'lehrkraft', true, 'Altlast') RETURNING id`).Scan(&kollege); err != nil {
		t.Fatalf("Kollegen anlegen: %v", err)
	}
	altesBuch := titelMitMeldebestand(t, th.pool, "Altes Buch KOL", 1)
	verloren := exemplar(t, th.pool, altesBuch, "B-KOL-ALT", false, "")
	if _, err := th.pool.Exec(context.Background(), `INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, art)
		VALUES ($1, $2, 'Verloren', 12.00, 'nicht_zurueckgegeben')`, verloren, kollege); err != nil {
		t.Fatalf("alte Forderung anlegen: %v", err)
	}
	th.neuesGeraet(t, "G-SPERRWEG-K", "")

	if err := th.buch(t, buecherei, kollege, false); err != nil {
		t.Errorf("Buch an den Kollegen abgewiesen: %v", err)
	}
	if _, err := th.geraet(t, "G-SPERRWEG-K", kollege, true, false); err != nil {
		t.Errorf("Gerät an den Kollegen abgewiesen: %v", err)
	}
	if th.offen(t, buecherei) != 1 || th.offen(t, "G-SPERRWEG-K") != 1 {
		t.Error("Buch oder Gerät stehen nicht als Ausleihe da")
	}
}

// Eine offene Forderung zählt aus jedem Topf — auch die für ein verlorenes Schulbuch (Topf
// des Landes) hält Bücherei und Gerät auf, ein Konto je Leser wie in Littera. Sie ist ein
// Hinweis: override_block übergeht sie, am Buch wie seit dem 24.09.2026 am Gerät, und das
// Protokoll hält es fest — erst wenn die Ausleihe steht. Die Zubehör-Liste liegt beim Gerät
// zwischen Prüfung und Ausleihe; sie schreibt nichts.
// Rot gesehen am Rückbau: ladeAkteur gibt override_block nicht an pruefeAusleihSperren weiter
// — das Gerät bleibt gesperrt; ebenso, wenn die Omnibox es nicht ans Gerät gibt;
// protokolliereUebergangen vor die Zubehör-Liste gezogen — zwei Einträge statt einem.
func TestTheke_ForderungIstUebergehbarAmBuchUndAmGeraet(t *testing.T) {
	th := neueSperrTheke(t)
	ctx := context.Background()
	buecherei, schulbuch := th.bestand(t, "FOR")
	kind := schuelerAnlegen(t, th.pool, "Forderung", "08B", "S-SPERRWEG-4")

	// Die Forderung hängt an einem verlorenen Schulbuch — Topf des Landes.
	ausleiheUeberDenDienst(t, th.pool, schulbuch, kind, th.bearbeiter)
	var ex string
	if err := th.pool.QueryRow(ctx, `SELECT id FROM buecher_exemplare WHERE barcode_id = $1`, schulbuch).Scan(&ex); err != nil {
		t.Fatal(err)
	}
	if _, err := th.pool.Exec(ctx, `INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, art)
		VALUES ($1, $2, 'Verloren', 25.00, 'nicht_zurueckgegeben')`, ex, kind); err != nil {
		t.Fatalf("Forderung anlegen: %v", err)
	}
	th.neuesGeraet(t, "G-SPERRWEG-4", "Ladekabel")

	uebergehbar := func(err error) bool {
		return errors.Is(err, service.ErrBlocked) && service.IstUebergehbareSperre(err)
	}

	// Buch der Bücherei: ohne override_block gesperrt, mit ihm durch und protokolliert.
	if err := th.buch(t, buecherei, kind, false); !uebergehbar(err) {
		t.Fatalf("Buch ohne override_block: erwartet übergehbare Sperre, bekam %v", err)
	}
	if err := th.buch(t, buecherei, kind, true); err != nil {
		t.Fatalf("Buch mit override_block: %v", err)
	}
	if th.offen(t, buecherei) != 1 || th.uebergangen(t, kind) != 1 {
		t.Errorf("Buch: offen %d, Protokoll %d — erwartet 1 und 1", th.offen(t, buecherei), th.uebergangen(t, kind))
	}

	// Gerät: dieselbe Regel.
	if _, err := th.geraet(t, "G-SPERRWEG-4", kind, false, false); !uebergehbar(err) {
		t.Fatalf("Gerät ohne override_block: erwartet übergehbare Sperre, bekam %v", err)
	}
	res, err := th.geraet(t, "G-SPERRWEG-4", kind, false, true)
	if err != nil || res.Type != "geraet_check" {
		t.Fatalf("Gerät mit override_block, Zubehör offen: erwartet geraet_check, bekam %v / %v", res, err)
	}
	if th.offen(t, "G-SPERRWEG-4") != 0 || th.uebergangen(t, kind) != 1 {
		t.Errorf("die Zubehör-Liste hat gebucht oder protokolliert: offen %d, Protokoll %d",
			th.offen(t, "G-SPERRWEG-4"), th.uebergangen(t, kind))
	}
	if _, err := th.geraet(t, "G-SPERRWEG-4", kind, true, true); err != nil {
		t.Fatalf("Gerät mit override_block und bestätigtem Zubehör: %v", err)
	}
	if th.offen(t, "G-SPERRWEG-4") != 1 || th.uebergangen(t, kind) != 2 {
		t.Errorf("Gerät: offen %d, Protokoll %d — erwartet 1 und 2", th.offen(t, "G-SPERRWEG-4"), th.uebergangen(t, kind))
	}
}

// Übergangen ist erst, was ausgeliehen wurde. Weist nach der Sperre noch das Ausleihlimit ab,
// steht nichts im Protokoll — bis zum 24.09.2026 schrieb die Prüfung den Eintrag, bevor
// feststand, ob überhaupt ausgeliehen wird.
// Rot gesehen am Rückbau: protokolliereUebergangen in HandleUnifiedCheckout direkt hinter
// die Prüfung gezogen — ein Eintrag ohne Ausleihe.
func TestTheke_UebergehenOhneAusleiheStehtNichtImProtokoll(t *testing.T) {
	th := neueSperrTheke(t)
	ctx := context.Background()
	var vorher *string
	if err := th.pool.QueryRow(ctx,
		`SELECT (SELECT wert FROM system_einstellungen WHERE schluessel = 'max_ausleihen_schueler')`).Scan(&vorher); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		var err error
		if vorher == nil {
			_, err = th.pool.Exec(context.Background(), `DELETE FROM system_einstellungen WHERE schluessel = 'max_ausleihen_schueler'`)
		} else {
			_, err = th.pool.Exec(context.Background(),
				`UPDATE system_einstellungen SET wert = $1 WHERE schluessel = 'max_ausleihen_schueler'`, *vorher)
		}
		if err != nil {
			t.Errorf("Einstellung zurücksetzen: %v", err)
		}
	})
	if _, err := th.pool.Exec(ctx, `INSERT INTO system_einstellungen (schluessel, wert) VALUES ('max_ausleihen_schueler', '1')
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`); err != nil {
		t.Fatalf("Ausleihlimit setzen: %v", err)
	}

	buecherei, _ := th.bestand(t, "LIM")
	kind := schuelerAnlegen(t, th.pool, "Limit", "08B", "S-SPERRWEG-5")
	erstes := titelMitMeldebestand(t, th.pool, "Erstes Buch LIM", 1)
	exemplar(t, th.pool, erstes, "B-LIM-1", true, "")
	ausleiheUeberDenDienst(t, th.pool, "B-LIM-1", kind, th.bearbeiter) // das Limit ist erreicht
	altes := titelMitMeldebestand(t, th.pool, "Verlorenes Buch LIM", 1)
	verloren := exemplar(t, th.pool, altes, "B-LIM-ALT", false, "")
	if _, err := th.pool.Exec(ctx, `INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, art)
		VALUES ($1, $2, 'Verloren', 9.00, 'nicht_zurueckgegeben')`, verloren, kind); err != nil {
		t.Fatalf("Forderung anlegen: %v", err)
	}

	err := th.buch(t, buecherei, kind, true)
	if !errors.Is(err, service.ErrBlocked) || service.IstUebergehbareSperre(err) {
		t.Fatalf("erwartet die Abweisung am Ausleihlimit, bekam %v", err)
	}
	if n := th.uebergangen(t, kind); n != 0 {
		t.Errorf("nichts wurde ausgeliehen, trotzdem %d Protokolleinträge „übergangen“", n)
	}
}
