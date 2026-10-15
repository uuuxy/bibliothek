package api

// middleware.go — HTTP middleware chain for the library server.
// All middleware is side-effect free and does not depend on application state
// beyond the Server struct. Middleware is registered in router.go.

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pkg/kennung"
)

// langLaufendeFrist gilt für Vorgänge, die naturgemäß Minuten dauern: Katalog- und
// Schülerimporte berühren zehntausende Zeilen in einer Transaktion.
//
// 5 Minuten, abgestimmt auf das Upload-Limit des Frontends (apiFetch.js) und auf Caddys
// response_header_timeout. Alle drei müssen dieselbe Frist meinen — läuft die kürzeste
// vorne weg, bricht der Vorgang dort ab, wo niemand die Meldung sieht.
const langLaufendeFrist = 5 * time.Minute

// langLaufendePfade sind die Endpunkte, für die langLaufendeFrist gilt.
//
// Sie standen zuvor unter demselben 15-Sekunden-Limit wie ein Barcode-Scan. Der
// Littera-Import (11.076 Titel, 61.580 Exemplare im echten Altbestand) kann darunter
// nicht durchlaufen: Der Server brach nach 15 s am DB-Kontext ab, während der Browser
// noch 5 Minuten wartete. Der Abbruch traf also mitten in die Transaktion, ohne dass
// vorne jemand etwas davon sah.
var langLaufendePfade = []string{
	"/api/import/",          // Littera-Katalogimport
	"/api/admin/import-",    // Bestandsimport
	"/api/books/import",     // Listenimport: je unbekannter ISBN ein Katalog-Lookup, 10 parallel
	"/api/admin/sync-",      // Cover-Abgleich über den ganzen Bestand
	"/api/lusd/",            // LUSD-Vorschau und -Import (ganze Jahrgänge)
	"/api/admin/mahnungen/", // Sammeldruck über alle Klassen
}

// LesefristVorgabe ist die Frist, in der eine gewöhnliche Anfrage vollständig
// eingelesen sein muss (Kopf UND Rumpf). Sie steht am http.Server als ReadTimeout.
//
// 30 Sekunden sind für jede normale Anfrage dieser Anwendung reichlich — der größte
// gewöhnliche Rumpf ist ein Foto-Upload. Die Import-Endpunkte brauchen mehr und holen
// es sich in erweitereLesefristFuerLangeUploads.
const LesefristVorgabe = 30 * time.Second

// bearbeitungsfristVorgabe ist die Frist der fristMiddleware für einen gewöhnlichen Aufruf
// (Barcode-Scan, Suche, Formular); lang laufende Pfade bekommen langLaufendeFrist. Benannt seit
// dem 15.09.2026, weil die Idempotenz der Theke an ihr hängt: Eine Reservierung des Schlüssels
// gilt nach repository.IdempotenzReservierungsfrist als verwaist, und das stimmt nur, wenn jede
// Arbeit vorher abgebrochen und ihre Antwort gespeichert ist (idempotenz_fristen_test.go).
const bearbeitungsfristVorgabe = 15 * time.Second

// anfrageFrist bestimmt die Frist für einen Pfad. Ausgelagert, damit die Zuordnung
// ohne HTTP-Aufbau prüfbar ist — eine Ausnahmeliste, die man nur im Betrieb testen
// kann, ist genau die Sorte Schutz, die man irrtümlich für wirksam hält.
func anfrageFrist(pfad string, vorgabe time.Duration) time.Duration {
	for _, praefix := range langLaufendePfade {
		if strings.HasPrefix(pfad, praefix) {
			return langLaufendeFrist
		}
	}
	return vorgabe
}

// erweitereLesefristFuerLangeUploads hebt die Lesefrist der Verbindung für genau die
// Endpunkte an, die große Dateien entgegennehmen.
//
// Sie ist die Gegenseite von ReadTimeout am Server: Ohne diese Middleware müsste die
// serverweite Frist so großzügig sein wie der langsamste Import (100 MB Littera-Datei
// über eine Schulleitung) — und damit wäre sie für die 200 anderen Endpunkte wirkungslos.
// Umgekehrt bräche eine strenge serverweite Frist mitten in den Upload, den niemand
// abbrechen will. Also: streng als Vorgabe, großzügig für die fünf bekannten Pfade,
// dieselbe Liste wie bei der Bearbeitungsfrist (langLaufendePfade).
//
// Schlägt das Setzen der Frist fehl (ResponseWriter ohne SetReadDeadline, z. B. in
// manchen Testaufbauten), läuft die Anfrage weiter — die serverweite Frist gilt dann
// unverändert. Ein Upload, der daran scheitert, ist ein sichtbarer Fehler; ein
// abgewiesener Request wegen eines fehlenden Optionalinterfaces wäre ein Rätsel.
func erweitereLesefristFuerLangeUploads(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if anfrageFrist(r.URL.Path, LesefristVorgabe) != LesefristVorgabe {
			if err := http.NewResponseController(w).SetReadDeadline(time.Now().Add(langLaufendeFrist)); err != nil {
				log.Printf("Lesefrist für %s nicht verlängerbar, es gilt die Vorgabe: %v", r.URL.Path, err)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// istLiveStrom erkennt den SSE-Strom. Er bleibt offen und schickt jedes Ereignis einzeln;
// Bearbeitungsfrist und Kompression gelten für ihn nicht. Der genaue Pfad, nicht das
// Teilwort: Eine Datei der Oberfläche darf events-….js heißen.
func istLiveStrom(pfad string) bool {
	return pfad == "/events"
}

// fristMiddleware wraps the request with a context timeout.
//
// Der SSE-Stream bleibt ohne Frist — er ist definitionsgemäß offen. Alles andere bekommt
// eine, auch die langen Vorgänge: Ein Request ganz ohne Frist hängt im Zweifel für immer
// und hält eine Datenbankverbindung fest.
func fristMiddleware(frist time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if istLiveStrom(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), anfrageFrist(r.URL.Path, frist))
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// panikMiddleware catches panics during HTTP request handling, logs them, and returns a 500 error.
func panikMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// maskiereToken auch hier: Der Bestätigungs-Token steht im Pfad, und ein
				// Panic auf der Lieferantenseite schreibt genau diesen Pfad ins Log. Die
				// 500er-Zeile in logMiddleware maskierte bereits, diese nicht — ein
				// weitergereichtes Logfile hätte damit weiter funktionierende Links
				// enthalten und die Hash-Speicherung in der Datenbank entwertet.
				log.Printf("PANIC RECOVERED in request %s %s: %v", r.Method, maskiereToken(r.URL.Path), err)
				apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("interner server fehler: %v", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// httpsMiddleware automatically redirects unencrypted HTTP requests to HTTPS.
func (s *Server) httpsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Bypass HTTPS redirection in local/development mode when nurHTTPS is disabled
		if !s.nurHTTPS {
			next.ServeHTTP(w, r)
			return
		}

		// /health wird NIE umgeleitet.
		//
		// Die Sonde kommt aus dem Container selbst (Docker HEALTHCHECK, wget gegen
		// 127.0.0.1) und damit ohne TLS und ohne X-Forwarded-Proto. Mit Umleitung bekam
		// sie eine 301 auf https://127.0.0.1:PORT, versuchte dort einen TLS-Handshake
		// gegen den Klartext-Port und scheiterte mit "wrong version number" — in
		// Produktion (COOKIE_SECURE=true) also IMMER, während lokal (false) alles grün
		// aussah. Der Container galt seitdem dauerhaft als unhealthy: Alles, was daran
		// hängt (Neustart-Regeln, depends_on: service_healthy, das Deploy-Skript), traf
		// seine Entscheidungen auf einer Anzeige, die nie stimmte.
		//
		// Unbedenklich: Die Antwort ist {"status":"healthy"} — kein Geheimnis, keine
		// Sitzung, kein Cookie. Es gibt hier nichts, was eine Klartext-Verbindung
		// preisgeben könnte.
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		istHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
		if istHTTPS {
			next.ServeHTTP(w, r)
			return
		}

		erlaubterUrsprung := os.Getenv("ALLOWED_ORIGIN")
		var zielHost string

		if erlaubterUrsprung != "" {
			// Use trusted configuration to prevent Open Redirect / Host Header Injection
			zielHost = strings.TrimPrefix(erlaubterUrsprung, "https://")
			zielHost = strings.TrimPrefix(zielHost, "http://")
		} else {
			// Fallback to r.Host for multi-domain support if not strictly configured
			zielHost = r.Host
			if strings.ContainsAny(zielHost, "\r\n\t") {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
		}

		ziel := "https://" + zielHost + r.URL.RequestURI()
		http.Redirect(w, r, ziel, http.StatusMovedPermanently) // #nosec G710
	})
}

// rumpfGrenzeMiddleware limits the request body size to prevent DoS.
func rumpfGrenzeMiddleware(grenze int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, grenze)
			next.ServeHTTP(w, r)
		})
	}
}

// Hinweis: Die frühere RBACBlockMiddleware wurde entfernt. Sie erzwang eine hartkodierte
// Pfad-Allowlist für LEHRER/HELFER, die das konfigurierbare role_permissions-System überstimmte
// (ein LEHRER konnte seine im PermissionManager gewährten Rechte nicht nutzen). Autorisierung
// erfolgt jetzt einheitlich pro Route über verlangeRecht bzw. RequireRoles.

// corsMiddleware restricts cross-origin requests to the configured school domain.
// Set ALLOWED_ORIGIN env var to the school's frontend URL (e.g. https://bibliothek.schule.de).
// Falls back to same-origin only if not configured.
func corsMiddleware(next http.Handler) http.Handler {
	erlaubterUrsprung := os.Getenv("ALLOWED_ORIGIN")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ursprung := r.Header.Get("Origin")
		if ursprung == "" {
			// Same-origin request (no Origin header) – always allowed
			next.ServeHTTP(w, r)
			return
		}
		// Only allow explicitly configured origin; reject everything else
		if erlaubterUrsprung != "" && ursprung == erlaubterUrsprung {
			w.Header().Set("Access-Control-Allow-Origin", erlaubterUrsprung)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			// X-CSRF-Token gehört zwingend dazu: Ohne den Header in dieser Liste
			// scheitert bei gesetztem ALLOWED_ORIGIN jede mutierende Cross-Origin-Anfrage
			// schon am Preflight — die CSRF-Härtung hätte das Frontend ausgesperrt statt
			// Angreifer. Im Same-Origin-Betrieb hinter Caddy fällt das nicht auf, weil
			// es dort gar keinen Preflight gibt.
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// uuidPfadMiddleware intercepts requests and validates {id} path parameters
// against a standard UUID format before they hit the database.
// uuidPfadParameter nennt die Pfad-Parameter, deren Wert IMMER eine UUID ist —
// namentlich und abschließend, nicht nach einem Namensmuster.
//
// Ein Muster wie "alles was auf _id endet" wäre hier gefährlich: {barcode_id} endet
// genauso auf _id, trägt aber einen Barcode (VARCHAR(100), z. B. "S-abg1-ms3p73…").
// Genau diese Verwechslung hat das Foto-GET lahmgelegt — es hieß {id}, und die
// Prüfung wies damit jeden Barcode mit 400 ab, bevor der Handler lief. Kein einziges
// Schülerfoto wurde je ausgeliefert. Deshalb steht hier eine Liste, die man nur
// erweitert, wenn die dahinterliegende Spalte nachweislich UUID ist.
//
// Belegt in schema.sql: schueler.id (Zeile 151) und ausleihen.id (Zeile 458) sind
// UUID. {klasse} ist ein Klassenname und gehört bewusst NICHT dazu.
var uuidPfadParameter = []string{"id", "schueler_id", "ausleihe_id"}

// uuidPfadMiddleware weist Anfragen ab, deren UUID-Pfadparameter keine UUID
// sind — bevor sie die Datenbank erreichen. Geprüft wird nur, was in
// uuidPfadParameter steht; die Begründung für diese Liste steht dort.
func uuidPfadMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, name := range uuidPfadParameter {
			wert := r.PathValue(name)
			if wert != "" && !kennung.IstUUID(wert) {
				apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ungültiges UUID Format im Pfadparameter"))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// verlangeRecht returns a middleware that validates if the authenticated user
// has the required permission dynamically defined in the database.
func (s *Server) verlangeRecht(recht string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Die UUID-Prüfung sitzt hier und NICHT mehr global um den Mux herum.
		//
		// Global gelegt las sie r.PathValue("id"), bevor der ServeMux die Route
		// aufgelöst hatte — der Wert war dort immer leer, die Prüfung lief also nie
		// (Audit-Befund vom 01.08.2026, belegt in middleware_frist_test.go). An dieser
		// Stelle ist die Route bereits getroffen und PathValue gefüllt.
		return uuidPfadMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.PruefeRecht(w, r, next, recht)
		}))
	}
}

// statusMerker is a custom ResponseWriter that tracks the HTTP status code
type statusMerker struct {
	http.ResponseWriter
	status int
}

func (m *statusMerker) WriteHeader(status int) {
	m.status = status
	m.ResponseWriter.WriteHeader(status)
}

// Unwrap returns the underlying ResponseWriter so that http.NewResponseController
// can traverse the middleware chain (e.g. to call SetWriteDeadline for SSE).
func (m *statusMerker) Unwrap() http.ResponseWriter {
	return m.ResponseWriter
}

// Flush implements http.Flusher to support Server-Sent Events (SSE)
func (m *statusMerker) Flush() {
	if f, ok := m.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// logMiddleware records the HTTP status and prints the exact stack trace for 500 errors.
// tokenPfade sind die einzigen Pfade, die ein Geheimnis TRAGEN: Im Bestätigungs-Link an
// den Lieferanten steht der Token in der Adresse — sowohl beim Aufruf der Seite selbst
// als auch bei ihren API-Aufrufen. Die längere Variante steht zuerst, damit sie greift.
var tokenPfade = []string{"/api/public/bestellung/", "/bestellung/"}

// maskiereToken schneidet den Bestätigungs-Token aus einem Pfad, bevor er ins Log geht.
//
// Ein Logfile wird weitergereicht, in Tickets gehängt und selten so geschützt wie die
// Datenbank. Ein Token darin wäre ein funktionierender Link — genau das, was die
// Hash-Speicherung in der Datenbank verhindern soll.
func maskiereToken(pfad string) string {
	for _, praefix := range tokenPfade {
		if !strings.HasPrefix(pfad, praefix) {
			continue
		}
		rest := strings.TrimPrefix(pfad, praefix)
		if i := strings.Index(rest, "/"); i >= 0 {
			return praefix + "***" + rest[i:]
		}
		return praefix + "***"
	}
	return pfad
}

// logMiddleware schreibt eine Zeile, wenn eine Anfrage mit 5xx endet: Status, Methode,
// Pfad. Mehr nicht — keine Dauer, keine Anfragekennung, und bei 2xx bis 4xx gar nichts.
// Die Zeile je Anfrage („Incoming Request", Methode und Pfad, VOR der Verarbeitung) schreibt
// baueKette in router.go. Bis zum 21.09.2026 stand hier „jede Anfrage mit Status und Dauer";
// das hat diese Funktion nie getan.
//
// Der Pfad läuft an beiden Stellen durch die Maskierung darüber: Ein Geheimnis im Pfad
// (Bestätigungs-Token) darf nicht im Klartext im Log landen.
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		merker := &statusMerker{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(merker, r)

		// Bis zum 07.09.2026 hing hier debug.Stack() an — der Stack DIESER Middleware nach
		// next.ServeHTTP, nie der der Fehlerstelle: 40 Zeilen ohne Information bei jedem
		// 5xx. Den echten Stack eines Panics schreibt panikMiddleware; die Ursache
		// eines 5xx steht in der Zeile, die apierrors beim Senden loggt.
		if merker.status >= 500 {
			log.Printf("HTTP %d on %s %s", merker.status, r.Method, maskiereToken(r.URL.Path))
		}
	})
}
