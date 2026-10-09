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
	"bibliothek/pkg/httpresp"
	"bibliothek/pkg/leserart"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// pruefeSchuelerLoeschbar prüft, ob ein Schüler gelöscht werden darf. Rückgabe
// (0, nil) bedeutet löschbar; andernfalls der passende HTTP-Status samt Fehler.
func (s *Server) pruefeSchuelerLoeschbar(ctx context.Context, id string) (int, error) {
	// `leser` statt der Sicht `schueler`: Die zeigt nur Schüler, und ein Kollege kam hier
	// als „nicht gefunden" zurück, bevor auch nur eine Regel geprüft war.
	studentExists, err := repository.LeserVorhanden(ctx, s.DB.Pool, id)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if !studentExists {
		return http.StatusNotFound, errors.New("leser nicht gefunden")
	}

	hasActiveLoans, err := repository.LeserHatOffeneAusleihen(ctx, s.DB.Pool, id)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if hasActiveLoans {
		return http.StatusBadRequest, errors.New("löschen nicht möglich: Dieser Leser hat noch entliehene Bücher")
	}

	hasUnpaidDamages, err := repository.LeserHatUnbezahlteForderungen(ctx, s.DB.Pool, id)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if hasUnpaidDamages {
		return http.StatusBadRequest, errors.New("löschen nicht möglich: Dieser Leser hat noch unbezahlte Schadensfälle/Gebühren")
	}

	return 0, nil
}

// DeleteStudentHandler deletes a student after checking for outstanding loans and unpaid damage cases, logging it to the audit trail.
// @Summary      Delete student
// @Description  Transactionally deletes a student from the system, checks for active loans or unpaid damage fees, anonymizes historical loans, and writes to audit_log.
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Student ID (UUID)"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /schueler/{id} [delete]
func (s *Server) DeleteStudentHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("missing session information"))
			return
		}

		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("fehlende Schüler-ID"))
			return
		}

		ctx := r.Context()

		// Niemand löscht seine eigene Leserzeile. Beim Kollegium fiele damit auch das
		// eigene Konto (DeleteStudent) — der Löschende spielte sich mitten im Vorgang
		// selbst aus der Anmeldung. Die Benutzerverwaltung hält dieselbe Regel für Konten
		// (DeleteUserHandler, „eigenes Konto kann nicht gelöscht werden").
		eigene, err := repository.LeserIDVonKonto(ctx, s.DB.Pool, claims.UserID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if eigene != "" && eigene == id {
			apierrors.SendHTTPError(w, http.StatusForbidden,
				//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung in der Akte
				errors.New("Der eigene Eintrag lässt sich nicht löschen — mit ihm fiele der eigene Zugang."))
			return
		}

		if status, err := s.pruefeSchuelerLoeschbar(ctx, id); err != nil {
			apierrors.SendHTTPError(w, status, err)
			return
		}

		// Transaktionales Löschen mit Audit-Log
		if err := auditRepo.DeleteStudent(ctx, id, claims.UserID, "Manuelle Löschung"); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Admin audit log
		// Schlüssel `schueler_id` wie überall: Auskunft und Tilgung fragen genau diesen
		// Schlüssel ab (Migration 091 hat die alte `student_id`-Form umgeschlüsselt).
		details := fmt.Sprintf(`{"schueler_id":"%s"}`, id)
		s.schreibeAdminProtokoll(ctx, claims.UserID, "DELETE_STUDENT", getIP(r), details)

		RespondJSON(w, http.StatusOK, map[string]any{
			"status": "success",
		})
	}
}

// parseGeburtsdatum parst ein optionales ISO-Datum. Leerstring ergibt (nil, nil)
// und setzt das Feld damit auf NULL.
func parseGeburtsdatum(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(dateFormatISO, raw)
	if err != nil {
		return nil, fmt.Errorf("ungültiges Datumsformat für Geburtsdatum: %q — erwartet YYYY-MM-DD", raw)
	}
	return &t, nil
}

// PatchStudentHandler aktualisiert editierbare Felder eines Schülers (klasse, abgaenger_jahr).
// Wird nun auch für das Bearbeiten aller Stammdaten in der UI genutzt.
func (s *Server) PatchStudentHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.handlePatchStudent(w, r, auditRepo)
	}
}

// handlePatchStudent prüft die Änderung, schreibt sie und trägt nach, was erst nach dem
// Schreiben entstehen darf.
func (s *Server) handlePatchStudent(w http.ResponseWriter, r *http.Request, auditRepo repository.AuditRepository) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("Sitzungs-Information fehlt oder ist abgelaufen"))
		return
	}

	id := r.PathValue("id")
	if id == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("fehlende Schüler-ID"))
		return
	}

	var req patchStudentRequest
	if !DecodeAndValidate(w, r, &req) {
		return
	}

	ctx := r.Context()
	aenderung, ok := s.pruefeSchuelerAenderung(ctx, w, id, &req)
	if !ok {
		return
	}
	if !s.fuehreSchuelerUpdateAus(ctx, w, id, aenderung.update) {
		return
	}
	if !s.trageNachUndProtokolliere(w, r, auditRepo, claims, id, aenderung) {
		return
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	response := map[string]any{"status": "success"}
	if req.AbgaengerJahr != nil {
		response["abgaenger_jahr"] = *req.AbgaengerJahr
	}
	httpresp.Encode(w, response)
}

// schuelerAenderung ist eine geprüfte Änderung an der Leserzeile samt dem, was nach dem
// Schreiben noch aussteht.
type schuelerAenderung struct {
	update *repository.LeserAenderung
	// lusdNachgetragen ist die LUSD-ID, die diese Änderung einträgt; sie gehört ins Protokoll.
	lusdNachgetragen string
	// kontoAdresse ist die Schul-E-Mail, für die nach dem Schreiben ein Konto entsteht.
	kontoAdresse string
}

// pruefeSchuelerAenderung baut die Zuweisungen aus der Anfrage und prüft jedes Feld, das
// eigene Regeln hat. false heißt: Die Ablehnung ist schon beantwortet.
func (s *Server) pruefeSchuelerAenderung(ctx context.Context, w http.ResponseWriter, id string, req *patchStudentRequest) (schuelerAenderung, bool) {
	var keine schuelerAenderung
	b, ok := baueSchuelerUpdate(w, req)
	if !ok {
		return keine, false
	}
	// Die LUSD-ID lässt sich nur nachtragen: wenn sie bisher leer war und der neue Wert
	// eindeutig ist.
	lusdNachgetragen, ok := s.pruefeUndSetzeLusdID(ctx, w, id, req.LusdID, b)
	if !ok {
		return keine, false
	}
	if !s.pruefeUndSetzeArt(ctx, w, id, req.Art, b) {
		return keine, false
	}
	// Die Schul-E-Mail wird vor dem Schreiben geprüft und nach ihm eingetragen: Das Konto
	// soll nicht an einer Leserzeile hängen, deren Änderung gescheitert ist. Eine Adresse
	// kommt nur zurück, wenn es noch kein Konto gibt und sie gültig ist.
	kontoAdresse, ok := s.pruefeSchulEmail(ctx, w, id, req.Email, req.Art)
	if !ok {
		return keine, false
	}
	if !s.pruefeAusweisLeerung(ctx, w, id, req.BarcodeID) {
		return keine, false
	}
	// Ohne Zuweisung läuft kein UPDATE, etwa wenn nur die unveränderte LUSD-ID mitkam. Eine
	// nachzutragende Adresse ist Arbeit, auch wenn an der Leserzeile nichts steht; sonst
	// wäre „nur die Schul-E-Mail nachtragen" ein 400.
	if b.Leer() && kontoAdresse == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("keine zu aktualisierenden Felder angegeben"))
		return keine, false
	}
	return schuelerAenderung{update: b, lusdNachgetragen: lusdNachgetragen, kontoAdresse: kontoAdresse}, true
}

// trageNachUndProtokolliere legt nach dem Schreiben das Konto zur Schul-E-Mail an und
// schreibt die Protokolleinträge der Nachträge. false heißt: Die Ablehnung ist schon
// beantwortet.
func (s *Server) trageNachUndProtokolliere(w http.ResponseWriter, r *http.Request, auditRepo repository.AuditRepository, claims *auth.Claims, id string, a schuelerAenderung) bool {
	ctx := r.Context()
	if a.kontoAdresse != "" {
		// Das Konto entsteht immer, aktiv nur bei manage_users — dieselbe Paarung wie beim
		// Anlegen (student_create.go). Wer hier nur edit_students hat, bekäme sonst still
		// das Recht, Zugänge freizuschalten.
		kontoID, ok := s.trageKontoNach(ctx, w, id, a.kontoAdresse, s.BesitztRecht(r, "manage_users"))
		if !ok {
			return false
		}
		// ziel_id ist die Kennung des Kontos: Daran findet die Auskunft den Eintrag beim Konto,
		// auch nachdem es gelöscht ist. Mit dem Leser fällt sie wieder (Tilgung).
		if logErr := auditRepo.LogAdminAktion(ctx, claims.UserID, "KOLLEGIUMSKONTO_NACHGETRAGEN", getIP(r), map[string]any{
			"schueler_id": id,
			"ziel_id":     kontoID,
		}); logErr != nil {
			log.Printf("Audit für Kontonachtrag fehlgeschlagen: %v", logErr)
		}
	}

	if a.lusdNachgetragen != "" {
		if logErr := auditRepo.LogAdminAktion(ctx, claims.UserID, "LUSD_ID_NACHGETRAGEN", getIP(r), map[string]any{
			"schueler_id": id,
			"lusd_id":     a.lusdNachgetragen,
		}); logErr != nil {
			log.Printf("Audit für LUSD-ID-Nachtrag fehlgeschlagen: %v", logErr)
		}
	}
	return true
}

// pruefeUndSetzeLusdID trägt die LUSD-ID kontrolliert nach. Die LUSD-ID ist der
// einzige Zuordnungsschlüssel des Landesabgleichs (kein Adress-/Geburtsdatum-Fallback
// im Import); wer sie roh überschreibt, kann still die falsche Identität verknüpfen.
// Regeln: nur NACHTRAGBAR, wenn bisher leer (Waise adoptieren) — ein bestehender Wert
// wird nicht überschrieben und nicht geleert; der neue Wert muss eindeutig sein. Der
// automatische Gegenpart ist das Import-Auto-Matching (Name+Geburtsdatum).
//
// Rückgabe: (nachgetragenerWert, ok). nachgetragenerWert != "" heißt: bitte auditieren.
// Ist req nil oder ein No-op (gleicher Wert), wird ("", true) zurückgegeben.
func (s *Server) pruefeUndSetzeLusdID(ctx context.Context, w http.ResponseWriter, id string, reqLusd *string, b *repository.LeserAenderung) (string, bool) {
	if reqLusd == nil {
		return "", true
	}
	neu := strings.TrimSpace(*reqLusd)

	aktuell, err := repository.LusdIDDesLesers(ctx, s.DB.Pool, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("schüler nicht gefunden"))
			return "", false
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return "", false
	}

	if neu == aktuell {
		return "", true // No-op: Formular schickt den unveränderten Wert mit.
	}
	if aktuell != "" {
		// Bestehende Verknüpfung: weder überschreiben noch leeren.
		apierrors.SendHTTPError(w, http.StatusForbidden,
			errors.New("die LUSD-ID dieses Schülers ist bereits gesetzt und kann über das Formular nicht geändert oder geleert werden"))
		return "", false
	}
	if neu == "" {
		return "", true // war leer, bleibt leer.
	}

	// Eindeutigkeit VOR dem Schreiben prüfen, damit statt eines 500 am partiellen
	// Unique-Index (uniq_schueler_lusd_id_active) eine klare 409 zurückkommt.
	belegt, err := repository.LusdIDBeiAnderemSchueler(ctx, s.DB.Pool, neu, id)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return "", false
	}
	if belegt {
		apierrors.SendHTTPError(w, http.StatusConflict,
			errors.New("diese LUSD-ID ist bereits einem anderen aktiven Schüler zugeordnet"))
		return "", false
	}

	b.LusdID = &neu
	return neu, true
}

// pruefeUndSetzeArt aendert die Art eines Lesers (Migration 123). Erlaubt ist der Wechsel
// innerhalb des Kollegiums: Lehrkraft, LiV, Praktikum, Sekretariat, U-plus, Fachbereich
// (Migration 153). Beide Richtungen ueber die Grenze zum Schueler sind zu — und zwar nicht
// aus Vorsicht, sondern weil jede von ihnen einen echten Schaden anrichtet:
//
//	Schueler -> Kollege: Die Zeile verliert damit ihre LUSD-Bindung. Traegt sie eine
//	  lusd_id, bricht chk_leser_nur_schueler_werden_abgaenger; traegt sie keine, waere
//	  der Schueler beim naechsten Import ein unbekannter Name und liefe als Abgaenger
//	  samt Anonymisierung durch. Absprache vom 16.09.2026: "ein Schueler kann nie ein Lehrer
//	  werden!"
//	Kollege -> Schueler: chk_leser_schueler_pflichtfelder verlangt Klasse, Abgaengerjahr
//	  UND Ausweisnummer. Ein Kollege hat die ersten beiden nicht; das UPDATE liefe in
//	  den CHECK und damit in eine 500.
//
// Ein Mensch wechselt die Seite nicht. Wer wirklich falsch angelegt wurde, wird
// geloescht und neu angelegt — ein sichtbarer Vorgang statt einer stillen Umwidmung.
//
// nil heisst "nicht mitgeschickt"; derselbe Wert ist ein No-op, damit das Formular
// die Art unveraendert mitschicken darf.
//
// Ein bestehendes Konto bleibt beim Wechsel stehen, auch zu Praktikum oder Fachbereich: Neu
// angelegt wird dort keines (leserart.MitKonto), ueber ein vorhandenes entscheidet die
// Benutzerverwaltung.
func (s *Server) pruefeUndSetzeArt(ctx context.Context, w http.ResponseWriter, id string, reqArt *string, b *repository.LeserAenderung) bool {
	if reqArt == nil {
		return true
	}
	neu := strings.TrimSpace(*reqArt)
	// Dieselbe Menge wie beim Anlegen (leserart.Bekannt) und dieselbe wie
	// chk_leser_art in der Datenbank. Eine unbekannte Art ist ein Tippfehler, kein neuer
	// Personenkreis — und soll als Auskunft zurueckkommen, nicht als CHECK-500.
	if !leserart.Bekannt(neu) {
		//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Formular
		apierrors.SendHTTPError(w, http.StatusBadRequest,
			fmt.Errorf("Unbekannte Art %q. Möglich sind: %s.", neu, leserart.Moegliche()))
		return false
	}

	// Gelesen wird die Tabelle leser, nicht die Sicht schueler: Ein Kollege steht nicht in der
	// Sicht.
	aktuell, err := repository.LeserArt(ctx, s.DB.Pool, id)
	if err != nil {
		if errors.Is(err, repository.ErrLeserNichtGefunden) {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("leser nicht gefunden"))
			return false
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return false
	}

	if neu == aktuell {
		return true // No-op: Formular schickt den unveraenderten Wert mit.
	}
	if leserart.IstSchueler(aktuell) || leserart.IstSchueler(neu) {
		//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Formular
		apierrors.SendHTTPError(w, http.StatusBadRequest,
			errors.New("Aus einem Schüler wird kein Kollege und umgekehrt. "+
				"Lege die Person neu an und lösche den falschen Eintrag."))
		return false
	}

	b.Art = &neu
	return true
}

// patchStudentRequest bündelt die optional aktualisierbaren Stammdatenfelder (nil = unverändert).
type patchStudentRequest struct {
	Vorname       *string `json:"vorname"`
	Nachname      *string `json:"nachname"`
	Klasse        *string `json:"klasse"`
	LusdID        *string `json:"lusd_id"`
	BarcodeID     *string `json:"barcode_id"`
	AbgaengerJahr *int    `json:"abgaenger_jahr"`
	Geburtsdatum  *string `json:"geburtsdatum"`
	// Art des Lesers (pkg/leserart, Migration 123 und 153). Sie steht hier, weil die Akte
	// sie zeigen und ein Kollege innerhalb des Kollegiums wechseln koennen muss. Sie hat einen eigenen kontrollierten Pfad (pruefeUndSetzeArt) und liegt
	// NICHT im generischen Feld-Beutel: Ein Wechsel der Art verschiebt die Zeile
	// zwischen zwei Pflichtfeld-Welten (chk_leser_schueler_pflichtfelder,
	// chk_leser_nur_schueler_werden_abgaenger) und liefe roh in einen CHECK — also in
	// eine 500, die der Sanitizer zu "interner Datenbankfehler" macht.
	Art *string `json:"art"`
	// KEINE Sperrfelder hier. Sperren und Entsperren läuft ausschliesslich über
	// PATCH /api/admin/students/{id}/lock (api/student_lock.go) — und das aus zwei
	// Gründen, die dieser Weg beide nicht erfüllte:
	//
	//   - Der Sperr-Endpunkt verlangt einen GRUND. Über diesen PATCH gesetzt, lief eine
	//     Sperre ohne Grund in chk_schueler_block_reason und kam als 500 „Ein interner
	//     Datenbankfehler ist aufgetreten" zurück (der Sanitizer ersetzt jede 500-Meldung).
	//     Derselbe Vorgang, eine Tür weiter: 400 mit dem Satz, was zu tun ist.
	//   - Beim ENTSPERREN räumt der Sperr-Endpunkt den Grund nur weg, wenn keine
	//     Systemsperre mehr besteht. Dieser PATCH liess ihn schlicht stehen.
	//
	// Zwei Türen zu demselben Zustand, von denen nur eine die Regeln kennt, sind keine
	// Bequemlichkeit — die falsche Tür geht irgendwann auf.
	Strasse     *string `json:"strasse"`
	Hausnummer  *string `json:"hausnummer"`
	Plz         *string `json:"plz"`
	Ort         *string `json:"ort"`
	ElternEmail *string `json:"eltern_email"`
	// Email ist die SCHUL-Adresse eines Kollegen mit Zugang und steht nicht in `leser`,
	// sondern am Konto (benutzer.email). Sie liegt deshalb ebenfalls nicht im
	// generischen Feld-Beutel, sondern hat ihren eigenen Pfad (pruefeSchulEmail,
	// api/student_schul_email.go): nachtragbar, solange keine da ist — dann entsteht
	// das Konto; steht eine da, ist sie eine Anzeige und wird in der
	// Benutzerverwaltung geändert.
	Email *string `json:"email"`
}

// baueSchuelerUpdate prüft die Felder der Anfrage und nennt sie als Änderung der Leserzeile
// (samt Ableitung des Abgängerjahrs aus der Klasse und dem gelesenen Geburtsdatum). Welche
// Spalte ein Feld setzt, steht in repository.LeserAenderung. ok=false: Die Ablehnung ist
// schon beantwortet.
func baueSchuelerUpdate(w http.ResponseWriter, req *patchStudentRequest) (*repository.LeserAenderung, bool) {
	// Die Sperre des Spezialwerts 'lehrer' gilt auch für die zweite Tür (PATCH) —
	// siehe pruefeKlassenname und Migration 072.
	if req.Klasse != nil {
		if err := pruefeKlassenname(*req.Klasse); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return nil, false
		}
	}
	// Bei Klassenänderung ohne explizites Abgängerjahr dieses automatisch ableiten.
	if req.Klasse != nil && req.AbgaengerJahr == nil {
		newJahr := repository.AbgaengerJahr(*req.Klasse)
		req.AbgaengerJahr = &newJahr
	}

	// Pflichtfelder lassen sich nicht über den PATCH wegräumen.
	//
	// Beim ANLEGEN sind vorname/nachname/klasse `validate:"required"` (student_create.go);
	// hier waren sie es nicht, und ein leerer String kam mit 200 durch — der Schüler
	// verlor Namen, Klasse und Ausweisnummer, die Oberfläche meldete "Änderungen
	// gespeichert". Bei der Klasse kam ein zweiter Schaden dazu: Aus dem leeren Namen
	// leitete repository.AbgaengerJahr noch ein Abgängerjahr ab.
	//
	// Dass das bis zum 23.08.2026 nie passierte, war Zufall und keine Regel: Das
	// Formular schickte geräumte Felder als JSON-null, und null landet im *string als
	// nil ("nicht mitgeschickt"). Dieselbe Zufälligkeit hielt die Löschung der
	// Postanschrift auf — sie kam ebenfalls nie an.
	//
	// Die Ausweisnummer steht NICHT mehr in dieser Liste: Ihre Pflicht ist an die Art
	// gepaart, wie in der Datenbank (chk_leser_schueler_pflichtfelder), und die Art steht
	// hier nicht fest. Ein Schüler ohne Nummer ist an der Theke unauffindbar; ein Kollege
	// muss eine falsch eingetragene wieder loswerden können — mit aktivem Konto bekommt er
	// dabei eine neue aus dem Generator (Migration 145). Geprüft wird sie deshalb im Handler
	// an der wirklichen Art (pruefeAusweisLeerung).
	for _, feld := range []struct {
		bezeichnung string
		wert        *string
	}{
		{"Vorname", req.Vorname},
		{"Nachname", req.Nachname},
		{"Klasse", req.Klasse},
	} {
		if feld.wert != nil && strings.TrimSpace(*feld.wert) == "" {
			//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Formular
			apierrors.SendHTTPError(w, http.StatusBadRequest,
				errors.New(feld.bezeichnung+" darf nicht leer sein."))
			return nil, false
		}
	}

	// LusdID und Art stehen nicht hier: Beide haben einen eigenen geprüften Pfad
	// (pruefeUndSetzeLusdID, pruefeUndSetzeArt) und kämen sonst ungeprüft in die Zeile.
	// Anschrift und Elternkontakt sind die Angaben, deren Entfernung jemand verlangen kann:
	// Ein mitgeschicktes leeres Feld löscht sie, ebenso die Ausweisnummer.
	b := &repository.LeserAenderung{
		Vorname:       req.Vorname,
		Nachname:      req.Nachname,
		Ausweisnummer: req.BarcodeID,
		Klasse:        req.Klasse,
		AbgaengerJahr: req.AbgaengerJahr,
		Strasse:       req.Strasse,
		Hausnummer:    req.Hausnummer,
		Plz:           req.Plz,
		Ort:           req.Ort,
		ElternEmail:   req.ElternEmail,
	}

	if req.Geburtsdatum != nil {
		// Pflichtfeld seit 21.08.2026 (Schlüssel des LUSD-Abgleichs): POST verlangt es,
		// PATCH durfte es bis 22.08. mit "" still auf NULL setzen — zweite Tür, gleiche
		// Regel (Prüfung 22.08., B). Leer = 400, nicht blanken.
		if strings.TrimSpace(*req.Geburtsdatum) == "" {
			//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Formular
			apierrors.SendHTTPError(w, http.StatusBadRequest,
				errors.New("Geburtsdatum kann nicht geleert werden — es ist der Schlüssel für den LUSD-Abgleich"))
			return nil, false
		}
		parsedDate, err := parseGeburtsdatum(*req.Geburtsdatum)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return nil, false
		}
		b.Geburtsdatum = parsedDate
	}

	// Ob die Änderung leer ist, prüft der Handler nach pruefeUndSetzeLusdID: Eine nachgetragene
	// LUSD-ID ist eine gültige Änderung, und ihr Feld nennt erst dieser Pfad.
	return b, true
}

// fuehreSchuelerUpdateAus schreibt die Änderung an die Leserzeile. ok=false: Die Ablehnung
// (409 bei einer vergebenen Nummer oder einem vergebenen Namen, 404 bei unbekanntem Leser,
// sonst 500) ist schon beantwortet.
func (s *Server) fuehreSchuelerUpdateAus(ctx context.Context, w http.ResponseWriter, id string, b *repository.LeserAenderung) bool {
	gefunden, err := repository.AendereLeser(ctx, s.DB.Pool, id, *b)
	if err != nil {
		// Eine vergebene Ausweisnummer (unter den Schülern oder im Kollegium, Migration 118) ist
		// eine Auskunft, kein Serverfehler.
		if repository.IstAusweisKollision(err) {
			apierrors.SendHTTPError(w, http.StatusConflict,
				errors.New("diese Ausweisnummer trägt bereits eine andere Person (Schüler oder Kollegium)"))
			return false
		}
		if repository.IstNummerBuchOderAusweisKollision(err) {
			apierrors.SendHTTPError(w, http.StatusConflict,
				errors.New("diese Nummer ist der Barcode eines Buchs und kann kein Ausweis sein"))
			return false
		}
		// Name und Geburtsdatum eines anderen Lesers: dieselbe Auskunft wie beim Anlegen.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "unique_schueler_name_gebdatum" {
			apierrors.SendHTTPError(w, http.StatusConflict, errors.New(meldungSchuelerDuplikat))
			return false
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return false
	}
	if !gefunden {
		apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("leser nicht gefunden"))
		return false
	}
	return true
}
