package bestellung

import (
	"context"
	"net/http"
	"testing"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/pkg/mitteltopf"
	"bibliothek/internal/testdaten"
)

// Der Topf auf der Bestätigungsseite und an der Etiketten-Tür, am echten Weg: Bestellung →
// Token → HTTP-Aufruf.
//
// Der Händler bekommt am selben Tag mehrere gleich aussehende Links; die Seite nennt deshalb den
// Topf. Beide Größen gehen in jedem Topf: Wer druckt, wählt die Größe, denn auch ein Klassensatz
// der Schülerbücherei oder einer Spende kann das große Etikett brauchen.
func TestLieferantenSeite_TopfUndBeideGroessen(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	faelle := []struct {
		name string
		// mittel der Bestellung; "" heißt: nach dem Anlegen auf NULL gesetzt (Alt-Bestellung
		// ohne Zuordnung, Migration 109).
		mittel   string
		wantWort string
	}{
		{"Lernmittelfreiheit", mitteltopf.Land, "Lernmittelfreiheit"},
		{"Schülerbücherei", mitteltopf.Schultraeger, "Schülerbücherei"},
		{"Spende", mitteltopf.Spende, "Spende"},
		{"Alt-Bestellung ohne Zuordnung", "", ""},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			testdaten.LeereBestandsdaten(t, pool)
			srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

			anlegenMit := f.mittel
			if anlegenMit == "" {
				anlegenMit = mitteltopf.Land
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
			for _, groesse := range []string{"klein", "gross"} {
				if rec := etikettenBogenHolen(t, srv, token, groesse, ""); rec.Code != http.StatusOK {
					t.Errorf("Tür: Bogen %q liefert Status %d, erwartet 200 — %s", groesse, rec.Code, rec.Body.String())
				}
			}
		})
	}
}
