package jobs

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"
)

type antolinAPIResp struct {
	Result []struct {
		Stufen string `json:"stufen"`
		Punkte int    `json:"punkte"`
	} `json:"result"`
}

// RunAntolinSync ruft Antolin-Daten für Bücher mit ISBNs ab und speichert sie in der Datenbank.
func (s *Scheduler) RunAntolinSync() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	log.Println("Scheduler Antolin Sync: starte Hintergrund-Sync...")

	// Suche alle Titel mit einer ISBN, bei denen die Antolin-Daten in den letzten 30 Tagen nicht geprüft wurden
	// Oder antolin_geprueft_am ist NULL.
	query := `
		SELECT id, isbn
		FROM buecher_titel
		WHERE isbn IS NOT NULL AND isbn != ''
		  AND (antolin_geprueft_am IS NULL OR antolin_geprueft_am < NOW() - INTERVAL '30 days')
		ORDER BY aktualisiert_am ASC
	`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		log.Printf("Scheduler Antolin Sync: failed to fetch titles: %v", err)
		return
	}

	type SyncTarget struct {
		ID   string
		ISBN string
	}
	var targets []SyncTarget
	for rows.Next() {
		var t SyncTarget
		if err := rows.Scan(&t.ID, &t.ISBN); err != nil {
			log.Printf("Scheduler Antolin Sync: scan error: %v", err)
			continue
		}
		targets = append(targets, t)
	}
	rows.Close()

	if len(targets) == 0 {
		log.Println("Scheduler Antolin Sync: no titles require syncing.")
		return
	}
	log.Printf("Scheduler Antolin Sync: found %d titles to sync", len(targets))

	client := &http.Client{Timeout: 5 * time.Second}
	updatedCount := 0

	for _, target := range targets {
		if ctx.Err() != nil {
			log.Println("Scheduler Antolin Sync: timed out")
			break
		}

		time.Sleep(500 * time.Millisecond) // KRITISCH: Verzögerung für Rate Limiting

		apiURL := "https://www.antolin.de/all/jsonBuecher.do?isbn=" + url.QueryEscape(target.ISBN)
		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			log.Printf("Scheduler Antolin Sync: req error for ISBN %s: %v", target.ISBN, err)
			continue
		}
		req.Header.Set("User-Agent", "Bibliothek-OPAC/1.0")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Scheduler Antolin Sync: request failed for ISBN %s: %v", target.ISBN, err)
			continue
		}
		
		var apiData antolinAPIResp
		if resp.StatusCode == http.StatusOK {
			_ = json.NewDecoder(resp.Body).Decode(&apiData)
		}
		resp.Body.Close()

		// Vorbereitung des Updates
		var stufen *string
		var punkte *int

		if len(apiData.Result) > 0 {
			st := apiData.Result[0].Stufen
			pt := apiData.Result[0].Punkte
			stufen = &st
			punkte = &pt
		}

		updateQuery := `
			UPDATE buecher_titel
			SET antolin_stufen = $1,
			    antolin_punkte = $2,
			    antolin_geprueft_am = CURRENT_TIMESTAMP
			WHERE id = $3
		`
		_, err = s.db.Exec(ctx, updateQuery, stufen, punkte, target.ID)
		if err != nil {
			log.Printf("Scheduler Antolin Sync: failed to update title ID %s: %v", target.ID, err)
		} else {
			updatedCount++
		}
	}

	log.Printf("Scheduler Antolin Sync: completed, updated %d titles.", updatedCount)
}
