package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/service"
	"bibliothek/repository"
)

// Merkmal einer Sperre, die die Theke übergehen darf (service.IstUebergehbareSperre): Am
// Header öffnet das Frontend den Override-Dialog, das Cache-Feld trägt es durch eine
// Wiederholung mit demselben Idempotenz-Schlüssel (sperr_merkmal_test.go).
const (
	sperrKopf        = "X-Sperre"
	sperrUebergehbar = "uebergehbar"
	sperrCacheFeld   = "sperre"
)

// ohneSperrgrund nimmt einem Sperr-Fehler den Freitext (block_reason), wenn der
// Aufrufer ihn nicht sehen darf. Die Theke erfährt weiterhin DASS gesperrt ist
// (403, generische Meldung) — aber nicht das WARUM: Der Grund kann Zahlungs-
// rückstände oder Familieninterna nennen (PII-Matrix Stufe 2, Fund vom
// 18.08.2026 am Live-Pfad mit Helfer-Konto). Dieselbe Grenze wie die
// Geräteliste: view_students entscheidet, nicht die Route.
func ohneSperrgrund(err error, darfGrundSehen bool) error {
	var sg *service.SperrGrundFehler
	if err == nil || darfGrundSehen || !errors.As(err, &sg) {
		return err
	}
	return fmt.Errorf("%w — bitte an die Bibliotheksleitung wenden", sg.Kern)
}

func mapServiceErrorToStatus(err error) int {
	switch {
	case errors.Is(err, service.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, service.ErrBlocked):
		return http.StatusForbidden
	case errors.Is(err, service.ErrInvalidState):
		return http.StatusBadRequest
	case errors.Is(err, service.ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// idempotenzLage ist das Ergebnis von erlangeIdempotenz: genau eines der drei gilt, oder
// keines (kein Schlüssel, oder die Datenbank hat die Reservierung verweigert — dann läuft
// die Anfrage ohne Schutz, wie vor dem 15.09.2026).
type idempotenzLage struct {
	reserviert bool                          // der Schlüssel gehört dieser Anfrage
	antwort    *repository.IdempotenzAntwort // eine frühere Anfrage hat schon geantwortet
	inArbeit   bool                          // eine frühere Anfrage arbeitet noch, Warten hat nicht gereicht
}

// idempotenzWartezeit: wie lange eine Anfrage auf die Antwort einer laufenden mit demselben
// Schlüssel wartet, bevor sie „in Arbeit" meldet. Tests setzen IdempotenzWartezeit.
func (s *Server) idempotenzWartezeit() time.Duration {
	if s.IdempotenzWartezeit > 0 {
		return s.IdempotenzWartezeit
	}
	return 3 * time.Second
}

// erlangeIdempotenz reserviert den Schlüssel VOR der Arbeit (Commit 7, 15.09.2026). Bis dahin
// wurde die Antwort erst nach der Arbeit eingefügt: Eine zweite Anfrage mit demselben
// Schlüssel, die nach dem Commit der ersten und vor dem Speichern ihrer Antwort eintraf,
// fand nichts und buchte neu — das Buch lag schon beim Kind, also wurde es zurückgenommen
// (idempotenz_pg_test.go). Läuft die Arbeit noch, wartet die zweite Anfrage auf die Antwort;
// ist die Reservierung nach einem Serverfehler freigegeben worden, übernimmt sie den Schlüssel.
func (s *Server) erlangeIdempotenz(ctx context.Context, key string) idempotenzLage {
	if key == "" {
		return idempotenzLage{}
	}
	frist := time.Now().Add(s.idempotenzWartezeit())
	for {
		reserviert, antwort, err := repository.ReserviereIdempotenzSchluessel(ctx, s.DB.Pool, key)
		if err != nil {
			log.Printf("idempotenz: Schlüssel nicht reservierbar, Anfrage läuft ohne Schutz: %v", err)
			return idempotenzLage{}
		}
		if reserviert {
			return idempotenzLage{reserviert: true}
		}
		if !antwort.InArbeit() {
			return idempotenzLage{antwort: antwort}
		}
		if ctx.Err() != nil || time.Now().After(frist) {
			return idempotenzLage{inArbeit: true}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// idempotenzSpeicherfrist: wie lange saveToCache nach der Arbeit noch schreiben darf — mit
// WithoutCancel, also auch nach Ablauf der Bearbeitungsfrist. Bearbeitungsfrist plus diese
// Frist müssen unter repository.IdempotenzReservierungsfrist bleiben, sonst kann die
// Waisenübernahme einen Besitzer treffen, der noch lebt (idempotenz_fristen_test.go).
const idempotenzSpeicherfrist = 5 * time.Second

// saveToCache schreibt die Antwort in die Reservierung — mit WithoutCancel (Vorbild
// mahnwesen_bulk_mail.go): Bricht der Aufrufer ab, nachdem die Buchung committet ist, muss
// die Antwort trotzdem stehen, sonst bucht die Wiederholung neu. Ein Serverfehler wird nicht
// gespeichert, sondern gibt den Schlüssel frei: Die Wiederholung soll es neu versuchen.
func (s *Server) saveToCache(ctx context.Context, key string, data interface{}, status int) {
	if key == "" {
		return
	}
	ctx, abbruch := context.WithTimeout(context.WithoutCancel(ctx), idempotenzSpeicherfrist)
	defer abbruch()
	if status >= 500 {
		if _, err := repository.GibIdempotenzSchluesselFrei(ctx, s.DB.Pool, key); err != nil {
			log.Printf("idempotenz: Schlüssel nach Serverfehler nicht freigegeben: %v", err)
		}
		return
	}
	respData, err := json.Marshal(data)
	if err != nil {
		log.Printf("idempotenz: Antwort konnte nicht serialisiert werden: %v", err)
		return
	}
	if err := repository.SpeichereIdempotenzAntwort(ctx, s.DB.Pool, key, respData, status); err != nil {
		log.Printf("idempotenz: Antwort nicht gespeichert: %v", err)
	}
}

// serveCachedActionResponse liefert die gespeicherte Antwort aus, falls unversehrt.
// true = Antwort wurde gesendet (Aufrufer beendet); false = beschädigter Eintrag, Aufrufer
// berechnet neu.
func (s *Server) serveCachedActionResponse(w http.ResponseWriter, antwort *repository.IdempotenzAntwort) bool {
	cachedRespJSON, cachedStatus := antwort.Daten, antwort.Status

	if cachedStatus >= 400 {
		var errData map[string]string
		if uerr := json.Unmarshal(cachedRespJSON, &errData); uerr != nil {
			log.Printf("idempotenz: beschädigte Fehler-Antwort im Cache, wird neu berechnet: %v", uerr)
			return false
		}
		if errData[sperrCacheFeld] == sperrUebergehbar {
			w.Header().Set(sperrKopf, sperrUebergehbar)
		}
		apierrors.SendHTTPError(w, cachedStatus, errors.New(errData["error"]))
		return true
	}

	var cachedResp ActionResponse
	if uerr := json.Unmarshal(cachedRespJSON, &cachedResp); uerr != nil {
		log.Printf("idempotenz: beschädigte Antwort im Cache, wird neu berechnet: %v", uerr)
		return false
	}
	RespondJSON(w, cachedStatus, cachedResp)
	return true
}

// cachedBatchItem liefert das gespeicherte Batch-Ergebnis zu einer Antwort. ok=false:
// beschädigter Eintrag — der Aufrufer berechnet neu.
func cachedBatchItem(antwort *repository.IdempotenzAntwort, index int) (ActionBatchResponseItem, bool) {
	cachedRespJSON, cachedStatus := antwort.Daten, antwort.Status

	item := ActionBatchResponseItem{
		Index:   index,
		Status:  cachedStatus,
		Success: cachedStatus >= 200 && cachedStatus < 300,
	}
	if item.Success {
		var data ActionResponse
		if uerr := json.Unmarshal(cachedRespJSON, &data); uerr != nil {
			log.Printf("idempotenz: beschädigte Batch-Antwort im Cache, wird neu berechnet: %v", uerr)
			return ActionBatchResponseItem{}, false
		}
		item.Data = &data
	} else {
		var errData map[string]string
		if uerr := json.Unmarshal(cachedRespJSON, &errData); uerr != nil {
			log.Printf("idempotenz: beschädigte Batch-Fehler-Antwort im Cache, wird neu berechnet: %v", uerr)
			return ActionBatchResponseItem{}, false
		}
		item.Error = errData["error"]
	}
	return item, true
}

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

		lage := s.erlangeIdempotenz(ctx, req.IdempotencyKey)
		if lage.antwort != nil && s.serveCachedActionResponse(w, lage.antwort) {
			return
		}
		if lage.inArbeit {
			apierrors.SendHTTPError(w, http.StatusConflict,
				errors.New("in_arbeit: dieser Scan wird gerade gebucht — bitte einen Moment warten und erneut scannen"))
			return
		}

		// Eine Sperre aufheben ist ein Verwaltungsakt, kein Theken-Vorgang:
		// override_block wirkt nur mit edit_students — demselben Recht, das auch
		// das Sperren/Entsperren erlaubt. Ohne es bleibt die Sperre bestehen; wer
		// das Feld trotzdem setzt, wird behandelt, als hätte er es nicht gesetzt.
		// Am 18.08.2026 live gefunden: perform_actions allein reichte, ein Helfer
		// konnte jede Sperre per Request-Feld aushebeln — die UI bot den Schalter
		// nie an, der Schutz war also nur Konvention (bewertung-Muster F1/F4).
		res, err := omniboxSvc.ProcessQuery(ctx, service.OmniboxQuery{
			Query:              req.Query,
			ActiveLeserID:      req.ActiveLeserID,
			ConfirmedChecklist: req.ConfirmedChecklist,
			StaffID:            claims.UserID,
			StaffRole:          string(claims.Rolle),
			OverrideBlock:      req.OverrideBlock && s.BesitztRecht(r, "edit_students"),
		})

		if err != nil {
			// VOR dem Cachen kürzen — sonst läge der Freitext im Idempotenz-Cache.
			err = ohneSperrgrund(err, s.BesitztRecht(r, "view_students"))
			status := mapServiceErrorToStatus(err)
			cacheDaten := map[string]string{"error": err.Error()}
			if service.IstUebergehbareSperre(err) {
				// Merkmal für den Override-Dialog der Theke — im Header, damit der Body die
				// eine kanonische Fehlerform behält, und im Cache, damit eine Wiederholung
				// es nicht verliert (sperr_merkmal_test.go).
				w.Header().Set(sperrKopf, sperrUebergehbar)
				cacheDaten[sperrCacheFeld] = sperrUebergehbar
			}
			s.saveToCache(ctx, req.IdempotencyKey, cacheDaten, status)
			apierrors.SendHTTPError(w, status, err)
			return
		}

		// Map to API response
		resp := mapOmniboxResultToActionResponse(res)

		s.saveToCache(ctx, req.IdempotencyKey, resp, http.StatusOK)

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
		Type:                 res.Type,
		Message:              res.Message,
		Student:              zumKioskSchueler(res.Student),
		Book:                 res.Book,
		Geraet:               res.Geraet,
		DueDate:              res.DueDate,
		LoanID:               res.LoanID,
		Fremdrueckgabe:       res.Fremdrueckgabe,
		Vorbesitzer:          zumKioskSchueler(res.Vorbesitzer),
		SearchResults:        res.SearchResults,
		HasVormerkung:        res.HasVormerkung,
		VormerkungTitel:      res.VormerkungTitel,
		VormerkungUser:       res.VormerkungUser,
		RegalfreigabeBarcode: res.RegalfreigabeBarcode,
		AufsichtInformieren:  res.AufsichtInformieren,
		Abholbereit:          zuAbholbereitInfos(res.Abholbereit),
	}
}

// zuAbholbereitInfos überträgt den Abholfach-Hinweis in die API-Form.
func zuAbholbereitInfos(vs []service.AbholbereiteVormerkung) []AbholbereitInfo {
	if len(vs) == 0 {
		return nil
	}
	out := make([]AbholbereitInfo, 0, len(vs))
	for _, v := range vs {
		out = append(out, AbholbereitInfo{Titel: v.Titel, BereitgestelltBis: v.BereitgestelltBis})
	}
	return out
}

// ActionBatchHandler processes a batch of Omnibox requests.
func (s *Server) ActionBatchHandler(omniboxSvc service.OmniboxService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("Sitzungs-Information fehlt oder ist abgelaufen"))
			return
		}

		// Nur dekodieren, NICHT Struct-validieren: ActionBatchRequest ist ein SLICE,
		// und go-playground/validator kann nur Structs — Validate.Struct darauf machte
		// bis zum 01.09.2026 JEDEN wohlgeformten Batch-Body zum 400 („validator: …").
		// Das Frontend schickt ein nacktes Array (offlineSync.svelte.js); der
		// Offline-Sync der Theke scheiterte damit still bei jedem Anlauf. Die
		// Pflichtfeld-Prüfung sitzt je Eintrag in processSingleBatchItem (leerer
		// Query → Fehler-Item statt Stapel-Abbruch).
		var batchReq ActionBatchRequest
		if err := json.NewDecoder(r.Body).Decode(&batchReq); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		ctx := r.Context()
		var batchResp ActionBatchResponse

		stapel := batchKontext{
			omnibox:        omniboxSvc,
			userID:         claims.UserID,
			rolle:          string(claims.Rolle),
			darfGrundSehen: s.BesitztRecht(r, "view_students"),
			darfOverride:   s.BesitztRecht(r, "edit_students"),
		}
		for i, req := range batchReq {
			batchResp.Results = append(batchResp.Results, s.processSingleBatchItem(ctx, stapel, req, i))
		}

		RespondJSON(w, http.StatusOK, batchResp)
	}
}

// batchKontext bündelt, was für ALLE Einträge eines Stapels gleich ist: der Dienst,
// das anfragende Konto und dessen Rechte. Ohne das Bündel reicht der Handler acht
// Einzelwerte durch die Schleife (go:S107) — und die beiden Rechte-Flags stehen als
// zwei gleichartige bool nebeneinander, wo ein Dreher niemandem auffiele.
type batchKontext struct {
	omnibox        service.OmniboxService
	userID         string
	rolle          string
	darfGrundSehen bool // Sperrgrund im Klartext statt nur "gesperrt"
	darfOverride   bool // darf eine Sperre bewusst übergehen
}

func (s *Server) processSingleBatchItem(ctx context.Context, k batchKontext, req ActionRequest, index int) ActionBatchResponseItem {
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		return ActionBatchResponseItem{
			Index:   index,
			Success: false,
			Status:  http.StatusBadRequest,
			Error:   "Query ist leer",
		}
	}
	// Kennungen je Eintrag: Der Stapel ist ein Slice und läuft deshalb nicht durch
	// DecodeAndValidate (siehe Handler). Eine aktive Schüler-ID, die keine UUID ist, ginge
	// sonst an Postgres und käme als 500 zurück (uuid_eingaben_test.go).
	if err := Validate.Struct(req); err != nil {
		return ActionBatchResponseItem{
			Index:   index,
			Success: false,
			Status:  http.StatusBadRequest,
			Error:   meldeValidierung(err).Error(),
		}
	}

	lage := s.erlangeIdempotenz(ctx, req.IdempotencyKey)
	if lage.antwort != nil {
		if item, ok := cachedBatchItem(lage.antwort, index); ok {
			return item
		}
	}
	if lage.inArbeit {
		// 503, nicht 409: Der Sync der Theke bucht 4xx aus und meldet; ein Eintrag, dessen
		// Buchung gerade läuft, soll liegen bleiben und in der nächsten Runde die Antwort
		// aus dem Cache bekommen (offlineSync.svelte.js).
		return ActionBatchResponseItem{
			Index:  index,
			Status: http.StatusServiceUnavailable,
			Error:  "wird gerade gebucht — kommt in der nächsten Runde",
		}
	}

	res, err := k.omnibox.ProcessQuery(ctx, service.OmniboxQuery{
		Query:              req.Query,
		ActiveLeserID:      req.ActiveLeserID,
		ConfirmedChecklist: req.ConfirmedChecklist,
		StaffID:            k.userID,
		StaffRole:          k.rolle,
		OverrideBlock:      req.OverrideBlock && k.darfOverride,
	})

	err = ohneSperrgrund(err, k.darfGrundSehen)
	status := http.StatusOK
	if err != nil {
		status = mapServiceErrorToStatus(err)
	}

	item := ActionBatchResponseItem{
		Index:   index,
		Status:  status,
		Success: err == nil,
	}
	if err != nil {
		item.Error = err.Error()
		s.saveToCache(ctx, req.IdempotencyKey, map[string]string{"error": err.Error()}, status)
	} else {
		item.Data = mapOmniboxResultToActionResponse(res)
		s.saveToCache(ctx, req.IdempotencyKey, item.Data, status)
		// Broadcast updates
		if item.Data.Type == "ausleihe" || item.Data.Type == "rueckgabe" {
			s.broadcastActionEvent(*item.Data)
		}
	}

	return item
}
