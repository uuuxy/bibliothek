package service

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/inventur"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// processCover holt für einen Titel der Auswahl das Cover und schreibt Ergebnis und Stand.
// Gemessen am 08.10.2026 führte kein Test die Funktion aus (cover_service.go 18,2 %,
// OFFEN.md 5.10).
//
// Echte Datenbank, nachgestellte Katalogdienste. Die Cover-Datei landet in einem
// Wegwerf-Verzeichnis (t.Chdir), nicht im Paket.

const dnbAntwortMitSatz = `<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/">
  <numberOfRecords>1</numberOfRecords>
  <records><record><recordData>
    <record xmlns="http://www.loc.gov/MARC21/slim">
      <datafield tag="245" ind1="1" ind2="0"><subfield code="a">Cover-Probe</subfield></datafield>
      <datafield tag="100" ind1="1" ind2=" "><subfield code="a">Muster, Erika</subfield></datafield>
    </record>
  </recordData></record></records>
</searchRetrieveResponse>`

const dnbAntwortOhneSatz = `<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/">
  <numberOfRecords>0</numberOfRecords><records></records>
</searchRetrieveResponse>`

// katalog stellt die Katalogdienste nach: die DNB mit oder ohne Satz zur ISBN und ihren
// Cover-Dienst mit oder ohne Bild. Google Books und OpenLibrary antworten, kennen aber nichts.
type katalog struct {
	ausfall bool   // kein Dienst ist erreichbar
	dnbSatz bool   // die DNB kennt die ISBN
	cover   []byte // das Bild des DNB-Cover-Dienstes; nil = keines
}

func (k katalog) client() *inventur.MetadatenClient {
	c := inventur.NeuerMetadatenClient()
	c.SetzeHTTPClientFuerTest(&http.Client{Transport: &orderMockTransport{roundTripFunc: k.antworte}})
	return c
}

func (k katalog) antworte(req *http.Request) (*http.Response, error) {
	if k.ausfall {
		return nil, errors.New("Testnetz: Dienst nicht erreichbar")
	}
	antwort := func(status int, typ string, rumpf []byte) (*http.Response, error) {
		kopf := make(http.Header)
		kopf.Set("Content-Type", typ)
		return &http.Response{StatusCode: status, Header: kopf, Body: io.NopCloser(bytes.NewReader(rumpf)), Request: req}, nil
	}
	switch {
	case req.URL.Host == "services.dnb.de" && k.dnbSatz:
		return antwort(http.StatusOK, "application/xml", []byte(dnbAntwortMitSatz))
	case req.URL.Host == "services.dnb.de":
		return antwort(http.StatusOK, "application/xml", []byte(dnbAntwortOhneSatz))
	case req.URL.Host == "portal.dnb.de" && k.cover != nil:
		return antwort(http.StatusOK, "image/png", k.cover)
	}
	return antwort(http.StatusNotFound, "text/plain", nil)
}

// probeBild ist ein dekodierbares Bild in der Größe eines kleinen Covers.
func probeBild(t *testing.T) []byte {
	t.Helper()
	bild := image.NewRGBA(image.Rect(0, 0, 60, 90))
	for y := range 90 {
		for x := range 60 {
			bild.Set(x, y, color.RGBA{R: uint8(x * 4), G: uint8(y * 2), B: 120, A: 255})
		}
	}
	var puffer bytes.Buffer
	if err := png.Encode(&puffer, bild); err != nil {
		t.Fatal(err)
	}
	return puffer.Bytes()
}

// coverTitel legt einen Titel in dem Stand an, in dem ihn die Auswahl des Abgleichs findet.
func coverTitel(t *testing.T, pool *pgxpool.Pool, isbn, status, url string) missingCover {
	t.Helper()
	var mc missingCover
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO buecher_titel (titel, isbn, cover_status, cover_url)
		VALUES ('Cover-Probe ' || $1, $1, $2, NULLIF($3, ''))
		ON CONFLICT (isbn) DO UPDATE SET cover_status = EXCLUDED.cover_status, cover_url = EXCLUDED.cover_url
		RETURNING id, isbn`, isbn, status, url).Scan(&mc.ID, &mc.ISBN); err != nil {
		t.Fatalf("Titel %s anlegen: %v", isbn, err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE id = $1`, mc.ID); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})
	return mc
}

func coverStand(t *testing.T, pool *pgxpool.Pool, id string) (status, url string) {
	t.Helper()
	if err := pool.QueryRow(context.Background(),
		`SELECT cover_status, COALESCE(cover_url, '') FROM buecher_titel WHERE id = $1`, id).Scan(&status, &url); err != nil {
		t.Fatal(err)
	}
	return status, url
}

// inDerAuswahl sagt, ob der nächste Lauf des Abgleichs den Titel wieder anfasst.
func inDerAuswahl(t *testing.T, pool *pgxpool.Pool, id string) bool {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM (`+repository.CoverAbgleichAuswahl+`) a WHERE a.id = $1`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n == 1
}

func verarbeite(pool *pgxpool.Pool, k katalog, mc missingCover) (gefunden, ohneCover, gescheitert int64) {
	var found, notFound, failed atomic.Int64
	NewCoverService(pool).processCover(context.Background(), k.client(), mc, &found, &notFound, &failed)
	return found.Load(), notFound.Load(), failed.Load()
}

func TestProcessCover_SchreibtErgebnisUndStand(t *testing.T) {
	pool := pgtest.Pool(t)
	t.Chdir(t.TempDir())
	bild := probeBild(t)

	t.Run("Titel und Cover gefunden: lokale Datei, FOUND, nicht wieder in der Auswahl", func(t *testing.T) {
		mc := coverTitel(t, pool, "9789991940014", "PENDING", "")
		if gefunden, _, _ := verarbeite(pool, katalog{dnbSatz: true, cover: bild}, mc); gefunden != 1 {
			t.Errorf("als gefunden gezählt: %d, erwartet 1", gefunden)
		}
		status, url := coverStand(t, pool, mc.ID)
		if status != "FOUND" || !strings.HasPrefix(url, "/uploads/cover_auto_") {
			t.Fatalf("Stand %q, Cover %q — erwartet FOUND und eine Datei unter /uploads/cover_auto_…", status, url)
		}
		if _, err := os.Stat(strings.TrimPrefix(url, "/")); err != nil {
			t.Errorf("die Cover-Datei liegt nicht im Ablageordner: %v", err)
		}
		if inDerAuswahl(t, pool, mc.ID) {
			t.Error("der Titel steht mit lokalem Cover weiter in der Auswahl")
		}
	})

	t.Run("fremde Cover-Adresse wird durch die lokale Datei ersetzt", func(t *testing.T) {
		mc := coverTitel(t, pool, "9789991940021", "FOUND", "https://portal.dnb.de/opac/mvb/cover?isbn=9789991940021")
		verarbeite(pool, katalog{dnbSatz: true, cover: bild}, mc)
		if status, url := coverStand(t, pool, mc.ID); status != "FOUND" || !strings.HasPrefix(url, "/uploads/cover_auto_") {
			t.Errorf("Stand %q, Cover %q — erwartet FOUND und eine lokale Datei", status, url)
		}
	})

	t.Run("Titel bekannt, kein Cover: NOT_FOUND, nicht wieder in der Auswahl", func(t *testing.T) {
		mc := coverTitel(t, pool, "9789991940038", "PENDING", "")
		if _, ohneCover, _ := verarbeite(pool, katalog{dnbSatz: true}, mc); ohneCover != 1 {
			t.Errorf("als „ohne Cover“ gezählt: %d, erwartet 1", ohneCover)
		}
		if status, url := coverStand(t, pool, mc.ID); status != "NOT_FOUND" || url != "" {
			t.Errorf("Stand %q, Cover %q — erwartet NOT_FOUND ohne Cover", status, url)
		}
		if inDerAuswahl(t, pool, mc.ID) {
			t.Error("ein Titel ohne Cover bei allen Diensten steht weiter in der Auswahl")
		}
	})

	// Ein Ausfall darf einen Titel nicht abschreiben: Der nächste Lauf fragt wieder.
	t.Run("Dienste nicht erreichbar: FAILED, beim nächsten Lauf wieder in der Auswahl", func(t *testing.T) {
		mc := coverTitel(t, pool, "9789991940045", "PENDING", "")
		if _, _, gescheitert := verarbeite(pool, katalog{ausfall: true}, mc); gescheitert != 1 {
			t.Errorf("als gescheitert gezählt: %d, erwartet 1", gescheitert)
		}
		if status, url := coverStand(t, pool, mc.ID); status != "FAILED" || url != "" {
			t.Errorf("Stand %q, Cover %q — erwartet FAILED ohne Cover", status, url)
		}
		if !inDerAuswahl(t, pool, mc.ID) {
			t.Error("nach einem Ausfall der Dienste fragt der nächste Lauf den Titel nicht wieder")
		}
	})

	t.Run("kein Dienst kennt die ISBN: FAILED", func(t *testing.T) {
		mc := coverTitel(t, pool, "9789991940052", "PENDING", "")
		verarbeite(pool, katalog{}, mc)
		if status, _ := coverStand(t, pool, mc.ID); status != "FAILED" {
			t.Errorf("Stand %q, erwartet FAILED", status)
		}
	})
}

// Zwischen der Auswahl und dem Schreiben liegt die Laufzeit des Abgleichs, beim Altbestand
// Stunden. Lädt in dieser Zeit jemand ein Cover von Hand hoch, bleibt es stehen, mit seinem
// Stand — gleich, was der Abgleich für den Titel findet.
func TestProcessCover_HandUploadWaehrendDesLaufsBleibt(t *testing.T) {
	pool := pgtest.Pool(t)
	t.Chdir(t.TempDir())
	const handCover = "/uploads/covers/von-hand.webp"

	faelle := map[string]struct {
		isbn string
		k    katalog
	}{
		"der Abgleich findet ein Cover":     {"9789991940069", katalog{dnbSatz: true, cover: probeBild(t)}},
		"der Abgleich findet keines":        {"9789991940076", katalog{dnbSatz: true}},
		"die Dienste sind nicht erreichbar": {"9789991940014", katalog{ausfall: true}},
		"kein Dienst kennt die ISBN":        {"9789991940021", katalog{}},
	}
	for name, f := range faelle {
		t.Run(name, func(t *testing.T) {
			mc := coverTitel(t, pool, f.isbn, "PENDING", "")
			if !inDerAuswahl(t, pool, mc.ID) {
				t.Fatal("Gegenprobe: Der Titel steht vor dem Hand-Upload nicht in der Auswahl — der Test misst nichts")
			}
			// Der Hand-Upload, wie ihn inventur.UpdateBookMetadata schreibt.
			if _, err := pool.Exec(context.Background(),
				`UPDATE buecher_titel SET cover_url = $2, cover_status = 'FOUND' WHERE id = $1`, mc.ID, handCover); err != nil {
				t.Fatal(err)
			}
			verarbeite(pool, f.k, mc)
			if status, url := coverStand(t, pool, mc.ID); status != "FOUND" || url != handCover {
				t.Errorf("nach dem Abgleich: Stand %q, Cover %q — erwartet FOUND und das von Hand hochgeladene Cover", status, url)
			}
		})
	}
}

// Ein ganzer Lauf fasst jeden Titel der Auswahl an, lässt die übrigen in Ruhe und gibt den
// Lauf danach wieder frei.
func TestSyncMissingCovers_EinLaufUeberDieAuswahl(t *testing.T) {
	pool := pgtest.Pool(t)
	t.Chdir(t.TempDir())
	coverSyncRunning.Store(false)
	t.Cleanup(func() { coverSyncRunning.Store(false) })
	const handCover = "/uploads/covers/von-hand.webp"

	offen := coverTitel(t, pool, "9789991940014", "PENDING", "")
	gescheitert := coverTitel(t, pool, "9789991940021", "FAILED", "")
	vonHand := coverTitel(t, pool, "9789991940038", "PENDING", handCover)
	abgeschrieben := coverTitel(t, pool, "9789991940045", "NOT_FOUND", "")

	// Die Auswahl läuft über die ganze Tabelle, und je Titel wartet der Lauf auf seine
	// Drossel. Reste anderer Tests machten ihn lang; dann lieber gleich scheitern.
	var inAuswahl int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM (`+repository.CoverAbgleichAuswahl+`) a`).Scan(&inAuswahl); err != nil {
		t.Fatal(err)
	}
	if inAuswahl > 10 {
		t.Fatalf("%d Titel in der Auswahl, erwartet die zwei dieses Tests und höchstens wenige Reste", inAuswahl)
	}

	svc := NewCoverService(pool)
	svc.katalog = katalog{dnbSatz: true, cover: probeBild(t)}.client
	svc.SyncMissingCoversAsync()

	for name, mc := range map[string]missingCover{"unversucht": offen, "zuvor gescheitert": gescheitert} {
		if status, url := coverStand(t, pool, mc.ID); status != "FOUND" || !strings.HasPrefix(url, "/uploads/cover_auto_") {
			t.Errorf("%s: Stand %q, Cover %q — erwartet FOUND und eine lokale Datei", name, status, url)
		}
	}
	if status, url := coverStand(t, pool, vonHand.ID); status != "PENDING" || url != handCover {
		t.Errorf("Titel mit Cover von Hand: Stand %q, Cover %q — der Lauf hat ihn angefasst", status, url)
	}
	if status, url := coverStand(t, pool, abgeschrieben.ID); status != "NOT_FOUND" || url != "" {
		t.Errorf("Titel ohne Cover bei allen Diensten: Stand %q, Cover %q — der Lauf hat ihn erneut gefragt", status, url)
	}
	if coverSyncRunning.Load() {
		t.Error("nach dem Lauf steht der Merker noch: Jeder weitere Lauf würde übersprungen")
	}
}

// Start, Zeitplan und Handauslösung bauen je einen eigenen Dienst und teilen sich nur den
// Merker. Ein zweiter Lauf, während einer läuft, tut nichts und lässt den Merker stehen.
func TestSyncMissingCovers_ZweiterLaufWirdUebersprungen(t *testing.T) {
	pool := pgtest.Pool(t)
	coverSyncRunning.Store(true)
	t.Cleanup(func() { coverSyncRunning.Store(false) })
	offen := coverTitel(t, pool, "9789991940052", "PENDING", "")

	svc := NewCoverService(pool)
	svc.katalog = func() *inventur.MetadatenClient {
		t.Error("der übersprungene Lauf fragt die Katalogdienste")
		return katalog{}.client()
	}
	svc.SyncMissingCoversAsync()

	if status, _ := coverStand(t, pool, offen.ID); status != "PENDING" {
		t.Errorf("Stand %q — der übersprungene Lauf hat den Titel angefasst", status)
	}
	if !coverSyncRunning.Load() {
		t.Error("der übersprungene Lauf hat den Merker des laufenden zurückgesetzt")
	}
}
