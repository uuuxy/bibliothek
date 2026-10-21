package testdaten

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LeereBescheide leert Bescheide und Nummernkreis. LeereBestandsdaten nimmt beide nicht mit:
// Der Bescheid hängt am Schüler mit ON DELETE SET NULL, und der Zähler der Nummern steht für
// sich.
func LeereBescheide(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`TRUNCATE schadensersatz_bescheide, schadensersatz_nummern CASCADE`); err != nil {
		t.Fatalf("Bescheid-Reset: %v", err)
	}
}

// SetzeBescheidAngaben hinterlegt die Pflichtangaben, ohne die kein Bescheid entsteht, und
// entfernt sie am Ende des Tests.
func SetzeBescheidAngaben(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	werte := map[string]string{
		"bescheid_bereich_nr":   "5830",
		"bescheid_schulnummer":  "1234",
		"bescheid_aufsicht":     "Staatliches Schulamt, Musterstraße 1, 12345 Musterstadt",
		"bescheid_schulleitung": "Dr. Beispiel, Schulleitung",
		"schule_name":           "Testschule",
		"schule_strasse":        "Schulweg 1",
		"schule_plz":            "12345",
		"schule_ort":            "Musterstadt",
	}
	for k, v := range werte {
		if _, err := pool.Exec(ctx, `
			INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)
			ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, k, v); err != nil {
			t.Fatalf("Einstellung %s: %v", k, err)
		}
	}
	t.Cleanup(func() {
		for k := range werte {
			if _, err := pool.Exec(context.Background(),
				`DELETE FROM system_einstellungen WHERE schluessel = $1`, k); err != nil {
				t.Logf("Aufräumen %s: %v", k, err)
			}
		}
	})
}

// LegeLernmittelAn legt einen Titel als Lernmittel an: Nur Forderungen zu Lernmitteln gehören
// auf einen Bescheid des Landes.
func LegeLernmittelAn(t *testing.T, pool *pgxpool.Pool, titel string) string {
	t.Helper()
	id := LegeTitelMitCoverUndAlterAn(t, pool, titel, "Autor", true, 0)
	if _, err := pool.Exec(context.Background(), `UPDATE buecher_titel SET ist_lernmittel = true WHERE id = $1`, id); err != nil {
		t.Fatalf("Lernmittel setzen: %v", err)
	}
	return id
}

// LegeForderungAn legt eine offene Forderung an und liefert ihre Kennung.
func LegeForderungAn(t *testing.T, pool *pgxpool.Pool, schuelerID, exemplarID, art, beschreibung string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, art)
		VALUES ($1, $2, $3, 0, $4) RETURNING id`,
		exemplarID, schuelerID, beschreibung, art).Scan(&id); err != nil {
		t.Fatalf("Forderung anlegen: %v", err)
	}
	return id
}

// BescheidRumpf baut den Rumpf der Anfrage, die einen Bescheid über die genannten Forderungen
// anlegt.
func BescheidRumpf(frist string, betraege map[string]float64) string {
	teile := make([]string, 0, len(betraege))
	for id, betrag := range betraege {
		teile = append(teile, fmt.Sprintf(`{"schadensfall_id":%q,"betrag":%.2f}`, id, betrag))
	}
	return fmt.Sprintf(`{"mittel":"land","frist_bis":%q,"positionen":[%s]}`, frist, strings.Join(teile, ","))
}

// In28Tagen ist der Kalendertag in 28 Tagen, als Frist eines Bescheids.
func In28Tagen() string { return schulzeit.Jetzt().AddDate(0, 0, 28).Format("2006-01-02") }

// LegeOffenenSchadenAn hängt dem Schüler einen unbezahlten Schadensfall an.
func LegeOffenenSchadenAn(t *testing.T, pool *pgxpool.Pool, schuelerID, barcodePraefix string) {
	t.Helper()
	titelID := LegeTitelAn(t, pool, "SchadenTitel-"+barcodePraefix, 1)
	exemplarID := LegeExemplarAn(t, pool, titelID, "SEX-"+barcodePraefix, true, "")
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, ist_bezahlt)
		 VALUES ($1, $2, 'Wasserschaden', 12.50, false)`, exemplarID, schuelerID); err != nil {
		t.Fatalf("offenen Schaden anlegen: %v", err)
	}
}
