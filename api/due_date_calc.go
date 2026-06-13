package api

import (
	"context"
	"strconv"
	"strings"
	"time"

	"bibliothek/repository"
)

func calculateDueDate(titel, medientyp, lmfStichtag string, fristBuchTage, fristMedienTage int) time.Time {
	now := time.Now()
	if strings.HasPrefix(strings.ToLower(titel), "lmf-") {
		year := now.Year()
		if now.Month() >= time.August {
			year++
		}
		month := time.July
		day := 31
		parts := strings.SplitN(lmfStichtag, "-", 2)
		if len(parts) == 2 {
			m, err1 := strconv.Atoi(parts[0])
			d, err2 := strconv.Atoi(parts[1])
			if err1 == nil && err2 == nil && m >= 1 && m <= 12 && d >= 1 && d <= 31 {
				month = time.Month(m)
				day = d
			}
		}
		return time.Date(year, month, day, 23, 59, 59, 0, now.Location())
	}
	lower := strings.ToLower(medientyp)
	if strings.Contains(lower, "cd") || strings.Contains(lower, "dvd") || strings.Contains(lower, "audio") {
		return now.AddDate(0, 0, fristMedienTage)
	}
	return now.AddDate(0, 0, fristBuchTage)
}

func (s *Server) resolveCheckoutDueDate(ctx context.Context, copy *repository.BookCopy) (time.Time, error) {
	settings, err := s.querySettings(ctx)
	if err != nil {
		return calculateDueDate(copy.Titel, copy.Medientyp, "07-31", 21, 7), nil
	}
	isLMF := strings.HasPrefix(strings.ToLower(copy.Titel), "lmf-")
	if !isLMF && settings.FerienLeseclubAktiv && settings.FerienLeseclubZieldatum != nil {
		t, parseErr := time.Parse("2006-01-02", *settings.FerienLeseclubZieldatum)
		if parseErr == nil {
			end := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.Local)
			return end, nil
		}
	}
	return calculateDueDate(copy.Titel, copy.Medientyp, settings.LmfStichtag, settings.FristBuchTage, settings.FristMedienTage), nil
}
