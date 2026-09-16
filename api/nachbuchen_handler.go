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
)

// Die Nachbuch-Tür der Theke (Stufe 2 des Offline-Baus, Commit 11, OFFEN.md 2.2).
//
// Der Theken-Rechner schickt seine Warteschlange, der Server bucht je Eintrag die
// WIRKLICHKEIT und antwortet je Eintrag mit einem Ergebnis. Abweichungen vom Scan stehen
// zusätzlich als Nachbuch-Meldung am Server, bis jemand sie quittiert — nichts
// verschwindet still (entschieden am 13.09.2026, c).
//
// Der Handler ist dünn: Er prüft die Form, misst die Uhr des Rechners, klärt den
// Idempotenz-Schlüssel (nachbuchen_schluessel.go) und reicht durch. Die Regeln liegen im
// Dienst (internal/service/nachbuchen.go), das SQL im repository — wie überall.

// nachbuchenHoechstens: 1–50 Einträge je Aufruf. Mehr wäre eine Transaktion, die zu lange
// offen steht; die Theke schickt in Portionen (Stufe 3).
const nachbuchenHoechstens = 50

// uhrVersatzWarnenAb: Ab diesem Versatz steht die Uhr des Theken-Rechners im Log. Gebucht wird
// trotzdem richtig (der Versatz wird herausgerechnet); die Zeile sagt nur, dass die Uhr eines
// Rechners gestellt werden sollte.
const uhrVersatzWarnenAb = 2 * time.Minute

// NachbuchenRequest ist eine Portion der Warteschlange.
type NachbuchenRequest struct {
	// GesendetAm ist die Uhrzeit des Theken-Rechners beim Versand dieser Portion — von DERSELBEN
	// Uhr wie gescannt_am. Der Server misst daran den Versatz der Rechner-Uhr und rechnet die
	// Scan-Zeitpunkte der Portion auf seine Uhr um. Pflicht: Ohne sie ließe sich eine falsch
	// gehende Uhr nicht erkennen.
	GesendetAm time.Time           `json:"gesendet_am" validate:"required"`
	Eintraege  []NachbuchenEintrag `json:"eintraege" validate:"required,min=1,max=50,dive"`
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
	// GescanntAm ist der Zeitpunkt am Theken-Rechner, nach seiner Uhr. Der Server rechnet den
	// Versatz der Uhr heraus (gesendet_am) und nimmt höchstens seine eigene Zeit.
	GescanntAm time.Time `json:"gescannt_am" validate:"required"`
	// Person: was der Rechner beim Scan schon auflösen konnte …
	LeserID *string `json:"leser_id,omitempty" validate:"omitempty,uuid_oder_leer"`
	// … sonst der offline gescannte Ausweis, den der Server auflöst.
	AusweisBarcode *string `json:"ausweis_barcode,omitempty"`
}

// NachbuchenErgebnis ist die Antwort zu einem Eintrag.
type NachbuchenErgebnis struct {
	Schluessel string `json:"schluessel"`
	// Ergebnis: ausgeliehen · umgebucht · bereits_ausgeliehen · zurueckgegeben ·
	// nur_reaktiviert · nicht_gebucht · veraltet · bereits_gebucht · wiederholen.
	Ergebnis string `json:"ergebnis"`
	// Grund steht bei nicht_gebucht, veraltet und wiederholen — der Satz, den die Meldung trägt.
	Grund string `json:"grund,omitempty"`
	// Daten ist die gebuchte Wirkung in der Form des Theken-Scans (bei Erfolg und bei
	// bereits_gebucht die Wirkung des Online-Versands).
	Daten *ActionResponse `json:"daten,omitempty"`
	// AufsichtInformieren: Das Buch stand auf einem Bescheid, der schon bei der
	// Schulaufsicht liegt — eigenes Feld, weil das eine Aufgabe ist, keine Meldung.
	AufsichtInformieren string `json:"aufsicht_informieren,omitempty"`
}

// NachbuchenResponse ist die Antwort auf eine Portion.
type NachbuchenResponse struct {
	// UhrVersatzSekunden ist der gemessene Versatz der Rechner-Uhr (Serverzeit minus Rechnerzeit):
	// negativ = die Uhr des Rechners geht vor. Die Theke kann eine falsch gehende Uhr damit melden.
	UhrVersatzSekunden int                  `json:"uhr_versatz_sekunden"`
	Ergebnisse         []NachbuchenErgebnis `json:"ergebnisse"`
}

// nachbuchWiederholen: Der Server konnte diesen Eintrag nicht beurteilen (Datenbank weg, oder
// eine Buchung unter demselben Schlüssel läuft noch). Er bleibt auf dem Theken-Rechner liegen und
// wird später erneut geschickt — deshalb steht er NICHT in den Ergebnissen der Meldungstabelle.
const nachbuchWiederholen = "wiederholen"

// nachbuchBereitsGebucht: Der Online-Versand unter demselben Schlüssel hat den Scan schon
// vollständig gebucht, nur seine Antwort kam nicht an. Nichts wird gebucht, nichts gemeldet;
// Daten trägt die damals gebuchte Wirkung. Kein Wort der Meldungstabelle.
const nachbuchBereitsGebucht = "bereits_gebucht"

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
		// Die Empfangszeit wird VOR dem Einlesen festgehalten. Der gemessene Versatz enthält die
		// Laufzeit der Anfrage bis hierher: Umgerechnet liegt ein Scan um diese Laufzeit zu spät,
		// im Schulnetz Millisekunden — weniger, als ein Buch braucht, um von einer Theke zur
		// anderen zu kommen.
		empfangen := time.Now()
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

		versatz := empfangen.Sub(req.GesendetAm)
		if versatz >= uhrVersatzWarnenAb || versatz <= -uhrVersatzWarnenAb {
			log.Printf("nachbuchen: WARNUNG die Uhr des Theken-Rechners weicht um %s ab (Konto %s) — Scan-Zeitpunkte werden umgerechnet, die Uhr sollte gestellt werden",
				(-versatz).Round(time.Second), claims.UserID)
		}

		ctx := r.Context()
		antwort := NachbuchenResponse{
			UhrVersatzSekunden: int(versatz.Round(time.Second) / time.Second),
			Ergebnisse:         make([]NachbuchenErgebnis, 0, len(req.Eintraege)),
		}
		// Die Zahl VOR dem Nachbuchen. Gemeldet wird hinterher an der WIRKUNG, nicht an
		// einer Liste der Ergebnisse, die eine Meldung erzeugen: Welche das sind, weiß der
		// Dienst, und eine zweite Liste hier wäre die zweite Wahrheitsquelle — sie stimmte
		// genau bis zur nächsten Meldung, die jemand hinzufügt.
		offenVorher, zaehlbar := s.zaehleMeldungen(ctx)
		for _, e := range req.Eintraege {
			antwort.Ergebnisse = append(antwort.Ergebnisse, s.bucheEintragNach(ctx, nachbuchSvc, e, claims.UserID, versatz))
		}
		if zaehlbar {
			if offenNachher, ok := s.zaehleMeldungen(ctx); ok && offenNachher != offenVorher {
				s.meldeMeldungsstand()
			}
		}
		RespondJSON(w, http.StatusOK, antwort)
		return nil
	})
}

// bucheEintragNach bucht einen Eintrag. Ein Serverfehler wird NICHT zum Fehler des ganzen
// Aufrufs: Die übrigen Einträge sollen durchlaufen, und der gescheiterte bleibt auf dem
// Rechner liegen („wiederholen"). Schweigen wäre hier das Schlimmste — die Theke hielte
// ihn für erledigt (Stufe 1, Commit 6: „erledigt ist nur, was der Server gebucht hat").
func (s *Server) bucheEintragNach(ctx context.Context, svc service.NachbuchService, e NachbuchenEintrag, staffID string, uhrVersatz time.Duration) NachbuchenErgebnis {
	lage, err := s.ergreifeNachbuchSchluessel(ctx, e)
	if err != nil {
		log.Printf("nachbuchen: Schlüssel %s nicht lesbar: %v", e.Schluessel, err)
		return NachbuchenErgebnis{Schluessel: e.Schluessel, Ergebnis: nachbuchWiederholen, Grund: "Server nicht erreichbar"}
	}
	if lage.antwort != nil {
		fertig := *lage.antwort
		fertig.Schluessel = e.Schluessel
		return fertig
	}
	erg, err := svc.Nachbuchen(ctx, service.NachbuchEintrag{
		Schluessel: e.Schluessel, Absicht: e.Absicht, Barcode: e.Barcode,
		GescanntAm: e.GescanntAm, UhrVersatz: uhrVersatz,
		LeserID: e.LeserID, AusweisBarcode: e.AusweisBarcode,
		NachFremdrueckgabeVon: lage.fremdrueckgabeVon, StaffID: staffID,
	})
	if err != nil {
		log.Printf("nachbuchen: Eintrag %s konnte nicht gebucht werden: %v", e.Schluessel, err)
		s.gibNachbuchSchluesselZurueck(ctx, e.Schluessel, lage)
		return NachbuchenErgebnis{Schluessel: e.Schluessel, Ergebnis: nachbuchWiederholen, Grund: "konnte nicht gebucht werden"}
	}
	out := NachbuchenErgebnis{
		Schluessel: e.Schluessel, Ergebnis: erg.Ergebnis, Grund: erg.Grund,
		AufsichtInformieren: erg.AufsichtHinweis,
	}
	if erg.Result != nil {
		out.Daten = mapOmniboxResultToActionResponse(service.LoanResultAlsOmnibox(erg.Result))
	}
	s.legeNachbuchErgebnisAb(ctx, out)
	return out
}

// zaehleMeldungen liefert die Zahl der offenen Nachbuch-Meldungen. Der zweite Wert ist
// false, wenn sie sich nicht lesen ließ — dann wird nichts gemeldet, statt eine Änderung
// zu behaupten. Der Zähler holt sich seinen Stand bei der nächsten Anmeldung ohnehin neu.
func (s *Server) zaehleMeldungen(ctx context.Context) (int, bool) {
	n, err := repository.ZaehleOffeneNachbuchMeldungen(ctx, s.DB.Pool)
	if err != nil {
		log.Printf("nachbuchen: Meldungen nicht zählbar: %v", err)
		return 0, false
	}
	return n, true
}

// meldeMeldungsstand sagt allen Arbeitsplätzen, dass sich die Zahl der offenen Meldungen
// geändert hat. Ohne Inhalt: Die Zeilen nennen Schüler und Vorbesitzer, und die Leitung
// geht an JEDE Sitzung — auch an die eines Helfers, der die Liste nicht sehen darf. Wer
// sie sehen darf, holt sie sich hinter seinem Recht ab.
func (s *Server) meldeMeldungsstand() {
	if s.Broker != nil {
		s.Broker.Broadcast("nachbuch-meldungen", "{}")
	}
}
