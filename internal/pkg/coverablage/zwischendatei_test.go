package coverablage

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// Wer ein Cover ausliefert, während es geschrieben wird, sah eine leere oder halbe Datei: Der
// Cache prüft nur, ob der Name da ist, und der Browser behält das Bild ein Jahr. Über die
// Zwischendatei steht unter dem Namen nur ganzes, und es bleibt keine Zwischendatei liegen.
func TestSchreibeUeberZwischendatei_NieHalb(t *testing.T) {
	verzeichnis := t.TempDir()
	wurzel, err := os.OpenRoot(verzeichnis)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := wurzel.Close(); err != nil {
			t.Errorf("Wurzel schließen: %v", err)
		}
	})

	inhalt := bytes.Repeat([]byte("x"), 1<<20)
	ende := time.Now().Add(time.Second)
	gesehen := make(chan string, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for time.Now().Before(ende) {
			if daten, err := wurzel.ReadFile("cover.webp"); err == nil && len(daten) != len(inhalt) {
				gesehen <- fmt.Sprintf("%d von %d Bytes", len(daten), len(inhalt))
				return
			}
		}
	}()
	for time.Now().Before(ende) {
		if err := SchreibeUeberZwischendatei(wurzel, "cover.webp", inhalt); err != nil {
			t.Fatalf("SchreibeUeberZwischendatei: %v", err)
		}
	}
	wg.Wait()
	select {
	case halb := <-gesehen:
		t.Errorf("unter dem Namen stand eine halbe Datei: %s", halb)
	default:
	}

	eintraege, err := os.ReadDir(verzeichnis)
	if err != nil {
		t.Fatal(err)
	}
	if len(eintraege) != 1 || eintraege[0].Name() != "cover.webp" {
		t.Errorf("im Verzeichnis liegt %v, erwartet nur cover.webp", eintraege)
	}
}
