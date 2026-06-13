package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"bibliothek/api"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/jobs"
	"bibliothek/plugins/vorlage"
	"bibliothek/repository"
	"bibliothek/sse"
)

// @title           Schulbibliothek API
// @version         1.0
// @description     Backend-API fuer das Schulbibliothek-Verwaltungssystem.
// @host            localhost:8080
// @BasePath        /api

// main is the entry point of the school library system backend application.
// It initializes configs, setups database connection pools, starts the SSE broker,
// mounts middleware-protected routes, and starts the server with graceful shutdown.
func main() {
	// 0. Setup strukturiertes JSON-Logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Initialize optional plugins
	vorlage.Init()

	// 1. Config environment resolution
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		slog.Error("FATAL: DATABASE_URL environment variable is required and cannot be empty")
		os.Exit(1)
	}

	// Zero Hardcoded Secrets: Fail hard if JWT_SECRET is not set
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		slog.Error("FATAL: JWT_SECRET environment variable is required and cannot be empty")
		os.Exit(1)
	}

	port := os.Getenv("PORT")
	if port == "" {
		slog.Error("FATAL: PORT environment variable is required and cannot be empty")
		os.Exit(1)
	}

	cookieSecure, err := strconv.ParseBool(os.Getenv("COOKIE_SECURE"))
	if err != nil {
		cookieSecure = false
	}

	// Capture interrupt and termination signals for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. Database Connection pool setup
	slog.Info("Establishing database connection pool...")
	database, err := db.Connect(ctx, dsn)
	if err != nil {
		slog.Error("Database connection pool failed", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	slog.Info("Database connection pool successfully initialized.")

	// 2a. Run pending SQL migrations
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

	// 6. Initialize API Server and routing
	server := api.NewServer(database, authenticator, broker, cookieSecure)
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
