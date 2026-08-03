package repository

import (
	"context"
	"testing"
)

// TestInventurFehlbestandNenntDieBuecher belegt, dass der Abschluss einer Inventur nicht
// nur ZÄHLT, sondern festhält, WELCHE Exemplare fehlen.
//
// Vorher gab FinishInventurSession allein eine Zahl zurück: „47 Verluste". Damit kann
// niemand ins Regal gehen und nachsehen, ob ein Buch wirklich fehlt oder nur falsch
// einsortiert war. Und rekonstruieren liess es sich danach auch nicht mehr — die
// Exemplare fallen durch die Aussonderung aus der Scope-Bedingung.
//
// Geprüft wird an echtem Postgres: Die Buchung ist ein CTE aus UPDATE ... RETURNING mit
// anschliessendem INSERT. Ob dabei wirklich die nicht erfassten Zeilen ankommen, ist
// Datenbankverhalten und mit pgxmock nicht nachzubilden.
func TestInventurFehlbestandNenntDieBuecher(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	// Ein Titel mit Signatur, damit der Bericht auch die Regalangabe trägt — danach
	// sortiert man beim Nachsuchen. Quelle ist buecher_titel.signatur, der Text vom
	// Buchrücken: Die Abschrift lief früher über einen Join auf die nie gepflegte
	// Tabelle `signatures` und war deshalb immer leer (Migration 060).
	var titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, signatur) VALUES ('Deutschbuch 7', 'Biermann', 'LMF Deu 7')
		RETURNING id
	`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}

	exemplar := func(barcode string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			VALUES ($1, $2, true) RETURNING id
		`, titelID, barcode).Scan(&id); err != nil {
			t.Fatalf("Exemplar %s anlegen: %v", barcode, err)
		}
		return id
	}
	gescannt := exemplar("INV-DA")
	fehlt1 := exemplar("INV-WEG-1")
	_ = exemplar("INV-WEG-2")

	var sessionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO inventur_sessions (scope_type) VALUES ('global') RETURNING id
	`).Scan(&sessionID); err != nil {
		t.Fatalf("Session anlegen: %v", err)
	}

	repo := NewInventoryRepository(pool)
	if err := repo.RecordInventurScan(ctx, sessionID, gescannt); err != nil {
		t.Fatalf("Scan erfassen: %v", err)
	}

	verloren, err := repo.FinishInventurSession(ctx, sessionID, InventurScope{})
	if err != nil {
		t.Fatalf("Abschluss: %v", err)
	}
	if verloren != 2 {
		t.Fatalf("%d Verluste gemeldet, erwartet 2 (das gescannte Exemplar darf nicht dabei sein)", verloren)
	}

	liste, err := repo.LadeInventurVerluste(ctx, sessionID)
	if err != nil {
		t.Fatalf("Fehlbestand laden: %v", err)
	}
	if len(liste) != 2 {
		t.Fatalf("%d Zeilen im Fehlbestand, erwartet 2", len(liste))
	}

	// Die Abschrift muss lesbar sein — eine Liste aus Barcodes allein hilft im Regal nicht.
	gefunden := map[string]InventurVerlust{}
	for _, v := range liste {
		gefunden[v.BarcodeID] = v
	}
	if _, ok := gefunden["INV-DA"]; ok {
		t.Error("das gescannte Exemplar steht faelschlich im Fehlbestand")
	}
	eintrag, ok := gefunden["INV-WEG-1"]
	if !ok {
		t.Fatal("INV-WEG-1 fehlt im Fehlbestand")
	}
	if eintrag.Titel != "Deutschbuch 7" || eintrag.Autor != "Biermann" || eintrag.Signatur != "LMF Deu 7" {
		t.Errorf("Abschrift unvollstaendig: %+v", eintrag)
	}

	// Und sie ueberlebt das Loeschen des Exemplars — genau dann fragt jemand nach.
	if _, err := pool.Exec(ctx, `DELETE FROM buecher_exemplare WHERE id = $1`, fehlt1); err != nil {
		t.Fatalf("Exemplar loeschen: %v", err)
	}
	nachher, err := repo.LadeInventurVerluste(ctx, sessionID)
	if err != nil {
		t.Fatalf("Fehlbestand nach Loeschung: %v", err)
	}
	if len(nachher) != 2 {
		t.Fatalf("%d Zeilen nach dem Loeschen, erwartet weiterhin 2 — die Abschrift muss bleiben", len(nachher))
	}
}
