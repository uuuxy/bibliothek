package leser

import (
	"context"
	"net/http"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/repository"
)

// Das endgültige Löschen lehnt die Tür mit 409 ab, solange der Leser nicht im Papierkorb liegt,
// ein Buch draußen hat oder eine Forderung offen ist. Der Satz nennt den Grund und bei Ausleihen
// und Forderungen die Zahl; daran sieht die Bibliothek, was vor dem Löschen zu erledigen ist.
func TestLeserEndgueltigLoeschen_MeldungNenntDenGrund(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	// legeLeserAn legt einen Leser an, auf Wunsch im Papierkorb, mit einem Exemplar für die
	// Ausleihe oder den Schaden des Falls.
	legeLeserAn := func(t *testing.T, imPapierkorb bool) (leserID, exemplarID string) {
		t.Helper()
		testdaten.LeereBestandsdaten(t, pool)
		leserID = testdaten.LegeSchuelerAn(t, pool, "S-ENDGUELTIG-1", "Paula", "09B")
		if imPapierkorb {
			if _, err := pool.Exec(ctx, `UPDATE leser SET deleted_at = now() WHERE id = $1`, leserID); err != nil {
				t.Fatalf("in den Papierkorb legen: %v", err)
			}
		}
		titelID := testdaten.LegeTitelAn(t, pool, "Papierkorbtitel", 0)
		exemplarID = testdaten.LegeExemplarAn(t, pool, titelID, "B-ENDGUELTIG-1", true, "")
		return leserID, exemplarID
	}
	loesche := func(t *testing.T, leserID string) (status int, meldung string) {
		t.Helper()
		rec := tuertest.RufeAlsAdmin(t, "00000000-0000-0000-0000-0000000000ad",
			srv.LeserEndgueltigLoeschen(repository.NewProtokollAbfragen(pool)),
			http.MethodDelete, "/api/schueler/"+leserID+"/endgueltig", "", map[string]string{"id": leserID})
		return rec.Code, tuertest.Fehlermeldung(t, rec)
	}

	t.Run("nicht im Papierkorb", func(t *testing.T) {
		leserID, _ := legeLeserAn(t, false)
		status, meldung := loesche(t, leserID)
		const erwartet = "Endgültiges Löschen blockiert: Der Datensatz liegt nicht im Papierkorb. Erst löschen, dann endgültig entfernen."
		if status != http.StatusConflict || meldung != erwartet {
			t.Errorf("LeserEndgueltigLoeschen = %d %q, erwartet 409 %q", status, meldung, erwartet)
		}
	})
	t.Run("offene Ausleihe", func(t *testing.T) {
		leserID, exemplarID := legeLeserAn(t, true)
		testdaten.LegeAusleiheVorZweiTagenAn(t, pool, exemplarID, leserID)
		status, meldung := loesche(t, leserID)
		const erwartet = "Endgültiges Löschen blockiert: 1 offene Ausleihe"
		if status != http.StatusConflict || meldung != erwartet {
			t.Errorf("LeserEndgueltigLoeschen = %d %q, erwartet 409 %q", status, meldung, erwartet)
		}
	})
	t.Run("unbezahlte Forderung", func(t *testing.T) {
		leserID, exemplarID := legeLeserAn(t, true)
		if _, err := pool.Exec(ctx, `
			INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, ist_bezahlt)
			VALUES ($1, $2, 'Einband gerissen', 12.50, false)`, exemplarID, leserID); err != nil {
			t.Fatalf("Schadensfall anlegen: %v", err)
		}
		status, meldung := loesche(t, leserID)
		const erwartet = "Endgültiges Löschen blockiert: 1 unbezahlter Schadensfall"
		if status != http.StatusConflict || meldung != erwartet {
			t.Errorf("LeserEndgueltigLoeschen = %d %q, erwartet 409 %q", status, meldung, erwartet)
		}
	})
}
