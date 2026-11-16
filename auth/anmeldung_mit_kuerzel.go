package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"bibliothek/db"
	"bibliothek/internal/adressabgleich"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Anmeldung mit dem Kürzel, nur für Mein Portal: Eine Lehrkraft meldet sich mit ihrem Kürzel und
// dem Passwort des PaedNet an. Trägt noch kein Konto diese LDAP-Kennung, sucht das Programm das
// Kürzel im globalen Adressbuch des Webmails und das Konto zur Adresse von dort. Gibt es keins,
// entsteht nach der Prüfung am Verzeichnisdienst ein Konto des Kollegiums mit Name und Adresse aus
// dem Adressbuch; ein aktives oder beantragtes Konto des Kollegiums wird auf das Kürzel
// umgestellt. Beides gilt sofort, ohne Freischalten: Das Adressbuch ist das Verzeichnis des
// Personals, das die Schule selbst pflegt, und das Passwort des PaedNet beweist, dass die Person
// das Kürzel ist (ARCHITEKTUR A30).
//
// Alles andere bleibt, wie es war: Ein Konto mit einer anderen Rolle, ein deaktiviertes Konto und
// ein Kürzel ohne sichere Adresse erreichen auf diesem Weg den Verzeichnisdienst nicht, und die
// Anmeldung geht weiter wie zuvor.

// AdressbuchQuelle holt das globale Adressbuch des Webmails (api/system: Tueren.HoleAdressbuch).
type AdressbuchQuelle func(ctx context.Context) ([]adressabgleich.Eintrag, error)

// SetzeAdressbuch schaltet die Anmeldung mit dem Kürzel ein. Ohne Quelle, ohne Zugang zum Webmail
// und ohne LDAP gibt es sie nicht. Aufgerufen wird es beim Start, bevor die erste Anfrage kommt.
func (a *Anmeldung) SetzeAdressbuch(quelle AdressbuchQuelle) {
	a.adressbuch = &adressbuchVorrat{quelle: quelle}
}

// adressbuchAbstand ist die kürzeste Zeit zwischen zwei Abrufen. Jede Eingabe ohne Konto fragt
// das Adressbuch; wer Namen durchprobiert, meldete sonst bei jedem Versuch das Dienstkonto am
// Webmail an.
const adressbuchAbstand = 2 * time.Minute

// adressbuchFrist begrenzt einen Abruf während der Anmeldung, damit innerhalb von anmeldeFrist
// Zeit für den Verzeichnisdienst bleibt.
const adressbuchFrist = 10 * time.Second

// adressbuchVorrat hält das zuletzt geholte Adressbuch.
type adressbuchVorrat struct {
	quelle    AdressbuchQuelle
	mu        sync.Mutex
	eintraege []adressabgleich.Eintrag
	geholtAm  time.Time
}

// aktuell liefert das Adressbuch, höchstens adressbuchAbstand alt. Scheitert ein Abruf, steht er
// im Log, und das vorige Adressbuch bleibt; ist der Zugang nicht mehr eingerichtet, gilt keins.
func (v *adressbuchVorrat) aktuell(ctx context.Context) []adressabgleich.Eintrag {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.geholtAm.IsZero() && time.Since(v.geholtAm) < adressbuchAbstand {
		return v.eintraege
	}
	// Vom Browser gelöst: Bricht er ab, schlüge sonst der Abruf für alle fehl, die gerade warten.
	ctx, abbrechen := context.WithTimeout(context.WithoutCancel(ctx), adressbuchFrist)
	defer abbrechen()
	eintraege, err := v.quelle(ctx)
	v.geholtAm = time.Now()
	switch {
	case err == nil:
		v.eintraege = eintraege
	case errors.Is(err, adressabgleich.ErrNichtEingerichtet):
		v.eintraege = nil
	default:
		slog.Error("Anmeldung mit Kürzel: Das Adressbuch des Webmails ist nicht abrufbar", "fehler", err)
	}
	return v.eintraege
}

// finde sucht die Person zum Kürzel. Ein Kürzel ohne sichere Adresse steht im Log: Für diese
// Person greift die Anmeldung mit dem Kürzel nicht, bis das Adressbuch eindeutig ist. Fehlt im
// Adressbuch ein Teil des Namens, gilt der Name aus der Adresse wie bei der Selbstanmeldung.
func (v *adressbuchVorrat) finde(ctx context.Context, kuerzel string) (adressabgleich.Person, bool) {
	p, grund, gefunden := adressabgleich.FindeKuerzel(v.aktuell(ctx), kuerzel, SelbstanmeldeDomain())
	if !gefunden {
		return adressabgleich.Person{}, false
	}
	if grund != "" {
		slog.Error("Anmeldung mit Kürzel: Das Adressbuch nennt zum Kürzel keine sichere Adresse", "kuerzel", kuerzel, "grund", grund)
		return adressabgleich.Person{}, false
	}
	if p.Vorname == "" || p.Nachname == "" {
		p.Vorname, p.Nachname = namenAusAdresse(p.Adresse)
	}
	return p, true
}

// pruefeAnmeldungMitKuerzel behandelt eine Eingabe ohne Konto. zustaendig ist false, wenn die
// Anmeldung mit dem Kürzel nicht greift; dann geht die Anmeldung den Weg wie bisher. Ob sie greift,
// steht fest, bevor der Verzeichnisdienst gefragt wird.
func (a *Anmeldung) pruefeAnmeldungMitKuerzel(ctx context.Context, pool db.Pool, eingabe anmeldeEingabe) (konto anmeldeKonto, zustaendig bool, err error) {
	kuerzel := strings.TrimSpace(eingabe.eingabe)
	if a.adressbuch == nil || strings.Contains(kuerzel, "@") || !leseLDAPKonfig().istEingerichtet() {
		return anmeldeKonto{}, false, nil
	}
	person, gefunden := a.adressbuch.finde(ctx, kuerzel)
	if !gefunden {
		return anmeldeKonto{}, false, nil
	}
	vorhanden, zustaendig, err := kontoZurAdresse(ctx, pool, kuerzel, person)
	if err != nil || !zustaendig {
		return anmeldeKonto{}, zustaendig, err
	}
	if err := pruefeLDAPAnmeldung(ctx, "", kuerzel, eingabe.passwort); err != nil {
		return anmeldeKonto{}, true, err
	}
	if vorhanden == nil {
		konto, err = legeKontoFuerKuerzelAn(ctx, pool, kuerzel, person)
		return konto, true, err
	}
	konto, err = stelleAufKuerzelUm(ctx, pool, *vorhanden, kuerzel)
	return konto, true, err
}

// kontoZurAdresse entscheidet, ob die Anmeldung mit dem Kürzel zuständig ist, und liefert das
// Konto, das sie umstellt; nil heißt, sie legt eins an. Zuständig ist sie, wenn zur Adresse aus
// dem Adressbuch kein Konto besteht und der Name noch nicht in der Leserdatei steht, oder für ein
// aktives oder beantragtes Konto des Kollegiums. Eine andere Rolle bleibt bei der
// Benutzerverwaltung, ein deaktiviertes Konto, wie es ist.
func kontoZurAdresse(ctx context.Context, pool db.Pool, kuerzel string, p adressabgleich.Person) (*repository.AnmeldeKonto, bool, error) {
	k, err := repository.LadeAnmeldeKonto(ctx, pool, p.Adresse, p.Adresse)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// Dieselbe Regel wie beim Anlegen einer Lehrkraft an der Theke: Steht der Name schon in der
		// Leserdatei, mit Konto unter einer anderen Adresse oder ohne Konto, stünde dieselbe Person
		// sonst zweimal dort, einmal mit Ausweis und Ausleihen, einmal leer.
		doppelt, err := repository.IstLeserNamensdubletteVorhanden(ctx, pool, p.Vorname, p.Nachname)
		if err != nil {
			return nil, true, fmt.Errorf("%w: leserdatei prüfen: %w", errAnmeldedienstGestoert, err)
		}
		if doppelt {
			slog.Error("Anmeldung mit Kürzel: Der Name steht schon in der Leserdatei, kein Konto angelegt; "+
				"die Schul-E-Mail am vorhandenen Eintrag eintragen oder das Konto unter Benutzer & Rechte anlegen", "kuerzel", kuerzel)
		}
		return nil, !doppelt, nil
	case errors.Is(err, repository.ErrAnmeldungMehrdeutig):
		slog.Error("Anmeldung mit Kürzel: Die Adresse aus dem Adressbuch trifft zwei Konten", "kuerzel", kuerzel)
		return nil, false, nil
	case err != nil:
		return nil, true, fmt.Errorf("%w: konto zur adresse aus dem adressbuch lesen: %w", errAnmeldedienstGestoert, err)
	}
	switch {
	case !strings.EqualFold(k.Rolle, string(RolleKollegium)):
		return nil, false, nil
	case !k.Aktiv && !k.ZugangBeantragt:
		return nil, false, nil
	case k.Anmeldeverfahren == repository.AnmeldeverfahrenLDAP && !strings.EqualFold(k.LDAPKennung, kuerzel):
		slog.Error("Anmeldung mit Kürzel: Das Konto zur Adresse aus dem Adressbuch trägt eine andere LDAP-Kennung", "kuerzel", kuerzel, "konto", k.ID)
		return nil, false, nil
	}
	return &k, true, nil
}

// stelleAufKuerzelUm stellt ein Konto des Kollegiums auf die Anmeldung mit dem Kürzel um; ein
// offener Antrag ist damit erledigt.
func stelleAufKuerzelUm(ctx context.Context, pool db.Pool, k repository.AnmeldeKonto, kuerzel string) (anmeldeKonto, error) {
	if k.Anmeldeverfahren == repository.AnmeldeverfahrenLDAP {
		return anmeldeKontoAus(k), nil
	}
	verfahren := repository.AnmeldeverfahrenLDAP
	aenderung := repository.KontoAenderung{ID: k.ID, Anmeldeverfahren: &verfahren, LDAPKennung: &kuerzel, EmailBisher: &k.Email}
	if !k.Aktiv {
		aktiv := true
		aenderung.Aktiv = &aktiv
	}
	stand, err := repository.NewKontoAbfragen(pool).Aendere(ctx, aenderung)
	if err != nil {
		return anmeldeKonto{}, fmt.Errorf("%w: anmeldung mit kürzel: konto %s umstellen: %w", errAnmeldedienstGestoert, k.ID, err)
	}
	protokolliereKuerzelKonto(ctx, pool, k.ID, "USER_UPDATE", map[string]any{
		"ziel_id": k.ID, "email": stand.Email, "rolle": stand.Rolle, "aktiv": stand.Aktiv,
		"anmeldeverfahren": stand.Anmeldeverfahren, "ldap_kennung": stand.LDAPKennung,
	})
	return ladeKuerzelKonto(ctx, pool, kuerzel, stand.Email)
}

// legeKontoFuerKuerzelAn legt das Konto des Kollegiums über die Tür der Benutzerverwaltung an;
// die Leserzeile entsteht dabei von selbst.
func legeKontoFuerKuerzelAn(ctx context.Context, pool db.Pool, kuerzel string, p adressabgleich.Person) (anmeldeKonto, error) {
	kontoID, err := repository.NewKontoAbfragen(pool).LegeAn(ctx, repository.KontoNeu{
		Vorname: p.Vorname, Nachname: p.Nachname, Email: p.Adresse, Rolle: "kollegium",
		Anmeldeverfahren: repository.AnmeldeverfahrenLDAP, LDAPKennung: kuerzel,
	})
	// Zwei gleichzeitige erste Anmeldungen derselben Person: Die zweite trifft auf das Konto der
	// ersten und meldet sich damit an.
	if repository.IstAdressenKollision(err) {
		return ladeKuerzelKonto(ctx, pool, kuerzel, p.Adresse)
	}
	if err != nil {
		return anmeldeKonto{}, fmt.Errorf("%w: anmeldung mit kürzel: konto anlegen: %w", errAnmeldedienstGestoert, err)
	}
	protokolliereKuerzelKonto(ctx, pool, kontoID, "USER_CREATE", map[string]any{
		"ziel_id": kontoID, "email": p.Adresse, "rolle": "kollegium", "vorname": p.Vorname, "nachname": p.Nachname,
		"anmeldeverfahren": repository.AnmeldeverfahrenLDAP, "ldap_kennung": kuerzel,
	})
	return ladeKuerzelKonto(ctx, pool, kuerzel, p.Adresse)
}

// ladeKuerzelKonto liest das Konto nach dem Anlegen oder Umstellen so, wie die Anmeldung es liest.
// Es muss das Konto mit dieser Kennung sein; jedes andere Ergebnis ist ein Fehler.
func ladeKuerzelKonto(ctx context.Context, pool db.Pool, kuerzel, adresse string) (anmeldeKonto, error) {
	k, err := repository.LadeAnmeldeKonto(ctx, pool, kuerzel, adresse)
	if err != nil {
		return anmeldeKonto{}, fmt.Errorf("%w: anmeldung mit kürzel: konto lesen: %w", errAnmeldedienstGestoert, err)
	}
	if k.Anmeldeverfahren != repository.AnmeldeverfahrenLDAP || !strings.EqualFold(k.LDAPKennung, kuerzel) {
		return anmeldeKonto{}, fmt.Errorf("%w: anmeldung mit kürzel: konto %s trägt eine andere kennung", errAnmeldedienstGestoert, k.ID)
	}
	return anmeldeKontoAus(k), nil
}

// protokolliereKuerzelKonto schreibt Anlage oder Umstellung ins Protokoll, mit dem Konto selbst
// als Bearbeiter wie bei der Selbstanmeldung: Es hat sich angemeldet. Ein Fehler beim Schreiben
// hält die Anmeldung nicht auf.
func protokolliereKuerzelKonto(ctx context.Context, pool db.Pool, kontoID, aktion string, details map[string]any) {
	text, err := json.Marshal(details)
	if err != nil {
		text = []byte("{}")
	}
	if err := repository.SchreibeAdminProtokoll(ctx, pool, kontoID, aktion, "", string(text)); err != nil {
		slog.Error("Anmeldung mit Kürzel: Protokolleintrag fehlgeschlagen", "aktion", aktion, "fehler", err)
	}
}
