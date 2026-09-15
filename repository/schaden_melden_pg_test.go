package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Aussonderungsgrund folgt der Fallgruppe. Bis zum 15.09.2026 stand bei jedem
// gemeldeten Schaden BESCHAEDIGUNG, auch bei „nicht zurückgegeben" — die Fund-Meldung
// und das endgültige Löschen des Fehlbestandsberichts fanden diese Exemplare nie
// (beide kennen nur VERLUST, OFFEN.md 5.3). Rot am alten Code: grund = BESCHAEDIGUNG.
func TestReportDamage_NichtZurueckgegebenIstVerlust(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewDamageRepository(pool)

	copyID := seedSignaturMitExemplaren(t, pool, "VerlustGrund", 1)[0]
	schueler := seedSchueler(t, pool, "VG-A", "Dana", "8b")
	bearbeiter := seedBearbeiter(t, pool)
	loan := seedAusleihe(t, pool, copyID, schueler, bearbeiter)

	if _, err := repo.ReportDamage(ctx, copyID, loan, schueler, bearbeiter, "nicht zurückgegeben", SchadensArtNichtZurueck, 0); err != nil {
		t.Fatalf("Verlust melden: %v", err)
	}
	if grund := aussonderungsGrund(t, pool, copyID); grund != "VERLUST" {
		t.Errorf("aussonderung_grund = %q, want VERLUST", grund)
	}
	var beendet bool
	if err := pool.QueryRow(ctx, `SELECT rueckgabe_am IS NOT NULL FROM ausleihen WHERE id = $1`, loan).Scan(&beendet); err != nil {
		t.Fatal(err)
	}
	if !beendet {
		t.Error("die Ausleihe läuft nach dem Verlust weiter")
	}
}

// Das Buch taucht beim Nachsuchen wieder auf: Der Fund im Fehlbestandsbericht beendet
// auch die Forderung, die das Buch abgerechnet hatte — dieselbe Regel wie an der Theke.
// Rot am alten Code: Das Exemplar kam zurück in den Umlauf, die Forderung blieb offen.
func TestMarkiereVerlustAlsGefunden_BeendetForderung(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	copyID := seedSignaturMitExemplaren(t, pool, "FundForderung", 1)[0]
	schueler := seedSchueler(t, pool, "FF-A", "Emil", "9c")
	bearbeiter := seedBearbeiter(t, pool)
	loan := seedAusleihe(t, pool, copyID, schueler, bearbeiter)
	schadensID, err := NewDamageRepository(pool).ReportDamage(ctx, copyID, loan, schueler, bearbeiter, "nicht zurückgegeben", SchadensArtNichtZurueck, 18.50)
	if err != nil {
		t.Fatalf("Verlust melden: %v", err)
	}
	// Unabhängig vom Grund, den ReportDamage setzt: Der Fund verlangt VERLUST.
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET aussonderung_grund = 'VERLUST' WHERE id = $1`, copyID); err != nil {
		t.Fatal(err)
	}

	gefunden, befund, err := NewInventoryRepository(pool).MarkiereVerlustAlsGefunden(ctx, copyID, bearbeiter)
	if err != nil {
		t.Fatalf("MarkiereVerlustAlsGefunden: %v", err)
	}
	if !gefunden {
		t.Fatal("gefunden=false, erwartet true")
	}
	if befund.StornierteForderungen != 1 || befund.StornierterBetrag != 18.50 {
		t.Errorf("Befund = %+v, want 1 stornierte Forderung über 18,50", befund)
	}
	var storniert bool
	var grund string
	if err := pool.QueryRow(ctx,
		`SELECT storniert_am IS NOT NULL AND ist_bezahlt, coalesce(stornierungsgrund, '') FROM schadensfaelle WHERE id = $1`,
		schadensID).Scan(&storniert, &grund); err != nil {
		t.Fatal(err)
	}
	if !storniert {
		t.Errorf("Forderung nach dem Fund noch offen (Grund %q)", grund)
	}
}

// aussonderungsGrund liest den Grund eines Exemplars (” = keiner).
func aussonderungsGrund(t *testing.T, pool *pgxpool.Pool, copyID string) string {
	t.Helper()
	var grund string
	if err := pool.QueryRow(context.Background(),
		`SELECT coalesce(aussonderung_grund, '') FROM buecher_exemplare WHERE id = $1`, copyID).Scan(&grund); err != nil {
		t.Fatalf("Aussonderungsgrund lesen: %v", err)
	}
	return grund
}
