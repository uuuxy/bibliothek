package lernmittel

// Der LMF-Plan lesend: Rückgabe- und Ausgabetermine je Klasse. Das Kollegium liest sie im
// Portal, für alle gleich, und als PDF in der Form der früheren Excel-Liste. Lesen verlangt nur
// eine Sitzung (Stufe 0: Daten und Klassen, kein Bezug zu Schülern). Geschrieben wird der Plan
// als Reihenfolge (lmf_plan.go). Sichtbar sind nur veröffentlichte Pläne; den Entwurf als PDF
// für die Schulleitung bekommt nur der Planer (edit_books) über eine eigene Route.

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// lmfTermineAntwort ist die Antwort von GET /api/lmf-termine.
type lmfTermineAntwort struct {
	// Ab ist das Datum, ab dem gelistet wird (Beginn des laufenden Schuljahres), leer bei ?alle=1.
	Ab      string                 `json:"ab"`
	Termine []repository.LMFTermin `json:"termine"`
	// Eingangsjahrgaenge (Einstellung): die Jahrgänge, die nach den Ferien Bücher
	// bekommen — für den erklärenden Satz über der Tabelle.
	Eingangsjahrgaenge []int `json:"eingangsjahrgaenge"`
}

// lmfPlanAb bestimmt, ab wann gelistet wird: der 1. August des laufenden Schuljahres,
// damit im Juni die neue Rückgabe und die August-Ausgabe zusammen stehen und im
// Herbst die alte Rückgabe verschwunden ist; ?alle=1 hebt die Grenze auf.
func (t *Tueren) lmfPlanAb(r *http.Request) time.Time {
	if r.URL.Query().Get("alle") == "1" {
		return time.Time{}
	}
	return repository.SchuljahrBeginn(t.Jetzt())
}

// LMFTermine listet den Plan.
func (t *Tueren) LMFTermine() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		abfragen := repository.NewLMFAbfragen(t.DB.Pool)
		ab := t.lmfPlanAb(r)
		eingang, err := t.lmfEingangsjahrgaenge(r.Context())
		if err != nil {
			return apierrors.Internal("Einstellungen laden", err)
		}
		termine, err := abfragen.LadeTermine(r.Context(), ab, repository.LMFListenFilter{})
		if err != nil {
			return apierrors.Internal("LMF-Plan laden", err)
		}
		antwort := lmfTermineAntwort{Termine: termine, Eingangsjahrgaenge: eingang}
		if !ab.IsZero() {
			antwort.Ab = ab.Format("2006-01-02")
		}
		tuer.AntworteJSON(w, http.StatusOK, antwort)
		return nil
	})
}

// lmfArtTitel ist die Überschrift je Art, dieselben Worte wie ARTEN in lmfplanDienst.js.
// „Rückgabe" und „Ausgabe" allein sagen zu wenig: Vor den Ferien tauschen die Klassen (alte ab,
// neue direkt mit), nach den Ferien bekommen nur die neu gebildeten Klassen ihre Bücher.
func lmfArtTitel(art string) string {
	if art == repository.LMFTerminAusgabe {
		return "Bücherausgabe nach den Sommerferien"
	}
	return "Büchertausch vor den Sommerferien"
}

// lmfArtErklaerung ist der eine Satz unter der Überschrift, mit den Eingangsjahrgängen.
func lmfArtErklaerung(art string, eingang []int) string {
	if art == repository.LMFTerminAusgabe {
		// Ohne bekannte Jahrgänge kein leeres Klammerpaar („(Jahrgang )"): Der Satz nennt dann
		// nur die Regel. Beide Seiten müssen das gleich halten; TestLMFTexte prüft es.
		if len(eingang) == 0 {
			return "Nur die neu gebildeten Klassen bekommen ihre Schulbücher."
		}
		return "Nur die neu gebildeten Klassen (Jahrgang " + jahrgaengeText(eingang) + ") bekommen ihre Schulbücher."
	}
	return "Alle Klassen geben die alten Schulbücher ab und bekommen direkt die neuen. " +
		"„Nur Rückgabe“: Abschlussklassen und Klassen, die zum neuen Schuljahr neu gebildet werden."
}

// jahrgaengeText: „5 und 7", „5, 7 und 11".
func jahrgaengeText(eingang []int) string {
	teile := make([]string, 0, len(eingang))
	for _, j := range eingang {
		teile = append(teile, fmt.Sprint(j))
	}
	if len(teile) <= 1 {
		return strings.Join(teile, "")
	}
	return strings.Join(teile[:len(teile)-1], ", ") + " und " + teile[len(teile)-1]
}

// lmfPlanAbschnitte gruppiert die Termine für das PDF: erst der Tausch vor den Ferien, dann
// die Ausgabe danach, je Abschnitt nach Zeitpunkt: Datum, Stunde und bei Gleichstand die
// Position im Plan.
//
// Das ist nicht die Reihenfolge der Zeilen im Planer: Ein fester Platz kann eine Zeile nach
// hinten schieben, ohne ihre Nummer zu ändern; im Planer bleibt sie Zeile 2, im Portal und im
// PDF steht sie dort, wo sie stattfindet. Das Kollegium liest einen Fahrplan, die Bibliothek
// arbeitet eine Reihenfolge ab. Die Position als letztes Kriterium hält die Ausgabe stabil; die
// Kennung einer Zeile ist nach jedem Speichern neu.
//
// „Nur Rückgabe" steht, wo die Bibliothek es hingeschrieben hat: im Vermerk.
func lmfPlanAbschnitte(termine []repository.LMFTermin, eingang []int) ([]pdf.LMFPlanAbschnitt, error) {
	abschnitte := []pdf.LMFPlanAbschnitt{
		{Titel: strings.ToUpper(lmfArtTitel(repository.LMFTerminRueckgabe)), Untertitel: lmfArtErklaerung(repository.LMFTerminRueckgabe, eingang)},
		{Titel: strings.ToUpper(lmfArtTitel(repository.LMFTerminAusgabe)), Untertitel: lmfArtErklaerung(repository.LMFTerminAusgabe, eingang)},
	}
	for _, t := range termine {
		datum, err := schulzeit.Kalendertag(t.Datum)
		if err != nil {
			return nil, err
		}
		z := pdf.LMFPlanZeile{Datum: datum, Stunde: t.Stunde, Klassen: strings.Join(t.Klassen, "/"), Vermerk: t.Vermerk}
		if t.Art == repository.LMFTerminAusgabe {
			abschnitte[1].Zeilen = append(abschnitte[1].Zeilen, z)
		} else {
			abschnitte[0].Zeilen = append(abschnitte[0].Zeilen, z)
		}
	}
	return abschnitte, nil
}

// LMFPlanPDF liefert den Plan als PDF in der Form der früheren Excel-Liste.
// mitEntwuerfen = true ist die Planer-Route (edit_books): auch der unveröffentlichte
// Entwurf, damit er zur Abnahme an die Schulleitung gehen kann.
func (t *Tueren) LMFPlanPDF(mitEntwuerfen bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		abfragen := repository.NewLMFAbfragen(t.DB.Pool)
		eingang, err := t.lmfEingangsjahrgaenge(r.Context())
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		termine, err := abfragen.LadeTermine(r.Context(), t.lmfPlanAb(r),
			repository.LMFListenFilter{MitEntwuerfen: mitEntwuerfen})
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		if len(termine) == 0 {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Der LMF-Plan ist leer", nil)
			return
		}
		abschnitte, err := lmfPlanAbschnitte(termine, eingang)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		pdfDaten, err := pdf.ErzeugeLMFPlan(abschnitte, t.Jetzt())
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		dateiname := "LMF-Plan.pdf"
		w.Header().Set(tuer.HeaderContentType, tuer.ContentTypePDF)
		w.Header().Set(tuer.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, dateiname))
		w.Header().Set(tuer.HeaderContentLength, fmt.Sprint(len(pdfDaten)))
		http.ServeContent(w, r, dateiname, time.Now(), bytes.NewReader(pdfDaten))
	}
}
