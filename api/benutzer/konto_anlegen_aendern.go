package benutzer

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
)

// kontoAnlegenAnfrage nennt die Angaben eines neuen Kontos.
type kontoAnlegenAnfrage struct {
	BarcodeID string `json:"barcode_id"`
	Vorname   string `json:"vorname" validate:"required"`
	Nachname  string `json:"nachname" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Rolle     string `json:"rolle" validate:"required"`
}

// KontoAnlegen legt ein Konto an. Ein Passwort gibt es nicht: Angemeldet wird über den
// Schul-Mailserver (IMAP) oder mit Barcode und PIN.
// @Summary      Konto anlegen
// @Description  Legt ein Konto des Personals mit seiner Rolle an. Angemeldet wird über IMAP oder Barcode, nicht über ein Passwort.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        body  body      kontoAnlegenAnfrage  true  "Angaben des Kontos"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /benutzer [post]
func (t *Tueren) KontoAnlegen(kontoAbfragen repository.KontoAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.legeKontoAn(w, r, kontoAbfragen)
	}
}

// legeKontoAn prüft Rolle, Adresse und Ausweis, legt das Konto an und protokolliert es.
func (t *Tueren) legeKontoAn(w http.ResponseWriter, r *http.Request, kontoAbfragen repository.KontoAbfragen) {
	var req kontoAnlegenAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return
	}

	if req.Vorname == "" || req.Nachname == "" || req.Email == "" || req.Rolle == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("alle Felder sind Pflichtfelder"))
		return
	}

	// Einen Administrator darf nur ein Administrator anlegen (siehe admin_schutz.go).
	if !pruefeAdminVergabe(w, r, req.Rolle) {
		return
	}

	ctx := r.Context()

	if !pruefeEmailEindeutig(ctx, w, kontoAbfragen, req.Email, "") {
		return
	}
	if !pruefeKeineLeserzeileOhneKonto(ctx, w, kontoAbfragen, req.Vorname, req.Nachname) {
		return
	}

	nummer, ok := pruefeAusweisnummerEindeutig(ctx, w, kontoAbfragen, ausweisPruefung{
		nummer:          req.BarcodeID,
		ausgenommeneID:  "",
		konfliktmeldung: "dieser Barcode wird bereits verwendet",
	})
	if !ok {
		return
	}

	rolle := normalisiereRolle(req.Rolle)

	// Die Leserzeile — und damit der Platz für Ausweis und Ausleihen — entsteht dabei
	// von selbst (Trigger trg_benutzer_hat_leserzeile).
	kontoID, err := kontoAbfragen.LegeAn(ctx, nummer, req.Vorname, req.Nachname, req.Email, rolle)
	if err != nil {
		if meldeAusweisKollision(w, err) {
			return
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	t.ProtokolliereKontoAnlage(r.Context(), kontoID, req.Email, rolle, req.Vorname, req.Nachname)

	w.Header().Set(tuer.HeaderContentType, tuer.ContentTypeJSON)
	httpresp.Schreibe(w, []byte(`{"status":"success"}`))
}

// kontoAendernAnfrage nennt, was an einem Konto geändert wird. Ein fehlendes Feld bleibt, wie es
// ist: Die Maske schickt nur, was sie seit dem Öffnen geändert hat, damit sie nichts
// überschreibt, was ein anderer Platz inzwischen gespeichert hat. Ein Passwort gibt es hier
// nicht; Anmeldungen laufen über den Schul-Mailserver (IMAP).
type kontoAendernAnfrage struct {
	BarcodeID *string `json:"barcode_id"`
	Vorname   *string `json:"vorname"`
	Nachname  *string `json:"nachname"`
	Email     *string `json:"email" validate:"omitempty,email"`
	Rolle     *string `json:"rolle"`
	Aktiv     *bool   `json:"aktiv"`
}

// KontoAendern ändert Name, E-Mail, Ausweisnummer, Rolle oder „aktiv" eines Kontos.
// @Summary      Konto ändern
// @Description  Ändert an einem Konto die Felder, die der Rumpf nennt; ein fehlendes Feld bleibt, wie es ist.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id    path      string             true  "Kennung des Kontos (UUID)"
// @Param        body  body      kontoAendernAnfrage  true  "Geänderte Felder"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /benutzer/{id} [put]
func (t *Tueren) KontoAendern(kontoAbfragen repository.KontoAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.aendereKonto(w, r, kontoAbfragen)
	}
}

// aendereKonto prüft die Schutzregeln vor jedem Schreibzugriff, ändert das Konto und
// protokolliert es.
func (t *Tueren) aendereKonto(w http.ResponseWriter, r *http.Request, kontoAbfragen repository.KontoAbfragen) {
	id := r.PathValue("id")
	if id == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing user ID parameter"))
		return
	}

	var req kontoAendernAnfrage
	// Streng: Ein vertippter Feldname nennte sonst nichts, und die Tür meldete Erfolg, ohne zu
	// schreiben („aktive" statt „aktiv" ließe ein Konto freigeschaltet).
	if !tuer.LeseAnfrageStreng(w, r, &req) {
		return
	}
	if !pruefeGenanntePflichtfelder(w, req) {
		return
	}

	ctx := r.Context()
	if req.nenntNichts() {
		antworteOhneKontoAenderung(ctx, w, kontoAbfragen, id)
		return
	}

	// Reihenfolge ist Absicht: erst die eigene Rolle/Aktivierung schützen, dann die
	// Vergabe der Admin-Rolle, dann den Schutz bestehender Admin-Konten. Alle drei
	// laufen vor jedem Schreibzugriff (siehe admin_schutz.go).
	if !pruefeSelbstschutz(w, r, id, req.Rolle, req.Aktiv) {
		return
	}
	if req.Rolle != nil && !pruefeAdminVergabe(w, r, *req.Rolle) {
		return
	}
	if !pruefeAdminZiel(ctx, w, r, kontoAbfragen, id) {
		return
	}

	aenderung, ok := kontoAenderungAus(ctx, w, kontoAbfragen, id, req)
	if !ok {
		return
	}
	stand, err := kontoAbfragen.Aendere(ctx, aenderung)
	if err != nil {
		// Kein Protokolleintrag und kein Leeren des Rechte-Caches für eine Änderung, die nie
		// stattfand.
		antworteAufKontoAenderungsfehler(w, err)
		return
	}

	// Konten und Rollen stehen im Protokoll, damit sich feststellen lässt, wer ein Konto wann
	// angelegt oder geändert hat. Der Eintrag nennt den Stand nach der Änderung, auch für
	// Felder, die der Rumpf nicht nannte.
	t.ProtokolliereVerwaltung(r.Context(), "USER_UPDATE", map[string]any{
		"ziel_id": id, "email": stand.Email, "rolle": stand.Rolle, "aktiv": stand.Aktiv,
	})

	tuer.LeereRechteCache()

	w.Header().Set(tuer.HeaderContentType, tuer.ContentTypeJSON)
	httpresp.Schreibe(w, []byte(`{"status":"success"}`))
}

// nenntNichts sagt, ob der Rumpf kein Feld des Kontos nennt.
func (a kontoAendernAnfrage) nenntNichts() bool {
	return a.BarcodeID == nil && a.Vorname == nil && a.Nachname == nil &&
		a.Email == nil && a.Rolle == nil && a.Aktiv == nil
}

// pruefeGenanntePflichtfelder lehnt ein genanntes, aber leeres Pflichtfeld ab: Vorname,
// Nachname, E-Mail und Rolle lassen sich ändern, nicht leeren.
func pruefeGenanntePflichtfelder(w http.ResponseWriter, req kontoAendernAnfrage) bool {
	for _, wert := range []*string{req.Vorname, req.Nachname, req.Email, req.Rolle} {
		if wert != nil && strings.TrimSpace(*wert) == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest,
				errors.New("vorname, Nachname, E-Mail und Rolle dürfen nicht leer sein"))
			return false
		}
	}
	return true
}

// antworteOhneKontoAenderung beantwortet einen Rumpf, der nichts nennt: nichts geschrieben,
// kein Protokolleintrag. Ein unbekanntes Konto bleibt eine 404.
func antworteOhneKontoAenderung(ctx context.Context, w http.ResponseWriter, kontoAbfragen repository.KontoAbfragen, id string) {
	rolle, err := kontoAbfragen.LadeRolle(ctx, id)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}
	if rolle == "" {
		apierrors.SendHTTPError(w, http.StatusNotFound, repository.ErrKontoNichtGefunden)
		return
	}
	w.Header().Set(tuer.HeaderContentType, tuer.ContentTypeJSON)
	httpresp.Schreibe(w, []byte(`{"status":"success"}`))
}

// kontoAenderungAus prüft die genannte E-Mail und Ausweisnummer auf Eindeutigkeit und baut die
// Änderung für die Abfragen. Bei false ist die Ablehnung schon beantwortet.
func kontoAenderungAus(ctx context.Context, w http.ResponseWriter, kontoAbfragen repository.KontoAbfragen, id string, req kontoAendernAnfrage) (repository.KontoAenderung, bool) {
	aenderung := repository.KontoAenderung{
		ID: id, Vorname: req.Vorname, Nachname: req.Nachname, Email: req.Email, Aktiv: req.Aktiv,
	}
	if req.Email != nil && !pruefeEmailEindeutig(ctx, w, kontoAbfragen, *req.Email, id) {
		return aenderung, false
	}
	if req.BarcodeID != nil {
		nummer, ok := pruefeAusweisnummerEindeutig(ctx, w, kontoAbfragen, ausweisPruefung{
			nummer:          *req.BarcodeID,
			ausgenommeneID:  id,
			konfliktmeldung: "dieser Barcode wird bereits von einem anderen Benutzer verwendet",
		})
		if !ok {
			return aenderung, false
		}
		aenderung.BarcodeGenannt, aenderung.Barcode = true, nummer
	}
	if req.Rolle != nil {
		rolle := normalisiereRolle(*req.Rolle)
		aenderung.Rolle = &rolle
	}
	return aenderung, true
}

// antworteAufKontoAenderungsfehler ordnet den Fehler beim Ändern eines Kontos ein: unbekannte
// Kennung 404, belegter Ausweis 409, alles andere 500.
func antworteAufKontoAenderungsfehler(w http.ResponseWriter, err error) {
	if errors.Is(err, repository.ErrKontoNichtGefunden) {
		apierrors.SendHTTPError(w, http.StatusNotFound, err)
		return
	}
	if meldeAusweisKollision(w, err) {
		return
	}
	apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
}
