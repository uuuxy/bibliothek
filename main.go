/*
 * Dieses Programm ist freie Software: Sie können es unter den Bedingungen
 * der European Union Public Licence (EUPL), Version 1.2 (oder jeder späteren
 * Version, die von der Europäischen Kommission veröffentlicht wird),
 * weitergeben und/oder modifizieren.
 * * Dieses Programm wird in der Hoffnung vertrieben, dass es nützlich sein wird,
 * jedoch OHNE JEDE GARANTIE; auch ohne die implizite Garantie der
 * MARKTGÄNGIGKEIT oder der EIGNUNG FÜR EINEN BESTIMMTEN ZWECK.
 * Weitere Details finden Sie in der vollständigen EUPL 1.2.
 * * Eine Kopie der EUPL 1.2 sollte in diesem Repository unter der Datei LICENSE
 * verfügbar sein. Andernfalls siehe: https://joinup.ec.europa.eu/collection/eupl/eupl-text-eupl-12
 */

package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"bibliothek/api"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/crypto"
	"bibliothek/internal/service"
	"bibliothek/jobs"
	"bibliothek/pkg/clientip"
	"bibliothek/repository"
	"bibliothek/sse"

	"github.com/getsentry/sentry-go"
)

// @title           Schulbibliothek API
// @version         1.0
// @description     Backend-API fuer das Schulbibliothek-Verwaltungssystem.
// @host            localhost:8080
// @BasePath        /api

// main is the entry point of the school library system backend application.
// It initializes configs, setups database connection pools, starts the SSE broker,
// mounts middleware-protected routes, and starts the server with graceful shutdown.

func startServer(port string, server *api.Server) *http.Server {
	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: server.Routes(),

		// ReadHeaderTimeout begrenzt das Senden der Kopfzeilen. Es begrenzt NICHT das
		// Senden des Rumpfes: Wer korrekte Header schickt, "Content-Length: 10000000"
		// ankündigt und danach ein Byte pro Minute nachreicht, hielt bisher Verbindung,
		// Goroutine und ggf. eine Datenbankverbindung beliebig lange fest. Die
		// TimeoutMiddleware half dagegen nicht — ein Kontext-Deadline bricht kein
		// blockierendes Read auf der Verbindung ab.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       api.StandardLesefrist,

		// IdleTimeout gilt zwischen zwei Anfragen derselben Keep-Alive-Verbindung.
		// Ohne den Wert übernimmt Go hier ReadTimeout — was funktioniert, aber die
		// beiden Fristen aneinanderkettet; getrennt gesetzt bleibt jede änderbar.
		IdleTimeout: 120 * time.Second,

		// BEWUSST KEIN WriteTimeout: Es gilt für die gesamte Antwort, und der
		// SSE-Stream (/events) ist definitionsgemäß eine Antwort, die nie endet. Ein
		// globaler Wert würde jede Live-Verbindung nach Ablauf kappen — die Oberfläche
		// verlöre reihum ihre Aktualisierung, ohne dass ein Fehler sichtbar würde.
		// Die Schreibrichtung begrenzt stattdessen Caddy (write_timeout 600s) und für
		// die Bearbeitungsdauer die TimeoutMiddleware (api.RequestFrist).
	}

	// Start server asynchronously
	go func() {
		slog.Info("Server listening", "url", "http://localhost:"+port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	return httpServer
}

// startBereitschaftsWaechter meldet kritische Befunde der Selbstprüfung täglich per
// Mail an die Admins — der Wächter, der sich meldet, statt gelesen werden zu müssen
// (api/betriebsbereitschaft_alarm.go; auf Spielwiesen schweigt er von selbst).
// Erster Lauf 3 Minuten nach dem Start: ein Deploy mit kaputter Umgebung meldet sich
// so noch am selben Vormittag, nicht erst am nächsten Tag.
func startBereitschaftsWaechter(ctx context.Context, server *api.Server) {
	go func() {
		select {
		case <-time.After(3 * time.Minute):
			server.BereitschaftsAlarm(ctx)
		case <-ctx.Done():
			return
		}

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				server.BereitschaftsAlarm(ctx)
			case <-ctx.Done():
				slog.Info("Background Worker: Bereitschafts-Wächter gracefully stopped.")
				return
			}
		}
	}()
}

func startGDPRWorker(ctx context.Context, scheduler *jobs.Scheduler) {
	go func() {
		slog.Info("Background Worker: Running initial GDPR cleanup on startup...")
		scheduler.RunGDPRAnonymizeLoans()
		scheduler.RunGDPRDeleteAbgaenger()

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				slog.Info("Background Worker: Running scheduled 24h GDPR cleanup...")
				scheduler.RunGDPRAnonymizeLoans()
				scheduler.RunGDPRDeleteAbgaenger()
			case <-ctx.Done():
				slog.Info("Background Worker: GDPR worker gracefully stopped.")
				return
			}
		}
	}()
}

func setupDatabase(ctx context.Context, dsn string) *db.Database {
	slog.Info("Establishing database connection pool...")
	database, err := db.Connect(ctx, dsn)
	if err != nil {
		slog.Error("Database connection pool failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Database connection pool successfully initialized.")

	slog.Info("Running database migrations...")
	if err := database.RunMigrations(ctx, "migrations"); err != nil {
		slog.Error("Database migration failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Initializing role permissions...")
	if err := database.InitPermissions(ctx); err != nil {
		slog.Error("Database permission initialization failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Initializing suppliers...")
	if err := database.InitLieferanten(ctx); err != nil {
		slog.Error("Database supplier initialization failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Bootstrapping initial admin (if database is empty)...")
	if err := database.InitAdmin(ctx); err != nil {
		slog.Error("Admin bootstrapping failed", "error", err)
		os.Exit(1)
	}

	// Muss vor dem ersten Versand laufen: Übernimmt die SMTP-Zugangsdaten aus der
	// Umgebung in die Datenbank, solange dort die Schema-Vorgabe steht. Ab dann gilt
	// die Konfiguration aus der Oberfläche — ohne diese Übernahme gingen die Mahnungen
	// nach dem Umstieg an localhost:1025.
	// Meldet sich nur, wenn wirklich übernommen wurde — sonst stünde bei jedem Start
	// "übernehme", obwohl nichts passiert.
	slog.Info("Prüfe gespeicherte SMTP-Konfiguration...")
	if err := database.InitMailKonfig(ctx); err != nil {
		slog.Error("Übernahme der Mail-Konfiguration fehlgeschlagen", "error", err)
		os.Exit(1)
	}

	return database
}

// ermittleCookieSecure entscheidet über das Secure-Flag der Sitzungs- und
// CSRF-Cookies.
//
// Vorher galt bei fehlender oder unlesbarer Variable still `false` — ausgerechnet
// die unsichere Richtung. Ein vergessenes COOKIE_SECURE im Deploy reichte, damit
// Sitzungscookies ohne Secure ausgeliefert werden; auffallen würde das niemandem,
// weil über HTTPS trotzdem alles funktioniert. Jetzt ist die Vorgabe außerhalb der
// lokalen Entwicklung `true`: Wer es unsicher will, muss es hinschreiben.
//
// Bewusst KEIN harter Abbruch bei explizitem false: Der geplante Server in der
// Schule läuft möglicherweise über schlichtes HTTP im LAN, und dort würde ein
// Secure-Cookie den Login unmöglich machen (der Browser sendet es über http nicht
// mit). Diese Entscheidung darf der Betrieb treffen — sie soll nur nicht
// versehentlich passieren, deshalb die deutliche Warnung.
func ermittleCookieSecure() bool {
	appEnv := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	lokal := appEnv == "local" || appEnv == "development" || appEnv == "test"

	roh, gesetzt := os.LookupEnv("COOKIE_SECURE")
	if !gesetzt || strings.TrimSpace(roh) == "" {
		if lokal {
			return false
		}
		slog.Warn("COOKIE_SECURE ist nicht gesetzt — sichere Vorgabe (true) wird verwendet",
			"app_env", appEnv)
		return true
	}

	wert, err := strconv.ParseBool(strings.TrimSpace(roh))
	if err != nil {
		// Ein unlesbarer Wert ist immer ein Fehler und nie eine Absicht. Ihn als
		// "false" zu deuten, war genau die stille Fehlannahme von vorher.
		log.Fatalf("FATAL: COOKIE_SECURE=%q ist kein Wahrheitswert (erlaubt: true/false)", roh)
	}

	if !wert && !lokal {
		slog.Warn("COOKIE_SECURE=false außerhalb der lokalen Entwicklung — Sitzungscookies werden OHNE Secure-Flag ausgeliefert. Nur zulässig, wenn die Anwendung ausschließlich über HTTP im lokalen Netz erreichbar ist.",
			"app_env", appEnv)
	}
	return wert
}

func loadConfig() (dsn, jwtSecret, port string, cookieSecure bool) {
	dsn = os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatalf("FATAL: DATABASE_URL environment variable is required and cannot be empty")
	}

	jwtSecret = os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		log.Fatalf("FATAL: JWT_SECRET environment variable must be at least 32 characters long for security")
	}

	aesKey := os.Getenv(crypto.SchluesselVariable)
	if len(aesKey) != 32 && len(aesKey) != 64 {
		log.Fatalf("FATAL: %s must be exactly 32 bytes (or 64 hex characters) long", crypto.SchluesselVariable)
	}

	// Der frühere Zweitname darf nicht still danebenstehen. internal/crypto las ihn bis
	// zum 06.08.2026 VORRANGIG — an allen Prüfungen dieser Funktion vorbei. Er wird jetzt
	// nicht mehr gelesen; wer ihn gesetzt hat, erwartet aber, dass er gilt. Also lieber
	// hier laut abbrechen als verschlüsseln, womit der Betrieb nicht rechnet.
	if alt := os.Getenv(crypto.AltName); alt != "" && alt != aesKey {
		log.Fatalf("FATAL: %s ist gesetzt und weicht von %s ab. Der Zweitname wird nicht mehr gelesen — "+
			"bitte %s entfernen und ausschließlich %s setzen. Achtung: Mit dem falschen Schlüssel sind "+
			"Schülerfotos und das gespeicherte SMTP-Passwort nicht mehr entschlüsselbar.",
			crypto.AltName, crypto.SchluesselVariable, crypto.AltName, crypto.SchluesselVariable)
	}

	// Sicherheit: Die im Repo committeten Default-Secrets dürfen im echten Produktionsbetrieb
	// NICHT verwendet werden. Sonst könnte jeder mit Repo-Zugriff Admin-JWTs fälschen (JWT_SECRET)
	// bzw. die AES-verschlüsselten Schülerfotos entschlüsseln (APP_ENCRYPTION_KEY).
	//
	// Diese harte Start-Verweigerung ist bewusst per dediziertem Schalter EINSCHALTBAR und von
	// APP_ENV entkoppelt (APP_ENV steuert weiterhin Cookie-Secure & Swagger-Sichtbarkeit). Während
	// der Test-/Pilotphase bleibt sie aus (ENFORCE_PROD_SECRETS ungesetzt/false); vor dem echten
	// Prod-Deploy einfach ENFORCE_PROD_SECRETS=true setzen — dann verweigert der Server den Start
	// mit den bekannten Default-Werten.
	enforceProdSecrets := strings.ToLower(os.Getenv("ENFORCE_PROD_SECRETS")) == "true"
	if enforceProdSecrets {
		// Die Liste der Beispiel-Geheimnisse steht seit dem 11.08.2026 in
		// api.IstBekanntesDefaultGeheimnis und wird von der Selbstpruefung
		// (api/betriebsbereitschaft.go) mitbenutzt. Zwei Listen waeren genau die Fehlerart,
		// gegen die die Selbstpruefung antritt: Sie meldete "alles gut", waehrend der Server
		// aus demselben Grund den Start verweigert.
		if api.IstBekanntesDefaultGeheimnis(jwtSecret) {
			log.Fatalf("FATAL: JWT_SECRET nutzt einen bekannten Default-Wert. Setze ein eigenes, geheimes JWT_SECRET (≥32 Zeichen) — oder ENFORCE_PROD_SECRETS=false während der Testphase.")
		}
		if api.IstBekanntesDefaultGeheimnis(aesKey) {
			log.Fatalf("FATAL: APP_ENCRYPTION_KEY nutzt einen bekannten Default-Wert. Setze einen eigenen 32-Byte-Schlüssel — oder ENFORCE_PROD_SECRETS=false während der Testphase.")
		}
	}

	// Anmeldekonfiguration früh prüfen: Ein fehlender oder auf "mock" stehender
	// IMAP_HOST ist kein Detail, das beim ersten Login auffallen darf — im einen Fall
	// kann sich niemand anmelden, im anderen jeder.
	if err := auth.PruefeIMAPKonfiguration(); err != nil {
		log.Fatalf("FATAL: %v", err)
	}

	// Ausdrücklich protokolliert, weil eine vergessene Einstellung sonst genauso aussieht
	// wie eine funktionierende: Die Lehrkraft bekommt in beiden Fällen nur eine
	// abgelehnte Anmeldung, und niemand käme auf die Idee, in der .env nachzusehen.
	slog.Info(auth.SelbstanmeldungStatus())

	port = os.Getenv("PORT")
	if port == "" {
		log.Fatalf("FATAL: PORT environment variable is required and cannot be empty")
	}

	cookieSecure = ermittleCookieSecure()

	// Trusted-Proxy-Konfiguration für die Client-IP-Ermittlung (Rate-Limiting,
	// Login-Brute-Force, Audit-Logs). Ohne TRUSTED_PROXIES wird nur Loopback
	// vertraut — hinter Caddy im Docker-Netz muss das Netz dort gesetzt werden,
	// sonst kollabieren alle Clients auf die Proxy-IP.
	clientip.ConfigureFromEnv()

	return
}

func main() {
	// 0. Setup strukturiertes JSON-Logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	// slog.SetDefault haengt seit Go 1.21 AUCH das Standard-log-Paket hier ein. Alle 171
	// log.Printf-Aufrufe im Haus laufen damit durch diesen JSONHandler.
	//
	// Das ist zugleich der Schutz gegen eingeschleuste Protokollzeilen (CodeQL meldet
	// dafuer 25-mal go/log-injection): Ein Zeilenumbruch in einem Wert aus einer Anfrage
	// wird beim Serialisieren zu \n INNERHALB des JSON-Strings; er kann keinen zweiten
	// Eintrag erzeugen. Am laufenden Server nachgemessen (11.08.2026) mit
	//   GET /api/%0ASYSTEM:%20Alle%20Sperren%20aufgehoben
	// — r.URL.Path ist bereits dekodiert, der Umbruch kommt also wirklich an, und die
	// Ausgabe blieb ein einziger JSON-Eintrag.
	//
	// Wer diese Zeile entfernt oder log.SetOutput dahinter setzt, hebt beides auf: die
	// strukturierte Ausgabe UND den Schutz. Genau das hatte ich am 11.08. versucht, bevor
	// die Messung es zeigte.
	slog.SetDefault(logger)

	// Initialize Sentry
	sentryDsn := os.Getenv("SENTRY_DSN")
	if sentryDsn != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn: sentryDsn,
		})
		if err != nil {
			slog.Error("sentry.Init failed", "error", err)
		} else {
			defer sentry.Flush(2 * time.Second)
			slog.Info("Sentry initialized successfully.")
		}
	}

	// vorlage.Init() ABSICHTLICH nicht aufgerufen.
	//
	// Das ist das BEISPIEL-Plugin aus plugins/vorlage. Registriert lief es in
	// Produktion mit und schrieb bei jeder Rückgabe am Tresen zwei Zeilen ins Log —
	// eine aus DispatchEvent ("Dispatching event … to 1 listener(s)"), eine aus dem
	// Beispiel selbst, mit Titel, Barcode und Bearbeiter-UUID. Kein Schülername, also
	// kein Datenschutzproblem, aber Demo-Code im Betrieb und Log-Volumen an der
	// meistbenutzten Stelle des Systems. Ohne Zuhörer kehrt DispatchEvent still zurück.
	//
	// Der Erweiterungspunkt selbst BLEIBT: Die fünf DispatchEvent-Aufrufe stehen
	// jeweils nach dem Commit, neben dem Audit-Eintrag ("nur melden, was dauerhaft
	// passiert ist"). Ihn auszubauen ist Kategorie C und laut eigener Regel nichts,
	// was man mitten im Pilotbetrieb an Rückgabe-, Ausleih- und Gerätepfaden tut.
	// Nach dem Pilot zu entscheiden — Stand heute spricht alles fürs Löschen: ein
	// einziger Ereignistyp, kein echtes Plugin in 15 Monaten, Frontend-Teil nie
	// angeschlossen (docs/befunde.md).

	// 1. Config environment resolution
	dsn, jwtSecret, port, cookieSecure := loadConfig()

	// Capture interrupt and termination signals for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. Database Connection pool setup
	database := setupDatabase(ctx, dsn)
	defer database.Close()

	// 3. Authenticator initialization (12 hours token expiration duration)
	authenticator, err := auth.NewAuthenticator(jwtSecret, database.Pool, 12*time.Hour)
	if err != nil {
		slog.Error("Failed to initialize authenticator", "error", err)
		os.Exit(1)
	}

	// 4. Server-Sent Events broker initialization
	broker := sse.NewBroker()
	go broker.Start(ctx)
	slog.Info("Server-Sent Events (SSE) broker started.")

	// 5. Background Jobs & Scheduler
	auditRepo := repository.NewAuditRepository(database.Pool)
	scheduler := jobs.NewScheduler(database.Pool, auditRepo)
	scheduler.Start()
	defer scheduler.Stop()

	// Native async background worker for GDPR cleanup (runs on startup + every 24h)
	startGDPRWorker(ctx, scheduler)

	// 6. Initialize API Server and routing
	server := api.NewServer(database, authenticator, broker, cookieSecure)
	startBereitschaftsWaechter(ctx, server)
	httpServer := startServer(port, server)

	// 7. Autostart: Resume downloading missing covers
	go service.NewCoverService(database.Pool).SyncMissingCoversAsync()

	// Block until signal is received
	<-ctx.Done()
	slog.Info("Shutdown signal received. Commencing graceful stop...")

	// Timeout context for pending connections to finish
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("Graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	// Die Aufraeumschleife der Token-Sperrliste anhalten (Ticker, alle 15 Minuten).
	//
	// Stop() war bis zum 11.08.2026 geschrieben, getestet — und von niemandem aufgerufen;
	// die Suche nach unerreichbarem Code hat es gefunden. Beim Beenden des Prozesses waere
	// das folgenlos geblieben, die Goroutine stirbt ohnehin mit. Trotzdem gehoert es hier
	// hin: Ein geordnetes Herunterfahren, das die Haelfte stehen laesst, ist keins, und die
	// naechste Hintergrundaufgabe wuerde sich an dieser Stelle orientieren.
	authenticator.Blacklist.Stop()

	slog.Info("Server stopped successfully.")
}
