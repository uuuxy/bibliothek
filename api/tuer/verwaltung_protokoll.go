package tuer

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
	AuditKlassenleitungGeaendert = "KLASSENLEITUNG_GEAENDERT"
	AuditKlassenleitungEntfernt  = "KLASSENLEITUNG_ENTFERNT"
	AuditMailvorlageGeaendert    = "MAILVORLAGE_GEAENDERT"
	AuditLieferantAngelegt       = "LIEFERANT_ANGELEGT"
	AuditLieferantGeaendert      = "LIEFERANT_GEAENDERT"
	AuditLieferantGeloescht      = "LIEFERANT_GELOESCHT"
	AuditFristKlasseGeaendert    = "FRIST_KLASSE_GEAENDERT"
	// Schlagworte: Zusammenführen, Umleiten und Löschen lassen sich nicht zurücknehmen. Die
	// Wörter sind Vokabular des Katalogs und stehen deshalb im Eintrag.
	AuditSchlagwortZusammengefuehrt = "SCHLAGWORT_ZUSAMMENGEFUEHRT"
	AuditSchlagwortVerweis          = "SCHLAGWORT_VERWEIS"
	AuditSchlagwortGeloescht        = "SCHLAGWORT_GELOESCHT"
)

// ProtokolliereVerwaltung schreibt einen Eintrag mit der angemeldeten Person als Bearbeiter.
// Die Änderung gilt auch, wenn das Protokoll klemmt; der Fehlversuch steht im Server-Log.
func (s *Basis) ProtokolliereVerwaltung(ctx context.Context, aktion string, details map[string]any) {
	claims, ok := auth.GetClaims(ctx)
	if !ok {
		return
	}
	// Server ohne Datenbank (nackte Testkonstruktion &Server{}): nichts zu schreiben.
	if s.DB == nil || s.DB.Pool == nil {
		return
	}
	if err := repository.NewProtokollAbfragen(s.DB.Pool).
		ProtokolliereAdminAktion(ctx, claims.UserID, aktion, "", details); err != nil {
		log.Printf("Protokoll (%s) fehlgeschlagen: %v", aktion, err)
	}
}

// ProtokolliereKontoAnlage schreibt USER_CREATE mit der angemeldeten Person als Bearbeiter. Zwei
// Themen legen ein Konto an, Benutzer & Rechte und die Leserdatei beim Anlegen einer Lehrkraft;
// der Eintrag steht deshalb hier. ziel_id ist die Kennung des Kontos: Daran findet die Auskunft
// den Eintrag, auch nachdem das Konto gelöscht ist (repository/dsgvo_konto.go).
func (s *Basis) ProtokolliereKontoAnlage(ctx context.Context, kontoID, email, rolle, vorname, nachname string) {
	s.ProtokolliereVerwaltung(ctx, "USER_CREATE", map[string]any{
		"ziel_id": kontoID, "email": email, "rolle": rolle, "vorname": vorname, "nachname": nachname,
	})
}

// SchreibeAdminProtokoll schreibt einen Eintrag, dessen Details als JSON-Text vorliegen, mit
// der Adresse des Aufrufers oder ohne. Der Vorgang gilt auch, wenn das Protokoll klemmt; der
// Fehlversuch steht im Server-Log.
func (s *Basis) SchreibeAdminProtokoll(ctx context.Context, adminID, aktion, ip, details string) {
	if err := repository.SchreibeAdminProtokoll(ctx, s.DB.Pool, adminID, aktion, ip, details); err != nil {
		log.Printf("audit/idempotenz: schreibvorgang fehlgeschlagen: %v", err)
	}
}

// ProtokolliereGeaenderteFelder schreibt den Eintrag einer Änderung mit den Namen der
// geänderten Felder. Ein Speichern, das nichts geändert hat, schreibt keinen Eintrag.
func (s *Basis) ProtokolliereGeaenderteFelder(ctx context.Context, aktion string, details map[string]any, felder []string) {
	if len(felder) == 0 {
		return
	}
	details["felder"] = felder
	s.ProtokolliereVerwaltung(ctx, aktion, details)
}

// GeaenderteFelder liefert die Namen, deren Wert sich geändert hat, in der Reihenfolge der
// Aufzählung.
func GeaenderteFelder(wechsel ...FeldWechsel) []string {
	felder := []string{}
	for _, w := range wechsel {
		if w.Geaendert {
			felder = append(felder, w.Name)
		}
	}
	return felder
}

// FeldWechsel nennt ein Feld und ob sein Wert sich geändert hat.
type FeldWechsel struct {
	Name      string
	Geaendert bool
}
