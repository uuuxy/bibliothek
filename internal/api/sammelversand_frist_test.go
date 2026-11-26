package api

import (
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

// Wartet die Oberfläche auf eine Route FRIST_SAMMELVERSAND_MS lang (die Frist der lang laufenden
// Vorgänge, TestMailFristen_DeckenDenVersandAb), steht die Route in langLaufendePfade. Sonst
// endet der Kontext der Anfrage nach 15 Sekunden, und jeder Schritt im Versand, der ihn fragt,
// bricht mitten im Lauf ab. Ohne den Eintrag liefen die Sammelläufe nur zu Ende, weil sich
// Konfiguration, SMTP und Protokoll vom Ende der Anfrage lösen.
//
// Blind für einen Aufruf, dessen Adresse die Oberfläche aus Teilen zusammensetzt.
func TestSammelversand_HatDieFristDerLangLaufendenVorgaenge(t *testing.T) {
	pfade := sammelversandPfade(oberflaechenQuellen(t))
	if len(pfade) < 2 {
		t.Fatalf("nur %d Aufrufe mit FRIST_SAMMELVERSAND_MS gefunden (%v): Der Sammler sieht sie nicht mehr", len(pfade), pfade)
	}
	for _, pfad := range pfade {
		if !istLangLaufend(pfad) {
			t.Errorf("%s: Die Oberfläche wartet FRIST_SAMMELVERSAND_MS, der Server gibt der Anfrage "+
				"15 s. Den Pfad in langLaufendePfade (middleware.go) aufnehmen.", pfad)
		}
	}
}

func TestSammelversandPfade_Selbstprobe(t *testing.T) {
	quellen := map[string]string{
		"sammel.js": "return apiFetch('/api/mail/send-bulk-overdue', {\n\tmethod: 'POST',\n\ttimeoutMs: FRIST_SAMMELVERSAND_MS\n});",
		"platz.js":  "await apiFetch(`/api/klassen/${k}/mail`, { method: 'POST', timeoutMs: FRIST_SAMMELVERSAND_MS });",
		"einzel.js": "await apiPost('/api/bestellungen', { a: 1 }, { timeoutMs: FRIST_MAILVERSAND_MS });",
		"kurz.js":   "await apiFetch('/api/abgaenger/mail', { method: 'POST' });",
	}
	gefunden := sammelversandPfade(quellen)
	if want := []string{"/api/klassen/", "/api/mail/send-bulk-overdue"}; !slices.Equal(gefunden, want) {
		t.Errorf("sammelversandPfade = %v, erwartet %v", gefunden, want)
	}
}

// aufrufMitAdresse ist der Anfang eines Aufrufs, dessen erstes Argument eine Adresse unter /api/
// ist; die Adresse endet vor einem Platzhalter oder den Parametern.
var aufrufMitAdresse = regexp.MustCompile("\\(\\s*['\"`](/api/[^'\"`?$]+)")

// sammelversandPfade liefert sortiert die Adressen der Aufrufe, die FRIST_SAMMELVERSAND_MS als
// Frist tragen.
func sammelversandPfade(quellen map[string]string) []string {
	var pfade []string
	for _, quelle := range quellen {
		for _, stelle := range aufrufMitAdresse.FindAllStringSubmatchIndex(quelle, -1) {
			if strings.Contains(quelle[stelle[0]:aufrufEnde(quelle, stelle[0])], "FRIST_SAMMELVERSAND_MS") {
				pfade = append(pfade, quelle[stelle[2]:stelle[3]])
			}
		}
	}
	sort.Strings(pfade)
	return slices.Compact(pfade)
}
