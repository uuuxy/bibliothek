package repository

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
)

// Migration 147 nimmt Einträgen im Admin-Protokoll, deren Leser es nicht mehr gibt, die fünf
// Schlüssel, die die Tilgung entfernt (repository.spurTilgungen, Anweisung „audit_logs").
// Gemessen am Testserver am 25.09.2026: 3 solche Einträge mit grund oder reason — Leser,
// die vor 5b50202d endgültig gelöscht wurden; die nächtliche Nachbereinigung erreicht sie
// nicht, weil sie über anonymized_at vorhandener Zeilen sucht.
//
// Geprüft wird die echte Migrationsdatei, zweimal hintereinander, in einer Transaktion, die
// am Ende zurückgerollt wird. Was nicht verwaist ist, bleibt, wie es war: ein vorhandener
// Leser (auch in Großbuchstaben genannt), ein anonymisierter (den räumt der Nachtlauf) und
// ein Eintrag ohne Leser-Kennung.
func TestProtokollVerwaisteLeser_Migration147(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	leser := func(barcode string) string {
		t.Helper()
		var id string
		if err := tx.QueryRow(ctx,
			`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			 VALUES ($1, 'Probe', 'Leser', '10R', 2026) RETURNING id::text`, barcode).Scan(&id); err != nil {
			t.Fatalf("Leser %s anlegen: %v", barcode, err)
		}
		return id
	}
	eintrag := func(details map[string]any) string {
		t.Helper()
		roh, err := json.Marshal(details)
		if err != nil {
			t.Fatal(err)
		}
		var id string
		if err := tx.QueryRow(ctx,
			`INSERT INTO audit_logs (aktion, details) VALUES ('OVERRIDE_BLOCK', $1::jsonb) RETURNING id::text`,
			string(roh)).Scan(&id); err != nil {
			t.Fatalf("Protokolleintrag anlegen: %v", err)
		}
		return id
	}
	schluessel := func(id string) map[string]any {
		t.Helper()
		var roh []byte
		if err := tx.QueryRow(ctx, `SELECT details FROM audit_logs WHERE id::text = $1`, id).Scan(&roh); err != nil {
			t.Fatalf("Protokolleintrag %s lesen: %v", id, err)
		}
		var d map[string]any
		if err := json.Unmarshal(roh, &d); err != nil {
			t.Fatal(err)
		}
		return d
	}

	vorhanden := leser("M147-VORHANDEN")
	anonym := leser("M147-ANONYM")
	if _, err := tx.Exec(ctx, `UPDATE schueler SET anonymized_at = now() WHERE id = $1::uuid`, anonym); err != nil {
		t.Fatalf("anonymisieren: %v", err)
	}
	const verwaist = "3f2c9a0e-7b1d-4c5e-9a8f-0d6b2e4c1a77" // eine Kennung ohne Leser
	pii := func(kennung string) map[string]any {
		return map[string]any{
			"schueler_id": kennung, "lusd_id": "LUSD-PROBE", "barcode": "A-PROBE",
			"aufgeloest_barcode": "S-PROBE", "grund": "PROBE nennt Dritte", "reason": "PROBE übergangen",
			"quelle": "theke",
		}
	}
	idVerwaist := eintrag(pii(verwaist))
	idVorhanden := eintrag(pii(vorhanden))
	idGross := eintrag(pii(strings.ToUpper(vorhanden)))
	idAnonym := eintrag(pii(anonym))
	idOhneLeser := eintrag(map[string]any{"reason": "PROBE ohne Leser"})

	getilgt := []string{"lusd_id", "barcode", "aufgeloest_barcode", "grund", "reason"}
	// Positivkontrolle: Ohne diesen Eintrag wäre die Prüfung unten grün, ohne etwas zu prüfen.
	if d := schluessel(idVerwaist); d["grund"] == nil || d["lusd_id"] == nil {
		t.Fatalf("Positivkontrolle: der verwaiste Eintrag trägt die Schlüssel nicht: %v", d)
	}

	migration, err := os.ReadFile(filepath.Join("..", "migrations", "147_protokoll_verwaiste_leser.sql"))
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	for lauf := 1; lauf <= 2; lauf++ {
		if _, err := tx.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("Lauf %d: Migration scheitert: %v", lauf, err)
		}
	}

	d := schluessel(idVerwaist)
	for _, s := range getilgt {
		if _, da := d[s]; da {
			t.Errorf("Eintrag zu einer Kennung ohne Leser trägt nach der Migration noch %q: %v", s, d)
		}
	}
	// Der Eintrag selbst bleibt: Wer wann was getan hat, ist Rechenschaft.
	if d["schueler_id"] != verwaist || d["quelle"] != "theke" {
		t.Errorf("Die Migration hat mehr genommen als die fünf Schlüssel: %v", d)
	}
	for name, id := range map[string]string{
		"vorhandener Leser":                   idVorhanden,
		"vorhandener Leser in Großbuchstaben": idGross,
		"anonymisierter Leser (Nachtlauf)":    idAnonym,
	} {
		d := schluessel(id)
		for _, s := range getilgt {
			if _, da := d[s]; !da {
				t.Errorf("%s: %q fehlt nach der Migration — sie darf nur Einträge ohne Leser anfassen", name, s)
			}
		}
	}
	if d := schluessel(idOhneLeser); d["reason"] == nil {
		t.Errorf("Eintrag ohne Leser-Kennung verlor seinen Grund: %v", d)
	}
}
