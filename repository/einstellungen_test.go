package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

// gemerkteStrings erfüllt den Argument-Vergleich von pgxmock und hält die übergebene Liste
// fest, statt sie zu vergleichen. Der Test liest danach, welche Schlüssel geschrieben wurden.
type gemerkteStrings struct{ got []string }

func (g *gemerkteStrings) Match(v any) bool {
	s, ok := v.([]string)
	if ok {
		g.got = s
	}
	return ok
}

// ptr macht aus einem Wert einen Zeiger: Im Patch ist der Zeiger die Aussage „dieses Feld
// wurde mitgeschickt".
func ptr[T any](v T) *T { return &v }

// Der Fall, für den es den Patch gibt: Ein Speichern aus dem vollen Typ, in dem ein fehlendes
// Feld als false, 0 oder "" ankommt, schriebe jeden Schlüssel. Ein Speichern in „Datenschutz &
// Sitzung" schaltete dann Leseclub, Bestellbedarf-Warnung und Preiserfassung aus und setzte
// die Fristen auf die Vorgabe zurück, mit einer Erfolgsmeldung.
func TestEinstellungenSchreibe_PatchSchreibtNurMitgeschickteFelder(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewEinstellungsAbfragen(mock)
	schluessel := &gemerkteStrings{}

	mock.ExpectExec("INSERT INTO system_einstellungen").
		WithArgs(schluessel, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 2))

	// Ein Speichern in „Datenschutz & Sitzung": zwei Felder, sonst nichts.
	if err := repo.Schreibe(context.Background(), &EinstellungenPatch{
		LesehistorieTage: ptr(0),
		SperreMinuten:    ptr(30),
	}); err != nil {
		t.Fatalf("Schreibe: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unerfüllte Mock-Erwartungen: %v", err)
	}

	if len(schluessel.got) != 2 {
		t.Fatalf("nur die zwei mitgeschickten Schlüssel dürfen ins Upsert, geschrieben wurden: %v", schluessel.got)
	}
	for _, k := range []string{
		"ferien_leseclub_aktiv", "bestellbedarf_warnung_aktiv", "preise_erfassen",
		"frist_buch_tage", "frist_medien_tage", "max_ausleihen_schueler",
		"max_overdue_days", "max_overdue_items", "lmf_stichtag",
		"schule_name", "schule_strasse", "schule_plz", "schule_ort",
	} {
		if enthaelt(schluessel.got, k) {
			t.Errorf("fremde Kategorie %q wurde mitgeschrieben; geschriebene Schlüssel: %v", k, schluessel.got)
		}
	}
}

// Die Gegenrichtung: Was die Kategorie mitschickt, kommt an, auch der leere Text. Hieße ein
// leeres Feld „nicht anfassen", ließe sich ein falscher Eigentumsvermerk nicht entfernen.
func TestEinstellungenSchreibe_PatchSchreibtAuchLeereWerte(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewEinstellungsAbfragen(mock)
	schluessel := &gemerkteStrings{}
	werte := &gemerkteStrings{}

	mock.ExpectExec("INSERT INTO system_einstellungen").
		WithArgs(schluessel, werte).
		WillReturnResult(pgxmock.NewResult("INSERT", 5))

	if err := repo.Schreibe(context.Background(), &EinstellungenPatch{
		SchuleName:              ptr("Grundschule Musterhausen"),
		SchuleStrasse:           ptr("Schulstraße 1"),
		SchulePLZ:               ptr("12345"),
		SchuleOrt:               ptr("Musterhausen"),
		EtikettEigentumsvermerk: ptr(""),
	}); err != nil {
		t.Fatalf("Schreibe: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unerfüllte Mock-Erwartungen: %v", err)
	}

	for _, k := range []string{"schule_name", "schule_strasse", "schule_plz", "schule_ort", "etikett_eigentumsvermerk"} {
		if !enthaelt(schluessel.got, k) {
			t.Errorf("mitgeschicktes Feld %q fehlt im Upsert; geschriebene Schlüssel: %v", k, schluessel.got)
		}
	}
	for i, k := range schluessel.got {
		if k == "etikett_eigentumsvermerk" && werte.got[i] != "" {
			t.Errorf("ein geleertes Feld muss als leerer Wert geschrieben werden, kam %q", werte.got[i])
		}
	}
}

// Ein Patch ohne ein einziges Feld schreibt nicht: Ein leerer Rumpf setzte sonst ein UNNEST mit
// null Zeilen ab und täuschte ein Speichern vor, das es nicht gab.
func TestEinstellungenSchreibe_LeererPatchSchreibtNichts(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	if err := NewEinstellungsAbfragen(mock).Schreibe(context.Background(), &EinstellungenPatch{}); err != nil {
		t.Fatalf("Schreibe: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("es hätte gar keine Abfrage geben dürfen: %v", err)
	}
}

// Eine negative Karenz: Für die Nachbarschlüssel heißt 0 „aus" (die Daten bleiben), für die
// Karenz der Abgänger heißt 0 „sofort anonymisieren", der schärfste Wert. Ein Tippfehler „-90"
// darf deshalb weder beim Speichern noch beim Lesen zur 0 werden.
//
// Die beiden Hälften unterscheiden sich: Beim Speichern wird der Wert abgelehnt und nicht
// ersetzt, denn ein Ersatz, der sich „gespeichert" nennt, zeigt etwas anderes, als getippt
// wurde. Beim Lesen bleibt der Ersatz richtig: Dort steht schon ein Wert in der Datenbank, und
// die Vorgabe ist die Antwort, die niemandem schadet.
func TestAbgaengerKarenz_NegativWirdAbgelehntUndBeimLesenVorgabe(t *testing.T) {
	patch := &EinstellungenPatch{AbgaengerKarenzTage: ptr(-5)}
	err := patch.PruefeZahlen()
	if err == nil {
		t.Fatal("-5 Tage Karenz wurde angenommen: Der Wert muss mit einer Meldung abgewiesen werden")
	}
	if !strings.Contains(err.Error(), "Karenz") {
		t.Errorf("die Meldung nennt das Feld nicht: %q", err)
	}
	if paare := paareAusPatch(patch); len(paare) != 1 || paare[0][1] != "-5" {
		t.Errorf("der Sammler darf nichts mehr ersetzen, bekam %v", paare)
	}

	einst := &SystemEinstellungen{}
	wert := "-5"
	setzeDatenschutzEinstellung(einst, AbgaengerKarenzSchluessel, &wert)
	if einst.AbgaengerKarenzTage == nil || *einst.AbgaengerKarenzTage != AbgaengerKarenzTageVorgabe {
		t.Errorf("DB-Wert -5 muss als Vorgabe gelesen werden, bekam %v", einst.AbgaengerKarenzTage)
	}
	null := "0"
	setzeDatenschutzEinstellung(einst, AbgaengerKarenzSchluessel, &null)
	if einst.AbgaengerKarenzTage == nil || *einst.AbgaengerKarenzTage != 0 {
		t.Error("0 bleibt 0 (sofort), nur negativ wird ersetzt")
	}
}
