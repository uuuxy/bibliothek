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
	"fmt"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/repository"
)

// abgaengerBedingung ist die EINE Definition, wer in der Abgängerliste steht: nicht
// gelöscht, noch an der Schule und in einer Abschlussklasse nach der Regel der Versetzung.
// Liste, Kontoauszug-Druck und Versand lesen alle dieses Prädikat.
var abgaengerBedingung = "s.deleted_at IS NULL AND s.ist_abgaenger = false AND " +
	repository.AbschlussklasseSQL("s.klasse")

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
	query := `
		SELECT s.id, s.barcode_id, s.vorname, s.nachname, s.klasse, s.abgaenger_jahr, s.ist_gesperrt,
		       COUNT(a.id)                                        AS offene_buecher,
		       COUNT(a.id) FILTER (WHERE a.rueckgabe_frist < now()) AS ueberfaellig,
		       coalesce(m.lehrer_email, '')                        AS lehrer_email
		FROM schueler s
		JOIN ausleihen a ON s.id = a.schueler_id
		-- Klassenleitung mitliefern: Ohne sie kann die Oberfläche VOR dem Versand nicht
		-- zeigen, welche Klasse überhaupt eine Adresse hat. Verglichen wird normalisiert
		-- (getrimmt, Kleinschreibung) — „5a", „5A" und „5a " sind dieselbe Klasse, und
		-- ein unsichtbares Leerzeichen darf keinen stillen Nullversand auslösen.
		LEFT JOIN klassen_lehrer_mapping m ON lower(btrim(m.klasse)) = lower(btrim(s.klasse))
		WHERE ` + abgaengerBedingung + `
		  AND a.rueckgabe_am IS NULL
		GROUP BY s.id, s.barcode_id, s.vorname, s.nachname, s.klasse, s.abgaenger_jahr, s.ist_gesperrt, m.lehrer_email
		ORDER BY s.klasse, s.nachname
	`
	rows, err := s.DB.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	zeilen := []AbgaengerZeile{}
	for rows.Next() {
		var z AbgaengerZeile
		if err := rows.Scan(&z.ID, &z.BarcodeID, &z.Vorname, &z.Nachname, &z.Klasse, &z.AbgaengerJahr,
			&z.IstGesperrt, &z.OffeneBuecher, &z.Ueberfaellig, &z.LehrerEmail); err != nil {
			return nil, err
		}
		zeilen = append(zeilen, z)
	}
	return zeilen, rows.Err()
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
func (s *Server) queryAbgaengerKontoauszug(ctx context.Context, klasse string) ([]pdf.KontoauszugEintrag, error) {
	detailQuery := `
		SELECT s.id, s.vorname, s.nachname, s.klasse,
		       t.titel,
		       coalesce(e.barcode_id, '') AS ex_barcode,
		       a.ausgeliehen_am,
		       a.rueckgabe_frist
		FROM schueler s
		JOIN ausleihen a ON s.id = a.schueler_id AND a.rueckgabe_am IS NULL
		JOIN buecher_exemplare e ON a.exemplar_id = e.id
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE ` + abgaengerBedingung + `
		  AND ($1 = '' OR s.klasse = $1)
		ORDER BY s.klasse, s.nachname, t.titel
	`
	rows, err := s.DB.Pool.Query(ctx, detailQuery, klasse)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	studMap := map[string]*pdf.KontoauszugEintrag{}
	studOrder := make([]string, 0)
	for rows.Next() {
		var id, vorname, nachname, klasse, titel, exBarcode string
		var ausgeliehenAm, frist time.Time
		if err := rows.Scan(&id, &vorname, &nachname, &klasse,
			&titel, &exBarcode, &ausgeliehenAm, &frist); err != nil {
			return nil, err
		}

		if _, ok := studMap[id]; !ok {
			studMap[id] = &pdf.KontoauszugEintrag{
				Schueler: pdf.KontoauszugSchueler{Vorname: vorname, Nachname: nachname, Klasse: klasse},
				Buecher:  []pdf.KontoauszugBuch{},
			}
			studOrder = append(studOrder, id)
		}

		studMap[id].Buecher = append(studMap[id].Buecher, pdf.KontoauszugBuch{
			Titel:          titel,
			Barcode:        exBarcode,
			Ausleihdatum:   ausgeliehenAm,
			Rueckgabedatum: frist,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
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

		result, err := s.queryAbgaengerKontoauszug(ctx, klasse)
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
		if klasse != "" {
			filename = fmt.Sprintf("Kontoauszuege_Klasse_%s.pdf", klasse)
		}

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Header().Set(headerContentLength, fmt.Sprint(len(pdfBytes)))
		http.ServeContent(w, r, filename, time.Now(), bytes.NewReader(pdfBytes))
	}
}
