package bestellung

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/repository"
	"bibliothek/internal/testdaten"
)

// Ohne öffentliche Adresse führte ein Link ins Leere: Auch eine Bestellung im Bestätigungsweg
// bekommt beim erneuten Versand keinen, und an ihr wird kein Hash gespeichert.
func TestBaueBestellmailErneut_OhneAdresseKeinLink(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	if a := srv.oeffentlicheAdresse(ctx); a != "" {
		t.Fatalf("Voraussetzung: keine öffentliche Adresse eingestellt, gefunden %q", a)
	}

	haupt := testdaten.LegeLieferantAn(t, pool, fmt.Sprintf("Erneut-Haupt-%d", time.Now().UnixNano()), true)
	var id string
	if err := pool.QueryRow(ctx, `INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email, lieferant_id)
		VALUES ('Erneut', 'erneut@example.invalid', $1) RETURNING id`, haupt).Scan(&id); err != nil {
		t.Fatalf("Bestellung anlegen: %v", err)
	}

	daten, err := srv.baueBestellmailErneut(ctx, id, &repository.BestellmailAuftrag{Empfaenger: "erneut@example.invalid"})
	if err != nil {
		t.Fatalf("baueBestellmailErneut: %v", err)
	}
	if daten.Token != "" || daten.LinkGueltigBis != nil {
		t.Errorf("Link in der Mail ohne öffentliche Adresse: Token %q, gültig bis %v", daten.Token, daten.LinkGueltigBis)
	}
	var mitHash bool
	if err := pool.QueryRow(ctx, `SELECT bestaetigungs_token_hash IS NOT NULL FROM bestellungen_verlauf WHERE id = $1`, id).
		Scan(&mitHash); err != nil {
		t.Fatalf("Bestellung lesen: %v", err)
	}
	if mitHash {
		t.Error("die Bestellung hat ohne öffentliche Adresse einen Link-Hash bekommen")
	}
}
