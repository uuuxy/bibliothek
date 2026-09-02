package api

import (
	"context"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Zweite Tür zum Abgänger (Rasterdurchgang 02.09.2026): Neben dem LUSD-Import macht
// auch die Versetzung (POST /api/students/promote) Schüler zu Abgängern. Sie muss
// dieselbe Karenz-Uhr stempeln (abgaenger_seit) und dasselbe Sperrgrund-Präfix
// schreiben — sonst kehrt ein Versetzungs-Abgänger per LUSD als AKTIVER, aber
// weiterhin GESPERRTER Schüler zurück (Ghost-Block), und seine Karenz läuft über den
// Rückfall aktualisiert_am, den jeder Klick neu stellt.
func TestVersetzung_AbgaengerTraegtKarenzUhrUndKehrtEntsperrtZurueck(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id, geburtsdatum)
		VALUES ('VERS-1', 'Vera', 'Versetzt', '13', 2031, 'L-VERS', '2008-02-02') RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	resp := versetzungAusfuehren(t, pool, false)
	if resp.ArchivedCount != 1 {
		t.Fatalf("Versetzung sollte genau einen Abgänger machen: %+v", resp)
	}

	var seit bool
	var grund string
	if err := pool.QueryRow(ctx, `SELECT abgaenger_seit IS NOT NULL, COALESCE(block_reason, '') FROM schueler WHERE id = $1`, id).Scan(&seit, &grund); err != nil {
		t.Fatal(err)
	}
	if !seit {
		t.Error("Versetzung stempelt abgaenger_seit nicht — Karenz-Uhr läuft über aktualisiert_am")
	}
	if !strings.HasPrefix(grund, repository.AbgaengerSperrPraefix) {
		t.Errorf("Sperrgrund %q trägt nicht das Präfix %q, das Import und Zusammenführen erkennen", grund, repository.AbgaengerSperrPraefix)
	}

	// Rückkehr per LUSD-Import: aktiv UND entsperrt, wie beim Import-Abgänger.
	s := &Server{DB: &db.Database{Pool: pool}}
	if _, err := s.computeLusdChanges(ctx, []parsedStudentRow{
		{LusdID: "L-VERS", Vorname: "Vera", Nachname: "Versetzt", Klasse: "13"},
	}, true, true); err != nil {
		t.Fatal(err)
	}
	gesperrt, abgaenger, reason, _ := leseSchuelerStatus(t, pool, id)
	if gesperrt || abgaenger || reason != nil {
		t.Errorf("Versetzungs-Abgänger kehrt als Ghost-Block zurück: gesperrt=%v abgaenger=%v grund=%v", gesperrt, abgaenger, reason)
	}
}
