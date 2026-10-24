package katalogdienste

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// sucheOpenLibrary durchsucht das Internet-Archive/OpenLibrary.
func (c *MetadatenClient) sucheOpenLibrary(kontext context.Context, isbn string) (*MetadatenErgebnis, error) {
	adresse := fmt.Sprintf("https://openlibrary.org/api/books?bibkeys=ISBN:%s&format=json&jscmd=data", isbn)
	koerper, fehler := c.holeInhalt(kontext, adresse)
	if fehler != nil {
		return nil, fehler
	}

	var nutzlast map[string]struct {
		Titel      string `json:"title"`
		Untertitel string `json:"subtitle"`
		Autoren    []struct {
			Name string `json:"name"`
		} `json:"authors"`
		Cover struct {
			Mittel string `json:"medium"`
			Gross  string `json:"large"`
		} `json:"cover"`
	}
	if fehler := json.Unmarshal(koerper, &nutzlast); fehler != nil {
		return nil, fehler
	}

	eintrag, existiert := nutzlast["ISBN:"+isbn]
	if !existiert {
		return nil, fmt.Errorf("nicht gefunden")
	}

	autor := ""
	if len(eintrag.Autoren) > 0 {
		autor = eintrag.Autoren[0].Name
	}
	coverBild := eintrag.Cover.Gross
	if coverBild == "" {
		coverBild = eintrag.Cover.Mittel
	}

	return &MetadatenErgebnis{
		ISBN:       isbn,
		Titel:      eintrag.Titel,
		Untertitel: eintrag.Untertitel,
		Autor:      autor,
		CoverURL:   coverBild,
	}, nil
}

// sucheBeiGoogle ist ein robuster Fallback auf Googles Buch-API.
func (c *MetadatenClient) sucheBeiGoogle(kontext context.Context, isbn string) (*MetadatenErgebnis, error) {
	adresse := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=isbn:%s", isbn)
	koerper, fehler := c.holeInhalt(kontext, adresse)
	if fehler != nil {
		return nil, fehler
	}

	var nutzlast struct {
		Eintraege []struct {
			Angaben struct {
				Titel      string   `json:"title"`
				Untertitel string   `json:"subtitle"`
				Autoren    []string `json:"authors"`
				Bilder     struct {
					Vorschau       string `json:"thumbnail"`
					KleineVorschau string `json:"smallThumbnail"`
				} `json:"imageLinks"`
			} `json:"volumeInfo"`
		} `json:"items"`
	}
	if fehler := json.Unmarshal(koerper, &nutzlast); fehler != nil {
		return nil, fehler
	}
	if len(nutzlast.Eintraege) == 0 {
		return nil, fmt.Errorf("nicht gefunden")
	}

	angaben := nutzlast.Eintraege[0].Angaben
	autor := ""
	if len(angaben.Autoren) > 0 {
		autor = angaben.Autoren[0]
	}
	coverBild := angaben.Bilder.Vorschau
	if coverBild == "" {
		coverBild = angaben.Bilder.KleineVorschau
	}
	coverBild = strings.ReplaceAll(coverBild, "http://", "https://")

	return &MetadatenErgebnis{
		ISBN:       isbn,
		Titel:      angaben.Titel,
		Untertitel: angaben.Untertitel,
		Autor:      autor,
		CoverURL:   coverBild,
	}, nil
}
