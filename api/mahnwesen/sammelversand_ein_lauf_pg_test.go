package mahnwesen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/mailservice"
	"bibliothek/repository"
)

// Ein Sammelversand dauert je Klasse eine Mail lang. Ein zweiter Start in dieser Zeit, vom
// selben oder von einem anderen Platz, verschickte jede Mail noch einmal; die Tür weist ihn ab,
// bis der erste fertig ist. Die Oberfläche hält den zweiten Klick schon vorher auf
// (frontend/src/lib/stores/sammelversand.svelte.js), ein anderer Platz kommt nur hier vorbei.

// angehaltenePost hält jede Mail an, bis der Test sie freigibt, und meldet, dass eine wartet.
type angehaltenePost struct {
	wartet chan struct{}
	weiter chan struct{}
}

func (p *angehaltenePost) Sende(ctx context.Context, _ mailservice.Nachricht) error {
	select {
	case p.wartet <- struct{}{}:
	default:
	}
	select {
	case <-p.weiter:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *angehaltenePost) IstKonfiguriert(context.Context) bool { return true }

func TestSammelversand_EinZweiterStartWartetAufDenErsten(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	// Ein Kind einer Abschlussklasse mit überfälligem Buch: Mahnliste und Kontoauszug haben je
	// eine Mail an dieselbe Klassenleitung.
	lena := testdaten.LegeSchuelerMitAusleiheAn(t, pool, "S-LAUF-1", "Lena", "9H1")
	inDieVergangenheit(t, pool, lena, 20)
	testdaten.SetzeKlassenleitung(t, pool, "9H1", "leitung9h1@schule.invalid")
	// Die Schreibweise, die das Klassen-Vokabular gespeichert hat; die Oberfläche bietet genau sie an.
	var klasse string
	if err := pool.QueryRow(context.Background(), `SELECT klasse FROM schueler WHERE id = $1::uuid`, lena).Scan(&klasse); err != nil {
		t.Fatalf("Klasse lesen: %v", err)
	}
	rumpf := `{"klassen":["` + klasse + `"]}`

	for _, fall := range []struct {
		name string
		pfad string
		tuer func(*Tueren) http.HandlerFunc
	}{
		{"Mahnlisten", "/api/mail/send-bulk-overdue", func(s *Tueren) http.HandlerFunc {
			return s.MahnlistenVersenden(repository.NewMahnwesenAbfragen(pool))
		}},
		{"Kontoauszüge", "/api/abgaenger/mail", (*Tueren).AbgaengerKontoauszuegeVersenden},
	} {
		t.Run(fall.name, func(t *testing.T) {
			post := &angehaltenePost{wartet: make(chan struct{}, 1), weiter: make(chan struct{})}
			srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}, Uhr: testdaten.SaisonUhr, Mail: post}}
			versand := fall.tuer(srv)
			rufe := func() *httptest.ResponseRecorder {
				rec := httptest.NewRecorder()
				versand(rec, httptest.NewRequest(http.MethodPost, fall.pfad, strings.NewReader(rumpf)))
				return rec
			}

			erster := make(chan *httptest.ResponseRecorder, 1)
			go func() { erster <- rufe() }()
			select {
			case <-post.wartet:
			case rec := <-erster:
				t.Fatalf("der erste Lauf endete vor der ersten Mail: Status %d (%s)", rec.Code, rec.Body.String())
			case <-time.After(10 * time.Second):
				t.Fatal("der erste Lauf kam nicht bis zur Mail")
			}

			// Nebenher: Ohne Sperre käme der zweite Start bis zur angehaltenen Mail, und ein
			// Aufruf auf dieser Zeile hinge, statt rot zu werden.
			zweiter := make(chan *httptest.ResponseRecorder, 1)
			go func() { zweiter <- rufe() }()
			select {
			case rec := <-zweiter:
				if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "läuft bereits") {
					t.Errorf("zweiter Start während des Laufs: Status %d (%s), erwartet %d mit „läuft bereits“",
						rec.Code, rec.Body.String(), http.StatusConflict)
				}
			case <-post.wartet:
				t.Error("zweiter Start während des Laufs: kam bis zur Mail, erwartet die Abweisung")
			case <-time.After(10 * time.Second):
				t.Fatal("der zweite Start antwortete nicht")
			}

			close(post.weiter)
			if rec := <-erster; rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"sent_count":1`) {
				t.Errorf("erster Lauf: Status %d (%s), erwartet %d mit einer versendeten Mail",
					rec.Code, rec.Body.String(), http.StatusOK)
			}
			// Danach ist die Tür wieder offen.
			if rec := rufe(); rec.Code != http.StatusOK {
				t.Errorf("Start nach dem Lauf: Status %d (%s), erwartet %d", rec.Code, rec.Body.String(), http.StatusOK)
			}
		})
	}
}
