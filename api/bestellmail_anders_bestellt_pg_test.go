package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"bibliothek/mailservice"
)

// Der Vermerk „Mail nicht versendet" ging nur weg, wenn die Mail doch noch rausging oder der
// Händler bestätigte. Eine Bestellung ohne Bestätigungs-Link, die den Händler am Telefon
// erreicht hat, behielt ihn; „Erneut senden" hätte sie ein zweites Mal zu ihm gebracht.
// DELETE /api/bestellungen/{id}/mail nimmt den Vermerk, ohne zu senden.

const auditBestellmailVermerkEntferntTest = "BESTELLMAIL_VERMERK_ENTFERNT"

// andersBestelltLage ist die Lage der Bestellmail-Tests, die den Eintrag der neuen Tür abräumt.
func andersBestelltLage(t *testing.T, kennung string) *bestellmailLage {
	t.Helper()
	l := bestellmailLageAnlegen(t, kennung)
	t.Cleanup(func() {
		aufraeumen(t, l.pool, `DELETE FROM audit_logs WHERE aktion = $1`, auditBestellmailVermerkEntferntTest)
	})
	return l
}

// mitBestaetigung liest über die Detail-Route, ob die Oberfläche für diese Bestellung den
// Bestätigungsblock zeigt.
func (l *bestellmailLage) mitBestaetigung(t *testing.T, bestellungID string) bool {
	t.Helper()
	rec := l.rufe(t, http.MethodGet, "/api/bestellhistorie/"+bestellungID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("Detail: Status %d: %s", rec.Code, rec.Body.String())
	}
	var kopf struct {
		MitBestaetigung bool `json:"mit_bestaetigung"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &kopf); err != nil {
		t.Fatalf("Detail unlesbar: %v", err)
	}
	return kopf.MitBestaetigung
}

func TestBestellmailAndersBestellt_NimmtDenVermerkOhneZuSenden(t *testing.T) {
	l := andersBestelltLage(t, "anders")
	lieferant := haendler(t, l.pool, "Buchhandlung-Anders", false)
	titel := titelMitMeldebestand(t, l.pool, "LMF-Mathe-Anders", 0)

	mailserverNichtErreichbar(t, nil)
	bestellung, _ := l.bestelle(t, lieferant, titel)
	if !l.gescheitert(t, bestellung) {
		t.Fatal("Die Bestellung trägt keinen Vermerk, obwohl die Mail nicht rausging")
	}
	if l.mitBestaetigung(t, bestellung) {
		t.Fatal("Die Bestellung bei einem gewöhnlichen Händler gilt als Bestellung mit Bestätigungs-Link")
	}

	// Ab hier ist ein Mailserver erreichbar: Ginge eine Mail raus, käme sie hier an.
	sitzungen := mailAbfangen(t)
	rec := l.rufe(t, http.MethodDelete, "/api/bestellungen/"+bestellung+"/mail", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("Vermerk entfernen: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	if l.gescheitert(t, bestellung) {
		t.Error("Die Bestellung trägt den Vermerk weiter")
	}
	if l.inListeGescheitert(t, bestellung) {
		t.Error("Die Liste nennt weiter einen gescheiterten Versand")
	}

	// Das Protokoll nennt Bearbeiter und Bestellung, keine Adresse.
	var wer, protokollierte string
	var details []byte
	if err := l.pool.QueryRow(t.Context(), `
		SELECT coalesce(admin_id::text, ''), details->>'bestellung_id', details
		FROM audit_logs WHERE aktion = $1`, auditBestellmailVermerkEntferntTest).Scan(&wer, &protokollierte, &details); err != nil {
		t.Fatalf("Protokolleintrag lesen (genau einer erwartet): %v", err)
	}
	if wer != l.adminID || protokollierte != bestellung {
		t.Errorf("Eintrag nennt Bearbeiter %q und Bestellung %q", wer, protokollierte)
	}
	if strings.Contains(string(details), "@") {
		t.Errorf("Der Eintrag trägt eine Adresse: %s", details)
	}

	// Ein zweiter Klick und „Erneut senden" finden keinen Vermerk mehr.
	if rec := l.rufe(t, http.MethodDelete, "/api/bestellungen/"+bestellung+"/mail", ""); rec.Code != http.StatusConflict {
		t.Errorf("Zweites Entfernen: Status %d, erwartet 409: %s", rec.Code, rec.Body.String())
	}
	if rec := l.rufe(t, http.MethodPost, "/api/bestellungen/"+bestellung+"/mail", ""); rec.Code != http.StatusConflict {
		t.Errorf("Erneut senden nach dem Entfernen: Status %d, erwartet 409: %s", rec.Code, rec.Body.String())
	}
	var eintraege int
	if err := l.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM audit_logs WHERE aktion = $1`, auditBestellmailVermerkEntferntTest).Scan(&eintraege); err != nil {
		t.Fatal(err)
	}
	if eintraege != 1 {
		t.Errorf("%d Protokolleinträge, erwartet einer: Das abgewiesene zweite Entfernen schreibt keinen", eintraege)
	}
	select {
	case mail := <-sitzungen:
		t.Errorf("Eine Mail ging raus, an %v", mail.Empfaenger)
	case <-time.After(300 * time.Millisecond):
	}
}

// Ohne eingerichteten Mailserver trägt jede Bestellung den Vermerk, und „Erneut senden" kann
// ihn nicht nehmen. Die neue Tür nimmt ihn.
func TestBestellmailAndersBestellt_OhneMailserver(t *testing.T) {
	l := andersBestelltLage(t, "ohneserver")
	alterLader := smtpKonfigLader
	smtpKonfigLader = func() (mailservice.SMTPKonfig, error) { return mailservice.SMTPKonfig{}, nil }
	t.Cleanup(func() { smtpKonfigLader = alterLader })

	lieferant := haendler(t, l.pool, "Buchhandlung-OhneServer", false)
	bestellung, _ := l.bestelle(t, lieferant, titelMitMeldebestand(t, l.pool, "LMF-Mathe-OhneServer", 0))
	if rec := l.rufe(t, http.MethodPost, "/api/bestellungen/"+bestellung+"/mail", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("Erneut senden ohne Mailserver: Status %d, erwartet 400", rec.Code)
	}
	if !l.gescheitert(t, bestellung) {
		t.Fatal("Die Bestellung trägt keinen Vermerk, obwohl kein Mailserver eingerichtet ist")
	}
	if rec := l.rufe(t, http.MethodDelete, "/api/bestellungen/"+bestellung+"/mail", ""); rec.Code != http.StatusOK {
		t.Fatalf("Vermerk entfernen: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	if l.gescheitert(t, bestellung) {
		t.Error("Die Bestellung trägt den Vermerk weiter")
	}
}

// Eine Bestellung mit Bestätigungs-Link behält den Vermerk: Für sie zeigt die Oberfläche den
// Bestätigungsblock, und der nennte nach dem Entfernen den Link als mit der Mail verschickt.
// Ihren Vermerk nimmt die nachgetragene Zusage.
func TestBestellmailAndersBestellt_NichtMitBestaetigungsLink(t *testing.T) {
	l := andersBestelltLage(t, "mitlink")
	lieferant := haendler(t, l.pool, "Naacher-MitLink", true)
	titel := titelMitMeldebestand(t, l.pool, "LMF-Mathe-MitLink", 0)

	mailserverNichtErreichbar(t, nil)
	bestellung, _ := l.bestelle(t, lieferant, titel)
	if !l.gescheitert(t, bestellung) {
		t.Fatal("Die Bestellung trägt keinen Vermerk, obwohl die Mail nicht rausging")
	}
	if !l.mitBestaetigung(t, bestellung) {
		t.Fatal("Die Bestellung beim Hauptlieferanten gilt nicht als Bestellung mit Bestätigungs-Link")
	}

	rec := l.rufe(t, http.MethodDelete, "/api/bestellungen/"+bestellung+"/mail", "")
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "nachtragen") {
		t.Errorf("Entfernen bei einer Bestellung mit Link: Status %d, erwartet 409 mit dem Weg über das Nachtragen: %s",
			rec.Code, rec.Body.String())
	}
	if !l.gescheitert(t, bestellung) {
		t.Error("Der Vermerk ist weg, obwohl die Tür abgewiesen hat")
	}
	var eintraege int
	if err := l.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM audit_logs WHERE aktion = $1`, auditBestellmailVermerkEntferntTest).Scan(&eintraege); err != nil {
		t.Fatal(err)
	}
	if eintraege != 0 {
		t.Errorf("%d Protokolleinträge, obwohl nichts entfernt wurde", eintraege)
	}
}

// Wo kein Vermerk steht, gibt es nichts zu entfernen; eine unbekannte Bestellung ist 404.
func TestBestellmailAndersBestellt_Abweisungen(t *testing.T) {
	l := andersBestelltLage(t, "abweisung")
	sitzungen := mailAbfangen(t)
	lieferant := haendler(t, l.pool, "Buchhandlung-Abweisung", false)
	bestellung, _ := l.bestelle(t, lieferant, titelMitMeldebestand(t, l.pool, "LMF-Mathe-Abweisung", 0))
	warteAufMail(t, sitzungen)

	if rec := l.rufe(t, http.MethodDelete, "/api/bestellungen/"+bestellung+"/mail", ""); rec.Code != http.StatusConflict {
		t.Errorf("Entfernen nach gelungenem Versand: Status %d, erwartet 409: %s", rec.Code, rec.Body.String())
	}
	if rec := l.rufe(t, http.MethodDelete, "/api/bestellungen/00000000-0000-4000-8000-000000000000/mail", ""); rec.Code != http.StatusNotFound {
		t.Errorf("Unbekannte Bestellung: Status %d, erwartet 404", rec.Code)
	}
	if rec := l.rufe(t, http.MethodDelete, "/api/bestellungen/keine-kennung/mail", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("Keine Kennung: Status %d, erwartet 400", rec.Code)
	}
}
