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

// Command bibliothek ist der Server der Schulbibliothek. Es liest und prüft die Konfiguration,
// öffnet die Datenbank, fährt die Migrationen, startet die zeitgesteuerten Läufe und bedient die
// Anfragen, bis es geordnet herunterfährt.
package main

import (
	"context"
	"io"
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
	"bibliothek/internal/bereitschaft"
	"bibliothek/internal/crypto"
	"bibliothek/internal/service"
	"bibliothek/jobs"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/logger"
	"bibliothek/repository"
	"bibliothek/sse"

	"github.com/getsentry/sentry-go"

	// Die Bildformate, die der Server liest: Cover und Ausweisfotos. WebP dekodiert libwebp aus
	// chai2010/webp, mit dem pkg/imageutil die Cover auch schreibt.
	_ "github.com/chai2010/webp"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// starteServer startet den HTTP-Server im Hintergrund und liefert ihn für das Herunterfahren.
func starteServer(port string, server *api.Server) *http.Server {
	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: server.Routes(),

		// ReadHeaderTimeout begrenzt das Senden der Kopfzeilen, nicht das des Rumpfes: Wer korrekte
		// Kopfzeilen schickt, "Content-Length: 10000000" ankündigt und danach ein Byte pro Minute
		// nachreicht, hielte Verbindung, Goroutine und womöglich eine Verbindung der Datenbank beliebig
		// lange fest. Die fristMiddleware hilft dagegen nicht, ein Ablauf des Kontexts bricht kein
		// blockierendes Read auf der Verbindung ab. Deshalb dazu ReadTimeout.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       api.LesefristVorgabe,

		// IdleTimeout gilt zwischen zwei Anfragen derselben Verbindung mit Keep-Alive. Ohne den Wert
		// übernimmt Go hier ReadTimeout; das wirkt, kettet aber die beiden Fristen aneinander, getrennt
		// gesetzt bleibt jede änderbar.
		IdleTimeout: 120 * time.Second,

		// Absichtlich kein WriteTimeout: Es gilt für die ganze Antwort, und der Strom der Server-Sent
		// Events (/events) ist eine Antwort, die nie endet. Ein Wert für alle kappte jede laufende
		// Verbindung nach Ablauf; die Oberfläche verlöre reihum ihre Aktualisierung, ohne dass ein Fehler
		// sichtbar würde. Die Richtung des Schreibens begrenzen stattdessen Caddy (write_timeout 600s)
		// und für die Dauer der Bearbeitung die fristMiddleware (api.anfrageFrist).
	}

	// Der Server läuft im Hintergrund; main wartet auf das Signal zum Beenden.
	go func() {
		slog.Info("Server listening", "url", "http://localhost:"+port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "fehler", err)
			os.Exit(1)
		}
	}()

	return httpServer
}

// starteBereitschaftsWaechter meldet kritische Befunde der Selbstprüfung täglich per Mail an die
// Admins, ein Wächter, der sich meldet, statt gelesen werden zu müssen
// (api/system/betriebsbereitschaft_alarm.go; auf Spielwiesen schweigt er von selbst). Der erste
// Lauf kommt drei Minuten nach dem Start: Ein Deploy mit kaputter Umgebung meldet sich so noch am
// selben Vormittag, nicht erst am nächsten Tag.
func starteBereitschaftsWaechter(ctx context.Context, server *api.Server) {
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

// starteDSGVOLauf fährt die Anonymisierung der Bearbeiter und die Löschung der Abgänger beim
// Start und danach alle 24 Stunden, neben den Läufen des Zeitplans.
func starteDSGVOLauf(ctx context.Context, zeitplan *jobs.Zeitplan) {
	go func() {
		slog.Info("Background Worker: Running initial GDPR cleanup on startup...")
		zeitplan.FuehreBearbeiterAnonymisierungAus(ctx)
		zeitplan.FuehreAbgaengerLoeschungAus(ctx)

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				slog.Info("Background Worker: Running scheduled 24h GDPR cleanup...")
				zeitplan.FuehreBearbeiterAnonymisierungAus(ctx)
				zeitplan.FuehreAbgaengerLoeschungAus(ctx)
			case <-ctx.Done():
				slog.Info("Background Worker: GDPR worker gracefully stopped.")
				return
			}
		}
	}()
}

// richteDatenbankEin verbindet die Datenbank, fährt die Migrationen, schreibt die Vorgabe der
// Rechte, legt bei leerer Datenbank den ersten Admin an und übernimmt die Mail-Konfiguration.
func richteDatenbankEin(ctx context.Context, dsn string) *db.Verbindung {
	slog.Info("Establishing database connection pool...")
	verbindung, err := db.Verbinde(ctx, dsn)
	if err != nil {
		slog.Error("Database connection pool failed", "fehler", err)
		os.Exit(1)
	}
	slog.Info("Database connection pool successfully initialized.")

	slog.Info("Running database migrations...")
	if err := verbindung.FuehreMigrationenAus(ctx, "migrations"); err != nil {
		slog.Error("Database migration failed", "fehler", err)
		os.Exit(1)
	}

	slog.Info("Initializing role permissions...")
	if err := verbindung.SchreibeRechteVorgabe(ctx); err != nil {
		slog.Error("Database permission initialization failed", "fehler", err)
		os.Exit(1)
	}

	slog.Info("Bootstrapping initial admin (if database is empty)...")
	if err := verbindung.LegeErstenAdminAn(ctx); err != nil {
		slog.Error("Admin bootstrapping failed", "fehler", err)
		os.Exit(1)
	}

	// Muss vor dem ersten Versand laufen: Übernimmt die Zugangsdaten für SMTP aus der Umgebung in die
	// Datenbank, solange dort die Vorgabe des Schemas steht. Danach gilt die Konfiguration aus der
	// Oberfläche; ohne diese Übernahme gingen die Mahnungen nach dem Umstieg an localhost:1025. Die
	// Meldung kommt nur, wenn wirklich übernommen wurde, sonst stünde sie bei jedem Start da.
	slog.Info("Prüfe gespeicherte SMTP-Konfiguration...")
	if err := verbindung.UebernimmMailKonfig(ctx); err != nil {
		slog.Error("Übernahme der Mail-Konfiguration fehlgeschlagen", "fehler", err)
		os.Exit(1)
	}

	return verbindung
}

// ermittleCookieSecure entscheidet über das Merkmal Secure der Cookies für Sitzung und CSRF.
//
// Ohne Variable gilt außerhalb der lokalen Entwicklung `true`: Bei einer stillen Vorgabe `false`
// reichte ein vergessenes COOKIE_SECURE im Deploy, damit Sitzungscookies ohne Secure ausgeliefert
// werden, und auffallen würde das niemandem, weil über HTTPS trotzdem alles funktioniert. Wer es
// unsicher will, muss es hinschreiben.
//
// Kein Abbruch bei ausdrücklichem false: Ein Server in der Schule läuft womöglich über schlichtes
// HTTP im LAN, und dort machte ein Secure-Cookie die Anmeldung unmöglich (der Browser sendet es
// über http nicht mit). Diese Entscheidung darf der Betrieb treffen; sie soll nur nicht
// versehentlich geschehen, deshalb die deutliche Warnung.
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
		// Ein unlesbarer Wert ist immer ein Fehler und nie eine Absicht; ihn als "false" zu deuten, wäre
		// eine stille Fehlannahme.
		slog.Error("COOKIE_SECURE ist kein Wahrheitswert (erlaubt: true/false)", "wert", roh)
		os.Exit(1)
	}

	if !wert && !lokal {
		slog.Warn("COOKIE_SECURE=false außerhalb der lokalen Entwicklung — Sitzungscookies werden OHNE Secure-Flag ausgeliefert. Nur zulässig, wenn die Anwendung ausschließlich über HTTP im lokalen Netz erreichbar ist.",
			"app_env", appEnv)
	}
	return wert
}

// leseKonfig liest die Konfiguration aus der Umgebung und bricht beim Start ab, wenn sie
// unbrauchbar oder gefährlich ist.
func leseKonfig() (dsn, jwtGeheimnis, port string, server api.ServerKonfig) {
	dsn = os.Getenv("DATABASE_URL")
	if dsn == "" {
		slog.Error("DATABASE_URL environment variable is required and cannot be empty")
		os.Exit(1)
	}

	jwtGeheimnis = os.Getenv("JWT_SECRET")
	if len(jwtGeheimnis) < 32 {
		slog.Error("JWT_SECRET environment variable must be at least 32 characters long for security")
		os.Exit(1)
	}

	schluessel := os.Getenv(crypto.SchluesselVariable)
	if len(schluessel) != 32 && len(schluessel) != 64 {
		slog.Error("encryption key must be exactly 32 bytes (or 64 hex characters) long", "variable", crypto.SchluesselVariable)
		os.Exit(1)
	}

	// Der frühere Zweitname darf nicht unbemerkt danebenstehen. Er wird nicht mehr gelesen; wer ihn
	// gesetzt hat, erwartet aber, dass er gilt. Also lieber hier laut abbrechen als mit einem
	// Schlüssel verschlüsseln, mit dem der Betrieb nicht rechnet.
	if alt := os.Getenv(crypto.FruehereVariable); alt != "" && alt != schluessel {
		slog.Error("Der frühere Name des Schlüssels ist gesetzt und weicht ab. Er wird nicht mehr gelesen — "+
			"bitte entfernen und ausschließlich den neuen Namen setzen. Achtung: Mit dem falschen Schlüssel sind "+
			"Schülerfotos und das gespeicherte SMTP-Passwort nicht mehr entschlüsselbar.",
			"frueher", crypto.FruehereVariable, "variable", crypto.SchluesselVariable)
		os.Exit(1)
	}

	// Die Beispiel-Geheimnisse aus dem Repository dürfen im echten Betrieb nicht verwendet werden.
	// Sonst könnte jeder mit Zugang zum Repository Tokens eines Admins fälschen (JWT_SECRET) oder die
	// verschlüsselten Schülerfotos entschlüsseln (APP_ENCRYPTION_KEY).
	//
	// Außerhalb von local/development/test verweigert der Server die Beispielwerte von selbst. Nur
	// ein ausdrückliches ENFORCE_PROD_SECRETS=false schaltet das für eine Testphase ab
	// (bereitschaft.IstGeheimnisSchutzScharf, geteilt mit der Selbstprüfung).
	geheimnisSchutz := bereitschaft.IstGeheimnisSchutzScharf(os.Getenv("APP_ENV"), os.Getenv("ENFORCE_PROD_SECRETS"))
	if !geheimnisSchutz && strings.EqualFold(strings.TrimSpace(os.Getenv("ENFORCE_PROD_SECRETS")), "false") {
		slog.Warn("ENFORCE_PROD_SECRETS=false — der Server startet auch mit Beispiel-Geheimnissen aus dem Repository. Nur für die Testphase zulässig.",
			"app_env", os.Getenv("APP_ENV"))
	}
	if geheimnisSchutz {
		// Die Liste der Beispiel-Geheimnisse steht in bereitschaft.IstBeispielGeheimnis und gilt auch für
		// die Selbstprüfung (internal/bereitschaft/geheimnisse.go). Zwei Listen wären genau die
		// Fehlerart, gegen die die Selbstprüfung antritt: Sie meldete „alles gut", während der Server aus
		// demselben Grund den Start verweigert.
		if bereitschaft.IstBeispielGeheimnis(jwtGeheimnis) {
			slog.Error("JWT_SECRET nutzt einen bekannten Default-Wert. Setze ein eigenes, geheimes JWT_SECRET (≥32 Zeichen) — oder ENFORCE_PROD_SECRETS=false während der Testphase.")
			os.Exit(1)
		}
		if bereitschaft.IstBeispielGeheimnis(schluessel) {
			slog.Error("APP_ENCRYPTION_KEY nutzt einen bekannten Default-Wert. Setze einen eigenen 32-Byte-Schlüssel — oder ENFORCE_PROD_SECRETS=false während der Testphase.")
			os.Exit(1)
		}
	}

	// Die Konfiguration der Anmeldung früh prüfen: Ein fehlender oder auf "mock" stehender IMAP_HOST
	// ist kein Detail, das bei der ersten Anmeldung auffallen darf; im einen Fall kann sich niemand
	// anmelden, im anderen jeder.
	if err := auth.PruefeIMAPKonfiguration(); err != nil {
		slog.Error("IMAP-Konfiguration ungültig", "fehler", err)
		os.Exit(1)
	}

	// Ausdrücklich protokolliert, weil eine vergessene Einstellung sonst genauso aussieht
	// wie eine funktionierende: Die Lehrkraft bekommt in beiden Fällen nur eine
	// abgelehnte Anmeldung, und niemand käme auf die Idee, in der .env nachzusehen.
	slog.Info("Selbstanmeldung", "stand", auth.SelbstanmeldungStatus())

	port = os.Getenv("PORT")
	if port == "" {
		slog.Error("PORT environment variable is required and cannot be empty")
		os.Exit(1)
	}

	server.NurHTTPS = ermittleCookieSecure()

	server.VertrauteProxys = vertrauteProxysAusUmgebung()

	return dsn, jwtGeheimnis, port, server
}

// vertrauteProxysAusUmgebung liefert die vertrauten Proxys für die Adresse des Browsers
// (Bremsen, Anmeldung, Protokoll) aus TRUSTED_PROXIES. Ohne den Wert wird nur Loopback vertraut;
// hinter Caddy im Docker-Netz muss das Netz dort stehen, sonst fallen alle Browser auf die
// Adresse des Proxys zusammen. Ein unlesbarer Eintrag verhindert den Start nicht; er steht als
// Warnung im Log und auf der Seite „Betriebsbereitschaft", weil er sonst niemandem auffällt.
func vertrauteProxysAusUmgebung() []string {
	eintraege := strings.Split(os.Getenv("TRUSTED_PROXIES"), ",")
	if unlesbar := clientip.UnlesbareEintraege(eintraege); len(unlesbar) > 0 {
		slog.Warn("TRUSTED_PROXIES: Einträge übergangen, die weder Netz noch Adresse sind", "eintraege", unlesbar)
	}
	return eintraege
}

// baueLog baut das Log des Servers: JSON ab der Stufe INFO, davor der Fehlerstand, der die Zeilen
// der Stufe ERROR für die Selbstprüfung zählt. Er steht vor allem anderen, damit er auch die
// Zeilen des Starts zählt.
func baueLog(ausgabe io.Writer) (*slog.Logger, *logger.Fehlerstand) {
	fehlerstand := logger.NewFehlerstand()
	return slog.New(fehlerstand.SchalteVor(slog.NewJSONHandler(ausgabe, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))), fehlerstand
}

func main() {
	// 0. Strukturiertes Log als JSON, mit dem Fehlerstand davor.
	logAusgabe, fehlerstand := baueLog(os.Stdout)
	// slog.SetDefault hängt seit Go 1.21 auch das Standardpaket log hier ein: Bibliotheken, die über
	// log schreiben, landen im selben JSON, mit der Stufe INFO. Der eigene Code schreibt über slog
	// mit Stufe (docs/STIL.md, Abschnitt 11).
	//
	// Das ist zugleich der Schutz gegen eingeschleuste Zeilen im Log (CodeQL meldet dafür
	// go/log-injection): Ein Zeilenumbruch in einem Wert aus einer Anfrage wird beim Serialisieren zu
	// \n innerhalb der Zeichenkette in JSON; er kann keinen zweiten Eintrag erzeugen. Am laufenden
	// Server nachgemessen mit
	//   GET /api/%0ASYSTEM:%20Alle%20Sperren%20aufgehoben
	// r.URL.Path ist schon dekodiert, der Umbruch kommt also wirklich an, und die Ausgabe blieb ein
	// einziger Eintrag in JSON.
	//
	// Wer diese Zeile entfernt oder log.SetOutput dahinter setzt, hebt beides auf: die strukturierte
	// Ausgabe und den Schutz.
	slog.SetDefault(logAusgabe)

	// Sentry, falls eingestellt.
	sentryDSN := os.Getenv("SENTRY_DSN")
	if sentryDSN != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn: sentryDSN,
		})
		if err != nil {
			slog.Error("sentry.Init failed", "fehler", err)
		} else {
			defer sentry.Flush(2 * time.Second)
			slog.Info("Sentry initialized successfully.")
		}
	}

	// 1. Konfiguration aus der Umgebung.
	dsn, jwtGeheimnis, port, serverKonfig := leseKonfig()

	// Auf Interrupt und SIGTERM geordnet herunterfahren.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. Datenbank.
	verbindung := richteDatenbankEin(ctx, dsn)
	defer verbindung.Close()

	// Den Fehlerstand des vorigen Laufs übernehmen und den eigenen fortlaufend ablegen.
	go jobs.FuehreFehlerstandAblageAus(ctx, verbindung.Pool, fehlerstand)

	// 3. Anmeldung, Tokens zwölf Stunden gültig.
	anmeldung, err := auth.NewAnmeldung(jwtGeheimnis, verbindung.Pool, 12*time.Hour)
	if err != nil {
		slog.Error("Failed to initialize authenticator", "fehler", err)
		os.Exit(1)
	}
	// Abgelaufene Widerrufe und Sitzungen abräumen, bis der Server herunterfährt.
	go anmeldung.RaeumeAuf(ctx)

	// 4. Verteiler für Server-Sent Events.
	broker := sse.NewBroker()
	go broker.FuehreAus(ctx)
	slog.Info("Server-Sent Events (SSE) broker started.")

	// 5. Zeitplan der Läufe. Den Abgleich der Cover teilen Zeitplan, Start und der Knopf in der
	// Verwaltung; so läuft nie mehr als einer.
	protokollAbfragen := repository.NewProtokollAbfragen(verbindung.Pool)
	coverAbgleich := service.NewCoverDienst(verbindung.Pool)
	zeitplan := jobs.NewZeitplan(verbindung.Pool, protokollAbfragen)
	zeitplan.Starte(ctx, coverAbgleich)
	defer zeitplan.Stoppe()

	// Anonymisierung und Löschung nach DSGVO beim Start und alle 24 Stunden.
	starteDSGVOLauf(ctx, zeitplan)

	// 6. Server und Routen.
	serverKonfig.CoverAbgleich = coverAbgleich
	serverKonfig.Fehlerstand = fehlerstand
	server := api.NewServer(verbindung, anmeldung, broker, serverKonfig)
	starteBereitschaftsWaechter(ctx, server)
	httpServer := starteServer(port, server)

	// 7. Fehlende Cover im Hintergrund nachladen.
	go coverAbgleich.GleicheFehlendeCoverAb(ctx)

	// Warten auf das Signal zum Beenden.
	<-ctx.Done()
	slog.Info("Shutdown signal received. Commencing graceful stop...")

	// Frist, in der offene Verbindungen enden dürfen.
	herunterfahrKontext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Der Vermerk „geordnet beendet" geht hinaus, bevor der Server auf offene Anfragen wartet:
	// Docker bricht nach derselben Frist ab, und ein Update ist kein Absturz.
	fehlerstand.Beende()
	jobs.SchreibeFehlerstand(herunterfahrKontext, verbindung.Pool, fehlerstand)

	if err := httpServer.Shutdown(herunterfahrKontext); err != nil {
		slog.Error("Graceful shutdown failed", "fehler", err)
		// Die Frist ist um; die Zeile darüber soll trotzdem im abgelegten Stand stehen.
		letzteAblage, abbrechen := context.WithTimeout(context.Background(), 3*time.Second)
		jobs.SchreibeFehlerstand(letzteAblage, verbindung.Pool, fehlerstand)
		abbrechen()
		os.Exit(1)
	}
	// Der letzte Stand, bevor das Update den nächsten Lauf startet.
	jobs.SchreibeFehlerstand(herunterfahrKontext, verbindung.Pool, fehlerstand)

	slog.Info("Server stopped successfully.")
}
