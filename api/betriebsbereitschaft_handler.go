package api

// betriebsbereitschaft_handler.go — trägt die Lage zusammen und liefert sie aus.
//
// Getrennt von den Regeln in betriebsbereitschaft.go, und zwar aus einem Grund: Die
// Regeln sollen ohne Umgebungsvariablen und ohne Datenbank prüfbar sein. Alles, was die
// Aussenwelt befragt, steht hier.

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"bibliothek/jobs"
	"bibliothek/repository"
)

// BetriebsbereitschaftResponse ist die Antwort des Endpunkts.
//
// Gesamt fasst zusammen, damit die Oberfläche nicht selbst rechnen muss — die schärfste
// Stufe gewinnt. Sonst stünde die Regel an zwei Stellen und liefe auseinander.
type BetriebsbereitschaftResponse struct {
	Gesamt  string   `json:"gesamt"`
	Befunde []Befund `json:"befunde"`
}

// schaerfste liefert die höchste vorkommende Stufe.
func schaerfste(befunde []Befund) string {
	gesamt := StufeOK
	for _, b := range befunde {
		switch b.Stufe {
		case StufeKritisch:
			return StufeKritisch
		case StufeWarnung:
			gesamt = StufeWarnung
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
) Lage {
	lage := Lage{
		AppEnv:             strings.ToLower(os.Getenv("APP_ENV")),
		S3Endpoint:         os.Getenv("S3_ENDPOINT"),
		S3AccessKey:        os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:        os.Getenv("S3_SECRET_KEY"),
		S3Bucket:           os.Getenv("S3_BUCKET"),
		EnforceProdSecrets: strings.ToLower(os.Getenv("ENFORCE_PROD_SECRETS")) == "true",
		JWTSecret:          os.Getenv("JWT_SECRET"),
		AppEncryptionKey:   os.Getenv("APP_ENCRYPTION_KEY"),
		ImapHost:           os.Getenv("IMAP_HOST"),
	}

	// Öffentliche Adresse und SMTP-Host kommen aus der DATENBANK, nicht aus der .env:
	// Beides ist über die Oberfläche einstellbar, die .env füllt nur beim ersten Start
	// vor. Wer hier die Umgebung befragte, meldete „eingerichtet", während die
	// Anwendung längst mit einem anderen Wert arbeitet — genau der Grund, warum beim
	// Mail-Debugging die DB-Zeile gilt und nicht die .env.
	if settings, err := settingsRepo.GetSettings(ctx); err == nil && settings != nil {
		if settings.OeffentlicheAdresse != nil {
			lage.OeffentlicheAdresse = strings.TrimSpace(*settings.OeffentlicheAdresse)
		}
		if settings.AlarmEmpfaenger != nil {
			lage.AlarmEmpfaenger = strings.TrimSpace(*settings.AlarmEmpfaenger)
		}
		lage.LesehistorieAus = repository.TageOderStandard(settings.LesehistorieTage, repository.StandardLesehistorieTage) <= 0
	}

	// DSGVO-Löschroutinen: Zustand statt Log. Bei Fehler bleibt DsgvoFaellig nil → Warnung
	// „nicht erhoben" statt eines falschen „alles gut".
	if n, err := zustandRepo.ZaehleFaelligeAnonymisierungen(ctx); err == nil {
		lage.DsgvoFaellig = &n
	}
	if mail, err := mailRepo.GetConfig(ctx); err == nil && mail != nil {
		lage.SmtpHost = strings.TrimSpace(mail.SMTPHost)
	}

	// Ein Fehler hier ist kein Grund, die ganze Auskunft zu verweigern: Der Bereich ist
	// eine Warnung, und 0 heisst schlicht „keine gefunden".
	if anzahl, err := zustandRepo.ZaehleDemoSchueler(ctx); err == nil {
		lage.DemoSchueler = anzahl
	}

	// Bei einem Fehler bleibt RechteLive nil — die Prüfung meldet dann „nicht
	// lesbar" statt fälschlich jede Vorgabe-Zeile als fehlend zu deklarieren.
	if rechte, err := zustandRepo.LadeRollenRechte(ctx); err == nil {
		lage.RechteLive = rechte
	}

	// Klassen-Drift (F3): Bei einem Fehler bleiben die Listen nil — die Prüfung
	// meldet dann „nicht erhoben" statt fälschlich „alles verbunden".
	if schueler, zuordnungen, listen, err := zustandRepo.KlassenBestand(ctx); err == nil {
		lage.KlassenOhneLehrkraft = fehlendeEintraege(schueler, zuordnungen)
		lage.VerwaisteZuordnungen = fehlendeEintraege(zuordnungen, schueler)
		lage.VerwaisteBuecherliste = fehlendeEintraege(listen, schueler)
	}

	if admins, err := zustandRepo.AktiveAdmins(ctx); err == nil {
		lage.AdminKonten = make([]string, 0, len(admins))
		for _, a := range admins {
			lage.AdminKonten = append(lage.AdminKonten, a.Name+" ("+a.Email+")")
		}
	}

	// Backup-Zustand aus derselben Quelle wie das Dashboard-Badge (backup_status.go).
	encKey := os.Getenv("BACKUP_ENCRYPTION_KEY")
	lage.BackupKeySet = encKey != ""
	lage.BackupKeyWeak = jobs.SchluesselIstSchwach(encKey)
	backupDir := os.Getenv("BACKUP_DIR")
	if backupDir == "" {
		backupDir = "./backups" // identischer Default wie jobs/backup.go
	}
	lage.LetztesBackup = newestBackupTime(backupDir)
	lage.Jetzt = time.Now()

	// Ergebnis der wöchentlichen Restore-Probe. Unlesbar oder nie gelaufen → nil,
	// die Prüfung meldet dann „noch kein Probelauf" statt eines falschen Urteils.
	if probeJSON, err := zustandRepo.LadeEinstellungswert(ctx, jobs.RestoreProbeSchluessel); err == nil {
		var probe jobs.RestoreProbeErgebnis
		if json.Unmarshal([]byte(probeJSON), &probe) == nil {
			lage.RestoreProbe = &probe
		}
	}

	return lage
}

// BetriebsbereitschaftHandler beantwortet: Was ist eingerichtet, aber nicht in Betrieb?
// GET /api/admin/system/betriebsbereitschaft
func (s *Server) BetriebsbereitschaftHandler(
	settingsRepo repository.SystemSettingsRepository,
	mailRepo *repository.MailSettingsRepository,
	zustandRepo *repository.BetriebszustandRepository,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		befunde := Pruefe(s.sammleLage(r.Context(), settingsRepo, mailRepo, zustandRepo))
		RespondJSON(w, http.StatusOK, BetriebsbereitschaftResponse{
			Gesamt:  schaerfste(befunde),
			Befunde: befunde,
		})
	}
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
