package api

// graduates.go — die Abgängerliste: Abschlussklassen mit offenen Büchern, noch an der
// Schule, sichtbar in der Saison (Mai bis Juli). Daraus entstehen die Kontoauszüge zum
// Einsammeln vor der Entlassung (Druck hier, Versand in graduates_mail.go).
//
// „Abgänger" heißt hier, was die Schule damit meint: die Kinder, die zum Schuljahresende
// gehen. Wer laut LUSD schon WEG ist (ist_abgaenger = true, Klasse ABG), steht nicht hier,
// sondern im Mahnwesen — zwei Bedeutungen, die vom 25.06. bis 05.09.2026 denselben Namen
// trugen (Register, Entscheidung 2).

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/pkg/kennung"
	"bibliothek/repository"
)

// AbgaengerZeile ist eine Zeile der Liste: ein Schüler mit der Zahl seiner offenen und
// davon überfälligen Bücher — das ist die handlungsrelevante Information (was muss noch
// zurück?), nicht die Ausweisnummer.
type AbgaengerZeile struct {
	ID            string `json:"id"`
	BarcodeID     string `json:"barcode_id"`
	Vorname       string `json:"vorname"`
	Nachname      string `json:"nachname"`
	Klasse        string `json:"klasse"`
	AbgaengerJahr int    `json:"abgaenger_jahr"`
	IstGesperrt   bool   `json:"ist_gesperrt"`
	OffeneBuecher int    `json:"offene_buecher"`
	Ueberfaellig  int    `json:"ueberfaellig"`
	LehrerEmail   string `json:"lehrer_email"`
}

// AbgaengerAntwort ist die Antwort von GET /api/abgaenger: das Saisonfenster und die
// Zeilen. Außerhalb der Saison ist die Liste leer, ohne dass die Abfrage läuft — die
// Oberfläche zeigt dann den Hinweis mit den Daten statt „alle entlastet".
type AbgaengerAntwort struct {
	Fenster   AbgaengerFenster `json:"fenster"`
	Abgaenger []AbgaengerZeile `json:"abgaenger"`
}

// queryGraduatesBasic liefert eine Zeile je Abgänger mit offenen Ausleihen.
func (s *Server) queryGraduatesBasic(ctx context.Context) ([]AbgaengerZeile, error) {
	liste, err := repository.ListeAbgaengerMitOffenenBuechern(ctx, s.DB.Pool)
	if err != nil {
		return nil, err
	}
	zeilen := make([]AbgaengerZeile, 0, len(liste))
	for _, z := range liste {
		zeilen = append(zeilen, AbgaengerZeile(z))
	}
	return zeilen, nil
}

// GetGraduatesHandler liefert die Abgängerliste samt Saisonfenster.
// @Summary      Abgängerliste
// @Description  Abschlussklassen (9H/10H, 10R, 13) mit noch offenen Büchern, in der Saison vom 01.05. bis 31.07.; außerhalb leer mit offen=false.
// @Tags         admin
// @Produce      json
// @Success      200  {object}  AbgaengerAntwort
// @Failure      500  {object}  map[string]string
// @Router       /abgaenger [get]
func (s *Server) GetGraduatesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		antwort := AbgaengerAntwort{
			Fenster:   abgaengerFensterFuer(s.jetzt()),
			Abgaenger: []AbgaengerZeile{},
		}
		if antwort.Fenster.Offen {
			zeilen, err := s.queryGraduatesBasic(r.Context())
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			antwort.Abgaenger = zeilen
		}
		RespondJSON(w, http.StatusOK, antwort)
	}
}

// queryAbgaengerKontoauszug lädt die Abgänger MIT noch offenen Ausleihen als Kontoauszug-
// Einträge (ein Eintrag je Abgänger, seine Bücher gruppiert). Genutzt für den Stapel-
// Kontoauszug beim Schulabgang (der frühere „Laufzettel" — jetzt ein Kontoauszug mit
// Unterschriftszeile). Ein Abgänger ohne offene Bücher braucht keinen: deshalb INNER JOIN
// auf ausleihen (früher LEFT JOIN) — sonst kamen beim Massendruck von 150 Abgängern 140
// komplett leere Seiten aus dem Drucker.
//
// Ein leerer klasse-Filter ("") liefert alle Abgänger; sonst nur die genannte Klasse (für
// den klassenweisen Druck via /api/abgaenger/pdf?klasse=…).
//
// nurIDs engt zusätzlich auf genannte Schüler ein (nil = keine Einengung). Es ist ein
// SCHNITT mit dieser Abfrage, keine zweite Auswahl: Wer nicht Abgänger mit offenem Buch
// ist, bekommt auch mit seiner Kennung keine Seite. So folgt der Druck der Suche der
// Oberfläche, ohne dass der Server die Suche ein zweites Mal formuliert.
func (s *Server) queryAbgaengerKontoauszug(ctx context.Context, klasse string, nurIDs []string) ([]pdf.KontoauszugEintrag, error) {
	ausleihen, err := repository.ListeAbgaengerAusleihen(ctx, s.DB.Pool, klasse, nurIDs)
	if err != nil {
		return nil, err
	}

	studMap := map[string]*pdf.KontoauszugEintrag{}
	studOrder := make([]string, 0)
	for _, a := range ausleihen {
		if _, ok := studMap[a.SchuelerID]; !ok {
			studMap[a.SchuelerID] = &pdf.KontoauszugEintrag{
				Schueler: pdf.KontoauszugSchueler{Vorname: a.Vorname, Nachname: a.Nachname, Klasse: a.Klasse},
				Buecher:  []pdf.KontoauszugBuch{},
			}
			studOrder = append(studOrder, a.SchuelerID)
		}

		studMap[a.SchuelerID].Buecher = append(studMap[a.SchuelerID].Buecher, pdf.KontoauszugBuch{
			Titel:          a.Titel,
			Barcode:        a.ExemplarBarcode,
			Ausleihdatum:   a.AusgeliehenAm,
			Rueckgabedatum: a.Frist,
		})
	}

	result := make([]pdf.KontoauszugEintrag, 0, len(studOrder))
	for _, id := range studOrder {
		result = append(result, *studMap[id])
	}
	return result, nil
}

// GetGraduatesPDFHandler erzeugt die Kontoauszüge der Abgänger als PDF (eine Seite
// je Schüler, mit Freigabezeile). Hieß früher „Laufzettel" — der Name hing dem
// Dokument noch an, obwohl längst der Kontoauszug erzeugt wird.
// @Summary      Get Kontoauszug PDF
// @Description  Generates a printable PDF for graduating students with their unreturned books (season 01.05.–31.07.).
// @Tags         admin
// @Produce      application/pdf
// @Param        klasse  query  string  false  "nur diese Klasse"
// @Param        ids     query  string  false  "nur diese Schüler (Kennungen, mit Komma getrennt) — der Schnitt mit der Abgängerliste"
// @Router       /abgaenger/pdf [get]
func (s *Server) GetGraduatesPDFHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Außerhalb der Saison gibt es keinen Ausdruck — dieselbe Regel wie die Liste,
		// sonst druckte ein direkter Aufruf im Oktober, was der Bildschirm nicht zeigt.
		if fenster := abgaengerFensterFuer(s.jetzt()); !fenster.Offen {
			apierrors.SendHTTPError(w, http.StatusNotFound, abgaengerAusserhalbDerSaison(fenster))
			return
		}

		// Optionaler Klassenfilter: /api/abgaenger/pdf?klasse=10a druckt nur diese Klasse.
		klasse := r.URL.Query().Get("klasse")

		// Optional: nur die Schüler, die die Oberfläche gerade zeigt (aktive Suche).
		nurIDs, err := abgaengerAuswahlAusQuery(r.URL.Query())
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		result, err := s.queryAbgaengerKontoauszug(ctx, klasse, nurIDs)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if len(result) == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("no graduates found"))
			return
		}

		// Der Abgänger-„Laufzettel" ist jetzt ein Kontoauszug MIT Unterschriftszeile
		// (eine Seite je Abgänger). Ein Dokument statt zweier — Freigabezeile optional.
		pdfBytes, err := pdf.GenerateKontoauszugBatch(result, true)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		filename := "Kontoauszuege_Abgaenger.pdf"
		switch {
		case nurIDs != nil:
			filename = "Kontoauszuege_Auswahl.pdf"
		case klasse != "":
			filename = fmt.Sprintf("Kontoauszuege_Klasse_%s.pdf", klasse)
		}

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Header().Set(headerContentLength, fmt.Sprint(len(pdfBytes)))
		http.ServeContent(w, r, filename, time.Now(), bytes.NewReader(pdfBytes))
	}
}

// abgaengerAuswahlAusQuery liest den Parameter `ids` des Kontoauszug-Drucks: die Kennungen
// der Schüler, die die Oberfläche bei aktiver Suche gerade zeigt, mit Komma getrennt.
//
// Drei Fälle, und der mittlere ist der gefährliche:
//
//   - `ids` fehlt → nil: keine Einengung, es gilt nur der Klassenfilter (wie bisher).
//   - `ids` ist da, aber leer → Fehler. Eine Suche ohne Treffer darf nicht als „keine
//     Einengung" gelesen werden — sonst druckte „nichts gefunden" alle Kontoauszüge.
//     Fehlendes Feld und leeres Feld bedeuten hier Verschiedenes.
//   - jede Kennung muss eine UUID in der Form sein, die die Datenbank annimmt (400 statt
//     500, pkg/kennung).
func abgaengerAuswahlAusQuery(q url.Values) ([]string, error) {
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
			return nil, fmt.Errorf("ids: %q ist keine gültige Kennung", id)
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, errors.New("ids ist leer — es ist niemand ausgewählt")
	}
	return ids, nil
}
