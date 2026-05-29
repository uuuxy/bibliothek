package inventur

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// sucheOpenLibrary durchsucht das Internet-Archive/OpenLibrary.
func (client *MetadatenClient) sucheOpenLibrary(kontext context.Context, isbn string) (*MetadatenErgebnis, error) {
	url := fmt.Sprintf("https://openlibrary.org/api/books?bibkeys=ISBN:%s&format=json&jscmd=data", isbn)
	koerper, fehler := client.holeInhalt(kontext, url)
	if fehler != nil {
		return nil, fehler
	}

	var nutzlast map[string]struct {
		Title    string `json:"title"`
		Subtitle string `json:"subtitle"`
		Authors  []struct {
			Name string `json:"name"`
		} `json:"authors"`
		Cover struct {
			Medium string `json:"medium"`
			Large  string `json:"large"`
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
	if len(eintrag.Authors) > 0 {
		autor = eintrag.Authors[0].Name
	}
	coverBild := eintrag.Cover.Large
	if coverBild == "" {
		coverBild = eintrag.Cover.Medium
	}

	return &MetadatenErgebnis{
		ISBN:       isbn,
		Titel:      eintrag.Title,
		Untertitel: eintrag.Subtitle,
		Autor:      autor,
		CoverURL:   coverBild,
	}, nil
}

// sucheGoogleBooks ist ein robuster Fallback auf Googles Buch-API.
func (client *MetadatenClient) sucheGoogleBooks(kontext context.Context, isbn string) (*MetadatenErgebnis, error) {
	url := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=isbn:%s", isbn)
	koerper, fehler := client.holeInhalt(kontext, url)
	if fehler != nil {
		return nil, fehler
	}

	var nutzlast struct {
		Items []struct {
			VolumeInfo struct {
				Title      string   `json:"title"`
				Subtitle   string   `json:"subtitle"`
				Authors    []string `json:"authors"`
				ImageLinks struct {
					Thumbnail      string `json:"thumbnail"`
					SmallThumbnail string `json:"smallThumbnail"`
				} `json:"imageLinks"`
			} `json:"volumeInfo"`
		} `json:"items"`
	}
	if fehler := json.Unmarshal(koerper, &nutzlast); fehler != nil {
		return nil, fehler
	}
	if len(nutzlast.Items) == 0 {
		return nil, fmt.Errorf("nicht gefunden")
	}

	buchInfo := nutzlast.Items[0].VolumeInfo
	autor := ""
	if len(buchInfo.Authors) > 0 {
		autor = buchInfo.Authors[0]
	}
	coverBild := buchInfo.ImageLinks.Thumbnail
	if coverBild == "" {
		coverBild = buchInfo.ImageLinks.SmallThumbnail
	}
	coverBild = strings.ReplaceAll(coverBild, "http://", "https://")

	return &MetadatenErgebnis{
		ISBN:       isbn,
		Titel:      buchInfo.Title,
		Untertitel: buchInfo.Subtitle,
		Autor:      autor,
		CoverURL:   coverBild,
	}, nil
}
