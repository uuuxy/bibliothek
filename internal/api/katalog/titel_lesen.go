package katalog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/apierrors"
	"bibliothek/internal/pkg/ordnung"
	"bibliothek/internal/repository"
)

// TitelListe verarbeitet GET-Anfragen für die Bücherübersicht.
// Die Funktion liest Suchparameter (Fach, Suchbegriff) aus und fragt die Datenbank ab; den
// Suchbegriff löst die Titelsuche des Programms auf (Wörter, Kürzel, Jahrgang).
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

// sortiereBuecherNachTitel ordnet die Liste nach dem Titel, wie ein deutsches Register
// (ordnung.Sortiere). Dieselbe Reihenfolge gibt der Browser mit Intl.Collator('de', {numeric:
// true}), nach dem die Listen der Oberfläche ordnen; beide Seiten prüft
// titelReihenfolge.faelle.json. Gleiche Titel behalten die Reihenfolge der Abfrage.
func sortiereBuecherNachTitel(buecher []repository.KatalogTitel) {
	ordnung.Sortiere(buecher, func(b repository.KatalogTitel) string { return b.Titel })
}

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
// Die Maske „Titel anlegen" fragt damit schon bei der Eingabe der ISBN, wie Littera; das Format
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
