package api

// Die Glieder der Middleware-Kette; baueKette in router.go legt sie um den Router.

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pkg/kennung"
)

// langLaufendeFrist gilt für Vorgänge, die naturgemäß Minuten dauern: Katalog- und
// Schülerimporte berühren zehntausende Zeilen in einer Transaktion.
//
// 5 Minuten, abgestimmt auf die Grenze für Uploads in der Oberfläche (apiFetch.js) und auf Caddys
// response_header_timeout. Alle drei müssen dieselbe Frist meinen — läuft die kürzeste
// vorne weg, bricht der Vorgang dort ab, wo niemand die Meldung sieht.
const langLaufendeFrist = 5 * time.Minute

// langLaufendePfade sind die Endpunkte, für die langLaufendeFrist gilt.
//
// Mit der Frist eines Barcode-Scans (15 Sekunden) liefe der Littera-Import (11.076 Titel,
// 61.580 Exemplare im echten Altbestand) nicht durch: Der Server bräche nach 15 s am Kontext
// der Datenbank ab, mitten in der Transaktion, während der Browser noch 5 Minuten wartet.
var langLaufendePfade = []string{
	"/api/import/",          // Littera-Katalogimport
	"/api/admin/import-",    // Bestandsimport
	"/api/books/import",     // Listenimport: je unbekannter ISBN eine Abfrage der Katalogdienste, 10 parallel
	"/api/admin/sync-",      // Cover-Abgleich über den ganzen Bestand
	"/api/lusd/",            // LUSD-Vorschau und -Import (ganze Jahrgänge)
	"/api/admin/mahnungen/", // Sammeldruck über alle Klassen
}

// LesefristVorgabe ist die Frist, in der eine gewöhnliche Anfrage vollständig
// eingelesen sein muss (Kopf und Rumpf). Sie steht am http.Server als ReadTimeout.
//
// 30 Sekunden sind für jede normale Anfrage dieser Anwendung reichlich — der größte
// gewöhnliche Rumpf ist ein Foto-Upload. Die Import-Endpunkte brauchen mehr und holen
// es sich in erweitereLesefristFuerLangeUploads.
const LesefristVorgabe = 30 * time.Second

// bearbeitungsfristVorgabe ist die Frist der fristMiddleware für einen gewöhnlichen Aufruf
// (Barcode-Scan, Suche, Formular); lang laufende Pfade bekommen langLaufendeFrist. Sie trägt
// einen Namen, weil die Idempotenz der Theke an ihr hängt: Eine Reservierung des Schlüssels
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
// serverweite Frist so großzügig sein wie der langsamste Import (100 MB Littera-Datei über
// die Leitung einer Schule), und damit wäre sie für alle anderen Endpunkte wirkungslos.
// Umgekehrt bräche eine strenge serverweite Frist mitten in den Upload, den niemand abbrechen
// will. Also: streng als Vorgabe, großzügig für die bekannten Pfade, dieselbe Liste wie bei
// der Bearbeitungsfrist (langLaufendePfade).
//
// Schlägt das Setzen der Frist fehl (ResponseWriter ohne SetReadDeadline, etwa in manchen
// Testaufbauten), läuft die Anfrage weiter, und die serverweite Frist gilt unverändert. Ein
// Upload, der daran scheitert, ist ein sichtbarer Fehler; eine Anfrage, abgewiesen wegen einer
// fehlenden freiwilligen Schnittstelle, wäre ein Rätsel.
func erweitereLesefristFuerLangeUploads(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if anfrageFrist(r.URL.Path, LesefristVorgabe) != LesefristVorgabe {
			if err := http.NewResponseController(w).SetReadDeadline(time.Now().Add(langLaufendeFrist)); err != nil {
				slog.Warn("Lesefrist nicht verlängerbar, es gilt die Vorgabe", "pfad", r.URL.Path, "fehler", err)
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

// fristMiddleware gibt jeder Anfrage eine Frist im Kontext (anfrageFrist).
//
// Der Live-Strom bleibt ohne Frist, er ist immer offen. Alles andere bekommt eine, auch die
// langen Vorgänge: Eine Anfrage ganz ohne Frist hinge im Zweifel für immer und hielte eine
// Verbindung zur Datenbank fest.
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

// panikMiddleware fängt einen Panic in einer Anfrage, schreibt ihn ins Log und antwortet mit 500.
func panikMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// maskiereToken auch hier: Der Bestätigungs-Token steht im Pfad, und ein Panic
				// auf der Seite für Lieferanten schreibt genau diesen Pfad ins Log. Ohne die
				// Maskierung enthielte ein weitergereichtes Logfile funktionierende Links, und
				// die Speicherung als Hash in der Datenbank wäre entwertet.
				slog.Error("PANIC RECOVERED", "methode", r.Method, "pfad", maskiereToken(r.URL.Path), "fehler", err, "stack", string(debug.Stack()))
				// Ohne Ursache: Die Zeile darüber nennt sie samt Stack, eine zweite zählte den Fehler doppelt.
				apierrors.AntworteFehler(w, http.StatusInternalServerError, nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// httpsMiddleware leitet unverschlüsselte Anfragen auf HTTPS um.
func (s *Server) httpsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ohne Umleitung (APP_ENV local, development, test und der Nullwert in Tests) geht jede
		// Anfrage durch.
		if !s.umleitungAufHTTPS {
			next.ServeHTTP(w, r)
			return
		}

		// /health wird nie umgeleitet.
		//
		// Die Sonde kommt aus dem Container selbst (Docker HEALTHCHECK, wget gegen 127.0.0.1) und
		// damit ohne TLS und ohne X-Forwarded-Proto. Mit Umleitung bekäme sie eine 301 auf
		// https://127.0.0.1:PORT, versuchte dort einen TLS-Handshake gegen den Klartext-Port und
		// scheiterte mit "wrong version number", im Betrieb also immer. Der Container gälte dauerhaft
		// als unhealthy, und alles, was daran hängt (Neustart-Regeln, depends_on: service_healthy,
		// das Deploy-Skript), entschiede nach einer falschen Anzeige.
		//
		// Unbedenklich: Die Antwort ist {"status":"healthy"}, kein Geheimnis, keine Sitzung, kein
		// Cookie. Eine Verbindung im Klartext gibt hier nichts preis.
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
			// Der eingestellte Ursprung verhindert Open Redirect und eine Einschleusung über
			// den Kopf Host.
			zielHost = strings.TrimPrefix(erlaubterUrsprung, "https://")
			zielHost = strings.TrimPrefix(zielHost, "http://")
		} else {
			// Ohne Einstellung gilt r.Host, für den Betrieb unter mehreren Domains.
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

// rumpfGrenzeMiddleware begrenzt, wie viel vom Rumpf einer Anfrage gelesen werden darf.
func rumpfGrenzeMiddleware(grenze int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, grenze)
			next.ServeHTTP(w, r)
		})
	}
}

// corsMiddleware lässt Anfragen anderer Ursprünge nur von der eingestellten Domain der Schule zu
// (ALLOWED_ORIGIN, etwa https://bibliothek.schule.de). Ohne Einstellung gilt nur der eigene
// Ursprung.
func corsMiddleware(next http.Handler) http.Handler {
	erlaubterUrsprung := os.Getenv("ALLOWED_ORIGIN")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ursprung := r.Header.Get("Origin")
		if ursprung == "" {
			// Anfrage vom eigenen Ursprung (ohne Kopf Origin): immer erlaubt.
			next.ServeHTTP(w, r)
			return
		}
		// Nur der eingestellte Ursprung, sonst nichts.
		if erlaubterUrsprung != "" && ursprung == erlaubterUrsprung {
			w.Header().Set("Access-Control-Allow-Origin", erlaubterUrsprung)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			// X-CSRF-Token gehört zwingend dazu: Ohne den Header in dieser Liste
			// scheitert bei gesetztem ALLOWED_ORIGIN jede ändernde Anfrage eines anderen Ursprungs
			// schon am Preflight: Der Schutz gegen CSRF sperrte die Oberfläche aus statt
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

// uuidPfadParameter nennt die Pfad-Parameter, deren Wert immer eine UUID ist, namentlich und
// abschließend, nicht nach einem Namensmuster.
//
// Ein Muster wie „alles, was auf _id endet" wäre hier gefährlich: {barcode_id} endet genauso
// auf _id, trägt aber einen Barcode (VARCHAR(100), etwa "S-abg1-ms3p73…"). Hieße die Route des
// Fotos {id}, wiese die Prüfung jeden Barcode mit 400 ab, bevor die Tür läuft, und kein Foto
// käme je an. Deshalb steht hier eine Liste, die man nur erweitert, wenn die Spalte dahinter
// nachweislich eine UUID ist.
//
// In schema.sql sind leser.id und ausleihen.id UUID. {klasse} ist ein Klassenname und gehört
// nicht dazu.
var uuidPfadParameter = []string{"id", "schueler_id", "ausleihe_id"}

// uuidPfadMiddleware weist Anfragen ab, deren UUID-Pfadparameter keine UUID
// sind, bevor sie die Datenbank erreichen. Geprüft wird nur, was in
// uuidPfadParameter steht; die Begründung für diese Liste steht dort.
func uuidPfadMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, name := range uuidPfadParameter {
			wert := r.PathValue(name)
			if wert != "" && !kennung.IstUUID(wert) {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung in der Adresse ist ungültig", nil)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// verlangeRecht lässt eine Anfrage nur durch, wenn das angemeldete Konto das Recht hat
// (tuer.Basis.PruefeRecht, nach der Tabelle der Rechte je Rolle).
func (s *Server) verlangeRecht(recht string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Die Prüfung der UUID sitzt hier und nicht um den ganzen Mux: Dort läse sie
		// r.PathValue("id"), bevor der ServeMux die Route aufgelöst hat, der Wert wäre immer leer,
		// und die Prüfung liefe nie (TestUUIDPruefungGreiftAmMux). Hier ist die Route getroffen und
		// PathValue gefüllt.
		return uuidPfadMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.PruefeRecht(w, r, next, recht)
		}))
	}
}

// statusMerker reicht die Antwort durch und merkt sich, was die Zeile zu einem 5xx braucht: den
// Status, die Route (merkeRoute) und die Ursache, die apierrors beim Antworten übergibt.
type statusMerker struct {
	http.ResponseWriter
	status  int
	route   string
	ursache string
}

// MerkeUrsache nimmt die Ursache eines 5xx von apierrors an; es gilt die erste.
func (m *statusMerker) MerkeUrsache(ursache string) {
	if m.ursache == "" {
		m.ursache = ursache
	}
}

func (m *statusMerker) WriteHeader(status int) {
	m.status = status
	m.ResponseWriter.WriteHeader(status)
}

// Unwrap liefert den ResponseWriter darunter, damit http.NewResponseController durch die Kette
// greifen kann (etwa SetWriteDeadline für den Live-Strom).
func (m *statusMerker) Unwrap() http.ResponseWriter {
	return m.ResponseWriter
}

// Flush erfüllt http.Flusher, damit der Live-Strom (SSE) jedes Ereignis sofort schickt.
func (m *statusMerker) Flush() {
	if f, ok := m.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// tokenPfade sind die einzigen Pfade, die ein Geheimnis tragen: Im Bestätigungs-Link an
// den Lieferanten steht der Token in der Adresse, sowohl beim Aufruf der Seite selbst
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

// logMiddleware schreibt eine Zeile, wenn eine Anfrage mit 5xx endet: Status, Methode, Pfad, Route
// und Ursache. Mehr nicht, keine Dauer, keine Anfragekennung, und bei 2xx bis 4xx gar nichts. So
// steht ein Serverfehler einmal im Log und zählt auf der Seite „Fehler im Log“ einmal, je Route.
// Die Zeile je Anfrage („Incoming Request", Methode und Pfad, vor der Verarbeitung) schreibt
// baueKette in router.go.
//
// Der Pfad läuft an beiden Stellen durch die Maskierung darüber: Ein Geheimnis im Pfad
// (Bestätigungs-Token) darf nicht im Klartext im Log landen.
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		merker := &statusMerker{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(merker, r.WithContext(context.WithValue(r.Context(), statusMerkerSchluessel{}, merker)))

		// Kein debug.Stack() hier: Er zeigte den Stack dieser Middleware nach next.ServeHTTP,
		// nie den der Fehlerstelle. Den Stack eines Panics schreibt panikMiddleware.
		if merker.status >= 500 {
			schreibeStatuszeile(r, merker)
		}
	})
}

// statusMerkerSchluessel ist der Schlüssel des statusMerker im Kontext der Anfrage; merkeRoute
// findet ihn darüber.
type statusMerkerSchluessel struct{}

// schreibeStatuszeile schreibt die Zeile zu einem 5xx: als ERROR, oder als WARN, wenn der Browser
// die Anfrage abgebrochen hat. Dann erreicht die Antwort niemanden mehr, und ein Fehler des Servers
// ist es nicht (docs/STIL.md, Abschnitt 11).
func schreibeStatuszeile(r *http.Request, merker *statusMerker) {
	felder := []any{"status", merker.status, "methode", r.Method, "pfad", maskiereToken(r.URL.Path)}
	if merker.route != "" {
		felder = append(felder, "route", merker.route)
	}
	if merker.ursache != "" {
		felder = append(felder, "fehler", merker.ursache)
	}
	if errors.Is(r.Context().Err(), context.Canceled) {
		slog.WarnContext(r.Context(), "Anfrage vom Browser abgebrochen", felder...)
		return
	}
	slog.ErrorContext(r.Context(), "Anfrage mit Serverfehler beendet", felder...)
}

// merkeRoute trägt nach dem Routing das Muster der Route in den statusMerker ein. Der Mux setzt
// r.Pattern an der Anfrage, die er bekommt; die Zeile zu einem 5xx zählt so je Route, nicht je Pfad,
// in dem eine Kennung stehen kann.
func merkeRoute(mux http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
		if merker, ok := r.Context().Value(statusMerkerSchluessel{}).(*statusMerker); ok {
			merker.route = r.Pattern
		}
	})
}
