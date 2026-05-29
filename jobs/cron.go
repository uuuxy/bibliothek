package jobs

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
)

// Scheduler manages background automation tasks.
type Scheduler struct {
	db   *pgxpool.Pool
	cron *cron.Cron
}

// NewScheduler builds and returns a new Scheduler instance.
func NewScheduler(db *pgxpool.Pool) *Scheduler {
	return &Scheduler{
		db:   db,
		cron: cron.New(),
	}
}

// Start registers GDPR deletion and anonymization cron schedules.
func (s *Scheduler) Start() {
	// Run once daily at midnight
	_, err := s.cron.AddFunc("0 0 * * *", func() {
		s.RunGDPRAnonymizeLoans()
		s.RunGDPRDeleteStudents()
	})
	if err != nil {
		log.Printf("Scheduler: Failed to register GDPR jobs: %v", err)
		return
	}

	s.cron.Start()
	log.Println("Scheduler: GDPR background jobs successfully started.")
}

// Stop halts the scheduler's cron runner.
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// RunGDPRAnonymizeLoans nullifies staff operators' IDs for loans closed for more than 14 days.
func (s *Scheduler) RunGDPRAnonymizeLoans() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		UPDATE ausleihen
		SET bearbeiter_id = NULL,
		    rueckgabe_bearbeiter_id = NULL
		WHERE rueckgabe_am < NOW() - INTERVAL '14 days'
		  AND (bearbeiter_id IS NOT NULL OR rueckgabe_bearbeiter_id IS NOT NULL)
	`
	tag, err := s.db.Exec(ctx, query)
	if err != nil {
		log.Printf("Scheduler GDPR: Error anonymizing historical operator IDs: %v", err)
		return
	}
	log.Printf("Scheduler GDPR: Anonymized %d loans returned >14 days ago", tag.RowsAffected())
}

// RunGDPRDeleteStudents transactionally purges students who graduated/left and have no outstanding dues.
func (s *Scheduler) RunGDPRDeleteStudents() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Printf("Scheduler GDPR: Failed to open transaction: %v", err)
		return
	}
	defer tx.Rollback(ctx)

	// Fetch student IDs where graduation year has passed and they have no active loans or unpaid balances
	queryEligible := `
		SELECT id 
		FROM schueler
		WHERE abgaenger_jahr < EXTRACT(YEAR FROM CURRENT_DATE)
		  AND NOT EXISTS (
		      SELECT 1 FROM ausleihen 
		      WHERE schueler_id = schueler.id AND rueckgabe_am IS NULL
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM schadensfaelle 
		      WHERE schueler_id = schueler.id AND ist_bezahlt = false
		  )
	`
	rows, err := tx.Query(ctx, queryEligible)
	if err != nil {
		log.Printf("Scheduler GDPR: Failed to fetch eligible students: %v", err)
		return
	}
	defer rows.Close()

	var studentIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Printf("Scheduler GDPR: Scanning error: %v", err)
			return
		}
		studentIDs = append(studentIDs, id)
	}

	if len(studentIDs) == 0 {
		return
	}

	// Unlink student references from past closed loans to preserve statistical reports
	queryAnonymizeLoans := `
		UPDATE ausleihen
		SET schueler_id = NULL
		WHERE schueler_id = ANY($1)
	`
	_, err = tx.Exec(ctx, queryAnonymizeLoans, studentIDs)
	if err != nil {
		log.Printf("Scheduler GDPR: Anonymizing loans failed: %v", err)
		return
	}

	// Purge historical paid damage cases linked to these students
	queryDeleteDamages := `
		DELETE FROM schadensfaelle
		WHERE schueler_id = ANY($1) AND ist_bezahlt = true
	`
	_, err = tx.Exec(ctx, queryDeleteDamages, studentIDs)
	if err != nil {
		log.Printf("Scheduler GDPR: Deleting paid damages failed: %v", err)
		return
	}

	// Delete student records
	queryDeleteStudents := `
		DELETE FROM schueler
		WHERE id = ANY($1)
	`
	tag, err := tx.Exec(ctx, queryDeleteStudents, studentIDs)
	if err != nil {
		log.Printf("Scheduler GDPR: Purging students failed: %v", err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("Scheduler GDPR: Transaction commit failed: %v", err)
		return
	}

	log.Printf("Scheduler GDPR: Successfully purged %d students and anonymized history", tag.RowsAffected())
}
