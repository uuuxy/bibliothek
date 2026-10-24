package inventur

import (
	"bibliothek/pkg/isbnutil"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

// FremdeCoverListe ist die Tür GET /api/admin/books/external-covers: die Titel, deren Cover
// noch von außen geladen wird.
func (t *Tueren) FremdeCoverListe(w http.ResponseWriter, r *http.Request) {
	eintraege, err := t.abfragen.ladeTitelMitFremdemCover(r.Context(), 300)
	if err != nil {
		slog.Error("Fehler beim Laden externer Cover-Bücher", "fehler", err)
		schreibeFehler(w, http.StatusInternalServerError, "externe cover konnten nicht geladen werden")
		return
	}

	schreibeJSON(w, http.StatusOK, map[string]any{"data": eintraege})
}

// FremdeCoverNeuLaden ist die Tür POST /api/admin/books/retry-covers: Sie versucht die Cover
// von außen noch einmal zu laden.
func (t *Tueren) FremdeCoverNeuLaden(w http.ResponseWriter, r *http.Request) {
	var eingabe struct {
		IDs   []string `json:"ids"`
		Limit int      `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&eingabe); err != nil {
		schreibeFehler(w, http.StatusBadRequest, "Die Anfrage ließ sich nicht lesen")
		return
	}
	if !alleUUIDs(eingabe.IDs) {
		schreibeFehler(w, http.StatusBadRequest, "Eine der Kennungen ist ungültig")
		return
	}

	var (
		eintraege []Titel
		err       error
	)
	if len(eingabe.IDs) > 0 {
		eintraege, err = t.abfragen.LadeTitelNachIDs(r.Context(), eingabe.IDs)
	} else {
		eintraege, err = t.abfragen.ladeTitelMitFremdemCover(r.Context(), eingabe.Limit)
	}
	if err != nil {
		slog.Error("Fehler beim Laden externer Cover für Retry", "fehler", err)
		schreibeFehler(w, http.StatusInternalServerError, "cover-retry konnte nicht gestartet werden")
		return
	}

	zahlen := map[string]int{"retried": 0, coverAktualisiert: 0, coverUebersprungen: 0, coverGescheitert: 0}
	for _, eintrag := range eintraege {
		zahlen["retried"]++
		zahlen[t.ladeCoverErneut(r.Context(), eintrag)]++
	}

	schreibeJSON(w, http.StatusOK, map[string]any{
		"message": "Cover erneut gesucht",
		"data":    zahlen,
	})
}

// Die Zahlen der Antwort, unter denen ein Titel nach dem erneuten Laden zählt.
const (
	coverAktualisiert  = "updated"
	coverUebersprungen = "skipped"
	coverGescheitert   = "failed"
)

// ladeCoverErneut holt das Cover eines Titels über die Katalogdienste, trägt den lokalen Pfad
// des heruntergeladenen Bilds ein und nennt, unter welcher Zahl der Titel in der Antwort zählt.
//
// Nachgeladen wird nur, was noch eine Adresse im Netz trägt: Ein Cover, das lokal liegt, ist
// schon da oder von Hand hochgeladen, und das ersetzt dieser Lauf nicht. Die Suche liefert
// immer den lokalen Pfad (MetadatenClient.aufloeseCover), nie eine Adresse im Netz.
func (t *Tueren) ladeCoverErneut(ctx context.Context, eintrag Titel) string {
	if !isbnutil.HatForm(eintrag.ISBN) || !strings.HasPrefix(eintrag.CoverURL, "http") {
		return coverUebersprungen
	}

	nachschlagen, nachschlageFehler := t.metadaten.SucheNachISBN(ctx, eintrag.ISBN)
	if nachschlageFehler != nil || nachschlagen == nil || nachschlagen.CoverURL == "" {
		return coverGescheitert
	}

	if aenderFehler := t.abfragen.aendereTitelMetadaten(ctx, eintrag.ID, "", "", nachschlagen.CoverURL); aenderFehler != nil {
		return coverGescheitert
	}
	return coverAktualisiert
}
