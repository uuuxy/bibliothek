package leser

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/httpresp"
	"bibliothek/pkg/leserart"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// pruefeLeserLoeschbar prüft, ob ein Schüler gelöscht werden darf. Rückgabe
// (0, nil) bedeutet löschbar; andernfalls der passende HTTP-Status samt Fehler.
func (t *Tueren) pruefeLeserLoeschbar(ctx context.Context, id string) (int, error) {
	// `leser` statt der Sicht `schueler`: Die zeigt nur Schüler, und ein Kollege kam hier
	// als „nicht gefunden" zurück, bevor auch nur eine Regel geprüft war.
	leserVorhanden, err := repository.IstLeserVorhanden(ctx, t.DB.Pool, id)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if !leserVorhanden {
		return http.StatusNotFound, apierrors.NotFound("Leser nicht gefunden", nil)
	}

	hatOffeneAusleihen, err := repository.HatLeserOffeneAusleihen(ctx, t.DB.Pool, id)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if hatOffeneAusleihen {
		return http.StatusBadRequest, errors.New("löschen nicht möglich: Dieser Leser hat noch entliehene Bücher")
	}

	hatOffeneSchaeden, err := repository.HatLeserUnbezahlteForderungen(ctx, t.DB.Pool, id)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if hatOffeneSchaeden {
		return http.StatusBadRequest, errors.New("löschen nicht möglich: Dieser Leser hat noch unbezahlte Schadensfälle/Gebühren")
	}

	return 0, nil
}

// LeserLoeschen legt einen Leser in den Papierkorb, wenn er keine offenen Ausleihen und keine
// offenen Schäden hat, und schreibt den Vorgang ins Protokoll.
// @Summary      Leser in den Papierkorb legen
// @Description  Legt den Leser in den Papierkorb (gelöscht und gesperrt, die Daten bleiben), wenn er keine offenen Ausleihen und keine offenen Schäden hat, und schreibt den Vorgang ins Protokoll. Endgültig löscht LeserEndgueltigLoeschen.
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Kennung des Lesers (UUID)"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /schueler/{id} [delete]
func (t *Tueren) LeserLoeschen(protokollAbfragen repository.ProtokollAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsAusKontext(r.Context())
		if !ok {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, auth.MeldungNichtAngemeldet, nil)
			return
		}

		id := r.PathValue("id")
		if id == "" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung des Lesers fehlt", nil)
			return
		}

		ctx := r.Context()

		// Niemand löscht seine eigene Leserzeile. Beim Kollegium fiele damit auch das
		// eigene Konto (LegeLeserInPapierkorb) — der Löschende spielte sich mitten im Vorgang
		// selbst aus der Anmeldung. Die Benutzerverwaltung hält dieselbe Regel für Konten
		// (benutzer.Tueren.KontoLoeschen, „eigenes Konto kann nicht gelöscht werden").
		eigene, err := repository.LadeLeserIDVonKonto(ctx, t.DB.Pool, claims.KontoID)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		if eigene != "" && eigene == id {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusForbidden,
				"Der eigene Eintrag lässt sich nicht löschen. Mit ihm fiele der eigene Zugang.", nil)
			return
		}

		if status, err := t.pruefeLeserLoeschbar(ctx, id); err != nil {
			apierrors.AntworteFehler(w, status, err)
			return
		}

		// In den Papierkorb, in einer Transaktion mit dem Eintrag im Protokoll.
		if err := protokollAbfragen.LegeLeserInPapierkorb(ctx, id, claims.KontoID, "Manuelle Löschung"); err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		// Protokoll der Verwaltung
		// Schlüssel `schueler_id` wie überall: Auskunft und Tilgung fragen genau diesen
		// Schlüssel ab (Migration 091 hat die alte `student_id`-Form umgeschlüsselt).
		details := fmt.Sprintf(`{"schueler_id":"%s"}`, id)
		t.SchreibeAdminProtokoll(ctx, claims.KontoID, "DELETE_STUDENT", clientip.AusAnfrage(r), details)

		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"status": "success",
		})
	}
}

// leseGeburtsdatum parst ein optionales ISO-Datum. Leerstring ergibt (nil, nil)
// und setzt das Feld damit auf NULL.
func leseGeburtsdatum(roh string) (*time.Time, error) {
	if roh == "" {
		return nil, nil
	}
	t, err := time.Parse(tuer.DatumsformatISO, roh)
	if err != nil {
		return nil, fmt.Errorf("ungültiges Datumsformat für Geburtsdatum: %q — erwartet YYYY-MM-DD", roh)
	}
	return &t, nil
}

// LeserAendern ändert die Stammdaten eines Lesers; ein fehlendes Feld bleibt, wie es ist.
func (t *Tueren) LeserAendern(protokollAbfragen repository.ProtokollAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.aendereLeser(w, r, protokollAbfragen)
	}
}

// aendereLeser prüft die Änderung, schreibt sie und trägt nach, was erst nach dem
// Schreiben entstehen darf.
func (t *Tueren) aendereLeser(w http.ResponseWriter, r *http.Request, protokollAbfragen repository.ProtokollAbfragen) {
	claims, ok := auth.ClaimsAusKontext(r.Context())
	if !ok {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, auth.MeldungNichtAngemeldet, nil)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung des Lesers fehlt", nil)
		return
	}

	var req leserAendernAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return
	}

	ctx := r.Context()
	aenderung, ok := t.pruefeLeserAenderung(ctx, w, id, &req)
	if !ok {
		return
	}
	if !t.schreibeLeserAenderung(ctx, w, id, aenderung.update) {
		return
	}
	if !t.trageNachUndProtokolliere(w, r, protokollAbfragen, claims, id, aenderung) {
		return
	}

	w.Header().Set(tuer.HeaderContentType, tuer.ContentTypeJSON)
	antwort := map[string]any{"status": "success"}
	if req.AbgaengerJahr != nil {
		antwort["abgaenger_jahr"] = *req.AbgaengerJahr
	}
	httpresp.SchreibeJSON(w, antwort)
}

// leserAenderung ist eine geprüfte Änderung an der Leserzeile samt dem, was nach dem
// Schreiben noch aussteht.
type leserAenderung struct {
	update *repository.LeserAenderung
	// lusdNachgetragen ist die LUSD-ID, die diese Änderung einträgt; sie gehört ins Protokoll.
	lusdNachgetragen string
	// kontoAdresse ist die Schul-E-Mail, für die nach dem Schreiben ein Konto entsteht.
	kontoAdresse string
}

// pruefeLeserAenderung baut die Zuweisungen aus der Anfrage und prüft jedes Feld, das
// eigene Regeln hat. false heißt: Die Ablehnung ist schon beantwortet.
func (t *Tueren) pruefeLeserAenderung(ctx context.Context, w http.ResponseWriter, id string, req *leserAendernAnfrage) (leserAenderung, bool) {
	var keine leserAenderung
	b, ok := baueLeserAenderung(w, req)
	if !ok {
		return keine, false
	}
	// Die LUSD-ID lässt sich nur nachtragen: wenn sie bisher leer war und der neue Wert
	// eindeutig ist.
	lusdNachgetragen, ok := t.pruefeUndSetzeLUSDID(ctx, w, id, req.LUSDID, b)
	if !ok {
		return keine, false
	}
	if !t.pruefeUndSetzeArt(ctx, w, id, req.Art, b) {
		return keine, false
	}
	// Die Schul-E-Mail wird vor dem Schreiben geprüft und nach ihm eingetragen: Das Konto
	// soll nicht an einer Leserzeile hängen, deren Änderung gescheitert ist. Eine Adresse
	// kommt nur zurück, wenn es noch kein Konto gibt und sie gültig ist.
	kontoAdresse, ok := t.pruefeSchulEmail(ctx, w, id, req.Email, req.Art)
	if !ok {
		return keine, false
	}
	if !t.pruefeAusweisLeerung(ctx, w, id, req.BarcodeID) {
		return keine, false
	}
	// Ohne Zuweisung läuft kein UPDATE, etwa wenn nur die unveränderte LUSD-ID mitkam. Eine
	// nachzutragende Adresse ist Arbeit, auch wenn an der Leserzeile nichts steht; sonst
	// wäre „nur die Schul-E-Mail nachtragen" ein 400.
	if b.IstLeer() && kontoAdresse == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Keine Felder zum Ändern angegeben", nil)
		return keine, false
	}
	return leserAenderung{update: b, lusdNachgetragen: lusdNachgetragen, kontoAdresse: kontoAdresse}, true
}

// trageNachUndProtokolliere legt nach dem Schreiben das Konto zur Schul-E-Mail an und
// schreibt die Protokolleinträge der Nachträge. false heißt: Die Ablehnung ist schon
// beantwortet.
func (t *Tueren) trageNachUndProtokolliere(w http.ResponseWriter, r *http.Request, protokollAbfragen repository.ProtokollAbfragen, claims *auth.Claims, id string, a leserAenderung) bool {
	ctx := r.Context()
	if a.kontoAdresse != "" {
		// Das Konto entsteht immer, aktiv nur bei manage_users — dieselbe Paarung wie beim
		// Anlegen (leser_anlegen.go). Wer hier nur edit_students hat, bekäme sonst still
		// das Recht, Zugänge freizuschalten.
		kontoID, ok := t.trageKontoNach(ctx, w, id, a.kontoAdresse, t.HatRecht(r, "manage_users"))
		if !ok {
			return false
		}
		// ziel_id ist die Kennung des Kontos: Daran findet die Auskunft den Eintrag beim Konto,
		// auch nachdem es gelöscht ist. Mit dem Leser fällt sie wieder (Tilgung).
		if protokollFehler := protokollAbfragen.ProtokolliereAdminAktion(ctx, claims.KontoID, "KOLLEGIUMSKONTO_NACHGETRAGEN", clientip.AusAnfrage(r), map[string]any{
			"schueler_id": id,
			"ziel_id":     kontoID,
		}); protokollFehler != nil {
			log.Printf("Audit für Kontonachtrag fehlgeschlagen: %v", protokollFehler)
		}
	}

	if a.lusdNachgetragen != "" {
		if protokollFehler := protokollAbfragen.ProtokolliereAdminAktion(ctx, claims.KontoID, "LUSD_ID_NACHGETRAGEN", clientip.AusAnfrage(r), map[string]any{
			"schueler_id": id,
			"lusd_id":     a.lusdNachgetragen,
		}); protokollFehler != nil {
			log.Printf("Audit für LUSD-ID-Nachtrag fehlgeschlagen: %v", protokollFehler)
		}
	}
	return true
}

// pruefeUndSetzeLUSDID trägt die LUSD-ID kontrolliert nach. Die LUSD-ID ist der
// einzige Zuordnungsschlüssel des Landesabgleichs (kein Adress-/Geburtsdatum-Fallback
// im Import); wer sie roh überschreibt, kann still die falsche Identität verknüpfen.
// Regeln: nur nachtragbar, wenn bisher leer (Waise adoptieren) — ein bestehender Wert
// wird nicht überschrieben und nicht geleert; der neue Wert muss eindeutig sein. Der
// automatische Gegenpart ist die Zuordnung im Import (Name und Geburtsdatum).
//
// Rückgabe: (nachgetragenerWert, ok). nachgetragenerWert != "" gehört ins Protokoll. Ohne
// Anfrage oder mit dem gleichen Wert (ohne Wirkung) ist die Rückgabe ("", true).
func (t *Tueren) pruefeUndSetzeLUSDID(ctx context.Context, w http.ResponseWriter, id string, angefragteLUSDID *string, b *repository.LeserAenderung) (string, bool) {
	if angefragteLUSDID == nil {
		return "", true
	}
	neu := strings.TrimSpace(*angefragteLUSDID)

	aktuell, err := repository.LadeLUSDIDDesLesers(ctx, t.DB.Pool, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Schüler nicht gefunden", nil)
			return "", false
		}
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return "", false
	}

	if neu == aktuell {
		return "", true // ohne Wirkung: Das Formular schickt den unveränderten Wert mit.
	}
	if aktuell != "" {
		// Bestehende Verknüpfung: weder überschreiben noch leeren.
		apierrors.AntworteFehlerMitMeldung(w, http.StatusForbidden, "Die LUSD-ID dieses Schülers ist gesetzt und lässt sich hier nicht ändern oder leeren", nil)
		return "", false
	}
	if neu == "" {
		return "", true // war leer, bleibt leer.
	}

	// Eindeutigkeit vor dem Schreiben prüfen, damit statt eines 500 am partiellen
	// Unique-Index (uniq_schueler_lusd_id_active) eine klare 409 zurückkommt.
	belegt, err := repository.HatAndererSchuelerLUSDID(ctx, t.DB.Pool, neu, id)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return "", false
	}
	if belegt {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, "Diese LUSD-ID gehört bereits einem anderen aktiven Schüler", nil)
		return "", false
	}

	b.LUSDID = &neu
	return neu, true
}

// pruefeUndSetzeArt ändert die Art eines Lesers (Migration 123). Erlaubt ist der Wechsel
// innerhalb des Kollegiums: Lehrkraft, LiV, Praktikum, Sekretariat, U-plus, Fachbereich
// (Migration 153). Beide Richtungen über die Grenze zum Schüler sind zu, und zwar nicht aus
// Vorsicht, sondern weil jede von ihnen einen echten Schaden anrichtet:
//
//	Schüler -> Kollege: Die Zeile verliert damit ihre LUSD-Bindung. Trägt sie eine
//	  lusd_id, bricht chk_leser_nur_schueler_werden_abgaenger; trägt sie keine, wäre der
//	  Schüler beim nächsten Import ein unbekannter Name und liefe als Abgänger samt
//	  Anonymisierung durch. Festgelegt ist: „ein Schüler kann nie ein Lehrer werden!"
//	Kollege -> Schüler: chk_leser_schueler_pflichtfelder verlangt Klasse, Abgängerjahr
//	  und Ausweisnummer. Ein Kollege hat die ersten beiden nicht; das UPDATE liefe in
//	  den CHECK und damit in eine 500.
//
// Ein Mensch wechselt die Seite nicht. Wer wirklich falsch angelegt wurde, wird gelöscht
// und neu angelegt: ein sichtbarer Vorgang statt einer stillen Umwidmung.
//
// nil heißt „nicht mitgeschickt"; derselbe Wert bleibt ohne Wirkung, damit das Formular die
// Art unverändert mitschicken darf.
//
// Ein bestehendes Konto bleibt beim Wechsel stehen, auch zu Praktikum oder Fachbereich: Neu
// angelegt wird dort keines (leserart.HatKonto), über ein vorhandenes entscheidet die
// Benutzerverwaltung.
func (t *Tueren) pruefeUndSetzeArt(ctx context.Context, w http.ResponseWriter, id string, angefragteArt *string, b *repository.LeserAenderung) bool {
	if angefragteArt == nil {
		return true
	}
	neu := strings.TrimSpace(*angefragteArt)
	// Dieselbe Menge wie beim Anlegen (leserart.IstBekannt) und dieselbe wie
	// chk_leser_art in der Datenbank. Eine unbekannte Art ist ein Tippfehler, kein neuer
	// Personenkreis, und soll als Auskunft zurückkommen, nicht als CHECK-500.
	if !leserart.IstBekannt(neu) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungUnbekannteArt(neu), nil)
		return false
	}

	// Gelesen wird die Tabelle leser, nicht die Sicht schueler: Ein Kollege steht nicht in der
	// Sicht.
	aktuell, err := repository.LadeLeserArt(ctx, t.DB.Pool, id)
	if err != nil {
		if errors.Is(err, repository.ErrLeserNichtGefunden) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Leser nicht gefunden", nil)
			return false
		}
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return false
	}

	if neu == aktuell {
		return true // ohne Wirkung: Das Formular schickt den unveränderten Wert mit.
	}
	if leserart.IstSchueler(aktuell) || leserart.IstSchueler(neu) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest,
			"Aus einem Schüler wird kein Kollege und umgekehrt. "+
				"Lege die Person neu an und lösche den falschen Eintrag.", nil)
		return false
	}

	b.Art = &neu
	return true
}

// leserAendernAnfrage bündelt die optional aktualisierbaren Stammdatenfelder (nil = unverändert).
type leserAendernAnfrage struct {
	Vorname       *string `json:"vorname"`
	Nachname      *string `json:"nachname"`
	Klasse        *string `json:"klasse"`
	LUSDID        *string `json:"lusd_id"`
	BarcodeID     *string `json:"barcode_id"`
	AbgaengerJahr *int    `json:"abgaenger_jahr"`
	Geburtsdatum  *string `json:"geburtsdatum"`
	// Art des Lesers (pkg/leserart, Migration 123 und 153). Sie steht hier, weil die Akte sie
	// zeigen und ein Kollege innerhalb des Kollegiums wechseln können muss. Sie hat einen
	// eigenen geprüften Pfad (pruefeUndSetzeArt) und gehört nicht in den allgemeinen
	// Feld-Beutel: Ein Wechsel der Art verschiebt die Zeile zwischen zwei Pflichtfeld-Welten
	// (chk_leser_schueler_pflichtfelder, chk_leser_nur_schueler_werden_abgaenger) und liefe
	// roh in einen CHECK, also in eine 500, die der Sanitizer zu "interner Datenbankfehler"
	// macht.
	Art *string `json:"art"`
	// Keine Sperrfelder hier. Sperren und Entsperren läuft ausschließlich über
	// PATCH /api/admin/students/{id}/lock (api/leser/leser_sperre.go), aus zwei Gründen, die
	// dieser Weg nicht erfüllen würde:
	//
	//   - Der Sperr-Endpunkt verlangt einen Grund. Über diesen PATCH gesetzt, liefe eine
	//     Sperre ohne Grund in chk_schueler_block_reason und käme als 500 „Ein interner
	//     Datenbankfehler ist aufgetreten" zurück (der Sanitizer ersetzt jede 500-Meldung).
	//     Derselbe Vorgang eine Tür weiter: 400 mit dem Satz, was zu tun ist.
	//   - Beim Entsperren räumt der Sperr-Endpunkt den Grund nur weg, wenn keine
	//     Systemsperre mehr besteht; dieser PATCH ließe ihn stehen.
	//
	// Zwei Türen zu demselben Zustand, von denen nur eine die Regeln kennt, sind keine
	// Bequemlichkeit: Die falsche Tür geht irgendwann auf.
	Strasse     *string `json:"strasse"`
	Hausnummer  *string `json:"hausnummer"`
	Plz         *string `json:"plz"`
	Ort         *string `json:"ort"`
	ElternEmail *string `json:"eltern_email"`
	// Email ist die Schul-Adresse eines Kollegen mit Zugang und steht nicht in `leser`,
	// sondern am Konto (benutzer.email). Sie liegt deshalb ebenfalls nicht im
	// generischen Feld-Beutel, sondern hat ihren eigenen Pfad (pruefeSchulEmail,
	// api/leser/leser_schul_email.go): nachtragbar, solange keine da ist — dann entsteht
	// das Konto; steht eine da, ist sie eine Anzeige und wird in der
	// Benutzerverwaltung geändert.
	Email *string `json:"email"`
}

// baueLeserAenderung prüft die Felder der Anfrage und nennt sie als Änderung der Leserzeile
// (samt Ableitung des Abgängerjahrs aus der Klasse und dem gelesenen Geburtsdatum). Welche
// Spalte ein Feld setzt, steht in repository.LeserAenderung. ok=false: Die Ablehnung ist
// schon beantwortet.
func baueLeserAenderung(w http.ResponseWriter, req *leserAendernAnfrage) (*repository.LeserAenderung, bool) {
	// Die Sperre des Spezialwerts 'lehrer' gilt auch für die zweite Tür (PATCH) —
	// siehe pruefeKlassenname und Migration 072.
	if req.Klasse != nil {
		if err := pruefeKlassenname(*req.Klasse); err != nil {
			apierrors.AntworteFehler(w, http.StatusBadRequest, err)
			return nil, false
		}
	}
	// Bei Klassenänderung ohne explizites Abgängerjahr dieses automatisch ableiten.
	if req.Klasse != nil && req.AbgaengerJahr == nil {
		neuesJahr := repository.AbgaengerJahr(*req.Klasse)
		req.AbgaengerJahr = &neuesJahr
	}

	// Pflichtfelder lassen sich nicht über den PATCH wegräumen.
	//
	// Beim Anlegen sind vorname/nachname/klasse `validate:"required"` (leser_anlegen.go); ohne
	// die Prüfung hier käme ein leerer String mit 200 durch: Der Leser verlöre Namen und
	// Klasse, die Oberfläche meldete "Änderungen gespeichert", und aus dem leeren Klassennamen
	// leitete repository.AbgaengerJahr noch ein Abgängerjahr ab. Dass ein Formular geräumte
	// Felder als JSON-null schickt (im *string nil, „nicht mitgeschickt"), ist Zufall und keine
	// Regel.
	//
	// Die Ausweisnummer steht nicht in dieser Liste: Ihre Pflicht ist an die Art
	// gepaart, wie in der Datenbank (chk_leser_schueler_pflichtfelder), und die Art steht
	// hier nicht fest. Ein Schüler ohne Nummer ist an der Theke unauffindbar; ein Kollege
	// muss eine falsch eingetragene wieder loswerden können — mit aktivem Konto bekommt er
	// dabei eine neue aus dem Generator (Migration 145). Geprüft wird sie deshalb in der Tür
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
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, feld.bezeichnung+" darf nicht leer sein.", nil)
			return nil, false
		}
	}

	// LUSDID und Art stehen nicht hier: Beide haben einen eigenen geprüften Pfad
	// (pruefeUndSetzeLUSDID, pruefeUndSetzeArt) und kämen sonst ungeprüft in die Zeile.
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
		// Pflichtfeld (Schlüssel des LUSD-Abgleichs): POST verlangt es, also darf der PATCH
		// es nicht mit "" still auf NULL setzen; zweite Tür, gleiche Regel. Leer = 400.
		if strings.TrimSpace(*req.Geburtsdatum) == "" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest,
				"Das Geburtsdatum lässt sich nicht leeren. Es ist der Schlüssel für den LUSD-Abgleich.", nil)
			return nil, false
		}
		datum, err := leseGeburtsdatum(*req.Geburtsdatum)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusBadRequest, err)
			return nil, false
		}
		b.Geburtsdatum = datum
	}

	// Ob die Änderung leer ist, prüft die Tür nach pruefeUndSetzeLUSDID: Eine nachgetragene
	// LUSD-ID ist eine gültige Änderung, und ihr Feld nennt erst dieser Pfad.
	return b, true
}

// schreibeLeserAenderung schreibt die Änderung an die Leserzeile. ok=false: Die Ablehnung
// (409 bei einer vergebenen Nummer oder einem vergebenen Namen, 404 bei unbekanntem Leser,
// sonst 500) ist schon beantwortet.
func (t *Tueren) schreibeLeserAenderung(ctx context.Context, w http.ResponseWriter, id string, b *repository.LeserAenderung) bool {
	gefunden, err := repository.AendereLeser(ctx, t.DB.Pool, id, *b)
	if err != nil {
		// Eine vergebene Ausweisnummer (unter den Schülern oder im Kollegium, Migration 118) ist
		// eine Auskunft, kein Serverfehler.
		if repository.IstAusweisKollision(err) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, "Diese Ausweisnummer trägt bereits eine andere Person (Schüler oder Kollegium)", nil)
			return false
		}
		if repository.IstNummerBuchOderAusweisKollision(err) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, "Diese Nummer ist der Barcode eines Buchs und kann kein Ausweis sein", nil)
			return false
		}
		// Name und Geburtsdatum eines anderen Lesers: dieselbe Auskunft wie beim Anlegen.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "unique_schueler_name_gebdatum" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, meldungSchuelerDuplikat, err)
			return false
		}
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return false
	}
	if !gefunden {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Leser nicht gefunden", nil)
		return false
	}
	return true
}
