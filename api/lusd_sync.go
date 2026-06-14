package api

import (
	"context"
	"fmt"
	"strings"

	"bibliothek/repository"
)

// syncLUSDData synchronizes the parsed LUSD rows with the database, performing batch inserts, updates, and flagging graduates.
// It relies completely on the StudentRepository to execute the database operations, enforcing clean architecture.
func syncLUSDData(ctx context.Context, studentRepo repository.StudentRepository, parsedRows []parsedStudentRow, lusdIDs []string) (*LUSDImportResponse, error) {
	startNum, err := studentRepo.GetNextSequence(ctx)
	if err != nil {
		// Log error but default to 10001
		startNum = 10001
	}

	var newCount int
	var updatedCount int

	dbStudents, err := studentRepo.GetAllLUSDStudents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load existing students: %w", err)
	}

	mapLusd := make(map[string]string)
	mapFallback := make(map[string]string)
	for _, s := range dbStudents {
		if s.LusdID != nil && *s.LusdID != "" {
			mapLusd[*s.LusdID] = s.ID
		}
		var gebStr string
		if s.Geburtsdatum != nil {
			gebStr = *s.Geburtsdatum
		} else {
			gebStr = "1900-01-01"
		}
		key := strings.ToLower(s.Vorname) + "|" + strings.ToLower(s.Nachname) + "|" + gebStr
		mapFallback[key] = s.ID
	}

	var updates []repository.StudentUpdate
	var inserts []repository.StudentInsert

	for _, p := range parsedRows {
		var dbID string
		if p.LusdID != "" {
			dbID = mapLusd[p.LusdID]
		}
		if dbID == "" {
			gebStr := "1900-01-01"
			if p.GebDatum != nil {
				gebStr = p.GebDatum.Format("2006-01-02")
			}
			key := strings.ToLower(p.Vorname) + "|" + strings.ToLower(p.Nachname) + "|" + gebStr
			dbID = mapFallback[key]
		}

		var ptrLusd *string
		if p.LusdID != "" {
			lusd := p.LusdID
			ptrLusd = &lusd
		}
		var ptrGeb *string
		if p.GebDatum != nil {
			g := p.GebDatum.Format("2006-01-02")
			ptrGeb = &g
		}

		if dbID != "" && dbID != "processing" {
			updates = append(updates, repository.StudentUpdate{
				ID:           dbID,
				Vorname:      p.Vorname,
				Nachname:     p.Nachname,
				Klasse:       p.Klasse,
				Geburtsdatum: ptrGeb,
				LusdID:       ptrLusd,
			})
			updatedCount++
		} else {
			barcode := fmt.Sprintf("S-%05d", startNum)
			startNum++
			inserts = append(inserts, repository.StudentInsert{
				BarcodeID:     barcode,
				Vorname:       p.Vorname,
				Nachname:      p.Nachname,
				Klasse:        p.Klasse,
				Geburtsdatum:  ptrGeb,
				AbgaengerJahr: calculateAbgaengerJahr(p.Klasse),
				LusdID:        ptrLusd,
				IstAbgaenger:  false,
			})
			newCount++

			if p.LusdID != "" {
				mapLusd[p.LusdID] = "processing"
			}
		}
	}

	abgaengerOpenCount, err := studentRepo.BulkSyncLUSD(ctx, updates, inserts, lusdIDs)
	if err != nil {
		return nil, fmt.Errorf("bulk sync failed: %w", err)
	}

	return &LUSDImportResponse{
		Neu:                         newCount,
		Aktualisiert:                updatedCount,
		AbgaengerMitOffenenBuechern: abgaengerOpenCount,
	}, nil
}
