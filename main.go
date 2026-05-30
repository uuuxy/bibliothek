package main

import (
	"context"
	"log"
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
	// Initialize optional plugins
	vorlage.Init()

	// 1. Config environment resolution
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("FATAL: DATABASE_URL environment variable is required and cannot be empty")
	}
	
	// Zero Hardcoded Secrets: Fail hard if JWT_SECRET is not set
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: JWT_SECRET environment variable is required and cannot be empty")
	}
	
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("FATAL: PORT environment variable is required and cannot be empty")
	}
	
	cookieSecure, err := strconv.ParseBool(os.Getenv("COOKIE_SECURE"))
	if err != nil {
		cookieSecure = false
	}

	// Capture interrupt and termination signals for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. Database Connection pool setup
	log.Println("Establishing database connection pool...")
	database, err := db.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("Database connection pool failed: %v", err)
	}
	defer database.Close()
	log.Println("Database connection pool successfully initialized.")

	log.Println("Initializing role permissions...")
	if err := database.InitPermissions(ctx); err != nil {
		log.Fatalf("Database permission initialization failed: %v", err)
	}

	log.Println("Initializing suppliers...")
	if err := database.InitLieferanten(ctx); err != nil {
		log.Fatalf("Database supplier initialization failed: %v", err)
	}

	// 3. Authenticator initialization (12 hours token expiration duration)
	authenticator, err := auth.NewAuthenticator(jwtSecret, 12*time.Hour)
	if err != nil {
		log.Fatalf("Failed to initialize authenticator: %v", err)
	}

	// 4. Server-Sent Events broker initialization
	broker := sse.NewBroker()
	go broker.Start(ctx)
	log.Println("Server-Sent Events (SSE) broker started.")

	// 5. GDPR Cron Scheduler initialization
	auditRepo := repository.NewAuditRepository(database.Pool)
	scheduler := jobs.NewScheduler(database.Pool, auditRepo)
	scheduler.Start()
	defer scheduler.Stop()

	// 6. Initialize API Server and routing
	server := api.NewServer(database, authenticator, broker, cookieSecure)
	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: server.Routes(),
	}

	// Start server asynchronously
	go func() {
		log.Printf("Server listening on http://localhost:%s/", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Block until signal is received
	<-ctx.Done()
	log.Println("Shutdown signal received. Commencing graceful stop...")

	// Timeout context for pending connections to finish
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Graceful shutdown failed: %v", err)
	}
	log.Println("Server stopped successfully.")
}

