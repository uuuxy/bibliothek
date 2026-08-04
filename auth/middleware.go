package auth

import (
	"context"
)

type contextKey string

// ClaimsContextKey ist der Schlüssel, der verwendet wird, um Authentifizierungs-Claims im Request-Kontext zu speichern und abzurufen.
const ClaimsContextKey contextKey = "auth_claims"

// Hier standen bis zum 04.08.2026 RequireRoles samt Helfern (authClaimsAusRequest,
// enthaeltRolle) — eine zweite Autorisierungsmechanik, die anhand von ROLLENNAMEN
// entschied statt anhand der konfigurierbaren role_permissions-Tabelle.
//
// Ihre letzten beiden Nutzer waren die Demo-Dashboards /admin/dashboard und
// /teacher/dashboard, die nichts taten als einen englischen Satz auszugeben; mit deren
// Entfernung wurde die Kette unerreichbar (bestätigt von golang.org/x/tools/deadcode,
// das Erreichbarkeit über den echten Aufrufgraphen ab main() rechnet).
//
// Ersatzlos: Autorisierung läuft einheitlich über api.RequirePermission. Zwei parallele
// Mechaniken waren schon einmal ein Problem — die frühere RBACBlockMiddleware überstimmte
// role_permissions und nahm einem LEHRER Rechte, die ein Admin ihm ausdrücklich erteilt hatte.

// GetClaims ruft Authentifizierungs-Claims aus dem Request-Kontext ab.
// Gibt die Claims und einen booleschen Wert zurück, der angibt, ob die Claims vorhanden waren.
func GetClaims(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(*Claims)
	return claims, ok
}
