package schaden

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Das Erstellen eines Bescheids: Eingabe prüfen, Pflichtangaben, schreiben, protokollieren.

// bescheidErstellenAnfrage ist die Eingabe des Dialogs.
type bescheidErstellenAnfrage struct {
	Mittel     string `json:"mittel"`
	FristBis   string `json:"frist_bis"`
	Positionen []struct {
		SchadensfallID string  `json:"schadensfall_id" validate:"omitempty,uuid_oder_leer"`
		Betrag         float64 `json:"betrag"`
	} `json:"positionen" validate:"dive"`
	// Ausleihen: überfällige Bücher, die mit dem Brief als Verlust gebucht werden.
	Ausleihen []struct {
		AusleiheID string  `json:"ausleihe_id" validate:"omitempty,uuid_oder_leer"`
		Betrag     float64 `json:"betrag"`
	} `json:"ausleihen" validate:"dive"`
}

// BescheidErstellen schreibt den Bescheid und liefert ihn zurück.
//
// @Summary      Bescheid für einen Schüler erstellen
// @Tags         schadensersatz
// @Accept       json
// @Produce      json
// @Param        id path string true "Kennung des Schülers"
// @Param        body body bescheidErstellenAnfrage true "Topf, Frist und Positionen"
// @Success      201 {object} repository.Bescheid
// @Router       /schueler/{id}/bescheide [post]
func (t *Tueren) BescheidErstellen(bescheidAbfragen repository.BescheidAbfragen, protokollAbfragen repository.ProtokollAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		return t.erstelleBescheid(w, r, bescheidAbfragen, protokollAbfragen)
	})
}

// erstelleBescheid prüft die Anfrage und die Pflichtangaben, schreibt den Bescheid und
// protokolliert ihn.
func (t *Tueren) erstelleBescheid(w http.ResponseWriter, r *http.Request, bescheidAbfragen repository.BescheidAbfragen, protokollAbfragen repository.ProtokollAbfragen) error {
	id := r.PathValue("id")
	var req bescheidErstellenAnfrage
	if id == "" {
		return apierrors.BadRequest(tuer.MeldungIDLeer, errors.New("missing id"))
	}
	if !tuer.LeseAnfrage(w, r, &req) {
		return nil
	}
	frist, err := pruefeBescheidAnfrage(req)
	if err != nil {
		return err
	}

	ctx := r.Context()
	angaben, schule, err := t.bescheidAngaben(ctx)
	if err != nil {
		return apierrors.Internal(tuer.MeldungEinstellungenUnlesbar, err)
	}
	// Ohne die Pflichtangaben entsteht kein Bescheid: Ein Brief ohne Referenznummer
	// oder ohne Aufsichtsbehörde ist keiner, und die Nummer wäre verbraucht.
	if fehlt := angaben.FehlendeAngaben(schule); len(fehlt) > 0 {
		meldung := fmt.Sprintf("Es fehlen Angaben für den Bescheid: %s. Bitte in den Einstellungen unter „Schadensersatz\" eintragen.",
			strings.Join(fehlt, ", "))
		return apierrors.Conflict(meldung, errors.New("bescheid-angaben unvollständig"))
	}

	snapshot, err := bescheidSnapshotAus(ctx, bescheidAbfragen, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apierrors.NotFound("Schüler nicht gefunden", err)
		}
		return apierrors.Internal("Empfänger konnte nicht gelesen werden", err)
	}

	eingabe := repository.BescheidEingabe{
		SchuelerID:     id,
		Mittel:         req.Mittel,
		Kassenjahr:     frist.Year(),
		FristBis:       frist,
		Empfaenger:     snapshot,
		Absender:       bescheidAbsenderAus(angaben, schule),
		ErstelltVon:    bescheidAkteur(ctx),
		Referenznummer: func(nr int) string { return angaben.Referenznummer(frist.Year(), nr) },
	}
	eingabe.Positionen, eingabe.Verluste = bescheidPostenAus(req)

	bescheid, err := bescheidAbfragen.LegeAn(ctx, eingabe)
	if err != nil {
		// Die Zuordnungs-Prüfung ist ein Bedienfehler (Forderung bezahlt, storniert
		// oder schon auf einem Brief), kein Serverfehler — ebenso ein Buch, das
		// inzwischen zurück ist, schon gemeldet wurde oder einem anderen Kind gehört.
		if meldung, ok := lageGeaendert(err); ok {
			return apierrors.Conflict(meldung, err)
		}
		return apierrors.Internal("Bescheid konnte nicht erstellt werden", err)
	}

	protokolliereBescheid(ctx, t, protokollAbfragen, aktionBescheidErstellt, map[string]any{
		"bescheid_id":    bescheid.ID,
		"schueler_id":    id,
		"referenznummer": bescheid.Referenznummer,
		"mittel":         bescheid.Mittel,
		"gesamtbetrag":   bescheid.Gesamtbetrag,
		"positionen":     bescheid.AnzahlPositionen,
	}, clientip.AusAnfrage(r))

	tuer.AntworteJSON(w, http.StatusCreated, bescheid)
	return nil
}

// pruefeBescheidAnfrage prüft Topf, Auswahl und Frist, bevor irgendetwas gelesen oder
// geschrieben wird, und liefert die Frist als Kalendertag der Schule.
func pruefeBescheidAnfrage(req bescheidErstellenAnfrage) (time.Time, error) {
	if !mitteltopf.IstGueltig(req.Mittel) {
		return time.Time{}, apierrors.BadRequest(mitteltopf.MeldungUngueltig, mitteltopf.ErrUngueltig)
	}
	// Der Brief kennt nur einen Wortlaut und ein Konto: die des Landes. Für die Bücher der
	// Schülerbücherei (Mittel des Schulträgers) gibt es noch keinen Brief; „schultraeger" gäbe
	// sonst einen Landes-Bescheid mit Landeskonto für ein Buch des Trägers.
	if req.Mittel != mitteltopf.Land {
		return time.Time{}, apierrors.Conflict("Ein Bescheid entsteht nur für Lernmittel des Landes. Die Rechnung für Bücher der Schülerbücherei ist noch nicht gebaut.",
			errors.New("bescheid nur für mittel=land"))
	}
	if len(req.Positionen) == 0 && len(req.Ausleihen) == 0 {
		return time.Time{}, apierrors.BadRequest("Bitte mindestens ein Buch auswählen", errors.New("keine positionen"))
	}
	frist, err := schulzeit.Kalendertag(req.FristBis)
	if err != nil {
		return time.Time{}, apierrors.BadRequest("Die Frist muss ein Datum der Form JJJJ-MM-TT sein", err)
	}
	// Frühestens morgen: Eine Frist von heute oder gestern machte den Bescheid sofort
	// übergabefähig. Höchstens ein Jahr voraus: Ihr Jahr geht als Kassenjahr in die
	// Referenznummer, ein Tippfehler (2062) gäbe eine Nummer, der keine Zahlung zugeordnet
	// wird. Die Arbeitshilfe nennt eine „Vierwochen-Frist … ab dem Briefdatum".
	jetzt := schulzeit.Jetzt()
	heute := time.Date(jetzt.Year(), jetzt.Month(), jetzt.Day(), 0, 0, 0, 0, schulzeit.Zone())
	if !frist.After(heute) {
		return time.Time{}, apierrors.BadRequest("Die Frist muss nach dem heutigen Tag liegen. Üblich sind vier Wochen ab Briefdatum.",
			errors.New("frist nicht nach heute"))
	}
	if frist.After(heute.AddDate(1, 0, 0)) {
		return time.Time{}, apierrors.BadRequest("Die Frist liegt mehr als ein Jahr in der Zukunft. Bitte das Datum prüfen, ihr Jahr geht als Kassenjahr in die Referenznummer.",
			errors.New("frist mehr als ein jahr voraus"))
	}
	return frist, nil
}

// bescheidPostenAus übersetzt die Auswahl des Dialogs: Forderungen, die es schon gibt, und
// überfällige Bücher, die mit dem Brief als Verlust gebucht werden.
func bescheidPostenAus(req bescheidErstellenAnfrage) ([]repository.BescheidPositionEingabe, []repository.BescheidVerlustEingabe) {
	var positionen []repository.BescheidPositionEingabe
	for _, p := range req.Positionen {
		positionen = append(positionen,
			repository.BescheidPositionEingabe{SchadensfallID: p.SchadensfallID, Betrag: p.Betrag})
	}
	var verluste []repository.BescheidVerlustEingabe
	for _, a := range req.Ausleihen {
		verluste = append(verluste,
			repository.BescheidVerlustEingabe{AusleiheID: a.AusleiheID, Betrag: a.Betrag})
	}
	return positionen, verluste
}

// lageGeaendert nennt zu einem Bedienfehler, bei dem sich die Lage geändert hat, seit Akte oder
// Dialog offen stehen, den Satz für die Bibliothekskraft; ok ist false für jeden anderen Fehler.
// Schadensmeldung und Bescheid antworten darauf mit 409.
func lageGeaendert(err error) (meldung string, ok bool) {
	var nichtZugeordnet *repository.ForderungenNichtZugeordnet
	switch {
	case errors.As(err, &nichtZugeordnet):
		return fmt.Sprintf("%d von %d Forderungen konnten nicht zugeordnet werden — "+
			"gehören sie diesem Schüler, sind sie offen, noch auf keinem Bescheid und Bücher des Landes?",
			nichtZugeordnet.Fehlend, nichtZugeordnet.Verlangt), true
	case errors.Is(err, repository.ErrAusleiheInzwischenZurueck):
		return "Ein gewähltes Buch ist inzwischen zurückgegeben — bitte den Dialog neu öffnen.", true
	case errors.Is(err, repository.ErrAusleiheSchonGemeldet):
		return "Für ein gewähltes Buch ist schon eine Forderung erfasst — bitte den Dialog neu öffnen.", true
	case errors.Is(err, repository.ErrAusleiheFremd):
		return "Ein gewähltes Buch ist nicht an dieses Kind ausgeliehen.", true
	case errors.Is(err, repository.ErrExemplarNeuVerliehen):
		return "Exemplar wurde zwischenzeitlich neu ausgeliehen — bitte den Vorgang neu laden", true
	}
	return "", false
}
