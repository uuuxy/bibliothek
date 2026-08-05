package repository

import (
	"context"
	"testing"
)

// seedVerlorenesExemplar legt einen Titel mit einem Exemplar an, schliesst eine
// eigene Inventur-Session ohne diesen Scan ab und liefert die entstandene
// Exemplar-ID — also genau den Zustand, den MarkiereVerlustAlsGefunden und
// EndgueltigLoescheVerlustExemplare in der Praxis vorfinden (ist_ausgesondert=true,
// aussonderung_grund='VERLUST', eine Zeile in inventur_verluste).
func seedVerlorenesExemplar(t *testing.T, ctx context.Context, repo *InventoryRepository, titel, barcode string) string {
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
	if _, err := repo.FinishInventurSession(ctx, sessionID, InventurScope{}); err != nil {
		t.Fatalf("Session abschliessen (Exemplar als Verlust buchen): %v", err)
	}
	return exemplarID
}

// TestMarkiereVerlustAlsGefunden belegt den Regelfall des "Gefunden"-Knopfs: Das
// Exemplar kommt zurück in Umlauf, UND der Fehlbestandsbericht merkt sich, DASS es
// gefunden wurde (nicht nur das Endergebnis) — sonst könnte man aus dem Bericht nicht
// mehr nachvollziehen, dass ein Buch zwischenzeitlich als verloren galt.
func TestMarkiereVerlustAlsGefunden(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventoryRepository(pool)

	bearbeiterID := seedBearbeiter(t, pool)
	exemplarID := seedVerlorenesExemplar(t, ctx, repo, "Physikbuch 8", "GEF-1")

	gefunden, err := repo.MarkiereVerlustAlsGefunden(ctx, exemplarID, bearbeiterID)
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

	// Zweiter Aufruf auf dasselbe (jetzt wieder normale) Exemplar: kein Verlust mehr
	// offen, also nichts zu tun — kein Fehler, aber auch keine Wirkung.
	nochmal, err := repo.MarkiereVerlustAlsGefunden(ctx, exemplarID, bearbeiterID)
	if err != nil {
		t.Fatalf("zweiter Aufruf: %v", err)
	}
	if nochmal {
		t.Error("zweiter Aufruf meldet gefunden=true, aber es gibt keinen offenen Verlust mehr")
	}
}

// TestMarkiereVerlustAlsGefunden_UnbekannteID belegt den Fall einer falschen oder
// fremden ID: kein Fehler, sondern schlicht false — der Handler macht daraus ein
// sauberes 404 statt eines 500.
func TestMarkiereVerlustAlsGefunden_UnbekannteID(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	repo := NewInventoryRepository(pool)

	gefunden, err := repo.MarkiereVerlustAlsGefunden(context.Background(), "00000000-0000-0000-0000-000000000000", "bearbeiter-1")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if gefunden {
		t.Error("gefunden=true für eine nicht existierende ID")
	}
}

// TestEndgueltigLoescheVerlustExemplare belegt den Lösch-Knopf: Nur tatsächlich als
// VERLUST gebuchte Exemplare verschwinden, ein normales (nicht ausgesondertes)
// Exemplar in derselben Anfrage bleibt unangetastet — die Route darf nicht zum
// allgemeinen Lösch-Weg werden. Und der Fehlbestandsbericht bleibt lesbar (Migration
// 059: ON DELETE SET NULL, die Textspalten sind eine Abschrift).
func TestEndgueltigLoescheVerlustExemplare(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventoryRepository(pool)

	bearbeiterID := seedBearbeiter(t, pool)
	verloren1 := seedVerlorenesExemplar(t, ctx, repo, "Chemiebuch 9", "LOE-1")
	verloren2 := seedVerlorenesExemplar(t, ctx, repo, "Biobuch 9", "LOE-2")

	// Ein ganz normales, nicht ausgesondertes Exemplar dazwischen — darf nicht
	// gelöscht werden, selbst wenn seine ID mitgeschickt wird.
	var titelID, normalID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel) VALUES ('Reguläres Buch') RETURNING id`,
	).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar) VALUES ($1, 'LOE-NORMAL', true) RETURNING id`,
		titelID,
	).Scan(&normalID); err != nil {
		t.Fatalf("normales Exemplar anlegen: %v", err)
	}

	anzahl, err := repo.EndgueltigLoescheVerlustExemplare(ctx,
		[]string{verloren1, verloren2, normalID, "00000000-0000-0000-0000-000000000000"}, bearbeiterID)
	if err != nil {
		t.Fatalf("EndgueltigLoescheVerlustExemplare: %v", err)
	}
	if anzahl != 2 {
		t.Fatalf("%d gelöscht, erwartet 2 (nur die beiden Verlust-Exemplare)", anzahl)
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
