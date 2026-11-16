package system

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/adressabgleich"
	"bibliothek/internal/bereitschaft"
	"bibliothek/internal/crypto"
	"bibliothek/mailservice"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Gründe, die erst beim Eintragen entstehen.
const (
	grundInzwischenGeaendert = "Adresse des Kontos inzwischen von Hand geändert"
	grundInzwischenVergeben  = "Adresse inzwischen an einem anderen Konto"
	grundNichtGespeichert    = "Adresse ließ sich nicht speichern"
)

// adressabgleichErgebnis ist die Antwort eines Laufs: die Zahlen und die Listen mit Namen, die
// nur der Admin am Bildschirm sieht.
type adressabgleichErgebnis struct {
	Probelauf   bool                         `json:"probelauf"`
	Stand       adressabgleichStand          `json:"stand"`
	Uebernahmen []adressabgleich.Uebernahme  `json:"uebernahmen"`
	Uebergangen []adressabgleich.Uebergangen `json:"uebergangen"`
}

// AdressabgleichLauf führt den Abgleich auf Knopfdruck aus; im Probelauf ändert er nichts.
func (t *Tueren) AdressabgleichLauf() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req adressabgleichLaufAnfrage
		if !tuer.LeseAnfrageStreng(w, r, &req) {
			return
		}
		adminID := ""
		if claims, ok := auth.ClaimsAusKontext(r.Context()); ok {
			adminID = claims.KontoID
		}
		ergebnis, err := t.fuehreAdressabgleichAus(r.Context(), req.Probelauf, adminID)
		if err != nil {
			status, meldung := adressabgleichFehlerantwort(err)
			apierrors.AntworteFehlerMitMeldung(w, status, meldung, err)
			return
		}
		tuer.AntworteJSON(w, http.StatusOK, ergebnis)
	}
}

// adressabgleichFehlerantwort ordnet einen gescheiterten Lauf ein: Was der Admin beheben kann,
// steht mit seinem Satz da, ein Webmail, das nicht wie erwartet antwortet, als 502.
func adressabgleichFehlerantwort(err error) (int, string) {
	var abbruch *adressabgleichAbbruch
	if !errors.As(err, &abbruch) {
		return http.StatusInternalServerError, meldungAdressabgleichGescheitert
	}
	switch abbruch.meldung {
	case meldungWebmailNichtEingerichtet, meldungWebmailPasswortUnlesbar:
		return http.StatusConflict, abbruch.meldung
	case meldungSchulDomainFehlt:
		return http.StatusConflict, abbruch.meldung
	}
	return http.StatusBadGateway, abbruch.meldung
}

// adressabgleichAbbruch ist ein Lauf, der vor dem Eintragen endete; meldung ist der Satz für die
// Oberfläche, Mail und Stand.
type adressabgleichAbbruch struct {
	meldung string
	ursache error
}

func (a *adressabgleichAbbruch) Error() string {
	if a.ursache == nil {
		return "adressabgleich abgebrochen: " + a.meldung
	}
	return "adressabgleich abgebrochen: " + a.meldung + ": " + a.ursache.Error()
}

func (a *adressabgleichAbbruch) Unwrap() error { return a.ursache }

// fuehreAdressabgleichAus holt das globale Adressbuch und trägt den Konten ihre Adresse der Schule
// ein (Regeln: adressabgleich.BerechneZuordnung); der Probelauf rechnet nur. adminID ist beim Lauf des Zeitplans leer. Der Stand
// eines Laufs, der eintragen durfte, wird gespeichert, auch wenn er scheiterte.
func (t *Tueren) fuehreAdressabgleichAus(ctx context.Context, probelauf bool, adminID string) (adressabgleichErgebnis, error) {
	ergebnis := adressabgleichErgebnis{Probelauf: probelauf, Stand: adressabgleichStand{Zeitpunkt: time.Now()}}
	zuordnung, eintraege, err := t.berechneAdressabgleich(ctx)
	ergebnis.Stand.Eintraege = eintraege
	if err != nil {
		ergebnis.Stand.Fehler = meldungAdressabgleichGescheitert
		var abbruch *adressabgleichAbbruch
		if errors.As(err, &abbruch) {
			ergebnis.Stand.Fehler = abbruch.meldung
		}
		// Ohne eingerichteten Zugang ist kein Lauf gescheitert; der Stand bleibt, wie er war.
		if !probelauf && ergebnis.Stand.Fehler != meldungWebmailNichtEingerichtet {
			t.schreibeAdressabgleichStand(ctx, ergebnis.Stand)
		}
		return ergebnis, err
	}

	// Leere Listen gehen als [] hinaus, nicht als null: Die Maske zählt ihre Einträge.
	ergebnis.Uebernahmen, ergebnis.Uebergangen = []adressabgleich.Uebernahme{}, append([]adressabgleich.Uebergangen{}, zuordnung.Uebergangen...)
	if probelauf {
		ergebnis.Uebernahmen = append(ergebnis.Uebernahmen, zuordnung.Uebernahmen...)
	} else {
		t.trageAdressenEin(ctx, zuordnung.Uebernahmen, adminID, &ergebnis)
	}
	ergebnis.Stand.Uebernommen = len(ergebnis.Uebernahmen)
	ergebnis.Stand.Uebergangen = len(ergebnis.Uebergangen)
	if !probelauf {
		t.schreibeAdressabgleichStand(ctx, ergebnis.Stand)
	}
	return ergebnis, nil
}

// berechneAdressabgleich lädt Zugang, Adressbuch und Konten und berechnet die Zuordnung. Es
// liefert die Zahl der Einträge im Adressbuch mit, auch wenn danach etwas scheitert.
func (t *Tueren) berechneAdressabgleich(ctx context.Context) (adressabgleich.Zuordnung, int, error) {
	eintraege, err := t.HoleAdressbuch(ctx)
	if err != nil {
		return adressabgleich.Zuordnung{}, 0, err
	}
	schulDomain := auth.SelbstanmeldeDomain()

	konten, err := repository.NewKontoAbfragen(t.DB.Pool).LadeAlle(ctx)
	if err != nil {
		return adressabgleich.Zuordnung{}, len(eintraege), err
	}
	abgleichKonten := make([]adressabgleich.Konto, 0, len(konten))
	for _, k := range konten {
		abgleichKonten = append(abgleichKonten, adressabgleich.Konto{
			ID: k.ID, Vorname: k.Vorname, Nachname: k.Nachname, Email: k.Email, Aktiv: k.Aktiv, LDAPKennung: k.LDAPKennung,
			Kollegium: strings.EqualFold(k.Rolle, string(auth.RolleKollegium)),
		})
	}
	return adressabgleich.BerechneZuordnung(eintraege, abgleichKonten, schulDomain, repository.PlatzhalterDomain), len(eintraege), nil
}

// HoleAdressbuch lädt mit dem Zugang aus Einstellungen → Mail das globale Adressbuch des Webmails.
// Der Abgleich und die erste Anmeldung mit dem Kürzel (auth.Anmeldung.SetzeAdressbuch) holen es
// beide hier. Ein Fehler ist ein *adressabgleichAbbruch mit der Meldung für die Oberfläche oder
// ein Fehler der Datenbank.
func (t *Tueren) HoleAdressbuch(ctx context.Context) ([]adressabgleich.Eintrag, error) {
	konfig, err := repository.NewMailEinstellungsAbfragen(t.DB.Pool).Lade(ctx)
	if err != nil {
		return nil, err
	}
	if konfig.WebmailAdresse == "" {
		return nil, &adressabgleichAbbruch{meldung: meldungWebmailNichtEingerichtet, ursache: adressabgleich.ErrNichtEingerichtet}
	}
	passwort, err := crypto.Entschluessele(konfig.WebmailPasswortVerschluesselt)
	if err != nil || len(passwort) == 0 {
		return nil, &adressabgleichAbbruch{meldung: meldungWebmailPasswortUnlesbar, ursache: err}
	}
	if auth.SelbstanmeldeDomain() == "" {
		return nil, &adressabgleichAbbruch{meldung: meldungSchulDomainFehlt, ursache: adressabgleich.ErrNichtEingerichtet}
	}

	vcf, err := adressabgleich.HoleGlobalesAdressbuch(ctx, adressabgleich.Zugang{
		Adresse: konfig.WebmailAdresse, Benutzer: konfig.WebmailBenutzer, Passwort: string(passwort),
	})
	switch {
	case errors.Is(err, adressabgleich.ErrAnmeldungAbgelehnt):
		return nil, &adressabgleichAbbruch{meldung: meldungWebmailAbgelehnt, ursache: err}
	case errors.Is(err, adressabgleich.ErrKeinGlobalesAdressbuch):
		return nil, &adressabgleichAbbruch{meldung: meldungKeinGlobalesAdressbuch, ursache: err}
	case err != nil:
		return nil, &adressabgleichAbbruch{meldung: meldungWebmailUnerreichbar, ursache: err}
	}
	return adressabgleich.LeseAdressbuch(vcf), nil
}

// trageAdressenEin schreibt jede Übernahme über die Tür der Kontoänderung, nur solange das Konto
// noch seine bisherige Adresse trägt, und protokolliert sie. Was sich nicht eintragen ließ, wandert mit
// Grund zu den übergangenen Konten.
func (t *Tueren) trageAdressenEin(ctx context.Context, uebernahmen []adressabgleich.Uebernahme, adminID string, ergebnis *adressabgleichErgebnis) {
	kontoAbfragen := repository.NewKontoAbfragen(t.DB.Pool)
	for _, u := range uebernahmen {
		adresse, bisher := u.Adresse, u.EmailBisher
		_, err := kontoAbfragen.Aendere(ctx, repository.KontoAenderung{ID: u.KontoID, Email: &adresse, EmailBisher: &bisher})
		if err != nil {
			grund := grundNichtGespeichert
			switch {
			case errors.Is(err, repository.ErrKontoNichtGefunden):
				grund = grundInzwischenGeaendert
			case repository.IstAdressenKollision(err):
				grund = grundInzwischenVergeben
			default:
				slog.Error("Adressabgleich: Adresse nicht gespeichert", "konto", u.KontoID, "fehler", err)
			}
			ergebnis.Uebergangen = append(ergebnis.Uebergangen, adressabgleich.Uebergangen{KontoID: u.KontoID, Name: u.Name, Grund: grund})
			continue
		}
		ergebnis.Uebernahmen = append(ergebnis.Uebernahmen, u)
		details, err := json.Marshal(map[string]any{"ziel_id": u.KontoID, "email": adresse})
		if err != nil {
			slog.Error("audit: Details der Übernahme nicht serialisierbar", "fehler", err)
			continue
		}
		t.SchreibeAdminProtokoll(ctx, adminID, "ADRESSE_AUS_WEBMAIL", "", string(details))
	}
}

func (t *Tueren) schreibeAdressabgleichStand(ctx context.Context, stand adressabgleichStand) {
	wert, err := json.Marshal(stand)
	if err == nil {
		err = repository.NewBetriebszustandAbfragen(t.DB.Pool).SchreibeEinstellungswert(ctx, adressabgleichStandSchluessel, string(wert))
	}
	if err != nil {
		slog.Error("Adressabgleich: Stand nicht gespeichert", "fehler", err)
	}
}

// AdressabgleichNachZeitplan ist der monatliche Lauf. Ohne eingerichteten Zugang tut er nichts.
// Hat er etwas eingetragen oder ist er gescheitert, geht eine Mail an die Empfänger der Alarme,
// nur mit Zahlen: Namen und Adressen verlassen den Server nicht.
func (t *Tueren) AdressabgleichNachZeitplan(ctx context.Context) {
	ergebnis, err := t.fuehreAdressabgleichAus(ctx, false, "")
	var abbruch *adressabgleichAbbruch
	if errors.As(err, &abbruch) && abbruch.meldung == meldungWebmailNichtEingerichtet {
		slog.Info("Adressabgleich: kein Zugang zum Webmail eingerichtet, nichts zu tun")
		return
	}
	if err != nil {
		slog.Error("Adressabgleich: Lauf gescheitert", "meldung", ergebnis.Stand.Fehler, "fehler", err)
	} else {
		slog.Info("Adressabgleich: Lauf beendet", "eintraege", ergebnis.Stand.Eintraege,
			"uebernommen", ergebnis.Stand.Uebernommen, "uebergangen", ergebnis.Stand.Uebergangen)
	}
	if err == nil && ergebnis.Stand.Uebernommen == 0 {
		return
	}
	t.sendeAdressabgleichBericht(ctx, ergebnis.Stand)
}

func (t *Tueren) sendeAdressabgleichBericht(ctx context.Context, stand adressabgleichStand) {
	// Wie der Bereitschafts-Alarm: Ein Entwicklungsrechner verschickt keine Mails an die Schule.
	if !bereitschaft.IstEchterBetrieb(strings.ToLower(os.Getenv("APP_ENV"))) {
		return
	}
	einstellungen, err := repository.NewEinstellungsAbfragen(t.DB.Pool).Lade(ctx)
	if err != nil {
		slog.Error("Adressabgleich: Bericht nicht verschickt, Einstellungen nicht lesbar", "fehler", err)
		return
	}
	admins, err := repository.NewBetriebszustandAbfragen(t.DB.Pool).LadeAktiveAdmins(ctx)
	if err != nil {
		slog.Error("Adressabgleich: Bericht nicht verschickt, Admin-Konten nicht lesbar", "fehler", err)
		return
	}
	konfiguriert := ""
	if einstellungen.AlarmEmpfaenger != nil {
		konfiguriert = *einstellungen.AlarmEmpfaenger
	}
	empfaenger, beschreibung := waehleAlarmEmpfaenger(konfiguriert, admins)
	betreff, text := formatiereAdressabgleichBericht(stand)
	text += "\nDiese Mail ging an " + beschreibung + "\n"
	for _, an := range empfaenger {
		if err := t.Post().Sende(ctx, mailservice.Nachricht{An: an, Betreff: betreff, Text: text}); err != nil {
			slog.Error("Adressabgleich: Bericht nicht verschickt", "empfaenger", an, "fehler", err)
		}
	}
}

// formatiereAdressabgleichBericht schreibt den Bericht eines Laufs nach Zeitplan.
func formatiereAdressabgleichBericht(stand adressabgleichStand) (betreff, text string) {
	zeit := stand.Zeitpunkt.In(schulzeit.Zone()).Format("02.01.2006 um 15:04")
	if stand.Fehler != "" {
		return "Adressabgleich mit dem Webmail gescheitert", fmt.Sprintf(
			"Der monatliche Abgleich mit dem globalen Adressbuch des Webmails am %s ist gescheitert:\n%s\n\n"+
				"Abhilfe: Zugang unter Einstellungen → Mail prüfen und dort „Probelauf“ wählen.\n",
			zeit, ohneMailadressen(stand.Fehler))
	}
	return fmt.Sprintf("Adressabgleich: %d Adressen aus dem Webmail übernommen", stand.Uebernommen), fmt.Sprintf(
		"Der monatliche Abgleich mit dem globalen Adressbuch des Webmails am %s hat %d Konten ihre Adresse "+
			"der Schule eingetragen; %d Konten blieben ohne eindeutigen Treffer.\n\n"+
			"Die Namen zeigt der Probelauf unter Einstellungen → Mail; jede Übernahme steht im Protokoll "+
			"der Verwaltung („Adresse aus dem Webmail übernommen“).\n",
		zeit, stand.Uebernommen, stand.Uebergangen)
}
