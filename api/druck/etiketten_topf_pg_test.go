package druck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pdftest"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/pkg/mitteltopf"
)

// Der Eigentumsvermerk folgt dem Topf des Exemplars — an allen vier Wegen, die
// Etikettendaten bauen, geprüft am fertigen PDF (OFFEN.md 4.21, entschieden am 21.09.2026).
// Hier stehen die zwei Wege des Hauses, Buchformular und Druck-Center; die zwei Wege zum
// Händler prüft api/bestellung/bestellung_etiketten_topf_pg_test.go.
//
// Die Regel (repository.ExemplarTopfSQL): zuerst die Zuordnung der Bestellung, und wo es
// keine gibt, ist_lernmittel am Titel. Bis zum 21.09.2026 trug jedes Etikett ab 30 mm den
// EINEN Vermerk „Eigentum des Landes Hessen" — auch das Buch der Schülerbücherei, das der
// Schulträger bezahlt hat.

// etikettenUeberBuchformular druckt über GET /api/buecher/titel/{id}/etiketten
// (buchetikett.DesTitels).
func etikettenUeberBuchformular(t *testing.T, srv *Tueren, titelID string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/buecher/titel/x/etiketten?format="+testdaten.EtikettFormatMitVermerk+"&start=1", nil)
	req.SetPathValue("id", titelID)
	rec := httptest.NewRecorder()
	srv.LabelsHandler()(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Buchformular-Weg: Status %d — %s", rec.Code, rec.Body.String())
	}
	return pdftest.Suchtext(t, rec.Body.Bytes())
}

// etikettenUeberDruckCenter druckt über POST /api/print/labels
// (buchetikett.ErgaenzeServerfelder). Die
// Nutzlast trägt zusätzlich ein Feld „Topf" mit dem FALSCHEN Wert: Der Topf kommt vom
// Server, und ein Druckauftrag aus dem Browser darf ihn nicht mitbringen können.
func etikettenUeberDruckCenter(t *testing.T, srv *Tueren, barcode, untergeschobenerTopf string) string {
	t.Helper()
	payload := `{"formatId": "` + testdaten.EtikettFormatMitVermerk + `", "startPosition": 1, "isQR": false,
		"items": [{"BarcodeID": "` + barcode + `", "Titel": "Probe", "Autor": "", "Topf": "` + untergeschobenerTopf + `"}]}`
	rec := httptest.NewRecorder()
	srv.PrintLabelsHandler()(rec, httptest.NewRequest(http.MethodPost, "/api/print/labels", strings.NewReader(payload)))
	if rec.Code != http.StatusOK {
		t.Fatalf("Druck-Center-Weg: Status %d — %s", rec.Code, rec.Body.String())
	}
	return pdftest.Suchtext(t, rec.Body.Bytes())
}

// Die Regel an den beiden Wegen des Hauses — alle sieben Formen, die ein Exemplar haben kann.
func TestEigentumsvermerkFolgtDemTopf_BuchformularUndDruckCenter(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}

	faelle := []struct {
		name string
		// bestellt: Hängt das Exemplar an einer Bestellung? mittel "" heißt dann NULL
		// (Alt-Bestellung ohne Zuordnung).
		bestellt      bool
		mittel        string
		istLernmittel bool
		// eigentum: am Exemplar ausdrücklich gesetzt (Migration 150), "" heißt NULL.
		eigentum    string
		want, nicht string
	}{
		{"Altbestand, Lernmittel", false, "", true, "", testdaten.VermerkLand, testdaten.VermerkStadt},
		{"Altbestand, kein Lernmittel", false, "", false, "", testdaten.VermerkStadt, testdaten.VermerkLand},
		// Die Bestellung schlägt den Titel — das Eigentum folgt dem Geld.
		{"Bestellung Schülerbücherei, Titel ist Lernmittel", true, mitteltopf.Schultraeger, true, "", testdaten.VermerkStadt, testdaten.VermerkLand},
		{"Bestellung Lernmittelfreiheit, Titel ist keins", true, mitteltopf.Land, false, "", testdaten.VermerkLand, testdaten.VermerkStadt},
		{"Alt-Bestellung ohne Zuordnung, Lernmittel", true, "", true, "", testdaten.VermerkLand, testdaten.VermerkStadt},
		// Das Eigentum am Exemplar schlägt beides (4.24): die Lektüre aus LMF-Mitteln, die
		// Littera dem Land zuschreibt, und ein ausdrücklich umgesetztes Exemplar einer Bestellung.
		{"Altbestand, kein Lernmittel, Eigentum Land", false, "", false, mitteltopf.Land, testdaten.VermerkLand, testdaten.VermerkStadt},
		{"Bestellung Lernmittelfreiheit, Eigentum Schulträger", true, mitteltopf.Land, true, mitteltopf.Schultraeger, testdaten.VermerkStadt, testdaten.VermerkLand},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			testdaten.ResetBestandsdaten(t, pool)
			testdaten.SetzeVermerkSchuelerbuecherei(t, pool, testdaten.VermerkStadt)

			var titelID string
			if err := pool.QueryRow(ctx,
				`INSERT INTO buecher_titel (titel, ist_lernmittel) VALUES ('Topfprobe', $1) RETURNING id`,
				f.istLernmittel).Scan(&titelID); err != nil {
				t.Fatalf("Titel anlegen: %v", err)
			}
			var bestellungID *string
			if f.bestellt {
				lieferant := testdaten.Haendler(t, pool, "Topf-Händler", false)
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
				`INSERT INTO buecher_exemplare (titel_id, barcode_id, bestellung_id, eigentum, eigentum_quelle)
				 VALUES ($1, 'B-TOPF-1', $2, NULLIF($3, ''), CASE WHEN $3 = '' THEN NULL ELSE 'hand' END)`,
				titelID, bestellungID, f.eigentum); err != nil {
				t.Fatalf("Exemplar anlegen: %v", err)
			}

			pdftest.Aufdruck(t, "Buchformular", etikettenUeberBuchformular(t, srv, titelID), f.want, f.nicht)

			// Dem Druck-Center wird der jeweils ANDERE Topf untergeschoben.
			falsch := mitteltopf.Schultraeger
			if f.want == testdaten.VermerkStadt {
				falsch = mitteltopf.Land
			}
			pdftest.Aufdruck(t, "Druck-Center", etikettenUeberDruckCenter(t, srv, "B-TOPF-1", falsch), f.want, f.nicht)
		})
	}
}

// Der Topf kommt vom Server (json:"-" an BarcodeLabelDetail.Topf). Bei einem Barcode, den
// die Datenbank nicht kennt — Vorab-Druck —, trägt buchetikett.ErgaenzeServerfelder nichts nach; ein
// vom Browser mitgeschicktes „Topf" bliebe dann stehen und wählte den Vermerk.
func TestEigentumsvermerk_TopfLaesstSichNichtUnterschieben(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.ResetBestandsdaten(t, pool)
	testdaten.SetzeVermerkSchuelerbuecherei(t, pool, testdaten.VermerkStadt)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}

	text := etikettenUeberDruckCenter(t, srv, "B-GIBT-ES-NICHT", mitteltopf.Schultraeger)
	pdftest.Aufdruck(t, "Druck-Center, unbekannter Barcode", text, testdaten.VermerkLand, testdaten.VermerkStadt)
}
