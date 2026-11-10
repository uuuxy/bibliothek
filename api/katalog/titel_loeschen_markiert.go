package katalog

import (
	"errors"
	"fmt"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// maxTitelJeLoeschen begrenzt, wie viele Titel ein Auftrag löscht. Die Vorschau trägt die
// Kennungen in der Adresse, und für beide Türen gilt dieselbe Grenze; an einem Aufräumtag gingen
// in Littera höchstens 20 Titel auf einmal.
const maxTitelJeLoeschen = 200

// MarkierteTitelLoeschen ist die Tür DELETE /api/books, der einzige Weg, Titel mit ihren Exemplaren
// zu löschen: die markierten der Bestandstabelle und der eine aus Buchakte und Buchmaske. Der Körper
// nennt die Titel, den Grund für die Exemplare im Haus und die verliehenen Exemplare, die der
// Mensch zum Mitlöschen bestätigt hat ({"ids": [...], "grund": "AUSSORTIERT", "verliehene": [...]};
// TitelAbfragen.LoescheTitel). Die Antwort nennt die gelöschten Titel und die, die bleiben.
func (t *Tueren) MarkierteTitelLoeschen() http.HandlerFunc {
	return func(antwort http.ResponseWriter, anfrage *http.Request) {
		var eingabe struct {
			IDs        []string `json:"ids"`
			Grund      string   `json:"grund"`
			Verliehene []string `json:"verliehene"`
		}
		if fehler := tuer.DekodiereRumpf(antwort, anfrage, &eingabe); fehler != nil {
			tuer.AntworteUnlesbar(antwort, fehler)
			return
		}
		if meldung := meldungZuTitelKennungen(eingabe.IDs, alleUUIDs(eingabe.IDs)); meldung != "" {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, meldung, nil)
			return
		}
		if !repository.IstAussonderungsgrund(eingabe.Grund) {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, meldungGrundFehlt, nil)
			return
		}
		if !alleUUIDs(eingabe.Verliehene) {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, "Eine der Kennungen ist ungültig", nil)
			return
		}
		claims, ok := auth.ClaimsAusKontext(anfrage.Context())
		if !ok {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusUnauthorized, auth.MeldungNichtAngemeldet, nil)
			return
		}
		ergebnis, fehler := t.titelAbfragen().LoescheTitel(anfrage.Context(), repository.TitelLoeschauftrag{
			TitelIDs: eingabe.IDs, Grund: eingabe.Grund, Verliehene: eingabe.Verliehene, BearbeiterID: claims.KontoID,
		})
		switch {
		case errors.Is(fehler, repository.ErrTitelNichtGefunden):
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusNotFound, "Keiner der ausgewählten Titel wurde gefunden", nil)
		case errors.Is(fehler, repository.ErrTitelInzwischenVerliehen):
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusConflict,
				"Inzwischen wurde ein Exemplar ausgeliehen. Bitte noch einmal löschen.", fehler)
		case fehler != nil:
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusInternalServerError, "Interner Serverfehler beim Löschen der Bücher", fmt.Errorf("markierte titel löschen: %w", fehler))
		default:
			tuer.AntworteJSON(antwort, http.StatusOK, ergebnis)
		}
	}
}

// meldungZuTitelKennungen liefert die Meldung zu den Kennungen eines Lösch- oder
// Vorschauauftrags, wenn sie nicht taugen, sonst "". Ob alle Kennungen UUIDs sind, prüft die Tür
// selbst mit alleUUIDs und gibt es mit.
func meldungZuTitelKennungen(ids []string, alleSindUUIDs bool) string {
	switch {
	case len(ids) == 0:
		return "Keine Titel ausgewählt"
	case len(ids) > maxTitelJeLoeschen:
		return fmt.Sprintf("Höchstens %d Titel auf einmal", maxTitelJeLoeschen)
	case !alleSindUUIDs:
		return "Eine der Kennungen ist ungültig"
	}
	return ""
}
