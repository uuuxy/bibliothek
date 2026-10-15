package katalog

import (
	"errors"
	"net/http"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pkg/leserart"
	"bibliothek/repository"
)

// ausleiherZeile represents a student who is currently borrowing a copy of a title.
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
	// nach einem Jahr rot — die Akte wusste es seit dem 16.09.2026 besser (OFFEN.md 5.18).
	IstDauerleihe bool `json:"ist_dauerleihe"`
}

// TitelAusleiher lists all active borrowers for a book title.
// @Router       /buecher/titel/{id}/ausleiher [get]
func (t *Tueren) TitelAusleiher() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing title ID parameter"))
			return
		}

		ctx := r.Context()

		zeilen, err := repository.LadeAusleiherDesTitels(ctx, t.DB.Pool, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Beim Kollegen steht statt der Klasse das Wort seiner Art (leserart.KlasseOderArt), dieselbe
		// Auskunft wie in der Titel-Historie; der Klassenfilter des Reiters liest dieses Feld.
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

// historieZeile represents a historical loan for a copy of a title.
type historieZeile struct {
	Vorname         string     `json:"schueler_name"`
	Nachname        string     `json:"schueler_nachname"`
	Klasse          string     `json:"klasse"`
	ExemplarBarcode string     `json:"exemplar_barcode"`
	AusgeliehenAm   time.Time  `json:"ausgeliehen_am"`
	RueckgabeAm     *time.Time `json:"rueckgabe_am"` // Can be null if still borrowed
}

// TitelHistorie lists all historical loans for a book title.
// @Router       /buecher/titel/{id}/historie [get]
func (t *Tueren) TitelHistorie() http.HandlerFunc {
	return t.gibTitelHistorie
}

// gibTitelHistorie liefert die letzten 200 Ausleih-Vorgänge eines Titels. Seit der
// Lesehistorie-Befristung (jobs/cron_dsgvo_lesehistorie.go) trägt ein Großteil davon keine
// schueler_id mehr: Name → "Anonym", Klasse leer. Das Wort einer Art („Lehrkraft",
// „Fachbereich" …, leserart.KlasseOderArt) steht nur, wenn wirklich ein Kollege ausgeliehen hat —
// bis zum 22.08.2026 machte COALESCE(s.klasse, 'Lehrer') aus jeder getrennten
// Schüler-Ausleihe eine Lehrer-Ausleihe, bis zum 30.09.2026 hieß jeder Kollege „Lehrer". Als
// Top-Level-Methode ausgelagert (nicht Inline-Closure), damit die Scan-Schleife nicht
// zusätzlich als Verschachtelung zählt (S3776).
func (t *Tueren) gibTitelHistorie(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing title ID parameter"))
		return
	}

	ctx := r.Context()

	zeilen, err := repository.LadeAusleihhistorieDesTitels(ctx, t.DB.Pool, id)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
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

// textOder liefert den Wert von s, oder fallback wenn s nil ist.
func textOder(wert *string, sonst string) string {
	if wert != nil {
		return *wert
	}
	return sonst
}
