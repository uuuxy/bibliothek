package api

// betriebsbereitschaft_handler.go — trägt die Lage zusammen und liefert sie aus.
//
// Getrennt von den Regeln in betriebsbereitschaft.go, und zwar aus einem Grund: Die
// Regeln sollen ohne Umgebungsvariablen und ohne Datenbank prüfbar sein. Alles, was die
// Aussenwelt befragt, steht hier.

import (
	"bibliothek/repository"
	"net/http"
	"os"
	"strings"
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

// BetriebsbereitschaftHandler beantwortet: Was ist eingerichtet, aber nicht in Betrieb?
// GET /api/admin/system/betriebsbereitschaft
func (s *Server) BetriebsbereitschaftHandler(
	settingsRepo repository.SystemSettingsRepository,
	mailRepo *repository.MailSettingsRepository,
	zustandRepo *repository.BetriebszustandRepository,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		if settings, err := settingsRepo.GetSettings(r.Context()); err == nil && settings != nil {
			if settings.OeffentlicheAdresse != nil {
				lage.OeffentlicheAdresse = strings.TrimSpace(*settings.OeffentlicheAdresse)
			}
		}
		if mail, err := mailRepo.GetConfig(r.Context()); err == nil && mail != nil {
			lage.SmtpHost = strings.TrimSpace(mail.SMTPHost)
		}

		// Ein Fehler hier ist kein Grund, die ganze Auskunft zu verweigern: Der Bereich ist
		// eine Warnung, und 0 heisst schlicht „keine gefunden".
		if anzahl, err := zustandRepo.ZaehleDemoSchueler(r.Context()); err == nil {
			lage.DemoSchueler = anzahl
		}

		befunde := Pruefe(lage)
		RespondJSON(w, http.StatusOK, BetriebsbereitschaftResponse{
			Gesamt:  schaerfste(befunde),
			Befunde: befunde,
		})
	}
}
