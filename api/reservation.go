package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// KlassensatzReservierungRequest is the payload for a class-set reservation.
type KlassensatzReservierungRequest struct {
	TitelID string `json:"titel_id"`
	Klasse  string `json:"klasse"`
	Anzahl  int    `json:"anzahl"`
	Notiz   string `json:"notiz,omitempty"`
	// IdempotencyKey: vom Client pro Absende-Vorgang vergeben. Ein Doppelklick schickt
	// denselben Schlüssel und wird serverseitig zum No-op (keine zweite Reservierung,
	// keine zweite Mail). Optional — ohne Schlüssel läuft alles wie bisher.
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// CreateKlassensatzReservierungHandler lets a LEHRER submit a class-set reservation.
// POST /api/reservierungen/klassensatz
func (s *Server) CreateKlassensatzReservierungHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req KlassensatzReservierungRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}
		if !validateKlassensatzRequest(w, &req) {
			return
		}

		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("fehlende Sitzungsinformationen"))
			return
		}

		ctx := r.Context()
		repo := repository.NewReservationRepository(s.DB.Pool)

		// Verify the title exists. Ein DB-Fehler ist KEIN „nicht gefunden" (Fehler-Kollaps,
		// Sweep 29.08.2026): Vorher endete ein Verbindungsabbruch als 404, und die
		// Lehrkraft las „Buchtitel nicht gefunden" für einen Titel, der da war.
		exists, err := repo.CheckTitleExists(ctx, req.TitelID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if !exists {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("buchtitel nicht gefunden"))
			return
		}

		// Bestandsdeckelung: Mehr Exemplare zu reservieren, als die Bibliothek überhaupt
		// besitzt, erzeugt eine dauerhaft unerfüllbare Aufgabe im Dashboard (Ghost-Order).
		// Die Wunschmenge wird deshalb gegen den physischen Bestand (nicht ausgesondert)
		// geprüft.
		bestand, err := repo.CountTitleStock(ctx, req.TitelID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if req.Anzahl > bestand {
			apierrors.SendHTTPError(w, http.StatusBadRequest,
				fmt.Errorf("nur %d Exemplare im Bestand — %d können nicht reserviert werden", bestand, req.Anzahl))
			return
		}

		newID, neu, err := repo.CreateKlassensatzReservierung(ctx, req.TitelID, req.Klasse, req.Anzahl, nullableString(req.Notiz), claims.UserID, nullableString(req.IdempotencyKey))
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// neu=false: derselbe Idempotenz-Schlüssel lief schon durch (Doppelklick) — die
		// bestehende Reservierung zurückgeben, kein zweiter Eintrag, keine zweite Mail.
		status := "erstellt"
		if !neu {
			status = "bereits_vorhanden"
		}
		RespondJSON(w, http.StatusCreated, map[string]string{"id": newID, "status": status})
	}
}

// GetKlassensatzReservierungenHandler lists all pending class-set reservations for admins.
// GET /api/reservierungen/klassensatz
func (s *Server) GetKlassensatzReservierungenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo := repository.NewReservationRepository(s.DB.Pool)
		result, err := repo.GetKlassensatzReservierungen(r.Context())
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Ensure we don't return null for empty slices
		if result == nil {
			result = []repository.KlassensatzReservierung{}
		}

		RespondJSON(w, http.StatusOK, result)
	}
}

// GetKlassensatzReservierungenAnzahlHandler returns the count of open reservations (for red badge).
// GET /api/reservierungen/klassensatz/anzahl
func (s *Server) GetKlassensatzReservierungenAnzahlHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo := repository.NewReservationRepository(s.DB.Pool)
		count, err := repo.GetKlassensatzReservierungenAnzahl(r.Context())
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, map[string]int{"anzahl": count})
	}
}

// OffeneKlassensatzReservierungenHandler liefert die Warteschlange für das
// Kollegiums-Portal: wer vor der eigenen Reservierung steht (Titel, Klasse, Menge) —
// bewusst ohne Personendaten des Anfordernden. Eigener Endpunkt, weil die volle
// Liste (GET /api/reservierungen/klassensatz) hinter view_orders liegt und dem
// Kollegium verschlossen bleibt.
// GET /api/reservierungen/klassensatz/offen
func (s *Server) OffeneKlassensatzReservierungenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo := repository.NewReservationRepository(s.DB.Pool)
		offene, err := repo.OffeneKlassensatzReservierungen(r.Context())
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, offene)
	}
}

// ErledigeKlassensatzReservierungHandler schliesst eine Klassensatz-Reservierung ab.
// PUT /api/reservierungen/klassensatz/{id}/erledigen  { "notiz": "24 von 30, Rest bei der 8a" }
//
// Der Body ist optional (leer = ohne Notiz): Der Weg bestand vor Migration 088 ohne
// Body, und ein Client mit altem Bundle soll weiter abschliessen koennen.
func (s *Server) ErledigeKlassensatzReservierungHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("id fehlt"))
			return
		}
		notiz, ok := leseErledigtNotiz(w, r)
		if !ok {
			return
		}

		repo := repository.NewReservationRepository(s.DB.Pool)
		erledigt, err := repo.ErledigeKlassensatzReservierung(r.Context(), id, notiz)

		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if erledigt == nil {
			apierrors.SendHTTPError(w, http.StatusNotFound, pgx.ErrNoRows)
			return
		}

		// Bereit-Mail an die anfragende Lehrkraft — Betreiber-Entscheidung 16.08.2026:
		// KEIN neuer Menuepunkt, das aktive Signal genuegt. Das Abschliessen gilt, auch
		// wenn der Mailversand klemmt — aber der Ausfall ist seit der Ausfallmatrix
		// (20.08.2026) nicht mehr SPURLOS: Vorher stand hier 204 + Logzeile, die Theke
		// sah "abgeschlossen", die Lehrkraft wartete auf eine Mail, die nie kam, und
		// niemand wusste es. Jetzt traegt die Antwort den Mail-Status und die
		// Oberflaeche warnt ("bitte selbst benachrichtigen"). Ohne Konto-Adresse
		// (per SQL angelegt, Konto geloescht) gibt es schlicht keine Mail.
		mailStatus := mailStatusKeineAdresse
		if erledigt.AnfragendeMail != nil && *erledigt.AnfragendeMail != "" {
			mailStatus = mailStatusVersendet
			if err := SendEmail(MailRequest{
				To:      *erledigt.AnfragendeMail,
				Subject: fmt.Sprintf("Ihr Klassensatz liegt bereit: %s", erledigt.TitelName),
				Body:    klassensatzBereitText(erledigt),
			}); err != nil {
				log.Printf("Klassensatz-Bereit-Mail an %s fehlgeschlagen: %v", *erledigt.AnfragendeMail, err)
				mailStatus = mailStatusFehlgeschlagen
			}
		}

		RespondJSON(w, http.StatusOK, map[string]string{"mail": mailStatus})
	}
}

// leseErledigtNotiz liest die optionale Notiz aus dem Body. Kein Body → "" (Altclient,
// Erledigen ohne Antwort); ein vorhandener, aber kaputter Body ist ein 400.
func leseErledigtNotiz(w http.ResponseWriter, r *http.Request) (string, bool) {
	if r.Body == nil || r.ContentLength == 0 {
		return "", true
	}
	var body struct {
		Notiz string `json:"notiz"`
	}
	if !DecodeAndValidate(w, r, &body) {
		return "", false
	}
	return kuerze(strings.TrimSpace(body.Notiz), 500), true
}

// klassensatzBereitText ist die Bereit-Mail. Seit Migration 088 traegt sie die Notiz
// der Lehrkraft (damit die Antwort einen Bezug hat) und die Antwort der Bibliothek —
// dasselbe Muster wie die Anliegen-Mail. Vorher war der Text fest und liess sich
// weder ergaenzen (Teilmenge, Abholort) noch korrigieren.
func klassensatzBereitText(e *repository.KlassensatzErledigt) string {
	text := fmt.Sprintf("Ihre Klassensatz-Reservierung ist fertig zusammengestellt und liegt in der Bibliothek zur Abholung bereit.\n\n"+
		"  Titel:  %s\n  Klasse: %s\n  Anzahl: %d Exemplare\n",
		e.TitelName, e.Klasse, e.Anzahl)
	if e.Notiz != "" {
		text += fmt.Sprintf("  Ihre Notiz: %s\n", e.Notiz)
	}
	if e.ErledigtNotiz != "" {
		text += fmt.Sprintf("\nNotiz der Bibliothek:\n  %s\n", e.ErledigtNotiz)
	}
	return text + "\nDiese Mail wurde automatisch beim Abschliessen der Reservierung verschickt."
}

// validateKlassensatzRequest prüft die Pflichtfelder und normalisiert die Anzahl
// (Default 1, Obergrenze 200). ok=false: die Fehlerantwort wurde bereits geschrieben.
func validateKlassensatzRequest(w http.ResponseWriter, req *KlassensatzReservierungRequest) bool {
	if req.TitelID == "" || req.Klasse == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("titel_id und klasse sind erforderlich"))
		return false
	}
	if req.Anzahl <= 0 {
		req.Anzahl = 1
	}
	if req.Anzahl > 200 {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("anzahl darf 200 nicht überschreiten"))
		return false
	}
	return true
}

// nullableString converts an empty string to nil for nullable DB columns.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
