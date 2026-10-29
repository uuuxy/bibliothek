package katalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// Klassensaetze ist die Tür GET /api/class-books (und GET /api/portal/klassensaetze für das
// Kollegium): die Klassensätze je Klasse mit Bestand und Zulauf.
func (t *Tueren) Klassensaetze() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zweig := r.URL.Query().Get("branch")
		reihenfolge := r.URL.Query().Get("sort")
		gruppen, err := t.titelAbfragen().LadeKlassensatzGruppen(r.Context(), zweig, reihenfolge)
		if err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "klassen konnten nicht geladen werden", fmt.Errorf("klassengruppen laden: %w", err))
			return
		}

		if gruppen == nil {
			gruppen = []repository.KlassensatzGruppe{}
		}

		tuer.AntworteJSON(w, http.StatusOK, gruppen)
	}
}

// KlassensaetzeAendern ist die Tür POST /api/admin/class-books: Sie setzt die Titel einer
// Klasse neu.
func (t *Tueren) KlassensaetzeAendern() http.HandlerFunc {
	return t.aendereKlassensaetze
}

// aendereKlassensaetze ist der Rumpf von KlassensaetzeAendern. Er steht auf der obersten Ebene,
// weil eine Closure für die Komplexitätsmessung als eigene Ebene zählt.
func (t *Tueren) aendereKlassensaetze(w http.ResponseWriter, r *http.Request) {
	var eingabe struct {
		AlteKlasse string   `json:"oldClassName,omitempty"`
		Klasse     string   `json:"className,omitempty"` // die ältere Form des Rumpfs mit einer Klasse
		Klassen    []string `json:"classNames,omitempty"`
		TitelIDs   []string `json:"bookIds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&eingabe); err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Anfrage ließ sich nicht lesen", nil)
		return
	}

	// Der Rumpf nennt eine Klasse (className) oder mehrere (classNames).
	var zielKlassen []string
	if len(eingabe.Klassen) > 0 {
		for _, name := range eingabe.Klassen {
			bereinigt := strings.TrimSpace(name)
			if bereinigt != "" {
				zielKlassen = append(zielKlassen, normalisiereKlassenname(bereinigt))
			}
		}
	} else if strings.TrimSpace(eingabe.Klasse) != "" {
		zielKlassen = append(zielKlassen, normalisiereKlassenname(strings.TrimSpace(eingabe.Klasse)))
	}

	if len(zielKlassen) == 0 {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Bitte mindestens eine Klasse angeben", nil)
		return
	}

	// Klassennamen-Länge begrenzen (max. 20 Zeichen Schutz gegen Missbrauch)
	for _, cn := range zielKlassen {
		if len(cn) > 20 {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Der Name einer Klasse hat höchstens 20 Zeichen", nil)
			return
		}
	}

	// Umbenennen lässt sich sicher nur eine einzelne Klasse.
	alterName := strings.TrimSpace(eingabe.AlteKlasse)

	if !alleUUIDs(eingabe.TitelIDs) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Eine der Kennungen ist ungültig", nil)
		return
	}
	err := t.titelAbfragen().AendereKlassensaetze(r.Context(), alterName, zielKlassen, eingabe.TitelIDs)
	if err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "klassenbücher konnten nicht aktualisiert werden", fmt.Errorf("klassenbücher aktualisieren: %w", err))
		return
	}

	tuer.AntworteJSON(w, http.StatusOK, map[string]any{"message": "Gespeichert"})
}

// KlassensatzGruppeLoeschen ist die Tür DELETE /api/admin/class-books: Sie löscht die
// Zuordnungen einer Klasse.
func (t *Tueren) KlassensatzGruppeLoeschen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		abfrageNamen := r.URL.Query()["className"]
		var klassen []string

		for _, name := range abfrageNamen {
			bereinigt := strings.TrimSpace(name)
			if bereinigt != "" {
				klassen = append(klassen, bereinigt)
			}
		}

		if len(klassen) == 0 {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Der Name der Klasse fehlt", nil)
			return
		}

		err := t.titelAbfragen().LoescheKlassensatzGruppe(r.Context(), klassen)
		if err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "klasse konnte nicht gelöscht werden", fmt.Errorf("klassengruppen löschen: %w", err))
			return
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]any{"message": "Klasse gelöscht"})
	}
}

func normalisiereKlassenname(name string) string {
	name = strings.ReplaceAll(name, " ", "")
	if len(name) > 0 && name[0] >= '1' && name[0] <= '9' {
		if len(name) == 1 || (name[1] < '0' || name[1] > '9') {
			return "0" + name
		}
	}
	return name
}

// KlassensaetzeZuordnen ist die Tür POST /api/admin/class-books/add: Sie ordnet Titel weiteren
// Klassen zu.
func (t *Tueren) KlassensaetzeZuordnen() http.HandlerFunc {
	return t.ordneKlassensaetzeZu
}

// ordneKlassensaetzeZu ist der Rumpf von KlassensaetzeZuordnen. Er steht auf der obersten Ebene,
// weil eine Closure für die Komplexitätsmessung als eigene Ebene zählt.
func (t *Tueren) ordneKlassensaetzeZu(w http.ResponseWriter, r *http.Request) {
	var eingabe struct {
		Klassen  []string `json:"classNames"`
		TitelIDs []string `json:"bookIds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&eingabe); err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Anfrage ließ sich nicht lesen", nil)
		return
	}

	var zielKlassen []string
	for _, name := range eingabe.Klassen {
		bereinigt := strings.TrimSpace(name)
		if bereinigt != "" {
			zielKlassen = append(zielKlassen, normalisiereKlassenname(bereinigt))
		}
	}

	if len(zielKlassen) == 0 {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Bitte mindestens eine Klasse angeben", nil)
		return
	}

	for _, cn := range zielKlassen {
		if len(cn) > 20 {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Der Name einer Klasse hat höchstens 20 Zeichen", nil)
			return
		}
	}

	if !alleUUIDs(eingabe.TitelIDs) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Eine der Kennungen ist ungültig", nil)
		return
	}
	err := t.titelAbfragen().OrdneTitelKlassenZu(r.Context(), zielKlassen, eingabe.TitelIDs)
	if err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "bücher konnten nicht hinzugefügt werden", fmt.Errorf("bücher der klasse hinzufügen: %w", err))
		return
	}

	tuer.AntworteJSON(w, http.StatusOK, map[string]any{"message": "Hinzugefügt"})
}
