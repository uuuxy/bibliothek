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

// ptr macht aus einem Wert einen Zeiger — im Patch ist der Zeiger die Aussage
// „dieses Feld wurde mitgeschickt", und die braucht man in Tests ständig.
func ptr[T any](v T) *T { return &v }

// Die Regression, um derentwillen es den Patch gibt.
//
// Bis zum 23.08.2026 schrieb SaveSettings ELF Schlüssel bei jedem Aufruf — aus dem
// vollen Struct, in dem ein fehlendes Feld als false/0/"" ankommt. Das ging gut,
// solange die Oberfläche IMMER alles auf einmal schickte. Mit dem Speichern je
// Kategorie hätte ein Klick in „Datenschutz & Sitzung" den Ferien-Leseclub und die
// Bestellbedarf-Warnung ausgeschaltet, die Preiserfassung abgeschaltet und fünf
// Fristen auf die Vorgabe zurückgesetzt — mit grüner Erfolgsmeldung.
func TestSaveSettings_PatchSchreibtNurMitgeschickteFelder(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewSystemSettingsRepository(mock)
	keys := &captureStringSlice{}

	mock.ExpectExec("INSERT INTO system_einstellungen").
		WithArgs(keys, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 2))

	// Ein Speichern in „Datenschutz & Sitzung": zwei Felder, sonst nichts.
	if err := repo.SaveSettings(context.Background(), &EinstellungenPatch{
		LesehistorieTage: ptr(0),
		SperreMinuten:    ptr(30),
	}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unerfüllte Mock-Erwartungen: %v", err)
	}

	if len(keys.got) != 2 {
		t.Fatalf("nur die zwei mitgeschickten Schlüssel dürfen ins Upsert, geschrieben wurden: %v", keys.got)
	}
	for _, k := range []string{
		"ferien_leseclub_aktiv", "bestellbedarf_warnung_aktiv", "preise_erfassen",
		"frist_buch_tage", "frist_medien_tage", "max_ausleihen_schueler",
		"max_overdue_days", "max_overdue_items", "lmf_stichtag",
		"schule_name", "schule_strasse", "schule_plz", "schule_ort",
	} {
		if contains(keys.got, k) {
			t.Errorf("fremde Kategorie %q wurde mitgeschrieben; geschriebene Keys: %v", k, keys.got)
		}
	}
}

// Die Gegenrichtung: Was die Kategorie mitschickt, muss auch ankommen — samt der
// LEEREN Zeichenkette. Vorher war ein leeres Schulfeld „nicht anfassen"; damit ließ
// sich ein falscher Eigentumsvermerk nie wieder entfernen.
func TestSaveSettings_PatchSchreibtAuchLeereWerte(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewSystemSettingsRepository(mock)
	keys := &captureStringSlice{}
	werte := &captureStringSlice{}

	mock.ExpectExec("INSERT INTO system_einstellungen").
		WithArgs(keys, werte).
		WillReturnResult(pgxmock.NewResult("INSERT", 5))

	if err := repo.SaveSettings(context.Background(), &EinstellungenPatch{
		SchuleName:              ptr("Grundschule Musterhausen"),
		SchuleStrasse:           ptr("Schulstraße 1"),
		SchulePLZ:               ptr("12345"),
		SchuleOrt:               ptr("Musterhausen"),
		EtikettEigentumsvermerk: ptr(""),
	}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unerfüllte Mock-Erwartungen: %v", err)
	}

	for _, k := range []string{"schule_name", "schule_strasse", "schule_plz", "schule_ort", "etikett_eigentumsvermerk"} {
		if !contains(keys.got, k) {
			t.Errorf("mitgeschicktes Feld %q fehlt im Upsert; geschriebene Keys: %v", k, keys.got)
		}
	}
	for i, k := range keys.got {
		if k == "etikett_eigentumsvermerk" && werte.got[i] != "" {
			t.Errorf("ein geleertes Feld muss als leerer Wert geschrieben werden, kam %q", werte.got[i])
		}
	}
}

// Ein Patch ohne ein einziges Feld darf gar nicht erst schreiben — sonst setzte ein
// leerer Rumpf ein UNNEST mit null Zeilen ab (harmlos, aber es täuscht einen
// Speichervorgang vor, den es nicht gab).
func TestSaveSettings_LeererPatchSchreibtNichts(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	if err := NewSystemSettingsRepository(mock).SaveSettings(context.Background(), &EinstellungenPatch{}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("es hätte gar keine Abfrage geben dürfen: %v", err)
	}
}
