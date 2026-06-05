package inventur

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

type ClassBook struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subject  string `json:"subject"`
	Track    string `json:"track"`
	CoverURL string `json:"coverUrl"`
	ISBN       string `json:"isbn"`
	Stock      int    `json:"stock"`
	Verfuegbar int    `json:"verfuegbar"`
	Gesamt     int    `json:"gesamt"`
}

type ClassGroup struct {
	ClassName string      `json:"className"`
	Books     []ClassBook `json:"books"`
}

func (handler *APIHandler) handleClassBooks(writer http.ResponseWriter, request *http.Request) {
	branch := request.URL.Query().Get("branch")
	sortOrder := request.URL.Query().Get("sort")
	groups, err := handler.repo.GetClassGroups(request.Context(), branch, sortOrder)
	if err != nil {
		log.Printf("Fehler beim Laden der Klassengruppen: %v", err)
		writeError(writer, http.StatusInternalServerError, "klassen konnten nicht geladen werden")
		return
	}

	if groups == nil {
		groups = []ClassGroup{}
	}

	writeJSON(writer, http.StatusOK, map[string]any{"data": groups})
}

func (handler *APIHandler) handleUpdateClassBooks(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		OldClassName string   `json:"oldClassName,omitempty"`
		ClassName    string   `json:"className,omitempty"` // For backwards compatibility
		ClassNames   []string `json:"classNames,omitempty"`
		BookIDs      []string `json:"bookIds"`
	}

	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeError(writer, http.StatusBadRequest, "ungültiges JSON")
		return
	}

	// Support both single className and multi classNames payload
	var targetClasses []string
	if len(input.ClassNames) > 0 {
		for _, name := range input.ClassNames {
			trimmed := strings.TrimSpace(name)
			if trimmed != "" {
				targetClasses = append(targetClasses, formatClassName(trimmed))
			}
		}
	} else if strings.TrimSpace(input.ClassName) != "" {
		targetClasses = append(targetClasses, formatClassName(strings.TrimSpace(input.ClassName)))
	}

	if len(targetClasses) == 0 {
		writeError(writer, http.StatusBadRequest, "es muss mindestens ein klassenname angegeben werden")
		return
	}

	// Klassennamen-Länge begrenzen (max. 20 Zeichen Schutz gegen Missbrauch)
	for _, cn := range targetClasses {
		if len(cn) > 20 {
			writeError(writer, http.StatusBadRequest, "klassenname darf maximal 20 zeichen lang sein")
			return
		}
	}

	// We can only rename a single class safely
	oldName := strings.TrimSpace(input.OldClassName)

	err := handler.repo.UpdateClassBooks(request.Context(), oldName, targetClasses, input.BookIDs)
	if err != nil {
		log.Printf("Fehler beim Aktualisieren der Klassenbücher: %v", err)
		writeError(writer, http.StatusInternalServerError, "klassenbücher konnten nicht aktualisiert werden")
		return
	}

	writeJSON(writer, http.StatusOK, map[string]any{"message": "erfolgreich gespeichert"})
}

func (handler *APIHandler) handleDeleteClassGroup(writer http.ResponseWriter, request *http.Request) {
	className := request.URL.Query().Get("className")
	className = strings.TrimSpace(className)

	if className == "" {
		writeError(writer, http.StatusBadRequest, "klassenname fehlt")
		return
	}

	err := handler.repo.DeleteClassGroup(request.Context(), className)
	if err != nil {
		log.Printf("Fehler beim Löschen der Klassengruppe %s: %v", className, err)
		writeError(writer, http.StatusInternalServerError, "klasse konnte nicht gelöscht werden")
		return
	}

	writeJSON(writer, http.StatusOK, map[string]any{"message": "klasse gelöscht"})
}

func formatClassName(name string) string {
	name = strings.ReplaceAll(name, " ", "")
	if len(name) > 0 && name[0] >= '1' && name[0] <= '9' {
		if len(name) == 1 || (name[1] < '0' || name[1] > '9') {
			return "0" + name
		}
	}
	return name
}

func (handler *APIHandler) handleAddClassBooks(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		ClassNames []string `json:"classNames"`
		BookIDs    []string `json:"bookIds"`
	}

	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeError(writer, http.StatusBadRequest, "ungültiges JSON")
		return
	}

	var targetClasses []string
	for _, name := range input.ClassNames {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" {
			targetClasses = append(targetClasses, formatClassName(trimmed))
		}
	}

	if len(targetClasses) == 0 {
		writeError(writer, http.StatusBadRequest, "es muss mindestens ein klassenname angegeben werden")
		return
	}

	for _, cn := range targetClasses {
		if len(cn) > 20 {
			writeError(writer, http.StatusBadRequest, "klassenname darf maximal 20 zeichen lang sein")
			return
		}
	}

	err := handler.repo.AddBooksToClasses(request.Context(), targetClasses, input.BookIDs)
	if err != nil {
		log.Printf("Fehler beim Hinzufügen der Bücher: %v", err)
		writeError(writer, http.StatusInternalServerError, "bücher konnten nicht hinzugefügt werden")
		return
	}

	writeJSON(writer, http.StatusOK, map[string]any{"message": "erfolgreich hinzugefügt"})
}
