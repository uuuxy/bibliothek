package system

// Der eine zweckgebundene Leseweg in audit_log.details. Das Protokoll speichert zu Lösch- und
// Ausleihvorgängen Angaben (Barcodes, schueler_id), die die Anwendung sonst nirgends zeigt
// (Datenminimierung, PII-Matrix: /api/audit ist Stufe 0). Liegt ein Buch auf der Theke, dessen
// Exemplar längst gelöscht ist, steht nur dort, wem es gehörte.
//
// Der Zuschnitt ist eng: nur die Suche nach einem Barcode, ein eigenes Recht (audit_details,
// ab Werk nur ADMIN), Stufe 2 in der PII-Matrix, und jeder Abruf schreibt selbst einen Eintrag
// ins Protokoll (audit_logs, mit IP); wer nachschlägt, hinterlässt dieselbe Spur wie bei einer
// DSGVO-Auskunft. Nach Tilgung oder Befristung der Lesehistorie (details ohne schueler_id)
// zeigt auch dieser Weg nichts mehr: Er liest nur, was rechtlich noch da sein darf. Die
// Abfragen stehen in repository/protokoll_theken_auskunft.go.

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/leserart"
	"bibliothek/repository"
)

// thekenExemplar ist ein Exemplar, das der Barcode heute oder früher bezeichnete.
type thekenExemplar struct {
	Titel string `json:"titel"`
	// Status: "im_bestand", "ausgesondert" oder "geloescht" (nur noch im Protokoll).
	Status string `json:"status"`
}

// thekenEreignis ist ein Ausleih-/Rückgabevorgang aus dem Protokoll. Bewusst ohne
// IDs: Für die Theke zählt der Name, nicht der Schlüssel (Datenminimierung).
type thekenEreignis struct {
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

// thekenAuskunft ist die Antwort der Barcode-Suche.
type thekenAuskunft struct {
	Barcode    string           `json:"barcode"`
	Exemplare  []thekenExemplar `json:"exemplare"`
	Ereignisse []thekenEreignis `json:"ereignisse"`
}

// thekenMaxEreignisse kappt die Vorgangsliste — ein Schulbuch sieht selten mehr als
// ein Dutzend Entleiher, und unbegrenzte Listen-Endpunkte sind eine bekannte
// Bugklasse dieses Projekts (siehe protokollMaxZeilen).
const thekenMaxEreignisse = 100

// deuteThekenEreignis deutet eine Protokollzeile: der Leser (Schüler oder Kollege) vor dem
// Konto alter Einträge, und ohne auflösbaren Namen ist der Personenbezug getilgt — keine
// leere Zelle. Bei einem Kollegen steht statt der Klasse die Art; dieselben Wörter wie
// frontend/src/lib/components/leser/leserArt.js (leserart.KlasseOderArt).
func deuteThekenEreignis(z repository.ThekenEreignisZeile) thekenEreignis {
	e := thekenEreignis{Zeitpunkt: z.Zeitpunkt, Aktion: z.Aktion, Bearbeiter: z.BearbeiterName}
	switch {
	case z.LeserName != "":
		e.Entleiher, e.Klasse = z.LeserName, leserart.KlasseOderArt(z.LeserKlasse, z.LeserArt)
	case z.KontoName != "":
		e.Entleiher, e.Klasse = z.KontoName, "Lehrkraft"
	default:
		// Entweder war der Schlüssel nie da (Tilgung/Befristung) oder die Person
		// ist gelöscht — für die Theke dieselbe Antwort: kein Bezug mehr.
		e.PersonenbezugGetilgt = true
	}
	return e
}

// ThekenAuskunft beantwortet GET /api/audit/tresen-auskunft?barcode=…
func (t *Tueren) ThekenAuskunft(protokollAbfragen *repository.ProtokollAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.gibThekenAuskunft(w, r, protokollAbfragen)
	}
}

// gibThekenAuskunft sucht die Exemplare zum Barcode und ihre Vorgänge und protokolliert den
// Blick, bevor er antwortet.
func (t *Tueren) gibThekenAuskunft(w http.ResponseWriter, r *http.Request, protokollAbfragen *repository.ProtokollAbfragen) {
	barcode := strings.TrimSpace(r.URL.Query().Get("barcode"))
	if barcode == "" || len(barcode) > 64 {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Der Barcode fehlt oder ist zu lang", nil)
		return
	}

	exemplarZeilen, err := repository.FindeThekenExemplare(r.Context(), t.DB.Pool, barcode)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}
	exemplare := []thekenExemplar{}
	ids := make([]string, 0, len(exemplarZeilen))
	for _, z := range exemplarZeilen {
		ids = append(ids, z.ExemplarID)
		exemplare = append(exemplare, thekenExemplar{Titel: z.Titel, Status: z.Status})
	}

	ereignisse := []thekenEreignis{}
	if len(ids) > 0 {
		zeilen, err := repository.FindeThekenEreignisse(r.Context(), t.DB.Pool, ids, thekenMaxEreignisse)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		for _, z := range zeilen {
			ereignisse = append(ereignisse, deuteThekenEreignis(z))
		}
	}

	// Die Protokollierung ist Teil der Zusage, nicht Kür: Ein Blick in die
	// Ausleihhistorie ohne eigene Spur wäre genau die stille Tür, gegen die
	// dieser Endpunkt so eng gebaut ist. Scheitert sie, gibt es keine Auskunft.
	claims, ok := auth.ClaimsAusKontext(r.Context())
	if !ok {
		apierrors.AntworteFehler(w, http.StatusInternalServerError,
			errors.New("keine sitzung im kontext"))
		return
	}
	if err := protokollAbfragen.ProtokolliereAdminAktion(r.Context(), claims.KontoID, "TRESEN_AUSKUNFT",
		clientip.AusAnfrage(r), map[string]any{
			"barcode":            barcode,
			"treffer_exemplare":  len(exemplare),
			"treffer_ereignisse": len(ereignisse),
		}); err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	tuer.AntworteJSON(w, http.StatusOK, thekenAuskunft{
		Barcode: barcode, Exemplare: exemplare, Ereignisse: ereignisse,
	})
}
