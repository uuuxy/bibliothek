package coverquelle

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// attrappe beantwortet jede Anfrage mit antworte, ohne Netz, und merkt sich die Anfragen.
type attrappe struct {
	antworte http.HandlerFunc
	anfragen []*http.Request
}

func (a *attrappe) RoundTrip(anfrage *http.Request) (*http.Response, error) {
	a.anfragen = append(a.anfragen, anfrage)
	if a.antworte == nil {
		return nil, errors.New("netz gestört")
	}
	aufnahme := httptest.NewRecorder()
	a.antworte(aufnahme, anfrage)
	return aufnahme.Result(), nil
}

func pngBild(t *testing.T, breite, hoehe int) []byte {
	t.Helper()
	var puffer bytes.Buffer
	if err := png.Encode(&puffer, image.NewRGBA(image.Rect(0, 0, breite, hoehe))); err != nil {
		t.Fatal(err)
	}
	return puffer.Bytes()
}

func antworteMit(t *testing.T, typ string, inhalt []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", typ)
		if _, err := w.Write(inhalt); err != nil {
			t.Errorf("Antwort der Attrappe schreiben: %v", err)
		}
	}
}

func keinErsatzbild([]byte) bool { return false }

const coverAdresse = "https://covers.openlibrary.org/b/id/1-L.jpg"

// Was kein Cover ist, kommt nicht zurück: Die Anzeige liefert dann das durchsichtige Bild, das
// Anlegen eines Titels fragt die nächste Quelle. ErrKeinCover trennt den gewöhnlichen Fall
// „Quelle kennt den Titel nicht" von einer Störung.
func TestHole_Abweisungen(t *testing.T) {
	ersatzbild := pngBild(t, 128, 170)
	faelle := []struct {
		name      string
		adresse   string
		antworte  http.HandlerFunc
		keinCover bool
		meldung   string
	}{
		{"leere Adresse", "", antworteMit(t, "image/png", pngBild(t, 20, 20)), true, ""},
		{"Adresse einer leeren ISBN", leeresOpenLibraryCover, antworteMit(t, "image/png", pngBild(t, 20, 20)), true, ""},
		{"fremder Host", "https://angreifer.example/x.png", antworteMit(t, "image/png", pngBild(t, 20, 20)), false, "erlaubten Host"},
		{"Netz gestört", coverAdresse, nil, false, "netz gestört"},
		{"Status 404", coverAdresse, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }, true, ""},
		{"Status 502", coverAdresse, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) }, false, "unerwarteter Status 502"},
		{"Prüfseite der Bot-Schranke", coverAdresse, antworteMit(t, "text/html; charset=utf-8", pngBild(t, 20, 20)), false, "Nicht-Bild-Antwort"},
		{"Antwort über der Grenze", coverAdresse, antworteMit(t, "image/png", make([]byte, maxBytes+1)), false, "überschreitet 10 MB"},
		{"kein Bild", coverAdresse, antworteMit(t, "image/png", []byte("kein Bild")), false, "Bild-Header"},
		{"Ersatzbild der Quelle", coverAdresse, antworteMit(t, "image/png", ersatzbild), true, ""},
		{"Platzhalter von 5×5 Pixeln", coverAdresse, antworteMit(t, "image/png", pngBild(t, 5, 5)), true, ""},
	}
	istErsatzbild := func(antwort []byte) bool { return bytes.Equal(antwort, ersatzbild) }
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			client := &http.Client{Transport: &attrappe{antworte: f.antworte}}
			fertig, err := Hole(context.Background(), client, f.adresse, istErsatzbild)
			if err == nil {
				t.Fatalf("Hole(%q) = %d Bytes, erwartet eine Abweisung", f.adresse, len(fertig))
			}
			if errors.Is(err, ErrKeinCover) != f.keinCover {
				t.Errorf("Hole(%q) = %v; errors.Is(…, ErrKeinCover) = %v, erwartet %v", f.adresse, err, !f.keinCover, f.keinCover)
			}
			if !strings.Contains(err.Error(), f.meldung) {
				t.Errorf("Hole(%q) = %v, erwartet eine Meldung mit %q", f.adresse, err, f.meldung)
			}
		})
	}
}

// Eine Adresse, die vor dem Abruf scheitert, erreicht das Netz nie.
func TestHole_AbgewieseneAdresseFragtNicht(t *testing.T) {
	for _, adresse := range []string{"", leeresOpenLibraryCover, "https://angreifer.example/x.png"} {
		netz := &attrappe{antworte: antworteMit(t, "image/png", pngBild(t, 20, 20))}
		if _, err := Hole(context.Background(), &http.Client{Transport: netz}, adresse, keinErsatzbild); err == nil {
			t.Errorf("Hole(%q) nimmt die Adresse an", adresse)
		}
		if len(netz.anfragen) != 0 {
			t.Errorf("Hole(%q) stellt %d Anfragen, erwartet keine", adresse, len(netz.anfragen))
		}
	}
}

// Angefragt wird die neu gebaute Adresse mit der Kennung des Programms; ein Cover kommt als
// WebP zurück, ein großes verkleinert auf 600 × 900.
func TestHole_LiefertWebP(t *testing.T) {
	faelle := []struct {
		name                  string
		breite, hoehe         int
		willBreite, willHoehe int
	}{
		{"passt hinein", 300, 400, 300, 400},
		{"zu groß", 1200, 1800, 600, 900},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			netz := &attrappe{antworte: antworteMit(t, "image/png", pngBild(t, f.breite, f.hoehe))}
			fertig, err := Hole(context.Background(), &http.Client{Transport: netz}, "http://covers.openlibrary.org:2222/b/id/1-L.jpg", keinErsatzbild)
			if err != nil {
				t.Fatalf("Hole(…) = %v, erwartet ein Cover", err)
			}
			bild, format, err := image.Decode(bytes.NewReader(fertig))
			if err != nil || format != "webp" {
				t.Fatalf("Hole(…) liefert Format %q (%v), erwartet webp", format, err)
			}
			if got := bild.Bounds(); got.Dx() != f.willBreite || got.Dy() != f.willHoehe {
				t.Errorf("Hole(%d×%d) = %d×%d, erwartet %d×%d", f.breite, f.hoehe, got.Dx(), got.Dy(), f.willBreite, f.willHoehe)
			}
			anfrage := netz.anfragen[0]
			if got := anfrage.URL.String(); got != coverAdresse {
				t.Errorf("angefragt %q, erwartet die neu gebaute Adresse %q", got, coverAdresse)
			}
			if got := anfrage.Header.Get("User-Agent"); got != kennung {
				t.Errorf("User-Agent %q, erwartet %q", got, kennung)
			}
		})
	}
}
