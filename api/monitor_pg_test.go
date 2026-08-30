package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Öffentliche Sichtbarkeit: Katalog und Flur-Monitor urteilen über denselben Titel gleich.
//
// Bis zum 30.08.2026 hatte jede der beiden Seiten ohne Anmeldung ihre eigene Regel: Der
// Katalog (api/opac.go) ließ Lernmittel weg und zeigte nur Titel mit einem Exemplar im
// Haus; der Monitor kannte beides nicht. Auf einem Schulserver, dessen Bestand zum
// größten Teil aus Schulbüchern besteht, wäre das Mathebuch der 7 „Buch des Monats"
// gewesen — das Demo-Seed musste genau darum herumtricksen („sonst gewinnen
// Lernmittel"). Ein Paar, das nur zufällig einig war (docs/sweeps.md,
// Geschwister-Asymmetrie).
//
// Seit dem Umbau setzen beide Seiten DASSELBE Prädikat ein
// (repository.OeffentlichSichtbar). Dieser Test hält das Paar zusammen: Er seedet je
// einen Titel pro Ausschlussgrund und fragt BEIDE echten Handler — nicht eine
// nachgebaute Abfrage, die eine Regression im Handler nie sähe. Jeder ausgeschlossene
// Titel bekommt dabei jeden Weg auf den Monitor, den es gibt (Leser in den letzten
// Tagen, Cover, jüngstes Anlagedatum) — alle müssen zu sein.
func TestOeffentlicheSichtbarkeit_KatalogUndMonitorUrteilenGleich(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	leser := []string{
		seedSchueler(t, pool, "S-MON-1", "Mia", "5a"),
		seedSchueler(t, pool, "S-MON-2", "Ben", "5a"),
	}

	// Schulbuch, dessen Kennzeichen NUR in der Signatur steht — die manuelle Neuanlage
	// schreibt es dorthin, der Sammelimport in den Titel (pkg/lmf kennt beide Wege).
	lmf := seedMonitorTitel(t, pool, "Mathematik Neue Wege 7", "LMF Mathe 7", true, 0)
	for i, sid := range leser {
		seedLeserAusleihe(t, pool, exemplar(t, pool, lmf, fmt.Sprintf("MON-LMF-%d", i), true, ""), sid)
	}

	// Freihand-Roman: das Einzige, was beide Seiten zeigen dürfen.
	tschick := seedMonitorTitel(t, pool, "Tschick", "Jug Her", true, 10)
	seedLeserAusleihe(t, pool, exemplar(t, pool, tschick, "MON-TSCHICK-1", true, ""), leser[0])

	// Komplett ausgesondert — mit Leser, Cover und jüngstem Datum.
	ausgesondert := seedMonitorTitel(t, pool, "Ausgesonderter Roman", "Jug Aus", true, 0)
	seedLeserAusleihe(t, pool, seedAusgesondertesExemplar(t, pool, ausgesondert, "MON-AUS-1"), leser[1])

	// Nur bestellt, noch nicht im Haus — als jüngster Titel wäre er „Neu eingetroffen".
	bestellt := seedMonitorTitel(t, pool, "Bestellter Roman", "Jug Bes", true, 0)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus)
		VALUES ($1, 'MON-BES-1', false, 'bestellt')`, bestellt); err != nil {
		t.Fatalf("bestelltes Exemplar anlegen: %v", err)
	}

	aufMonitor := monitorAntwortLesen(t, pool).alleTitel()
	for titel, sichtbar := range map[string]bool{
		"Mathematik Neue Wege 7": false,
		"Tschick":                true,
		"Ausgesonderter Roman":   false,
		"Bestellter Roman":       false,
	} {
		imKatalog := opacGesamt(t, pool, titel) > 0
		if imKatalog != sichtbar {
			t.Errorf("Katalog: %q sichtbar=%v, erwartet %v", titel, imKatalog, sichtbar)
		}
		if aufMonitor[titel] != sichtbar {
			t.Errorf("Monitor: %q sichtbar=%v, erwartet %v — Katalog und Monitor urteilen verschieden",
				titel, aufMonitor[titel], sichtbar)
		}
	}
}

// monitorTitelAntwort und monitorAntwort spiegeln nur die Felder der JSON-Antwort, die
// die Tests lesen — bewusst nicht der Antworttyp des Handlers, damit der Test auch
// gegen einen älteren Stand übersetzt (Rot-Probe am Rückbau).
type monitorTitelAntwort struct {
	Titel string `json:"titel"`
}

type monitorAntwort struct {
	BuchDesMonats   *monitorTitelAntwort  `json:"buch_des_monats"`
	NeuEingetroffen []monitorTitelAntwort `json:"neu_eingetroffen"`
	Beliebt         []monitorTitelAntwort `json:"beliebt"`
}

// alleTitel liefert jeden Titel, der auf irgendeiner der drei Folien steht.
func (a monitorAntwort) alleTitel() map[string]bool {
	titel := map[string]bool{}
	if a.BuchDesMonats != nil {
		titel[a.BuchDesMonats.Titel] = true
	}
	for _, liste := range [][]monitorTitelAntwort{a.NeuEingetroffen, a.Beliebt} {
		for _, t := range liste {
			titel[t.Titel] = true
		}
	}
	return titel
}

// monitorAntwortLesen fragt den ÖFFENTLICHEN Monitor-Endpunkt über den echten Handler ab.
func monitorAntwortLesen(t *testing.T, pool *pgxpool.Pool) monitorAntwort {
	t.Helper()
	srv := &Server{DB: &db.Database{Pool: pool}}
	req := httptest.NewRequest(http.MethodGet, "/api/monitor/slides", nil)
	rec := httptest.NewRecorder()
	srv.GetMonitorSlidesHandler()(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Monitor: HTTP %d: %s", rec.Code, rec.Body.String())
	}
	var antwort monitorAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Monitor-Antwort unlesbar: %v / %s", err, rec.Body.String())
	}
	return antwort
}

// seedMonitorTitel legt einen Titel mit Signatur, wahlweise Cover und einem Anlagedatum
// vor alterTage Tagen an — das Datum steuert „Neu eingetroffen", das Cover die
// Cover-Pflicht von „Buch des Monats" und „Neu eingetroffen".
func seedMonitorTitel(t *testing.T, pool *pgxpool.Pool, titel, signatur string, mitCover bool, alterTage int) string {
	t.Helper()
	var cover *string
	if mitCover {
		url := "https://cover.invalid/" + titel
		cover = &url
	}
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO buecher_titel (titel, signatur, cover_url, erstellt_am)
		VALUES ($1, $2, $3, NOW() - make_interval(days => $4))
		RETURNING id`, titel, signatur, cover, alterTage).Scan(&id); err != nil {
		t.Fatalf("Titel %q anlegen: %v", titel, err)
	}
	return id
}

// seedAusgesondertesExemplar legt ein ausgesondertes Exemplar an (Grund ist Pflicht,
// chk_aussonderung_grund).
func seedAusgesondertesExemplar(t *testing.T, pool *pgxpool.Pool, titelID, barcode string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, ist_ausgesondert, aussonderung_grund)
		VALUES ($1, $2, false, true, 'AUSSORTIERT') RETURNING id`, titelID, barcode).Scan(&id); err != nil {
		t.Fatalf("ausgesondertes Exemplar %q anlegen: %v", barcode, err)
	}
	return id
}

// seedLeserAusleihe bucht eine Schüler-Ausleihe von vor zwei Tagen — innerhalb JEDES
// Zeitfensters des Monitors (7 und 30 Tage). Zeit aus der Datenbank, nicht aus Go: Die
// Fenster werden dort gerechnet.
func seedLeserAusleihe(t *testing.T, pool *pgxpool.Pool, exemplarID, schuelerID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
		VALUES ($1, $2, NOW() - interval '2 days', NOW() + interval '21 days')`,
		exemplarID, schuelerID); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
}
