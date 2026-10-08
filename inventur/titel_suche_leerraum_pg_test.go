package inventur

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"bibliothek/internal/pgtest"
)

// Titel-Verwaltung und Lernmittel-Liste suchen am Server und vergleichen den Suchtext mit
// dem Wortlaut. Littera führt Titel mit zwei Leerzeichen in Folge („La  Peste") und mit
// geschütztem Leerzeichen; „La Peste" traf sie nicht. Die Datenbank speichert Titeltexte
// mit einem Leerzeichen zwischen den Wörtern (Migration 160), und die Suche nimmt den
// Suchtext in derselben Form. Der Titel entsteht hier an der Tür der Maske.
func TestSuche_LeerraumInFolgeTrenntSuchtextUndTitelNicht(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)
	handler := &APIHandler{repo: repo, metadaten: offlineMetadatenClient()}
	const gespeichert, barcode = "Leerraumprobe La Peste", "B-LEERRAUM-INV-1"
	geschuetzt := string(rune(0x00A0))

	raeume := func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_exemplare WHERE barcode_id = $1`, barcode); err != nil {
			t.Errorf("aufräumen: Probe-Exemplar löschen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE titel LIKE 'Leerraumprobe%'`); err != nil {
			t.Errorf("aufräumen: Probe-Titel löschen: %v", err)
		}
	}
	raeume()
	t.Cleanup(raeume)

	id, err := repo.CreateBook(ctx, Book{
		Title: "Leerraumprobe La  Peste", Author: "Camus," + geschuetzt + "Albert", IstLernmittel: true,
	})
	if err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2)`, id, barcode); err != nil {
		t.Fatal(err)
	}

	// Die Lernmittel-Liste über ihre Tür: Liste, Fach-Zahlen und Ausdruck lesen den Suchtext
	// an einer Stelle (lernmittelFilterParam).
	lernmittel := func(suchtext string) []LernmittelTitel {
		t.Helper()
		rec := httptest.NewRecorder()
		handler.handlePortalLernmittel(rec, httptest.NewRequest(http.MethodGet,
			"/api/portal/lernmittel?q="+url.QueryEscape(suchtext), nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("Lernmittel, Suche %+q: %d %s", suchtext, rec.Code, rec.Body.String())
		}
		var antwort struct {
			Titel []LernmittelTitel `json:"titel"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatal(err)
		}
		return antwort.Titel
	}

	for _, suchtext := range []string{
		"Leerraumprobe La Peste",               // getippt, wie man tippt
		"Leerraumprobe La  Peste",              // wie der Titel in Littera steht
		"leerraumprobe la" + geschuetzt + "pe", // aus einer Liste kopiert, Wortanfang
		"Camus, Albert",
		"Camus,  Alb",
	} {
		liste, err := repo.ListBooks(ctx, "", suchtext, false)
		if err != nil {
			t.Fatalf("ListBooks(%+q): %v", suchtext, err)
		}
		if len(liste) != 1 || liste[0].ID != id || liste[0].Title != gespeichert {
			t.Errorf("Titel-Verwaltung, Suche %+q: %d Treffer, erwartet den Titel %q", suchtext, len(liste), gespeichert)
		}
		if treffer := lernmittel(suchtext); len(treffer) != 1 || treffer[0].ID != id {
			t.Errorf("Lernmittel, Suche %+q: %d Treffer, erwartet den Titel %q", suchtext, len(treffer), gespeichert)
		}
	}

	// Gegenprobe: Der Leerraum fällt nicht weg, zwei Wörter bleiben zwei.
	if liste, err := repo.ListBooks(ctx, "", "Leerraumprobe LaPeste", false); err != nil || len(liste) != 0 {
		t.Errorf("Titel-Verwaltung ohne Leerzeichen: %d Treffer, %v — erwartet keinen", len(liste), err)
	}
	if treffer := lernmittel("Leerraumprobe LaPeste"); len(treffer) != 0 {
		t.Errorf("Lernmittel ohne Leerzeichen: %d Treffer, erwartet keinen", len(treffer))
	}
}
