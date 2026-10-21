package bestellung_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
)

// Trifft ein Exemplar aus der Bestellung ein, hatte der Händler sie: Der Vermerk „Mail nicht
// versendet" gilt dann nicht mehr, und die Mail geht nicht noch einmal raus. Dass der Link in
// der gescheiterten Mail stand, bleibt im Detail stehen.
func TestBestellmail_WareneingangNimmtDenVermerk(t *testing.T) {
	l := bestellmailLageAnlegen(t, "geliefert")
	testdaten.SetzeOeffentlicheAdresse(t, l.pool, "https://bib.example.invalid")
	lieferant := testdaten.LegeLieferantAn(t, l.pool, "Naacher-Geliefert", true)
	titel := testdaten.LegeTitelAn(t, l.pool, "LMF-Mathe-Geliefert", 0)

	l.srv.Mail = mailserverNichtErreichbar(t, nil)
	bestellung, _ := l.bestelle(t, lieferant, titel)
	if !l.gescheitert(t, bestellung) {
		t.Fatal("Die Bestellung trägt keinen Vermerk, obwohl die Mail nicht rausging")
	}

	// Der Händler liefert trotzdem, die Bibliothek bucht eines der zwei Exemplare ein.
	var exemplar string
	if err := l.pool.QueryRow(t.Context(),
		`SELECT id FROM buecher_exemplare WHERE bestellung_id = $1 ORDER BY barcode_id LIMIT 1`,
		bestellung).Scan(&exemplar); err != nil {
		t.Fatalf("Exemplar der Bestellung lesen: %v", err)
	}
	if rec := l.rufe(t, http.MethodPost, "/api/bestellungen/bulk-receive",
		`{"exemplar_ids":["`+exemplar+`"]}`); rec.Code != http.StatusOK {
		t.Fatalf("Wareneingang: Status %d: %s", rec.Code, rec.Body.String())
	}

	if l.gescheitert(t, bestellung) {
		t.Error("Die gelieferte Bestellung trägt den Vermerk weiter")
	}
	if l.inListeGescheitert(t, bestellung) {
		t.Error("Die Liste nennt für die gelieferte Bestellung einen gescheiterten Versand")
	}
	rec := l.rufe(t, http.MethodGet, "/api/bestellhistorie/"+bestellung, "")
	var kopf struct {
		LinkNichtErhalten bool `json:"link_nicht_erhalten"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &kopf); err != nil {
		t.Fatalf("Detail unlesbar (%d): %v", rec.Code, err)
	}
	if !kopf.LinkNichtErhalten {
		t.Error("Das Detail nennt den Link nach der Lieferung als erhalten, obwohl die Mail mit ihm nicht rausging")
	}

	post, sitzungen := tuertest.FangeMailAb(t)
	l.srv.Mail = post
	if rec := l.rufe(t, http.MethodPost, "/api/bestellungen/"+bestellung+"/mail", ""); rec.Code != http.StatusConflict {
		t.Errorf("Erneut senden nach der Lieferung: Status %d, erwartet 409: %s", rec.Code, rec.Body.String())
	}
	select {
	case mail := <-sitzungen:
		t.Errorf("Die gelieferte Bestellung ging noch einmal raus, an %v", mail.Empfaenger)
	case <-time.After(300 * time.Millisecond):
	}
}
