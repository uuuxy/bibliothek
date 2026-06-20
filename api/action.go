package api

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/service"
)

// ActionHandler dispatches requests from the Omnibox.
func (s *Server) ActionHandler(omniboxSvc service.OmniboxService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("Sitzungs-Information fehlt oder ist abgelaufen"))
			return
		}

		var req ActionRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		req.Query = strings.TrimSpace(req.Query)
		if req.Query == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("such- oder Barcode-Abfrage ist leer"))
			return
		}

		ctx := r.Context()

		res, err := omniboxSvc.ProcessQuery(ctx, req.Query, req.ActiveStudentID, req.ActiveTeacherID, req.ConfirmedChecklist, claims.UserID, string(claims.Rolle))
		if err != nil {
			status := http.StatusInternalServerError
			switch {
			case errors.Is(err, service.ErrNotFound):
				status = http.StatusNotFound
			case errors.Is(err, service.ErrBlocked), errors.Is(err, service.ErrInvalidState):
				status = http.StatusBadRequest
			case errors.Is(err, service.ErrConflict):
				status = http.StatusConflict
			}
			apierrors.SendHTTPError(w, status, err)
			return
		}

		// Map to API response
		resp := mapOmniboxResultToActionResponse(res)

		// Broadcast updates to all monitoring dashboards (SSE)
		if resp.Type == "ausleihe" || resp.Type == "rueckgabe" {
			s.broadcastActionEvent(*resp)
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

func mapOmniboxResultToActionResponse(res *service.OmniboxResult) *ActionResponse {
	if res == nil {
		return nil
	}
	return &ActionResponse{
		Type:            res.Type,
		Message:         res.Message,
		Student:         res.Student,
		Teacher:         res.Teacher,
		Book:            res.Book,
		Geraet:          res.Geraet,
		DueDate:         res.DueDate,
		LoanID:          res.LoanID,
		Fremdrueckgabe:  res.Fremdrueckgabe,
		Vorbesitzer:     res.Vorbesitzer,
		VorbesitzerUser: res.VorbesitzerUser,
		SearchResults:   res.SearchResults,
		HasVormerkung:   res.HasVormerkung,
		VormerkungTitel: res.VormerkungTitel,
		VormerkungUser:  res.VormerkungUser,
	}
}

// ActionBatchHandler processes a batch of Omnibox requests.
func (s *Server) ActionBatchHandler(omniboxSvc service.OmniboxService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("Sitzungs-Information fehlt oder ist abgelaufen"))
			return
		}

		var batchReq ActionBatchRequest
		if !DecodeAndValidate(w, r, &batchReq) {
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

			res, err := omniboxSvc.ProcessQuery(ctx, req.Query, req.ActiveStudentID, req.ActiveTeacherID, req.ConfirmedChecklist, claims.UserID, string(claims.Rolle))

			status := http.StatusOK
			if err != nil {
				status = http.StatusInternalServerError
				switch {
				case errors.Is(err, service.ErrNotFound):
					status = http.StatusNotFound
				case errors.Is(err, service.ErrBlocked), errors.Is(err, service.ErrInvalidState):
					status = http.StatusBadRequest
				case errors.Is(err, service.ErrConflict):
					status = http.StatusConflict
				}
			}

			item := ActionBatchResponseItem{
				Index:   i,
				Status:  status,
				Success: err == nil,
			}
			if err != nil {
				item.Error = err.Error()
			} else {
				item.Data = mapOmniboxResultToActionResponse(res)
				// Broadcast updates
				if item.Data.Type == "ausleihe" || item.Data.Type == "rueckgabe" {
					s.broadcastActionEvent(*item.Data)
				}
			}
			batchResp.Results = append(batchResp.Results, item)
		}

		RespondJSON(w, http.StatusOK, batchResp)
	}
}
