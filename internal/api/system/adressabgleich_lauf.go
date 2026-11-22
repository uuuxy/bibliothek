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

	"bibliothek/internal/adressabgleich"
	"bibliothek/internal/api/tuer"
	"bibliothek/internal/apierrors"
	"bibliothek/internal/auth"
	"bibliothek/internal/bereitschaft"
	"bibliothek/internal/crypto"
	"bibliothek/internal/mailservice"
	"bibliothek/internal/pkg/schulzeit"
	"bibliothek/internal/repository"
)

// Gründe, die der Lauf setzt: vor dem Eintragen und dabei.
const (
	grundInzwischenGeaendert = "Adresse des Kontos inzwischen von Hand geändert"
	grundInzwischenVergeben  = "Adresse inzwischen an einem anderen Konto"
	grundNichtGespeichert    = "Adresse ließ sich nicht speichern"
	grundKontoEinesAdmins    = "Konto eines Administrators: Den Abgleich dafür löst ein Administrator aus"
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
		adminID, alsAdmin := "", false
		if claims, ok := auth.ClaimsAusKontext(r.Context()); ok {
			adminID, alsAdmin = claims.KontoID, claims.Rolle.IstAdmin()
		}
		ergebnis, err := t.fuehreAdressabgleichAus(r.Context(), req.Probelauf, adminID, alsAdmin)
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

// fuehreAdressabgleichAus holt das globale Adressbuch und trägt den Konten mit Platzhalter ihre
// Adresse ein; der Probelauf rechnet nur. adminID ist beim Lauf des Zeitplans leer. Die Konten von
// Administratoren füllt nur ein Lauf, den ein Administrator auslöst (alsAdmin). Der Stand eines
// Laufs, der eintragen durfte, wird gespeichert, auch wenn er scheiterte.
func (t *Tueren) fuehreAdressabgleichAus(ctx context.Context, probelauf bool, adminID string, alsAdmin bool) (adressabgleichErgebnis, error) {
	ergebnis := adressabgleichErgebnis{Probelauf: probelauf, Stand: adressabgleichStand{Zeitpunkt: time.Now()}}
	zuordnung, eintraege, err := t.berechneAdressabgleich(ctx, alsAdmin)
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
func (t *Tueren) berechneAdressabgleich(ctx context.Context, alsAdmin bool) (adressabgleich.Zuordnung, int, error) {
	konfig, err := repository.NewMailEinstellungsAbfragen(t.DB.Pool).Lade(ctx)
	if err != nil {
		return adressabgleich.Zuordnung{}, 0, err
	}
	if konfig.WebmailAdresse == "" {
		return adressabgleich.Zuordnung{}, 0, &adressabgleichAbbruch{meldung: meldungWebmailNichtEingerichtet}
	}
	passwort, err := crypto.Entschluessele(konfig.WebmailPasswortVerschluesselt)
	if err != nil || len(passwort) == 0 {
		return adressabgleich.Zuordnung{}, 0, &adressabgleichAbbruch{meldung: meldungWebmailPasswortUnlesbar, ursache: err}
	}
	schulDomain := auth.SelbstanmeldeDomain()
	if schulDomain == "" {
		return adressabgleich.Zuordnung{}, 0, &adressabgleichAbbruch{meldung: meldungSchulDomainFehlt}
	}

	vcf, err := adressabgleich.HoleGlobalesAdressbuch(ctx, adressabgleich.Zugang{
		Adresse: konfig.WebmailAdresse, Benutzer: konfig.WebmailBenutzer, Passwort: string(passwort),
	})
	switch {
	case errors.Is(err, adressabgleich.ErrAnmeldungAbgelehnt):
		return adressabgleich.Zuordnung{}, 0, &adressabgleichAbbruch{meldung: meldungWebmailAbgelehnt, ursache: err}
	case errors.Is(err, adressabgleich.ErrKeinGlobalesAdressbuch):
		return adressabgleich.Zuordnung{}, 0, &adressabgleichAbbruch{meldung: meldungKeinGlobalesAdressbuch, ursache: err}
	case err != nil:
		return adressabgleich.Zuordnung{}, 0, &adressabgleichAbbruch{meldung: meldungWebmailUnerreichbar, ursache: err}
	}
	eintraege := adressabgleich.LeseAdressbuch(vcf)

	konten, err := repository.NewKontoAbfragen(t.DB.Pool).LadeAlle(ctx)
	if err != nil {
		return adressabgleich.Zuordnung{}, len(eintraege), err
	}
	abgleichKonten := make([]adressabgleich.Konto, 0, len(konten))
	for _, k := range konten {
		abgleichKonten = append(abgleichKonten, adressabgleich.Konto{
			ID: k.ID, Vorname: k.Vorname, Nachname: k.Nachname, Email: k.Email, Aktiv: k.Aktiv, LDAPKennung: k.LDAPKennung,
		})
	}
	zuordnung := adressabgleich.BerechneZuordnung(eintraege, abgleichKonten, schulDomain, repository.PlatzhalterDomain)
	if !alsAdmin {
		zuordnung = ohneAdminKonten(zuordnung, konten)
	}
	return zuordnung, len(eintraege), nil
}

// ohneAdminKonten nimmt die Konten von Administratoren aus den Übernahmen und führt sie mit Grund
// bei den übergangenen. Wer die Adresse eines Kontos schreibt, übernimmt es; ein solches Konto
// bearbeitet nur ein Administrator (internal/api/benutzer/admin_schutz.go), und die Quelle des
// Abgleichs stellt schon ein, wer Einstellungen ändern darf.
func ohneAdminKonten(z adressabgleich.Zuordnung, konten []repository.Konto) adressabgleich.Zuordnung {
	admins := make(map[string]bool)
	for _, k := range konten {
		if auth.Rolle(k.Rolle).IstAdmin() {
			admins[k.ID] = true
		}
	}
	uebernahmen := make([]adressabgleich.Uebernahme, 0, len(z.Uebernahmen))
	for _, u := range z.Uebernahmen {
		if admins[u.KontoID] {
			z.Uebergangen = append(z.Uebergangen, adressabgleich.Uebergangen{KontoID: u.KontoID, Name: u.Name, Grund: grundKontoEinesAdmins})
			continue
		}
		uebernahmen = append(uebernahmen, u)
	}
	z.Uebernahmen = uebernahmen
	return z
}

// trageAdressenEin schreibt jede Übernahme über die Tür der Kontoänderung, nur solange das Konto
// noch seinen Platzhalter trägt, und protokolliert sie. Was sich nicht eintragen ließ, wandert mit
// Grund zu den übergangenen Konten.
func (t *Tueren) trageAdressenEin(ctx context.Context, uebernahmen []adressabgleich.Uebernahme, adminID string, ergebnis *adressabgleichErgebnis) {
	kontoAbfragen := repository.NewKontoAbfragen(t.DB.Pool)
	for _, u := range uebernahmen {
		adresse, platzhalter := u.Adresse, u.Platzhalter
		_, err := kontoAbfragen.Aendere(ctx, repository.KontoAenderung{ID: u.KontoID, Email: &adresse, EmailBisher: &platzhalter})
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

// schreibeAdressabgleichStand schreibt auch, wenn die Anfrage schon geendet hat (Frist abgelaufen,
// Browser getrennt): Gerade dann soll die Maske den gescheiterten Lauf zeigen.
func (t *Tueren) schreibeAdressabgleichStand(ctx context.Context, stand adressabgleichStand) {
	ctx, abbrechen := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer abbrechen()
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
	ergebnis, err := t.fuehreAdressabgleichAus(ctx, false, "", false)
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
			"der Schule eingetragen; %d Konten mit Platzhalter blieben ohne eindeutigen Treffer.\n\n"+
			"Die Namen zeigt der Probelauf unter Einstellungen → Mail; jede Übernahme steht im Protokoll "+
			"der Verwaltung als „Adresse aus dem Webmail übernommen“.\n",
		zeit, stand.Uebernommen, stand.Uebergangen)
}
