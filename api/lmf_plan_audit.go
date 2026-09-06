package api

// lmf_plan_audit.go — die Spur der Massenänderungen am Plan.
//
// Rasterdurchgang 06.09.2026, Frage 9 (Ausleitung/Nachvollziehbarkeit): Veröffentlichen,
// Speichern eines veröffentlichten Plans und Verwerfen schreiben über
// koppleLmfPlanFristen die Rückgabefristen ALLER Klassen des Plans um und löschen dabei,
// wo die neue Frist in der Zukunft liegt, Mahnstufe und Mahndatum der betroffenen
// Ausleihen (repository.SetzeLernmittelFristFuerKlassen). In Peters Plan sind das rund
// siebzig Klassen in einem Klick.
//
// Für die Frist EINER Ausleihe hat dieses Projekt die Antwort längst gegeben
// (api/ausleihe.go, „FRIST_OVERRIDE"): „Ein manuell überschriebenes Fälligkeitsdatum ist
// ein Eingriff in eine Sanktion — er gehört revisionssicher protokolliert." Für die
// tausendfache Fassung desselben Eingriffs stand bis heute nichts im Protokoll.
//
// Best effort wie überall sonst: Ein klemmendes Audit bricht den Vorgang nicht ab, der
// Fehlversuch steht im Server-Log.

import (
	"log"
	"net/http"

	"bibliothek/auth"
	"bibliothek/repository"
)

const (
	auditLmfPlanVeroeffentlicht = "LMF_PLAN_VEROEFFENTLICHT"
	auditLmfPlanGespeichert     = "LMF_PLAN_GESPEICHERT"
	auditLmfPlanVerworfen       = "LMF_PLAN_VERWORFEN"
)

// auditiereLmfPlan protokolliert einen Eingriff am Plan samt der Zahl der Ausleihen,
// deren Frist er bewegt hat.
func (s *Server) auditiereLmfPlan(r *http.Request, aktion, art string, planID string, fristen int64) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		return
	}
	// Server ohne Datenbank (nackte Testkonstruktion &Server{}): nichts zu schreiben.
	if s.DB == nil || s.DB.Pool == nil {
		return
	}
	details := map[string]any{"art": art, "plan_id": planID, "fristen_angepasst": fristen}
	if err := repository.NewAuditRepository(s.DB.Pool).
		LogAdminAktion(r.Context(), claims.UserID, aktion, getIP(r), details); err != nil {
		log.Printf("LMF-Plan-Audit (%s) fehlgeschlagen: %v", aktion, err)
	}
}
