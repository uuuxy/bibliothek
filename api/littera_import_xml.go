package api

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"bibliothek/auth"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/charmap"
)

// Feld repräsentiert ein einzelnes XML-Feld aus LITTERA mit MAB-Code und Wert.
type litteraFeld struct {
	MAB   string `xml:"MAB,attr"`
	Value string `xml:",chardata"`
}

// Katalogisat bündelt die Felder für einen Datensatz (ein Buch/Medium).
type litteraKatalogisat struct {
	Felder []litteraFeld `xml:"Feld"`
}

func extractValue(felder []litteraFeld, mabCode string) string {
	for _, f := range felder {
		if strings.TrimSpace(f.MAB) == mabCode {
			return strings.TrimSpace(f.Value)
		}
	}
	return ""
}

func cleanTitle(titel string) string {
	return strings.ReplaceAll(titel, "¬", "")
}

func parseYear(y string) int {
	for i := 0; i <= len(y)-4; i++ {
		if y[i] >= '0' && y[i] <= '9' && y[i+1] >= '0' && y[i+1] <= '9' && y[i+2] >= '0' && y[i+2] <= '9' && y[i+3] >= '0' && y[i+3] <= '9' {
			var year int
			fmt.Sscanf(y[i:i+4], "%d", &year)
			return year
		}
	}
	return 0
}

func extractAllValues(felder []litteraFeld, mabCode string) []string {
	var vals []string
	for _, f := range felder {
		if strings.TrimSpace(f.MAB) == mabCode {
			val := strings.TrimSpace(f.Value)
			if val != "" {
				vals = append(vals, val)
			}
		}
	}
	return vals
}

func toUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	dec := charmap.Windows1252.NewDecoder()
	out, err := dec.String(s)
	if err == nil {
		return out
	}
	return strings.Map(func(r rune) rune {
		if r == utf8.RuneError {
			return -1
		}
		return r
	}, s)
}

func limitString(s string, limit int) string {
	runes := []rune(s)
	if len(runes) > limit {
		return string(runes[:limit-3]) + "..."
	}
	return s
}

func (s *Server) handleLitteraXMLImport(w http.ResponseWriter, r *http.Request, content []byte) {
	ctx, cancel := context.WithTimeout(r.Context(), 180*time.Second)
	defer cancel()

	decoder := xml.NewDecoder(bytes.NewReader(content))
	decoder.CharsetReader = charset.NewReaderLabel

	var updatedCount int
	var processedCount int

	for {
		t, err := decoder.Token()
		if err != nil {
			break
		}

		switch se := t.(type) {
		case xml.StartElement:
			if strings.EqualFold(se.Name.Local, "katalogisat") {
				var k litteraKatalogisat
				if err := decoder.DecodeElement(&k, &se); err != nil {
					continue
				}

				processedCount++
				titel := cleanTitle(toUTF8(extractValue(k.Felder, "310")))
				isbn := toUTF8(extractValue(k.Felder, "540"))
				untertitel := toUTF8(extractValue(k.Felder, "335"))
				autor := toUTF8(extractValue(k.Felder, "100"))
				verlag := toUTF8(extractValue(k.Felder, "412"))
				erscheinungsjahr := parseYear(extractValue(k.Felder, "425"))
				beschreibung := toUTF8(extractValue(k.Felder, "750f"))
				signatur := toUTF8(extractValue(k.Felder, "700"))
				standort := toUTF8(extractValue(k.Felder, "108a"))

				subjects := extractAllValues(k.Felder, "710")
				var subject string
				if len(subjects) > 0 {
					for i, sub := range subjects {
						subjects[i] = toUTF8(sub)
					}
					subject = limitString(strings.Join(subjects, ", "), 100)
				}

				if titel != "" {
					query := `
						UPDATE buecher_titel 
						SET 
							isbn = COALESCE(NULLIF(isbn, ''), NULLIF($1, '')),
							untertitel = COALESCE(NULLIF(untertitel, ''), NULLIF($3, '')),
							autor = COALESCE(NULLIF(autor, ''), NULLIF($4, '')),
							verlag = COALESCE(NULLIF(verlag, ''), NULLIF($5, '')),
							erscheinungsjahr = COALESCE(NULLIF(erscheinungsjahr, 0), $6),
							beschreibung = COALESCE(NULLIF(beschreibung, ''), NULLIF($7, '')),
							subject = COALESCE(NULLIF(subject, ''), NULLIF($8, '')),
							erweiterte_eigenschaften = 
								CASE 
									WHEN $9::text <> '' AND $10::text <> '' THEN
										jsonb_set(jsonb_set(COALESCE(erweiterte_eigenschaften, '{}'::jsonb), '{signatur}', to_jsonb($9::text)), '{standort}', to_jsonb($10::text))
									WHEN $9::text <> '' THEN
										jsonb_set(COALESCE(erweiterte_eigenschaften, '{}'::jsonb), '{signatur}', to_jsonb($9::text))
									WHEN $10::text <> '' THEN
										jsonb_set(COALESCE(erweiterte_eigenschaften, '{}'::jsonb), '{standort}', to_jsonb($10::text))
									ELSE COALESCE(erweiterte_eigenschaften, '{}'::jsonb)
								END
						WHERE titel ILIKE $2`

					res, err := s.DB.Pool.Exec(ctx, query,
						isbn, titel, untertitel, autor, verlag, erscheinungsjahr,
						beschreibung, subject, signatur, standort,
					)

					if err == nil && res.RowsAffected() > 0 {
						updatedCount += int(res.RowsAffected())
					}
				}
			}
		}
	}

	if claims, ok := auth.GetClaims(r.Context()); ok {
		details := fmt.Sprintf(`{"updated_titles":%d,"processed":%d,"type":"xml"}`, updatedCount, processedCount)
		_, _ = s.DB.Pool.Exec(ctx, "INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse) VALUES ($1, $2, $3::jsonb, $4)", claims.UserID, "LUSD_IMPORT", details, r.RemoteAddr)
	}

	RespondJSON(w, http.StatusOK, LitteraImportResponse{
		UpdatedTitles: updatedCount,
		Type:          "xml",
	})
}
