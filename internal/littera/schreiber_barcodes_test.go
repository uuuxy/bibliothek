package littera

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"bibliothek/internal/uebernahme"
)

// In diesen Tests testen wir die barcodeWunsch-Logik und reine Go-Methoden ohne Datenbank.
func TestSchreiber_barcodeWunsch(t *testing.T) {
	prot, err := uebernahme.NeuesProtokoll(filepath.Join(t.TempDir(), "prot.log"), "littera_id")
	if err != nil {
		t.Fatalf("NeuesProtokoll: %v", err)
	}
	s := NeuerSchreiber(nil, prot, StandardOptionen(time.Now()))

	t.Run("Fremdbarcode gewinnt", func(t *testing.T) {
		e := Exemplar{ID: "E1", Exemplarnummer: "101", Bibliotheksnummer: "1"}
		fremd := "FREMD123"

		got := s.barcodeWunsch(e, fremd)
		if got != fremd {
			t.Errorf("erwartet %q, bekommen %q", fremd, got)
		}
	})

	t.Run("Opt BarcodeNeu liefert leeren Wunsch", func(t *testing.T) {
		opt := StandardOptionen(time.Now())
		opt.Barcodes = BarcodeNeu
		sNeu := NeuerSchreiber(nil, prot, opt)

		e := Exemplar{ID: "E2", Exemplarnummer: "102", Bibliotheksnummer: "2"}
		got := sNeu.barcodeWunsch(e, "")
		if got != "" {
			t.Errorf("erwartet leeren String für BarcodeNeu, bekommen %q", got)
		}
	})

	t.Run("Littera Barcode normal", func(t *testing.T) {
		e := Exemplar{ID: "E3", Exemplarnummer: "20798", Bibliotheksnummer: "13"}
		got := s.barcodeWunsch(e, "")

		if got == "" {
			t.Errorf("erwartet nicht leeren String für Littera Barcode")
		}
	})

	t.Run("das Etikett gewinnt vor der Rechnung, mit Vermerk", func(t *testing.T) {
		// So in der Sicherung von 2010: Spalte Bibliotheksnummer 0, Etikett 0395.
		e := Exemplar{ID: "E6", Exemplarnummer: "50149", Bibliotheksnummer: "0", Barcode: "5 *pafopö#-c.bbc*"}
		vorher := prot.Warnungen()
		if got := s.barcodeWunsch(e, ""); got != "5014900039553" {
			t.Errorf("erwartet den EAN vom Etikett 5014900039553, bekommen %q", got)
		}
		if prot.Warnungen() != vorher+1 {
			t.Errorf("die Abweichung zwischen Etikett und Spalte braucht einen Vermerk im Protokoll")
		}
	})

	t.Run("Etikett und Rechnung gleich: kein Vermerk", func(t *testing.T) {
		e := Exemplar{ID: "E7", Exemplarnummer: "808", Bibliotheksnummer: "395", Barcode: "8 *pkpööp#-c.bc-*"}
		vorher := prot.Warnungen()
		if got := s.barcodeWunsch(e, ""); got != "8080000039530" {
			t.Errorf("erwartet 8080000039530, bekommen %q", got)
		}
		if prot.Warnungen() != vorher {
			t.Errorf("ohne Abweichung kein Vermerk")
		}
	})

	t.Run("ohne Druckzeichenkette wird gerechnet", func(t *testing.T) {
		e := Exemplar{ID: "E8", Exemplarnummer: "58968", Bibliotheksnummer: "395"}
		if got := s.barcodeWunsch(e, ""); got != "5896800039556" {
			t.Errorf("erwartet den gescannten Wert 5896800039556, bekommen %q", got)
		}
	})

	t.Run("Ungültiger Littera Barcode", func(t *testing.T) {
		e := Exemplar{ID: "E4", Exemplarnummer: "ungültig", Bibliotheksnummer: "13"}
		got := s.barcodeWunsch(e, "")
		if got != "" {
			t.Errorf("erwartet leeren String für ungültigen Barcode, bekommen %q", got)
		}
	})

	t.Run("Fremdbarcode zu lang", func(t *testing.T) {
		e := Exemplar{ID: "E5", Exemplarnummer: "20798", Bibliotheksnummer: "13"}

		fremd := ""
		for i := 0; i < uebernahme.MaxBarcode+1; i++ {
			fremd += "A"
		}

		got := s.barcodeWunsch(e, fremd)
		if got != "" {
			t.Errorf("erwartet leeren String für zu langen Barcode, bekommen %q", got)
		}
	})
}

func TestSchreiber_naechsterFreier(t *testing.T) {
	s := &Schreiber{}

	t.Run("nimmt ersten freien Barcode", func(t *testing.T) {
		vorrat := []string{"B-1", "B-2", "B-3"}
		belegt := map[string]bool{"B-1": true}

		got, rest, err := s.naechsterFreier(context.Background(), vorrat, belegt)
		if err != nil {
			t.Fatalf("unerwarteter Fehler: %v", err)
		}
		if got != "B-2" {
			t.Errorf("erwartet B-2, bekommen %q", got)
		}
		if len(rest) != 1 || rest[0] != "B-3" {
			t.Errorf("erwartet Rest [B-3], bekommen %v", rest)
		}
	})
}
