package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/service"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Die Nachbuch-Tür der Theke (Stufe 2 des Offline-Baus, Commit 11, OFFEN.md 2.2).
//
// Der Theken-Rechner schickt seine Warteschlange, der Server bucht je Eintrag die
// WIRKLICHKEIT und antwortet je Eintrag mit einem Ergebnis. Abweichungen vom Scan stehen
// zusätzlich als Nachbuch-Meldung am Server, bis jemand sie quittiert — nichts
// verschwindet still (Entscheidung Peter, 13.09.2026, c).
//
// Der Handler ist dünn: Er prüft die Form, löst den bekannten Idempotenz-Schlüssel auf und
// reicht durch. Die Regeln liegen im Dienst (internal/service/nachbuchen.go), das SQL im
// repository — wie überall.

// nachbuchenHoechstens: 1–50 Einträge je Aufruf. Mehr wäre eine Transaktion, die zu lange
// offen steht; die Theke schickt in Portionen (Stufe 3).
const nachbuchenHoechstens = 50

// NachbuchenRequest ist eine Portion der Warteschlange.
type NachbuchenRequest struct {
	Eintraege []NachbuchenEintrag `json:"eintraege" validate:"required,min=1,max=50,dive"`
}

// NachbuchenEintrag ist ein offline gescannter Vorgang.
type NachbuchenEintrag struct {
	// Schluessel ist der Idempotenz-Schlüssel des Eintrags — derselbe, den der
	// Online-Versand benutzt hätte. Daran erkennt der Server einen abgebrochenen Versand.
	Schluessel string `json:"schluessel" validate:"required,uuid"`
	// Absicht: was der Bediener beim Scan meinte ("ausleihe" | "rueckgabe").
	Absicht string `json:"absicht" validate:"required,oneof=ausleihe rueckgabe"`
	// Barcode des Buchs, wie gescannt (B-…, nackte Ziffern, LMF-…).
	Barcode string `json:"barcode" validate:"required"`
	// GescanntAm ist der Zeitpunkt am Theken-Rechner; der Server nimmt höchstens seine
	// eigene Zeit (eine falsch gehende Theken-Uhr datiert nichts vor).
	GescanntAm time.Time `json:"gescannt_am" validate:"required"`
	// Person: was der Rechner beim Scan schon auflösen konnte …
	SchuelerID *string `json:"schueler_id,omitempty" validate:"omitempty,uuid_oder_leer"`
	LehrerID   *string `json:"lehrer_id,omitempty" validate:"omitempty,uuid_oder_leer"`
	// … sonst der offline gescannte Ausweis, den der Server auflöst.
	AusweisBarcode *string `json:"ausweis_barcode,omitempty"`
}

// NachbuchenErgebnis ist die Antwort zu einem Eintrag.
type NachbuchenErgebnis struct {
	Schluessel string `json:"schluessel"`
	// Ergebnis: ausgeliehen · umgebucht · bereits_ausgeliehen · zurueckgegeben ·
	// nur_reaktiviert · nicht_gebucht · veraltet · wiederholen.
	Ergebnis string `json:"ergebnis"`
	// Grund steht bei nicht_gebucht und veraltet — der Satz, den die Meldung trägt.
	Grund string `json:"grund,omitempty"`
	// Daten ist die gebuchte Wirkung in der Form des Theken-Scans (nur bei Erfolg).
	Daten *ActionResponse `json:"daten,omitempty"`
	// AufsichtInformieren: Das Buch stand auf einem Bescheid, der schon bei der
	// Schulaufsicht liegt — eigenes Feld, weil das eine Aufgabe ist, keine Meldung.
	AufsichtInformieren string `json:"aufsicht_informieren,omitempty"`
}

// NachbuchenResponse ist die Antwort auf eine Portion.
type NachbuchenResponse struct {
	Ergebnisse []NachbuchenErgebnis `json:"ergebnisse"`
}

// nachbuchWiederholen: Der Server konnte diesen Eintrag nicht beurteilen (Datenbank weg).
// Er bleibt auf dem Theken-Rechner liegen und wird später erneut geschickt — deshalb steht
// er NICHT in den Ergebnissen der Meldungstabelle.
const nachbuchWiederholen = "wiederholen"

// NachbuchenHandler nimmt eine Portion der Warteschlange entgegen.
// @Summary      Offline-Warteschlange nachbuchen
// @Description  Bucht je Eintrag die Wirklichkeit (Umbuchung, Rückgabe, Rückholen) und meldet jede Abweichung vom Scan.
// @Tags         theke
// @Accept       json
// @Produce      json
// @Param        body body NachbuchenRequest true "Einträge der Warteschlange"
// @Success      200 {object} NachbuchenResponse
// @Router       /action/nachbuchen [post]
func (s *Server) NachbuchenHandler(nachbuchSvc service.NachbuchService) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			return apierrors.Unauthorized("nicht angemeldet", errors.New("missing session information"))
		}
		var req NachbuchenRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		if len(req.Eintraege) > nachbuchenHoechstens {
			return apierrors.BadRequest("höchstens 50 Einträge je Aufruf", errors.New("zu viele einträge"))
		}

		ctx := r.Context()
		antwort := NachbuchenResponse{Ergebnisse: make([]NachbuchenErgebnis, 0, len(req.Eintraege))}
		for _, e := range req.Eintraege {
			antwort.Ergebnisse = append(antwort.Ergebnisse, s.bucheEintragNach(ctx, nachbuchSvc, e, claims.UserID))
		}
		RespondJSON(w, http.StatusOK, antwort)
		return nil
	})
}

// bucheEintragNach bucht einen Eintrag. Ein Serverfehler wird NICHT zum Fehler des ganzen
// Aufrufs: Die übrigen Einträge sollen durchlaufen, und der gescheiterte bleibt auf dem
// Rechner liegen („wiederholen"). Schweigen wäre hier das Schlimmste — die Theke hielte
// ihn für erledigt (Stufe 1, Commit 6: „erledigt ist nur, was der Server gebucht hat").
func (s *Server) bucheEintragNach(ctx context.Context, svc service.NachbuchService, e NachbuchenEintrag, staffID string) NachbuchenErgebnis {
	bekannt, err := s.schluesselSchonGesehen(ctx, e.Schluessel)
	if err != nil {
		return NachbuchenErgebnis{Schluessel: e.Schluessel, Ergebnis: nachbuchWiederholen, Grund: "Server nicht erreichbar"}
	}
	erg, err := svc.Nachbuchen(ctx, service.NachbuchEintrag{
		Schluessel: e.Schluessel, Absicht: e.Absicht, Barcode: e.Barcode, GescanntAm: e.GescanntAm,
		SchuelerID: e.SchuelerID, LehrerID: e.LehrerID, AusweisBarcode: e.AusweisBarcode,
		SchluesselBekannt: bekannt, StaffID: staffID,
	})
	if err != nil {
		log.Printf("nachbuchen: Eintrag %s konnte nicht gebucht werden: %v", e.Schluessel, err)
		return NachbuchenErgebnis{Schluessel: e.Schluessel, Ergebnis: nachbuchWiederholen, Grund: "konnte nicht gebucht werden"}
	}
	out := NachbuchenErgebnis{
		Schluessel: e.Schluessel, Ergebnis: erg.Ergebnis, Grund: erg.Grund,
		AufsichtInformieren: erg.AufsichtHinweis,
	}
	if erg.Result != nil {
		out.Daten = mapOmniboxResultToActionResponse(service.LoanResultAlsOmnibox(erg.Result))
	}
	return out
}

// schluesselSchonGesehen fragt die Idempotenz-Tabelle: Hat der Online-Versand diesen
// Schlüssel schon einmal gebucht? Dann darf der Wächter den älteren Scan durchlassen und
// die fehlende Hälfte nachholen (OFFEN.md 2.2, Commit 11).
func (s *Server) schluesselSchonGesehen(ctx context.Context, schluessel string) (bool, error) {
	antwort, err := repository.LiesIdempotenzAntwort(ctx, s.DB.Pool, schluessel)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return antwort != nil && !antwort.InArbeit(), nil
}
