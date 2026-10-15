package inventur

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"bibliothek/repository"
)

// klassensatzTitel ist ein Klassensatz-Titel, wie ihn das Lehrerportal anzeigt.
type klassensatzTitel struct {
	ID         string `json:"id"`
	Titel      string `json:"title"`
	Fach       string `json:"subject"`
	Zweig      string `json:"track"`
	CoverURL   string `json:"coverUrl"`
	ISBN       string `json:"isbn"`
	Verfuegbar int    `json:"verfuegbar"`
	Gesamt     int    `json:"gesamt"`
	ImZulauf   int    `json:"imZulauf"`
	// Quelle: "hand" = in class_books gepflegt; "ausleihe" = live aus den laufenden
	// Ausleihen abgeleitet (mehr als die Hälfte der Klasse hält den Titel). Leser = wie
	// viele Kinder der Klasse ihn gerade haben (nur bei "ausleihe" gefüllt).
	Quelle string `json:"quelle"`
	Leser  int    `json:"leser"`
	// Auflagen: Haben Kinder der Klasse eine andere Auflage desselben Buchs als die der
	// Kachel, stehen hier alle Auflagen, die die Klasse hat, mit der Zahl der Kinder — die
	// meisten zuerst (docs/OFFEN.md 4.18, Stufe 5). Sonst fehlt das Feld.
	Auflagen []repository.AuflageInKlasse `json:"auflagen,omitempty"`
}

// klassensatzGruppe bündelt die Klassensatz-Titel EINER Klasse — die Gruppierung passiert
// serverseitig, damit die Oberfläche nicht über alle Zuordnungen iterieren muss.
type klassensatzGruppe struct {
	Klasse string             `json:"className"`
	Titel  []klassensatzTitel `json:"books"`
}

// Klassensaetze ist die Tür GET /api/class-books (und GET /api/portal/klassensaetze für das
// Kollegium): die Klassensätze je Klasse mit Bestand und Zulauf.
func (t *Tueren) Klassensaetze(w http.ResponseWriter, r *http.Request) {
	zweig := r.URL.Query().Get("branch")
	reihenfolge := r.URL.Query().Get("sort")
	gruppen, err := t.abfragen.LadeKlassensatzGruppen(r.Context(), zweig, reihenfolge)
	if err != nil {
		log.Printf("Fehler beim Laden der Klassengruppen: %v", err)
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
		OldClassName string   `json:"oldClassName,omitempty"`
		ClassName    string   `json:"className,omitempty"` // For backwards compatibility
		ClassNames   []string `json:"classNames,omitempty"`
		BookIDs      []string `json:"bookIds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&eingabe); err != nil {
		schreibeFehler(w, http.StatusBadRequest, "ungültiges JSON")
		return
	}

	// Support both single className and multi classNames payload
	var zielKlassen []string
	if len(eingabe.ClassNames) > 0 {
		for _, name := range eingabe.ClassNames {
			bereinigt := strings.TrimSpace(name)
			if bereinigt != "" {
				zielKlassen = append(zielKlassen, normalisiereKlassenname(bereinigt))
			}
		}
	} else if strings.TrimSpace(eingabe.ClassName) != "" {
		zielKlassen = append(zielKlassen, normalisiereKlassenname(strings.TrimSpace(eingabe.ClassName)))
	}

	if len(zielKlassen) == 0 {
		schreibeFehler(w, http.StatusBadRequest, "es muss mindestens ein klassenname angegeben werden")
		return
	}

	// Klassennamen-Länge begrenzen (max. 20 Zeichen Schutz gegen Missbrauch)
	for _, cn := range zielKlassen {
		if len(cn) > 20 {
			schreibeFehler(w, http.StatusBadRequest, "klassenname darf maximal 20 zeichen lang sein")
			return
		}
	}

	// We can only rename a single class safely
	alterName := strings.TrimSpace(eingabe.OldClassName)

	if !alleUUIDs(eingabe.BookIDs) {
		schreibeFehler(w, http.StatusBadRequest, "bookIds enthält eine ungültige Kennung")
		return
	}
	err := t.abfragen.AendereKlassensaetze(r.Context(), alterName, zielKlassen, eingabe.BookIDs)
	if err != nil {
		log.Printf("Fehler beim Aktualisieren der Klassenbücher: %v", err)
		schreibeFehler(w, http.StatusInternalServerError, "klassenbücher konnten nicht aktualisiert werden")
		return
	}

	schreibeJSON(w, http.StatusOK, map[string]any{"message": "erfolgreich gespeichert"})
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
		schreibeFehler(w, http.StatusBadRequest, "klassenname fehlt")
		return
	}

	err := t.abfragen.LoescheKlassensatzGruppe(r.Context(), klassen)
	if err != nil {
		log.Printf("Fehler beim Löschen der Klassengruppen %v: %v", klassen, err)
		schreibeFehler(w, http.StatusInternalServerError, "klasse konnte nicht gelöscht werden")
		return
	}

	schreibeJSON(w, http.StatusOK, map[string]any{"message": "klasse gelöscht"})
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
		ClassNames []string `json:"classNames"`
		BookIDs    []string `json:"bookIds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&eingabe); err != nil {
		schreibeFehler(w, http.StatusBadRequest, "ungültiges JSON")
		return
	}

	var zielKlassen []string
	for _, name := range eingabe.ClassNames {
		bereinigt := strings.TrimSpace(name)
		if bereinigt != "" {
			zielKlassen = append(zielKlassen, normalisiereKlassenname(bereinigt))
		}
	}

	if len(zielKlassen) == 0 {
		schreibeFehler(w, http.StatusBadRequest, "es muss mindestens ein klassenname angegeben werden")
		return
	}

	for _, cn := range zielKlassen {
		if len(cn) > 20 {
			schreibeFehler(w, http.StatusBadRequest, "klassenname darf maximal 20 zeichen lang sein")
			return
		}
	}

	if !alleUUIDs(eingabe.BookIDs) {
		schreibeFehler(w, http.StatusBadRequest, "bookIds enthält eine ungültige Kennung")
		return
	}
	err := t.abfragen.OrdneTitelKlassenZu(r.Context(), zielKlassen, eingabe.BookIDs)
	if err != nil {
		log.Printf("Fehler beim Hinzufügen der Bücher: %v", err)
		schreibeFehler(w, http.StatusInternalServerError, "bücher konnten nicht hinzugefügt werden")
		return
	}

	schreibeJSON(w, http.StatusOK, map[string]any{"message": "erfolgreich hinzugefügt"})
}
