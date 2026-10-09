package api

import (
	"context"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/internal/auskunft"
	"bibliothek/internal/pdftest"
	"bibliothek/pdf"
	"bibliothek/repository"
)

// Das Blatt der Auskunft am echten Weg: Die Einträge entstehen über die Schreiber des Programms,
// das Blatt aus der Abfrage der Auskunft. Die zwei Protokoll-Abschnitte nennen jeden Vorgang in
// Worten, bei Ausleihe und Rückgabe Titel und Nummer des Buchs, und weder eine Kennung des
// Programms noch das Konto, das gebucht hat.
//
// Der Test sieht, was die Ratsche der Schlüssel (repository,
// TestProtokollSchluessel_FuerDieAuskunftEingeordnet) nicht sieht: die Einträge an der
// Leserzeile selbst und einen Schlüssel oder Vorgang, den das Blatt mit seinem Namen druckt.
//
// Blind für: Schreiber, die hier nicht aufgerufen werden — das Zusammenführen und den Bescheid
// (ihre Angaben prüft TestDsgvoAngabenZeilen an der Form, die die Schreiber ablegen).
func TestDsgvoBlatt_ProtokollzeilenInWorten(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	sid := seedSchueler(t, pool, "S-BLATT-1", "Blattkind", "7a")
	bearbeiter := seedPortalLehrkraft(t, pool, "blatt-bearbeiter@test.invalid")
	titelID := seedMonitorTitel(t, pool, "Blatt-Titel Mathematik", "Aut Or", false, 0)
	exID := exemplar(t, pool, titelID, "BLATT-EX-1", true, "")
	auditRepo := repository.NewAuditRepository(pool)

	// Ausleihe und Rückgabe.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := auditRepo.LogAusleihe(ctx, tx, exID, sid, "", bearbeiter); err != nil {
		t.Fatalf("LogAusleihe: %v", err)
	}
	if err := auditRepo.LogRueckgabe(ctx, tx, exID, sid, "", bearbeiter); err != nil {
		t.Fatalf("LogRueckgabe: %v", err)
	}
	// Die Spur einer Vormerkung, deren Titel gelöscht wurde.
	if err := repository.ProtokolliereWartendeBezuege(ctx, tx, []repository.WartenderBezug{{
		Tabelle: "vormerkungen", TitelID: titelID, Titel: "Blatt-Titel Vorgemerkt", Wer: "Blattkind Test",
		Status: "wartend", Seit: "2026-09-17", SchuelerD: &sid, Kontext: "Titel gelöscht, eine Vormerkung stand noch offen",
	}}); err != nil {
		t.Fatalf("ProtokolliereWartendeBezuege: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	// Eine Forderung, von Hand storniert.
	var schadenID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, ist_bezahlt)
		VALUES ($1, $2, 'Blatt-Schaden', 24.50, false) RETURNING id`, exID, sid).Scan(&schadenID); err != nil {
		t.Fatalf("Forderung anlegen: %v", err)
	}
	if err := auditRepo.StornierungGebuehr(ctx, schadenID, bearbeiter, "Blatt-Grund der Stornierung"); err != nil {
		t.Fatalf("StornierungGebuehr: %v", err)
	}

	// Sperre gesetzt und aufgehoben, LUSD-ID nachgetragen.
	anfrage := httptest.NewRequest("POST", "/api/schueler/"+sid+"/sperre", nil)
	protokolliereSperre(anfrage, auditRepo, bearbeiter, sid, true, "Blatt-Sperrgrund", repository.LeserSperrStand{})
	protokolliereSperre(anfrage, auditRepo, bearbeiter, sid, false, "", repository.LeserSperrStand{VonHand: true, Grund: "Blatt-Sperrgrund"})
	if err := auditRepo.LogAdminAktion(ctx, bearbeiter, "LUSD_ID_NACHGETRAGEN", "127.0.0.1", map[string]any{
		"schueler_id": sid, "lusd_id": "BLATT-LUSD-4711",
	}); err != nil {
		t.Fatalf("LogAdminAktion: %v", err)
	}

	// Der Löscheintrag eines Zugangskontos steht bei den früheren Konten, nicht im Protokoll.
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, details)
		VALUES ('benutzer', 'DELETE', gen_random_uuid(), 'USER',
		        jsonb_build_object('schueler_id', $1::text, 'vorname', 'Blattkonto', 'nachname', 'Test',
		                           'email', 'blattkonto@test.invalid', 'rolle', 'kollegium'))`, sid); err != nil {
		t.Fatalf("Löscheintrag eines Kontos: %v", err)
	}

	// In den Papierkorb: der Eintrag an der Leserzeile selbst.
	if err := auditRepo.DeleteStudent(ctx, sid, bearbeiter, "Blatt-Löschgrund"); err != nil {
		t.Fatalf("DeleteStudent: %v", err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	daten, err := srv.sammleDsgvoDaten(ctx, sid, true)
	if err != nil {
		t.Fatalf("sammleDsgvoDaten: %v", err)
	}
	roh, err := auskunft.GenerateDsgvoAuskunftPDF(dsgvoAntwort(daten, time.Now()), pdf.SchuleInfo{Name: "Testschule"})
	if err != nil {
		t.Fatalf("GenerateDsgvoAuskunftPDF: %v", err)
	}
	// Mit Leerzeichen verbunden: Eine lange Zeile bricht das Blatt um, die Angabe steht dann in
	// zwei Textstücken.
	blatt := strings.Join(pdftest.TexteInReihenfolge(t, roh), " ")
	_, ab8, gefunden := strings.Cut(blatt, "8. Protokolleinträge zu diesem Datensatz")
	protokoll, _, gefunden10 := strings.Cut(ab8, "10. Zugangskonto")
	if !gefunden || !gefunden10 {
		t.Fatalf("die Abschnitte 8 bis 10 stehen nicht auf dem Blatt:\n%s", blatt)
	}

	for _, soll := range []string{
		"Ausleihe", "Rückgabe", "Blatt-Titel Mathematik (Nummer BLATT-EX-1)",
		"Forderung storniert", "Betrag: 24,50 EUR", "Grund: Blatt-Grund der Stornierung",
		"Titel gelöscht, während eine Vormerkung offen war (automatisch)", "Titel: Blatt-Titel Vorgemerkt", "Name: Blattkind Test",
		"Leserdatensatz in den Papierkorb gelegt", "Ausweisnummer: S-BLATT-1", "Grund: Blatt-Löschgrund",
		"Von Hand gesperrt", "Sperrgrund: Blatt-Sperrgrund",
		"Sperre aufgehoben", "Grund der aufgehobenen Sperre: Blatt-Sperrgrund", "Sperre war von Hand gesetzt: Ja",
		"LUSD-ID nachgetragen", "LUSD-ID: BLATT-LUSD-4711",
	} {
		if !strings.Contains(protokoll, soll) {
			t.Errorf("in den Abschnitten 8 und 9 fehlt %q", soll)
		}
	}

	// Nichts in der Schreibweise des Programms: keine Aktion in Großbuchstaben, kein Schlüssel
	// mit Unterstrich, keine Kennung. DSGVO, LUSD und EUR sind Wörter des Blatts.
	grossbuchstaben := regexp.MustCompile(`\b[A-ZÄÖÜ][A-ZÄÖÜ_]{3,}\b`)
	for _, fund := range grossbuchstaben.FindAllString(protokoll, -1) {
		if fund != "DSGVO" && fund != "LUSD" && fund != "BLATT" {
			t.Errorf("in den Abschnitten 8 und 9 steht %q in der Schreibweise des Programms", fund)
		}
	}
	if fund := regexp.MustCompile(`\b[a-z]+_[a-z_]+\b`).FindString(protokoll); fund != "" {
		t.Errorf("in den Abschnitten 8 und 9 steht der Schlüssel %q", fund)
	}
	if fund := regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).FindString(protokoll); fund != "" {
		t.Errorf("in den Abschnitten 8 und 9 steht die Kennung %s", fund)
	}
	// Das Konto, das gebucht hat (seedPortalLehrkraft: Portal Lehrkraft), und der Löscheintrag
	// eines Kontos, den Abschnitt 10 nennt.
	for _, nicht := range []string{"blatt-bearbeiter@test.invalid", "Portal", "Blattkonto", "127.0.0.1"} {
		if strings.Contains(protokoll, nicht) {
			t.Errorf("in den Abschnitten 8 und 9 steht %q", nicht)
		}
	}
	if !strings.Contains(blatt, "blattkonto@test.invalid") {
		t.Error("das frühere Zugangskonto steht nicht auf dem Blatt (Abschnitt 10)")
	}
}
