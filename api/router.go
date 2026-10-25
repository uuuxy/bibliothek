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

// Package api ist die HTTP-Schicht des Servers: der Router mit den Routenlisten und den Zugängen
// zu den Themen, die Middleware, CSRF, die Bremse je Adresse und die Abmeldung. Die Türen stehen
// je Thema in einem Paket darunter (docs/ARCHITEKTUR.md 5.2.2).
package api

// Hier stehen der Server, sein Router und die Zugänge zu den Themen; die Middleware steht in
// middleware.go, jede Route mit ihrem Recht in routen_*.go.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"bibliothek/api/benutzer"
	"bibliothek/api/bestellung"
	"bibliothek/api/druck"
	"bibliothek/api/inventurlauf"
	"bibliothek/api/katalog"
	"bibliothek/api/lernmittel"
	"bibliothek/api/leser"
	"bibliothek/api/mahnwesen"
	"bibliothek/api/portal"
	"bibliothek/api/schaden"
	"bibliothek/api/system"
	"bibliothek/api/theke"
	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/katalogdienste"
	"bibliothek/internal/middleware"
	"bibliothek/internal/service"
	"bibliothek/mailservice"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/httpresp"
	"bibliothek/pkg/logger"
	"bibliothek/repository"
	"bibliothek/sse"

	sentryhttp "github.com/getsentry/sentry-go/http"
)

// Server hält, was jede Tür braucht (Datenbank, Anmeldung, Live-Leitung in tuer.Basis), ob er
// Anfragen über HTTP auf HTTPS umleitet und welchen Proxys er die Adresse des Browsers glaubt.
type Server struct {
	tuer.Basis
	umleitungAufHTTPS bool
	// vertrauteProxys sind die Netze der Proxys, deren X-Forwarded-For gilt; Loopback gilt immer.
	vertrauteProxys []string
	// coverAbgleich ist der Abgleich der Cover, den der Knopf in der Verwaltung startet.
	coverAbgleich *service.CoverDienst
	// fehlerstand zählt die Zeilen der Stufe ERROR im Log; die Selbstprüfung zeigt ihn.
	fehlerstand *logger.Fehlerstand
	// bremse zählt die gescheiterten Anmeldungen je Adresse; der Nullwert ist leer.
	bremse anmeldeBremse
	// katalogdienste fragt DNB, Google Books und OpenLibrary, ein Client für alle Türen.
	katalogdienste *katalogdienste.MetadatenClient
}

// ServerKonfig ist, was der Start dem Server über Datenbank, Anmeldung und Live-Leitung hinaus
// mitgibt. Der Nullwert gilt für Tests: ohne Umleitung auf HTTPS, als Proxy nur Loopback, mit
// eigenem Abgleich der Cover.
type ServerKonfig struct {
	// UmleitungAufHTTPS leitet Anfragen über HTTP auf HTTPS um; der Start setzt sie im Betrieb.
	UmleitungAufHTTPS bool
	// VertrauteProxys sind CIDR-Angaben oder Adressen der Proxys, deren X-Forwarded-For gilt
	// (TRUSTED_PROXIES).
	VertrauteProxys []string
	// CoverAbgleich ist der Abgleich der Cover, den Start und Zeitplan mit dem Knopf in der
	// Verwaltung teilen; nil heißt, der Server legt einen eigenen an.
	CoverAbgleich *service.CoverDienst
	// Fehlerstand zählt die Zeilen der Stufe ERROR im Log des Servers (main.go); nil heißt, die
	// Selbstprüfung meldet den Bereich als nicht erhoben.
	Fehlerstand *logger.Fehlerstand
}

// NewServer baut den Server. Sein Mailversand liest die in der Oberfläche gespeicherte
// SMTP-Konfiguration, nicht die Umgebungsvariablen des Containers; so nutzen der Test-Knopf und
// jeder echte Versand dieselbe Quelle.
func NewServer(datenbank *db.Verbindung, authenticator *auth.Anmeldung, broker *sse.Broker, konfig ServerKonfig) *Server {
	coverAbgleich := konfig.CoverAbgleich
	if coverAbgleich == nil {
		coverAbgleich = service.NewCoverDienst(datenbank.Pool)
	}
	return &Server{
		Basis: tuer.Basis{
			DB: datenbank, Auth: authenticator, Broker: broker,
			Mail: mailservice.NewPostausgang(mailservice.KonfigAusDatenbank(datenbank.Pool)),
		},
		umleitungAufHTTPS: konfig.UmleitungAufHTTPS,
		vertrauteProxys:   konfig.VertrauteProxys,
		coverAbgleich:     coverAbgleich,
		fehlerstand:       konfig.Fehlerstand,
		katalogdienste:    katalogdienste.NewMetadatenClient(),
	}
}

// lernmittel ist der Zugang zu den Türen des LMF-Plans. Ein Thema bekommt die Basis des Servers
// als Zeiger, keine Kopie: Was am Server gesetzt wird, gilt auch für seine Türen.
func (s *Server) lernmittel() *lernmittel.Tueren {
	return &lernmittel.Tueren{Basis: &s.Basis}
}

// inventurlauf ist der Zugang zu den Türen der Inventur.
func (s *Server) inventurlauf() *inventurlauf.Tueren {
	return &inventurlauf.Tueren{Basis: &s.Basis}
}

// benutzer ist der Zugang zu den Türen zu Benutzer & Rechte.
func (s *Server) benutzer() *benutzer.Tueren {
	return &benutzer.Tueren{Basis: &s.Basis}
}

// portal ist der Zugang zu den Türen des Portals.
func (s *Server) portal() *portal.Tueren {
	return &portal.Tueren{Basis: &s.Basis}
}

// system ist der Zugang zu den Türen der Systempflege.
func (s *Server) system() *system.Tueren {
	return &system.Tueren{Basis: &s.Basis, VertrauteProxys: s.vertrauteProxys, Fehlerstand: s.fehlerstand}
}

// BereitschaftsAlarm prüft die Betriebsbereitschaft und mailt kritische Befunde an die Admins.
// Der Start ruft ihn einmal am Tag (main.go). Die Prüfung steht bei den Türen der Systempflege,
// weil die Seite „Betriebsbereitschaft" dieselbe Lage sammelt.
func (s *Server) BereitschaftsAlarm(ctx context.Context) {
	s.system().BereitschaftsAlarm(ctx)
}

// mahnwesen ist der Zugang zu den Türen des Mahnwesens und der Abgänger.
func (s *Server) mahnwesen() *mahnwesen.Tueren {
	return &mahnwesen.Tueren{Basis: &s.Basis}
}

// schaden ist der Zugang zu den Türen zu Schaden und Ersatz.
func (s *Server) schaden() *schaden.Tueren {
	return &schaden.Tueren{Basis: &s.Basis}
}

// katalog ist der Zugang zu den Türen des Medienkatalogs zu einem Titel und seinen Exemplaren.
func (s *Server) katalog() *katalog.Tueren {
	return &katalog.Tueren{Basis: &s.Basis, Katalogdienste: s.katalogdienste}
}

// druck ist der Zugang zu den Türen des Druck-Centers.
func (s *Server) druck() *druck.Tueren {
	return &druck.Tueren{Basis: &s.Basis}
}

// bestellung ist der Zugang zu den Türen des Bestellwesens.
func (s *Server) bestellung() *bestellung.Tueren {
	return &bestellung.Tueren{Basis: &s.Basis}
}

// leser ist der Zugang zu den Türen der Leserdatei.
func (s *Server) leser() *leser.Tueren {
	return &leser.Tueren{Basis: &s.Basis}
}

// theke ist der Zugang zu den Türen der Theke.
func (s *Server) theke() *theke.Tueren {
	return &theke.Tueren{Basis: &s.Basis}
}

// Routes baut den Router: jede Adresse mit ihrer Tür und ihrem Recht (routen_*.go), darum die
// Middleware-Kette (baueKette).
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Die Abfragen und Dienste, die mehrere Routenlisten brauchen.
	leserAbfragen := repository.NewLeserAbfragen(s.DB.Pool)
	katalogAbfragen := repository.NewKatalogAbfragen(s.DB.Pool)
	ausleihAbfragen := repository.NewAusleihAbfragen(s.DB.Pool)
	protokollAbfragen := repository.NewProtokollAbfragen(s.DB.Pool)
	mahnAbfragen := repository.NewMahnwesenAbfragen(s.DB.Pool)
	kontoAbfragen := repository.NewKontoAbfragen(s.DB.Pool)

	ausleihDienst := service.NewAusleihDienst(s.DB.Pool, leserAbfragen, katalogAbfragen, ausleihAbfragen, protokollAbfragen)
	geraeteDienst := service.NewGeraeteDienst(s.DB.Pool, leserAbfragen, ausleihAbfragen, protokollAbfragen)
	omniboxDienst := service.NewOmniboxDienst(s.DB.Pool, leserAbfragen, katalogAbfragen, ausleihAbfragen, ausleihDienst, geraeteDienst)
	nachbuchDienst := service.NewNachbuchDienst(s.DB.Pool, leserAbfragen, katalogAbfragen, ausleihAbfragen, protokollAbfragen)
	bestellDienst := service.NewBestellDienst(s.DB, katalogAbfragen)

	s.registriereUploads(mux)
	s.registriereAnmeldeRouten(mux)

	// Die Routenlisten je Sache.
	s.registriereOeffentlicheRouten(mux)
	s.registriereThekenUndInventurRouten(mux, leserAbfragen, katalogAbfragen, omniboxDienst, nachbuchDienst)
	s.registriereLeserRouten(mux, leserAbfragen, mahnAbfragen, protokollAbfragen)
	s.registriereKatalogRouten(mux, katalogAbfragen, protokollAbfragen)
	s.registriereSystemRouten(mux, protokollAbfragen, kontoAbfragen, s.DB.Pool)
	s.registriereBestellRouten(mux, bestellDienst)

	s.registriereImportRouten(mux)
	s.registriereOberflaechenRouten(mux)

	return s.baueKette(mux)
}

func (s *Server) registriereUploads(mux *http.ServeMux) {
	// Das Verzeichnis der hochgeladenen und abgerufenen Cover.
	if err := os.MkdirAll("uploads", 0750); err != nil {
		slog.Error("router: Upload-Verzeichnis konnte nicht angelegt werden", "fehler", err)
	}
	// Ein Unterordner uploads/fotos entsteht hier nicht: Schülerfotos liegen verschlüsselt
	// in der Datenbank, und /uploads/ ist ohne Anmeldung lesbar. Ausgeliefert wird nur,
	// was direkt in uploads/ liegt (api/katalog/uploads.go).
	mux.Handle("GET /uploads/", s.katalog().Uploads())
}

func (s *Server) registriereAnmeldeRouten(mux *http.ServeMux) {
	// Ohne Anmeldung erreichbar.
	mux.Handle("POST /login", s.bremse.schuetze(http.HandlerFunc(auth.Anmelden(s.DB.Pool, s.Auth))))

	// Cover aus dem Cache, ohne Anmeldung.
	mux.HandleFunc("GET /api/images/cover", s.katalog().Cover())

	// Das CSRF-Token, ohne Anmeldung: Aufrufer ohne Browser holen Token und Cookie vor der ersten
	// Änderung.
	mux.HandleFunc("GET /api/csrf-token", s.csrfToken())

	// Die Sitzung verlängern; von der CSRF-Prüfung ausgenommen (istPruefungsAusnahme).
	mux.HandleFunc("POST /api/auth/refresh", auth.SitzungVerlaengern(s.Auth))

	// Die Sitzung beim Start der Oberfläche wiederherstellen: prüft Cookie und Zustand in der
	// Datenbank und antwortet wie die Anmeldung.
	mux.HandleFunc("GET /api/auth/me", auth.AngemeldetesKonto(s.DB.Pool, s.Auth))

	// Abmelden: widerruft das Token und löscht das Sitzungscookie.
	mux.HandleFunc("POST /api/auth/logout", s.abmelden())

	// Sperre nach Inaktivität: gilt am Server für die Anmeldung, mit der die Anfrage kommt.
	// Entsperren prüft ein Passwort und hängt deshalb an derselben Bremse wie /login.
	mux.HandleFunc("POST /api/auth/sperren", auth.Sperren(s.Auth))
	mux.Handle("POST /api/auth/entsperren", s.bremse.schuetze(auth.Entsperren(s.DB.Pool, s.Auth)))

	mux.HandleFunc("GET /health", s.lebenszeichen())
}

func (s *Server) registriereImportRouten(mux *http.ServeMux) {
	// Der Import aus Littera, der Bestandsimport und der Abgleich der Cover (manage_inventory).
	mux.Handle("POST /api/import/littera", s.verlangeRecht("manage_inventory")(s.system().LitteraImport()))
	mux.Handle("POST /api/admin/import-bestand", s.verlangeRecht("manage_inventory")(s.system().BestandImport()))
	mux.Handle("POST /api/admin/sync-covers", s.verlangeRecht("manage_inventory")(s.katalog().CoverAbgleichen(s.coverAbgleich)))
}

func (s *Server) registriereOberflaechenRouten(mux *http.ServeMux) {
	// Ein fehlendes favicon.ico antwortet 204, statt auf index.html zu fallen.
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Die gebaute Oberfläche aus dist/. os.OpenRoot bindet jeden Dateizugriff im Betriebssystem
	// an dist/; ein os.Stat auf einen zusammengesetzten Pfad verriete, ob eine Datei außerhalb
	// des Verzeichnisses existiert (gosec G304).
	oberflaeche, oberflaecheFehler := os.OpenRoot("./frontend/dist")
	if oberflaecheFehler != nil {
		slog.Warn("router: frontend/dist nicht verfügbar — SPA-Auslieferung deaktiviert", "fehler", oberflaecheFehler)
	}
	mux.HandleFunc("/", liefereOberflaeche(oberflaeche))
}

func (s *Server) baueKette(mux http.Handler) http.Handler {
	// 100 MB — bewusst großzügig und bewusst global.
	//
	// Die Zahl klingt nach Speicherrisiko, ist aber keins: rumpfGrenzeMiddleware setzt
	// http.MaxBytesReader, und der begrenzt nur, wie viel eine Tür lesen darf. Er
	// puffert nichts. Wer 100 MB schickt, ohne dass eine Tür sie liest, belegt keinen
	// Speicher; wer mehr schickt, bekommt einen Lesefehler. Vorab liest nur die Prüfung auf
	// Null-Zeichen, von einem JSON-Rumpf höchstens 64 KiB (nullZeichenVorlauf).
	//
	// Kleiner setzen wäre riskanter als es aussieht: Über diesen einen Wert laufen auch
	// der Littera-Import, der Bestands-Import und der Excel-Katalogimport. Ein zu enger
	// Wert bricht sie mitten im Lauf, und zwar mit einer Meldung, die nicht nach
	// "Datei zu groß" klingt. Eine getrennte, kleinere Grenze für alles außer den
	// Import-Routen wäre der saubere Weg; dafür muss aber jede Upload-Route einzeln
	// belegt sein, sonst bricht genau die eine, die man übersehen hat.
	rumpfGrenze := rumpfGrenzeMiddleware(100 * 1024 * 1024)
	bremse := anfrageBremse(anfragenJeSekundeAusUmgebung())
	frist := fristMiddleware(bearbeitungsfristVorgabe)
	adressen := clientip.NewErmittler(s.vertrauteProxys)

	// Kette: Adresse → panik → sentry → Sicherheitskopfzeilen → cors → log → https → Lesefrist →
	// rumpfGrenze → frist → bremse → csrf → Null-Zeichen → kompression → Route merken → Mux.
	//
	// Die Adresse des Browsers ermittelt das erste Glied, damit jede Bremse und jede Tür
	// dahinter sie mit clientip.AusAnfrage liest.
	//
	// Kompression sitzt innen am Mux: Gepackt wird, was eine Tür liefert. Hinter CORS,
	// weil CORS den Vary-Kopf setzt statt ergänzt und Accept-Encoding sonst daraus fiele.
	//
	// Die Prüfung auf Null-Zeichen steht hinter csrf: Den Rumpf einer fremden Seite liest sie
	// gar nicht erst vorab.
	//
	// erweitereLesefristFuerLangeUploads steht vor rumpfGrenze und damit vor jedem Lesen des
	// Rumpfes; danach gesetzt käme die Frist zu spät. Sie hebt die strenge
	// serverweite ReadTimeout (main.go) für die Import-Endpunkte an.
	sentry := sentryhttp.New(sentryhttp.Options{Repanic: true}).Handle
	// uuidPfadMiddleware steht nicht in dieser Kette: Um den Mux gelegt läse sie
	// r.PathValue("id"), bevor die Route aufgelöst ist, und der Wert wäre immer leer. Sie sitzt
	// in verlangeRecht, hinter dem Routing, wo PathValue gefüllt ist.
	kette := adressen.Middleware(panikMiddleware(sentry(middleware.Sicherheitskopfzeilen(corsMiddleware(logMiddleware(s.httpsMiddleware(erweitereLesefristFuerLangeUploads(rumpfGrenze(frist(bremse(s.csrfMiddleware(nullZeichenMiddleware(kompressionMiddleware(merkeRoute(mux)))))))))))))))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Jede Anfrage ins Log, ohne die Adresse des Aufrufers (DSGVO).
		//
		// maskiereToken: Der Bestätigungs-Link an den Lieferanten trägt sein Geheimnis im
		// Pfad. Ohne die Maskierung stünde jeder verschickte Link im Klartext in dieser
		// Zeile und damit in einem Logfile, das weitergereicht und in Tickets gehängt
		// wird. Das hätte die Hash-Speicherung in der Datenbank wieder entwertet.
		// #nosec G706
		slog.Info("Incoming Request", "methode", r.Method, "pfad", maskiereToken(r.URL.Path))
		kette.ServeHTTP(w, r)
	})
}

// lebenszeichen antwortet, ob die Datenbank erreichbar ist; Docker fragt es als Sonde ab
// (HEALTHCHECK).
func (s *Server) lebenszeichen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(tuer.HeaderContentType, tuer.ContentTypeJSON)
		if err := s.DB.Pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			httpresp.Schreibe(w, []byte(`{"status":"unhealthy","error":"database unreachable"}`))
			return
		}
		httpresp.Schreibe(w, []byte(`{"status":"healthy"}`))
	}
}

// liefereOberflaeche liefert die Dateien der gebauten Oberfläche aus oberflaeche und fällt für
// unbekannte Pfade auf die App-Shell (index.html) zurück. os.OpenRoot bindet jeden Dateizugriff
// im Betriebssystem an dist/ (gegen Path-Traversal, gosec G304).
func liefereOberflaeche(oberflaeche *os.Root) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Ein unbekannter /api/-Pfad ist ein Fehler, keine Seite: Bekäme er die App-Shell,
		// antwortete ein vertippter, umbenannter oder mit der falschen Methode gerufener
		// Endpunkt mit "200 text/html", und der Aufrufer scheiterte erst beim Lesen des JSON,
		// mit einer Meldung, die nichts mit der Ursache zu tun hat. Das gilt für jede Methode:
		// Ein POST an einen umbenannten Endpunkt ist kein Fehler der Methode.
		//
		// Die App-Shell bleibt für alles andere richtig: /schuelerdatei, /bestellungen und jede
		// andere Route der Oberfläche gibt es nur im Browser.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, fmt.Sprintf("Unbekannter Endpunkt: %s %s", r.Method, r.URL.Path), nil)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusMethodNotAllowed, "Diese Methode ist hier nicht erlaubt", nil)
			return
		}
		if oberflaeche == nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusServiceUnavailable, "Die Oberfläche ist auf dem Server nicht vorhanden", nil)
			return
		}
		if liefereVorhandeneDatei(w, r, oberflaeche, strings.TrimPrefix(r.URL.Path, "/")) {
			return
		}

		// Alle übrigen Pfade bekommen die App-Shell.
		liefereOhneDatum(w, r, oberflaeche, "index.html")
	}
}

// liefereVorhandeneDatei liefert die Datei der Oberfläche unter relativ; false, wenn es keine
// gibt. Unter assets/ trägt jeder Name einen Prüfwert: Ein anderer Bau nennt andere Dateien, der
// Vergleich nach Datum ist dort harmlos.
func liefereVorhandeneDatei(w http.ResponseWriter, r *http.Request, oberflaeche *os.Root, relativ string) bool {
	if relativ == "" {
		return false
	}
	if angabe, err := oberflaeche.Stat(relativ); err != nil || angabe.IsDir() {
		return false
	}
	if strings.HasPrefix(relativ, "assets/") {
		http.ServeFileFS(w, r, oberflaeche.FS(), relativ)
		return true
	}
	liefereOhneDatum(w, r, oberflaeche, relativ)
	return true
}

// liefereOhneDatum liefert eine Datei ohne Prüfwert im Namen (index.html, sw.js, das Manifest)
// bei jedem Aufruf neu, ohne Vergleich nach Datum: Nach einem Rückbau ist sie älter als die im
// Browser, und If-Modified-Since bekäme 304. Mit Service Worker erkennt der Browser einen
// anderen Bau nur an sw.js; die App-Shell nimmt er dann aus dessen Speicher.
func liefereOhneDatum(w http.ResponseWriter, r *http.Request, oberflaeche *os.Root, name string) {
	inhalt, err := fs.ReadFile(oberflaeche.FS(), name)
	if errors.Is(err, fs.ErrNotExist) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set(tuer.HeaderCacheControl, "no-cache")
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(inhalt))
}
