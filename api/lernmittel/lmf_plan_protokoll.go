package lernmittel

// Die Spur der Massenänderungen am Plan. Veröffentlichen, Speichern eines veröffentlichten
// Plans und Verwerfen schreiben über service.KoppleLMFPlanFristen die Rückgabefristen aller
// Klassen des Plans um und löschen dabei, wo die neue Frist in der Zukunft liegt, Mahnstufe
// und Mahndatum der betroffenen Ausleihen (repository.LMFAbfragen.SetzeLernmittelFristFuerKlassen).
// Im Plan der Schule sind das rund siebzig Klassen in einem Klick.
//
// Die Frist einer einzelnen Ausleihe steht im Protokoll (api/theke/ausleihe.go,
// „FRIST_OVERRIDE"), weil eine überschriebene Frist ein Eingriff in eine Sanktion ist; die
// tausendfache Fassung desselben Eingriffs gehört ebenso hinein. Ein klemmendes Protokoll bricht
// den Vorgang nicht ab, der Fehlversuch steht im Log des Servers.

import (
	"log"
	"net/http"

	"bibliothek/auth"
	"bibliothek/pkg/clientip"
	"bibliothek/repository"
)

const (
	aktionLMFPlanVeroeffentlicht = "LMF_PLAN_VEROEFFENTLICHT"
	aktionLMFPlanGespeichert     = "LMF_PLAN_GESPEICHERT"
	aktionLMFPlanVerworfen       = "LMF_PLAN_VERWORFEN"
)

// protokolliereLMFPlan protokolliert einen Eingriff am Plan samt der Zahl der Ausleihen,
// deren Frist er bewegt hat.
func (t *Tueren) protokolliereLMFPlan(r *http.Request, aktion, art string, planID string, fristen int64) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		return
	}
	// Türen ohne Datenbank (Testkonstruktion mit leerer Basis): nichts zu schreiben.
	if t.DB == nil || t.DB.Pool == nil {
		return
	}
	details := map[string]any{"art": art, "plan_id": planID, "fristen_angepasst": fristen}
	if err := repository.NewProtokollAbfragen(t.DB.Pool).
		ProtokolliereAdminAktion(r.Context(), claims.UserID, aktion, clientip.AusAnfrage(r), details); err != nil {
		log.Printf("LMF-Plan-Audit (%s) fehlgeschlagen: %v", aktion, err)
	}
}
