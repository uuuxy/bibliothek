package inventur

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"sort"
	"strings"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// BearbeiteBuecherListe verarbeitet GET-Anfragen für die Bücherübersicht.
// Die Funktion liest Suchparameter (Fach, Klasse, Suchbegriff) aus,
// nutzt ein Wörterbuch für Synonyme (z.B. powi -> politik) und fragt die Datenbank ab.
// Danach stehen die Bücher nach dem Titel (sortiereBuecherNachTitel) und gehen als JSON hinaus.
func (handler *APIHandler) BearbeiteBuecherListe(antwort http.ResponseWriter, anfrage *http.Request) {
	anfrageParameter := anfrage.URL.Query()
	fach := strings.TrimSpace(anfrageParameter.Get("subject"))

	suchbegriff := strings.TrimSpace(strings.ToLower(anfrageParameter.Get("q")))
	if len(suchbegriff) > 200 {
		writeError(antwort, http.StatusBadRequest, "suchbegriff zu lang (max. 200 zeichen)")
		return
	}

	if uebersetzt, existiert := suchSynonyme[suchbegriff]; existiert {
		suchbegriff = uebersetzt
	}

	// bestand=ohne ist die Aufräumsicht der Verwaltung: nur Titel ohne ein einziges
	// nicht ausgesondertes Exemplar. Ohne den Parameter ist es der Katalog.
	sicht := strings.TrimSpace(anfrageParameter.Get("bestand"))
	if sicht != "" && sicht != "ohne" {
		writeError(antwort, http.StatusBadRequest, "ungültiger query-parameter bestand (erlaubt: ohne)")
		return
	}
	buecher, fehler := handler.repo.ListBooks(anfrage.Context(), fach, suchbegriff, sicht == "ohne")
	if fehler != nil {
		log.Printf("Fehler beim Laden der Bücherliste: %v", fehler)
		writeError(antwort, http.StatusInternalServerError, "Interner Serverfehler beim Laden der Bücher")
		return
	}

	sortiereBuecherNachTitel(buecher)

	writeJSON(antwort, http.StatusOK, map[string]any{"data": buecher})
}

// Wörterbuch für Synonyme bei der Suche
var suchSynonyme = map[string]string{
	"powi":  "politik",
	"mathe": "mathematik",
	"eng":   "englisch",
	"deu":   "deutsch",
	"franz": "französisch",
	"bio":   "biologie",
	"che":   "chemie",
	"phy":   "physik",
	"geo":   "geographie",
	"info":  "informatik",
	"lat":   "latein",
	"span":  "spanisch",
	"rel":   "religion",
	"reli":  "religion",
}

// sortiereBuecherNachTitel ordnet die Liste nach dem Titel, wie ein deutsches Register:
// Umlaute stehen bei ihrem Grundbuchstaben, Zahlen nach ihrer Größe („Teil 2" vor „Teil 10"),
// Groß- und Kleinschreibung trennt keine Buchstaben. Dieselbe Reihenfolge gibt der Browser
// mit Intl.Collator('de', {numeric: true}), nach dem die Listen der Oberfläche ordnen; beide
// Seiten prüft titelReihenfolge.faelle.json. Gleiche Titel behalten die Reihenfolge der
// Abfrage.
//
// Der Collator entsteht je Aufruf: Er hält Zustand und darf nicht zwei Anfragen zugleich
// dienen.
func sortiereBuecherNachTitel(buecher []Book) {
	if len(buecher) <= 1 {
		return
	}

	ordnung := collate.New(language.German)
	var puffer collate.Buffer
	schluessel := make([][]byte, len(buecher))
	reihenfolge := make([]int, len(buecher))
	for i := range buecher {
		titel := zahlenNachGroesse(strings.TrimSpace(buecher[i].Title))
		schluessel[i] = ordnung.KeyFromString(&puffer, titel)
		reihenfolge[i] = i
	}
	sort.SliceStable(reihenfolge, func(a, b int) bool {
		return bytes.Compare(schluessel[reihenfolge[a]], schluessel[reihenfolge[b]]) < 0
	})

	sortiert := make([]Book, len(buecher))
	for ziel, quelle := range reihenfolge {
		sortiert[ziel] = buecher[quelle]
	}
	copy(buecher, sortiert)
}

// zahlenNachGroesse schreibt jede Ziffernfolge so um, dass die Buchstabenfolge sie nach ihrer
// Größe ordnet: Führende Nullen fallen weg, davor steht zweistellig die Zahl der Stellen
// („5" wird „015", „10" wird „0210").
//
// collate.Numeric leistet das nicht: Es ordnet die Zahl 0 hinter jede andere, sobald Text
// folgt („Heft 5 …" vor „Heft 0 …").
func zahlenNachGroesse(text string) string {
	var umgeschrieben strings.Builder
	umgeschrieben.Grow(len(text) + 8)
	for i := 0; i < len(text); {
		if !istZiffer(text[i]) {
			umgeschrieben.WriteByte(text[i])
			i++
			continue
		}
		ende := i
		for ende < len(text) && istZiffer(text[ende]) {
			ende++
		}
		ziffern := strings.TrimLeft(text[i:ende], "0")
		if ziffern == "" {
			ziffern = "0"
		}
		stellen := min(len(ziffern), 99)
		umgeschrieben.WriteByte(byte('0' + stellen/10))
		umgeschrieben.WriteByte(byte('0' + stellen%10))
		umgeschrieben.WriteString(ziffern)
		i = ende
	}
	return umgeschrieben.String()
}

func istZiffer(zeichen byte) bool { return zeichen >= '0' && zeichen <= '9' }

// BearbeiteBuchLesen verarbeitet GET-Anfragen für ein einzelnes Buch.
func (handler *APIHandler) BearbeiteBuchLesen(antwort http.ResponseWriter, anfrage *http.Request) {
	id := anfrage.PathValue("id")
	if id == "" {
		writeError(antwort, http.StatusBadRequest, "ID fehlt")
		return
	}

	// Die Maske „Titel bearbeiten" lädt hierüber und schickt das Ganze per PUT zurück.
	// Scheitert das Nachladen der Schlagworte, antwortet der Einzel-Read mit einem Fehler
	// statt ohne sie: Die Maske öffnet dann gar nicht, statt mit einem leeren Feld, das beim
	// Speichern als Aussage „keine Schlagworte" zurückkäme.
	buch, fehler := handler.ladeTitelVoll(anfrage.Context(), id)
	if errors.Is(fehler, ErrBookNotFound) {
		writeError(antwort, http.StatusNotFound, "Buch nicht gefunden")
		return
	}
	if fehler != nil {
		log.Printf("Fehler beim Laden des Buches: %v", fehler)
		writeError(antwort, http.StatusInternalServerError, "Interner Serverfehler")
		return
	}

	writeJSON(antwort, http.StatusOK, buch)
}

// ladeTitelVoll liest einen Titel, wie die Maske ihn bekommt: mit den Bestandszahlen und den
// Schlagworten. Der Einzel-Read und die Antwort auf das Speichern haben damit dieselbe Form.
func (handler *APIHandler) ladeTitelVoll(ctx context.Context, id string) (Book, error) {
	buecher, err := handler.repo.ListBooksByIDs(ctx, []string{id})
	if err != nil {
		return Book{}, err
	}
	if len(buecher) == 0 {
		return Book{}, ErrBookNotFound
	}
	buch := buecher[0]
	buch.Schlagworte, err = handler.repo.SchlagworteDesTitels(ctx, id)
	if err != nil {
		return Book{}, err
	}
	return buch, nil
}

// gespeichert ist die Antwort auf Anlegen und Ändern: der Titel, wie er jetzt in der Datenbank
// steht. Die Titelliste ersetzt ihre Zeile durch die Antwort; aus den gesendeten Angaben
// allein stünde dort der Bestand 0. Scheitert das Nachlesen, bleibt es bei den gesendeten
// Angaben, denn gespeichert ist der Titel; nach einem Ändern sind das nur die Felder, die
// der Rumpf genannt hat.
func (handler *APIHandler) gespeichert(ctx context.Context, gesendet Book) Book {
	buch, err := handler.ladeTitelVoll(ctx, gesendet.ID)
	if err != nil {
		log.Printf("Titel %s nach dem Speichern nicht lesbar: %v", gesendet.ID, err)
		return gesendet
	}
	return buch
}

// BearbeiteBuchVorhanden sagt vor dem Speichern, was die Dublettenkontrolle zu einer ISBN
// sagen wird: den Titel, der sie schon trägt, samt der Meldung der Ablehnung — oder keinen.
// Die Maske „Neues Buch" fragt damit schon bei der Eingabe der ISBN, wie Littera; das Format
// der Nummer prüft erst das Speichern. Die zehn- und die dreizehnstellige Form einer ISBN sind
// dabei dieselbe Nummer (isbn_normalform).
func (handler *APIHandler) BearbeiteBuchVorhanden(antwort http.ResponseWriter, anfrage *http.Request) {
	isbn := strings.TrimSpace(anfrage.URL.Query().Get("isbn"))
	if isbn == "" {
		writeError(antwort, http.StatusBadRequest, "isbn fehlt")
		return
	}

	vorhanden, fehler := handler.repo.TitelMitISBN(anfrage.Context(), isbn)
	if fehler != nil {
		log.Printf("Dublettenkontrolle vorab: %v", fehler)
		writeError(antwort, http.StatusInternalServerError, "Interner Serverfehler")
		return
	}
	if vorhanden != nil {
		writeJSON(antwort, http.StatusOK, map[string]any{"data": map[string]any{
			"vorhanden": vorhanden.alsAntwort(),
			"meldung":   vorhanden.Meldung(),
		}})
		return
	}

	writeJSON(antwort, http.StatusOK, map[string]any{"data": map[string]any{"vorhanden": nil}})
}
