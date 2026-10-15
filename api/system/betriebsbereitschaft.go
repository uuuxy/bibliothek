package system

// Die Tür der Betriebsbereitschaft trägt die Lage zusammen und liefert sie aus.
//
// Die Regeln stehen in internal/bereitschaft und sind ohne Umgebungsvariablen und ohne
// Datenbank prüfbar. Hier steht, was Umgebung, Einstellungen und Datenbank befragt.

import (
	"bibliothek/auth"
	"bibliothek/internal/bereitschaft"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/jobs"
	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// betriebsbereitschaftAntwort ist die Antwort des Endpunkts. Gesamt fasst die Befunde zusammen
// (bereitschaft.Gesamt), damit die Oberfläche nicht selbst rechnet.
type betriebsbereitschaftAntwort struct {
	Gesamt  string                `json:"gesamt"`
	Befunde []bereitschaft.Befund `json:"befunde"`
}

// sammleLage trägt die Lage zusammen, geteilt zwischen der Tür (Seite) und dem
// Bereitschafts-Alarm (tägliche Mail). Zwei Sammler wären zwei Listen, die
// auseinanderlaufen: Der Alarm meldete dann „alles gut", während die Seite rot zeigt.
func (t *Tueren) sammleLage(
	ctx context.Context,
	einstellungsAbfragen repository.EinstellungsAbfragen,
	mailAbfragen *repository.MailEinstellungsAbfragen,
	zustandAbfragen *repository.BetriebszustandAbfragen,
) bereitschaft.Lage {
	lage := bereitschaft.Lage{
		Umgebung:              strings.ToLower(os.Getenv("APP_ENV")),
		S3Endpunkt:            os.Getenv("S3_ENDPOINT"),
		S3Zugangsschluessel:   os.Getenv("S3_ACCESS_KEY"),
		S3Geheimschluessel:    os.Getenv("S3_SECRET_KEY"),
		S3Bucket:              os.Getenv("S3_BUCKET"),
		GeheimnisSchutzScharf: bereitschaft.IstGeheimnisSchutzScharf(os.Getenv("APP_ENV"), os.Getenv("ENFORCE_PROD_SECRETS")),
		JWTGeheimnis:          os.Getenv("JWT_SECRET"),
		Schluessel:            os.Getenv("APP_ENCRYPTION_KEY"),
		IMAPHost:              os.Getenv("IMAP_HOST"),
		// Derselbe Vergleich wie in mailservice/versand.go: Nur "true" schaltet.
		SMTPUnsicheresTLSErlaubt: os.Getenv("SMTP_ALLOW_INSECURE_TLS") == "true",
		SMTPKlartextErlaubt:      os.Getenv("SMTP_ALLOW_PLAINTEXT") == "true",
		SelbstanmeldeDomain:      auth.SelbstanmeldeDomain(),
		VertrauteProxys:          t.VertrauteProxys,
	}

	// Öffentliche Adresse und SMTP-Host kommen aus der Datenbank, nicht aus der .env:
	// Beides ist über die Oberfläche einstellbar, die .env füllt nur beim ersten Start
	// vor. Wer hier die Umgebung befragte, meldete „eingerichtet", während die
	// Anwendung längst mit einem anderen Wert arbeitet.
	if einstellungen, err := einstellungsAbfragen.Lade(ctx); err == nil && einstellungen != nil {
		einstellungenInDieLage(&lage, einstellungen)
	}

	// DSGVO-Löschroutinen: Zustand statt Log, und zwar für alle, nicht nur die
	// Anonymisierung. Bei Fehler bleibt die Liste nil → Warnung „nicht erhoben" statt
	// eines falschen „alles gut".
	if rueckstand, err := zustandAbfragen.ZaehleLoeschRueckstand(ctx); err == nil {
		lage.LoeschRueckstand = rueckstand
	}
	// Ehemalige mit offenen Vorgängen: bei Fehler nil → „nicht erhoben" statt „alles gut".
	if n, err := zustandAbfragen.ZaehleEhemaligeMitOffenenVorgaengen(ctx, bereitschaft.EhemaligeOffenSeitTagen); err == nil {
		lage.EhemaligeMitOffenenVorgaengen = &n
	}
	// Nachbuch-Meldungen, die seit zwei Wochen niemand quittiert hat: bei Fehler nil.
	if n, err := zustandAbfragen.ZaehleNachbuchMeldungenOffenSeit(ctx, bereitschaft.NachbuchOffenSeitTagen); err == nil {
		lage.NachbuchMeldungenOffen = &n
	}
	if mail, err := mailAbfragen.Lade(ctx); err == nil && mail != nil {
		lage.SMTPHost = strings.TrimSpace(mail.SMTPHost)
	}

	// Ein Fehler hier ist kein Grund, die ganze Auskunft zu verweigern: Der Bereich ist
	// eine Warnung, und 0 heisst schlicht „keine gefunden".
	if anzahl, err := zustandAbfragen.ZaehleDemoSchueler(ctx); err == nil {
		lage.DemoSchueler = anzahl
	}
	if anzahl, err := zustandAbfragen.ZaehleDemoExemplare(ctx); err == nil {
		lage.DemoExemplare = anzahl
	}
	// Bei einem Fehler bleibt die Liste nil — „nicht lesbar", nicht „keiner".
	if namen, err := zustandAbfragen.LadeBeispielLieferanten(ctx); err == nil {
		lage.BeispielLieferanten = namen
	}

	rechteKlassenUndAdminsInDieLage(ctx, zustandAbfragen, &lage)

	if probe, err := repository.PruefeSchluesselGegenBestand(ctx, t.DB.Pool); err == nil {
		lage.SchluesselProbe = &probe
	}

	sicherungUndFerienInDieLage(ctx, zustandAbfragen, &lage)
	return lage
}

// einstellungenInDieLage übernimmt, was die Prüfungen aus den Einstellungen brauchen.
func einstellungenInDieLage(lage *bereitschaft.Lage, einstellungen *repository.SystemEinstellungen) {
	lage.OeffentlicheAdresse = einstellungen.AdresseFuerLinks()
	if einstellungen.AlarmEmpfaenger != nil {
		lage.AlarmEmpfaenger = strings.TrimSpace(*einstellungen.AlarmEmpfaenger)
	}
	// Schadensersatz-Bescheid: dieselbe Prüfung, mit der das Erstellen abweist.
	// FehlendeAngaben liefert nil, wenn nichts fehlt — in der Lage heißt nil aber
	// „nicht erhoben", also wird daraus die leere Liste.
	lage.BescheidFehlend = repository.BescheidAngabenAus(einstellungen).FehlendeAngaben(repository.SchuleAngabenAus(einstellungen))
	if lage.BescheidFehlend == nil {
		lage.BescheidFehlend = []string{}
	}
}

// rechteKlassenUndAdminsInDieLage liest Rechte, Klassen-Zuordnungen und Admin-Konten. Bei
// einem Lesefehler bleibt die jeweilige Liste nil: Die Prüfung meldet dann „nicht lesbar"
// oder „nicht erhoben" statt eines falschen „alles gut".
func rechteKlassenUndAdminsInDieLage(ctx context.Context, zustandAbfragen *repository.BetriebszustandAbfragen, lage *bereitschaft.Lage) {
	if rechte, err := zustandAbfragen.LadeRollenRechte(ctx); err == nil {
		lage.RechteLive = rechte
	}

	if schueler, zuordnungen, listen, err := zustandAbfragen.LadeKlassenBestand(ctx); err == nil {
		lage.TrageKlassenEin(schueler, zuordnungen, listen)
	}

	if admins, err := zustandAbfragen.LadeAktiveAdmins(ctx); err == nil {
		lage.AdminKonten = make([]string, 0, len(admins))
		for _, a := range admins {
			lage.AdminKonten = append(lage.AdminKonten, a.Name+" ("+a.Email+")")
		}
	}
}

// sicherungUndFerienInDieLage trägt den Stand der Backups, die Reichweite der Ferientabelle
// und das Ergebnis der Restore-Probe ein.
func sicherungUndFerienInDieLage(ctx context.Context, zustandAbfragen *repository.BetriebszustandAbfragen, lage *bereitschaft.Lage) {
	// Zustand der Backups aus derselben Quelle wie BackupStand (backup_stand.go).
	schluessel := os.Getenv("BACKUP_ENCRYPTION_KEY")
	lage.BackupSchluesselGesetzt = schluessel != ""
	lage.BackupSchluesselSchwach = jobs.IstSchluesselSchwach(schluessel)
	backupVerzeichnis := os.Getenv("BACKUP_DIR")
	if backupVerzeichnis == "" {
		backupVerzeichnis = "./backups" // dieselbe Vorgabe wie jobs/backup.go
	}
	lage.LetztesBackup = juengstesBackup(backupVerzeichnis)
	// Schulzeitzone, nicht die Zone des Containers: Mit TZ=UTC wäre `Jetzt.Year()` in der
	// Stunde nach Berliner Mitternacht des 1. Januar noch das alte Jahr.
	lage.Jetzt = schulzeit.Jetzt()
	// Programmtabelle plus die eingestellten Jahre (Einstellungen → LUSD & Versetzung);
	// ein Lesefehler zählt wie „nichts eingestellt".
	sommerferien, err := zustandAbfragen.LadeEinstellungswert(ctx, lmfplan.SommerferienSchluessel)
	if err != nil {
		sommerferien = ""
	}
	// Lückenlos ab dem laufenden Jahr, nicht das Maximum: Ein vergessenes Jahr mitten in
	// der Liste lässt den Planer ohne Vorgabe stehen, und genau das soll die Prüfung sagen.
	lage.FerientabelleBis = lmfplan.FerientabelleAus(sommerferien).LueckenlosBis(lage.Jetzt.Year())
	lage.UebrigeFerienBis = lmfplan.UebrigeFerienBis()

	// Ergebnis der wöchentlichen Restore-Probe. Unlesbar oder nie gelaufen → nil,
	// die Prüfung meldet dann „noch kein Probelauf" statt eines falschen Urteils.
	if probeJSON, err := zustandAbfragen.LadeEinstellungswert(ctx, jobs.RestoreProbeSchluessel); err == nil {
		var probe jobs.RestoreProbeErgebnis
		if json.Unmarshal([]byte(probeJSON), &probe) == nil {
			lage.RestoreProbe = &probe
		}
	}
}

// Betriebsbereitschaft beantwortet: Was ist eingerichtet, aber nicht in Betrieb?
// GET /api/admin/system/betriebsbereitschaft
func (t *Tueren) Betriebsbereitschaft(
	einstellungsAbfragen repository.EinstellungsAbfragen,
	mailAbfragen *repository.MailEinstellungsAbfragen,
	zustandAbfragen *repository.BetriebszustandAbfragen,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		befunde := bereitschaft.Pruefe(t.sammleLage(r.Context(), einstellungsAbfragen, mailAbfragen, zustandAbfragen))
		tuer.AntworteJSON(w, http.StatusOK, betriebsbereitschaftAntwort{
			Gesamt:  bereitschaft.Gesamt(befunde),
			Befunde: befunde,
		})
	}
}
