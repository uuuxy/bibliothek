package theke

// Ohne Netz gibt Anna mit ihrem Ausweis ein Buch ab, das sie am Netz ausgeliehen hatte. Am Netz
// nähme derselbe Scan es zurück. Die Theke legt ihn als „ausleihe" mit Person ab und sagt, wie
// lange Anna beim Scan schon an der Theke stand (besuch_seit_ms); ohne das bliebe das Buch auf
// ihrem Konto (bereits_ausgeliehen).

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNachbuchen_Besuch_RueckgabeMitAusweis(t *testing.T) {
	w := baueNachbuchWelt(t)
	if code, r := w.onlineScan(t, uuid.NewString(), w.anna); code != http.StatusOK || r.Art != "ausleihe" {
		t.Fatalf("Online-Ausleihe an Anna: %d %q", code, r.Art)
	}
	// Anna hat das Buch seit einer Stunde.
	if _, err := w.pool.Exec(context.Background(), `
		UPDATE ausleihen SET ausgeliehen_am = ausgeliehen_am - interval '1 hour', erfasst_am = erfasst_am - interval '1 hour'
		WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`, w.exemplarID); err != nil {
		t.Fatalf("Ausleihe zurückdatieren: %v", err)
	}

	jetzt := time.Now()
	code, antwort := w.nachbuchenMitUhr(t, &jetzt, map[string]any{
		"schluessel": uuid.NewString(), "absicht": "ausleihe", "barcode": w.code,
		"gescannt_am": jetzt, "leser_id": w.anna, "besuch_seit_ms": 20000,
	})
	if code != http.StatusOK || len(antwort.Ergebnisse) != 1 || antwort.Ergebnisse[0].Ergebnis != "zurueckgegeben" {
		t.Fatalf("Nachbuchen: %d %+v — erwartet zurueckgegeben", code, antwort.Ergebnisse)
	}
	if w.ausleiheOffen(t, w.anna) {
		t.Error("das Buch steht weiter auf Annas Konto")
	}
}
