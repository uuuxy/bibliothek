package theke

import (
	"net/http"
	"testing"

	"bibliothek/internal/service"
	"bibliothek/repository"
)

// Nur eine gebuchte Fremdrückgabe mit gültiger Ausleih-Kennung lässt den Eintrag „ausleihen" noch
// zu. Jede andere Buchung gilt als vollständig; der Eintrag buchte das Buch sonst ein zweites Mal.
func TestNurFremdrueckgabeVorAusleihe(t *testing.T) {
	gueltig := "5e1d0c8a-5b0e-4c55-9d0f-3a8b1c2d4e01"
	falsch := "urn:uuid:5e1d0c8a-5b0e-4c55-9d0f-3a8b1c2d4e01"
	faelle := []struct {
		name    string
		gebucht eingabeAntwort
		absicht string
		soll    bool
	}{
		{"Fremdrückgabe, Eintrag will ausleihen", eingabeAntwort{Art: "rueckgabe", Fremdrueckgabe: true, AusleiheID: &gueltig}, service.NachbuchAbsichtAusleihe, true},
		{"gewöhnliche Rückgabe", eingabeAntwort{Art: "rueckgabe", AusleiheID: &gueltig}, service.NachbuchAbsichtAusleihe, false},
		{"online schon ausgeliehen", eingabeAntwort{Art: "ausleihe", AusleiheID: &gueltig}, service.NachbuchAbsichtAusleihe, false},
		{"Ausleihe mit Merkmal der Fremdrückgabe", eingabeAntwort{Art: "ausleihe", Fremdrueckgabe: true, AusleiheID: &gueltig}, service.NachbuchAbsichtAusleihe, false},
		{"Eintrag will zurückgeben", eingabeAntwort{Art: "rueckgabe", Fremdrueckgabe: true, AusleiheID: &gueltig}, "rueckgabe", false},
		{"ohne Kennung", eingabeAntwort{Art: "rueckgabe", Fremdrueckgabe: true}, service.NachbuchAbsichtAusleihe, false},
		{"Kennung nicht in strenger Form", eingabeAntwort{Art: "rueckgabe", Fremdrueckgabe: true, AusleiheID: &falsch}, service.NachbuchAbsichtAusleihe, false},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if ist := nurFremdrueckgabeVorAusleihe(f.gebucht, f.absicht); ist != f.soll {
				t.Errorf("= %v, erwartet %v", ist, f.soll)
			}
		})
	}
}

// Als gebucht gilt nur eine geglückte Antwort, die eine Ausleihe oder Rückgabe nennt. Eine Ausweis-
// oder Suchantwort hat nichts gebucht: Hielte das Nachbuchen sie für gebucht, ginge der Scan verloren.
func TestOnlineGebucht(t *testing.T) {
	faelle := []struct {
		name   string
		status int
		daten  string
		soll   bool
	}{
		{"Ausleihe", http.StatusOK, `{"type":"ausleihe"}`, true},
		{"Rückgabe", http.StatusOK, `{"type":"rueckgabe"}`, true},
		{"letzter geglückter Status", http.StatusMultipleChoices - 1, `{"type":"ausleihe"}`, true},
		{"Ausweis", http.StatusOK, `{"type":"student"}`, false},
		{"Suche", http.StatusOK, `{"type":"search_results"}`, false},
		{"ohne Art", http.StatusOK, `{}`, false},
		{"kein JSON", http.StatusOK, `nicht lesbar`, false},
		{"vor den geglückten", http.StatusOK - 1, `{"type":"ausleihe"}`, false},
		{"erster nach den geglückten", http.StatusMultipleChoices, `{"type":"ausleihe"}`, false},
		{"Abweisung", http.StatusConflict, `{"type":"ausleihe"}`, false},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			gebucht, ok := onlineGebucht(&repository.IdempotenzAntwort{Status: f.status, Daten: []byte(f.daten)})
			if ok != f.soll {
				t.Fatalf("onlineGebucht(%d, %s) = %v, erwartet %v", f.status, f.daten, ok, f.soll)
			}
			if !ok && gebucht.Art != "" {
				t.Errorf("Ohne Buchung bleibt die Antwort leer, gefunden Art %q", gebucht.Art)
			}
		})
	}
}
