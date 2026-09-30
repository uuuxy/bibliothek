package service

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Seit Migration 154 speichert die Datenbank Titeltexte zusammengesetzt (NFC,
// trg_titel_text_nfc). Der Listenimport erkennt einen vorhandenen Titel ohne ISBN an seinem
// Text (repository.NormalisiereTitelKey). Kommt derselbe Titel in der Datei zerlegt an
// („a" + U+0308, so liefert die DNB Umlaute), muss er den gespeicherten treffen; sonst legt der
// Import ihn ein zweites Mal an (Rasterdurchgang vom 30.09.2026). Die zerlegten Zeichen stehen
// als Escape, damit kein Editor sie still zusammensetzt.
func TestImportDynamic_ZerlegterTitelTrifftGespeicherten(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const barcode = "NFC-PROBE-154-1"

	var id string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ($1) RETURNING id::text`,
		"NFC-Probe Anh\u00e4nge und Register").Scan(&id); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	t.Cleanup(func() {
		// Über die Kennungen: der eigene Titel und, falls der Import doch einen neuen anlegte,
		// der Titel am Probe-Exemplar.
		var anExemplar string
		if err := pool.QueryRow(ctx, `SELECT coalesce((SELECT titel_id::text FROM buecher_exemplare
			WHERE barcode_id = $1), '')`, barcode).Scan(&anExemplar); err != nil {
			t.Errorf("aufräumen: Probe-Exemplar lesen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_exemplare WHERE barcode_id = $1`, barcode); err != nil {
			t.Errorf("aufräumen: Probe-Exemplar löschen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE id::text = ANY ($1)`,
			[]string{id, anExemplar}); err != nil {
			t.Errorf("aufräumen: Probe-Titel löschen: %v", err)
		}
	})

	rows := [][]string{
		{"Titel", "Barcode"},
		{"NFC-Probe Anha\u0308nge und Register", barcode},
	}
	neueTitel, neueExemplare, err := NewImportService(nil, pool).
		ImportDynamic(ctx, rows, map[string]int{"titel": 0, "barcode": 1})
	if err != nil {
		t.Fatalf("ImportDynamic: %v", err)
	}
	var anExemplar string
	if err := pool.QueryRow(ctx, `SELECT titel_id::text FROM buecher_exemplare WHERE barcode_id = $1`, barcode).
		Scan(&anExemplar); err != nil {
		t.Fatalf("Probe-Exemplar lesen: %v", err)
	}
	if neueTitel != 0 || neueExemplare != 1 || anExemplar != id {
		t.Errorf("zerlegter Titel: %d neue Titel, %d neue Exemplare, Exemplar am Titel %s — "+
			"erwartet 0, 1 und der gespeicherte Titel %s", neueTitel, neueExemplare, anExemplar, id)
	}
}
