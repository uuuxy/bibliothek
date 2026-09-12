package service

// Das abgeschriebene Buch kommt doch zurück (#597, Etappe 2).
//
// Ist ein Lernmittel als „nicht zurückgegeben" abgerechnet, steht die Forderung beim
// Kind: Sperre, Löschblockade, ein Bescheid über den Wiederbeschaffungswert. Legt das
// Kind das Buch später auf die Theke, hat der Scan bis zum 12.09.2026 nur „Buch
// reaktiviert" gemeldet — die Forderung blieb offen, und niemand sah es.
//
// Vor der Übergabe an die Schulaufsicht gehört die Forderung storniert. Danach nicht:
// Der Fall liegt bei der Aufsicht, die Schule muss sie unverzüglich informieren
// (Leitfaden LMF Nr. 12.7). Dafür trägt der Bescheid `rueckgabe_nach_uebergabe`.
//
// Echtes Postgres über den ECHTEN Service — geprüft wird der Live-Pfad des Scans.

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

type rueckkehrAufbau struct {
	svc          OmniboxService
	bescheidRepo repository.BescheidRepository
	pool         interface {
		Exec(context.Context, string, ...any)
	}
	mitarbeiterID  string
	schuelerID     string
	titelID        string
	barcodeOffen   string
	barcodeUeberg  string
	bescheidOffen  string
	bescheidUeberg string
}

func TestRueckkehrEinesAbgerechnetenBuches(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	// — Beteiligte —
	var mitarbeiterID, schuelerID, titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ($1, 'Rueck', 'Kehr', $2, 'mitarbeiter', true) RETURNING id
	`, "MA-"+suffix, "rueckkehr-"+suffix+"@schule.invalid").Scan(&mitarbeiterID); err != nil {
		t.Fatalf("Mitarbeiter anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ($1, 'Wieder', 'Da', '07B', 2031) RETURNING id
	`, "SCH-"+suffix).Scan(&schuelerID); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, medientyp, ist_lernmittel)
		VALUES ('Rueckkehr-Testband', 'Prüfer', 'Buch', true) RETURNING id
	`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	t.Cleanup(func() {
		auf := context.Background()
		if _, err := pool.Exec(auf, `DELETE FROM schadensfaelle WHERE schueler_id = $1`, schuelerID); err != nil {
			t.Errorf("Aufräumen Forderungen: %v", err)
		}
		if _, err := pool.Exec(auf, `DELETE FROM schadensersatz_bescheide WHERE schueler_id = $1`, schuelerID); err != nil {
			t.Errorf("Aufräumen Bescheide: %v", err)
		}
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
		if _, err := pool.Exec(auf, `DELETE FROM benutzer WHERE id = $1`, mitarbeiterID); err != nil {
			t.Errorf("Aufräumen Mitarbeiter: %v", err)
		}
	})

	studentRepo := repository.NewStudentRepository(pool)
	bookRepo := repository.NewBookRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	loanRepo := repository.NewLoanRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)
	damageRepo := repository.NewDamageRepository(pool)
	bescheidRepo := repository.NewBescheidRepository(pool)
	loanSvc := NewLoanService(pool, studentRepo, bookRepo, loanRepo, auditRepo)
	deviceSvc := NewDeviceService(pool, studentRepo, loanRepo, auditRepo)
	svc := NewOmniboxService(pool, studentRepo, bookRepo, userRepo, loanRepo, loanSvc, deviceSvc)

	// abgerechnet: ausleihen → „nicht zurückgegeben" melden → Bescheid schreiben.
	abgerechnet := func(t *testing.T, barcode string, frist time.Time) string {
		t.Helper()
		var exemplarID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, einkaufspreis)
			VALUES ($1, $2, true, 20.00) RETURNING id`, titelID, barcode).Scan(&exemplarID); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
		ex, err := bookRepo.GetCopyByBarcode(ctx, barcode)
		if err != nil {
			t.Fatalf("Exemplar lesen: %v", err)
		}
		// overrideBlock: Der zweite Fall liehe sonst an einem Kind, das der erste Fall
		// gerade gesperrt hat — der Aufbau soll nicht vom Ausgang des vorigen abhängen.
		if _, err := loanSvc.HandleUnifiedCheckout(ctx, ex, &schuelerID, nil, mitarbeiterID, true); err != nil {
			t.Fatalf("Ausleihe: %v", err)
		}
		var loanID string
		if err := pool.QueryRow(ctx,
			`SELECT id FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`, exemplarID).Scan(&loanID); err != nil {
			t.Fatalf("Ausleihe lesen: %v", err)
		}
		schadensID, err := damageRepo.ReportDamage(ctx, exemplarID, loanID, schuelerID, mitarbeiterID,
			"Nicht zurückgegeben (Test)", repository.SchadensArtNichtZurueck, 12.00)
		if err != nil {
			t.Fatalf("Forderung melden: %v", err)
		}
		b, err := bescheidRepo.Erstelle(ctx, repository.BescheidEingabe{
			SchuelerID: schuelerID, Mittel: "land", Kassenjahr: time.Now().Year(), FristBis: frist,
			Positionen:     []repository.BescheidPositionEingabe{{SchadensfallID: schadensID, Betrag: 12.00}},
			Snapshot:       map[string]string{"name": "Wieder Da"},
			ErstelltVon:    mitarbeiterID,
			Referenznummer: func(nr int) string { return fmt.Sprintf("TEST-%s-%04d", suffix, nr) },
		})
		if err != nil {
			t.Fatalf("Bescheid erstellen: %v", err)
		}
		return b.ID
	}

	forderungOffen := func(t *testing.T, barcode string) (offen bool, grund string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `
			SELECT f.storniert_am IS NULL, coalesce(f.stornierungsgrund, '')
			FROM schadensfaelle f JOIN buecher_exemplare e ON e.id = f.exemplar_id
			WHERE e.barcode_id = $1`, barcode).Scan(&offen, &grund); err != nil {
			t.Fatalf("Forderung lesen: %v", err)
		}
		return offen, grund
	}

	// ── Fall 1: Der Bescheid liegt noch bei der Schule ────────────────────────────
	barcodeOffen := "B-RUECK-A-" + suffix
	abgerechnet(t, barcodeOffen, time.Now().AddDate(0, 0, 28))

	res, err := svc.ProcessQuery(ctx, OmniboxQuery{Query: barcodeOffen, StaffID: mitarbeiterID, StaffRole: "mitarbeiter"})
	if err != nil {
		t.Fatalf("Scan des zurückgebrachten Buches: %v", err)
	}
	if offen, grund := forderungOffen(t, barcodeOffen); offen {
		t.Errorf("die Forderung steht nach der Rückgabe noch offen — das Kind bleibt gesperrt und zahlt für ein Buch, das da ist")
	} else if !strings.Contains(grund, "Rückgabe") {
		t.Errorf("Stornierungsgrund nennt die Rückgabe nicht: %q", grund)
	}
	if !strings.Contains(res.Message, "storniert") {
		t.Errorf("die Theke erfährt nichts von der Stornierung: %q", res.Message)
	}
	if res.AufsichtInformieren != "" {
		t.Errorf("ohne Übergabe ist niemand zu informieren, gemeldet wurde: %q", res.AufsichtInformieren)
	}

	// ── Fall 2: Der Bescheid ist schon bei der Schulaufsicht ──────────────────────
	barcodeUeberg := "B-RUECK-B-" + suffix
	bescheidID := abgerechnet(t, barcodeUeberg, time.Now().AddDate(0, 0, -1))
	if err := bescheidRepo.Uebergebe(ctx, bescheidID); err != nil {
		t.Fatalf("Übergabe an die Schulaufsicht: %v", err)
	}

	res, err = svc.ProcessQuery(ctx, OmniboxQuery{Query: barcodeUeberg, StaffID: mitarbeiterID, StaffRole: "mitarbeiter"})
	if err != nil {
		t.Fatalf("Scan nach der Übergabe: %v", err)
	}
	if offen, _ := forderungOffen(t, barcodeUeberg); !offen {
		t.Error("nach der Übergabe darf die Schule die Forderung nicht selbst stornieren — der Fall liegt bei der Aufsicht")
	}
	b, err := bescheidRepo.Lies(ctx, bescheidID)
	if err != nil {
		t.Fatalf("Bescheid lesen: %v", err)
	}
	if !b.RueckgabeNachUebergabe {
		t.Error("der Bescheid trägt keinen Merker „Rückgabe nach Übergabe“ — das Sekretariat sieht nicht, dass es die Aufsicht informieren muss")
	}
	if !strings.Contains(res.AufsichtInformieren, "Schulaufsicht") ||
		!strings.Contains(res.AufsichtInformieren, b.Referenznummer) {
		t.Errorf("die Theke erfährt nicht, dass die Schulaufsicht zu informieren ist: %q", res.AufsichtInformieren)
	}
	// Und nicht als Erfolgsmeldung: Eine offene Aufgabe gehört nicht in denselben Kanal
	// wie „erledigt" — die Theke zeigt Message grün und diesen Hinweis als Warnung.
	if strings.Contains(res.Message, "storniert") {
		t.Errorf("die Meldung behauptet eine Stornierung, die es nicht gab: %q", res.Message)
	}

	// ── Gegenprobe: Ein beschädigtes Buch ist kein verschwundenes ─────────────────
	// Der Schaden wurde festgestellt, das Buch lag dabei vor. Taucht es „wieder auf",
	// ändert das an der Forderung nichts — eine Ratsche, die nur in eine Richtung
	// prüft, würde hier stillschweigend jede Schadensforderung mit erledigen.
	barcodeDefekt := "B-RUECK-C-" + suffix
	var defektID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, einkaufspreis)
		VALUES ($1, $2, true, 20.00) RETURNING id`, titelID, barcodeDefekt).Scan(&defektID); err != nil {
		t.Fatalf("Exemplar anlegen: %v", err)
	}
	if _, err := damageRepo.MarkCopyDefekt(ctx, defektID, nil, &schuelerID, mitarbeiterID, 7.50, "Wasserschaden (Test)"); err != nil {
		t.Fatalf("Defekt melden: %v", err)
	}
	if _, err := svc.ProcessQuery(ctx, OmniboxQuery{Query: barcodeDefekt, StaffID: mitarbeiterID, StaffRole: "mitarbeiter"}); err != nil {
		t.Fatalf("Scan des defekten Buches: %v", err)
	}
	if offen, _ := forderungOffen(t, barcodeDefekt); !offen {
		t.Error("die Schadensforderung wurde mit storniert — der Hook darf nur „nicht zurückgegeben“ beenden")
	}
}
