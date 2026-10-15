package katalog

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// regalansichtGrenze kappt die Regalansicht. Ohne Grenze liefert eine kurze
// Signatur ("BIB") den halben Bestand in einer Antwort — dieselbe Falle, in die
// /api/audit schon einmal gelaufen ist. Die Kappung wird gemeldet, nicht verschwiegen.
const regalansichtGrenze = 500

// signaturGruppe ist eine im Bestand vorkommende Regaladresse samt Umfang.
type signaturGruppe struct {
	Signatur  string `json:"signatur"`
	Titel     int    `json:"titel"`
	Exemplare int    `json:"exemplare"`
}

// Signaturen liefert die im Bestand vorkommenden Regaladressen mit Umfang.
//
// Die Liste wird aus buecher_titel.signatur ABGELEITET, nicht aus einer Stammtabelle.
// Die frühere Tabelle `signatures` war eine zweite, ungepflegte Wahrheit: Sie kannte
// Namen, die an keinem Buch hingen, und kannte die Signaturen der Bücher nicht.
//
// Zusammengefasst wird nach der Regaladresse (repository.sqlSignaturRegaladresse): „LMF Deu 7
// / Bie" und „LMF Deu 7 / Gri" stehen im selben Regal. Das Kürzel dahinter ordnet in Littera
// innerhalb des Regals (Vorgabe: die ersten drei Buchstaben des Verfassers) und gehört meist
// zu einem einzigen Titel; als Vorschlag für ein neues Buch und als Bereich einer Inventur
// taugt nur das Regal.
//
// @Summary      List shelf addresses in stock
// @Description  Returns the shelf addresses (the part of a signature before " / ") that occur on titles, with title and copy counts.
// @Tags         books
// @Produce      json
// @Success      200  {array}   signaturGruppe
// @Failure      500  {object}  map[string]string
// @Router       /signaturen [get]
func (t *Tueren) Signaturen() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
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

// TitelUnterSignatur liefert die Titel unter einer Signatur — in Regalreihenfolge.
//
// Die Signatur wird als PRÄFIX verstanden. Das Prädikat kommt aus
// repository.signaturPraefixBedingung — derselben Funktion, aus der sich auch der
// Inventur-Scope speist. Zwei eigene Kopien wären hier auseinandergelaufen, und der
// Unterschied fiele erst auf, wenn eine Inventur andere Bücher bucht, als die
// Regalansicht zeigt.
//
// @Summary      List books under a signature
// @Description  Returns titles whose signature matches the given prefix, in shelf order.
// @Tags         books
// @Produce      json
// @Param        signatur  query     string  true  "Signature prefix"
// @Success      200       {object}  titelUnterSignaturAntwort
// @Failure      400       {object}  map[string]string
// @Failure      500       {object}  map[string]string
// @Router       /signaturen/buecher [get]
func (t *Tueren) TitelUnterSignatur() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		signatur := strings.TrimSpace(r.URL.Query().Get("signatur"))
		if signatur == "" {
			return apierrors.BadRequest("signatur ist erforderlich", errors.New("leere signatur"))
		}

		zeilen, err := repository.LadeTitelUnterSignatur(r.Context(), t.DB.Pool, signatur, regalansichtGrenze+1)
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

		// Eine Zeile mehr als das Limit geholt: Nur so lässt sich "es gibt noch mehr"
		// von "genau voll" unterscheiden, ohne eine zweite Zählquery zu fahren.
		gekappt := len(titel) > regalansichtGrenze
		if gekappt {
			titel = titel[:regalansichtGrenze]
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
