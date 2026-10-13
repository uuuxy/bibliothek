// Package middleware setzt die Sicherheits-Kopfzeilen jeder Antwort und für alles unter /api/
// die Vorgabe, dass nichts zwischengespeichert wird.
package middleware

import (
	"net/http"
	"strings"
)

// Sicherheitskopfzeilen setzt an jeder Antwort die Kopfzeilen für die Sicherheit: eine
// Content-Security-Policy, die auf die eigene Herkunft beschränkt ist, X-Content-Type-Options,
// X-Frame-Options, X-XSS-Protection, Referrer-Policy und Permissions-Policy. Für alles unter
// /api/ kommt Cache-Control: no-store dazu, als Vorgabe, die eine Tür überschreiben kann.
func Sicherheitskopfzeilen(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Die Content-Security-Policy ist so eng wie möglich auf 'self' gestellt.
		//
		// base-uri 'self': base-uri fällt nicht auf default-src zurück. Ohne die Direktive böge
		// ein eingeschleustes <base href="…"> jede relative URL der Seite um, auch die Ziele der
		// Formulare und die Aufrufe der Schnittstelle. Die Oberfläche setzt selbst kein <base>.
		//
		// object-src 'none': <object> und <embed> kommen in der Oberfläche nicht vor, und
		// Inhalte von Plugins umgehen script-src.
		//
		// style-src behält 'unsafe-inline': Die Oberfläche setzt style-Attribute mit berechneten
		// Werten, etwa im Ausweis-Designer (style="left: {el.x}mm"). Eine Nonce gilt für
		// <style>-Elemente und nicht für style-Attribute, und der Hash eines berechneten Werts
		// ändert sich bei jedem Zeichnen. Entfallen kann die Angabe erst, wenn diese Stellen
		// CSS-Variablen nehmen.
		//
		// img-src ohne https: Mit einem pauschalen https: lüde ein aus Stammdaten
		// eingeschleustes <img src="https://fremder-host/?daten=…"> im Druckfenster und trüge den
		// Inhalt der gedruckten Liste nach außen; Skripte hält die Richtlinie auf, Bilder wären
		// der offene Weg. Cover kommen deshalb über den eigenen Proxy
		// (frontend/src/lib/utils/coverSrc.js), und kein Dritter erfährt, welche ISBN die Schule
		// ansieht. data: und blob: bleiben für erzeugte Barcodes und QR-Codes und für die
		// Vorschau einer Datei vor dem Hochladen.
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data: blob:; connect-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'self'; object-src 'none';")

		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions-Policy löst Feature-Policy ab. Die Kamera bleibt erlaubt, weil die
		// Oberfläche Barcodes mit ihr liest.
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(self)")

		// Antworten der Schnittstelle legt der Browser nicht ab, nach OWASP ASVS 5.0, 14.3.2:
		// „anti-caching HTTP response header fields (i.e., Cache-Control: no-store) so that
		// sensitive data is not cached in browsers". no-store verbietet auch jedem
		// Zwischenspeicher im Netz das Ablegen (RFC 9111, 5.2.2.5). Ohne den Kopf dürfte der
		// Browser Listen, Akten und die Auskunft als PDF auf der Festplatte ablegen, und das
		// Abmelden löscht dort nichts.
		//
		// Nur /api/: Die Programmdateien der Oberfläche tragen keine Personendaten. Die Vorgabe
		// steht vor der Tür, und ein eigener Kopf überschreibt sie, so bei den Covern und den
		// Buchnummern der Theke, beide ohne Personendaten. Dass keine Antwort mit Personendaten
		// sie aufweicht, prüft api/pii_antwort_gate_pg_test.go an jeder Route ab Stufe 1.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}

		next.ServeHTTP(w, r)
	})
}
