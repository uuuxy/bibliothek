package katalog

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// regalansichtGrenze kappt die Regalansicht. Ohne Grenze lieferte eine kurze Signatur („BIB")
// den halben Bestand in einer Antwort. Die Kappung wird gemeldet, nicht verschwiegen.
const regalansichtGrenze = 500

// signaturGruppe ist eine im Bestand vorkommende Regaladresse samt Umfang.
type signaturGruppe struct {
	Signatur  string `json:"signatur"`
	Titel     int    `json:"titel"`
	Exemplare int    `json:"exemplare"`
}

// Signaturen liefert die im Bestand vorkommenden Regaladressen mit Umfang.
//
// Die Liste wird aus buecher_titel.signatur abgeleitet, nicht aus einer eigenen Tabelle: Eine
// zweite Liste kennte Namen, die an keinem Buch hängen, und kennte die Signaturen der Bücher
// nicht.
//
// Zusammengefasst wird nach der Regaladresse (repository.sqlSignaturRegaladresse): „LMF Deu 7
// / Bie" und „LMF Deu 7 / Gri" stehen im selben Regal. Das Kürzel dahinter ordnet in Littera
// innerhalb des Regals (Vorgabe: die ersten drei Buchstaben des Verfassers) und gehört meist
// zu einem einzigen Titel; als Vorschlag für ein neues Buch und als Bereich einer Inventur
// taugt nur das Regal.
func (t *Tueren) Signaturen() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		zeilen, err := repository.LadeSignaturGruppen(r.Context(), t.DB.Pool)
		if err != nil {
			if errors.Is(err, repository.ErrZeileUnlesbar) {
				return apierrors.Internal("Signaturzeile unlesbar", err)
			}
			return apierrors.Internal("Signaturen konnten nicht geladen werden", err)
		}

		gruppen := make([]signaturGruppe, 0, len(zeilen))
		for _, z := range zeilen {
			gruppen = append(gruppen, signaturGruppe(z))
		}

		tuer.AntworteJSON(w, http.StatusOK, gruppen)
		return nil
	})
}

// signaturTitel ist eine Zeile der Regalansicht.
type signaturTitel struct {
	TitelID   string `json:"titel_id"`
	Signatur  string `json:"signatur"`
	Titel     string `json:"titel"`
	Autor     string `json:"autor"`
	ISBN      string `json:"isbn"`
	CoverURL  string `json:"cover_url"`
	Exemplare int    `json:"exemplare"`
	Verliehen int    `json:"verliehen"`
}

// titelUnterSignaturAntwort ist die Regalansicht zu einer Signatur.
type titelUnterSignaturAntwort struct {
	Signatur string          `json:"signatur"`
	Titel    []signaturTitel `json:"buecher"`
	Gesamt   int             `json:"gesamt"`
	Gekappt  bool            `json:"gekappt"`
}

// TitelUnterSignatur liefert die Titel unter einer Signatur, in der Reihenfolge des Regals.
//
// Die Signatur gilt als Anfang. Die Bedingung kommt aus repository.signaturPraefixBedingung,
// derselben Funktion, aus der der Bereich einer Inventur entsteht; zwei Kopien liefen
// auseinander, und der Unterschied fiele erst auf, wenn eine Inventur andere Bücher bucht, als
// die Regalansicht zeigt.
//
// Wahlweise: ?limit= senkt die Grenze von 500, ?nur_mit_cover=true lässt Titel ohne gespeichertes
// Cover weg. Beides braucht die Vorschau am Vorschlagsmenü der Signatur, die zwei Cover zeigt.
func (t *Tueren) TitelUnterSignatur() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		signatur := strings.TrimSpace(r.URL.Query().Get("signatur"))
		if signatur == "" {
			return apierrors.BadRequest("Bitte eine Signatur angeben", errors.New("leere signatur"))
		}

		grenze := regalansichtGrenze
		if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && n > 0 {
			grenze = min(n, regalansichtGrenze)
		}
		nurMitCover := r.URL.Query().Get("nur_mit_cover") == "true"

		zeilen, err := repository.LadeTitelUnterSignatur(r.Context(), t.DB.Pool, signatur, grenze+1, nurMitCover)
		if err != nil {
			if errors.Is(err, repository.ErrZeileUnlesbar) {
				return apierrors.Internal("Buchzeile unlesbar", err)
			}
			return apierrors.Internal("Bücher zur Signatur konnten nicht geladen werden", err)
		}

		titel := make([]signaturTitel, 0, len(zeilen))
		for _, z := range zeilen {
			titel = append(titel, signaturTitel(z))
		}

		// Eine Zeile mehr als die Grenze geholt: Nur so lässt sich „es gibt noch mehr" von
		// „genau voll" unterscheiden, ohne eine zweite Abfrage zum Zählen.
		gekappt := len(titel) > grenze
		if gekappt {
			titel = titel[:grenze]
		}

		tuer.AntworteJSON(w, http.StatusOK, titelUnterSignaturAntwort{
			Signatur: signatur,
			Titel:    titel,
			Gesamt:   len(titel),
			Gekappt:  gekappt,
		})
		return nil
	})
}
