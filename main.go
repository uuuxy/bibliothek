// Copyright (c) 2026 Peter Flasch. All rights reserved.
// This source code is proprietary and confidential.

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
	"bibliothek/internal/service"
	"bibliothek/jobs"
	"bibliothek/plugins/vorlage"
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
		Addr:              ":" + port,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
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

	return database
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

	aesKey := os.Getenv("APP_ENCRYPTION_KEY")
	if len(aesKey) != 32 && len(aesKey) != 64 {
		log.Fatalf("FATAL: APP_ENCRYPTION_KEY must be exactly 32 bytes (or 64 hex characters) long")
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
		knownDefaultSecrets := map[string]bool{
			"super-secret-default-key-at-least-32-bytes": true, // Default aus docker-compose.yml (JWT)
			"super-secure-aes-key-32-chars-ok":           true, // Default aus docker-compose.yml (AES)
			"supergeheim_lokal":                          true,
		}
		if knownDefaultSecrets[jwtSecret] {
			log.Fatalf("FATAL: JWT_SECRET nutzt einen bekannten Default-Wert. Setze ein eigenes, geheimes JWT_SECRET (≥32 Zeichen) — oder ENFORCE_PROD_SECRETS=false während der Testphase.")
		}
		if knownDefaultSecrets[aesKey] {
			log.Fatalf("FATAL: APP_ENCRYPTION_KEY nutzt einen bekannten Default-Wert. Setze einen eigenen 32-Byte-Schlüssel — oder ENFORCE_PROD_SECRETS=false während der Testphase.")
		}
	}

	port = os.Getenv("PORT")
	if port == "" {
		log.Fatalf("FATAL: PORT environment variable is required and cannot be empty")
	}

	var err error
	cookieSecure, err = strconv.ParseBool(os.Getenv("COOKIE_SECURE"))
	if err != nil {
		cookieSecure = false
	}
	return
}

func main() {
	// 0. Setup strukturiertes JSON-Logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
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

	// Initialize optional plugins
	vorlage.Init()

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
	slog.Info("Server stopped successfully.")
}
