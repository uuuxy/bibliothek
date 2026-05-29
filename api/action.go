package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// ActionHandler dispatches requests from the Omnibox.
// Routes prefixes ('S-', 'B-') and queries database indexes using the Repository Pattern.
func (s *Server) ActionHandler(
	studentRepo repository.StudentRepository,
	bookRepo repository.BookRepository,
	loanRepo repository.LoanRepository,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("missing session information"))
			return
		}

		var req ActionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		req.Query = strings.TrimSpace(req.Query)
		if req.Query == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("search/barcode query is empty"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var resp ActionResponse
		var err error

		// Route based on command pattern prefixes
		if strings.HasPrefix(req.Query, "S-") {
			err = s.handleStudentAction(ctx, req.Query, studentRepo, &resp)
		} else if strings.HasPrefix(req.Query, "L-") {
			err = s.handleTeacherAction(ctx, req.Query, &resp)
		} else if strings.HasPrefix(req.Query, "B-") {
			err = s.handleBookAction(ctx, req.Query, claims, req.ActiveStudentID, req.ActiveTeacherID, studentRepo, bookRepo, loanRepo, &resp)
		} else {
			err = s.handleSearchAction(ctx, req.Query, bookRepo, &resp)
		}

		if err != nil {
			status := http.StatusInternalServerError
			switch {
			case errors.Is(err, errNotFound):
				status = http.StatusNotFound
			case errors.Is(err, errBlocked), errors.Is(err, errInvalidState):
				status = http.StatusBadRequest
			}
			apierrors.SendHTTPError(w, status, err)
			return
		}

		// Broadcast updates to all monitoring dashboards (SSE)
		if resp.Type == "ausleihe" || resp.Type == "rueckgabe" {
			s.broadcastActionEvent(resp)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// handleBookAction processes transactions like checkouts, returns, and foreign transfers.
func (s *Server) handleBookAction(
	ctx context.Context,
	query string,
	claims *auth.Claims,
	activeStudentID *string,
	activeTeacherID *string,
	studentRepo repository.StudentRepository,
	bookRepo repository.BookRepository,
	loanRepo repository.LoanRepository,
	resp *ActionResponse,
) error {
	staffID := claims.UserID
	// 1. Resolve physical book item
	copy, err := bookRepo.GetCopyByBarcode(ctx, query)
	if err != nil {
		return err
	}
	if copy == nil {
		return fmt.Errorf("%w: book copy barcode %s not found", errNotFound, query)
	}

	// 2. Fetch active loan
	activeLoan, err := loanRepo.GetActiveLoanByCopyID(ctx, copy.ID)
	if err != nil {
		return err
	}

	// Case: Active Teacher context exists in session (dauerhafter Handapparat)
	if activeTeacherID != nil && *activeTeacherID != "" {
		return s.handleTeacherCheckoutFlow(ctx, copy, activeLoan, *activeTeacherID, staffID, studentRepo, loanRepo, resp)
	}

	// Case: Active Student context exists in session
	if activeStudentID != nil && *activeStudentID != "" {
		return s.handleStudentCheckoutFlow(ctx, copy, activeLoan, *activeStudentID, staffID, studentRepo, loanRepo, resp)
	}

	// Case: NO active student/teacher context exists -> Simple Return
	if activeLoan == nil {
		if claims.Rolle == auth.RoleLehrer {
			dueTime := time.Now().AddDate(1, 0, 0) // 1 year
			loan, err := loanRepo.CreateUserLoan(ctx, copy.ID, claims.UserID, claims.UserID, dueTime, true)
			if err != nil {
				return err
			}
			resp.Type = "ausleihe"
			resp.Book = copy
			resp.DueDate = &loan.RueckgabeFrist
			return nil
		}
		return fmt.Errorf("%w: book copy is not currently borrowed", errInvalidState)
	}

	if activeLoan.AusleiherBenutzerID != nil && *activeLoan.AusleiherBenutzerID == claims.UserID {
		err = loanRepo.ReturnLoan(ctx, activeLoan.ID, claims.UserID, false)
		if err != nil {
			return err
		}
		resp.Type = "rueckgabe"
		resp.Book = copy
		return nil
	}

	var borrowerStudent *repository.Student
	if activeLoan.SchuelerID != nil {
		borrowerStudent, err = studentRepo.GetByID(ctx, *activeLoan.SchuelerID)
		if err != nil {
			return err
		}
	}

	err = loanRepo.ReturnLoan(ctx, activeLoan.ID, staffID, false)
	if err != nil {
		return err
	}

	resp.Type = "rueckgabe"
	resp.Book = copy
	resp.Student = borrowerStudent
	return nil
}
