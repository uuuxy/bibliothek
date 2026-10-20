package mahnwesen

// Die Abgängerliste: Abschlussklassen mit offenen Büchern, noch an der Schule, sichtbar in der
// Saison (Mai bis Juli). Daraus entstehen die Kontoauszüge zum Einsammeln vor der Entlassung
// (Druck hier, Versand in abgaenger_mail.go).
//
// „Abgänger" heißt hier, was die Schule damit meint: die Kinder, die zum Schuljahresende
// gehen. Wer laut LUSD schon weg ist (ist_abgaenger = true, Klasse ABG), steht nicht hier,
// sondern im Mahnwesen; zwei Bedeutungen unter einem Namen hielte jeder Leser für eine.

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/pkg/kennung"
	"bibliothek/repository"
)

// AbgaengerZeile ist eine Zeile der Liste: ein Schüler mit der Zahl seiner offenen und
// davon überfälligen Bücher — das ist die handlungsrelevante Information (was muss noch
// zurück?), nicht die Ausweisnummer.
type AbgaengerZeile struct {
	ID              string `json:"id"`
	BarcodeID       string `json:"barcode_id"`
	Vorname         string `json:"vorname"`
	Nachname        string `json:"nachname"`
	Klasse          string `json:"klasse"`
	AbgaengerJahr   int    `json:"abgaenger_jahr"`
	IstGesperrt     bool   `json:"ist_gesperrt"`
	OffeneAusleihen int    `json:"offene_buecher"`
	Ueberfaellig    int    `json:"ueberfaellig"`
	LehrerEmail     string `json:"lehrer_email"`
}

// AbgaengerAntwort ist die Antwort von GET /api/abgaenger: das Saisonfenster und die
// Zeilen. Außerhalb der Saison ist die Liste leer, ohne dass die Abfrage läuft — die
// Oberfläche zeigt dann den Hinweis mit den Daten statt „alle entlastet".
type AbgaengerAntwort struct {
	Fenster   AbgaengerFenster `json:"fenster"`
	Abgaenger []AbgaengerZeile `json:"abgaenger"`
}

// ladeAbgaengerZeilen liefert eine Zeile je Abgänger mit offenen Ausleihen.
func (t *Tueren) ladeAbgaengerZeilen(ctx context.Context) ([]AbgaengerZeile, error) {
	liste, err := repository.LadeAbgaengerMitOffenenAusleihen(ctx, t.DB.Pool)
	if err != nil {
		return nil, err
	}
	zeilen := make([]AbgaengerZeile, 0, len(liste))
	for _, z := range liste {
		zeilen = append(zeilen, AbgaengerZeile(z))
	}
	return zeilen, nil
}

// Abgaenger liefert die Abgängerliste samt Saisonfenster.
func (t *Tueren) Abgaenger() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		antwort := AbgaengerAntwort{
			Fenster:   abgaengerFensterFuer(t.Jetzt()),
			Abgaenger: []AbgaengerZeile{},
		}
		if antwort.Fenster.Offen {
			zeilen, err := t.ladeAbgaengerZeilen(r.Context())
			if err != nil {
				apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
				return
			}
			antwort.Abgaenger = zeilen
		}
		tuer.AntworteJSON(w, http.StatusOK, antwort)
	}
}

// ladeAbgaengerKontoauszuege lädt die Abgänger mit noch offenen Ausleihen als
// Kontoauszug-Einträge (ein Eintrag je Abgänger, seine Bücher gruppiert), für den
// Stapel-Kontoauszug beim Schulabgang mit Unterschriftszeile. Ein Abgänger ohne offene Bücher
// braucht keinen: Mit einem LEFT JOIN statt des INNER JOIN auf ausleihen kämen beim Massendruck
// von 150 Abgängern 140 leere Seiten aus dem Drucker.
//
// Ein leerer klasse-Filter ("") liefert alle Abgänger; sonst nur die genannte Klasse (für
// den klassenweisen Druck via /api/abgaenger/pdf?klasse=…).
//
// nurIDs engt zusätzlich auf genannte Schüler ein (nil = keine Einengung). Es ist ein
// Schnitt mit dieser Abfrage, keine zweite Auswahl: Wer nicht Abgänger mit offenem Buch
// ist, bekommt auch mit seiner Kennung keine Seite. So folgt der Druck der Suche der
// Oberfläche, ohne dass der Server die Suche ein zweites Mal formuliert.
func (t *Tueren) ladeAbgaengerKontoauszuege(ctx context.Context, klasse string, nurIDs []string) ([]pdf.KontoauszugEintrag, error) {
	ausleihen, err := repository.LadeAbgaengerAusleihen(ctx, t.DB.Pool, klasse, nurIDs)
	if err != nil {
		return nil, err
	}

	nachSchueler := map[string]*pdf.KontoauszugEintrag{}
	reihenfolge := make([]string, 0)
	for _, a := range ausleihen {
		if _, ok := nachSchueler[a.SchuelerID]; !ok {
			nachSchueler[a.SchuelerID] = &pdf.KontoauszugEintrag{
				Schueler: pdf.KontoauszugSchueler{Vorname: a.Vorname, Nachname: a.Nachname, Klasse: a.Klasse},
				Buecher:  []pdf.KontoauszugBuch{},
			}
			reihenfolge = append(reihenfolge, a.SchuelerID)
		}

		nachSchueler[a.SchuelerID].Buecher = append(nachSchueler[a.SchuelerID].Buecher, pdf.KontoauszugBuch{
			Titel:          a.Titel,
			Barcode:        a.ExemplarBarcode,
			Ausleihdatum:   a.AusgeliehenAm,
			Rueckgabedatum: a.Frist,
		})
	}

	eintraege := make([]pdf.KontoauszugEintrag, 0, len(reihenfolge))
	for _, id := range reihenfolge {
		eintraege = append(eintraege, *nachSchueler[id])
	}
	return eintraege, nil
}

// AbgaengerKontoauszuegePDF erzeugt die Kontoauszüge der Abgänger als PDF, eine Seite je
// Schüler mit Freigabezeile.
func (t *Tueren) AbgaengerKontoauszuegePDF() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Außerhalb der Saison gibt es keinen Ausdruck — dieselbe Regel wie die Liste,
		// sonst druckte ein direkter Aufruf im Oktober, was der Bildschirm nicht zeigt.
		if fenster := abgaengerFensterFuer(t.Jetzt()); !fenster.Offen {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, meldungAusserhalbDerSaison(fenster), nil)
			return
		}

		// Optionaler Klassenfilter: /api/abgaenger/pdf?klasse=10a druckt nur diese Klasse.
		klasse := r.URL.Query().Get("klasse")

		// Optional: nur die Schüler, die die Oberfläche gerade zeigt (aktive Suche).
		nurIDs, err := leseAbgaengerAuswahl(r.URL.Query())
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusBadRequest, err)
			return
		}

		eintraege, err := t.ladeAbgaengerKontoauszuege(ctx, klasse, nurIDs)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		if len(eintraege) == 0 {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Keine Abgänger gefunden", nil)
			return
		}

		// Der Kontoauszug der Abgänger trägt die Unterschriftszeile (eine Seite je Abgänger).
		pdfDaten, err := pdf.ErzeugeKontoauszuege(eintraege, true)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		dateiname := "Kontoauszuege_Abgaenger.pdf"
		switch {
		case nurIDs != nil:
			dateiname = "Kontoauszuege_Auswahl.pdf"
		case klasse != "":
			dateiname = fmt.Sprintf("Kontoauszuege_Klasse_%s.pdf", klasse)
		}

		w.Header().Set(tuer.HeaderContentType, tuer.ContentTypePDF)
		w.Header().Set(tuer.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, dateiname))
		w.Header().Set(tuer.HeaderContentLength, fmt.Sprint(len(pdfDaten)))
		http.ServeContent(w, r, dateiname, time.Now(), bytes.NewReader(pdfDaten))
	}
}

// leseAbgaengerAuswahl liest den Parameter `ids` des Kontoauszug-Drucks: die Kennungen
// der Schüler, die die Oberfläche bei aktiver Suche gerade zeigt, mit Komma getrennt.
//
// Drei Fälle, und der mittlere ist der gefährliche:
//
//   - `ids` fehlt → nil: keine Einengung, es gilt nur der Klassenfilter.
//   - `ids` ist da, aber leer → Fehler. Eine Suche ohne Treffer darf nicht als „keine
//     Einengung" gelesen werden — sonst druckte „nichts gefunden" alle Kontoauszüge.
//     Fehlendes Feld und leeres Feld bedeuten hier Verschiedenes.
//   - jede Kennung muss eine UUID in der Form sein, die die Datenbank annimmt (400 statt
//     500, pkg/kennung).
func leseAbgaengerAuswahl(q url.Values) ([]string, error) {
	if !q.Has("ids") {
		return nil, nil
	}
	var ids []string
	for _, teil := range strings.Split(q.Get("ids"), ",") {
		id := strings.TrimSpace(teil)
		if id == "" {
			continue
		}
		if !kennung.IstUUID(id) {
			return nil, apierrors.BadRequest(fmt.Sprintf("„%s“ ist keine gültige Kennung", id), nil)
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, apierrors.BadRequest("Niemand ausgewählt", nil)
	}
	return ids, nil
}
