package auth

import (
	"context"
)

type kontextSchluessel string

// ClaimsSchluessel ist der Schlüssel, unter dem die Claims der Sitzung im Kontext der Anfrage
// stehen.
const ClaimsSchluessel kontextSchluessel = "auth_claims"

// ClaimsAusKontext liefert die Claims der Sitzung aus dem Kontext der Anfrage; ok ist false,
// wenn dort keine stehen. Was die Sitzung darf, entscheidet allein api.verlangeRecht an
// role_permissions, nicht der Name der Rolle.
func ClaimsAusKontext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ClaimsSchluessel).(*Claims)
	return claims, ok
}
