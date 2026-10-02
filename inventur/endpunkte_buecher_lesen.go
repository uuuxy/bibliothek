package inventur

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// BearbeiteBuecherListe verarbeitet GET-Anfragen für die Bücherübersicht.
// Die Funktion liest Suchparameter (Fach, Klasse, Suchbegriff) aus,
// nutzt ein Wörterbuch für Synonyme (z.B. powi -> politik) und fragt die Datenbank ab.
// Danach werden die Bücher logisch (natürlich) sortiert und als JSON gesendet.
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

	var klassenStufe *int16
	stufenRohwert := strings.TrimSpace(anfrageParameter.Get("gradeLevel"))
	if stufenRohwert == "" {
		stufenRohwert = strings.TrimSpace(anfrageParameter.Get("grade"))
	}
	if stufenRohwert != "" {
		geparsed, fehler := strconv.ParseInt(stufenRohwert, 10, 16)
		if fehler != nil {
			writeError(antwort, http.StatusBadRequest, "ungültiger query-parameter gradeLevel")
			return
		}
		stufenWert := int16(geparsed)
		klassenStufe = &stufenWert
	}

	// bestand=ohne ist die Aufräumsicht der Verwaltung: nur Titel ohne ein einziges
	// nicht ausgesondertes Exemplar. Ohne den Parameter ist es der Katalog.
	sicht := strings.TrimSpace(anfrageParameter.Get("bestand"))
	if sicht != "" && sicht != "ohne" {
		writeError(antwort, http.StatusBadRequest, "ungültiger query-parameter bestand (erlaubt: ohne)")
		return
	}
	buecher, fehler := handler.repo.ListBooks(anfrage.Context(), fach, klassenStufe, suchbegriff, sicht == "ohne")
	if fehler != nil {
		log.Printf("Fehler beim Laden der Bücherliste: %v", fehler)
		writeError(antwort, http.StatusInternalServerError, "Interner Serverfehler beim Laden der Bücher")
		return
	}

	sortiereBuecherNatuerlich(buecher)

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

// extrahiereZahlenUndBasis parst manuell die erste Zahl aus dem Titel
// und gibt diese sowie den bereinigten Basis-String (ohne Ziffern) zurück.
// Dies ersetzt langsame Regex-Operationen für performantes Sortieren.
func extrahiereZahlenUndBasis(titel string) (int, string) {
	var basis strings.Builder
	basis.Grow(len(titel))
	zahl := 0
	erstesGefunden := false
	inZahl := false
	var currentZahl int

	for i := 0; i < len(titel); i++ {
		b := titel[i]
		if b >= '0' && b <= '9' {
			if !erstesGefunden {
				currentZahl = currentZahl*10 + int(b-'0')
				inZahl = true
			}
			// Wir ignorieren alle Ziffern für die Basis (wie Regex ReplaceAllString)
		} else {
			if inZahl {
				zahl = currentZahl
				erstesGefunden = true
				inZahl = false
			}
			basis.WriteByte(b)
		}
	}

	if inZahl && !erstesGefunden {
		zahl = currentZahl
	}

	return zahl, strings.TrimSpace(basis.String())
}

// sortKey hält vorberechnete Werte für performantes Sortieren (Schwartzian Transform)
type sortKey struct {
	bookPtr   *Book
	basis     string
	zahl      int
	titel     string
	sortOrder int
}

// sortiereBuecherNatuerlich führt eine intelligente Sortierung auf dem Array aus.
// Sie sorgt dafür, dass Ziffern logisch sortiert werden (Teil 2 vor Teil 10).
// Nutzt zur Performance die Schwartzian Transform Methode. Groß-/Kleinschreibung wird ignoriert.
func sortiereBuecherNatuerlich(buecher []Book) {
	if len(buecher) <= 1 {
		return
	}

	keys := make([]sortKey, len(buecher))
	for i := range buecher {
		t := strings.ToLower(buecher[i].Title)
		zahl, basis := extrahiereZahlenUndBasis(t)
		keys[i] = sortKey{
			bookPtr:   &buecher[i],
			basis:     basis,
			zahl:      zahl,
			titel:     t,
			sortOrder: buecher[i].SortOrder,
		}
	}

	sort.SliceStable(keys, func(i, j int) bool {
		// Manuelle Sortierfolge (sort_order) des Admins respektieren, falls gesetzt
		if keys[i].sortOrder != keys[j].sortOrder {
			return keys[i].sortOrder < keys[j].sortOrder
		}

		if keys[i].basis != keys[j].basis {
			return keys[i].basis < keys[j].basis
		}

		if keys[i].zahl != keys[j].zahl {
			return keys[i].zahl < keys[j].zahl
		}

		return keys[i].titel < keys[j].titel
	})

	result := make([]Book, len(buecher))
	for i, k := range keys {
		result[i] = *k.bookPtr
	}
	copy(buecher, result)
}

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
// Angaben, denn gespeichert ist der Titel.
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
// der Nummer prüft erst das Speichern. Trägt kein Titel die ISBN, nennt die Antwort unter
// „andereForm" den Titel, der sie in der anderen Länge trägt.
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

	// Unter „vorhanden" steht nur, was das Speichern ablehnt. Ein Titel unter der anderen
	// Länge der ISBN ist ein Vorschlag: Die Maske fragt, und das Speichern legt an.
	andererTitel, andereForm, fehler := handler.repo.TitelUnterAndererForm(anfrage.Context(), isbn)
	if fehler != nil {
		log.Printf("Dublettenkontrolle vorab (andere Form): %v", fehler)
		writeError(antwort, http.StatusInternalServerError, "Interner Serverfehler")
		return
	}
	daten := map[string]any{"vorhanden": nil}
	if andererTitel != nil {
		vorschlag := andererTitel.alsAntwort()
		vorschlag["isbn"] = andereForm
		daten["andereForm"] = vorschlag
		daten["meldung"] = andererTitel.MeldungAndereForm(andereForm)
	}
	writeJSON(antwort, http.StatusOK, map[string]any{"data": daten})
}
