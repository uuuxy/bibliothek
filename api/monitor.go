package api

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"bibliothek/apierrors"

	"github.com/jackc/pgx/v5"
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

// queryBuchDesMonats liefert das „Buch des Monats" (meistausgeliehener Titel mit Cover
// der letzten 30 Tage); fehlt eines, wird der zuletzt hinzugefügte Titel mit Cover als
// Fallback genutzt. ok=false: die Fehlerantwort wurde bereits geschrieben.
func (s *Server) queryBuchDesMonats(ctx context.Context, w http.ResponseWriter) (*MonitorTitel, bool) {
	// Buch des Monats: most borrowed title in the last 30 days that has a cover.
	var bm MonitorTitel
	err := s.DB.Pool.QueryRow(ctx, `
		SELECT bt.id, bt.titel, COALESCE(bt.autor,''), COALESCE(bt.cover_url,'')
		FROM ausleihen a
		JOIN buecher_exemplare e ON e.id = a.exemplar_id
		JOIN buecher_titel bt ON bt.id = e.titel_id
		WHERE a.ausgeliehen_am >= NOW() - INTERVAL '30 days'
		  AND bt.cover_url IS NOT NULL AND bt.cover_url <> ''
		GROUP BY bt.id, bt.titel, bt.autor, bt.cover_url
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`).Scan(&bm.ID, &bm.Titel, &bm.Autor, &bm.CoverURL)

	if err == nil {
		return &bm, true
	}
	if err != pgx.ErrNoRows {
		log.Printf("DB Error in Monitor (Buch des Monats): %v", err)
		apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return nil, false
	}

	// Fallback: the most recently added title with a cover.
	var fb MonitorTitel
	err = s.DB.Pool.QueryRow(ctx, `
		SELECT id, titel, COALESCE(autor,''), COALESCE(cover_url,'')
		FROM buecher_titel
		WHERE cover_url IS NOT NULL AND cover_url <> ''
		ORDER BY erstellt_am DESC LIMIT 1
	`).Scan(&fb.ID, &fb.Titel, &fb.Autor, &fb.CoverURL)

	if err == nil {
		return &fb, true
	}
	if err != pgx.ErrNoRows {
		log.Printf("DB Error in Monitor (Fallback Buch des Monats): %v", err)
		apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return nil, false
	}
	return nil, true // kein Buch des Monats vorhanden, aber kein Fehler
}

// queryMonitorListe führt eine Titel-Listen-Query aus und mappt sie auf MonitorTitel.
// sektion dient als Label für die (unveränderten) Fehlermeldungen. ok=false: die
// Fehlerantwort wurde bereits geschrieben.
func (s *Server) queryMonitorListe(ctx context.Context, w http.ResponseWriter, query, sektion string) ([]MonitorTitel, bool) {
	rows, err := s.DB.Pool.Query(ctx, query)
	if err != nil {
		log.Printf("DB Error in Monitor (%s): %v", sektion, err)
		apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return nil, false
	}
	defer rows.Close()

	liste := []MonitorTitel{}
	for rows.Next() {
		var t MonitorTitel
		if err := rows.Scan(&t.ID, &t.Titel, &t.Autor, &t.CoverURL); err != nil {
			log.Printf("DB Error in Monitor (Scan %s): %v", sektion, err)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
			return nil, false
		}
		liste = append(liste, t)
	}
	if err := rows.Err(); err != nil {
		log.Printf("DB Error in Monitor (Iteration %s): %v", sektion, err)
		apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return nil, false
	}
	return liste, true
}

// GetMonitorSlidesHandler handles GET /api/monitor/slides (public, no auth).
func (s *Server) GetMonitorSlidesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		slides := MonitorSlides{
			NeuEingetroffen: []MonitorTitel{},
			Beliebt:         []MonitorTitel{},
		}

		bdm, ok := s.queryBuchDesMonats(ctx, w)
		if !ok {
			return
		}
		slides.BuchDesMonats = bdm

		// Neu eingetroffen: last 10 titles added with a cover.
		neu, ok := s.queryMonitorListe(ctx, w, `
			SELECT id, titel, COALESCE(autor,''), COALESCE(cover_url,'')
			FROM buecher_titel
			WHERE cover_url IS NOT NULL AND cover_url <> ''
			ORDER BY erstellt_am DESC
			LIMIT 10
		`, "Neu eingetroffen")
		if !ok {
			return
		}
		slides.NeuEingetroffen = neu

		// Beliebt: top 5 titles by loan count in the last 7 days.
		beliebt, ok := s.queryMonitorListe(ctx, w, `
			SELECT bt.id, bt.titel, COALESCE(bt.autor,''), COALESCE(bt.cover_url,'')
			FROM ausleihen a
			JOIN buecher_exemplare e ON e.id = a.exemplar_id
			JOIN buecher_titel bt ON bt.id = e.titel_id
			WHERE a.ausgeliehen_am >= NOW() - INTERVAL '7 days'
			GROUP BY bt.id, bt.titel, bt.autor, bt.cover_url
			ORDER BY COUNT(*) DESC
			LIMIT 5
		`, "Beliebt")
		if !ok {
			return
		}
		slides.Beliebt = beliebt

		RespondJSON(w, http.StatusOK, slides)
	}
}
