package repository

import (
	"bibliothek/internal/testdaten"
	"context"
	"testing"
)

// legeVerlorenesExemplarAn legt einen Titel mit einem Exemplar an, schließt einen eigenen Lauf
// ohne diesen Scan ab und liefert die ID des Exemplars: den Zustand, den
// MarkiereVerlustAlsGefunden und LoescheVerlusteEndgueltig vorfinden (ist_ausgesondert = true,
// aussonderung_grund = 'VERLUST', eine Zeile in inventur_verluste).
func legeVerlorenesExemplarAn(t *testing.T, ctx context.Context, repo *InventurAbfragen, titel, barcode, bearbeiterID string) string {
	t.Helper()
	var titelID string
	if err := repo.db.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel) VALUES ($1) RETURNING id`, titel,
	).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	var exemplarID string
	if err := repo.db.QueryRow(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar) VALUES ($1, $2, true) RETURNING id`,
		titelID, barcode,
	).Scan(&exemplarID); err != nil {
		t.Fatalf("Exemplar anlegen: %v", err)
	}
	var sessionID string
	if err := repo.db.QueryRow(ctx,
		`INSERT INTO inventur_sessions (scope_type) VALUES ('global') RETURNING id`,
	).Scan(&sessionID); err != nil {
		t.Fatalf("Session anlegen: %v", err)
	}
	if _, err := repo.SchliesseLaufAb(ctx, sessionID, InventurBereich{}, bearbeiterID); err != nil {
		t.Fatalf("Session abschliessen (Exemplar als Verlust buchen): %v", err)
	}
	return exemplarID
}

// Der Knopf „Gefunden": Das Exemplar kommt zurück in den Umlauf, und der Fehlbestandsbericht
// hält fest, dass es gefunden wurde; sonst ließe sich aus dem Bericht nicht mehr ablesen, dass
// ein Buch zeitweise als verloren galt.
func TestMarkiereVerlustAlsGefunden(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)

	bearbeiterID := legeBearbeiterAn(t, pool)
	exemplarID := legeVerlorenesExemplarAn(t, ctx, repo, "Physikbuch 8", "GEF-1", bearbeiterID)

	gefunden, _, err := repo.MarkiereVerlustAlsGefunden(ctx, exemplarID, bearbeiterID)
	if err != nil {
		t.Fatalf("MarkiereVerlustAlsGefunden: %v", err)
	}
	if !gefunden {
		t.Fatal("gefunden=false, erwartet true — das Exemplar war ein offener Verlust")
	}

	var istAusgesondert, istAusleihbar bool
	var grund string
	if err := pool.QueryRow(ctx,
		`SELECT ist_ausgesondert, ist_ausleihbar, coalesce(aussonderung_grund, '') FROM buecher_exemplare WHERE id = $1`,
		exemplarID,
	).Scan(&istAusgesondert, &istAusleihbar, &grund); err != nil {
		t.Fatalf("Exemplar nach Fund lesen: %v", err)
	}
	if istAusgesondert || !istAusleihbar || grund != "" {
		t.Errorf("Exemplar nicht sauber wiederhergestellt: ausgesondert=%v ausleihbar=%v grund=%q",
			istAusgesondert, istAusleihbar, grund)
	}

	var gefundenAmGesetzt bool
	if err := pool.QueryRow(ctx,
		`SELECT gefunden_am IS NOT NULL FROM inventur_verluste WHERE exemplar_id = $1`, exemplarID,
	).Scan(&gefundenAmGesetzt); err != nil {
		t.Fatalf("Verlust-Zeile nach Fund lesen: %v", err)
	}
	if !gefundenAmGesetzt {
		t.Error("gefunden_am wurde nicht gesetzt — der Bericht zeigt den Fund nicht an")
	}

	// Zweiter Aufruf auf dasselbe Exemplar: Kein Verlust ist mehr offen, also kein Fehler und
	// keine Wirkung.
	nochmal, _, err := repo.MarkiereVerlustAlsGefunden(ctx, exemplarID, bearbeiterID)
	if err != nil {
		t.Fatalf("zweiter Aufruf: %v", err)
	}
	if nochmal {
		t.Error("zweiter Aufruf meldet gefunden=true, aber es gibt keinen offenen Verlust mehr")
	}
}

// Eine falsche oder fremde ID liefert false und keinen Fehler; die Tür macht daraus 404 statt
// 500.
func TestMarkiereVerlustAlsGefunden_UnbekannteID(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	repo := NewInventurAbfragen(pool)

	gefunden, _, err := repo.MarkiereVerlustAlsGefunden(context.Background(), "00000000-0000-0000-0000-000000000000", "bearbeiter-1")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if gefunden {
		t.Error("gefunden=true für eine nicht existierende ID")
	}
}

// Nur als VERLUST gebuchte Exemplare verschwinden; ein gewöhnliches Exemplar in derselben
// Anfrage bleibt, damit die Route nicht zum allgemeinen Weg wird, ein Exemplar zu löschen. Der
// Fehlbestandsbericht bleibt lesbar (ON DELETE SET NULL, die Textspalten sind eine Abschrift).
func TestLoescheVerlusteEndgueltig(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)

	bearbeiterID := legeBearbeiterAn(t, pool)
	verloren1 := legeVerlorenesExemplarAn(t, ctx, repo, "Chemiebuch 9", "LOE-1", bearbeiterID)
	verloren2 := legeVerlorenesExemplarAn(t, ctx, repo, "Biobuch 9", "LOE-2", bearbeiterID)

	// Ein gewöhnliches, nicht ausgesondertes Exemplar dazwischen: Es wird nicht gelöscht, auch
	// wenn seine ID mitgeschickt wird.
	var titelID, normalID string
	titelID = testdaten.LegeTitelAn(t, pool, "Reguläres Buch", 5)
	normalID = testdaten.LegeExemplarAn(t, pool, titelID, "LOE-NORMAL", true, "")

	geloescht, err := repo.LoescheVerlusteEndgueltig(ctx,
		[]string{verloren1, verloren2, normalID, "00000000-0000-0000-0000-000000000000"}, bearbeiterID)
	if err != nil {
		t.Fatalf("LoescheVerlusteEndgueltig: %v", err)
	}
	if len(geloescht) != 2 {
		t.Fatalf("%d gelöscht, erwartet 2 (nur die beiden Verlust-Exemplare)", len(geloescht))
	}
	// Die Zahl allein genügt nicht: Die Oberfläche nimmt genau die zurückgemeldeten Zeilen aus
	// ihrer Liste. Kämen hier die angefragten IDs statt der gelöschten, verschwänden das
	// gewöhnliche Exemplar und die erfundene UUID nur auf dem Bildschirm.
	zurueck := map[string]bool{geloescht[0]: true, geloescht[1]: true}
	if !zurueck[verloren1] || !zurueck[verloren2] {
		t.Fatalf("zurückgemeldet %v, erwartet genau die beiden Verlust-Exemplare", geloescht)
	}

	var uebrig int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM buecher_exemplare WHERE id = ANY($1)`,
		[]string{verloren1, verloren2},
	).Scan(&uebrig); err != nil {
		t.Fatalf("Kontrolle nach dem Löschen: %v", err)
	}
	if uebrig != 0 {
		t.Errorf("%d Verlust-Exemplare noch da, erwartet 0", uebrig)
	}

	var normalNochDa bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM buecher_exemplare WHERE id = $1)`, normalID,
	).Scan(&normalNochDa); err != nil {
		t.Fatalf("Kontrolle des normalen Exemplars: %v", err)
	}
	if !normalNochDa {
		t.Error("das reguläre Exemplar wurde fälschlich mitgelöscht")
	}

	// Bericht bleibt lesbar: exemplar_id jetzt NULL, aber Barcode/Titel als Abschrift da.
	var barcode string
	var exemplarIDNachher *string
	if err := pool.QueryRow(ctx,
		`SELECT barcode_id, exemplar_id::text FROM inventur_verluste WHERE barcode_id = 'LOE-1'`,
	).Scan(&barcode, &exemplarIDNachher); err != nil {
		t.Fatalf("Verlust-Zeile nach dem Löschen lesen: %v", err)
	}
	if barcode != "LOE-1" {
		t.Errorf("Abschrift verloren: barcode=%q", barcode)
	}
	if exemplarIDNachher != nil {
		t.Errorf("exemplar_id sollte NULL sein (Exemplar gelöscht), war %q", *exemplarIDNachher)
	}
}
