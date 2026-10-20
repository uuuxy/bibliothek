package katalog

import (
	"net/http"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pkg/leserart"
	"bibliothek/repository"
)

// ausleiherZeile ist ein Leser, der gerade ein Exemplar des Titels ausgeliehen hat.
type ausleiherZeile struct {
	Vorname          string    `json:"schueler_name"`
	Nachname         string    `json:"schueler_nachname"`
	Klasse           string    `json:"klasse"`
	AusleiherBarcode string    `json:"schueler_barcode"`
	ExemplarBarcode  string    `json:"exemplar_barcode"`
	AusgeliehenAm    time.Time `json:"ausgeliehen_am"`
	RueckgabeFrist   time.Time `json:"rueckgabe_frist"`
	// IstDauerleihe: Ausleihe an jemanden, der kein Schüler ist (`ausleihen.ist_handapparat`).
	// Ohne das Merkmal färbten die Ausleiher-Liste und ihr Druck die Frist eines Kollegen
	// nach einem Jahr rot; die Leserakte kennt es ebenfalls.
	IstDauerleihe bool `json:"ist_dauerleihe"`
}

// TitelAusleiher liefert die Leser, die gerade ein Exemplar des Titels ausgeliehen haben.
func (t *Tueren) TitelAusleiher() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung des Titels fehlt", nil)
			return
		}

		ctx := r.Context()

		zeilen, err := repository.LadeAusleiherDesTitels(ctx, t.DB.Pool, id)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		// Beim Kollegen steht statt der Klasse das Wort seiner Art (leserart.KlasseOderArt),
		// dieselbe Auskunft wie in der Titel-Historie; der Klassenfilter des Reiters liest dieses
		// Feld.
		ausleiher := make([]ausleiherZeile, 0, len(zeilen))
		for _, z := range zeilen {
			ausleiher = append(ausleiher, ausleiherZeile{
				Vorname:          z.Vorname,
				Nachname:         z.Nachname,
				Klasse:           leserart.KlasseOderArt(z.Klasse, z.Art),
				AusleiherBarcode: z.AusleiherBarcode,
				ExemplarBarcode:  z.ExemplarBarcode,
				AusgeliehenAm:    z.AusgeliehenAm,
				RueckgabeFrist:   z.RueckgabeFrist,
				IstDauerleihe:    z.IstDauerleihe,
			})
		}

		tuer.AntworteJSON(w, http.StatusOK, ausleiher)
	}
}

// historieZeile ist eine Ausleihe eines Exemplars des Titels, offen oder zurückgegeben.
type historieZeile struct {
	Vorname         string     `json:"schueler_name"`
	Nachname        string     `json:"schueler_nachname"`
	Klasse          string     `json:"klasse"`
	ExemplarBarcode string     `json:"exemplar_barcode"`
	AusgeliehenAm   time.Time  `json:"ausgeliehen_am"`
	RueckgabeAm     *time.Time `json:"rueckgabe_am"` // null, solange das Exemplar ausgeliehen ist
}

// TitelHistorie liefert die letzten 200 Ausleihen eines Titels (gibTitelHistorie).
func (t *Tueren) TitelHistorie() http.HandlerFunc {
	return t.gibTitelHistorie
}

// gibTitelHistorie liefert die letzten 200 Ausleihen eines Titels. Wegen der befristeten
// Lesehistorie (jobs/cron_dsgvo_lesehistorie.go) trägt ein Großteil davon keinen Leser mehr:
// Der Name heißt dann „Anonym", die Klasse bleibt leer. Das Wort einer Art („Lehrkraft",
// „Fachbereich" …, leserart.KlasseOderArt) steht nur, wenn wirklich ein Kollege ausgeliehen
// hat; eine getrennte Ausleihe eines Schülers ist keine Ausleihe einer Lehrkraft.
func (t *Tueren) gibTitelHistorie(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung des Titels fehlt", nil)
		return
	}

	ctx := r.Context()

	zeilen, err := repository.LadeAusleihhistorieDesTitels(ctx, t.DB.Pool, id)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	historie := make([]historieZeile, 0, len(zeilen))
	for _, z := range zeilen {
		historie = append(historie, historieZeile{
			Vorname:         textOder(z.Vorname, "Anonym"),
			Nachname:        textOder(z.Nachname, ""),
			Klasse:          leserart.KlasseOderArt(textOder(z.Klasse, ""), textOder(z.Art, "")),
			ExemplarBarcode: z.ExemplarBarcode,
			AusgeliehenAm:   z.AusgeliehenAm,
			RueckgabeAm:     z.RueckgabeAm,
		})
	}

	tuer.AntworteJSON(w, http.StatusOK, historie)
}

// textOder liefert den Text hinter wert oder sonst, wenn wert nil ist.
func textOder(wert *string, sonst string) string {
	if wert != nil {
		return *wert
	}
	return sonst
}
