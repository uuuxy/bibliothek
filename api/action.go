package api

import (
	"encoding/json"
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

		if req.IdempotencyKey != "" {
			var cachedRespJSON []byte
			var cachedStatus int
			err := s.DB.Pool.QueryRow(ctx, "SELECT response_data, status_code FROM idempotency_keys WHERE idempotency_key = $1", req.IdempotencyKey).Scan(&cachedRespJSON, &cachedStatus)
			if err == nil {
				// Cache Hit
				if cachedStatus >= 400 {
					var errData map[string]string
					json.Unmarshal(cachedRespJSON, &errData)
					apierrors.SendHTTPError(w, cachedStatus, errors.New(errData["error"]))
					return
				}
				var cachedResp ActionResponse
				json.Unmarshal(cachedRespJSON, &cachedResp)
				RespondJSON(w, cachedStatus, cachedResp)
				return
			}
		}

		res, err := omniboxSvc.ProcessQuery(ctx, req.Query, req.ActiveStudentID, req.ActiveTeacherID, req.ConfirmedChecklist, claims.UserID, string(claims.Rolle), req.OverrideBlock)
		
		status := http.StatusOK
		if err != nil {
			status = http.StatusInternalServerError
			switch {
			case errors.Is(err, service.ErrNotFound):
				status = http.StatusNotFound
			case errors.Is(err, service.ErrBlocked):
				status = http.StatusForbidden
			case errors.Is(err, service.ErrInvalidState):
				status = http.StatusBadRequest
			case errors.Is(err, service.ErrConflict):
				status = http.StatusConflict
			}
			
			if req.IdempotencyKey != "" {
				errData, _ := json.Marshal(map[string]string{"error": err.Error()})
				_, _ = s.DB.Pool.Exec(ctx, "INSERT INTO idempotency_keys (idempotency_key, response_data, status_code) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", req.IdempotencyKey, errData, status)
			}
			
			apierrors.SendHTTPError(w, status, err)
			return
		}

		// Map to API response
		resp := mapOmniboxResultToActionResponse(res)

		if req.IdempotencyKey != "" {
			respData, _ := json.Marshal(resp)
			_, _ = s.DB.Pool.Exec(ctx, "INSERT INTO idempotency_keys (idempotency_key, response_data, status_code) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", req.IdempotencyKey, respData, status)
		}

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

			if req.IdempotencyKey != "" {
				var cachedRespJSON []byte
				var cachedStatus int
				err := s.DB.Pool.QueryRow(ctx, "SELECT response_data, status_code FROM idempotency_keys WHERE idempotency_key = $1", req.IdempotencyKey).Scan(&cachedRespJSON, &cachedStatus)
				if err == nil {
					item := ActionBatchResponseItem{
						Index:   i,
						Status:  cachedStatus,
						Success: cachedStatus >= 200 && cachedStatus < 300,
					}
					if item.Success {
						var data ActionResponse
						json.Unmarshal(cachedRespJSON, &data)
						item.Data = &data
					} else {
						var errData map[string]string
						json.Unmarshal(cachedRespJSON, &errData)
						item.Error = errData["error"]
					}
					batchResp.Results = append(batchResp.Results, item)
					continue
				}
			}

			res, err := omniboxSvc.ProcessQuery(ctx, req.Query, req.ActiveStudentID, req.ActiveTeacherID, req.ConfirmedChecklist, claims.UserID, string(claims.Rolle), req.OverrideBlock)

			status := http.StatusOK
			if err != nil {
				status = http.StatusInternalServerError
				switch {
				case errors.Is(err, service.ErrNotFound):
					status = http.StatusNotFound
				case errors.Is(err, service.ErrBlocked):
					status = http.StatusForbidden
				case errors.Is(err, service.ErrInvalidState):
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
				if req.IdempotencyKey != "" {
					errData, _ := json.Marshal(map[string]string{"error": err.Error()})
					_, _ = s.DB.Pool.Exec(ctx, "INSERT INTO idempotency_keys (idempotency_key, response_data, status_code) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", req.IdempotencyKey, errData, status)
				}
			} else {
				item.Data = mapOmniboxResultToActionResponse(res)
				if req.IdempotencyKey != "" {
					respData, _ := json.Marshal(item.Data)
					_, _ = s.DB.Pool.Exec(ctx, "INSERT INTO idempotency_keys (idempotency_key, response_data, status_code) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", req.IdempotencyKey, respData, status)
				}
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
