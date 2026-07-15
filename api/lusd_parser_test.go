package api

import (
	"testing"
	"time"
)

func TestParseLUSDCSV_BasicCommaDelimited(t *testing.T) {
	csv := "lusd_id,vorname,nachname,klasse,geburtsdatum\n" +
		"L1,Max,Mustermann,5a,15.03.2010\n" +
		"L2,Erika,Musterfrau,7b,2009-06-01\n"

	rows, ids, err := parseLUSDCSV([]byte(csv))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("erwartete 2 Zeilen, bekam %d", len(rows))
	}
	if len(ids) != 2 || ids[0] != "L1" || ids[1] != "L2" {
		t.Errorf("erwartete IDs [L1 L2], bekam %v", ids)
	}
	if rows[0].Vorname != "Max" || rows[0].Klasse != "5a" {
		t.Errorf("Zeile 0 falsch geparst: %+v", rows[0])
	}
	if rows[0].GebDatum == nil || !rows[0].GebDatum.Equal(time.Date(2010, 3, 15, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Geburtsdatum 15.03.2010 falsch geparst: %v", rows[0].GebDatum)
	}
	if rows[1].GebDatum == nil || !rows[1].GebDatum.Equal(time.Date(2009, 6, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("ISO-Geburtsdatum falsch geparst: %v", rows[1].GebDatum)
	}
}

func TestParseLUSDCSV_SemicolonAutoDetected(t *testing.T) {
	csv := "lusd_id;vorname;nachname;klasse\n" +
		"L1;Max;Mustermann;5a\n"

	rows, _, err := parseLUSDCSV([]byte(csv))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(rows) != 1 || rows[0].Nachname != "Mustermann" {
		t.Errorf("Semikolon-Trennung nicht erkannt: %+v", rows)
	}
}

func TestParseLUSDCSV_HeaderCaseAndWhitespaceInsensitive(t *testing.T) {
	csv := " LUSD_ID , Vorname ,NACHNAME, Klasse \n" +
		"L1,Max,Mustermann,5a\n"

	rows, _, err := parseLUSDCSV([]byte(csv))
	if err != nil {
		t.Fatalf("Header-Normalisierung soll greifen, Fehler: %v", err)
	}
	if len(rows) != 1 || rows[0].Vorname != "Max" {
		t.Errorf("normalisierte Header falsch zugeordnet: %+v", rows)
	}
}

func TestParseLUSDCSV_MissingRequiredColumn(t *testing.T) {
	// 'klasse' fehlt
	csv := "lusd_id,vorname,nachname\nL1,Max,Mustermann\n"

	_, _, err := parseLUSDCSV([]byte(csv))
	if err == nil {
		t.Fatal("fehlende Pflichtspalte 'klasse' soll Fehler liefern")
	}
}

func TestParseLUSDCSV_EmptyRequiredValueAborts(t *testing.T) {
	// Nachname leer in Datenzeile
	csv := "lusd_id,vorname,nachname,klasse\nL1,Max,,5a\n"

	_, _, err := parseLUSDCSV([]byte(csv))
	if err == nil {
		t.Fatal("leerer Pflichtwert soll Import abbrechen")
	}
}

func TestParseLUSDCSV_InvalidDateBecomesNilNotError(t *testing.T) {
	csv := "lusd_id,vorname,nachname,klasse,geburtsdatum\n" +
		"L1,Max,Mustermann,5a,kein-datum\n"

	rows, _, err := parseLUSDCSV([]byte(csv))
	if err != nil {
		t.Fatalf("ungültiges Datum soll kein harter Fehler sein: %v", err)
	}
	if rows[0].GebDatum != nil {
		t.Errorf("unparsbares Datum soll nil ergeben, bekam %v", rows[0].GebDatum)
	}
}

func TestParseLUSDCSV_DeduplicatesByLusdIDKeepingLast(t *testing.T) {
	// Gleiche LUSD-ID zweimal: spätere Zeile gewinnt, kein Duplikat in Ergebnis/IDs.
	csv := "lusd_id,vorname,nachname,klasse\n" +
		"L1,Max,Mustermann,5a\n" +
		"L1,Max,Mustermann,6a\n"

	rows, ids, err := parseLUSDCSV([]byte(csv))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("Dedup nach LUSD-ID erwartet 1 Zeile, bekam %d", len(rows))
	}
	if rows[0].Klasse != "6a" {
		t.Errorf("spätere Zeile soll gewinnen (Klasse 6a), bekam %q", rows[0].Klasse)
	}
	if len(ids) != 1 {
		t.Errorf("LUSD-ID soll nur einmal gelistet sein, bekam %v", ids)
	}
}

func TestParseLUSDCSV_RowsWithoutLusdIDAreKeptButNotDeduped(t *testing.T) {
	csv := "lusd_id,vorname,nachname,klasse\n" +
		",Max,Mustermann,5a\n" +
		",Erika,Musterfrau,7b\n"

	rows, ids, err := parseLUSDCSV([]byte(csv))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("Zeilen ohne LUSD-ID sollen erhalten bleiben, bekam %d", len(rows))
	}
	if len(ids) != 0 {
		t.Errorf("ohne LUSD-ID sollen keine IDs gesammelt werden, bekam %v", ids)
	}
}

func TestParseLUSDCSV_TrimsValues(t *testing.T) {
	csv := "lusd_id,vorname,nachname,klasse\n" +
		"  L1 ,  Max , Mustermann ,  5a \n"

	rows, _, err := parseLUSDCSV([]byte(csv))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if rows[0].LusdID != "L1" || rows[0].Vorname != "Max" || rows[0].Klasse != "5a" {
		t.Errorf("Werte sollen getrimmt werden: %+v", rows[0])
	}
}

func TestParseLUSDCSV_EmptyHeaderErrors(t *testing.T) {
	_, _, err := parseLUSDCSV([]byte(""))
	if err == nil {
		t.Fatal("leerer Inhalt soll Fehler bei Kopfzeile liefern")
	}
}

func TestParseLUSDCSV_AddressColumnsParsed(t *testing.T) {
	csv := "lusd_id,vorname,nachname,klasse,strasse,hausnummer,plz,ort,eltern_email\n" +
		"L1,Max,Mustermann,5a,Hauptstraße,12a,63500,Seligenstadt,eltern@example.de\n"

	rows, _, err := parseLUSDCSV([]byte(csv))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	got := rows[0]
	if got.Strasse != "Hauptstraße" || got.Hausnummer != "12a" || got.PLZ != "63500" ||
		got.Ort != "Seligenstadt" || got.ElternEmail != "eltern@example.de" {
		t.Errorf("Adressspalten falsch geparst: %+v", got)
	}
}

func TestParseLUSDCSV_AddressColumnsOptional(t *testing.T) {
	// Export ohne Adressspalten bleibt gültig; die Adressfelder sind dann leer.
	csv := "lusd_id,vorname,nachname,klasse\nL1,Max,Mustermann,5a\n"

	rows, _, err := parseLUSDCSV([]byte(csv))
	if err != nil {
		t.Fatalf("Adressspalten sind optional, Fehler: %v", err)
	}
	if rows[0].Strasse != "" || rows[0].ElternEmail != "" {
		t.Errorf("ohne Adressspalten sollen die Felder leer sein: %+v", rows[0])
	}
}

func TestParseLUSDCSV_PartialAddressColumns(t *testing.T) {
	// Nur ein Teil der optionalen Spalten vorhanden — der Rest bleibt leer.
	csv := "lusd_id,vorname,nachname,klasse,eltern_email\n" +
		"L1,Max,Mustermann,5a,eltern@example.de\n"

	rows, _, err := parseLUSDCSV([]byte(csv))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if rows[0].ElternEmail != "eltern@example.de" {
		t.Errorf("vorhandene E-Mail soll geparst werden: %+v", rows[0])
	}
	if rows[0].Strasse != "" || rows[0].PLZ != "" {
		t.Errorf("fehlende Adressspalten sollen leer bleiben: %+v", rows[0])
	}
}
