package inventurlauf

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// abschlussAnfrage nennt den Lauf, der abgeschlossen wird.
type abschlussAnfrage struct {
	LaufID string `json:"session_id" validate:"omitempty,uuid_oder_leer"`
}

// abschlussAntwort nennt, was der Abschluss als Verlust gebucht hat.
type abschlussAntwort struct {
	VerlorenGemeldet int `json:"verloren_gemeldet"`

	// Fehlbestand nennt die Exemplare, nicht nur ihre Zahl: Mit „47 Verluste" kann niemand im
	// Regal nachsehen.
	Fehlbestand []repository.InventurVerlust `json:"fehlbestand"`
}

// InventurAbschliessen schließt einen Lauf ab: Jedes Exemplar, das der Bereich erwartet, das
// dieser Lauf nicht erfasst hat und das nicht verliehen ist, wird als Verlust gebucht. Andere
// offene Läufe bleiben unberührt.
//
// @Summary      Inventur abschließen
// @Description  Bucht, was der Bereich erwartet und der Lauf nicht erfasst hat, als Verlust und schließt den Lauf.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        body  body      abschlussAnfrage   true  "Lauf"
// @Success      200   {object}  abschlussAntwort
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /inventur/finish [post]
func (t *Tueren) InventurAbschliessen() http.HandlerFunc {
	return t.schliesseInventurAb
}

func (t *Tueren) schliesseInventurAb(w http.ResponseWriter, r *http.Request) {
	var req abschlussAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return
	}
	if req.LaufID == "" {
		apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New("session_id fehlt"))
		return
	}

	// Wer abschließt, steht an jedem Exemplar, das der Abschluss als Verlust bucht.
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		apierrors.AntworteFehler(w, http.StatusUnauthorized, errors.New("nicht angemeldet"))
		return
	}

	ctx := r.Context()

	tx, err := t.DB.Pool.Begin(ctx)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}
	defer db.Verwerfe(ctx, tx)

	abfragen := repository.NewInventurAbfragen(tx)

	// Den Lauf in dieser Transaktion exklusiv sperren (FOR UPDATE): Das verhindert einen
	// doppelten Abschluss und wartet auf laufende Scans, die die Zeile mit FOR SHARE halten;
	// die Verluste werden erst bestimmt, wenn kein Scan mehr offen ist. Die Sperre liefert
	// zugleich den gespeicherten Bereich.
	lauf, err := abfragen.SperreLaufFuerAbschluss(ctx, req.LaufID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.AntworteFehler(w, http.StatusNotFound, errors.New("keine laufende Inventur zu dieser Session"))
			return
		}
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	verloren, err := abfragen.SchliesseLaufAb(ctx, lauf.ID, lauf.Bereich(), claims.UserID)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	// In derselben Transaktion gelesen, vor dem Commit: Danach ist der Fehlbestand
	// zwar dauerhaft gespeichert, aber ein Fehler beim Lesen dürfte den Abschluss
	// nicht mehr zurückrollen — die Aussonderung wäre dann schon geschrieben.
	fehlbestand, err := abfragen.LadeVerluste(ctx, lauf.ID)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	tuer.AntworteJSON(w, http.StatusOK, abschlussAntwort{
		VerlorenGemeldet: verloren,
		Fehlbestand:      fehlbestand,
	})
}

// Fehlbestand liefert den Fehlbestand einer bereits abgeschlossenen
// Inventur.
//
// Der Abschluss nennt die Liste zwar sofort, aber genau in dem Moment steht man mit dem
// Scanner im Regal und will sie nicht am Bildschirm lesen. Ohne diesen Weg wäre sie nach
// dem Schließen der Ansicht verloren, und nachrechnen kann man sie nicht, weil die
// Exemplare durch die Aussonderung aus dem Bereich fallen.
//
// @Summary      Fehlbestand einer abgeschlossenen Inventur
// @Tags         inventur
// @Produce      json
// @Param        session_id  query  string  true  "Lauf"
// @Success      200  {array}  repository.InventurVerlust
// @Router       /inventur/fehlbestand [get]
func (t *Tueren) Fehlbestand() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		laufID, err := tuer.LeseUUIDParameter(r, "session_id")
		if err != nil {
			return apierrors.BadRequest(err.Error(), nil)
		}
		if laufID == "" {
			return apierrors.BadRequest("session_id fehlt", nil)
		}

		abfragen := repository.NewInventurAbfragen(t.DB.Pool)
		liste, err := abfragen.LadeVerluste(r.Context(), laufID)
		if err != nil {
			return apierrors.Internal("Fehlbestand konnte nicht geladen werden", err)
		}

		tuer.AntworteJSON(w, http.StatusOK, liste)
		return nil
	})
}
