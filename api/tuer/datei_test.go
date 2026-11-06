package tuer

import (
	"mime"
	"net/http/httptest"
	"strings"
	"testing"
)

// Der Kopf trägt jeden Namen so, dass der Browser ihn unverändert zurückliest: ein Name aus
// Zeichen ohne Sonderbedeutung bleibt, wie er ist, ein Leerzeichen oder Anführungszeichen steht in
// Anführungszeichen und maskiert, ein Umlaut oder ein Zeilenumbruch kodiert in filename*.
func TestSetzeDateiKopf_KodiertDenNamen(t *testing.T) {
	faelle := []struct {
		name      string
		art       Auslieferung
		dateiname string
		kopf      string
	}{
		{"einfacher Name", Zeigen, "Rechnung.pdf", `inline; filename=Rechnung.pdf`},
		{"Umlaut", Zeigen, "Kontoauszug_Jürgen_Müller.pdf",
			`inline; filename*=utf-8''Kontoauszug_J%C3%BCrgen_M%C3%BCller.pdf`},
		{"Anführungszeichen und Leerzeichen", Speichern, `Bericht "Mai" 2026.pdf`,
			`attachment; filename="Bericht \"Mai\" 2026.pdf"`},
		{"Zeilenumbruch", Speichern, "a\r\nX-Probe: 1.pdf",
			`attachment; filename*=utf-8''a%0D%0AX-Probe%3A%201.pdf`},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			SetzeDateiKopf(rec, ContentTypePDF, f.art, f.dateiname)

			kopf := rec.Header().Get(HeaderContentDisposition)
			if kopf != f.kopf {
				t.Errorf("Content-Disposition = %q, erwartet %q", kopf, f.kopf)
			}
			if strings.ContainsAny(kopf, "\r\n") {
				t.Errorf("Content-Disposition = %q enthält einen Zeilenumbruch", kopf)
			}
			art, werte, err := mime.ParseMediaType(kopf)
			if err != nil {
				t.Fatalf("ParseMediaType(%q): %v", kopf, err)
			}
			if art != string(f.art) || werte["filename"] != f.dateiname {
				t.Errorf("zurückgelesen %q, %q; erwartet %q, %q", art, werte["filename"], f.art, f.dateiname)
			}
			if typ := rec.Header().Get(HeaderContentType); typ != ContentTypePDF {
				t.Errorf("Content-Type = %q, erwartet %q", typ, ContentTypePDF)
			}
		})
	}
}

func TestAntworteDatei_SchicktTypLaengeUndInhalt(t *testing.T) {
	rec := httptest.NewRecorder()
	AntworteDatei(rec, "text/csv; charset=utf-8", Speichern, "liste.csv", []byte("a;b\n"))

	if typ := rec.Header().Get(HeaderContentType); typ != "text/csv; charset=utf-8" {
		t.Errorf("Content-Type = %q", typ)
	}
	if kopf := rec.Header().Get(HeaderContentDisposition); kopf != "attachment; filename=liste.csv" {
		t.Errorf("Content-Disposition = %q", kopf)
	}
	if laenge := rec.Header().Get(HeaderContentLength); laenge != "4" {
		t.Errorf("Content-Length = %q, erwartet 4", laenge)
	}
	if rumpf := rec.Body.String(); rumpf != "a;b\n" {
		t.Errorf("Rumpf = %q", rumpf)
	}
}
