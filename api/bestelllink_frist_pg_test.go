package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/pkg/bestelllink"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die unter „Bestellwesen" eingestellte Frist kommt an beiden Stellen an, die einen
// Bestätigungs-Link erzeugen: beim Bestellen und bei „Neuen Link erzeugen". Gemessen wird am
// Ablauf in der Datenbank, in Tagen ab jetzt.

// setzeLinkFrist hinterlegt die Frist direkt in der Tabelle und nimmt sie nach dem Test weg.
func setzeLinkFrist(t *testing.T, pool *pgxpool.Pool, tage string) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO system_einstellungen (schluessel, wert) VALUES ('bestelllink_gueltigkeit_tage', $1)
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, tage); err != nil {
		t.Fatalf("Frist setzen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx,
			`DELETE FROM system_einstellungen WHERE schluessel = 'bestelllink_gueltigkeit_tage'`); err != nil {
			t.Errorf("Frist entfernen: %v", err)
		}
	})
}

// linkTageAbJetzt liest, in wie vielen Tagen der Link der Bestellung abläuft.
func linkTageAbJetzt(t *testing.T, pool *pgxpool.Pool, bestellungID string) int {
	t.Helper()
	var tage int
	if err := pool.QueryRow(context.Background(), `
		SELECT round(extract(epoch FROM token_gueltig_bis - now()) / 86400)::int
		FROM bestellungen_verlauf WHERE id = $1`, bestellungID).Scan(&tage); err != nil {
		t.Fatalf("Ablauf des Links lesen: %v", err)
	}
	return tage
}

func TestBestelllinkFrist_BeimBestellenGiltDieEinstellung(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	svc := NewOrderService(&db.Database{Pool: pool}, repository.NewBookRepository(pool))
	lieferant := haendler(t, pool, "Naacher", true)
	titel := titelMitMeldebestand(t, pool, "LMF-Frist", 0)
	bestelle := func() *OrderResult {
		t.Helper()
		res, err := svc.ProcessOrder(ctx, SubmitOrderRequest{
			Mittel:     mitteltopf.Land,
			SupplierID: lieferant,
			Items:      []OrderItemRequest{{TitelID: titel, Menge: 1, Preis: 10, GenerateBarcodes: true}},
		})
		if err != nil {
			t.Fatalf("Bestellung: %v", err)
		}
		return res
	}

	ohne := bestelle()
	if tage := linkTageAbJetzt(t, pool, ohne.BestellungID); tage != bestelllink.VorgabeTage {
		t.Errorf("ohne Einstellung läuft der Link in %d Tagen ab, erwartet die Vorgabe %d", tage, bestelllink.VorgabeTage)
	}

	setzeLinkFrist(t, pool, "45")
	mit := bestelle()
	if tage := linkTageAbJetzt(t, pool, mit.BestellungID); tage != 45 {
		t.Errorf("eingestellt sind 45 Tage, der Link läuft in %d Tagen ab", tage)
	}
	// Die Mail nennt den Ablauf aus dem Ergebnis; er ist derselbe wie in der Datenbank.
	if mit.LinkGueltigBis == nil {
		t.Fatal("das Ergebnis der Bestellung nennt keinen Ablauf des Links")
	}
	if tage := int(time.Until(*mit.LinkGueltigBis).Round(24*time.Hour) / (24 * time.Hour)); tage != 45 {
		t.Errorf("das Ergebnis der Bestellung nennt einen Ablauf in %d Tagen, erwartet 45", tage)
	}
}

func TestBestelllinkFrist_NeuerLinkBekommtDieEingestellteFrist(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	lieferant := haendler(t, pool, "Naacher", true)
	// Der alte Link läuft in drei Tagen ab; der neue bekommt die eingestellte Frist ab heute.
	bestellungID, _ := bestellungMitToken(t, pool, lieferant, 3)
	setzeOeffentlicheAdresse(t, pool, "https://bib.example.invalid")
	t.Cleanup(func() { setzeOeffentlicheAdresse(t, pool, "") })
	setzeLinkFrist(t, pool, "45")

	req := httptest.NewRequest(http.MethodPut, "/api/bestellungen/"+bestellungID+"/bestaetigungs-link", nil)
	req.SetPathValue("id", bestellungID)
	rec := httptest.NewRecorder()
	srv.NeuerBestaetigungsLinkHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	if tage := linkTageAbJetzt(t, pool, bestellungID); tage != 45 {
		t.Errorf("eingestellt sind 45 Tage, der neue Link läuft in %d Tagen ab", tage)
	}
	var antwort struct {
		GueltigBis time.Time `json:"gueltig_bis"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	if tage := int(time.Until(antwort.GueltigBis).Round(24*time.Hour) / (24 * time.Hour)); tage != 45 {
		t.Errorf("die Antwort nennt einen Ablauf in %d Tagen, erwartet 45", tage)
	}
}
