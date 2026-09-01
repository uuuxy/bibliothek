package api

// audit_tresen_auskunft.go — der EINE zweckgebundene Leseweg in audit_log.details.
//
// Betreiber-Entscheidung 01.09.2026 (Befund-Register): Das Prüfprotokoll speichert
// zu Lösch- und Ausleihvorgängen Details (Barcode-Snapshots, schueler_id), aber die
// Anwendung zeigt sie bewusst nirgends an (Datenminimierung, PII-Matrix: /api/audit
// ist Stufe 0). Der eine Fall, der dadurch unlösbar wurde: Ein Buch liegt auf dem
// Tresen, sein Exemplar ist längst gelöscht — wem gehörte es? Die Information
// existiert im Protokoll, war aber unerreichbar.
//
// Der Zuschnitt ist eng und bleibt es: NUR Barcode-Suche, eigenes Recht
// (audit_details, ab Werk nur ADMIN), Stufe 2 in der PII-Matrix, und jeder Abruf
// wird selbst protokolliert (audit_logs, mit IP) — wer nachschlägt, hinterlässt
// dieselbe Spur wie eine DSGVO-Auskunft. Nach DSGVO-Tilgung oder Lesehistorie-
// Befristung (details ohne schueler_id) zeigt auch dieser Weg bewusst nichts mehr:
// Er liest nur, was rechtlich noch da sein darf, und stellt nichts wieder her.
//
// Die Abfragen selbst stehen in repository/audit_tresen.go (Schichtungs-Regel:
// ein Handler formuliert kein SQL).

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// TresenExemplar ist ein Exemplar, das der Barcode heute oder früher bezeichnete.
type TresenExemplar struct {
	Titel string `json:"titel"`
	// Status: "im_bestand", "ausgesondert" oder "geloescht" (nur noch im Protokoll).
	Status string `json:"status"`
}

// TresenEreignis ist ein Ausleih-/Rückgabevorgang aus dem Protokoll. Bewusst ohne
// IDs: Für den Tresen zählt der Name, nicht der Schlüssel (Datenminimierung).
type TresenEreignis struct {
	Zeitpunkt time.Time `json:"zeitpunkt"`
	Aktion    string    `json:"aktion"` // CHECKOUT | RETURN
	// Entleiher ist der Klarname (Schüler oder Lehrkraft) — leer, wenn der
	// Personenbezug getilgt ist oder die Person nicht mehr existiert.
	Entleiher string `json:"entleiher"`
	Klasse    string `json:"klasse"`
	// PersonenbezugGetilgt: DSGVO-Tilgung/Befristung hat den Bezug entfernt, oder
	// die Person ist gelöscht. So liest sich eine leere Zelle nicht als Datenfehler.
	PersonenbezugGetilgt bool   `json:"personenbezug_getilgt"`
	Bearbeiter           string `json:"bearbeiter"`
}

// TresenAuskunft ist die Antwort der Barcode-Suche.
type TresenAuskunft struct {
	Barcode    string           `json:"barcode"`
	Exemplare  []TresenExemplar `json:"exemplare"`
	Ereignisse []TresenEreignis `json:"ereignisse"`
}

// tresenMaxEreignisse kappt die Vorgangsliste — ein Schulbuch sieht selten mehr als
// ein Dutzend Entleiher, und unbegrenzte Listen-Endpunkte sind eine bekannte
// Bugklasse dieses Projekts (siehe auditLogMaxZeilen).
const tresenMaxEreignisse = 100

// tresenEreignis deutet eine Protokollzeile: Schüler vor Lehrkraft, und ohne
// auflösbaren Namen ist der Personenbezug getilgt — keine leere Zelle.
func tresenEreignis(z repository.TresenEreignisZeile) TresenEreignis {
	e := TresenEreignis{Zeitpunkt: z.Zeitpunkt, Aktion: z.Aktion, Bearbeiter: z.BearbeiterName}
	switch {
	case z.SchuelerName != "":
		e.Entleiher, e.Klasse = z.SchuelerName, z.SchuelerKlasse
	case z.LehrkraftName != "":
		e.Entleiher, e.Klasse = z.LehrkraftName, "Lehrkraft"
	default:
		// Entweder war der Schlüssel nie da (Tilgung/Befristung) oder die Person
		// ist gelöscht — für den Tresen dieselbe Antwort: kein Bezug mehr.
		e.PersonenbezugGetilgt = true
	}
	return e
}

// TresenAuskunftHandler beantwortet GET /api/audit/tresen-auskunft?barcode=…
func (s *Server) TresenAuskunftHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		barcode := strings.TrimSpace(r.URL.Query().Get("barcode"))
		if barcode == "" || len(barcode) > 64 {
			apierrors.SendHTTPError(w, http.StatusBadRequest,
				errors.New("barcode fehlt oder ist unplausibel lang"))
			return
		}

		exemplarZeilen, err := repository.SucheTresenExemplare(r.Context(), s.DB.Pool, barcode)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		exemplare := []TresenExemplar{}
		ids := make([]string, 0, len(exemplarZeilen))
		for _, z := range exemplarZeilen {
			ids = append(ids, z.ExemplarID)
			exemplare = append(exemplare, TresenExemplar{Titel: z.Titel, Status: z.Status})
		}

		ereignisse := []TresenEreignis{}
		if len(ids) > 0 {
			zeilen, err := repository.SucheTresenEreignisse(r.Context(), s.DB.Pool, ids, tresenMaxEreignisse)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			for _, z := range zeilen {
				ereignisse = append(ereignisse, tresenEreignis(z))
			}
		}

		// Die Protokollierung ist Teil der Zusage, nicht Kür: Ein Blick in die
		// Ausleihhistorie ohne eigene Spur wäre genau die stille Tür, gegen die
		// dieser Endpunkt so eng gebaut ist. Scheitert sie, gibt es keine Auskunft.
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusInternalServerError,
				errors.New("keine sitzung im kontext"))
			return
		}
		if err := auditRepo.LogAdminAktion(r.Context(), claims.UserID, "TRESEN_AUSKUNFT",
			getIP(r), map[string]any{
				"barcode":            barcode,
				"treffer_exemplare":  len(exemplare),
				"treffer_ereignisse": len(ereignisse),
			}); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, TresenAuskunft{
			Barcode: barcode, Exemplare: exemplare, Ereignisse: ereignisse,
		})
	}
}
