package schaden

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"

	"github.com/google/uuid"
)

// Die Rechnung an die Eltern darf keine Forderung verlieren, weil ein Bezug fehlt.
// `schadensfaelle.exemplar_id` und `.ausleihe_id` sind nullbar (Letzteres steht auf ON DELETE
// SET NULL), und `geraet_id` wartet als dritte Bezugsspalte. Mit INNER JOINs fiele so eine
// Forderung lautlos aus dem Brief: Der Betrag wäre zu klein, oder es hieße „keine offenen
// Schadensfälle", während die Akte offene Beträge zeigt und der Schüler gesperrt bleibt.
func TestLadeRechnungPosten_ForderungOhneExemplarUndAusleihe(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	var schuelerID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 VALUES ('R-RECH', 'Rita', 'Rechnung', '07H1', 2030) RETURNING id`).Scan(&schuelerID); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	// Ein Geräteschaden: exemplar_id NULL, ausleihe_id NULL — genau die Form, die
	// check_damage_item ausdrücklich erlaubt und die INNER JOINs entginge.
	var geraetID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO geraete (modellname, barcode_id) VALUES ('iPad 9. Gen.', 'G-RECH-1') RETURNING id`).
		Scan(&geraetID); err != nil {
		t.Fatalf("Gerät anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO schadensfaelle (schueler_id, geraet_id, beschreibung, betrag)
		 VALUES ($1, $2, 'Kopfhörerbuchse abgebrochen', 24.90)`, schuelerID, geraetID); err != nil {
		t.Fatalf("Forderung ohne Exemplar/Ausleihe anlegen: %v", err)
	}

	items, err := ladeRechnungPosten(ctx, pool, schuelerID)
	if err != nil {
		t.Fatalf("Rechnungspositionen lesen: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("erwartet 1 Position, waren %d — eine Forderung ohne Exemplar/Ausleihe fällt "+
			"aus der Rechnung, und bei „alle betroffen\" antwortet der Weg mit 404", len(items))
	}
	if items[0].Titel != "iPad 9. Gen." {
		t.Errorf("Bezeichnung: erwartet den Modellnamen des Geräts, war %q", items[0].Titel)
	}
	if items[0].Barcode != "G-RECH-1" {
		t.Errorf("Barcode: erwartet den Geräte-Barcode, war %q", items[0].Barcode)
	}
	if items[0].Ersatzpreis != 24.90 {
		t.Errorf("Betrag: erwartet 24.90, war %v", items[0].Ersatzpreis)
	}
	if items[0].Ausleihdatum.IsZero() {
		t.Error("Ausleihdatum leer — ersatzweise gilt das Datum der Forderung")
	}
}

// Eine Forderung, die schon auf einem Schadensersatz-Bescheid steht, gehört nicht auf die
// Ersatzforderung. Der Bescheid verlangt die Überweisung mit Referenznummer aufs Konto des
// Landes, die Ersatzforderung „bar in der Bibliothek". Stünde dieselbe Forderung auf beiden
// Briefen, bekämen die Eltern zwei Zahlungsaufforderungen mit zwei verschiedenen Zahlungswegen.
// Der Bescheid entsteht hier über die echte Tür, damit der Test genau die Form trifft, die der
// Betrieb schreibt.
func TestLadeRechnungPosten_ForderungAufEinemBescheidBleibtDraussen(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	testdaten.LeereBescheide(t, pool)
	testdaten.SetzeBescheidAngaben(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	ctx := context.Background()

	sid := testdaten.LegeSchuelerAn(t, pool, "S-RECH-BESCH", "Rechnungskind", "08G2")
	titelID := testdaten.LegeLernmittelAn(t, pool, "Deutsch 8")
	exAufBescheid := testdaten.LegeExemplarAn(t, pool, titelID, "RECH-BESCH-1", true, "")
	exOhneBescheid := testdaten.LegeExemplarAn(t, pool, titelID, "RECH-BESCH-2", true, "")
	fAufBescheid := testdaten.LegeForderungAn(t, pool, sid, exAufBescheid, "nicht_zurueckgegeben", "Deutsch 8 nicht zurück")
	testdaten.LegeForderungAn(t, pool, sid, exOhneBescheid, "beschaedigt", "Deutsch 8 beschädigt")

	rec := bescheidErstellenUeberTuer(t, srv, pool, sid,
		testdaten.BescheidRumpf(testdaten.In28Tagen(), map[string]float64{fAufBescheid: 24.90}))
	if rec.Code != http.StatusCreated {
		t.Fatalf("Bescheid anlegen: Status %d, want 201: %s", rec.Code, rec.Body.String())
	}

	items, err := ladeRechnungPosten(ctx, pool, uuid.MustParse(sid))
	if err != nil {
		t.Fatalf("Rechnungspositionen lesen: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("erwartet 1 Position (die Forderung ohne Bescheid), waren %d — eine Forderung "+
			"auf einem Bescheid stünde zusätzlich mit „bar in der Bibliothek\" auf der Ersatzforderung", len(items))
	}
	if items[0].Barcode != "RECH-BESCH-2" {
		t.Errorf("Position: erwartet das Exemplar ohne Bescheid (RECH-BESCH-2), war %q", items[0].Barcode)
	}
}

// Stehen alle offenen Forderungen auf einem Bescheid, entsteht keine Ersatzforderung — und der
// Knopf in der Schülerakte sagt, warum, statt „keine offenen Schadensfälle" zu melden, während
// die Akte offene Beträge zeigt.
func TestRechnungPDF_AlleForderungenAufBescheid(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	testdaten.LeereBescheide(t, pool)
	testdaten.SetzeBescheidAngaben(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	sid := testdaten.LegeSchuelerAn(t, pool, "S-RECH-ALLE", "Bescheidkind", "08G2")
	titelID := testdaten.LegeLernmittelAn(t, pool, "Englisch 8")
	ex := testdaten.LegeExemplarAn(t, pool, titelID, "RECH-ALLE-1", true, "")
	f := testdaten.LegeForderungAn(t, pool, sid, ex, "nicht_zurueckgegeben", "Englisch 8 nicht zurück")
	if rec := bescheidErstellenUeberTuer(t, srv, pool, sid,
		testdaten.BescheidRumpf(testdaten.In28Tagen(), map[string]float64{f: 19.50})); rec.Code != http.StatusCreated {
		t.Fatalf("Bescheid anlegen: Status %d, want 201: %s", rec.Code, rec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/print/rechnung/"+sid, nil)
	req.SetPathValue("schueler_id", sid)
	rec := httptest.NewRecorder()
	srv.RechnungPDF()(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("Status %d, want 404 — die Forderung steht schon auf einem Bescheid, eine "+
			"Ersatzforderung mit Barzahlung darf nicht entstehen (Content-Type %q)",
			rec.Code, rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), "Bescheid") {
		t.Errorf("Meldung nennt den Bescheid nicht: %s", rec.Body.String())
	}
}
