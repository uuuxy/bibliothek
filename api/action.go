package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/service"
	"bibliothek/repository"
)

// ActionHandler dispatches requests from the Omnibox.
// Routes prefixes ('S-', 'B-') and queries database indexes using the Repository Pattern.
func (s *Server) ActionHandler(
	studentRepo repository.StudentRepository,
	bookRepo repository.BookRepository,
	loanRepo repository.LoanRepository,
	loanSvc service.LoanService,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("Sitzungs-Information fehlt oder ist abgelaufen"))
			return
		}

		var req ActionRequest
		if !DecodeJSON(w, r, &req) {
			return
		}

		req.Query = strings.TrimSpace(req.Query)
		if req.Query == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("such- oder Barcode-Abfrage ist leer"))
			return
		}

		ctx := r.Context()

		resp, status, err := s.processActionRequest(ctx, req, claims, studentRepo, bookRepo, loanRepo, loanSvc)
		if err != nil {
			apierrors.SendHTTPError(w, status, err)
			return
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

// processActionRequest contains the core routing logic for an individual action request.
func (s *Server) processActionRequest(
	ctx context.Context,
	req ActionRequest,
	claims *auth.Claims,
	studentRepo repository.StudentRepository,
	bookRepo repository.BookRepository,
	loanRepo repository.LoanRepository,
	loanSvc service.LoanService,
) (*ActionResponse, int, error) {
	var resp ActionResponse
	var err error

	// Route based on command pattern prefixes
	if strings.HasPrefix(req.Query, "S-") {
		err = s.handleStudentAction(ctx, req.Query, studentRepo, &resp)
	} else if strings.HasPrefix(req.Query, "L-") {
		err = s.handleTeacherAction(ctx, req.Query, &resp)
	} else if strings.HasPrefix(req.Query, "B-") {
		err = s.handleBookAction(ctx, req.Query, claims, req.ActiveStudentID, req.ActiveTeacherID, bookRepo, loanRepo, loanSvc, &resp)
	} else if strings.HasPrefix(req.Query, "G-") {
		err = s.handleGeraetAction(ctx, req.Query, claims, req.ActiveStudentID, req.ActiveTeacherID, req.ConfirmedChecklist, studentRepo, loanRepo, &resp)
	} else {
		// Fallback for Littera barcodes (which lack the "B-" prefix).
		if copy, _ := bookRepo.GetCopyByBarcode(ctx, req.Query); copy != nil {
			err = s.handleBookAction(ctx, req.Query, claims, req.ActiveStudentID, req.ActiveTeacherID, bookRepo, loanRepo, loanSvc, &resp)
		} else {
			err = s.handleSearchAction(ctx, req.Query, bookRepo, &resp)
		}
	}

	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, errNotFound):
			status = http.StatusNotFound
		case errors.Is(err, errBlocked), errors.Is(err, errInvalidState):
			status = http.StatusBadRequest
		case errors.Is(err, errConflict):
			status = http.StatusConflict
		}
		return nil, status, err
	}

	// Broadcast updates to all monitoring dashboards (SSE)
	if resp.Type == "ausleihe" || resp.Type == "rueckgabe" {
		s.broadcastActionEvent(resp)
	}

	return &resp, http.StatusOK, nil
}

// ActionBatchHandler processes a batch of Omnibox requests.
func (s *Server) ActionBatchHandler(
	studentRepo repository.StudentRepository,
	bookRepo repository.BookRepository,
	loanRepo repository.LoanRepository,
	loanSvc service.LoanService,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("Sitzungs-Information fehlt oder ist abgelaufen"))
			return
		}

		var batchReq ActionBatchRequest
		if !DecodeJSON(w, r, &batchReq) {
			return
		}

		ctx := r.Context()

		var batchResp ActionBatchResponse

		for i, req := range batchReq {
			req.Query = strings.TrimSpace(req.Query)
			if req.Query == "" {
				batchResp.Results = append(batchResp.Results, ActionBatchResponseItem{
					Index:   i,
					Success: false,
					Status:  http.StatusBadRequest,
					Error:   "Query ist leer",
				})
				continue
			}

			resp, status, err := s.processActionRequest(ctx, req, claims, studentRepo, bookRepo, loanRepo, loanSvc)

			item := ActionBatchResponseItem{
				Index:   i,
				Status:  status,
				Success: err == nil,
			}
			if err != nil {
				item.Error = err.Error()
			} else {
				item.Data = resp
			}
			batchResp.Results = append(batchResp.Results, item)
		}

		RespondJSON(w, http.StatusOK, batchResp)
	}
}

// handleBookAction processes transactions like checkouts, returns, and foreign transfers.
// The initial book-copy and loan lookups are cheap read operations (no lock needed yet).
// The actual mutation is performed inside a Read Committed transaction with row-level locks
// in the respective flow handlers.
func (s *Server) handleBookAction(
	ctx context.Context,
	query string,
	claims *auth.Claims,
	activeStudentID *string,
	activeTeacherID *string,

	bookRepo repository.BookRepository,
	loanRepo repository.LoanRepository,
	loanSvc service.LoanService,
	resp *ActionResponse,
) error {
	staffID := claims.UserID
	// 1. Resolve physical book item (read-only; no lock required at this stage)
	copy, err := bookRepo.GetCopyByBarcode(ctx, query)
	if err != nil {
		return err
	}
	if copy == nil {
		return fmt.Errorf("%w: Buchexemplar-Barcode %s wurde nicht gefunden", errNotFound, query)
	}
	if !copy.IstAusleihbar || copy.IstAusgesondert {
		// Recovery check: if the book is not currently loaned out and not reserved, reactivate it!
		activeLoan, err := loanRepo.GetActiveLoanByCopyID(ctx, copy.ID)
		if err != nil {
			return err
		}

		isReserved := strings.HasPrefix(copy.ZustandNotiz, "Reserviert für:")
		reservedForThisStudent := false

		if isReserved && activeStudentID != nil && *activeStudentID != "" {
			// Check if this student is the one it's reserved for
			v, checkErr := s.checkVormerkung(ctx, copy.TitelID)
			if checkErr == nil && v != nil && v.SchuelerID == *activeStudentID {
				reservedForThisStudent = true
			}
		}

		if activeLoan == nil && (!isReserved || reservedForThisStudent) {
			_, err = s.DB.Pool.Exec(ctx, "UPDATE buecher_exemplare SET ist_ausleihbar = true, ist_ausgesondert = false, zustand_notiz = '' WHERE id = $1", copy.ID)
			if err != nil {
				return err
			}
			copy.IstAusleihbar = true
			copy.ZustandNotiz = ""

			// If it was reserved for this student, do NOT return an info response, just continue to checkout
			if !reservedForThisStudent {
				resp.Type = "info"
				resp.Message = "Buch reaktiviert"
				return nil
			}
		} else if isReserved && !reservedForThisStudent {
			return fmt.Errorf("%w: Dieses Buchexemplar ist %s", errBlocked, copy.ZustandNotiz)
		} else if copy.IstAusgesondert {
			return fmt.Errorf("%w: Buchexemplar %s ist ausgesondert und kann nicht ausgeliehen werden", errInvalidState, query)
		} else {
			return fmt.Errorf("%w: Buchexemplar ist nicht ausleihbar", errInvalidState)
		}
	}

	// Case: Active Teacher or Student context exists in session.
	if (activeTeacherID != nil && *activeTeacherID != "") || (activeStudentID != nil && *activeStudentID != "") {
		lr, err := loanSvc.HandleUnifiedCheckout(ctx, copy, activeStudentID, activeTeacherID, staffID)
		if err != nil {
			return err
		}
		resp.Type = lr.Type
		resp.Book = lr.Book
		resp.Student = lr.Student
		resp.Teacher = lr.Teacher
		resp.DueDate = lr.DueDate
		resp.LoanID = lr.LoanID
		resp.Fremdrueckgabe = lr.Fremdrueckgabe
		resp.Vorbesitzer = lr.Vorbesitzer
		resp.VorbesitzerUser = lr.VorbesitzerUser
		resp.HasVormerkung = lr.HasVormerkung
		resp.VormerkungTitel = lr.VormerkungTitel
		resp.VormerkungUser = lr.VormerkungUser
		return nil
	}

	// Case: NO active student/teacher context -> Simple Return (or teacher self-checkout).
	lr, err := loanSvc.HandleSimpleReturn(ctx, copy, staffID, string(claims.Rolle))
	if err != nil {
		return err
	}
	resp.Type = lr.Type
	resp.Book = lr.Book
	resp.Student = lr.Student
	resp.Teacher = lr.Teacher
	resp.DueDate = lr.DueDate
	resp.LoanID = lr.LoanID
	resp.Fremdrueckgabe = lr.Fremdrueckgabe
	resp.Vorbesitzer = lr.Vorbesitzer
	resp.VorbesitzerUser = lr.VorbesitzerUser
	resp.HasVormerkung = lr.HasVormerkung
	resp.VormerkungTitel = lr.VormerkungTitel
	resp.VormerkungUser = lr.VormerkungUser
	return nil
}
