package katalog

import (
	"errors"
	"fmt"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pkg/kennung"
	"bibliothek/repository"
)

// Die Pflege der Schlagworte. Die Regeln stehen in repository/schlagworte_pflege.go; jede Tür
// hier ist eine Aktion der Pflegeseite, kein allgemeines PATCH, so trägt jede ihre Regel
// sichtbar mit.

// schlagwortWortAnfrage ist die neue Schreibweise für PUT /api/schlagworte/{id}/wort.
// AlteAlsVerweis: ob die alte Schreibweise als Verweis stehen bleibt
// (repository.BenenneSchlagwortUm). Pflicht wie ist_filter: Ohne das Feld hätte jede Tür ihre
// eigene Vorgabe (Zusammenführen ließe den Verweis immer stehen, Umbenennen nie).
type schlagwortWortAnfrage struct {
	Wort           string `json:"wort"`
	AlteAlsVerweis *bool  `json:"alte_als_verweis"`
}

// schlagwortFilterAnfrage setzt oder nimmt die Filter-Markierung. Zeiger: Ein Körper ohne
// das Feld ist ein Fehler, kein „aus".
type schlagwortFilterAnfrage struct {
	IstFilter *bool `json:"ist_filter"`
}

// schlagwortZielAnfrage nennt das Ziel eines Zusammenführens und, Pflicht wie beim
// Umbenennen, ob das alte Wort als Verweis stehen bleibt.
type schlagwortZielAnfrage struct {
	ZielID         string `json:"ziel_id" validate:"required,uuid_oder_leer"`
	AlteAlsVerweis *bool  `json:"alte_als_verweis"`
}

// schlagwortVerweisAnfrage legt einen Verweis an: die Schreibweise und ihr Ziel.
type schlagwortVerweisAnfrage struct {
	Wort   string `json:"wort"`
	ZielID string `json:"ziel_id" validate:"required,uuid_oder_leer"`
}

// schlagworteLoeschenAnfrage nennt die Wörter, die fallen sollen. Jede Kennung ist Pflicht und
// eine UUID; eine leere käme sonst als 500 aus der Datenbank zurück.
type schlagworteLoeschenAnfrage struct {
	IDs []string `json:"ids" validate:"required,dive,required,uuid_oder_leer"`
}

// schlagwortPflegeAntwort ist die Antwort der Pflege-Türen: was sich getan hat, als Zahlen.
// Woerter trägt nur das Löschen.
type schlagwortPflegeAntwort struct {
	Wort     string `json:"wort,omitempty"`
	Woerter  int    `json:"woerter,omitempty"`
	Titel    int    `json:"titel"`
	Verweise int    `json:"verweise"`
}

// SchlagworteZurPflege liefert die Schlagworte, die zu ?suche= passen (leer = alle), mit
// Titelzahl, Verweisziel, den Verweisen darauf und der Filter-Markierung, höchstens 200
// Zeilen, dazu die Zahlen der ganzen Tabelle (repository.LadeSchlagworteZurPflege).
//
// @Summary      Schlagworte für die Pflege
// @Tags         books
// @Produce      json
// @Param        suche  query  string  false  "Teil eines Worts oder eines Verweises darauf"
// @Success      200  {object}  repository.SchlagwortPflegeListe
// @Failure      500  {object}  map[string]string
// @Router       /schlagworte/pflege [get]
func (t *Tueren) SchlagworteZurPflege() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		liste, err := repository.LadeSchlagworteZurPflege(r.Context(), t.DB.Pool, r.URL.Query().Get("suche"))
		if err != nil {
			return apierrors.Internal("Schlagworte konnten nicht geladen werden", err)
		}
		tuer.AntworteJSON(w, http.StatusOK, liste)
		return nil
	})
}

// SchlagwortUmbenennen gibt einem Schlagwort eine neue Schreibweise.
//
// @Summary      Schlagwort umbenennen
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        id    path  string                 true  "Kennung des Schlagworts"
// @Param        body  body  schlagwortWortAnfrage  true  "Die neue Schreibweise"
// @Success      200  {object}  schlagwortPflegeAntwort
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Router       /schlagworte/{id}/wort [put]
func (t *Tueren) SchlagwortUmbenennen() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		id, err := schlagwortIDAusPfad(r)
		if err != nil {
			return err
		}
		var req schlagwortWortAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}
		if req.AlteAlsVerweis == nil {
			return alteAlsVerweisFehlt()
		}
		wort, verweis, err := repository.BenenneSchlagwortUm(r.Context(), t.DB.Pool, id, req.Wort, *req.AlteAlsVerweis)
		if err != nil {
			return schlagwortPflegeFehler(err)
		}
		antwort := schlagwortPflegeAntwort{Wort: wort}
		if verweis {
			antwort.Verweise = 1
		}
		tuer.AntworteJSON(w, http.StatusOK, antwort)
		return nil
	})
}

// SchlagwortFilterSetzen setzt oder nimmt die Markierung als Filter im Portal.
//
// @Summary      Schlagwort als Filter im Portal markieren
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        id    path  string                   true  "Kennung des Schlagworts"
// @Param        body  body  schlagwortFilterAnfrage  true  "Filter ja oder nein"
// @Success      204
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Router       /schlagworte/{id}/filter [put]
func (t *Tueren) SchlagwortFilterSetzen() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		id, err := schlagwortIDAusPfad(r)
		if err != nil {
			return err
		}
		var req schlagwortFilterAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}
		if req.IstFilter == nil {
			return apierrors.BadRequest("Die Angabe ist_filter fehlt", errors.New("feld ist_filter fehlt"))
		}
		if err := repository.SetzeSchlagwortFilter(r.Context(), t.DB.Pool, id, *req.IstFilter); err != nil {
			return schlagwortPflegeFehler(err)
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
}

// SchlagworteZusammenfuehren hängt die Titel des Wortes an das Ziel und macht
// das Wort zum Verweis darauf oder löscht es (alte_als_verweis).
//
// @Summary      Schlagwort mit einem anderen zusammenführen
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        id    path  string                 true  "Kennung des Schlagworts"
// @Param        body  body  schlagwortZielAnfrage  true  "Das Ziel"
// @Success      200  {object}  schlagwortPflegeAntwort
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Router       /schlagworte/{id}/zusammenfuehren [post]
func (t *Tueren) SchlagworteZusammenfuehren() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		id, err := schlagwortIDAusPfad(r)
		if err != nil {
			return err
		}
		var req schlagwortZielAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}
		if req.AlteAlsVerweis == nil {
			return alteAlsVerweisFehlt()
		}
		ergebnis, err := repository.FuehreSchlagworteZusammen(r.Context(), t.DB.Pool, id, req.ZielID, *req.AlteAlsVerweis)
		if err != nil {
			return schlagwortPflegeFehler(err)
		}
		t.ProtokolliereVerwaltung(r.Context(), tuer.AktionSchlagwortZusammengefuehrt, map[string]any{
			"von": ergebnis.Von, "in": ergebnis.In, "titel": ergebnis.Titel, "alte_als_verweis": *req.AlteAlsVerweis,
		})
		tuer.AntworteJSON(w, http.StatusOK, schlagwortPflegeAntwort{Titel: ergebnis.Titel})
		return nil
	})
}

// SchlagwortVerweisSetzen leitet eine Schreibweise auf ein Schlagwort.
//
// @Summary      Verweis auf ein Schlagwort anlegen
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        body  body  schlagwortVerweisAnfrage  true  "Schreibweise und Ziel"
// @Success      204
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Router       /schlagworte/verweise [post]
func (t *Tueren) SchlagwortVerweisSetzen() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		var req schlagwortVerweisAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}
		ergebnis, err := repository.SetzeSchlagwortVerweis(r.Context(), t.DB.Pool, req.Wort, req.ZielID)
		if err != nil {
			return schlagwortPflegeFehler(err)
		}
		// Trug die Schreibweise schon Titel, hängen sie jetzt am Ziel; „titel" nennt, wie viele.
		t.ProtokolliereVerwaltung(r.Context(), tuer.AktionSchlagwortVerweis, map[string]any{
			"wort": ergebnis.Von, "in": ergebnis.In, "titel": ergebnis.Titel,
		})
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
}

// SchlagworteLoeschen löscht die gewählten Schlagworte, alle oder keins; die Titel
// verlieren sie, Verweise darauf fallen mit. Eine Tür für ein Wort und für viele
// (repository.LoescheSchlagworte). Die Antwort nennt, wie viele Wörter, Titel und Verweise
// es waren.
//
// @Summary      Schlagworte löschen
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        body  body  schlagworteLoeschenAnfrage  true  "Kennungen der Schlagworte"
// @Success      200  {object}  schlagwortPflegeAntwort
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /schlagworte/loeschen [post]
func (t *Tueren) SchlagworteLoeschen() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		var req schlagworteLoeschenAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}
		geloescht, err := repository.LoescheSchlagworte(r.Context(), t.DB.Pool, req.IDs)
		if errors.Is(err, repository.ErrSchlagwortNichtGefunden) {
			// Alle oder keins: Der Satz sagt, dass auch die übrigen noch da sind.
			return apierrors.NotFound("Ein gewähltes Schlagwort gibt es nicht mehr, gelöscht wurde nichts", err)
		}
		if err != nil {
			return schlagwortPflegeFehler(err)
		}
		t.ProtokolliereVerwaltung(r.Context(), tuer.AktionSchlagwortGeloescht, map[string]any{
			"woerter": geloeschteWoerter(geloescht.Namen), "anzahl": geloescht.Woerter,
			"titel": geloescht.Titel, "verweise": geloescht.Verweise,
		})
		tuer.AntworteJSON(w, http.StatusOK, schlagwortPflegeAntwort{
			Woerter: geloescht.Woerter, Titel: geloescht.Titel, Verweise: geloescht.Verweise,
		})
		return nil
	})
}

// geloeschteWoerterMax begrenzt die Liste im Eintrag; „anzahl" nennt daneben alle.
const geloeschteWoerterMax = 50

// geloeschteWoerter kürzt die Liste der gelöschten Wörter für den Eintrag.
func geloeschteWoerter(namen []string) []string {
	if len(namen) > geloeschteWoerterMax {
		return namen[:geloeschteWoerterMax]
	}
	return namen
}

// alteAlsVerweisFehlt ist die Antwort auf einen Körper ohne alte_als_verweis.
func alteAlsVerweisFehlt() error {
	return apierrors.BadRequest("Die Angabe alte_als_verweis fehlt", errors.New("feld alte_als_verweis fehlt"))
}

// schlagwortIDAusPfad prüft die Kennung, bevor irgendetwas die Datenbank fragt.
func schlagwortIDAusPfad(r *http.Request) (string, error) {
	id := r.PathValue("id")
	if !kennung.IstUUID(id) {
		return "", apierrors.BadRequest("Die Kennung des Schlagworts ist ungültig", errors.New("keine UUID"))
	}
	return id, nil
}

// schlagwortPflegeFehler übersetzt die fachlichen Fehler der Pflege: verletzte Grenze 400,
// unbekanntes Wort 404, vorhandene Schreibweise und verletzte Regel 409 mit ihrem Grund. Der
// Grund einer vorhandenen Schreibweise ist das Wort, das es schon gibt.
func schlagwortPflegeFehler(err error) error {
	switch {
	case errors.Is(err, repository.ErrSchlagwortUngueltig):
		return apierrors.BadRequest(tuer.MitGrund(meldungSchlagwortNichtGeaendert, err), err)
	case errors.Is(err, repository.ErrSchlagwortNichtGefunden):
		return apierrors.NotFound("Schlagwort nicht gefunden", err)
	case errors.Is(err, repository.ErrSchlagwortGibtEs):
		return apierrors.Conflict(fmt.Sprintf("Das Schlagwort %s gibt es schon. Zum Zusammenlegen „Zusammenführen“ wählen.",
			tuer.GrundAus(err)), err)
	case errors.Is(err, repository.ErrSchlagwortRegel):
		return apierrors.Conflict(tuer.MitGrund(meldungSchlagwortRegel, err), err)
	default:
		return apierrors.Internal("Schlagwort konnte nicht geändert werden", err)
	}
}
