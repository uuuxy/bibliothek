package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"bibliothek/api/theke"
	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/internal/tuertest"
	"bibliothek/repository"
)

// Das Verhalten zu uuid_eingaben_test.go im Wurzelpaket: Eine Kennung, die keine UUID ist, wird
// mit 400 abgewiesen, bevor Datenbank oder Abfragen sie sehen. Die Fälle stammen aus einem Lauf
// von ZAP, der sie als 500 zeigte, und sind an einer lokalen Anlage nachgestellt.
//
// Der Server hat hier keine Datenbank (s.DB == nil), und die Abfragen sind Attrappen, die sich
// merken, ob sie gerufen wurden. Erreicht eine Tür die Datenbank doch, fängt
// tuertest.StatusOhneDB den Absturz und meldet ihn als „erreichte die Datenbank".

type vormerkungsAttrappe struct {
	gerufen bool
}

func (a *vormerkungsAttrappe) LadeListe(context.Context, string, string) ([]repository.Vormerkung, error) {
	a.gerufen = true
	return nil, nil
}

func (a *vormerkungsAttrappe) LegeAn(context.Context, string, string, string) (string, error) {
	a.gerufen = true
	return "neu", nil
}

// urnForm besteht uuid.Validate und uuid.Parse (google/uuid kennt das Präfix), Postgres weist
// sie ab: `invalid input syntax for type uuid`. POST /api/schueler/{id}/zusammenfuehren mit
// dieser quelle_id antwortete ohne die Prüfung mit 500.
const urnForm = "urn:uuid:7c9e6679-7425-40de-944b-e07fc1f90ae7"

func TestUngueltigeKennungIst400VorDerDatenbank(t *testing.T) {
	s := &Server{}
	faelle := []struct {
		name          string
		h             http.HandlerFunc
		methode, pfad string
		rumpf         string
	}{
		{"Fehlbestand ?session_id", s.inventurlauf().Fehlbestand(), http.MethodGet, "/api/inventur/fehlbestand?session_id=session_id", ""},
		{"Inventur abschließen", s.inventurlauf().InventurAbschliessen(), http.MethodPost, "/api/inventur/finish", `{"session_id":"x"}`},
		{"Inventur abbrechen", s.inventurlauf().InventurVerwerfen(), http.MethodPost, "/api/inventur/abort", `{"session_id":"x"}`},
		{"Verluste endgültig löschen", s.inventurlauf().VerlusteEndgueltigLoeschen(), http.MethodPost, "/api/buecher/exemplare/verlust-endgueltig-loeschen", `{"exemplar_ids":["x"]}`},
		// Eine leere Kennung in einer Liste: uuid_oder_leer lässt "" durch, `required` hinter
		// `dive` nicht. Ohne `required` erreichten diese vier Türen damit die Datenbank
		// (am Wareneingang nachgestellt: 500 statt 400).
		{"Verluste endgültig löschen, leeres Element", s.inventurlauf().VerlusteEndgueltigLoeschen(), http.MethodPost, "/api/buecher/exemplare/verlust-endgueltig-loeschen", `{"exemplar_ids":[""]}`},
		{"Wareneingang einbuchen, leeres Element", s.bestellung().WareneingangBuchen(), http.MethodPost, "/api/bestellungen/bulk-receive", `{"exemplar_ids":[""]}`},
		{"Mahnbriefe drucken, leeres Element", s.mahnwesen().MahnbriefeDrucken(), http.MethodPost, "/api/mahnwesen/bulk-print", `{"ausleih_ids":[""]}`},
		{"Schüler-Etiketten, leeres Element", s.druck().SchuelerEtiketten(nil), http.MethodPost, "/api/print/schueler-etiketten", `{"formatId":"x","schuelerIds":[""]}`},
		{"Fehlbestand ?session_id in urn-Form", s.inventurlauf().Fehlbestand(), http.MethodGet, "/api/inventur/fehlbestand?session_id=" + urnForm, ""},
		{"Inventur abschließen, urn-Form", s.inventurlauf().InventurAbschliessen(), http.MethodPost, "/api/inventur/finish", `{"session_id":"` + urnForm + `"}`},
		{"Schüler zusammenführen, urn-Form", s.leser().LeserZusammenfuehren(nil), http.MethodPost, "/api/schueler/11111111-1111-1111-1111-111111111111/zusammenfuehren", `{"quelle_id":"` + urnForm + `"}`},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if status := tuertest.StatusOhneDB(t, f.h, f.methode, f.pfad, f.rumpf); status != http.StatusBadRequest {
				t.Fatalf("Status %d, erwartet 400", status)
			}
		})
	}
}

// Der Schlüssel gegen Doppelbuchung geht als UUID in die Datenbank, und Schadensmeldung und
// Bescheid brauchen ihre Kennungen. Ohne Prüfung endete ein Schlüssel wie „scan-1" als 22P02:
// An der Theke lief die Buchung ohne Schutz und schrieb eine Fehlerzeile ins Log, Bestellung und
// Klassensatz antworteten mit 500. Eine leere Kennung erreichte bei Schaden und Bescheid die
// Datenbank, ebenfalls 500.
func TestSchluesselUndPflichtkennungSind400VorDerDatenbank(t *testing.T) {
	s := &Server{}
	const titel = "22222222-2222-2222-2222-222222222222"
	frist := time.Now().AddDate(0, 0, 28).Format(tuer.DatumsformatISO)
	faelle := []struct {
		name, pfad, rumpf string
		h                 http.HandlerFunc
	}{
		{"Theke, Schlüssel keine UUID", "/api/action", `{"query":"B-1","idempotency_key":"scan-1"}`,
			mitAnmeldung(s.theke().EingabeVerarbeiten(nil))},
		{"Bestellung, Schlüssel keine UUID", "/api/bestellungen",
			`{"supplier_id":"11111111-1111-1111-1111-111111111111","mittel":"land","idempotency_key":"abc",` +
				`"items":[{"titel_id":"` + titel + `","menge":1,"preis":100}]}`,
			s.bestellung().BestellungAnlegen(nil)},
		{"Klassensatz, Schlüssel keine UUID", "/api/reservierungen/klassensatz",
			`{"titel_id":"` + titel + `","klasse":"5a","anzahl":1,"idempotency_key":"abc"}`,
			mitAnmeldung(s.portal().KlassensatzReservierungAnlegen())},
		{"Schaden melden, leere Ausleihe", "/api/damage/report", `{"loan_id":"","art":"beschaedigt","betrag":1}`,
			mitAnmeldung(s.schaden().SchadenMelden(nil))},
		{"Bescheid, leere Forderung", "/api/schueler/" + titel + "/bescheide",
			`{"mittel":"land","frist_bis":"` + frist + `","positionen":[{"schadensfall_id":"","betrag":500}]}`,
			mitKennungImPfad(s.schaden().BescheidErstellen(nil, nil), titel)},
		{"Bescheid, leere Ausleihe", "/api/schueler/" + titel + "/bescheide",
			`{"mittel":"land","frist_bis":"` + frist + `","ausleihen":[{"ausleihe_id":"","betrag":500}]}`,
			mitKennungImPfad(s.schaden().BescheidErstellen(nil, nil), titel)},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if status := tuertest.StatusOhneDB(t, f.h, http.MethodPost, f.pfad, f.rumpf); status != http.StatusBadRequest {
				t.Fatalf("Status %d, erwartet 400", status)
			}
		})
	}
}

// mitAnmeldung legt eine Anmeldung in den Kontext: Theke, Schadensmeldung und Klassensatz fragen
// danach, bevor oder nachdem sie die Anfrage lesen.
func mitAnmeldung(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h(w, r.WithContext(context.WithValue(r.Context(), auth.ClaimsSchluessel,
			&auth.Claims{KontoID: "33333333-3333-3333-3333-333333333333", Rolle: auth.RolleAdmin})))
	}
}

// mitKennungImPfad setzt {id}, das sonst der Router füllt.
func mitKennungImPfad(h http.HandlerFunc, id string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.SetPathValue("id", id)
		h(w, r)
	}
}

func TestVormerkungen_UngueltigeKennungErreichtDieAbfragenNicht(t *testing.T) {
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
			repo := &vormerkungsAttrappe{}
			h := s.theke().VormerkungAnlegen(repo)
			if f.liste {
				h = s.theke().Vormerkungen(repo)
			}
			if status := tuertest.StatusOhneDB(t, h, f.methode, f.pfad, f.rumpf); status != http.StatusBadRequest {
				t.Fatalf("Status %d, erwartet 400", status)
			}
			if repo.gerufen {
				t.Fatal("Repository wurde mit der ungültigen Kennung gerufen")
			}
		})
	}
}

// Gegenprobe: Leer bleibt erlaubt; die Türen melden „… fehlt" selbst, und optionale Felder
// kommen von der Oberfläche auch als "". Das eingebaute `omitempty,uuid` wiese einen *string,
// der auf "" zeigt, mit 400 ab; uuid_oder_leer nicht.
func TestUUIDOderLeer_LeerUndGueltigGehenDurch(t *testing.T) {
	leer := ""
	gut := "11111111-1111-1111-1111-111111111111"
	if err := tuer.Pruefer.Struct(theke.EingabeAnfrage{Text: "B-1", AktiverLeserID: &leer}); err != nil {
		t.Fatalf("leerer Zeiger abgewiesen: %v", err)
	}
	if err := tuer.Pruefer.Struct(theke.EingabeAnfrage{Text: "B-1", AktiverLeserID: &gut}); err != nil {
		t.Fatalf("gültige UUID abgewiesen: %v", err)
	}
	if err := tuer.Pruefer.Struct(theke.EingabeAnfrage{Text: "B-1"}); err != nil {
		t.Fatalf("nil-Zeiger und leerer Schlüssel abgewiesen: %v", err)
	}
	if err := tuer.Pruefer.Struct(theke.EingabeAnfrage{Text: "B-1", IdempotenzSchluessel: gut}); err != nil {
		t.Fatalf("gültiger Schlüssel abgewiesen: %v", err)
	}
	if err := tuer.Pruefer.Struct(theke.EingabeAnfrage{Text: "B-1", AktiverLeserID: &[]string{"x"}[0]}); err == nil {
		t.Fatal("„x“ als active_leser_id durchgelassen")
	} else if msg := tuer.Pruefmeldung(err); !strings.Contains(msg, "Ungültige Kennung im Feld „active_leser_id“") {
		t.Fatalf("Meldung %q nennt die Kennung nicht", msg)
	}

	s := &Server{}
	repo := &vormerkungsAttrappe{}
	status := tuertest.StatusOhneDB(t, s.theke().VormerkungAnlegen(repo), http.MethodPost, "/api/vormerkungen",
		`{"titel_id":"`+gut+`","schueler_id":""}`)
	if status != http.StatusCreated || !repo.gerufen {
		t.Fatalf("gültige Vormerkung mit leerer schueler_id: Status %d, Repository gerufen=%v", status, repo.gerufen)
	}
}
