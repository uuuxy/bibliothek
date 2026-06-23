package api

import (
	"encoding/json"
	"errors"
	"log"
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
					if uerr := json.Unmarshal(cachedRespJSON, &errData); uerr != nil {
						log.Printf("idempotenz: beschädigte Fehler-Antwort im Cache, wird neu berechnet: %v", uerr)
					} else {
						apierrors.SendHTTPError(w, cachedStatus, errors.New(errData["error"]))
						return
					}
				} else {
					var cachedResp ActionResponse
					if uerr := json.Unmarshal(cachedRespJSON, &cachedResp); uerr != nil {
						log.Printf("idempotenz: beschädigte Antwort im Cache, wird neu berechnet: %v", uerr)
					} else {
						RespondJSON(w, cachedStatus, cachedResp)
						return
					}
				}
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
				if errData, merr := json.Marshal(map[string]string{"error": err.Error()}); merr == nil {
					logExec(s.DB.Pool.Exec(ctx, "INSERT INTO idempotency_keys (idempotency_key, response_data, status_code) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", req.IdempotencyKey, errData, status))
				} else {
					log.Printf("idempotenz: Fehler-Antwort konnte nicht serialisiert werden: %v", merr)
				}
			}

			apierrors.SendHTTPError(w, status, err)
			return
		}

		// Map to API response
		resp := mapOmniboxResultToActionResponse(res)

		if req.IdempotencyKey != "" {
			if respData, merr := json.Marshal(resp); merr == nil {
				logExec(s.DB.Pool.Exec(ctx, "INSERT INTO idempotency_keys (idempotency_key, response_data, status_code) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", req.IdempotencyKey, respData, status))
			} else {
				log.Printf("idempotenz: Antwort konnte nicht serialisiert werden: %v", merr)
			}
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
					decodeOK := true
					if item.Success {
						var data ActionResponse
						if uerr := json.Unmarshal(cachedRespJSON, &data); uerr != nil {
							log.Printf("idempotenz: beschädigte Batch-Antwort im Cache, wird neu berechnet: %v", uerr)
							decodeOK = false
						} else {
							item.Data = &data
						}
					} else {
						var errData map[string]string
						if uerr := json.Unmarshal(cachedRespJSON, &errData); uerr != nil {
							log.Printf("idempotenz: beschädigte Batch-Fehler-Antwort im Cache, wird neu berechnet: %v", uerr)
							decodeOK = false
						} else {
							item.Error = errData["error"]
						}
					}
					if decodeOK {
						batchResp.Results = append(batchResp.Results, item)
						continue
					}
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
					if errData, merr := json.Marshal(map[string]string{"error": err.Error()}); merr == nil {
						logExec(s.DB.Pool.Exec(ctx, "INSERT INTO idempotency_keys (idempotency_key, response_data, status_code) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", req.IdempotencyKey, errData, status))
					} else {
						log.Printf("idempotenz: Batch-Fehler-Antwort konnte nicht serialisiert werden: %v", merr)
					}
				}
			} else {
				item.Data = mapOmniboxResultToActionResponse(res)
				if req.IdempotencyKey != "" {
					if respData, merr := json.Marshal(item.Data); merr == nil {
						logExec(s.DB.Pool.Exec(ctx, "INSERT INTO idempotency_keys (idempotency_key, response_data, status_code) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", req.IdempotencyKey, respData, status))
					} else {
						log.Printf("idempotenz: Batch-Antwort konnte nicht serialisiert werden: %v", merr)
					}
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
