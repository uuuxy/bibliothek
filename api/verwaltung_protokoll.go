package api

import (
	"context"
	"log"

	"bibliothek/auth"
	"bibliothek/repository"
)

// Einträge der Verwaltung im Protokoll (audit_logs): wohin Mahnlisten und Bestellungen
// gehen, was in den Mails steht und wann eine Klasse ihre Schulbücher abgibt. Ein Eintrag
// nennt den Gegenstand und die Namen der geänderten Felder, keine Mailadresse: Sie bliebe
// bis zur Aufbewahrungsfrist des Protokolls stehen, und die Tilgung kennt sie dort nicht.
const (
	auditKlassenleitungGeaendert = "KLASSENLEITUNG_GEAENDERT"
	auditKlassenleitungEntfernt  = "KLASSENLEITUNG_ENTFERNT"
	auditMailvorlageGeaendert    = "MAILVORLAGE_GEAENDERT"
	auditLieferantAngelegt       = "LIEFERANT_ANGELEGT"
	auditLieferantGeaendert      = "LIEFERANT_GEAENDERT"
	auditLieferantGeloescht      = "LIEFERANT_GELOESCHT"
	auditFristKlasseGeaendert    = "FRIST_KLASSE_GEAENDERT"
	// Schlagworte: Zusammenführen, Umleiten und Löschen lassen sich nicht zurücknehmen. Die
	// Wörter sind Vokabular des Katalogs und stehen deshalb im Eintrag.
	auditSchlagwortZusammengefuehrt = "SCHLAGWORT_ZUSAMMENGEFUEHRT"
	auditSchlagwortVerweis          = "SCHLAGWORT_VERWEIS"
	auditSchlagwortGeloescht        = "SCHLAGWORT_GELOESCHT"
)

// protokolliereVerwaltung schreibt einen Eintrag mit der angemeldeten Person als Bearbeiter.
// Die Änderung gilt auch, wenn das Protokoll klemmt; der Fehlversuch steht im Server-Log.
func (s *Server) protokolliereVerwaltung(ctx context.Context, aktion string, details map[string]any) {
	claims, ok := auth.GetClaims(ctx)
	if !ok {
		return
	}
	// Server ohne Datenbank (nackte Testkonstruktion &Server{}): nichts zu schreiben.
	if s.DB == nil || s.DB.Pool == nil {
		return
	}
	if err := repository.NewAuditRepository(s.DB.Pool).
		LogAdminAktion(ctx, claims.UserID, aktion, "", details); err != nil {
		log.Printf("Protokoll (%s) fehlgeschlagen: %v", aktion, err)
	}
}

// schreibeAdminProtokoll schreibt einen Eintrag, dessen Details als JSON-Text vorliegen, mit
// der Adresse des Aufrufers oder ohne. Der Vorgang gilt auch, wenn das Protokoll klemmt; der
// Fehlversuch steht im Server-Log.
func (s *Server) schreibeAdminProtokoll(ctx context.Context, adminID, aktion, ip, details string) {
	if err := repository.SchreibeAdminProtokoll(ctx, s.DB.Pool, adminID, aktion, ip, details); err != nil {
		log.Printf("audit/idempotenz: schreibvorgang fehlgeschlagen: %v", err)
	}
}

// protokolliereGeaenderteFelder schreibt den Eintrag einer Änderung mit den Namen der
// geänderten Felder. Ein Speichern, das nichts geändert hat, schreibt keinen Eintrag.
func (s *Server) protokolliereGeaenderteFelder(ctx context.Context, aktion string, details map[string]any, felder []string) {
	if len(felder) == 0 {
		return
	}
	details["felder"] = felder
	s.protokolliereVerwaltung(ctx, aktion, details)
}

// geaenderteFelder liefert die Namen, deren Wert sich geändert hat, in der Reihenfolge der
// Aufzählung.
func geaenderteFelder(wechsel ...feldWechsel) []string {
	felder := []string{}
	for _, w := range wechsel {
		if w.geaendert {
			felder = append(felder, w.name)
		}
	}
	return felder
}

// feldWechsel: der Name eines Felds und ob sein Wert sich geändert hat.
type feldWechsel struct {
	name      string
	geaendert bool
}
