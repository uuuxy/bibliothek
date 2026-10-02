package coverquelle

import (
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestIstErsatzbild(t *testing.T) {
	bild := []byte("irgendein bild")
	if IstErsatzbild(bild) {
		t.Fatal("ein unbekanntes Bild gilt als Ersatzbild")
	}
	vergiss := MerkeErsatzbildFuerTest(bild)
	if !IstErsatzbild(bild) {
		t.Error("das gemerkte Bild wird nicht erkannt")
	}
	if IstErsatzbild([]byte("ein anderes bild")) {
		t.Error("ein anderes Bild gilt als Ersatzbild")
	}
	vergiss()
	if IstErsatzbild(bild) {
		t.Error("nach dem Zurücknehmen gilt das Bild weiter als Ersatzbild")
	}
	if len(ersatzbilder) != 2 {
		t.Errorf("%d Ersatzbilder in der Liste, erwartet die zwei von Google Books", len(ersatzbilder))
	}
}

// Zurückgenommen wird nur die eigene Angabe: Stand das Bild schon in der Liste, steht es
// danach weiter dort. Sonst fehlte allen späteren Tests des Pakets ein Eintrag des Programms.
func TestMerkeErsatzbildFuerTest_BekanntesBildBleibtBekannt(t *testing.T) {
	bild := []byte("ein bild, das schon in der liste steht")
	summe := pruefsumme(bild)
	ersatzbilder[summe] = true
	t.Cleanup(func() { delete(ersatzbilder, summe) })

	MerkeErsatzbildFuerTest(bild)()

	if !IstErsatzbild(bild) {
		t.Error("das Zurücknehmen hat einen Eintrag entfernt, der vorher schon galt")
	}
}

// Die Prüfsummen der Liste sind an Google Books gemessen. Ändert Google ein Ersatzbild,
// erkennt das Programm es nicht mehr, und in den Katalogen steht wieder „image not available".
// Dieser Test misst nach, braucht dafür das Netz und läuft deshalb nur auf Wunsch:
//
//	COVER_LIVE=1 go test ./pkg/coverquelle/ -run Live -v
//
// Gefragt wird mit zwei ISBN, zu denen Google kein Cover hat; die Antworten nennt er mit
// Prüfsumme.
func TestErsatzbilder_LiveGegenGoogle(t *testing.T) {
	if os.Getenv("COVER_LIVE") == "" {
		t.Skip("nur mit COVER_LIVE=1: fragt books.google.com")
	}
	client := &http.Client{Timeout: 20 * time.Second}
	for _, isbn := range []string{"9783000000003", "9783127337600"} {
		antwort, err := client.Get("https://books.google.com/books/content?vid=ISBN:" + isbn + "&printsec=frontcover&img=1&zoom=1")
		if err != nil {
			t.Fatalf("Google Books nicht erreichbar: %v", err)
		}
		bild, err := io.ReadAll(antwort.Body)
		if cerr := antwort.Body.Close(); cerr != nil {
			t.Logf("Antwort schließen: %v", cerr)
		}
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("ISBN %s: Status %d, %s, %d Byte, %s", isbn, antwort.StatusCode,
			antwort.Header.Get("Content-Type"), len(bild), pruefsumme(bild))
		if !IstErsatzbild(bild) {
			t.Errorf("die Antwort zu %s steht nicht in der Liste — ein neues Ersatzbild oder inzwischen ein Cover", isbn)
		}
	}
}
