package repository

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// captureStringSlice ist ein pgxmock-Argument-Matcher, der den übergebenen
// []string festhält, statt ihn auf Gleichheit zu prüfen. So können die Tests
// nach dem Aufruf den tatsächlichen Inhalt (z.B. die geschriebenen Settings-Keys)
// inspizieren.
type captureStringSlice struct{ got []string }

func (c *captureStringSlice) Match(v any) bool {
	s, ok := v.([]string)
	if ok {
		c.got = s
	}
	return ok
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// TestSaveSettings_AllgemeinDoesNotWipeSchoolIdentity sichert die Datenverlust-Regression ab:
// Die "Allgemein"-Sektion der Settings-UI sendet KEINE schule_*-Felder mit (sie hat dafür keine
// Eingabefelder). Der PUT-Handler dekodiert diesen Teil-Payload aber in ein volles
// SystemEinstellungen-Struct, sodass die Schul-Felder leer ("") sind. SaveSettings darf die in
// fünf PDF-Briefköpfen genutzte Schuladresse NICHT mit leeren Werten überschreiben — leere
// schule_*-Werte bedeuten "nicht angefasst" und dürfen gar nicht erst ins Upsert gelangen.
func TestSaveSettings_AllgemeinDoesNotWipeSchoolIdentity(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewSystemSettingsRepository(mock)
	keys := &captureStringSlice{}

	mock.ExpectExec("INSERT INTO system_einstellungen").
		WithArgs(keys, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 8))

	// Typischer Allgemein-Payload: allgemeine Werte gesetzt, schule_* leer.
	err = repo.SaveSettings(context.Background(), &SystemEinstellungen{
		FerienLeseclubAktiv:  true,
		LmfStichtag:          "08-01",
		MaxAusleihenSchueler: 10,
		FristBuchTage:        30,
		FristMedienTage:      14,
		MaxOverdueDays:       20,
		MaxOverdueItems:      5,
	})
	if err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unerfüllte Mock-Erwartungen: %v", err)
	}

	for _, k := range []string{"schule_name", "schule_strasse", "schule_plz", "schule_ort"} {
		if contains(keys.got, k) {
			t.Errorf("leeres Schulfeld %q darf NICHT geschrieben werden (würde Briefkopf löschen); geschriebene Keys: %v", k, keys.got)
		}
	}
	// Die allgemeinen Keys müssen weiterhin geschrieben werden.
	if !contains(keys.got, "lmf_stichtag") {
		t.Errorf("allgemeine Einstellungen fehlen im Upsert; geschriebene Keys: %v", keys.got)
	}
}

// TestSaveSettings_PersistsSchoolIdentityWhenProvided sichert die Gegenrichtung: Werden die
// Schul-Felder tatsächlich befüllt (echtes Schul-Save), müssen sie im Upsert landen.
func TestSaveSettings_PersistsSchoolIdentityWhenProvided(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewSystemSettingsRepository(mock)
	keys := &captureStringSlice{}

	mock.ExpectExec("INSERT INTO system_einstellungen").
		WithArgs(keys, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 12))

	err = repo.SaveSettings(context.Background(), &SystemEinstellungen{
		LmfStichtag:   "07-31",
		SchuleName:    "Grundschule Musterhausen",
		SchuleStrasse: "Schulstraße 1",
		SchulePLZ:     "12345",
		SchuleOrt:     "Musterhausen",
	})
	if err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unerfüllte Mock-Erwartungen: %v", err)
	}

	for _, k := range []string{"schule_name", "schule_strasse", "schule_plz", "schule_ort"} {
		if !contains(keys.got, k) {
			t.Errorf("befülltes Schulfeld %q muss geschrieben werden; geschriebene Keys: %v", k, keys.got)
		}
	}
}
