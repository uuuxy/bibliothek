package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// MonitorTitel is a slim book model for the public info monitor.
type MonitorTitel struct {
	ID       string `json:"id"`
	Titel    string `json:"titel"`
	Autor    string `json:"autor"`
	CoverURL string `json:"cover_url"`
}

// MonitorSlides is the full response for the public info monitor.
type MonitorSlides struct {
	BuchDesMonats   *MonitorTitel  `json:"buch_des_monats"`
	NeuEingetroffen []MonitorTitel `json:"neu_eingetroffen"`
	Beliebt         []MonitorTitel `json:"beliebt"`
}

// GetMonitorSlidesHandler handles GET /api/monitor/slides (public, no auth).
func (s *Server) GetMonitorSlidesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		slides := MonitorSlides{
			NeuEingetroffen: []MonitorTitel{},
			Beliebt:         []MonitorTitel{},
		}

		// Buch des Monats: most borrowed title in the last 30 days that has a cover.
		var bm MonitorTitel
		if s.DB.Pool.QueryRow(ctx, `
			SELECT bt.id, bt.titel, COALESCE(bt.autor,''), COALESCE(bt.cover_url,'')
			FROM ausleihen a
			JOIN buecher_exemplare e ON e.id = a.exemplar_id
			JOIN buecher_titel bt ON bt.id = e.titel_id
			WHERE a.ausgeliehen_am >= NOW() - INTERVAL '30 days'
			  AND bt.cover_url IS NOT NULL AND bt.cover_url <> ''
			GROUP BY bt.id, bt.titel, bt.autor, bt.cover_url
			ORDER BY COUNT(*) DESC
			LIMIT 1
		`).Scan(&bm.ID, &bm.Titel, &bm.Autor, &bm.CoverURL) == nil {
			slides.BuchDesMonats = &bm
		}
		// Fallback: the most recently added title with a cover.
		if slides.BuchDesMonats == nil {
			var fb MonitorTitel
			if s.DB.Pool.QueryRow(ctx, `
				SELECT id, titel, COALESCE(autor,''), COALESCE(cover_url,'')
				FROM buecher_titel
				WHERE cover_url IS NOT NULL AND cover_url <> ''
				ORDER BY erstellt_am DESC LIMIT 1
			`).Scan(&fb.ID, &fb.Titel, &fb.Autor, &fb.CoverURL) == nil {
				slides.BuchDesMonats = &fb
			}
		}

		// Neu eingetroffen: last 10 titles added with a cover.
		if rows, err := s.DB.Pool.Query(ctx, `
			SELECT id, titel, COALESCE(autor,''), COALESCE(cover_url,'')
			FROM buecher_titel
			WHERE cover_url IS NOT NULL AND cover_url <> ''
			ORDER BY erstellt_am DESC
			LIMIT 10
		`); err == nil {
			defer rows.Close()
			for rows.Next() {
				var t MonitorTitel
				if rows.Scan(&t.ID, &t.Titel, &t.Autor, &t.CoverURL) == nil {
					slides.NeuEingetroffen = append(slides.NeuEingetroffen, t)
				}
			}
		}

		// Beliebt: top 5 titles by loan count in the last 7 days.
		if rows2, err := s.DB.Pool.Query(ctx, `
			SELECT bt.id, bt.titel, COALESCE(bt.autor,''), COALESCE(bt.cover_url,'')
			FROM ausleihen a
			JOIN buecher_exemplare e ON e.id = a.exemplar_id
			JOIN buecher_titel bt ON bt.id = e.titel_id
			WHERE a.ausgeliehen_am >= NOW() - INTERVAL '7 days'
			GROUP BY bt.id, bt.titel, bt.autor, bt.cover_url
			ORDER BY COUNT(*) DESC
			LIMIT 5
		`); err == nil {
			defer rows2.Close()
			for rows2.Next() {
				var t MonitorTitel
				if rows2.Scan(&t.ID, &t.Titel, &t.Autor, &t.CoverURL) == nil {
					slides.Beliebt = append(slides.Beliebt, t)
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(slides)
	}
}
