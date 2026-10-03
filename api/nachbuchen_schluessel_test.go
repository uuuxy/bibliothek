package api

import (
	"testing"

	"bibliothek/repository"
)

// Was unter dem Schlüssel als Online-Antwort steht, gilt nur als gebucht, wenn es eine
// gelungene Ausleihe oder Rückgabe ist. Alles andere übernimmt die Nachbuch-Tür und bucht.
func TestOnlineGebucht(t *testing.T) {
	for _, f := range []struct {
		name    string
		status  int
		daten   string
		gebucht bool
	}{
		{"Ausleihe", 200, `{"type":"ausleihe","loan_id":"3f2504e0-4f89-11d3-9a0c-0305e82c3301"}`, true},
		{"Rückgabe", 200, `{"type":"rueckgabe","fremdrueckgabe":true}`, true},
		{"gescannter Ausweis", 200, `{"type":"student"}`, false},
		{"Suche", 200, `{"type":"search_results"}`, false},
		{"Fehlerantwort", 403, `{"error":"gesperrt","sperre":"leser"}`, false},
		{"Fehlerstatus mit Typ", 409, `{"type":"ausleihe"}`, false},
		{"Status unter 200", 100, `{"type":"ausleihe"}`, false},
		{"unlesbar", 200, `{"type":5}`, false},
	} {
		gebucht, ok := onlineGebucht(&repository.IdempotenzAntwort{Status: f.status, Daten: []byte(f.daten)})
		if ok != f.gebucht {
			t.Errorf("%s: gebucht %v, erwartet %v", f.name, ok, f.gebucht)
		}
		if !ok && gebucht.Type != "" {
			t.Errorf("%s: nicht gebucht, liefert aber eine Antwort vom Typ %q", f.name, gebucht.Type)
		}
		if ok && gebucht.Type == "" {
			t.Errorf("%s: gebucht, aber ohne Inhalt", f.name)
		}
	}
}
