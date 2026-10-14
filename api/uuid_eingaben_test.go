package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"bibliothek/api/theke"
	"bibliothek/api/tuer"
	"bibliothek/internal/tuertest"
	"bibliothek/repository"
)

// Verhalten zu uuid_eingaben_test.go (Root-Paket): Eine Kennung, die keine UUID ist, wird
// mit 400 abgewiesen, BEVOR Datenbank oder Repository sie sehen. Die Fälle sind die, die
// der ZAP-Lauf vom 13.09.2026 als 500 zeigte und die am Stack nachgestellt wurden.
//
// Der Server hat hier keine Datenbank (s.DB == nil) und die Repositories sind Attrappen,
// die sich merken, ob sie gerufen wurden. Erreicht ein Handler die Datenbank doch, fängt
// tuertest.AntwortOhneDB den Absturz und meldet ihn als „erreichte die Datenbank" — genau das war
// vor der Prüfung der Fall.

type vormerkungRepoZaehler struct {
	repository.VormerkungsAbfragen
	gerufen bool
}

func (v *vormerkungRepoZaehler) LadeListe(context.Context, string, string) ([]repository.Vormerkung, error) {
	v.gerufen = true
	return nil, nil
}

func (v *vormerkungRepoZaehler) LegeAn(context.Context, string, string, string) (string, error) {
	v.gerufen = true
	return "neu", nil
}

// urnForm besteht uuid.Validate und uuid.Parse (google/uuid kennt das Präfix), Postgres
// weist sie ab: `invalid input syntax for type uuid`. Am Stack am 13.09.2026 nachgestellt —
// POST /api/schueler/{id}/zusammenfuehren mit dieser quelle_id kam als 500 zurück.
const urnForm = "urn:uuid:7c9e6679-7425-40de-944b-e07fc1f90ae7"

func TestUngueltigeKennungIst400VorDerDatenbank(t *testing.T) {
	s := &Server{}
	faelle := []struct {
		name          string
		h             http.HandlerFunc
		methode, pfad string
		rumpf         string
	}{
		{"Fehlbestand ?session_id", s.inventurlauf().InventurFehlbestandHandler(), http.MethodGet, "/api/inventur/fehlbestand?session_id=session_id", ""},
		{"Inventur abschließen", s.inventurlauf().InventurFinishHandler(), http.MethodPost, "/api/inventur/finish", `{"session_id":"x"}`},
		{"Inventur abbrechen", s.inventurlauf().InventurAbortHandler(), http.MethodPost, "/api/inventur/abort", `{"session_id":"x"}`},
		{"Verluste endgültig löschen", s.inventurlauf().InventurVerlusteLoeschenHandler(), http.MethodPost, "/api/buecher/exemplare/verlust-endgueltig-loeschen", `{"exemplar_ids":["x"]}`},
		// Eine leere Kennung IN einer Liste: uuid_oder_leer lässt "" durch, `required` hinter
		// `dive` nicht. Bis zum 08.10.2026 erreichten diese vier Türen damit die Datenbank
		// (am Wareneingang nachgestellt: 500 statt 400).
		{"Verluste endgültig löschen, leeres Element", s.inventurlauf().InventurVerlusteLoeschenHandler(), http.MethodPost, "/api/buecher/exemplare/verlust-endgueltig-loeschen", `{"exemplar_ids":[""]}`},
		{"Wareneingang einbuchen, leeres Element", s.bestellung().BulkReceiveOrderHandler(), http.MethodPost, "/api/bestellungen/bulk-receive", `{"exemplar_ids":[""]}`},
		{"Mahnbriefe drucken, leeres Element", s.mahnwesen().BulkPrintMahnungenHandler(), http.MethodPost, "/api/mahnwesen/bulk-print", `{"ausleih_ids":[""]}`},
		{"Schüler-Etiketten, leeres Element", s.druck().SchuelerEtiketten(nil), http.MethodPost, "/api/print/schueler-etiketten", `{"formatId":"x","schuelerIds":[""]}`},
		{"Fehlbestand ?session_id in urn-Form", s.inventurlauf().InventurFehlbestandHandler(), http.MethodGet, "/api/inventur/fehlbestand?session_id=" + urnForm, ""},
		{"Inventur abschließen, urn-Form", s.inventurlauf().InventurFinishHandler(), http.MethodPost, "/api/inventur/finish", `{"session_id":"` + urnForm + `"}`},
		{"Schüler zusammenführen, urn-Form", s.leser().ZusammenfuehrenSchuelerHandler(nil), http.MethodPost, "/api/schueler/11111111-1111-1111-1111-111111111111/zusammenfuehren", `{"quelle_id":"` + urnForm + `"}`},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if status := tuertest.AntwortOhneDB(t, f.h, f.methode, f.pfad, f.rumpf); status != http.StatusBadRequest {
				t.Fatalf("Status %d, erwartet 400", status)
			}
		})
	}
}

func TestVormerkungen_UngueltigeKennungErreichtRepositoryNicht(t *testing.T) {
	s := &Server{}
	faelle := []struct {
		name, methode, pfad, rumpf string
		liste                      bool
	}{
		{"Liste ?titel_id", http.MethodGet, "/api/vormerkungen?titel_id=x", "", true},
		{"Liste ?schueler_id", http.MethodGet, "/api/vormerkungen?schueler_id=x", "", true},
		{"Anlegen titel_id", http.MethodPost, "/api/vormerkungen", `{"titel_id":"x"}`, false},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			repo := &vormerkungRepoZaehler{}
			h := s.theke().CreateVormerkungHandler(repo)
			if f.liste {
				h = s.theke().ListVormerkungHandler(repo)
			}
			if status := tuertest.AntwortOhneDB(t, h, f.methode, f.pfad, f.rumpf); status != http.StatusBadRequest {
				t.Fatalf("Status %d, erwartet 400", status)
			}
			if repo.gerufen {
				t.Fatal("Repository wurde mit der ungültigen Kennung gerufen")
			}
		})
	}
}

// Gegenprobe: Leer bleibt erlaubt — die Handler melden „… fehlt" selbst, und optionale
// Felder kommen vom Frontend auch als "". Das eingebaute `omitempty,uuid` wiese einen
// *string, der auf "" zeigt, mit 400 ab (am 13.09.2026 ausprobiert); uuid_oder_leer nicht.
func TestUUIDOderLeer_LeerUndGueltigGehenDurch(t *testing.T) {
	leer := ""
	gut := "11111111-1111-1111-1111-111111111111"
	if err := tuer.Pruefer.Struct(theke.ActionRequest{Query: "B-1", ActiveLeserID: &leer}); err != nil {
		t.Fatalf("leerer Zeiger abgewiesen: %v", err)
	}
	if err := tuer.Pruefer.Struct(theke.ActionRequest{Query: "B-1", ActiveLeserID: &gut}); err != nil {
		t.Fatalf("gültige UUID abgewiesen: %v", err)
	}
	if err := tuer.Pruefer.Struct(theke.ActionRequest{Query: "B-1"}); err != nil {
		t.Fatalf("nil-Zeiger abgewiesen: %v", err)
	}
	if err := tuer.Pruefer.Struct(theke.ActionRequest{Query: "B-1", ActiveLeserID: &[]string{"x"}[0]}); err == nil {
		t.Fatal("„x“ als active_leser_id durchgelassen")
	} else if msg := tuer.Pruefmeldung(err).Error(); !strings.Contains(msg, "ungültige Kennung") {
		t.Fatalf("Meldung %q nennt die Kennung nicht", msg)
	}

	s := &Server{}
	repo := &vormerkungRepoZaehler{}
	status := tuertest.AntwortOhneDB(t, s.theke().CreateVormerkungHandler(repo), http.MethodPost, "/api/vormerkungen",
		`{"titel_id":"`+gut+`","schueler_id":""}`)
	if status != http.StatusCreated || !repo.gerufen {
		t.Fatalf("gültige Vormerkung mit leerer schueler_id: Status %d, Repository gerufen=%v", status, repo.gerufen)
	}
}
