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

// abschlussAnfrage benennt die abzuschließende Session.
type abschlussAnfrage struct {
	LaufID string `json:"session_id" validate:"omitempty,uuid_oder_leer"`
}

// abschlussAntwort contains the outcome statistics of the completed inventory.
type abschlussAntwort struct {
	VerlorenGemeldet int `json:"verloren_gemeldet"`

	// Fehlbestand nennt die betroffenen Exemplare, nicht nur ihre Anzahl. Vorher stand
	// hier allein die Zahl — mit „47 Verluste" kann niemand ins Regal gehen und nachsehen.
	Fehlbestand []repository.InventurVerlust `json:"fehlbestand"`
}

// InventurAbschliessen schließt eine Inventur-Session ab: Alle im Scope erwarteten,
// aber in DIESER Session nicht erfassten (und nicht verliehenen) Exemplare werden als
// Verlust markiert. Der Fortschritt paralleler Sessions bleibt unberührt.
//
// @Summary      Finalize an inventory session
// @Description  Marks scope items not scanned in this session as lost and closes it.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        body  body      abschlussAnfrage   true  "Session to finalize"
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
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("session_id fehlt"))
		return
	}

	// Wer abschließt, steht an jedem Exemplar, das der Abschluss als Verlust bucht.
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("nicht angemeldet"))
		return
	}

	ctx := r.Context()

	tx, err := t.DB.Pool.Begin(ctx)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}
	defer db.SafeRollback(ctx, tx)

	abfragen := repository.NewInventurAbfragen(tx)

	// Session in derselben Transaktion EXKLUSIV sperren (FOR UPDATE): verhindert den
	// doppelten Abschluss UND koordiniert gegen gleichzeitige Scans (die die Zeile
	// FOR SHARE halten) — der Verlust-Snapshot wird erst gezogen, wenn kein Scan mehr
	// offen ist. Liefert zugleich den gespeicherten Scope für das Verlust-Update.
	lauf, err := abfragen.SperreLaufFuerAbschluss(ctx, req.LaufID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("keine laufende Inventur zu dieser Session"))
			return
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	verloren, err := abfragen.SchliesseLaufAb(ctx, lauf.ID, lauf.Bereich(), claims.UserID)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	// In DERSELBEN Transaktion gelesen, vor dem Commit: Danach ist der Fehlbestand
	// zwar dauerhaft gespeichert, aber ein Fehler beim Lesen dürfte den Abschluss
	// nicht mehr zurückrollen — die Aussonderung wäre dann schon geschrieben.
	fehlbestand, err := abfragen.LadeVerluste(ctx, lauf.ID)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
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
// dem Schliessen der Ansicht verloren — und nachrechnen kann man sie nicht, weil die
// Exemplare durch die Aussonderung aus dem Scope fallen.
//
// @Summary      Fehlbestand einer abgeschlossenen Inventur
// @Tags         inventur
// @Produce      json
// @Param        session_id  query  string  true  "Session"
// @Success      200  {array}  repository.InventurVerlust
// @Router       /inventur/fehlbestand [get]
func (t *Tueren) Fehlbestand() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
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
