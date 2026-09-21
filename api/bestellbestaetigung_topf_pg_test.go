package api

import (
	"context"
	"net/http"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Der Topf auf der Bestätigungsseite und an der Etiketten-Tür (OFFEN.md 4.11, entschieden
// am 21.09.2026) — am echten Weg: Bestellung → Token → HTTP-Aufruf.
//
// Der Händler bekommt am selben Tag zwei gleich aussehende Links, und bis hierher bot auch
// die Bestellung für die Schülerbücherei das große Lernmittel-Etikett „Eigentum des Landes"
// an. Die Seite nennt jetzt den Topf, und die TÜR verweigert das große Etikett — ein
// versteckter Knopf allein ließe die Adresse offen.
func TestLieferantenSeite_TopfUndGrossesEtikett(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	faelle := []struct {
		name string
		// mittel der Bestellung; "" heißt: nach dem Anlegen auf NULL gesetzt (Alt-Bestellung
		// ohne Zuordnung, Migration 109).
		mittel     string
		wantWort   string
		wantGross  bool
		wantStatus int
	}{
		{"Lernmittelfreiheit", repository.MittelLand, "Lernmittelfreiheit", true, http.StatusOK},
		{"Schülerbücherei", repository.MittelSchultraeger, "Schülerbücherei", false, http.StatusNotFound},
		{"Alt-Bestellung ohne Zuordnung", "", "", true, http.StatusOK},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			resetBestandsdaten(t, pool)
			srv := &Server{DB: &db.Database{Pool: pool}}

			anlegenMit := f.mittel
			if anlegenMit == "" {
				anlegenMit = repository.MittelLand
			}
			token := bestellungMitEtikettenAus(t, srv, pool, 3, anlegenMit)
			bestellungID, err := srv.bestellungPerToken(ctx, token)
			if err != nil {
				t.Fatalf("Token auflösen: %v", err)
			}
			if f.mittel == "" {
				if _, err := pool.Exec(ctx,
					`UPDATE bestellungen_verlauf SET mittel = NULL WHERE id = $1`, bestellungID); err != nil {
					t.Fatalf("mittel auf NULL setzen: %v", err)
				}
			}

			ansicht, err := srv.ladeOeffentlicheBestellung(ctx, bestellungID)
			if err != nil {
				t.Fatalf("Ansicht laden: %v", err)
			}
			if ansicht.Mittel != f.wantWort {
				t.Errorf("Seite nennt den Topf %q, erwartet %q", ansicht.Mittel, f.wantWort)
			}
			if ansicht.GrossesEtikett != f.wantGross {
				t.Errorf("Seite bietet das große Etikett an = %v, erwartet %v", ansicht.GrossesEtikett, f.wantGross)
			}

			if rec := etikettenBogenHolen(t, srv, token, "gross", ""); rec.Code != f.wantStatus {
				t.Errorf("Tür: großes Etikett liefert Status %d, erwartet %d", rec.Code, f.wantStatus)
			}
			// Der kleine Bogen geht in jedem Fall.
			if rec := etikettenBogenHolen(t, srv, token, "klein", ""); rec.Code != http.StatusOK {
				t.Errorf("Tür: kleiner Bogen liefert Status %d, erwartet 200", rec.Code)
			}
		})
	}
}
