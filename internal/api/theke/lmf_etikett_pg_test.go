package theke

// Ein Lernmittel mit kurzer Nummer kann noch sein altes LMF-Etikett tragen (lmf_barcode). Der
// Scan dieses Etiketts bucht an der Theke wie die Nummer selbst, am Netz und ohne Netz
// nachgebucht, und Antwort und Meldung nennen die Nummer, die das Exemplar jetzt trägt.

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// gibAltesLMFEtikett hängt dem Exemplar der Welt ein altes LMF-Etikett an und nennt dessen
// Nummer.
func (w *nachbuchWelt) gibAltesLMFEtikett(t *testing.T) string {
	t.Helper()
	alt := "LMF-9783060158423-" + w.code[strings.LastIndex(w.code, "-")+1:]
	if _, err := w.pool.Exec(context.Background(),
		`UPDATE buecher_exemplare SET lmf_barcode = $1 WHERE id = $2`, alt, w.exemplarID); err != nil {
		t.Fatalf("altes LMF-Etikett setzen: %v", err)
	}
	return alt
}

func TestAltesLMFEtikett_BuchtAmNetz(t *testing.T) {
	w := baueNachbuchWelt(t)
	alt := w.gibAltesLMFEtikett(t)

	code, r, rumpf := w.scanne(t, alt, uuid.NewString(), w.anna)
	if code != http.StatusOK || r.Art != "ausleihe" {
		t.Fatalf("Ausleihe über das alte Etikett: %d %s", code, rumpf)
	}
	if n, bei := w.offen(t); n != 1 || bei != w.anna {
		t.Fatalf("%d offene Ausleihen bei %q, erwartet eine bei Anna", n, bei)
	}
	if r.Exemplar == nil || r.Exemplar.BarcodeID != w.code {
		t.Errorf("die Antwort nennt %+v, erwartet die Nummer %s", r.Exemplar, w.code)
	}

	if code, r, rumpf := w.scanne(t, alt, uuid.NewString(), ""); code != http.StatusOK || r.Art != "rueckgabe" {
		t.Fatalf("Rückgabe über das alte Etikett: %d %s", code, rumpf)
	}
	if n, _ := w.offen(t); n != 0 {
		t.Errorf("nach der Rückgabe %d offene Ausleihen, erwartet keine", n)
	}
}

// Ein verliehenes Buch, das inzwischen ausgesondert ist, geht über sein altes Etikett bei einem
// anderen Leser als Fremdrückgabe zurück; die Antwort nennt seine Nummer, nicht das Etikett.
func TestAltesLMFEtikett_AusgesondertGehtZurueckUndNenntDieNummer(t *testing.T) {
	w := baueNachbuchWelt(t)
	alt := w.gibAltesLMFEtikett(t)
	if code, r, rumpf := w.scanne(t, alt, uuid.NewString(), w.anna); code != http.StatusOK || r.Art != "ausleihe" {
		t.Fatalf("Ausleihe über das alte Etikett: %d %s", code, rumpf)
	}
	if _, err := w.pool.Exec(context.Background(), `UPDATE buecher_exemplare
		SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'VERLUST' WHERE id = $1`, w.exemplarID); err != nil {
		t.Fatalf("aussondern: %v", err)
	}
	code, r, rumpf := w.scanne(t, alt, uuid.NewString(), w.ben)
	if code != http.StatusOK || r.Art != "rueckgabe" || !r.Fremdrueckgabe {
		t.Fatalf("Scan des ausgesonderten Buchs bei Ben: %d %s, erwartet die Fremdrückgabe", code, rumpf)
	}
	if r.Exemplar == nil || r.Exemplar.BarcodeID != w.code {
		t.Errorf("die Antwort nennt %+v, erwartet die Nummer %s statt des alten Etiketts", r.Exemplar, w.code)
	}
	if n, _ := w.offen(t); n != 0 {
		t.Errorf("nach der Fremdrückgabe %d offene Ausleihen, erwartet keine", n)
	}
}

func TestAltesLMFEtikett_WirdOhneNetzNachgebucht(t *testing.T) {
	w := baueNachbuchWelt(t)
	alt := w.gibAltesLMFEtikett(t)

	ergebnisse := w.nachbuchen(t, map[string]any{
		"schluessel": uuid.NewString(), "absicht": "ausleihe", "barcode": alt,
		"gescannt_am": time.Now(), "leser_id": w.anna,
	})
	if ergebnisse[0].Ergebnis != "ausgeliehen" {
		t.Fatalf("Nachbuchen über das alte Etikett: %q %s", ergebnisse[0].Ergebnis, ergebnisse[0].Grund)
	}
	if n, bei := w.offen(t); n != 1 || bei != w.anna {
		t.Errorf("%d offene Ausleihen bei %q, erwartet eine bei Anna", n, bei)
	}
}
