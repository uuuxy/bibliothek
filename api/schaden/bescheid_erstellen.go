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

// BescheidErstellenRequest ist die Eingabe des Dialogs.
type BescheidErstellenRequest struct {
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

// BescheidErstellenHandler schreibt den Bescheid und liefert ihn zurück.
//
// @Summary      Create a compensation notice for a student
// @Tags         schadensersatz
// @Accept       json
// @Produce      json
// @Param        id path string true "Student ID"
// @Param        body body BescheidErstellenRequest true "Pot, deadline and positions"
// @Success      201 {object} repository.Bescheid
// @Router       /schueler/{id}/bescheide [post]
func (s *Tueren) BescheidErstellenHandler(bescheidRepo repository.BescheidAbfragen, auditRepo repository.ProtokollAbfragen) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		return s.handleBescheidErstellen(w, r, bescheidRepo, auditRepo)
	})
}

// handleBescheidErstellen prüft die Anfrage und die Pflichtangaben, schreibt den Bescheid und
// protokolliert ihn.
func (s *Tueren) handleBescheidErstellen(w http.ResponseWriter, r *http.Request, bescheidRepo repository.BescheidAbfragen, auditRepo repository.ProtokollAbfragen) error {
	id := r.PathValue("id")
	var req BescheidErstellenRequest
	if id == "" {
		return apierrors.BadRequest(tuer.MeldungIDLeer, errors.New("missing id"))
	}
	if !tuer.DecodeAndValidate(w, r, &req) {
		return nil
	}
	frist, err := pruefeBescheidAnfrage(req)
	if err != nil {
		return err
	}

	ctx := r.Context()
	angaben, schule, err := s.bescheidAngaben(ctx)
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

	snapshot, err := bescheidSnapshotAus(ctx, bescheidRepo, id)
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

	bescheid, err := bescheidRepo.LegeAn(ctx, eingabe)
	if err != nil {
		// Die Zuordnungs-Prüfung ist ein Bedienfehler (Forderung bezahlt, storniert
		// oder schon auf einem Brief), kein Serverfehler — ebenso ein Buch, das
		// inzwischen zurück ist, schon gemeldet wurde oder einem anderen Kind gehört.
		if istBescheidBedienfehler(err) {
			return apierrors.Conflict(err.Error(), err)
		}
		return apierrors.Internal("Bescheid konnte nicht erstellt werden", err)
	}

	bescheidAudit(ctx, s, auditRepo, auditBescheidErstellt, map[string]any{
		"bescheid_id":    bescheid.ID,
		"schueler_id":    id,
		"referenznummer": bescheid.Referenznummer,
		"mittel":         bescheid.Mittel,
		"gesamtbetrag":   bescheid.Gesamtbetrag,
		"positionen":     bescheid.AnzahlPositionen,
	}, clientip.AusAnfrage(r))

	tuer.RespondJSON(w, http.StatusCreated, bescheid)
	return nil
}

// pruefeBescheidAnfrage prüft Topf, Auswahl und Frist, bevor irgendetwas gelesen oder
// geschrieben wird, und liefert die Frist als Kalendertag der Schule.
func pruefeBescheidAnfrage(req BescheidErstellenRequest) (time.Time, error) {
	if !mitteltopf.IstGueltig(req.Mittel) {
		return time.Time{}, apierrors.BadRequest(mitteltopf.ErrUngueltig.Error(), mitteltopf.ErrUngueltig)
	}
	// Der Brief kennt nur einen Wortlaut und ein Konto: die des Landes. Die Rechnung
	// der Schülerbücherei (Mittel des Schulträgers) ist Etappe 3 — bis dahin gäbe
	// „schultraeger" einen Landes-Bescheid mit Landeskonto für ein Buch des Trägers.
	if req.Mittel != mitteltopf.Land {
		//nolint:staticcheck // ST1005: ganzer Satz — die Meldung steht so vor der Bibliothekskraft.
		return time.Time{}, apierrors.Conflict("Ein Bescheid entsteht nur für Lernmittel des Landes. Die Rechnung für Bücher der Schülerbücherei ist noch nicht gebaut.",
			errors.New("bescheid nur für mittel=land"))
	}
	if len(req.Positionen) == 0 && len(req.Ausleihen) == 0 {
		//nolint:staticcheck // ST1005: ganzer Satz — die Meldung steht so vor der Bibliothekskraft.
		return time.Time{}, apierrors.BadRequest("Bitte mindestens ein Buch auswählen.", errors.New("keine positionen"))
	}
	frist, err := schulzeit.Kalendertag(req.FristBis)
	if err != nil {
		//nolint:staticcheck // ST1005: ganzer Satz.
		return time.Time{}, apierrors.BadRequest("Die Frist muss ein Datum sein (JJJJ-MM-TT).", err)
	}
	// Frühestens morgen: Eine Frist von heute oder gestern machte den Bescheid sofort
	// übergabefähig. Höchstens ein Jahr voraus: Ihr Jahr geht als Kassenjahr in die
	// Referenznummer, ein Tippfehler (2062) gäbe eine Nummer, der keine Zahlung zugeordnet
	// wird. Die Arbeitshilfe nennt eine „Vierwochen-Frist … ab dem Briefdatum" (OFFEN.md 5.2).
	jetzt := schulzeit.Jetzt()
	heute := time.Date(jetzt.Year(), jetzt.Month(), jetzt.Day(), 0, 0, 0, 0, schulzeit.Zone())
	if !frist.After(heute) {
		//nolint:staticcheck // ST1005: ganzer Satz.
		return time.Time{}, apierrors.BadRequest("Die Frist muss nach dem heutigen Tag liegen — üblich sind vier Wochen ab Briefdatum.",
			errors.New("frist nicht nach heute"))
	}
	if frist.After(heute.AddDate(1, 0, 0)) {
		//nolint:staticcheck // ST1005: ganzer Satz.
		return time.Time{}, apierrors.BadRequest("Die Frist liegt mehr als ein Jahr in der Zukunft. Bitte das Datum prüfen — ihr Jahr geht als Kassenjahr in die Referenznummer.",
			errors.New("frist mehr als ein jahr voraus"))
	}
	return frist, nil
}

// bescheidPostenAus übersetzt die Auswahl des Dialogs: Forderungen, die es schon gibt, und
// überfällige Bücher, die mit dem Brief als Verlust gebucht werden.
func bescheidPostenAus(req BescheidErstellenRequest) ([]repository.BescheidPositionEingabe, []repository.BescheidVerlustEingabe) {
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

// istBescheidBedienfehler: Die Lage hat sich geändert, seit der Dialog offen steht.
func istBescheidBedienfehler(err error) bool {
	var nichtZugeordnet *repository.ForderungenNichtZugeordnet
	return errors.As(err, &nichtZugeordnet) ||
		errors.Is(err, repository.ErrAusleiheInzwischenZurueck) ||
		errors.Is(err, repository.ErrAusleiheSchonGemeldet) ||
		errors.Is(err, repository.ErrAusleiheFremd) ||
		errors.Is(err, repository.ErrExemplarNeuVerliehen)
}
