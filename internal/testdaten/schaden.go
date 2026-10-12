package testdaten

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BescheidReset leert Bescheide und Nummernkreis. ResetBestandsdaten nimmt beide nicht mit: Der
// Bescheid hängt am Schüler mit ON DELETE SET NULL, und der Zähler der Nummern steht für sich.
func BescheidReset(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`TRUNCATE schadensersatz_bescheide, schadensersatz_nummern CASCADE`); err != nil {
		t.Fatalf("Bescheid-Reset: %v", err)
	}
}

// BescheidAngabenSetzen hinterlegt die Pflichtangaben, ohne die kein Bescheid entsteht, und
// entfernt sie am Ende des Tests.
func BescheidAngabenSetzen(t *testing.T, pool *pgxpool.Pool) {
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

// BescheidLernmittel legt einen Lernmittel-Titel an: Nur deren Forderungen gehören auf einen
// Bescheid des Landes.
func BescheidLernmittel(t *testing.T, pool *pgxpool.Pool, titel string) string {
	t.Helper()
	id := SeedMonitorTitel(t, pool, titel, "Autor", true, 0)
	if _, err := pool.Exec(context.Background(), `UPDATE buecher_titel SET ist_lernmittel = true WHERE id = $1`, id); err != nil {
		t.Fatalf("Lernmittel setzen: %v", err)
	}
	return id
}

// BescheidForderung legt eine offene Forderung an und liefert ihre Kennung.
func BescheidForderung(t *testing.T, pool *pgxpool.Pool, schuelerID, exemplarID, art, beschreibung string) string {
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

// BescheidRumpf baut den Rumpf der Anfrage für die genannten Forderungen.
func BescheidRumpf(frist string, betraege map[string]float64) string {
	teile := make([]string, 0, len(betraege))
	for id, betrag := range betraege {
		teile = append(teile, fmt.Sprintf(`{"schadensfall_id":%q,"betrag":%.2f}`, id, betrag))
	}
	return fmt.Sprintf(`{"mittel":"land","frist_bis":%q,"positionen":[%s]}`, frist, strings.Join(teile, ","))
}

// In28Tagen ist der Kalendertag in 28 Tagen, als Frist eines Bescheids.
func In28Tagen() string { return time.Now().AddDate(0, 0, 28).Format("2006-01-02") }

// SeedOffenerSchaden hängt dem Schüler einen unbezahlten Schadensfall an.
func SeedOffenerSchaden(t *testing.T, pool *pgxpool.Pool, schuelerID, barcodePraefix string) {
	t.Helper()
	titelID := TitelMitMeldebestand(t, pool, "SchadenTitel-"+barcodePraefix, 1)
	exID := Exemplar(t, pool, titelID, "SEX-"+barcodePraefix, true, "")
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, ist_bezahlt)
		 VALUES ($1, $2, 'Wasserschaden', 12.50, false)`, exID, schuelerID); err != nil {
		t.Fatalf("offenen Schaden anlegen: %v", err)
	}
}
