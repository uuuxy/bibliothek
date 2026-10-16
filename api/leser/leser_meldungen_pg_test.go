package leser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/repository"
)

// Was Anlegen, Akte und Löschen eines Lesers ablehnen, sagt ein Satz, den das Sekretariat liest:
// welche Angabe fehlt oder nicht passt und warum. Der Test hält jeden dieser Sätze an der Antwort
// der echten Tür fest.
func TestLeserTueren_MeldungJeAblehnung(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	admin := &auth.Claims{KontoID: "00000000-0000-0000-0000-0000000000aa", Rolle: auth.RolleAdmin}

	rufe := func(t *testing.T, tuer http.HandlerFunc, methode, pfad, id, rumpf string, claims *auth.Claims) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(methode, pfad, strings.NewReader(rumpf))
		req.Header.Set("Content-Type", "application/json")
		if id != "" {
			req.SetPathValue("id", id)
		}
		if claims != nil {
			req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel, claims))
		}
		rec := httptest.NewRecorder()
		tuer(rec, req)
		return rec
	}
	anlegen := func(t *testing.T, rumpf string) *httptest.ResponseRecorder {
		t.Helper()
		return rufe(t, srv.LeserAnlegen(), http.MethodPost, "/api/schueler", "", rumpf, admin)
	}
	aendern := func(t *testing.T, id, rumpf string) *httptest.ResponseRecorder {
		t.Helper()
		return rufe(t, srv.LeserAendern(repository.NewProtokollAbfragen(pool)), http.MethodPatch, "/api/schueler/"+id, id, rumpf, admin)
	}
	erwarte := func(t *testing.T, rec *httptest.ResponseRecorder, status int, meldung string) {
		t.Helper()
		if rec.Code != status {
			t.Fatalf("Status %d, erwartet %d: %s", rec.Code, status, rec.Body.String())
		}
		if got := tuertest.Fehlermeldung(t, rec); got != meldung {
			t.Errorf("Meldung %q, erwartet %q", got, meldung)
		}
	}
	leserID := func(t *testing.T, nachname string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `SELECT id::text FROM leser WHERE nachname = $1`, nachname).Scan(&id); err != nil {
			t.Fatalf("Leserzeile %s lesen: %v", nachname, err)
		}
		return id
	}
	legeAn := func(t *testing.T, rumpf string) {
		t.Helper()
		if rec := anlegen(t, rumpf); rec.Code != http.StatusCreated {
			t.Fatalf("Anlegen: Status %d, erwartet 201: %s", rec.Code, rec.Body.String())
		}
	}

	const (
		meldungLehrerKlasse = "Lehrkräfte werden nicht als Schüler angelegt. Bitte unter System → Benutzer & Rechte ein Konto mit der Rolle Kollegium anlegen; an der Theke findet man die Lehrkraft dann über ihren Ausweis."
		meldungSchuelerMail = "Ein Schüler bekommt kein Konto und keine E-Mail-Adresse"
		moeglicheArten      = "Schüler, Lehrkraft, LiV, Praktikum, Sekretariat, U-plus, Fachbereich"
	)

	// Anlegen
	for _, f := range []struct{ name, rumpf, meldung string }{
		{"Lehrkraft ohne Schul-E-Mail", `{"vorname":"Lena","nachname":"Ohnemail","art":"lehrkraft"}`,
			"Die Schul-E-Mail-Adresse ist Pflicht: An ihr erkennt die Anmeldung die Person, und sie verhindert einen zweiten Eintrag, wenn sie sich später selbst anmeldet."},
		{"Lehrkraft mit Adresse ohne @", `{"vorname":"Lena","nachname":"Ohneat","art":"lehrkraft","email":"lena-ohne-at"}`,
			"„lena-ohne-at“ ist keine E-Mail-Adresse"},
		{"unbekannte Art", `{"vorname":"Zora","nachname":"Zauber","art":"zauberer"}`,
			`Unbekannte Art "zauberer". Möglich sind: ` + moeglicheArten + "."},
		{"Schüler ohne Klasse", `{"vorname":"Karl","nachname":"Ohneklasse","geburtsdatum":"2012-03-04"}`,
			"Klasse fehlt. Ein Schüler ohne Klasse fällt aus jeder Klassenliste und jeder Mahnung."},
		{"Lehrkraft mit Klasse", `{"vorname":"Lena","nachname":"Mitklasse","art":"lehrkraft","email":"lena.mitklasse@lesertest.invalid","klasse":"7a"}`,
			"Nur ein Schüler hat eine Klasse. Mit einer Klasse stünde die Person in den Klassenlisten und im LUSD-Abgleich."},
		{"Schüler mit Schul-E-Mail", `{"vorname":"Karl","nachname":"Mitmail","klasse":"7a","geburtsdatum":"2012-03-04","email":"karl@lesertest.invalid"}`,
			meldungSchuelerMail},
		{"Praktikum mit Schul-E-Mail", `{"vorname":"Paul","nachname":"Praktikant","art":"praktikum","email":"paul@lesertest.invalid"}`,
			"Praktikum und Fachbereich bekommen keinen Zugang zu „Mein Portal“ und deshalb keine Schul-E-Mail"},
		{"Schüler mit Klasse Lehrer", `{"vorname":"Karl","nachname":"Lehrerklasse","klasse":"Lehrer","geburtsdatum":"2012-03-04"}`,
			meldungLehrerKlasse},
		{"Schüler ohne Geburtsdatum", `{"vorname":"Karl","nachname":"Ohnedatum","klasse":"7a"}`,
			"Geburtsdatum fehlt. Es ist der einzige Schlüssel, über den der LUSD-Import diesen Schüler später wiedererkennt. Ohne Geburtsdatum würde er beim nächsten Import doppelt angelegt."},
	} {
		t.Run("Anlegen: "+f.name, func(t *testing.T) {
			erwarte(t, anlegen(t, f.rumpf), http.StatusBadRequest, f.meldung)
		})
	}
	t.Run("Anlegen: Schul-E-Mail außerhalb der freigegebenen Domain", func(t *testing.T) {
		t.Setenv("SELBSTANMELDUNG_DOMAIN", "lesertest.invalid")
		erwarte(t, anlegen(t, `{"vorname":"Lena","nachname":"Fremddomain","art":"lehrkraft","email":"lena@anderswo.invalid"}`),
			http.StatusBadRequest, "Die Adresse muss auf @lesertest.invalid enden. Angemeldet wird über den Schulserver.")
	})
	t.Run("Anlegen: Schüler doppelt", func(t *testing.T) {
		legeAn(t, `{"vorname":"Dora","nachname":"Doppelt","klasse":"7a","geburtsdatum":"2012-05-06"}`)
		erwarte(t, anlegen(t, `{"vorname":"Dora","nachname":"Doppelt","klasse":"7b","geburtsdatum":"2012-05-06"}`), http.StatusConflict,
			"Achtung: Ein Schüler mit diesem Namen (auch in anderer Schreibweise: Müller/Mueller, Groß/Klein) und Geburtsdatum existiert bereits im System.")
	})
	t.Run("Anlegen: Kollege mit demselben Namen", func(t *testing.T) {
		legeAn(t, `{"vorname":"Nora","nachname":"Namensgleich","art":"praktikum"}`)
		erwarte(t, anlegen(t, `{"vorname":"Nora","nachname":"Namensgleich","art":"praktikum"}`), http.StatusConflict,
			"Achtung: Unter diesem Namen steht bereits ein Leser in der Leserdatei. Hat sich die Person über Mein Portal schon selbst angemeldet? Ein zweiter Eintrag teilt ihre Ausleihen auf zwei Akten.")
	})
	t.Run("Anlegen: Ausweisnummer vergeben", func(t *testing.T) {
		legeAn(t, `{"vorname":"Ali","nachname":"Ausweiserst","klasse":"7a","geburtsdatum":"2012-07-08","barcode_id":"S-MELDUNG-1"}`)
		erwarte(t, anlegen(t, `{"vorname":"Ben","nachname":"Ausweiszweit","klasse":"7a","geburtsdatum":"2012-07-09","barcode_id":"S-MELDUNG-1"}`),
			http.StatusBadRequest, "Barcode-ID 'S-MELDUNG-1' wird bereits verwendet")
	})
	t.Run("Anlegen: Schul-E-Mail mit Zugang", func(t *testing.T) {
		legeAn(t, `{"vorname":"Kai","nachname":"Kontoerst","art":"lehrkraft","email":"kai.konto@lesertest.invalid"}`)
		erwarte(t, anlegen(t, `{"vorname":"Kim","nachname":"Kontozweit","art":"lehrkraft","email":"kai.konto@lesertest.invalid"}`),
			http.StatusConflict, "Unter kai.konto@lesertest.invalid steht bereits ein Zugang. Die Person ist schon in der Leserdatei. Bitte dort suchen, statt einen zweiten Eintrag anzulegen.")
	})

	// Akte
	legeAn(t, `{"vorname":"Sina","nachname":"Aktenkind","klasse":"7a","geburtsdatum":"2012-09-10"}`)
	schueler := leserID(t, "Aktenkind")
	legeAn(t, `{"vorname":"Lea","nachname":"Aktenlehrkraft","art":"lehrkraft","email":"lea.akte@lesertest.invalid"}`)
	lehrkraft := leserID(t, "Aktenlehrkraft")
	for _, f := range []struct{ name, id, rumpf, meldung string }{
		{"unbekannte Art", schueler, `{"art":"zauberer"}`, `Unbekannte Art "zauberer". Möglich sind: ` + moeglicheArten + "."},
		{"Schüler wird Kollege", schueler, `{"art":"lehrkraft"}`,
			"Aus einem Schüler wird kein Kollege und umgekehrt. Lege die Person neu an und lösche den falschen Eintrag."},
		{"Vorname leer", schueler, `{"vorname":"  "}`, "Bitte das Feld „Vorname“ ausfüllen"},
		{"Geburtsdatum leer", schueler, `{"geburtsdatum":""}`,
			"Das Geburtsdatum lässt sich nicht leeren. Es ist der Schlüssel für den LUSD-Abgleich."},
		{"Klasse Lehrer", schueler, `{"klasse":"lehrer"}`, meldungLehrerKlasse},
		{"Schul-E-Mail für einen Schüler", schueler, `{"email":"sina@lesertest.invalid"}`, meldungSchuelerMail},
		{"Ausweisnummer leer", schueler, `{"barcode_id":""}`,
			"Ausweisnummer darf nicht leer sein. Ohne sie ist der Schüler an der Theke nicht zu finden."},
		{"Schul-E-Mail entfernen", lehrkraft, `{"email":""}`,
			"Die Schul-E-Mail lässt sich hier nicht entfernen, ohne sie kommt die Person nicht mehr ins Portal. Das Konto abschalten geht in der Benutzerverwaltung."},
		{"Schul-E-Mail ändern", lehrkraft, `{"email":"lea.neu@lesertest.invalid"}`,
			"Unter lea.akte@lesertest.invalid besteht bereits ein Zugang. Die Adresse ist der Schlüssel der Anmeldung und lässt sich nur in der Benutzerverwaltung ändern. Hier lässt sie sich nur nachtragen, solange keine eingetragen ist."},
	} {
		t.Run("Akte: "+f.name, func(t *testing.T) {
			erwarte(t, aendern(t, f.id, f.rumpf), http.StatusBadRequest, f.meldung)
		})
	}
	t.Run("Akte: ohne Sitzung", func(t *testing.T) {
		rec := rufe(t, srv.LeserAendern(repository.NewProtokollAbfragen(pool)), http.MethodPatch, "/api/schueler/"+schueler, schueler, `{"vorname":"X"}`, nil)
		erwarte(t, rec, http.StatusUnauthorized, "Nicht angemeldet. Bitte neu anmelden.")
	})

	// Löschen
	t.Run("Löschen: der eigene Eintrag", func(t *testing.T) {
		var kontoID string
		if err := pool.QueryRow(ctx, `SELECT id::text FROM benutzer WHERE leser_id = $1`, lehrkraft).Scan(&kontoID); err != nil {
			t.Fatalf("Konto der Lehrkraft lesen: %v", err)
		}
		selbst := &auth.Claims{KontoID: kontoID, Rolle: auth.RolleAdmin}
		rec := rufe(t, srv.LeserLoeschen(repository.NewLeserAbfragen(pool)), http.MethodDelete, "/api/schueler/"+lehrkraft, lehrkraft, "", selbst)
		erwarte(t, rec, http.StatusForbidden, "Der eigene Eintrag lässt sich nicht löschen. Mit ihm fiele der eigene Zugang.")
	})
}
