package auskunft

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pdftest"
	"bibliothek/pdf"
)

// Das Blatt nennt jeden Vorgang der Datensatz-Historie in Worten. Die Fälle sind die Einträge,
// die die Schreiber heute anlegen: Tabelle, Aktion, Merker und kontext stehen so in
// repository/audit_books.go, audit_users.go, audit_system.go, bescheid_rueckkehr.go,
// schueler_zusammenfuehren.go, titel_loeschen_wartende.go und inventur/db_books_delete_spur.go.
func TestDsgvoVorgang(t *testing.T) {
	satz := func(s string) *string { return &s }
	faelle := []struct {
		name    string
		eintrag DsgvoAuditEintrag
		details string
		worte   string
	}{
		{"Ausleihe", DsgvoAuditEintrag{Tabelle: "ausleihen", Aktion: "CHECKOUT", Akteur: "USER"}, `{"exemplar_id":"x"}`, "Ausleihe"},
		{"Rückgabe", DsgvoAuditEintrag{Tabelle: "ausleihen", Aktion: "RETURN", Akteur: "USER"}, `{}`, "Rückgabe"},
		{"Titel gelöscht bei laufender Ausleihe",
			DsgvoAuditEintrag{Tabelle: "ausleihen", Aktion: "DELETE", Akteur: "SYSTEM", Kontext: satz("Titel gelöscht, Buch war zu diesem Zeitpunkt verliehen")},
			`{"action":"titel_geloescht_mit_offener_ausleihe"}`, "Titel gelöscht, während das Buch ausgeliehen war (automatisch)"},
		{"Papierkorb", DsgvoAuditEintrag{Tabelle: "schueler", Aktion: "UPDATE", Akteur: "USER", Kontext: satz("Soft-Delete Routine")},
			`{"action":"soft_delete"}`, "Leserdatensatz in den Papierkorb gelegt"},
		{"Zusammenführen", DsgvoAuditEintrag{Tabelle: "schueler", Aktion: "ZUSAMMENGEFUEHRT", Akteur: "USER"},
			`{"action":"zusammenfuehren"}`, "Mit einem zweiten Datensatz derselben Person zusammengeführt"},
		{"Auskunft", DsgvoAuditEintrag{Tabelle: "schueler", Aktion: "dsgvo_auskunft", Akteur: "USER"}, `null`, "Auskunft nach Art. 15 DSGVO erteilt"},
		{"Stornierung von Hand", DsgvoAuditEintrag{Tabelle: "schadensfaelle", Aktion: "STORNIERUNG", Akteur: "USER", Kontext: satz("Gebühr storniert")},
			`{"betrag":12.5}`, "Forderung storniert"},
		{"Stornierung, weil das Buch zurück ist",
			DsgvoAuditEintrag{Tabelle: "schadensfaelle", Aktion: "STORNIERUNG", Akteur: "USER", Kontext: satz("Forderung storniert: das Buch ist zurück")},
			`{"anlass":"rueckgabe"}`, "Forderung storniert, weil das Buch zurück ist"},
		{"Titel gelöscht bei offener Forderung",
			DsgvoAuditEintrag{Tabelle: "schadensfaelle", Aktion: "DELETE", Akteur: "SYSTEM", Kontext: satz("Titel gelöscht, es stand noch eine unbezahlte Forderung offen")},
			`{"action":"titel_geloescht_mit_offener_forderung"}`, "Titel gelöscht, während eine Forderung offen war (automatisch)"},
		{"Titel gelöscht bei offener Vormerkung",
			DsgvoAuditEintrag{Tabelle: "vormerkungen", Aktion: "DELETE", Akteur: "SYSTEM", Kontext: satz("Titel gelöscht, eine Vormerkung stand noch offen")},
			`{"action":"titel_geloescht_mit_offenem_bezug"}`, "Titel gelöscht, während eine Vormerkung offen war (automatisch)"},
		// Was das Blatt nicht kennt, bleibt stehen: Aktion, Tabelle, kontext, Akteur.
		{"unbekannter Vorgang", DsgvoAuditEintrag{Tabelle: "neue_tabelle", Aktion: "NEUE_AKTION", Akteur: "DIENST", Kontext: satz("neuer Satz")},
			`{}`, "NEUE_AKTION (neue_tabelle) — neuer Satz (DIENST)"},
		{"bekannter Vorgang mit neuem kontext", DsgvoAuditEintrag{Tabelle: "ausleihen", Aktion: "RETURN", Akteur: "USER", Kontext: satz("am Automaten")},
			`{}`, "Rückgabe — am Automaten"},
	}
	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			details, _ := dsgvoDetails(json.RawMessage(fall.details))
			if ist := dsgvoVorgang(fall.eintrag, details); ist != fall.worte {
				t.Errorf("dsgvoVorgang = %q, erwartet %q", ist, fall.worte)
			}
		})
	}
}

// Die Angaben eines Eintrags stehen mit Bezeichnung da, die Kennungen des Programms und das
// buchende Konto nicht.
func TestDsgvoAngabenZeilen(t *testing.T) {
	const kennung = "3f2a9c1e-0000-4000-8000-000000000001"
	faelle := []struct {
		name    string
		aktion  string
		details string
		amLeser bool
		zeilen  []string
	}{
		{"Stornierung", "STORNIERUNG",
			`{"betrag":24.5,"grund":"Buch im Regal gefunden","storniert_am":"2026-10-08T10:15:00Z","bearbeiter_id":"` + kennung + `","schueler_id":"` + kennung + `"}`,
			false, []string{"Betrag: 24,50 EUR · Grund: Buch im Regal gefunden · storniert am: 08.10.2026, 12:15 Uhr"}},
		{"Sperre aufgehoben", "LESER_ENTSPERRT",
			`{"schueler_id":"` + kennung + `","grund":"Buch überfällig","von_hand":true,"vom_programm":false}`,
			false, []string{"Grund der aufgehobenen Sperre: Buch überfällig · Sperre war von Hand gesetzt: Ja · Sperre kam vom Programm: Nein"}},
		{"Sperre gesetzt", "LESER_GESPERRT", `{"schueler_id":"` + kennung + `","grund":"Buch überfällig"}`, false, []string{"Sperrgrund: Buch überfällig"}},
		{"Bescheid", "SCHADENSERSATZ_BESCHEID_ERSTELLT",
			`{"bescheid_id":"` + kennung + `","schueler_id":"` + kennung + `","referenznummer":"LMF-2026-0007","mittel":"land","gesamtbetrag":31,"positionen":2}`,
			false, []string{"Gesamtbetrag: 31,00 EUR · Referenznummer: LMF-2026-0007 · Mittel: land · Positionen: 2"}},
		{"Spur einer gelöschten Forderung", "DELETE",
			`{"schadensfall_id":"` + kennung + `","barcode_id":"2424","titel":"Mathematik heute 6","schuldner":"Erika Muster","betrag":"12.50","beschreibung":"Wasserschaden","erstellt_am":"2026-09-17","action":"titel_geloescht_mit_offener_forderung","schueler_id":"` + kennung + `"}`,
			false, []string{"Titel: Mathematik heute 6 · Buchnummer: 2424 · Name: Erika Muster · Beschreibung: Wasserschaden · Betrag: 12,50 EUR · erfasst am: 17.09.2026"}},
		{"Papierkorb: barcode_id ist die Ausweisnummer", "UPDATE",
			`{"vorname":"Erika","nachname":"Muster","klasse":"07A","barcode_id":"S-1001","abgaenger_jahr":2030,"grund":"Manuelle Löschung","action":"soft_delete","art":"schueler","konten_geloescht":0}`,
			true, []string{"Ausweisnummer: S-1001 · Vorname: Erika · Nachname: Muster · Klasse: 07A · Art des Lesers: Schüler/in · Grund: Manuelle Löschung · Abgangsjahr: 2030 · gelöschte Zugangskonten: 0"}},
		{"Zusammenführen: Abbild und Umgehängtes je in einer Zeile, Listen als Zahl", "ZUSAMMENGEFUEHRT",
			`{"action":"zusammenfuehren","aufgeloest_id":"` + kennung + `","aufgeloest_barcode":"S-2002",` +
				`"quelle":{"id":"` + kennung + `","barcode_id":"S-2002","vorname":"Erica","nachname":"Muster","geburtsdatum":"2014-05-01","ist_gesperrt":false},` +
				`"gewandert":{"ausleihen":["` + kennung + `","` + kennung + `"],"foto":true}}`,
			true, []string{
				"Ausweisnummer des zweiten Datensatzes: S-2002",
				"Zweiter Datensatz: Ausweisnummer: S-2002 · Vorname: Erica · Nachname: Muster · Geburtsdatum: 01.05.2014 · Gesperrt: Nein",
				"Vom zweiten Datensatz: umgehängte Ausleihen: 2 · Foto übernommen: Ja",
			}},
		{"ein Schlüssel ohne Bezeichnung steht mit seinem Namen da", "NEUE_AKTION",
			`{"schueler_id":"` + kennung + `","neuer_schluessel":"Wert","grund":"x"}`,
			false, []string{"Grund: x · neuer_schluessel: Wert"}},
		{"nur Kennungen: keine Zeile", "CHECKOUT", `{"schueler_id":"` + kennung + `","exemplar_id":"` + kennung + `","zeitpunkt":"2026-10-08T10:15:00Z"}`, false, nil},
	}
	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			details, rest := dsgvoDetails(json.RawMessage(fall.details))
			if len(rest) != 0 {
				t.Fatalf("die Angaben wurden nicht gelesen: %v", rest)
			}
			zeilen := dsgvoAngabenZeilen(details, fall.amLeser, fall.aktion)
			if strings.Join(zeilen, "\n") != strings.Join(fall.zeilen, "\n") {
				t.Errorf("Zeilen:\n%s\nerwartet:\n%s", strings.Join(zeilen, "\n"), strings.Join(fall.zeilen, "\n"))
			}
			if strings.Contains(strings.Join(zeilen, "\n"), kennung) {
				t.Errorf("eine Kennung des Programms steht auf dem Blatt: %v", zeilen)
			}
		})
	}
}

// Angaben, die kein Objekt sind, stehen als Text da.
func TestDsgvoDetails_KeinObjekt(t *testing.T) {
	for roh, soll := range map[string]string{`null`: "", ``: "", `"frei getippt"`: `"frei getippt"`, `[1,2]`: `[1,2]`} {
		details, rest := dsgvoDetails(json.RawMessage(roh))
		if details != nil || strings.Join(rest, "") != soll {
			t.Errorf("dsgvoDetails(%q) = %v, %v; erwartet Text %q", roh, details, rest, soll)
		}
	}
}

func TestDsgvoProtokollZeit(t *testing.T) {
	for text, soll := range map[string]string{
		"2026-10-08T10:15:00Z":          "08.10.2026, 12:15 Uhr",
		"2026-01-08T23:30:00Z":          "09.01.2026, 00:30 Uhr",
		"2026-10-08 10:15:00.123456+00": "08.10.2026, 12:15 Uhr",
		"2026-10-08 12:15:00+02:00":     "08.10.2026, 12:15 Uhr",
		"2026-09-17":                    "17.09.2026",
		"17.09.2026":                    "17.09.2026",
		"irgendwann":                    "irgendwann",
	} {
		if ist := dsgvoProtokollZeit(text); ist != soll {
			t.Errorf("dsgvoProtokollZeit(%q) = %q, erwartet %q", text, ist, soll)
		}
	}
}

// Auf dem gedruckten Blatt: Worte statt der Schreibweise des Programms, Titel und Nummer des
// Buchs, keine Kennung und kein buchendes Konto.
func TestDsgvoPDF_ProtokollzeilenInWorten(t *testing.T) {
	const kennung = "3f2a9c1e-0000-4000-8000-000000000001"
	zeit := time.Date(2026, 10, 8, 10, 15, 0, 0, time.UTC)
	a := DsgvoAuskunftResponse{
		Art:        "Auskunft nach Art. 15 DSGVO",
		ErstelltAm: zeit,
		Stammdaten: DsgvoStammdaten{ID: "leser-1", Vorname: "Erika", Nachname: "Muster", Klasse: "07A"},
		AuditEintraege: []DsgvoAuditEintrag{
			{Tabelle: "ausleihen", Aktion: "CHECKOUT", Akteur: "USER", Zeitpunkt: zeit, Gegenstand: "Mathematik heute 6", Barcode: "2424",
				Details: json.RawMessage(`{"exemplar_id":"` + kennung + `","schueler_id":"` + kennung + `","zeitpunkt":"2026-10-08T10:15:00Z"}`)},
			{Tabelle: "ausleihen", Aktion: "RETURN", Akteur: "USER", Zeitpunkt: zeit,
				Details: json.RawMessage(`{"exemplar_id":"` + kennung + `","schueler_id":"` + kennung + `"}`)},
			{Tabelle: "schadensfaelle", Aktion: "STORNIERUNG", Akteur: "USER", Zeitpunkt: zeit,
				Details: json.RawMessage(`{"betrag":24.5,"grund":"Buch im Regal gefunden","bearbeiter_id":"` + kennung + `","schueler_id":"` + kennung + `"}`)},
		},
		Verwaltung: []DsgvoVerwaltungsEintrag{
			{Aktion: "LESER_GESPERRT", Zeitpunkt: zeit, Details: json.RawMessage(`{"schueler_id":"` + kennung + `","grund":"Buch seit Mai überfällig"}`)},
			{Aktion: "LUSD_ID_NACHGETRAGEN", Zeitpunkt: zeit, Details: json.RawMessage(`{"schueler_id":"` + kennung + `","lusd_id":"4711"}`)},
		},
	}
	roh, err := GenerateDsgvoAuskunftPDF(a, pdf.SchuleInfo{Name: "Testschule"})
	if err != nil {
		t.Fatal(err)
	}
	blatt := strings.Join(pdftest.Texte(t, roh), "\n")
	for _, soll := range []string{
		"Ausleihe", "Mathematik heute 6 (Nummer 2424)", "Rückgabe", "nicht mehr im Bestand",
		"Forderung storniert", "Betrag: 24,50 EUR", "Grund: Buch im Regal gefunden",
		"Von Hand gesperrt", "Sperrgrund: Buch seit Mai überfällig", "LUSD-ID nachgetragen", "LUSD-ID: 4711",
		"Art. 15 Abs. 4 DSGVO",
	} {
		if !strings.Contains(blatt, soll) {
			t.Errorf("auf dem Blatt fehlt %q", soll)
		}
	}
	for _, nicht := range []string{"CHECKOUT", "RETURN", "STORNIERUNG", "LESER_GESPERRT", "LUSD_ID_NACHGETRAGEN", "USER", kennung, "schueler_id", "bearbeiter_id"} {
		if strings.Contains(blatt, nicht) {
			t.Errorf("auf dem Blatt steht %q", nicht)
		}
	}
}
