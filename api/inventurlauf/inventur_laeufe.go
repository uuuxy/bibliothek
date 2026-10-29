package inventurlauf

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/pkg/kennung"
	"bibliothek/repository"
)

// offenerLauf ist ein offener Lauf, wie die Oberfläche ihn zeigt, mit seinem Stand.
type offenerLauf struct {
	LaufID      string `json:"session_id"`
	Bereich     string `json:"scope"`
	Bezeichnung string `json:"label"`
	GestartetAm string `json:"gestartet_am"`
	repository.LaufStand
}

// abgeschlossenerLauf ist eine fertige Inventur, wie die Oberfläche sie zur Auswahl zeigt:
// wann, welcher Bereich, wie viele Verluste und ob sie verworfen wurde; dann ist der Bereich
// nicht geprüft, auch ohne Verlust.
type abgeschlossenerLauf struct {
	LaufID          string `json:"session_id"`
	Bezeichnung     string `json:"label"`
	AbgeschlossenAm string `json:"abgeschlossen_am"`
	Erfasst         int    `json:"erfasst"`
	Verluste        int    `json:"verluste"`
	Verworfen       bool   `json:"verworfen"`
}

// abgeschlosseneSeite ist eine Seite der früheren Inventuren: die Läufe, wie viele ältere danach
// kommen und das Zeichen, mit dem die Oberfläche die nächste Seite anfordert (leer, wenn keine
// mehr kommt).
type abgeschlosseneSeite struct {
	Laeufe        []abgeschlossenerLauf `json:"laeufe"`
	Weitere       int                   `json:"weitere"`
	NaechsteSeite string                `json:"naechste_seite"`
}

// laeufeJeSeite ist die Zahl der früheren Inventuren, die „Mehr laden" jeweils dazuholt.
const laeufeJeSeite = 10

// AbgeschlosseneInventuren liefert die abgeschlossenen Inventuren seitenweise, die neuesten
// zuerst. Der Parameter seite trägt naechste_seite der vorigen Antwort; ohne ihn kommt die erste.
//
// Über sie kommt die Oberfläche an die Kennung für GET /api/inventur/fehlbestand: Die Liste
// der laufenden Inventuren nennt eine fertige nicht mehr, und ohne Kennung bliebe der
// Fehlbestand nach einem Neuladen oder an einem zweiten Arbeitsplatz unerreichbar, obwohl er
// in inventur_verluste gespeichert ist.
func (t *Tueren) AbgeschlosseneInventuren() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vor, ok := leseSeitenzeichen(r.URL.Query().Get("seite"))
		if !ok {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Angabe der Seite in der Adresse ist ungültig", nil)
			return
		}
		abfragen := repository.NewInventurAbfragen(t.DB.Pool)

		seite, err := abfragen.LadeAbgeschlosseneLaeufe(r.Context(), vor, laeufeJeSeite)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		antwort := abgeschlosseneSeite{
			Laeufe:        make([]abgeschlossenerLauf, 0, len(seite.Laeufe)),
			Weitere:       seite.Weitere,
			NaechsteSeite: seitenzeichen(seite.Naechste),
		}
		for _, lauf := range seite.Laeufe {
			abgeschlossen := ""
			if lauf.AbgeschlossenAm != nil {
				abgeschlossen = *lauf.AbgeschlossenAm
			}
			antwort.Laeufe = append(antwort.Laeufe, abgeschlossenerLauf{
				LaufID:          lauf.ID,
				Bezeichnung:     lauf.Bezeichnung,
				AbgeschlossenAm: abgeschlossen,
				Erfasst:         lauf.Erfasst,
				Verluste:        lauf.Verluste,
				Verworfen:       lauf.Verworfen,
			})
		}
		tuer.AntworteJSON(w, http.StatusOK, antwort)
	}
}

// seitenzeichen schreibt den letzten Lauf einer Seite als Zeichen für die nächste: Abschluss und
// Kennung. Die Oberfläche reicht es unverändert zurück, wie ein page_token bei Google.
func seitenzeichen(marke *repository.LaufMarke) string {
	if marke == nil {
		return ""
	}
	return marke.AbgeschlossenAm.UTC().Format(time.RFC3339Nano) + "_" + marke.ID
}

// leseSeitenzeichen liest ein Zeichen aus seitenzeichen; leer heißt erste Seite. false heißt,
// das Zeichen stammt nicht aus seitenzeichen.
func leseSeitenzeichen(zeichen string) (*repository.LaufMarke, bool) {
	if zeichen == "" {
		return nil, true
	}
	zeit, id, gefunden := strings.Cut(zeichen, "_")
	am, err := time.Parse(time.RFC3339Nano, zeit)
	if !gefunden || err != nil || !kennung.IstUUID(id) {
		return nil, false
	}
	return &repository.LaufMarke{AbgeschlossenAm: am, ID: id}, true
}

// AbgeschlosseneInventurLoeschen löscht eine abgeschlossene Inventur mit ihrem
// Fehlbestandsbericht. Was sie an den Büchern gebucht hat, bleibt
// (repository.LoescheAbgeschlossenenLauf); eine laufende Inventur wird verworfen, nicht gelöscht.
func (t *Tueren) AbgeschlosseneInventurLoeschen() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := auth.ClaimsAusKontext(r.Context())
		if !ok {
			return apierrors.Unauthorized(auth.MeldungNichtAngemeldet, errors.New("missing session information"))
		}

		ctx := r.Context()
		tx, err := t.DB.Pool.Begin(ctx)
		if err != nil {
			return apierrors.Internal("Transaktion konnte nicht gestartet werden", err)
		}
		defer db.Verwerfe(ctx, tx)

		err = repository.NewInventurAbfragen(tx).LoescheAbgeschlossenenLauf(ctx, r.PathValue("id"), claims.KontoID)
		if errors.Is(err, repository.ErrInventurlaufNichtGefunden) {
			return apierrors.NotFound("Keine abgeschlossene Inventur mit dieser Kennung", err)
		}
		if err != nil {
			return apierrors.Internal("Die Inventur konnte nicht gelöscht werden", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return apierrors.Internal("Transaktion konnte nicht abgeschlossen werden", err)
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
}

// LaufendeInventuren liefert alle offenen Läufe. Die Oberfläche zeigt sie, damit eine
// Lehrkraft eine laufende Inventur fortsetzt, statt vergeblich eine zweite im selben Bereich
// zu starten.
func (t *Tueren) LaufendeInventuren() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		abfragen := repository.NewInventurAbfragen(t.DB.Pool)

		laeufe, err := abfragen.LadeOffeneLaeufe(ctx)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		liste := make([]offenerLauf, 0, len(laeufe))
		for i := range laeufe {
			stand, err := abfragen.ZaehleStand(ctx, laeufe[i].ID, laeufe[i].Bereich())
			if err != nil {
				apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
				return
			}
			liste = append(liste, offenerLauf{
				LaufID:      laeufe[i].ID,
				Bereich:     laeufe[i].BereichArt,
				Bezeichnung: laeufe[i].Bezeichnung,
				GestartetAm: laeufe[i].GestartetAm,
				LaufStand:   stand,
			})
		}
		tuer.AntworteJSON(w, http.StatusOK, liste)
	}
}

// verwerfenAnfrage nennt den Lauf, der verworfen wird.
type verwerfenAnfrage struct {
	LaufID string `json:"session_id" validate:"omitempty,uuid_oder_leer"`
}

// InventurVerwerfen verwirft einen Lauf, ohne Verluste zu buchen, etwa eine abgebrochene oder
// hängen gebliebene Inventur. Danach ist der Bereich für einen neuen Lauf frei.
func (t *Tueren) InventurVerwerfen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req verwerfenAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}
		if req.LaufID == "" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung des Inventurlaufs fehlt", nil)
			return
		}

		ctx := r.Context()
		abfragen := repository.NewInventurAbfragen(t.DB.Pool)
		if err := abfragen.VerwerfeLauf(ctx, req.LaufID); err != nil {
			if errors.Is(err, repository.ErrInventurlaufNichtGefunden) {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Inventurlauf nicht gefunden", err)
				return
			}
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		tuer.AntworteErfolg(w)
	}
}
