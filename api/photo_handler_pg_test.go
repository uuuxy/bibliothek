package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/crypto"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Foto-Upload über seine Route. Den Dienst prüfen internal/service/photo_service*_test.go;
// hier steht, was die Route dazutut: welche Eingabe sie mit welchem Status abweist und
// dass die Antwort für jede Ausweisnummer lesbar bleibt. Die Kamera-Maske liest die
// Adresse des Bildes aus dieser Antwort.

// fotoJPEG liefert ein kleines JPEG in der Form, in der die Kamera-Maske es schickt.
func fotoJPEG(t *testing.T) string {
	t.Helper()
	bild := image.NewRGBA(image.Rect(0, 0, 12, 16))
	draw.Draw(bild, bild.Bounds(), &image.Uniform{C: color.RGBA{R: 200, G: 40, B: 40, A: 255}}, image.Point{}, draw.Src)
	var puffer bytes.Buffer
	if err := jpeg.Encode(&puffer, bild, nil); err != nil {
		t.Fatalf("JPEG erzeugen: %v", err)
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(puffer.Bytes())
}

func fotoRumpf(t *testing.T, daten string) string {
	t.Helper()
	rumpf, err := json.Marshal(UploadPhotoRequest{PhotoData: daten})
	if err != nil {
		t.Fatal(err)
	}
	return string(rumpf)
}

// fotoHochladen ruft POST /api/schueler/{id}/photo.
func fotoHochladen(t *testing.T, srv *Server, schuelerID, rumpf string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/schueler/"+schuelerID+"/photo", strings.NewReader(rumpf))
	req.SetPathValue("id", schuelerID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.UploadStudentPhotoHandler()(rec, req)
	return rec
}

// fotoAntwort liest die Erfolgsantwort; sie muss JSON sein, sonst scheitert die Maske
// nach einem gespeicherten Bild.
func fotoAntwort(t *testing.T, rec *httptest.ResponseRecorder) (status, adresse string) {
	t.Helper()
	var antwort struct {
		Status string `json:"status"`
		URL    string `json:"url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort ist kein lesbares JSON: %v (%s)", err, rec.Body.String())
	}
	return antwort.Status, antwort.URL
}

func fotoZeilen(t *testing.T, pool *pgxpool.Pool, schuelerID string) int {
	t.Helper()
	return zaehleZeilen(t, pool, `SELECT count(*) FROM schueler_fotos WHERE schueler_id = $1`, schuelerID)
}

// Das Bild liegt danach verschlüsselt in der Datenbank, die Antwort nennt seine Adresse,
// und ein zweites Bild ersetzt das erste.
func TestFotoUpload_SpeichertVerschluesseltUndNenntDieAdresse(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	t.Setenv(crypto.SchluesselVariable, "12345678901234567890123456789012")
	srv := &Server{DB: &db.Database{Pool: pool}}
	schuelerID := seedSchueler(t, pool, "FOTO-1", "Fotokind", "07A")

	rec := fotoHochladen(t, srv, schuelerID, fotoRumpf(t, fotoJPEG(t)))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	if status, adresse := fotoAntwort(t, rec); status != "success" || adresse != "/api/schueler/FOTO-1/photo" {
		t.Errorf("Antwort status=%q url=%q, erwartet success und /api/schueler/FOTO-1/photo", status, adresse)
	}

	var gespeichert []byte
	if err := pool.QueryRow(context.Background(),
		`SELECT foto_encrypted FROM schueler_fotos WHERE schueler_id = $1`, schuelerID).Scan(&gespeichert); err != nil {
		t.Fatalf("gespeichertes Bild lesen: %v", err)
	}
	if bytes.HasPrefix(gespeichert, []byte("RIFF")) {
		t.Error("das Bild liegt unverschlüsselt in der Datenbank")
	}
	klar, err := crypto.Decrypt(gespeichert)
	if err != nil {
		t.Fatalf("gespeichertes Bild entschlüsseln: %v", err)
	}
	if len(klar) < 12 || string(klar[:4]) != "RIFF" || string(klar[8:12]) != "WEBP" {
		t.Errorf("entschlüsselt ist das Bild kein WebP (Anfang %q)", klar[:min(12, len(klar))])
	}

	if rec := fotoHochladen(t, srv, schuelerID, fotoRumpf(t, fotoJPEG(t))); rec.Code != http.StatusOK {
		t.Fatalf("zweites Bild: Status %d: %s", rec.Code, rec.Body.String())
	}
	if n := fotoZeilen(t, pool, schuelerID); n != 1 {
		t.Errorf("%d Bilder nach dem zweiten Upload, erwartet 1", n)
	}
}

// Eine Ausweisnummer wird von Hand eingegeben und darf jedes Zeichen tragen. Die Antwort
// muss auch dann JSON sein: Das Bild ist gespeichert, und die Maske meldete sonst einen
// Fehler.
func TestFotoUpload_AntwortLesbarBeiSonderzeichenInDerAusweisnummer(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	t.Setenv(crypto.SchluesselVariable, "12345678901234567890123456789012")
	srv := &Server{DB: &db.Database{Pool: pool}}
	schuelerID := seedSchueler(t, pool, `FO"TO\2`, "Sonderkind", "07A")

	rec := fotoHochladen(t, srv, schuelerID, fotoRumpf(t, fotoJPEG(t)))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	if status, adresse := fotoAntwort(t, rec); status != "success" || adresse != `/api/schueler/FO"TO\2/photo` {
		t.Errorf("Antwort status=%q url=%q", status, adresse)
	}
}

// Was kein Bild der Kamera-Maske ist, wird als Eingabefehler abgewiesen und nicht
// gespeichert. Eine 500 meldete eine Störung des Servers, die es nicht gibt.
func TestFotoUpload_Abweisungen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	t.Setenv(crypto.SchluesselVariable, "12345678901234567890123456789012")
	srv := &Server{DB: &db.Database{Pool: pool}}
	schuelerID := seedSchueler(t, pool, "FOTO-3", "Abweiskind", "07A")
	keinBild := base64.StdEncoding.EncodeToString([]byte("das ist kein Bild"))

	// meldung: der Teil, den die Maske dem Menschen zeigt; leer heißt, nur der Status zählt.
	const unlesbar = "Bild ließ sich nicht lesen"
	faelle := []struct {
		name, schuelerID, rumpf string
		status                  int
		meldung                 string
	}{
		{"unbekannter Leser", "00000000-0000-0000-0000-000000000000", fotoRumpf(t, fotoJPEG(t)), http.StatusNotFound, "nicht gefunden"},
		{"ohne Kennung", "", fotoRumpf(t, fotoJPEG(t)), http.StatusBadRequest, ""},
		{"kein JSON", schuelerID, `photo_data`, http.StatusBadRequest, ""},
		{"ohne Bild", schuelerID, `{}`, http.StatusBadRequest, ""},
		{"Text statt Bild", schuelerID, fotoRumpf(t, "ein Satz"), http.StatusBadRequest, ""},
		{"PNG statt JPEG", schuelerID, fotoRumpf(t, strings.Replace(fotoJPEG(t), "image/jpeg", "image/png", 1)), http.StatusBadRequest, ""},
		{"kein Base64", schuelerID, fotoRumpf(t, "data:image/jpeg;base64,@@kein-base64@@"), http.StatusBadRequest, unlesbar},
		{"Base64 ohne Bild", schuelerID, fotoRumpf(t, "data:image/jpeg;base64,"+keinBild), http.StatusBadRequest, unlesbar},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			rec := fotoHochladen(t, srv, f.schuelerID, f.rumpf)
			if rec.Code != f.status {
				t.Errorf("Status %d, erwartet %d: %s", rec.Code, f.status, rec.Body.String())
			}
			if meldung := fehlermeldung(t, rec); !strings.Contains(meldung, f.meldung) {
				t.Errorf("Meldung %q, erwartet %q darin", meldung, f.meldung)
			}
		})
	}
	if n := zaehleZeilen(t, pool, `SELECT count(*) FROM schueler_fotos`); n != 0 {
		t.Errorf("%d Bilder nach den Abweisungen gespeichert, erwartet keines", n)
	}
}
