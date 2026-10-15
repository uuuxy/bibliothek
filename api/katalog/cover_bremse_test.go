package katalog

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// quellenZaehler zählt die Abfragen der Quellen eines Covers. Sie kommen erst hinter der Bremse;
// jede scheitert, damit kein Download folgt und die Schleife schneller läuft, als die Bremse
// nachfüllt.
type quellenZaehler struct {
	db.Pool
	abfragen int
}

func (z *quellenZaehler) Query(context.Context, string, ...any) (pgx.Rows, error) {
	z.abfragen++
	return nil, errors.New("attrappe: keine Quelle")
}

// Die Bremse gehört der Tür, die Cover liefert: Über alle Anfragen an sie lässt sie von einer
// Adresse coverDownloadsProSekunde Fehltreffer zur Prüfung der Quelle durch, den nächsten nicht.
// Eine zweite Tür hat ihren eigenen Vorrat; im Betrieb baut der Router genau eine.
func TestCover_BremseGehoertDerTuer(t *testing.T) {
	zaehler := &quellenZaehler{}
	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: zaehler}}, cover: coverAbruf{verzeichnis: t.TempDir()}}
	hole := func(tuerCover http.HandlerFunc) {
		req := httptest.NewRequest(http.MethodGet, "/api/images/cover?isbn="+testISBN+
			"&url=https://covers.openlibrary.org/b/isbn/"+testISBN+"-L.jpg", nil)
		req.RemoteAddr = "198.51.100.30:4000"
		tuerCover(httptest.NewRecorder(), req)
	}

	erste := s.Cover()
	for range coverDownloadsProSekunde + 1 {
		hole(erste)
	}
	if zaehler.abfragen != coverDownloadsProSekunde {
		t.Errorf("Cover: %d von %d Fehltreffern bis zur Prüfung der Quelle, erwartet %d",
			zaehler.abfragen, coverDownloadsProSekunde+1, coverDownloadsProSekunde)
	}

	zaehler.abfragen = 0
	hole(s.Cover())
	if zaehler.abfragen != 1 {
		t.Errorf("Cover an einer zweiten Tür: %d Fehltreffer bis zur Prüfung der Quelle, erwartet 1; "+
			"sie teilt den Vorrat der ersten", zaehler.abfragen)
	}
}
