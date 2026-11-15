package system

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/adressabgleich"
	"bibliothek/repository"
)

// Der Abgleich der Adressen holt das globale Adressbuch aus dem Webmail der Schule und trägt
// Konten mit Platzhalter-Adresse ihre Adresse der Schule ein (internal/adressabgleich). Er läuft
// am Ersten jedes Monats von selbst (jobs/cron.go) und auf Knopfdruck unter Einstellungen → Mail;
// der Probelauf zeigt nur, was er übernähme.

// adressabgleichStandSchluessel ist der Schlüssel des letzten Laufs in system_einstellungen. Der
// Stand trägt nur Zahlen: Namen und Adressen zeigt der Probelauf, und jede Übernahme steht im
// Protokoll der Verwaltung, wo die Tilgung sie findet.
const adressabgleichStandSchluessel = "adressabgleich_stand"

// Sätze der Oberfläche zum Abgleich; der Stand eines gescheiterten Laufs trägt denselben Satz.
const (
	meldungWebmailNichtEingerichtet  = "Der Zugang zum Webmail ist nicht eingerichtet"
	meldungWebmailAbgelehnt          = "Das Webmail hat die Anmeldung abgelehnt. Bitte Benutzername und Passwort prüfen."
	meldungKeinGlobalesAdressbuch    = "Das Konto sieht im Webmail kein globales Adressbuch"
	meldungWebmailUnerreichbar       = "Das Webmail ist nicht erreichbar oder antwortet nicht wie Roundcube"
	meldungWebmailPasswortUnlesbar   = "Das gespeicherte Passwort des Webmails ist nicht lesbar. Bitte neu eintragen."
	meldungSchulDomainFehlt          = "Die Domain der Schule ist nicht eingestellt (SELBSTANMELDUNG_DOMAIN)"
	meldungWebmailBenutzerFehlt      = "Bitte den Benutzernamen des Webmails eintragen"
	meldungWebmailPasswortFehlt      = "Bitte das Passwort des Webmails eintragen"
	meldungAdressabgleichGescheitert = "Der Abgleich ließ sich nicht ausführen"
)

// adressabgleichStand ist das Ergebnis des letzten Laufs, der etwas eintragen durfte.
type adressabgleichStand struct {
	Zeitpunkt   time.Time `json:"zeitpunkt"`
	Eintraege   int       `json:"eintraege"`
	Uebernommen int       `json:"uebernommen"`
	Uebergangen int       `json:"uebergangen"`
	Fehler      string    `json:"fehler,omitempty"`
}

type adressabgleichZugangAntwort struct {
	Adresse     string               `json:"adresse"`
	Benutzer    string               `json:"benutzer"`
	HatPasswort bool                 `json:"has_password"`
	LetzterLauf *adressabgleichStand `json:"letzter_lauf"`
}

type adressabgleichZugangAnfrage struct {
	Adresse  string `json:"adresse"`
	Benutzer string `json:"benutzer"`
	Passwort string `json:"passwort"` // leer: das gespeicherte Passwort bleibt
}

type adressabgleichLaufAnfrage struct {
	Probelauf bool `json:"probelauf"`
}

// AdressabgleichZugang liefert den Zugang zum Webmail ohne Passwort und den letzten Lauf.
func (t *Tueren) AdressabgleichZugang(mailAbfragen *repository.MailEinstellungsAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		konfig, err := mailAbfragen.Lade(r.Context())
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		tuer.AntworteJSON(w, http.StatusOK, adressabgleichZugangAntwort{
			Adresse:     konfig.WebmailAdresse,
			Benutzer:    konfig.WebmailBenutzer,
			HatPasswort: len(konfig.WebmailPasswortVerschluesselt) > 0,
			LetzterLauf: t.ladeAdressabgleichStand(r.Context()),
		})
	}
}

// AdressabgleichZugangSchreiben speichert den Zugang zum Webmail. Eine leere Adresse schaltet den
// Abgleich ab.
func (t *Tueren) AdressabgleichZugangSchreiben(mailAbfragen *repository.MailEinstellungsAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req adressabgleichZugangAnfrage
		if !tuer.LeseAnfrageStreng(w, r, &req) {
			return
		}
		ctx := r.Context()
		if !pruefeWebmailZugang(ctx, w, mailAbfragen, &req) {
			return
		}
		if err := mailAbfragen.SchreibeWebmailZugang(ctx, req.Adresse, req.Benutzer, req.Passwort); err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		t.protokolliereWebmailZugang(ctx, req)
		tuer.AntworteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// pruefeWebmailZugang prüft den Zugang vor dem Speichern und setzt die Grundadresse ein. Bei
// einem Fehler hat sie geantwortet und liefert false.
func pruefeWebmailZugang(ctx context.Context, w http.ResponseWriter, mailAbfragen *repository.MailEinstellungsAbfragen, req *adressabgleichZugangAnfrage) bool {
	req.Adresse, req.Benutzer = strings.TrimSpace(req.Adresse), strings.TrimSpace(req.Benutzer)
	if req.Adresse == "" {
		return true
	}
	grundadresse, err := adressabgleich.Grundadresse(req.Adresse)
	if err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungWebmailAdresse(err), err)
		return false
	}
	req.Adresse = grundadresse
	if req.Benutzer == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungWebmailBenutzerFehlt, nil)
		return false
	}
	konfig, err := mailAbfragen.Lade(ctx)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return false
	}
	if req.Passwort == "" && len(konfig.WebmailPasswortVerschluesselt) == 0 {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungWebmailPasswortFehlt, nil)
		return false
	}
	return true
}

// meldungWebmailAdresse nennt, was an der Adresse des Webmails nicht stimmt.
func meldungWebmailAdresse(err error) string {
	if errors.Is(err, adressabgleich.ErrAdresseOhneHTTPS) {
		return "Die Adresse des Webmails muss mit https:// beginnen"
	}
	return "Die Adresse des Webmails ist keine gültige Adresse"
}

func (t *Tueren) protokolliereWebmailZugang(ctx context.Context, req adressabgleichZugangAnfrage) {
	claims, ok := auth.ClaimsAusKontext(ctx)
	if !ok {
		return
	}
	details, err := json.Marshal(map[string]any{
		"webmail_adresse":  req.Adresse,
		"webmail_benutzer": req.Benutzer,
		"password_changed": req.Passwort != "",
	})
	if err != nil {
		slog.Error("audit: Details zum Webmail-Zugang nicht serialisierbar", "fehler", err)
		return
	}
	t.SchreibeAdminProtokoll(ctx, claims.KontoID, "UPDATE_WEBMAIL_ZUGANG", "", string(details))
}

// ladeAdressabgleichStand liest den letzten Lauf; nil, wenn es noch keinen gab oder er nicht
// lesbar ist.
func (t *Tueren) ladeAdressabgleichStand(ctx context.Context) *adressabgleichStand {
	wert, err := repository.NewBetriebszustandAbfragen(t.DB.Pool).LadeEinstellungswert(ctx, adressabgleichStandSchluessel)
	if err != nil {
		return nil
	}
	var stand adressabgleichStand
	if err := json.Unmarshal([]byte(wert), &stand); err != nil {
		slog.Warn("Adressabgleich: gespeicherter Stand nicht lesbar", "fehler", err)
		return nil
	}
	return &stand
}
