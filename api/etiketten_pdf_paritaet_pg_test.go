package api

import (
	"bytes"
	"compress/zlib"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"testing"

	"bibliothek/db"
)

// tjText findet die gezeichneten Textstücke im Inhaltsstrom: `(Ansch.J. 2016 · LMF-Deutsch 5)Tj`.
var tjText = regexp.MustCompile(`\(((?:\\.|[^()\\])*)\)Tj`)

// etikettTexte liest heraus, was auf dem Bogen wirklich GEDRUCKT wird.
//
// Der Umweg über das fertige PDF ist Absicht. Beide bisherigen Etiketten-Bugs waren an
// den Zwischenschichten grün: Das Feld stand im Struct, die Abfrage lieferte es — nur
// gezeichnet wurde es nicht bzw. kam auf dem zweiten Weg nie im Struct an. Erst der
// Inhaltsstrom beantwortet die Frage, die die Bibliothek stellt: Was klebt am Buch?
//
// gofpdf komprimiert die Ströme (FlateDecode); im Rohbyte-PDF steht der Text deshalb
// nicht. Nach dem Inflaten liegt er als `(…)Tj` vor. Sonderzeichen bleiben in der
// Kodierung des Erzeugers (Windows-1252) — für den Vergleich zweier Bögen und die Suche
// nach ASCII-Signaturen ist das gleichgültig, beide Seiten werden gleich behandelt.
func etikettTexte(t *testing.T, roh []byte) []string {
	t.Helper()

	var inhalt bytes.Buffer
	rest := roh
	for {
		i := bytes.Index(rest, []byte("stream"))
		if i < 0 {
			break
		}
		body := bytes.TrimLeft(rest[i+len("stream"):], "\r\n")
		j := bytes.Index(body, []byte("endstream"))
		if j < 0 {
			break
		}
		// Nicht jeder Strom ist ein komprimierter Inhaltsstrom (Schriften, Bilder) — was
		// zlib nicht öffnet, wird übersprungen. Was es öffnet, muss aber vollständig
		// lesbar sein, sonst prüfte der Test nur den Teil, den er zufällig entpacken konnte.
		zr, err := zlib.NewReader(bytes.NewReader(body[:j]))
		if err == nil {
			entpackt, leseErr := io.ReadAll(zr)
			if leseErr != nil {
				t.Fatalf("Inhaltsstrom nicht vollständig lesbar: %v", leseErr)
			}
			inhalt.Write(entpackt)
			if closeErr := zr.Close(); closeErr != nil {
				t.Fatalf("Inhaltsstrom nicht sauber abgeschlossen: %v", closeErr)
			}
		}
		rest = body[j:]
	}
	if inhalt.Len() == 0 {
		t.Fatalf("kein lesbarer Inhaltsstrom im PDF (%d Bytes) — Textextraktion kaputt, "+
			"der Test würde ab hier alles durchwinken", len(roh))
	}

	var texte []string
	for _, treffer := range tjText.FindAllSubmatch(inhalt.Bytes(), -1) {
		if s := strings.TrimSpace(string(treffer[1])); s != "" {
			texte = append(texte, s)
		}
	}
	sort.Strings(texte)
	return texte
}

// Beide Druckwege müssen denselben Aufkleber erzeugen.
//
// Es gibt zwei Wege zum selben Etikett, und sie holen ihre Daten verschieden:
//
//   - GET /api/buecher/titel/{id}/etiketten — „Barcodes drucken" im Buchformular,
//     liest alles selbst aus der DB (queryLabelItems).
//   - POST /api/print/labels — die Etiketten-Verwaltung im Druck-Center. Sie schickt nur
//     Barcode, Titel und Autor; alles Weitere muss der Server nachtragen.
//
// Genau an dieser Naht ist die Signatur verlorengegangen: Nachgetragen wurde zuerst nur
// das Anschaffungsjahr, also trug dasselbe Buch je nach gedrücktem Knopf einen anderen
// Aufkleber — und beide Wege lieferten HTTP 200 mit einem tadellosen PDF. Kein Test auf
// Statuscode oder Dateigröße kann das sehen; der bestehende Paritätstest sah es auch
// nicht, weil er zwei DB-Abfragen vergleicht und dieser Weg über den Browser läuft.
//
// Deshalb: beide Handler über echtes HTTP fahren, das POST mit exakt der Nutzlast aus
// labels.svelte.js, und die gedruckten Texte vergleichen.
func TestEtikettenWegeDruckenDasselbe(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	const (
		signatur  = "LMF-Deutsch 5"
		buchtitel = "Seydlitz Geographie"
		autor     = "Klaus Berger"
		format    = "avery_3475"
	)

	titelID := titelMitSignatur(t, pool, buchtitel, signatur, 0)
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET autor = $1 WHERE id = $2`, autor, titelID); err != nil {
		t.Fatalf("Autor setzen: %v", err)
	}
	exemplar(t, pool, titelID, "B-ETI-1", true, "")
	exemplar(t, pool, titelID, "B-ETI-2", true, "")

	// Weg 1: Buchformular — der Server liest alles selbst.
	reqGet := httptest.NewRequest(http.MethodGet, "/api/buecher/titel/x/etiketten?format="+format+"&start=1", nil)
	reqGet.SetPathValue("id", titelID)
	recGet := httptest.NewRecorder()
	srv.LabelsHandler()(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("Buchformular-Weg: Status %d — %s", recGet.Code, recGet.Body.String())
	}

	// Weg 2: Druck-Center — genau die Nutzlast aus labels.svelte.js (triggerPrint):
	// nur BarcodeID, Titel und Autor. Wird hier ein Feld ergänzt, gehört es NICHT hierher,
	// sondern in ergaenzeServerfelder.
	payload := `{
		"formatId": "` + format + `",
		"startPosition": 1,
		"isQR": false,
		"items": [
			{"BarcodeID": "B-ETI-1", "Titel": "` + buchtitel + `", "Autor": "` + autor + `"},
			{"BarcodeID": "B-ETI-2", "Titel": "` + buchtitel + `", "Autor": "` + autor + `"}
		]
	}`
	reqPost := httptest.NewRequest(http.MethodPost, "/api/print/labels", strings.NewReader(payload))
	recPost := httptest.NewRecorder()
	srv.PrintLabelsHandler()(recPost, reqPost)
	if recPost.Code != http.StatusOK {
		t.Fatalf("Druck-Center-Weg: Status %d — %s", recPost.Code, recPost.Body.String())
	}

	buchformular := etikettTexte(t, recGet.Body.Bytes())
	druckzentrum := etikettTexte(t, recPost.Body.Bytes())

	// Erst die Gegenprobe gegen den stillen Durchmarsch: Stünde die Signatur auf KEINEM
	// der beiden Bögen, wären sie immer noch gleich — und der Vergleich unten grün,
	// obwohl genau das der Fehler ist, den dieser Test fangen soll.
	for _, fall := range []struct {
		weg   string
		texte []string
	}{{"Buchformular", buchformular}, {"Druck-Center", druckzentrum}} {
		if !strings.Contains(strings.Join(fall.texte, "\n"), signatur) {
			t.Errorf("%s-Weg druckt die Signatur %q nicht.\nGedruckt wurde:\n  %s",
				fall.weg, signatur, strings.Join(fall.texte, "\n  "))
		}
	}

	if len(buchformular) != len(druckzentrum) {
		t.Fatalf("Bögen tragen verschieden viele Textzeilen — Buchformular %d, Druck-Center %d:\n"+
			"Buchformular:\n  %s\nDruck-Center:\n  %s",
			len(buchformular), len(druckzentrum),
			strings.Join(buchformular, "\n  "), strings.Join(druckzentrum, "\n  "))
	}
	for i := range buchformular {
		if buchformular[i] != druckzentrum[i] {
			t.Errorf("Aufkleber unterscheiden sich: Buchformular %q, Druck-Center %q.\n"+
				"→ Dasselbe Buch bekäme je nach gedrücktem Knopf ein anderes Etikett. Ein Feld, "+
				"das der Server kennt, gehört in ergaenzeServerfelder — nicht in die Nutzlast "+
				"der Oberfläche.", buchformular[i], druckzentrum[i])
		}
	}
}
