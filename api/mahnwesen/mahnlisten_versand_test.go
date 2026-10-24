package mahnwesen

import (
	"bibliothek/mailservice"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/repository"

	"github.com/google/go-cmp/cmp"
	"github.com/pashagolub/pgxmock/v5"
)

// Ohne eingerichteten Mailversand → 503, kein Versand. Der Check
// greift vor jeder Klassen-Query — pgxmock erwartet keine einzige.
func TestMahnlistenVersenden_SMTPFehlt(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	server := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}
	mahnRepo := repository.NewMahnwesenAbfragen(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/mail/send-bulk-overdue", strings.NewReader(`{"klassen":["5a"]}`))
	rec := httptest.NewRecorder()
	server.MahnlistenVersenden(mahnRepo)(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("rec.Code = %d, erwartet 503 ohne SMTP: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unerfüllte Mock-Erwartungen: %v", err)
	}
}

// Kern der Datenschutz-Garantie: Klassen ohne Lehrer-E-Mail oder ohne Schüler
// werden übersprungen und erhalten keine Mail; gültige Klassen bekommen genau eine
// Mail an genau die hinterlegte Lehrer-Adresse (keine klassenübergreifenden Empfänger).
func TestVersendeKlassenMahnungen_SkipLogik(t *testing.T) {
	var empfaenger []string
	fakeSend := func(_ context.Context, m mailservice.Nachricht) error {
		empfaenger = append(empfaenger, m.An)
		return nil
	}
	fakePDF := func(_ []repository.MahnwesenKlasse) ([]byte, error) {
		return []byte("%PDF-fake"), nil
	}

	klassen := []repository.MahnwesenKlasse{
		{
			Klasse:      "5a",
			LehrerEmail: "lehrer5a@schule.de",
			Schueler: []repository.UeberfaelligerSchueler{
				{SchuelerID: "s1", Name: "Max", Klasse: "5a", Medien: []repository.UeberfaelligesMedium{{Titel: "Buch", TageUeberfaellig: 5}}},
			},
		},
		{ // keine Lehrer-Mail → übersprungen, darf nicht gesendet werden
			Klasse:      "6b",
			LehrerEmail: "",
			Schueler: []repository.UeberfaelligerSchueler{
				{SchuelerID: "s2", Name: "Erika", Klasse: "6b", Medien: []repository.UeberfaelligesMedium{{Titel: "Buch2"}}},
			},
		},
		{ // keine Schüler → übersprungen
			Klasse:      "7c",
			LehrerEmail: "lehrer7c@schule.de",
			Schueler:    nil,
		},
	}

	erg := versendeKlassenMahnungen(t.Context(), klassen, fakePDF, fakeSend)

	if erg.Versendet != 1 || erg.Uebersprungen != 2 || erg.Fehlgeschlagen != 0 {
		t.Fatalf("sent=%d skipped=%d failed=%d, erwartet sent=1 skipped=2 failed=0", erg.Versendet, erg.Uebersprungen, erg.Fehlgeschlagen)
	}
	if len(empfaenger) != 1 || empfaenger[0] != "lehrer5a@schule.de" {
		t.Fatalf("Empfänger = %v, erwartet genau [lehrer5a@schule.de] — keine Mail an Klassen ohne Adresse!", empfaenger)
	}
}

// Schlägt der Versand einer Klasse fehl (abgelehnte Adresse o. ä.), läuft der Rest
// weiter (best effort), aber die Klasse zählt als fehlgeschlagen, nicht als übersprungen:
// „übersprungen (keine E-Mail hinterlegt)" ist Absicht, ein Ausfall nicht.
func TestVersendeKlassenMahnungen_VersandfehlerZaehltAlsFehlgeschlagen(t *testing.T) {
	fakePDF := func(_ []repository.MahnwesenKlasse) ([]byte, error) {
		return []byte("%PDF-fake"), nil
	}
	fakeSend := func(_ context.Context, m mailservice.Nachricht) error {
		if m.An == "kaputt@schule.de" {
			return http.ErrHandlerTimeout // beliebiger Versandfehler
		}
		return nil
	}

	schueler := []repository.UeberfaelligerSchueler{
		{SchuelerID: "s1", Name: "Max", Klasse: "x", Medien: []repository.UeberfaelligesMedium{{Titel: "Buch"}}},
	}
	klassen := []repository.MahnwesenKlasse{
		{Klasse: "ok", LehrerEmail: "ok@schule.de", Schueler: schueler},
		{Klasse: "err", LehrerEmail: "kaputt@schule.de", Schueler: schueler},
	}

	erg := versendeKlassenMahnungen(t.Context(), klassen, fakePDF, fakeSend)
	if erg.Versendet != 1 || erg.Uebersprungen != 0 || erg.Fehlgeschlagen != 1 || erg.Abgebrochen {
		t.Fatalf("%+v, erwartet sent=1 skipped=0 failed=1 nicht abgebrochen (Einzelfehler = fehlgeschlagen, Lauf läuft weiter)", erg)
	}
}

// Mailserver-Fehler (ErrSMTPVersand) bricht den Lauf ab: Das Relay ist weg, jede weitere
// Klasse hinge nur bis zur Frist. Die übrigen zählen als fehlgeschlagen, und die Meldung sagt
// es; unter „übersprungen (keine E-Mail hinterlegt oder keine Fälle)" sähe der Ausfall aus wie
// Absicht.
func TestVersendeKlassenMahnungen_MailserverFehlerBrichtAbUndMeldet(t *testing.T) {
	fakePDF := func(_ []repository.MahnwesenKlasse) ([]byte, error) { return []byte("%PDF-fake"), nil }
	var versuche []string
	fakeSend := func(_ context.Context, m mailservice.Nachricht) error {
		versuche = append(versuche, m.An)
		if m.An == "zweite@schule.de" {
			return fmt.Errorf("%w (Server relay:587): connection refused", mailservice.ErrSMTPVersand)
		}
		return nil
	}
	schueler := []repository.UeberfaelligerSchueler{
		{SchuelerID: "s1", Name: "Max", Klasse: "x", Medien: []repository.UeberfaelligesMedium{{Titel: "Buch"}}},
	}
	klassen := []repository.MahnwesenKlasse{
		{Klasse: "5a", LehrerEmail: "erste@schule.de", Schueler: schueler},
		{Klasse: "5b", LehrerEmail: "zweite@schule.de", Schueler: schueler},
		{Klasse: "5c", LehrerEmail: "dritte@schule.de", Schueler: schueler},
		{Klasse: "5d", LehrerEmail: "", Schueler: schueler},
	}
	erg := versendeKlassenMahnungen(t.Context(), klassen, fakePDF, fakeSend)
	if erg.Versendet != 1 || erg.Fehlgeschlagen != 2 || erg.Uebersprungen != 1 || !erg.Abgebrochen {
		t.Fatalf("%+v, erwartet sent=1 failed=2 (5b + nicht versuchte 5c) skipped=1 abgebrochen", erg)
	}
	if len(versuche) != 2 {
		t.Fatalf("nach dem Mailserver-Fehler darf keine weitere Klasse versucht werden: %v", versuche)
	}
	msg := mahnlistenMeldung(erg, "")
	if !strings.Contains(msg, "Für 2 Klassen nicht zugestellt") || !strings.Contains(msg, "Der Lauf brach nach einem Fehler des Mailservers ab") {
		t.Fatalf("Meldung nennt den Ausfall nicht: %q", msg)
	}
}

// ── Auswahl und Umleitung ────────────────────────────────────────────────────

// argFaenger reicht ein Exec-Argument durch und schreibt es mit. pgxmock.AnyArg()
// würde die Angaben zwar passieren lassen, aber nicht herausgeben — und geprüft
// werden soll genau sein Inhalt.
type argFaenger struct{ ziel *string }

func (a *argFaenger) Match(v any) bool {
	s, ok := v.(string)
	if ok {
		*a.ziel = s
	}
	return ok
}

// Ohne genannte Klasse gibt es keinen Lauf: Ein fehlendes klassen-Feld heißt wie ein leeres
// Array „niemand" und wird abgewiesen. Läse die Tür eines davon als „alle", löste ein Rumpf ohne
// Auswahl den größten denkbaren Fehlversand aus.
func TestLeseKlassenVersandAnfrage(t *testing.T) {
	t.Setenv("SELBSTANMELDUNG_DOMAIN", "")
	tests := []struct {
		name        string
		body        string
		wantErr     bool
		wantKlassen []string
		wantEmail   string
	}{
		{name: "kein Body → Fehler", body: "", wantErr: true},
		{name: "leeres Objekt → Fehler", body: `{}`, wantErr: true},
		{name: "unbekanntes Feld → Fehler", body: `{"klasse":["5a"]}`, wantErr: true},
		{name: "leeres Array → Fehler", body: `{"klassen":[]}`, wantErr: true},
		{name: "nur Leerzeichen-Klassen → Fehler", body: `{"klassen":["  ",""]}`, wantErr: true},
		{name: "kaputtes JSON → Fehler", body: `{"klassen":`, wantErr: true},
		{
			name: "Auswahl wird getrimmt", body: `{"klassen":[" 5a ","6b"]}`,
			wantKlassen: []string{"5a", "6b"},
		},
		{
			name: "Override wird getrimmt", body: `{"klassen":["5a"],"override_email":"  sek@schule.de "}`,
			wantKlassen: []string{"5a"}, wantEmail: "sek@schule.de",
		},
		{name: "ungültige Override-Adresse → Fehler", body: `{"klassen":["5a"],"override_email":"sek@"}`, wantErr: true},
		{name: "Header-Injektion in der Adresse → Fehler", body: `{"klassen":["5a"],"override_email":"a@b.de\nBcc: c@d.de"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/mail/send-bulk-overdue", body)
			rec := httptest.NewRecorder()

			got, ok := leseKlassenVersandAnfrage(rec, req)
			if tt.wantErr {
				if ok || rec.Code != http.StatusBadRequest {
					t.Fatalf("leseKlassenVersandAnfrage(…) = %+v, Status %d, erwartet 400", got, rec.Code)
				}
				return
			}
			if !ok {
				t.Fatalf("unerwartet abgewiesen: %s", rec.Body.String())
			}
			if !cmp.Equal(got.Klassen, tt.wantKlassen) {
				t.Fatalf("Klassen = %v, erwartet %v", got.Klassen, tt.wantKlassen)
			}
			if got.Umleitung != tt.wantEmail {
				t.Fatalf("Umleitung = %q, erwartet %q", got.Umleitung, tt.wantEmail)
			}
		})
	}
}

// waehleKlassen darf nur die angehakten Klassen durchlassen — und muss melden, was
// es nicht (mehr) gibt, damit ein Format-Versatz („5a" vs. „05A") nicht als stiller
// Nullversand endet.
func TestWaehleKlassen(t *testing.T) {
	alle := []repository.MahnwesenKlasse{
		{Klasse: "5a", LehrerEmail: "a@schule.de"},
		{Klasse: "6b", LehrerEmail: "b@schule.de"},
		{Klasse: "7c", LehrerEmail: "c@schule.de"},
	}

	t.Run("ohne Auswahl keine Klasse", func(t *testing.T) {
		gewaehlt, unbekannt := waehleKlassen(alle, nil)
		if len(gewaehlt) != 0 || unbekannt != nil {
			t.Fatalf("gewaehlt=%d unbekannt=%v, erwartet 0 und nil", len(gewaehlt), unbekannt)
		}
	})

	t.Run("schneidet auf die Auswahl zu", func(t *testing.T) {
		auswahl := []string{"6b"}
		gewaehlt, unbekannt := waehleKlassen(alle, auswahl)
		if len(gewaehlt) != 1 || gewaehlt[0].Klasse != "6b" {
			t.Fatalf("gewaehlt = %+v, erwartet genau [6b]", gewaehlt)
		}
		if unbekannt != nil {
			t.Fatalf("unbekannt = %v, erwartet nil", unbekannt)
		}
	})

	t.Run("meldet unbekannte Klassen, statt sie zu verschlucken", func(t *testing.T) {
		auswahl := []string{"5a", "05A"}
		gewaehlt, unbekannt := waehleKlassen(alle, auswahl)
		if len(gewaehlt) != 1 || gewaehlt[0].Klasse != "5a" {
			t.Fatalf("gewaehlt = %+v, erwartet genau [5a]", gewaehlt)
		}
		if !cmp.Equal(unbekannt, []string{"05A"}) {
			t.Fatalf("unbekannt = %v, erwartet [05A]", unbekannt)
		}
	})

	t.Run("doppelte Auswahl mahnt nicht doppelt", func(t *testing.T) {
		auswahl := []string{"5a", "5a", "5a"}
		gewaehlt, _ := waehleKlassen(alle, auswahl)
		if len(gewaehlt) != 1 {
			t.Fatalf("gewaehlt = %d Klassen, erwartet 1 — sonst bekommt die Klasse drei Mahnungen", len(gewaehlt))
		}
	})
}

// Mit Umleitung gehen die Listen an genau diese eine Adresse — auch für
// Klassen ohne hinterlegte Klassenleitung (das ist der Vertretungsfall, für den
// das Feld existiert). Klassen ohne überfällige Fälle bleiben trotzdem aussen vor.
func TestLenkeKlassenUm_LenktAlleKlassenUm(t *testing.T) {
	schueler := []repository.UeberfaelligerSchueler{
		{SchuelerID: "s1", Name: "Max", Klasse: "x", Medien: []repository.UeberfaelligesMedium{{Titel: "Buch"}}},
	}
	original := []repository.MahnwesenKlasse{
		{Klasse: "5a", LehrerEmail: "lehrer5a@schule.de", Schueler: schueler},
		{Klasse: "6b", LehrerEmail: "", Schueler: schueler}, // ohne Klassenleitung, sonst übersprungen
		{Klasse: "7c", LehrerEmail: "lehrer7c@schule.de", Schueler: nil},
	}

	umgelenkt := lenkeKlassenUm(original, "sekretariat@schule.de")

	var empfaenger []string
	erg := versendeKlassenMahnungen(t.Context(),
		umgelenkt,
		func(_ []repository.MahnwesenKlasse) ([]byte, error) { return []byte("%PDF-fake"), nil },
		func(_ context.Context, m mailservice.Nachricht) error {
			empfaenger = append(empfaenger, m.An)
			return nil
		},
	)

	if erg.Versendet != 2 || erg.Uebersprungen != 1 {
		t.Fatalf("sent=%d skipped=%d, erwartet sent=2 skipped=1 (7c hat keine Fälle)", erg.Versendet, erg.Uebersprungen)
	}
	for _, e := range empfaenger {
		if e != "sekretariat@schule.de" {
			t.Fatalf("Empfänger = %v, erwartet ausschliesslich sekretariat@schule.de", empfaenger)
		}
	}
	if original[0].LehrerEmail != "lehrer5a@schule.de" {
		t.Fatalf("Original wurde mutiert (LehrerEmail = %q) — die geladenen Daten müssen unangetastet bleiben", original[0].LehrerEmail)
	}
}

// Ohne gültige Auswahl läuft kein Massenversand an, an keiner der beiden Türen: Die Tür weist
// den Rumpf ab, bevor sie eine Klasse lädt. Versand eingerichtet und Saison offen, damit keine
// andere Sperre den Test grün macht.
func TestKlassenVersand_OhneAuswahlKeinLauf(t *testing.T) {
	t.Setenv("SELBSTANMELDUNG_DOMAIN", "")
	faelle := []struct {
		pfad string
		tuer func(*Tueren, *repository.MahnwesenAbfragen) http.HandlerFunc
	}{
		{"/api/mail/send-bulk-overdue", (*Tueren).MahnlistenVersenden},
		{"/api/abgaenger/mail", func(s *Tueren, _ *repository.MahnwesenAbfragen) http.HandlerFunc {
			return s.AbgaengerKontoauszuegeVersenden()
		}},
	}
	ruempfe := []string{
		``,
		`{}`,
		`null`,
		`{"klassen":null}`,
		`{"klassen":[]}`,
		`{"klassen":["  ",""]}`,
		`{"klasse":["5a"]}`,
		`{"klassen":["5a"],"alle":true}`,
		`{"klassen":["5a"],"override_email":"kein-mail"}`,
	}
	for _, fall := range faelle {
		for _, rumpf := range ruempfe {
			t.Run(fall.pfad+" "+rumpf, func(t *testing.T) {
				mock, err := pgxmock.NewPool()
				if err != nil {
					t.Fatalf("pgxmock: %v", err)
				}
				defer mock.Close()
				post, sitzungen := tuertest.FangeMailAb(t)
				server := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}, Mail: post, Uhr: testdaten.SaisonUhr}}

				rec := httptest.NewRecorder()
				fall.tuer(server, repository.NewMahnwesenAbfragen(mock))(rec,
					httptest.NewRequest(http.MethodPost, fall.pfad, strings.NewReader(rumpf)))

				if rec.Code != http.StatusBadRequest {
					t.Fatalf("Status = %d, erwartet 400: %s", rec.Code, rec.Body.String())
				}
				select {
				case <-sitzungen:
					t.Error("eine Mail ging hinaus")
				default:
				}
			})
		}
	}
}

// Revisionssicherheit: Der Eintrag im Protokoll muss die Umleitung im Klartext
// führen und gültiges JSON sein — auch wenn die Adresse Zeichen enthält, die ein
// handgebautes JSON-Literal zerrissen hätten (früher: fmt.Sprintf).
func TestProtokolliereKlassenVersand_SchreibtGueltigesJSONMitEmpfaenger(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	var details string
	mock.ExpectExec(`INSERT INTO audit_logs`).
		WithArgs("admin-1", "BULK_OVERDUE_MAIL", &argFaenger{ziel: &details}, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	server := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}
	req := httptest.NewRequest(http.MethodPost, "/api/mail/send-bulk-overdue", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel, &auth.Claims{KontoID: "admin-1"}))

	server.protokolliereKlassenVersand(req, "BULK_OVERDUE_MAIL", klassenVersandProtokoll{
		Phase:         "ende",
		Klassen:       []string{"5a", "6b"},
		Umleitung:     `"seltsam"@schule.de`,
		Versendet:     2,
		Uebersprungen: 0,
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Audit-INSERT fehlt: %v", err)
	}

	// Die geschriebenen Angaben müssen als JSON lesbar sein (Spalte ist jsonb).
	var geparst klassenVersandProtokoll
	if err := json.Unmarshal([]byte(details), &geparst); err != nil {
		t.Fatalf("details ist kein gültiges JSON: %v (%s)", err, details)
	}
	if geparst.Umleitung != `"seltsam"@schule.de` {
		t.Fatalf("Umleitung = %q — die Adresse muss unverfälscht im Log stehen", geparst.Umleitung)
	}
}

// Alle Dienstadressen der Schule liegen auf derselben Domain. Sie jedes Mal
// mitzutippen ist Fehlerquelle — „mueller" muss reichen. Die Domain kommt aus ihrer einen
// Quelle, SELBSTANMELDUNG_DOMAIN. Vollständige Adressen bleiben unangetastet, damit der
// Versand an externe Stellen möglich bleibt.
func TestErgaenzeSchulDomain(t *testing.T) {
	t.Setenv("SELBSTANMELDUNG_DOMAIN", "schule.example")

	tests := []struct{ eingabe, want string }{
		{"mueller", "mueller@schule.example"},
		{"m.mueller", "m.mueller@schule.example"},
		{"extern@schulamt.hessen.de", "extern@schulamt.hessen.de"}, // vollständig → unverändert
		{"", ""},
	}
	for _, tt := range tests {
		if got := ergaenzeSchulDomain(tt.eingabe); got != tt.want {
			t.Errorf("ergaenzeSchulDomain(%q) = %q, erwartet %q", tt.eingabe, got, tt.want)
		}
	}
}

// Ohne eingetragene Domain wird nicht geraten: Die Eingabe läuft unverändert in
// die Adressprüfung und scheitert dort mit einer klaren Meldung.
func TestErgaenzeSchulDomain_OhneDomainKeineRaterei(t *testing.T) {
	t.Setenv("SELBSTANMELDUNG_DOMAIN", "")
	if got := ergaenzeSchulDomain("mueller"); got != "mueller" {
		t.Fatalf("ergaenzeSchulDomain = %q, erwartet unverändert %q", got, "mueller")
	}
}

// Der Weg über den echten Rumpf: „mueller" kommt als vollständige Adresse heraus.
func TestLeseKlassenVersandAnfrage_ErgaenztDomaene(t *testing.T) {
	t.Setenv("SELBSTANMELDUNG_DOMAIN", "schule.example")
	req := httptest.NewRequest(http.MethodPost, "/api/abgaenger/mail",
		strings.NewReader(`{"klassen":["9h"],"override_email":" mueller "}`))

	got, ok := leseKlassenVersandAnfrage(httptest.NewRecorder(), req)
	if !ok {
		t.Fatal("unerwartet abgewiesen")
	}
	if got.Umleitung != "mueller@schule.example" {
		t.Fatalf("Umleitung = %q, erwartet mueller@schule.example", got.Umleitung)
	}
}
