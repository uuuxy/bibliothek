package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Eigentumsvermerk folgt dem Topf des Exemplars — an allen vier Wegen, die
// Etikettendaten bauen, geprüft am fertigen PDF (OFFEN.md 4.21, entschieden am 21.09.2026).
//
// Die Regel (repository.ExemplarTopfSQL): zuerst die Zuordnung der Bestellung, und wo es
// keine gibt, ist_lernmittel am Titel. Bis zum 21.09.2026 trug jedes Etikett ab 30 mm den
// EINEN Vermerk „Eigentum des Landes Hessen" — auch das Buch der Schülerbücherei, das der
// Schulträger bezahlt hat.

const (
	vermerkLand  = "Eigentum des Landes Hessen" // Werksvorgabe, nichts hinterlegt
	vermerkStadt = "Eigentum der Stadt Friedrichsdorf"
	// 70 × 36 mm: ab 30 mm Höhe steht der Eigentumsvermerk auf dem kleinen Etikett.
	topfTestFormat = "avery_3475"
)

// setzeVermerkSchuelerbuecherei hinterlegt die zweite Einstellung direkt in der Tabelle.
func setzeVermerkSchuelerbuecherei(t *testing.T, pool *pgxpool.Pool, wert string) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO system_einstellungen (schluessel, wert)
		VALUES ('etikett_eigentumsvermerk_schuelerbuecherei', $1)
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, wert); err != nil {
		t.Fatalf("Einstellung setzen: %v", err)
	}
	t.Cleanup(func() {
		// Ein liegen gebliebener Wert färbte jeden späteren Etiketten-Test.
		if _, err := pool.Exec(ctx, `DELETE FROM system_einstellungen
			WHERE schluessel = 'etikett_eigentumsvermerk_schuelerbuecherei'`); err != nil {
			t.Errorf("Einstellung zurücknehmen: %v", err)
		}
	})
}

// etikettenUeberBuchformular druckt über GET /api/buecher/titel/{id}/etiketten (queryLabelItems).
func etikettenUeberBuchformular(t *testing.T, srv *Server, titelID string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/buecher/titel/x/etiketten?format="+topfTestFormat+"&start=1", nil)
	req.SetPathValue("id", titelID)
	rec := httptest.NewRecorder()
	srv.LabelsHandler()(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Buchformular-Weg: Status %d — %s", rec.Code, rec.Body.String())
	}
	return pdfText(t, rec.Body.Bytes())
}

// etikettenUeberDruckCenter druckt über POST /api/print/labels (ergaenzeServerfelder). Die
// Nutzlast trägt zusätzlich ein Feld „Topf" mit dem FALSCHEN Wert: Der Topf kommt vom
// Server, und ein Druckauftrag aus dem Browser darf ihn nicht mitbringen können.
func etikettenUeberDruckCenter(t *testing.T, srv *Server, barcode, untergeschobenerTopf string) string {
	t.Helper()
	payload := `{"formatId": "` + topfTestFormat + `", "startPosition": 1, "isQR": false,
		"items": [{"BarcodeID": "` + barcode + `", "Titel": "Probe", "Autor": "", "Topf": "` + untergeschobenerTopf + `"}]}`
	rec := httptest.NewRecorder()
	srv.PrintLabelsHandler()(rec, httptest.NewRequest(http.MethodPost, "/api/print/labels", strings.NewReader(payload)))
	if rec.Code != http.StatusOK {
		t.Fatalf("Druck-Center-Weg: Status %d — %s", rec.Code, rec.Body.String())
	}
	return pdfText(t, rec.Body.Bytes())
}

func pruefeVermerk(t *testing.T, weg, text, want, nicht string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Errorf("%s: Vermerk %q fehlt auf dem Etikett", weg, want)
	}
	if strings.Contains(text, nicht) {
		t.Errorf("%s: %q steht auf dem Etikett und gehört dort nicht hin", weg, nicht)
	}
}

// Die Regel an den beiden Wegen des Hauses — alle fünf Formen, die ein Exemplar haben kann.
func TestEigentumsvermerkFolgtDemTopf_BuchformularUndDruckCenter(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	faelle := []struct {
		name string
		// bestellt: Hängt das Exemplar an einer Bestellung? mittel "" heißt dann NULL
		// (Alt-Bestellung ohne Zuordnung).
		bestellt      bool
		mittel        string
		istLernmittel bool
		want, nicht   string
	}{
		{"Altbestand, Lernmittel", false, "", true, vermerkLand, vermerkStadt},
		{"Altbestand, kein Lernmittel", false, "", false, vermerkStadt, vermerkLand},
		// Die Bestellung schlägt den Titel — das Eigentum folgt dem Geld.
		{"Bestellung Schülerbücherei, Titel ist Lernmittel", true, repository.MittelSchultraeger, true, vermerkStadt, vermerkLand},
		{"Bestellung Lernmittelfreiheit, Titel ist keins", true, repository.MittelLand, false, vermerkLand, vermerkStadt},
		{"Alt-Bestellung ohne Zuordnung, Lernmittel", true, "", true, vermerkLand, vermerkStadt},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			resetBestandsdaten(t, pool)
			setzeVermerkSchuelerbuecherei(t, pool, vermerkStadt)

			var titelID string
			if err := pool.QueryRow(ctx,
				`INSERT INTO buecher_titel (titel, ist_lernmittel) VALUES ('Topfprobe', $1) RETURNING id`,
				f.istLernmittel).Scan(&titelID); err != nil {
				t.Fatalf("Titel anlegen: %v", err)
			}
			var bestellungID *string
			if f.bestellt {
				lieferant := haendler(t, pool, "Topf-Händler", false)
				var id string
				if err := pool.QueryRow(ctx, `
					INSERT INTO bestellungen_verlauf
						(lieferant_id, lieferant_name, lieferant_email, kundennummer, anzahl_exemplare, mittel)
					VALUES ($1, 'Topf-Händler', 'topf@example.invalid', 'K-1', 1, NULLIF($2, ''))
					RETURNING id`, lieferant, f.mittel).Scan(&id); err != nil {
					t.Fatalf("Bestellung anlegen: %v", err)
				}
				bestellungID = &id
			}
			if _, err := pool.Exec(ctx,
				`INSERT INTO buecher_exemplare (titel_id, barcode_id, bestellung_id) VALUES ($1, 'B-TOPF-1', $2)`,
				titelID, bestellungID); err != nil {
				t.Fatalf("Exemplar anlegen: %v", err)
			}

			pruefeVermerk(t, "Buchformular", etikettenUeberBuchformular(t, srv, titelID), f.want, f.nicht)

			// Dem Druck-Center wird der jeweils ANDERE Topf untergeschoben.
			falsch := repository.MittelSchultraeger
			if f.want == vermerkStadt {
				falsch = repository.MittelLand
			}
			pruefeVermerk(t, "Druck-Center", etikettenUeberDruckCenter(t, srv, "B-TOPF-1", falsch), f.want, f.nicht)
		})
	}
}

// Die beiden Wege zum Händler: der Lieferanten-Link (ladeBestellEtiketten) und der
// Mailanhang (Etiketten aus ProcessOrder). Bestellt wird ein LERNMITTEL-Titel für die
// Schülerbücherei — der Fall, in dem Titel und Bestellung einander widersprechen.
func TestEigentumsvermerkFolgtDemTopf_LieferantenLinkUndMailanhang(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	setzeVermerkSchuelerbuecherei(t, pool, vermerkStadt)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	svc := NewOrderService(srv.DB, repository.NewBookRepository(pool))
	res, err := svc.ProcessOrder(ctx, SubmitOrderRequest{
		Mittel:     repository.MittelSchultraeger,
		SupplierID: haendler(t, pool, "Naacher", true),
		Items: []OrderItemRequest{{
			TitelID: titelMitMeldebestand(t, pool, "LMF-Topfprobe", 0), Menge: 2, Preis: 10, GenerateBarcodes: true}},
	})
	if err != nil {
		t.Fatalf("Bestellung: %v", err)
	}

	// Weg 3: der Link. Das große Etikett gibt es zu dieser Bestellung nicht (4.11).
	rec := etikettenBogenHolen(t, srv, res.BestaetigungsToken, "klein", topfTestFormat)
	if rec.Code != http.StatusOK {
		t.Fatalf("Lieferanten-Link: Status %d — %s", rec.Code, rec.Body.String())
	}
	pruefeVermerk(t, "Lieferanten-Link", pdfText(t, rec.Body.Bytes()), vermerkStadt, vermerkLand)

	// Weg 4: der Mailanhang, mit dem Kopf, den der Bestell-Handler baut.
	einstellungen, err := repository.NewSystemSettingsRepository(pool).GetSettings(ctx)
	if err != nil {
		t.Fatalf("Einstellungen lesen: %v", err)
	}
	boegen, err := etikettenboegen(res.Labels, etikettKopfAus(einstellungen), true, res.Mittel)
	if err != nil {
		t.Fatalf("etikettenboegen: %v", err)
	}
	if len(boegen) == 0 {
		t.Fatal("kein Bogen im Anhang")
	}
	for _, b := range boegen {
		pruefeVermerk(t, "Mailanhang "+b.Name, pdfText(t, b.Data), vermerkStadt, vermerkLand)
	}
}

// Der Topf kommt vom Server (json:"-" an BarcodeLabelDetail.Topf). Bei einem Barcode, den
// die Datenbank nicht kennt — Vorab-Druck —, trägt ergaenzeServerfelder nichts nach; ein
// vom Browser mitgeschicktes „Topf" bliebe dann stehen und wählte den Vermerk.
func TestEigentumsvermerk_TopfLaesstSichNichtUnterschieben(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	setzeVermerkSchuelerbuecherei(t, pool, vermerkStadt)
	srv := &Server{DB: &db.Database{Pool: pool}}

	text := etikettenUeberDruckCenter(t, srv, "B-GIBT-ES-NICHT", repository.MittelSchultraeger)
	pruefeVermerk(t, "Druck-Center, unbekannter Barcode", text, vermerkLand, vermerkStadt)
}
