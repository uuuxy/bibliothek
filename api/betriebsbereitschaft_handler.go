package api

// betriebsbereitschaft_handler.go — trägt die Lage zusammen und liefert sie aus.
//
// Getrennt von den Regeln in internal/bereitschaft/bereitschaft.go, und zwar aus einem Grund: Die
// Regeln sollen ohne Umgebungsvariablen und ohne Datenbank prüfbar sein. Alles, was die
// Aussenwelt befragt, steht hier.

import (
	"bibliothek/auth"
	"bibliothek/internal/bereitschaft"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"bibliothek/internal/ausweis"
	"bibliothek/jobs"
	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// BetriebsbereitschaftResponse ist die Antwort des Endpunkts.
//
// Gesamt fasst zusammen, damit die Oberfläche nicht selbst rechnen muss — die schärfste
// Stufe gewinnt. Sonst stünde die Regel an zwei Stellen und liefe auseinander.
type BetriebsbereitschaftResponse struct {
	Gesamt  string                `json:"gesamt"`
	Befunde []bereitschaft.Befund `json:"befunde"`
}

// schaerfste liefert die höchste vorkommende Stufe.
func schaerfste(befunde []bereitschaft.Befund) string {
	gesamt := bereitschaft.StufeOK
	for _, b := range befunde {
		switch b.Stufe {
		case bereitschaft.StufeKritisch:
			return bereitschaft.StufeKritisch
		case bereitschaft.StufeWarnung:
			gesamt = bereitschaft.StufeWarnung
		}
	}
	return gesamt
}

// sammleLage trägt die Lage zusammen — geteilt zwischen dem Handler (Seite) und dem
// Bereitschafts-Alarm (tägliche Mail). Zwei Sammler wären zwei Listen, die
// auseinanderlaufen: Der Alarm meldete dann „alles gut", während die Seite rot zeigt.
func (s *Server) sammleLage(
	ctx context.Context,
	settingsRepo repository.SystemSettingsRepository,
	mailRepo *repository.MailSettingsRepository,
	zustandRepo *repository.BetriebszustandRepository,
) bereitschaft.Lage {
	lage := bereitschaft.Lage{
		AppEnv:             strings.ToLower(os.Getenv("APP_ENV")),
		S3Endpoint:         os.Getenv("S3_ENDPOINT"),
		S3AccessKey:        os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:        os.Getenv("S3_SECRET_KEY"),
		S3Bucket:           os.Getenv("S3_BUCKET"),
		EnforceProdSecrets: ErzwingeProdGeheimnisse(os.Getenv("APP_ENV"), os.Getenv("ENFORCE_PROD_SECRETS")),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		AppEncryptionKey:   os.Getenv("APP_ENCRYPTION_KEY"),
		ImapHost:           os.Getenv("IMAP_HOST"),
		// Derselbe Vergleich wie in mailservice/versand.go: Nur "true" schaltet.
		SmtpAllowInsecureTLS: os.Getenv("SMTP_ALLOW_INSECURE_TLS") == "true",
		SmtpAllowPlaintext:   os.Getenv("SMTP_ALLOW_PLAINTEXT") == "true",
		SelbstanmeldeDomain:  auth.SelbstanmeldeDomain(),
	}

	// Öffentliche Adresse und SMTP-Host kommen aus der Datenbank, nicht aus der .env:
	// Beides ist über die Oberfläche einstellbar, die .env füllt nur beim ersten Start
	// vor. Wer hier die Umgebung befragte, meldete „eingerichtet", während die
	// Anwendung längst mit einem anderen Wert arbeitet.
	if settings, err := settingsRepo.GetSettings(ctx); err == nil && settings != nil {
		einstellungenInDieLage(&lage, settings)
	}

	// DSGVO-Löschroutinen: Zustand statt Log, und zwar für ALLE, nicht nur die
	// Anonymisierung. Bei Fehler bleibt die Liste nil → Warnung „nicht erhoben" statt
	// eines falschen „alles gut".
	if rueckstand, err := zustandRepo.ZaehleLoeschRueckstand(ctx); err == nil {
		lage.LoeschRueckstand = rueckstand
	}
	// Ehemalige mit offenen Vorgängen: bei Fehler nil → „nicht erhoben" statt „alles gut".
	if n, err := zustandRepo.ZaehleEhemaligeMitOffenenVorgaengen(ctx, bereitschaft.EhemaligeOffenSeitTagen); err == nil {
		lage.EhemaligeMitOffenenVorgaengen = &n
	}
	// Nachbuch-Meldungen, die seit zwei Wochen niemand quittiert hat: bei Fehler nil.
	if n, err := zustandRepo.ZaehleNachbuchMeldungenOffenSeit(ctx, bereitschaft.NachbuchOffenSeitTagen); err == nil {
		lage.NachbuchMeldungenOffen = &n
	}
	if mail, err := mailRepo.GetConfig(ctx); err == nil && mail != nil {
		lage.SmtpHost = strings.TrimSpace(mail.SMTPHost)
	}

	// Ein Fehler hier ist kein Grund, die ganze Auskunft zu verweigern: Der Bereich ist
	// eine Warnung, und 0 heisst schlicht „keine gefunden".
	if anzahl, err := zustandRepo.ZaehleDemoSchueler(ctx); err == nil {
		lage.DemoSchueler = anzahl
	}
	if anzahl, err := zustandRepo.ZaehleDemoExemplare(ctx); err == nil {
		lage.DemoExemplare = anzahl
	}
	// Bei einem Fehler bleibt die Liste nil — „nicht lesbar", nicht „keiner".
	if namen, err := zustandRepo.BeispielLieferanten(ctx); err == nil {
		lage.BeispielLieferanten = namen
	}

	rechteKlassenUndAdminsInDieLage(ctx, zustandRepo, &lage)

	if probe, err := repository.PruefeSchluesselGegenBestand(ctx, s.DB.Pool); err == nil {
		lage.SchluesselProbe = &probe
	}

	sicherungUndFerienInDieLage(ctx, zustandRepo, &lage)
	return lage
}

// einstellungenInDieLage übernimmt, was die Prüfungen aus den Einstellungen brauchen.
func einstellungenInDieLage(lage *bereitschaft.Lage, settings *repository.SystemEinstellungen) {
	if settings.OeffentlicheAdresse != nil {
		lage.OeffentlicheAdresse = strings.TrimSpace(*settings.OeffentlicheAdresse)
	}
	if settings.AlarmEmpfaenger != nil {
		lage.AlarmEmpfaenger = strings.TrimSpace(*settings.AlarmEmpfaenger)
	}
	// Schadensersatz-Bescheid: dieselbe Prüfung, mit der das Erstellen abweist.
	// FehlendeAngaben liefert nil, wenn nichts fehlt — in der Lage heißt nil aber
	// „nicht erhoben", also wird daraus die leere Liste.
	lage.BescheidFehlend = repository.BescheidAngabenAus(settings).FehlendeAngaben(repository.SchuleAngabenAus(settings))
	if lage.BescheidFehlend == nil {
		lage.BescheidFehlend = []string{}
	}
}

// rechteKlassenUndAdminsInDieLage liest Rechte, Klassen-Zuordnungen und Admin-Konten. Bei
// einem Lesefehler bleibt die jeweilige Liste nil: Die Prüfung meldet dann „nicht lesbar"
// oder „nicht erhoben" statt eines falschen „alles gut".
func rechteKlassenUndAdminsInDieLage(ctx context.Context, zustandRepo *repository.BetriebszustandRepository, lage *bereitschaft.Lage) {
	if rechte, err := zustandRepo.LadeRollenRechte(ctx); err == nil {
		lage.RechteLive = rechte
	}

	if schueler, zuordnungen, listen, err := zustandRepo.KlassenBestand(ctx); err == nil {
		lage.KlassenOhneLehrkraft = fehlendeEintraege(mitKlassenleitung(schueler), zuordnungen)
		lage.VerwaisteZuordnungen = fehlendeEintraege(zuordnungen, schueler)
		lage.VerwaisteBuecherliste = fehlendeEintraege(listen, schueler)
	}

	if admins, err := zustandRepo.AktiveAdmins(ctx); err == nil {
		lage.AdminKonten = make([]string, 0, len(admins))
		for _, a := range admins {
			lage.AdminKonten = append(lage.AdminKonten, a.Name+" ("+a.Email+")")
		}
	}
}

// sicherungUndFerienInDieLage trägt den Stand der Backups, die Reichweite der Ferientabelle
// und das Ergebnis der Restore-Probe ein.
func sicherungUndFerienInDieLage(ctx context.Context, zustandRepo *repository.BetriebszustandRepository, lage *bereitschaft.Lage) {
	// Backup-Zustand aus derselben Quelle wie das Dashboard-Badge (backup_status.go).
	encKey := os.Getenv("BACKUP_ENCRYPTION_KEY")
	lage.BackupKeySet = encKey != ""
	lage.BackupKeyWeak = jobs.SchluesselIstSchwach(encKey)
	backupDir := os.Getenv("BACKUP_DIR")
	if backupDir == "" {
		backupDir = "./backups" // identischer Default wie jobs/backup.go
	}
	lage.LetztesBackup = newestBackupTime(backupDir)
	// Schulzeitzone, nicht die Zone des Containers: Mit TZ=UTC wäre `Jetzt.Year()` in der
	// Stunde nach Berliner Mitternacht des 1. Januar noch das alte Jahr.
	lage.Jetzt = schulzeit.Jetzt()
	// Programmtabelle plus die eingestellten Jahre (Einstellungen → LUSD & Versetzung);
	// ein Lesefehler zählt wie „nichts eingestellt".
	sommerferien, err := zustandRepo.LadeEinstellungswert(ctx, lmfplan.SommerferienSchluessel)
	if err != nil {
		sommerferien = ""
	}
	// Lückenlos ab dem laufenden Jahr, nicht das Maximum: Ein vergessenes Jahr mitten in
	// der Liste lässt den Planer ohne Vorgabe stehen, und genau das soll die Prüfung sagen.
	lage.FerientabelleBis = lmfplan.FerientabelleAus(sommerferien).LueckenlosBis(lage.Jetzt.Year())
	lage.UebrigeFerienBis = lmfplan.UebrigeFerienBis()

	// Ergebnis der wöchentlichen Restore-Probe. Unlesbar oder nie gelaufen → nil,
	// die Prüfung meldet dann „noch kein Probelauf" statt eines falschen Urteils.
	if probeJSON, err := zustandRepo.LadeEinstellungswert(ctx, jobs.RestoreProbeSchluessel); err == nil {
		var probe jobs.RestoreProbeErgebnis
		if json.Unmarshal([]byte(probeJSON), &probe) == nil {
			lage.RestoreProbe = &probe
		}
	}
}

// BetriebsbereitschaftHandler beantwortet: Was ist eingerichtet, aber nicht in Betrieb?
// GET /api/admin/system/betriebsbereitschaft
func (s *Server) BetriebsbereitschaftHandler(
	settingsRepo repository.SystemSettingsRepository,
	mailRepo *repository.MailSettingsRepository,
	zustandRepo *repository.BetriebszustandRepository,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		befunde := bereitschaft.Pruefe(s.sammleLage(r.Context(), settingsRepo, mailRepo, zustandRepo))
		RespondJSON(w, http.StatusOK, BetriebsbereitschaftResponse{
			Gesamt:  schaerfste(befunde),
			Befunde: befunde,
		})
	}
}

// mitKlassenleitung lässt die Klassen übrig, für die die Schule eine Klassenleitung hat: bis
// Jahrgang 10. Die Oberstufe (ET, 12T, 13T) hat keine, und ein Name ohne Ziffer („ABG") ist
// ein Sonderwert.
func mitKlassenleitung(klassen []string) []string {
	mit := []string{}
	for _, k := range klassen {
		if _, jahrgang, lesbar := ausweis.AblaufJahrgang(k); lesbar {
			if jahrgang <= ausweis.AbschlussMittelstufe {
				mit = append(mit, k)
			}
			continue
		}
		// Eine Zahl außerhalb 1 bis 13 („70R1") bleibt gemeldet: Ihre Mahnliste erreicht niemanden.
		if k != "" && k[0] >= '0' && k[0] <= '9' {
			mit = append(mit, k)
		}
	}
	return mit
}

// fehlendeEintraege liefert alle Werte aus `menge`, die in `referenz` fehlen —
// nie nil, damit „geprüft und leer" von „nicht erhoben" (nil) unterscheidbar bleibt.
func fehlendeEintraege(menge, referenz []string) []string {
	bekannt := make(map[string]bool, len(referenz))
	for _, r := range referenz {
		bekannt[r] = true
	}
	fehlt := []string{}
	for _, m := range menge {
		if !bekannt[m] {
			fehlt = append(fehlt, m)
		}
	}
	return fehlt
}
