package inventur

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

// Klassensaetze ist die Tür GET /api/class-books (und GET /api/portal/klassensaetze für das
// Kollegium): die Klassensätze je Klasse mit Bestand und Zulauf.
func (t *Tueren) Klassensaetze(w http.ResponseWriter, r *http.Request) {
	zweig := r.URL.Query().Get("branch")
	reihenfolge := r.URL.Query().Get("sort")
	gruppen, err := t.abfragen.ladeKlassensatzGruppen(r.Context(), zweig, reihenfolge)
	if err != nil {
		slog.Error("Fehler beim Laden der Klassengruppen", "fehler", err)
		schreibeFehler(w, http.StatusInternalServerError, "klassen konnten nicht geladen werden")
		return
	}

	if gruppen == nil {
		gruppen = []klassensatzGruppe{}
	}

	schreibeJSON(w, http.StatusOK, map[string]any{"data": gruppen})
}

// KlassensaetzeAendern ist die Tür POST /api/admin/class-books: Sie setzt die Titel einer
// Klasse neu.
func (t *Tueren) KlassensaetzeAendern(w http.ResponseWriter, r *http.Request) {
	var eingabe struct {
		AlteKlasse string   `json:"oldClassName,omitempty"`
		Klasse     string   `json:"className,omitempty"` // die ältere Form des Rumpfs mit einer Klasse
		Klassen    []string `json:"classNames,omitempty"`
		TitelIDs   []string `json:"bookIds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&eingabe); err != nil {
		schreibeFehler(w, http.StatusBadRequest, "Die Anfrage ließ sich nicht lesen")
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
		schreibeFehler(w, http.StatusBadRequest, "Bitte mindestens eine Klasse angeben")
		return
	}

	// Klassennamen-Länge begrenzen (max. 20 Zeichen Schutz gegen Missbrauch)
	for _, cn := range zielKlassen {
		if len(cn) > 20 {
			schreibeFehler(w, http.StatusBadRequest, "Der Name einer Klasse hat höchstens 20 Zeichen")
			return
		}
	}

	// Umbenennen lässt sich sicher nur eine einzelne Klasse.
	alterName := strings.TrimSpace(eingabe.AlteKlasse)

	if !alleUUIDs(eingabe.TitelIDs) {
		schreibeFehler(w, http.StatusBadRequest, "Eine der Kennungen ist ungültig")
		return
	}
	err := t.abfragen.aendereKlassensaetze(r.Context(), alterName, zielKlassen, eingabe.TitelIDs)
	if err != nil {
		slog.Error("Fehler beim Aktualisieren der Klassenbücher", "fehler", err)
		schreibeFehler(w, http.StatusInternalServerError, "klassenbücher konnten nicht aktualisiert werden")
		return
	}

	schreibeJSON(w, http.StatusOK, map[string]any{"message": "Gespeichert"})
}

// KlassensatzGruppeLoeschen ist die Tür DELETE /api/admin/class-books: Sie löscht die
// Zuordnungen einer Klasse.
func (t *Tueren) KlassensatzGruppeLoeschen(w http.ResponseWriter, r *http.Request) {
	abfrageNamen := r.URL.Query()["className"]
	var klassen []string

	for _, name := range abfrageNamen {
		bereinigt := strings.TrimSpace(name)
		if bereinigt != "" {
			klassen = append(klassen, bereinigt)
		}
	}

	if len(klassen) == 0 {
		schreibeFehler(w, http.StatusBadRequest, "Der Name der Klasse fehlt")
		return
	}

	err := t.abfragen.loescheKlassensatzGruppe(r.Context(), klassen)
	if err != nil {
		slog.Error("Fehler beim Löschen der Klassengruppen", "klassen", klassen, "fehler", err)
		schreibeFehler(w, http.StatusInternalServerError, "klasse konnte nicht gelöscht werden")
		return
	}

	schreibeJSON(w, http.StatusOK, map[string]any{"message": "Klasse gelöscht"})
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
func (t *Tueren) KlassensaetzeZuordnen(w http.ResponseWriter, r *http.Request) {
	var eingabe struct {
		Klassen  []string `json:"classNames"`
		TitelIDs []string `json:"bookIds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&eingabe); err != nil {
		schreibeFehler(w, http.StatusBadRequest, "Die Anfrage ließ sich nicht lesen")
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
		schreibeFehler(w, http.StatusBadRequest, "Bitte mindestens eine Klasse angeben")
		return
	}

	for _, cn := range zielKlassen {
		if len(cn) > 20 {
			schreibeFehler(w, http.StatusBadRequest, "Der Name einer Klasse hat höchstens 20 Zeichen")
			return
		}
	}

	if !alleUUIDs(eingabe.TitelIDs) {
		schreibeFehler(w, http.StatusBadRequest, "Eine der Kennungen ist ungültig")
		return
	}
	err := t.abfragen.ordneTitelKlassenZu(r.Context(), zielKlassen, eingabe.TitelIDs)
	if err != nil {
		slog.Error("Fehler beim Hinzufügen der Bücher", "fehler", err)
		schreibeFehler(w, http.StatusInternalServerError, "bücher konnten nicht hinzugefügt werden")
		return
	}

	schreibeJSON(w, http.StatusOK, map[string]any{"message": "Hinzugefügt"})
}
