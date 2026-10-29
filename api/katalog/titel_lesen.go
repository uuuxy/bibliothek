package katalog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// TitelListe verarbeitet GET-Anfragen für die Bücherübersicht.
// Die Funktion liest Suchparameter (Fach, Klasse, Suchbegriff) aus,
// nutzt ein Wörterbuch für Synonyme (z.B. powi -> politik) und fragt die Datenbank ab.
// Danach stehen die Bücher nach dem Titel (sortiereBuecherNachTitel) und gehen als JSON hinaus.
func (t *Tueren) TitelListe() http.HandlerFunc {
	return func(antwort http.ResponseWriter, anfrage *http.Request) {
		anfrageParameter := anfrage.URL.Query()
		fach := strings.TrimSpace(anfrageParameter.Get("subject"))

		suchbegriff := strings.TrimSpace(strings.ToLower(anfrageParameter.Get("q")))
		if len(suchbegriff) > 200 {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, "Der Suchbegriff ist zu lang, höchstens 200 Zeichen", nil)
			return
		}

		if uebersetzt, existiert := suchSynonyme[suchbegriff]; existiert {
			suchbegriff = uebersetzt
		}

		// bestand=ohne ist die Aufräumsicht der Verwaltung: nur Titel ohne ein einziges
		// nicht ausgesondertes Exemplar. Ohne den Parameter ist es der Katalog.
		sicht := strings.TrimSpace(anfrageParameter.Get("bestand"))
		if sicht != "" && sicht != "ohne" {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, "Unbekannter Wert für den Bestand", nil)
			return
		}
		buecher, fehler := t.titelAbfragen().LadeTitelListe(anfrage.Context(), fach, suchbegriff, sicht == "ohne")
		if fehler != nil {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusInternalServerError, "Interner Serverfehler beim Laden der Bücher", fmt.Errorf("bücherliste laden: %w", fehler))
			return
		}

		sortiereBuecherNachTitel(buecher)

		tuer.AntworteJSON(antwort, http.StatusOK, buecher)
	}
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
func sortiereBuecherNachTitel(buecher []repository.KatalogTitel) {
	if len(buecher) <= 1 {
		return
	}

	ordnung := collate.New(language.German)
	var puffer collate.Buffer
	schluessel := make([][]byte, len(buecher))
	reihenfolge := make([]int, len(buecher))
	for i := range buecher {
		titel := zahlenNachGroesse(strings.TrimSpace(buecher[i].Titel))
		schluessel[i] = ordnung.KeyFromString(&puffer, titel)
		reihenfolge[i] = i
	}
	sort.SliceStable(reihenfolge, func(a, b int) bool {
		return bytes.Compare(schluessel[reihenfolge[a]], schluessel[reihenfolge[b]]) < 0
	})

	sortiert := make([]repository.KatalogTitel, len(buecher))
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

// Titel verarbeitet GET-Anfragen für ein einzelnes Buch.
func (t *Tueren) Titel() http.HandlerFunc {
	return func(antwort http.ResponseWriter, anfrage *http.Request) {
		id := anfrage.PathValue("id")
		if id == "" {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, "ID fehlt", nil)
			return
		}

		// Die Maske „Titel bearbeiten" lädt hierüber und schickt das Ganze per PUT zurück.
		// Scheitert das Nachladen der Schlagworte, antwortet der Einzel-Read mit einem Fehler
		// statt ohne sie: Die Maske öffnet dann gar nicht, statt mit einem leeren Feld, das beim
		// Speichern als Aussage „keine Schlagworte" zurückkäme.
		buch, fehler := t.ladeTitelVoll(anfrage.Context(), id)
		if errors.Is(fehler, repository.ErrTitelNichtGefunden) {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusNotFound, "Buch nicht gefunden", nil)
			return
		}
		if fehler != nil {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusInternalServerError, "Interner Serverfehler", fmt.Errorf("buch laden: %w", fehler))
			return
		}

		tuer.AntworteJSON(antwort, http.StatusOK, buch)
	}
}

// ladeTitelVoll liest einen Titel, wie die Maske ihn bekommt: mit den Bestandszahlen und den
// Schlagworten. Der Einzel-Read und die Antwort auf das Speichern haben damit dieselbe Form.
func (t *Tueren) ladeTitelVoll(ctx context.Context, id string) (repository.KatalogTitel, error) {
	buecher, err := t.titelAbfragen().LadeTitelNachIDs(ctx, []string{id})
	if err != nil {
		return repository.KatalogTitel{}, err
	}
	if len(buecher) == 0 {
		return repository.KatalogTitel{}, repository.ErrTitelNichtGefunden
	}
	buch := buecher[0]
	buch.Schlagworte, err = t.titelAbfragen().LadeSchlagworte(ctx, id)
	if err != nil {
		return repository.KatalogTitel{}, err
	}
	return buch, nil
}

// gespeichert ist die Antwort auf Anlegen und Ändern: der Titel, wie er jetzt in der Datenbank
// steht. Die Titelliste ersetzt ihre Zeile durch die Antwort; aus den gesendeten Angaben
// allein stünde dort der Bestand 0. Scheitert das Nachlesen, bleibt es bei den gesendeten
// Angaben, denn gespeichert ist der Titel; nach einem Ändern sind das nur die Felder, die
// der Rumpf genannt hat.
func (t *Tueren) gespeichert(ctx context.Context, gesendet repository.KatalogTitel) repository.KatalogTitel {
	buch, err := t.ladeTitelVoll(ctx, gesendet.ID)
	if err != nil {
		slog.Error("Titel nach dem Speichern nicht lesbar", "titel", gesendet.ID, "fehler", err)
		return gesendet
	}
	return buch
}

// TitelVorhanden sagt vor dem Speichern, was die Dublettenkontrolle zu einer ISBN
// sagen wird: den Titel, der sie schon trägt, samt der Meldung der Ablehnung — oder keinen.
// Die Maske „Neues Buch" fragt damit schon bei der Eingabe der ISBN, wie Littera; das Format
// der Nummer prüft erst das Speichern. Die zehn- und die dreizehnstellige Form einer ISBN sind
// dabei dieselbe Nummer (isbn_normalform).
func (t *Tueren) TitelVorhanden() http.HandlerFunc {
	return func(antwort http.ResponseWriter, anfrage *http.Request) {
		isbn := strings.TrimSpace(anfrage.URL.Query().Get("isbn"))
		if isbn == "" {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, "Die ISBN fehlt", nil)
			return
		}

		vorhanden, fehler := t.titelAbfragen().FindeTitelMitISBN(anfrage.Context(), isbn)
		if fehler != nil {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusInternalServerError, "Interner Serverfehler", fmt.Errorf("dublettenkontrolle vorab: %w", fehler))
			return
		}
		if vorhanden != nil {
			tuer.AntworteJSON(antwort, http.StatusOK, map[string]any{
				"vorhanden": vorhandenAlsAntwort(vorhanden.ID, vorhanden.Titel, vorhanden.HatExemplar),
				"meldung":   vorhanden.Meldung(),
			})

			return
		}

		tuer.AntworteJSON(antwort, http.StatusOK, map[string]any{"vorhanden": nil})
	}
}
