package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pdf"
	"bibliothek/pkg/ersatzwert"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Die Türen des Schadensersatz-Bescheids.
//
// Der Ablauf hat drei Schritte, und jeder ist ein eigener Aufruf: Der Dialog holt den
// VORSCHLAG (offene Forderungen mit Staffel-Betrag), der Mensch prüft und ERSTELLT, und
// danach lässt sich der Brief NACHDRUCKEN. Nach Fristablauf kommt die ÜBERGABE dazu.
//
// Nichts davon läuft automatisch: Jede Referenznummer wird unwiderruflich verbraucht, und
// jeder Betrag ist eine Entscheidung der Schule.

// auditBescheidErstellt und auditBescheidUebergeben sind die Aktionen im Admin-Audit-Log.
const (
	auditBescheidErstellt   = "SCHADENSERSATZ_BESCHEID_ERSTELLT"
	auditBescheidUebergeben = "SCHADENSERSATZ_BESCHEID_UEBERGEBEN"
)

// BescheidVorschlagPosition ist eine offene Forderung mit dem vorgeschlagenen Betrag und
// seiner Herleitung — der Dialog zeigt beides, damit der Mensch die Zahl beurteilen kann.
type BescheidVorschlagPosition struct {
	SchadensfallID string  `json:"schadensfall_id"`
	Art            string  `json:"art"`
	Titel          string  `json:"titel"`
	ISBN           string  `json:"isbn"`
	Betrag         float64 `json:"betrag"`
	// Herleitung: „3. Verleihjahr → 60 % von 41,50 €" bzw. der Hinweis, dass kein Preis
	// hinterlegt ist.
	Herleitung string `json:"herleitung"`
	// IstLernmittel entscheidet den Topf: Lernmittel gehen an das Land, alles andere an
	// den Schulträger. Ein Brief trägt genau einen Topf.
	IstLernmittel bool `json:"ist_lernmittel"`
}

// BescheidVorschlag ist die Antwort für den Dialog.
type BescheidVorschlag struct {
	SchuelerName string                      `json:"schueler_name"`
	Klasse       string                      `json:"klasse"`
	FristBis     string                      `json:"frist_bis"`
	Positionen   []BescheidVorschlagPosition `json:"positionen"`
	// FehlendeAngaben nennt die Einstellungen, ohne die kein Bescheid entstehen kann.
	// Der Dialog zeigt sie, statt den Knopf stumm zu sperren.
	FehlendeAngaben []string `json:"fehlende_angaben"`
}

// BescheidErstellenRequest ist die Eingabe des Dialogs.
type BescheidErstellenRequest struct {
	Mittel     string `json:"mittel"`
	FristBis   string `json:"frist_bis"`
	Positionen []struct {
		SchadensfallID string  `json:"schadensfall_id"`
		Betrag         float64 `json:"betrag"`
	} `json:"positionen"`
}

// BescheidVorschlagHandler liefert die offenen Forderungen eines Schülers mit
// Betragsvorschlag.
//
// @Summary      Proposal for a new compensation notice
// @Tags         schadensersatz
// @Produce      json
// @Param        id path string true "Student ID"
// @Success      200 {object} BescheidVorschlag
// @Router       /schueler/{id}/bescheid-vorschlag [get]
func (s *Server) BescheidVorschlagHandler(bescheidRepo repository.BescheidRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("id darf nicht leer sein", errors.New("missing id"))
		}
		ctx := r.Context()

		angaben, schule, err := s.bescheidAngaben(ctx)
		if err != nil {
			return apierrors.Internal("Einstellungen konnten nicht gelesen werden", err)
		}

		vorschlag := BescheidVorschlag{
			FristBis:        schulzeit.Jetzt().AddDate(0, 0, angaben.FristTage).Format(dateFormatISO),
			FehlendeAngaben: angaben.FehlendeAngaben(schule),
			Positionen:      []BescheidVorschlagPosition{},
		}
		empfaenger, err := bescheidRepo.EmpfaengerFuerBescheid(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apierrors.NotFound("Schüler nicht gefunden", err)
			}
			return apierrors.Internal("Schüler konnte nicht gelesen werden", err)
		}
		vorschlag.SchuelerName = strings.TrimSpace(empfaenger.Vorname + " " + empfaenger.Nachname)
		vorschlag.Klasse = empfaenger.Klasse

		offene, err := bescheidRepo.OffeneForderungen(ctx, id)
		if err != nil {
			return apierrors.Internal("Offene Forderungen konnten nicht gelesen werden", err)
		}
		for _, f := range offene {
			vorschlag.Positionen = append(vorschlag.Positionen, bescheidVorschlagAus(f))
		}

		RespondJSON(w, http.StatusOK, vorschlag)
		return nil
	})
}

// bescheidAngaben liest Einstellungen und Schul-Identität in einem Zug.
func (s *Server) bescheidAngaben(ctx context.Context) (repository.BescheidAngaben, repository.SchuleAngaben, error) {
	einst, err := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(ctx)
	if err != nil {
		return repository.BescheidAngaben{}, repository.SchuleAngaben{}, err
	}
	return repository.BescheidAngabenAus(einst), repository.SchuleAngabenAus(einst), nil
}

// bescheidVorschlagAus rechnet den Staffel-Vorschlag für eine offene Forderung.
//
// Neupreis ist 0: Das System führt heute keinen Listenpreis (Entscheidung E4 im Konzept).
// Die Herleitung sagt deshalb, dass ersatzweise mit dem Kaufpreis gerechnet wurde —
// geraten wird nichts.
func bescheidVorschlagAus(f repository.OffeneForderung) BescheidVorschlagPosition {
	v := ersatzwert.Rechne(ersatzwert.Verleihjahr(f.Ausleihen, f.JahreImBestand), f.Kaufpreis, 0)
	return BescheidVorschlagPosition{
		SchadensfallID: f.SchadensfallID,
		Art:            f.Art,
		Titel:          f.Titel,
		ISBN:           f.ISBN,
		Betrag:         v.Betrag,
		Herleitung:     bescheidHerleitung(v),
		IstLernmittel:  f.IstLernmittel,
	}
}

// bescheidHerleitung formuliert, wie der Vorschlag zustande kommt.
func bescheidHerleitung(v ersatzwert.Vorschlag) string {
	if v.BasisPreis <= 0 {
		return "kein Preis hinterlegt — Betrag bitte eintragen"
	}
	basis := "Kaufpreis"
	switch v.Basis {
	case ersatzwert.BasisNeupreis:
		basis = "Neupreis"
	case ersatzwert.BasisKaufpreisErsatzweise:
		basis = "Kaufpreis (kein Neupreis hinterlegt)"
	case ersatzwert.BasisKaufpreis:
		basis = "Kaufpreis"
	}
	return fmt.Sprintf("%d. Verleihjahr → %d %% von %s (%s)",
		v.Verleihjahr, v.Prozent, euroBetrag(v.BasisPreis), basis)
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
func (s *Server) BescheidErstellenHandler(bescheidRepo repository.BescheidRepository, auditRepo repository.AuditRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		var req BescheidErstellenRequest
		if id == "" {
			return apierrors.BadRequest("id darf nicht leer sein", errors.New("missing id"))
		}
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		if !repository.MittelGueltig(req.Mittel) {
			return apierrors.BadRequest(ErrMittelUngueltig.Error(), ErrMittelUngueltig)
		}
		// Der Brief kennt nur einen Wortlaut und ein Konto: die des Landes. Die Rechnung
		// der Schülerbücherei (Mittel des Schulträgers) ist Etappe 3 — bis dahin gäbe
		// „schultraeger" einen Landes-Bescheid mit Landeskonto für ein Buch des Trägers.
		if req.Mittel != repository.MittelLand {
			//nolint:staticcheck // ST1005: ganzer Satz — die Meldung steht so vor der Bibliothekskraft.
			return apierrors.Conflict("Ein Bescheid entsteht nur für Lernmittel des Landes. Die Rechnung für Bücher der Schülerbücherei ist noch nicht gebaut.",
				errors.New("bescheid nur für mittel=land"))
		}
		if len(req.Positionen) == 0 {
			//nolint:staticcheck // ST1005: ganzer Satz — die Meldung steht so vor der Bibliothekskraft.
			return apierrors.BadRequest("Bitte mindestens eine Forderung auswählen.", errors.New("keine positionen"))
		}
		frist, err := time.ParseInLocation(dateFormatISO, req.FristBis, schulzeit.Zone())
		if err != nil {
			//nolint:staticcheck // ST1005: ganzer Satz.
			return apierrors.BadRequest("Die Frist muss ein Datum sein (JJJJ-MM-TT).", err)
		}

		ctx := r.Context()
		angaben, schule, err := s.bescheidAngaben(ctx)
		if err != nil {
			return apierrors.Internal("Einstellungen konnten nicht gelesen werden", err)
		}
		// Ohne die Pflichtangaben entsteht KEIN Bescheid: Ein Brief ohne Referenznummer
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
			Snapshot:       snapshot,
			ErstelltVon:    bescheidAkteur(ctx),
			Referenznummer: func(nr int) string { return angaben.Referenznummer(frist.Year(), nr) },
		}
		for _, p := range req.Positionen {
			eingabe.Positionen = append(eingabe.Positionen,
				repository.BescheidPositionEingabe{SchadensfallID: p.SchadensfallID, Betrag: p.Betrag})
		}

		bescheid, err := bescheidRepo.Erstelle(ctx, eingabe)
		if err != nil {
			// Die Zuordnungs-Prüfung ist ein Bedienfehler (Forderung bezahlt, storniert
			// oder schon auf einem Brief), kein Serverfehler.
			if strings.Contains(err.Error(), "zugeordnet werden") {
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
		}, getIP(r))

		RespondJSON(w, http.StatusCreated, bescheid)
		return nil
	})
}

// bescheidSnapshotAus friert Anrede, Name und Anschrift zum Briefdatum ein.
//
// Volljährig oder nicht entscheidet über Anrede und die Zeile über dem Namen: Bei
// minderjährigen Schülern geht der Brief an die Erziehungsberechtigten, bei volljährigen
// an die Person selbst. Ohne bekanntes Geschlecht ist bei Volljährigen die neutrale
// Anrede die richtige; den Namen trägt das Anschriftfeld.
func bescheidSnapshotAus(ctx context.Context, repo repository.BescheidRepository, schuelerID string) (map[string]string, error) {
	d, err := repo.EmpfaengerFuerBescheid(ctx, schuelerID)
	if err != nil {
		return nil, err
	}
	anrede := "Sehr geehrte Erziehungsberechtigte,"
	anZeile := bescheidAnAnErzieher
	if d.Volljaehrig {
		anrede = "Sehr geehrte Damen und Herren,"
		anZeile = ""
	}
	return map[string]string{
		"anrede":   anrede,
		"an_zeile": anZeile,
		"name":     strings.TrimSpace(d.Vorname + " " + d.Nachname),
		"strasse":  strings.TrimSpace(d.Strasse + " " + d.Hausnummer),
		"plz":      d.PLZ,
		"ort":      d.Ort,
	}, nil
}

// bescheidAkteur liefert die ID des angemeldeten Bearbeiters (leer ohne Anmeldung).
func bescheidAkteur(ctx context.Context) string {
	if claims, ok := auth.GetClaims(ctx); ok {
		return claims.UserID
	}
	return ""
}

// bescheidAudit protokolliert einen Eingriff. Dasselbe Muster wie der LMF-Plan: ohne
// Anmeldung oder ohne Datenbank gibt es nichts zu schreiben.
func bescheidAudit(ctx context.Context, s *Server, auditRepo repository.AuditRepository, aktion string, details map[string]any, ip string) {
	claims, ok := auth.GetClaims(ctx)
	if !ok || s.DB == nil || s.DB.Pool == nil || auditRepo == nil {
		return
	}
	if err := auditRepo.LogAdminAktion(ctx, claims.UserID, aktion, ip, details); err != nil {
		log.Printf("Audit %s fehlgeschlagen: %v", aktion, err)
	}
}

// BescheidListeHandler liefert die Bescheide — die Arbeitsliste des Sekretariats.
//
// @Summary      List compensation notices
// @Tags         schadensersatz
// @Produce      json
// @Param        status query string false "offen = nur nicht übergebene"
// @Success      200 {array} repository.Bescheid
// @Router       /bescheide [get]
func (s *Server) BescheidListeHandler(bescheidRepo repository.BescheidRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		liste, err := bescheidRepo.Liste(r.Context(), r.URL.Query().Get("status") == "offen")
		if err != nil {
			return apierrors.Internal("Bescheide konnten nicht gelesen werden", err)
		}
		RespondJSON(w, http.StatusOK, liste)
		return nil
	})
}

// BescheidSchuelerListeHandler liefert die Bescheide eines Schülers für die Akte.
//
// @Summary      List a student's compensation notices
// @Tags         schadensersatz
// @Produce      json
// @Param        id path string true "Student ID"
// @Success      200 {array} repository.Bescheid
// @Router       /schueler/{id}/bescheide [get]
func (s *Server) BescheidSchuelerListeHandler(bescheidRepo repository.BescheidRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("id darf nicht leer sein", errors.New("missing id"))
		}
		liste, err := bescheidRepo.ZuSchueler(r.Context(), id)
		if err != nil {
			return apierrors.Internal("Bescheide konnten nicht gelesen werden", err)
		}
		RespondJSON(w, http.StatusOK, liste)
		return nil
	})
}

// BescheidPDFHandler liefert den Brief — beim ersten Mal und bei jedem Nachdruck
// derselbe, aus dem Snapshot.
//
// @Summary      Print (or reprint) a compensation notice
// @Tags         schadensersatz
// @Produce      application/pdf
// @Param        id path string true "Notice ID"
// @Success      200 {file} file
// @Router       /bescheide/{id}/pdf [get]
func (s *Server) BescheidPDFHandler(bescheidRepo repository.BescheidRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("id darf nicht leer sein", errors.New("missing id"))
		}
		ctx := r.Context()

		brief, bescheid, err := s.bescheidBrief(ctx, bescheidRepo, id)
		if err != nil {
			return err
		}
		roh, err := GenerateBescheidPDF(brief)
		if err != nil {
			return apierrors.Internal("Bescheid konnte nicht erzeugt werden", err)
		}
		if err := bescheidRepo.DruckVermerken(ctx, id); err != nil {
			// Der Vermerk ist Buchführung, nicht der Zweck: Der Brief geht trotzdem raus.
			log.Printf("Druckvermerk für Bescheid %s fehlgeschlagen: %v", id, err)
		}

		w.Header().Set("Content-Type", contentTypePDF)
		w.Header().Set("Content-Disposition",
			fmt.Sprintf(`attachment; filename="bescheid_%s.pdf"`,
				strings.ReplaceAll(bescheid.Referenznummer, " ", "-")))
		if _, err := w.Write(roh); err != nil {
			log.Printf("Bescheid-PDF senden: %v", err)
		}
		return nil
	})
}

// bescheidBrief baut das Blatt aus Bescheid, Snapshot, Positionen und Einstellungen.
func (s *Server) bescheidBrief(ctx context.Context, bescheidRepo repository.BescheidRepository, id string) (BescheidBrief, *repository.Bescheid, error) {
	bescheid, err := bescheidRepo.Lies(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BescheidBrief{}, nil, apierrors.NotFound("Bescheid nicht gefunden", err)
		}
		return BescheidBrief{}, nil, apierrors.Internal("Bescheid konnte nicht gelesen werden", err)
	}
	snapshot, err := bescheidRepo.Snapshot(ctx, id)
	if err != nil {
		return BescheidBrief{}, nil, apierrors.Internal("Empfänger konnte nicht gelesen werden", err)
	}
	positionen, err := bescheidRepo.Positionen(ctx, id)
	if err != nil {
		return BescheidBrief{}, nil, apierrors.Internal("Positionen konnten nicht gelesen werden", err)
	}
	angaben, schule, err := s.bescheidAngaben(ctx)
	if err != nil {
		return BescheidBrief{}, nil, apierrors.Internal("Einstellungen konnten nicht gelesen werden", err)
	}

	brief := BescheidBrief{
		Schule: pdf.SchuleInfo{
			Name: schule.Name, Strasse: schule.Strasse, PLZ: schule.PLZ, Ort: schule.Ort,
		},
		Empfaenger:        bescheidEmpfaengerAus(snapshot),
		Geschaeftszeichen: angaben.Geschaeftszeichen,
		Bearbeiter:        angaben.Bearbeiter,
		Durchwahl:         angaben.Durchwahl,
		BriefDatum:        bescheid.BriefDatum,
		FristBis:          bescheid.FristBis,
		Gesamtbetrag:      bescheid.Gesamtbetrag,
		Zahlstelle:        angaben.Zahlstelle,
		Bankverbindung:    angaben.Bankverbindung,
		Referenznummer:    bescheid.Referenznummer,
		Aufsicht:          angaben.Aufsicht,
		Schulleitung:      angaben.Schulleitung,
	}
	name := brief.Empfaenger.Name
	for _, p := range positionen {
		zeile := BescheidPosition{SchuelerName: name, Titel: p.Titel, ISBN: p.ISBN, Betrag: p.Betrag}
		if p.Art == "nicht_zurueckgegeben" {
			brief.NichtZurueckgegeben = append(brief.NichtZurueckgegeben, zeile)
			continue
		}
		brief.Beschaedigt = append(brief.Beschaedigt, zeile)
	}
	return brief, bescheid, nil
}

// bescheidEmpfaengerAus baut den Empfänger aus dem Snapshot. Ist er leer (Anonymisierung),
// sagt der Brief das — ein Nachdruck mit erfundener Anschrift wäre schlimmer als keiner.
func bescheidEmpfaengerAus(snapshot map[string]string) BescheidEmpfaenger {
	if len(snapshot) == 0 {
		return BescheidEmpfaenger{
			Anrede: "Sehr geehrte Damen und Herren,",
			Name:   "(Empfängerangaben nach Löschfrist getilgt)",
		}
	}
	return BescheidEmpfaenger{
		Anrede:  snapshot["anrede"],
		AnZeile: snapshot["an_zeile"],
		Name:    snapshot["name"],
		Strasse: snapshot["strasse"],
		PLZ:     snapshot["plz"],
		Ort:     snapshot["ort"],
	}
}

// BescheidUebergebenHandler setzt einen Bescheid nach Fristablauf auf „übergeben".
//
// Kein Cron: Die Übergabe ist eine Handlung des Sekretariats (Original und Buchungsbeleg
// gehen aus dem Haus). Eine Automatik über Nacht wäre eine stille Zustandsänderung.
//
// @Summary      Mark a notice as handed over after the deadline
// @Tags         schadensersatz
// @Produce      json
// @Param        id path string true "Notice ID"
// @Success      200 {object} map[string]any
// @Router       /bescheide/{id}/uebergeben [post]
func (s *Server) BescheidUebergebenHandler(bescheidRepo repository.BescheidRepository, auditRepo repository.AuditRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("id darf nicht leer sein", errors.New("missing id"))
		}
		ctx := r.Context()
		if err := bescheidRepo.Uebergebe(ctx, id); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Entweder gibt es den Bescheid nicht, oder sein Zustand passt nicht.
				// Die Meldung nennt beides, statt einen 404 zu behaupten.
				return apierrors.Conflict(
					"Übergabe nicht möglich: Der Bescheid ist nicht (mehr) offen oder seine Frist läuft noch.",
					err)
			}
			return apierrors.Internal("Übergabe konnte nicht gebucht werden", err)
		}

		bescheidAudit(ctx, s, auditRepo, auditBescheidUebergeben, map[string]any{"bescheid_id": id}, getIP(r))
		RespondJSON(w, http.StatusOK, map[string]any{"id": id, "status": "uebergeben"})
		return nil
	})
}
