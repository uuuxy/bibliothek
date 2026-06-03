package api

import (
	"encoding/json"
	"errors"
	"time"

	"bibliothek/repository"
)

// ActionRequest holds the parameters for the Omnibox dispatcher.
type ActionRequest struct {
	Query           string  `json:"query"`
	ActiveStudentID    *string `json:"active_student_id,omitempty"`
	ActiveTeacherID    *string `json:"active_teacher_id,omitempty"`
	ConfirmedChecklist bool    `json:"confirmed_checklist,omitempty"`
}

// ActionResponse is the polymorphic output payload returned by the Omnibox.
type ActionResponse struct {
	Type            string                 `json:"type"`                       // "student", "teacher", "ausleihe", "rueckgabe", "search_results"
	Student         *repository.Student    `json:"student,omitempty"`          // The active student, or original borrower
	Teacher         *repository.User       `json:"teacher,omitempty"`          // The active teacher borrower (Handapparat)
	Book            *repository.BookCopy   `json:"book,omitempty"`             // Book copy details if applicable
	Geraet          *repository.Geraet     `json:"geraet,omitempty"`           // Hardware details if applicable
	DueDate         *time.Time             `json:"due_date,omitempty"`         // Return deadline for check-outs
	LoanID          *string                `json:"loan_id,omitempty"`          // Loan UUID (for Undo support on returns)
	Fremdrueckgabe  bool                   `json:"fremdrueckgabe,omitempty"`   // Flag for returns from another student/teacher
	Vorbesitzer     *repository.Student    `json:"vorbesitzer,omitempty"`      // Original student borrower if foreign return
	VorbesitzerUser *repository.User       `json:"vorbesitzer_user,omitempty"` // Original teacher borrower if foreign return
	SearchResults   []repository.BookTitle `json:"search_results,omitempty"`   // Full-text search list
	HasVormerkung   bool                   `json:"has_vormerkung,omitempty"`   // True if returned book has a pending reservation
	VormerkungTitel string                 `json:"vormerkung_titel,omitempty"` // Title name of the reserved book
	VormerkungUser  string                 `json:"vormerkung_user,omitempty"`  // Reserved for: student name & class
}

// ActionEvent represents the data broadcasted to SSE clients on updates.
type ActionEvent struct {
	Event     string `json:"event"` // "ausleihe", "rueckgabe", "fremdrueckgabe"
	StudentID string `json:"student_id,omitempty"`
	TeacherID string `json:"teacher_id,omitempty"`
	CopyID    string `json:"copy_id,omitempty"`
	GeraetID  string `json:"geraet_id,omitempty"`
	BarcodeID string `json:"barcode_id"`
	Titel     string `json:"titel"`
	Timestamp int64  `json:"timestamp"`
}

var (
	errNotFound     = errors.New("Eintrag nicht gefunden")
	errBlocked      = errors.New("Ausleihe für diese/n Schüler/in ist gesperrt")
	errInvalidState = errors.New("Ungültiger Transaktionszustand")
)

// broadcastActionEvent broadcasts the action event as JSON via SSE stream.
func (s *Server) broadcastActionEvent(resp ActionResponse) {
	if resp.Book == nil {
		return
	}
	var studentID string
	if resp.Student != nil {
		studentID = resp.Student.ID
	}
	var teacherID string
	if resp.Teacher != nil {
		teacherID = resp.Teacher.ID
	}
	event := ActionEvent{
		Event:     resp.Type,
		StudentID: studentID,
		TeacherID: teacherID,
		CopyID:    resp.Book.ID,
		BarcodeID: resp.Book.BarcodeID,
		Titel:     resp.Book.Titel,
		Timestamp: time.Now().Unix(),
	}
	if resp.Fremdrueckgabe {
		event.Event = "fremdrueckgabe"
	}

	data, err := json.Marshal(event)
	if err == nil {
		s.Broker.Broadcast("action", string(data))
	}
}
