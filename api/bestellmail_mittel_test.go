package api

import (
	"strings"
	"testing"

	"bibliothek/repository"
)

// Der Topf steht in der Händler-Mail — auch dann, wenn die frei editierbare Vorlage den
// Platzhalter {{.Mittel}} nicht kennt (Bestand vor Migration 109, umformulierte Vorlagen).
// Dieselbe Fehlerklasse wie der Bestätigungs-Link: Ein Absatz, der aus der Vorlage fällt,
// und ein Ablauf, der still weiterläuft.

func TestBestellMailTraegtDenTopfAuchOhnePlatzhalter(t *testing.T) {
	subject, body := resolveBestellMail("Buchbestellung {{.Datum}}", "Sehr geehrte Damen und Herren,\n\nanbei die Bestellung.",
		"K-1", 2, 5, "", nil, repository.MittelSchultraeger)

	if !strings.HasSuffix(subject, "– Schülerbücherei") {
		t.Errorf("Betreff ohne Topf: %q", subject)
	}
	if !strings.Contains(body, "Anschaffung für die Schülerbücherei") {
		t.Errorf("Text ohne Vermerk:\n%s", body)
	}
	if !strings.Contains(body, "auf der Rechnung") {
		t.Errorf("Text bittet nicht um den Vermerk auf der Rechnung:\n%s", body)
	}
}

func TestBestellMailErsetztDenPlatzhalterUndHaengtDannNichtsAn(t *testing.T) {
	subject, body := resolveBestellMail("Bestellung {{.Mittel}} {{.Datum}}", "Diese Bestellung: {{.Mittel}}.",
		"K-1", 2, 5, "", nil, repository.MittelLand)

	if !strings.HasPrefix(subject, "Bestellung Lernmittelfreiheit ") {
		t.Errorf("Platzhalter im Betreff nicht ersetzt: %q", subject)
	}
	if strings.Contains(subject, "–") {
		t.Errorf("Betreff trägt den Topf doppelt: %q", subject)
	}
	if body != "Diese Bestellung: Lernmittelfreiheit." {
		t.Errorf("Text: %q — Platzhalter nicht ersetzt oder Vermerk trotzdem angehängt", body)
	}
}

// Die Werksvorgabe (Fallback, wenn die Vorlage fehlt) trägt den Platzhalter selbst —
// sonst käme jede Fallback-Mail mit dem angehängten Absatz statt mit dem Topf im Betreff.
func TestBestellMailFallbackKenntDenTopf(t *testing.T) {
	for _, vorlage := range []string{bestellMailFallbackBetreff, bestellMailFallbackBody} {
		if !strings.Contains(vorlage, "{{.Mittel}}") {
			t.Errorf("Fallback ohne {{.Mittel}}: %q", vorlage)
		}
	}
	subject, _ := resolveBestellMail(bestellMailFallbackBetreff, bestellMailFallbackBody, "K-1", 1, 1, "", nil, repository.MittelLand)
	if !strings.Contains(subject, "Lernmittelfreiheit") || strings.Contains(subject, "{{") {
		t.Errorf("Fallback-Betreff: %q", subject)
	}
}

// Die Reihenfolge der Absätze: erst der Vermerk, dann der Link — der Link ist die
// Handlung, der Vermerk die Einordnung; beides muss da sein, wenn die Vorlage keins von
// beiden platziert.
func TestBestellMailVermerkStehtVorDemLinkAbsatz(t *testing.T) {
	_, body := resolveBestellMail("Betreff", "anbei die Bestellung.", "K-1", 1, 1, "https://bib.example.invalid/bestellung/x", nil, repository.MittelLand)
	vermerk := strings.Index(body, "Lernmittelfreiheit")
	link := strings.Index(body, "https://bib.example.invalid/bestellung/x")
	if vermerk < 0 || link < 0 || vermerk > link {
		t.Errorf("Vermerk (%d) muss vor dem Link (%d) stehen:\n%s", vermerk, link, body)
	}
}
