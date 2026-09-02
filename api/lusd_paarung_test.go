package api

import (
	"strings"
	"testing"
	"time"
)

// Die Paarungsregeln ohne Datenbank: Welche Signale ein Paar tragen, welche nicht, und
// wie Konkurrenz aufgelöst wird. Die Datenbankseite (Vorschau → Bestätigung → derselbe
// Datensatz) steht in lusd_umbenennung_pg_test.go.

func tag(j int, m time.Month, t int) *time.Time {
	d := time.Date(j, m, t, 0, 0, 0, 0, time.UTC)
	return &d
}

func bestandsZeile(id, vorname, nachname, klasse string, geb, eintritt *time.Time) lusdBestandsSchueler {
	return lusdBestandsSchueler{ID: id, Vorname: vorname, Nachname: nachname, Klasse: klasse, Geburtsdatum: geb, EintrittAm: eintritt}
}

func TestKlassenNachbar(t *testing.T) {
	faelle := []struct {
		a, b string
		soll bool
	}{
		{"05F1", "05F1", true}, {"5f1", "05F1", true}, {"05F1", "06F1", true}, {"06F1", "05F1", true},
		{"05F1", "07F1", false}, {"05F1", "06F2", false}, {"E1", "Q1", false}, {"E1", "E1", true},
	}
	for _, f := range faelle {
		if got := klassenNachbar(f.a, f.b); got != f.soll {
			t.Errorf("klassenNachbar(%q,%q)=%v, erwartet %v", f.a, f.b, got, f.soll)
		}
	}
}

func TestNamensteilGleich(t *testing.T) {
	faelle := []struct {
		a, b string
		soll bool
	}{
		{"Müller", "Mueller", true}, {"Al-Sayed", "Al Sayed", true}, {"Ayman Sharafudin", "Ayman", true},
		{"Anna", "anna ", true}, {"Anna", "Hanna", false}, {"", "Anna", false},
	}
	for _, f := range faelle {
		if got := namensteilGleich(f.a, f.b); got != f.soll {
			t.Errorf("namensteilGleich(%q,%q)=%v, erwartet %v", f.a, f.b, got, f.soll)
		}
	}
}

// Die Signal-Matrix: sicher nur mit Schuleintritt; vermutlich mit Geburtsdatum + einem
// weiteren Signal oder Name + Klasse; ohne tragendes Signal kein Paar.
func TestBewertePaar_Signale(t *testing.T) {
	geb, eintritt := tag(2013, 5, 4), tag(2024, 8, 19)
	s := bestandsZeile("s1", "Anna", "Müller", "05F1", geb, eintritt)
	faelle := []struct {
		name         string
		rec          parsedStudentRow
		paar, sicher bool
		grund        string
	}{
		{"Umbenennung mit Eintritt", parsedStudentRow{Vorname: "Anna", Nachname: "Mueller-Schmidt", Klasse: "06F1", GebDatum: geb, EintrittAm: eintritt}, true, true, "gleicher Schuleintritt"},
		{"Datumskorrektur mit Eintritt", parsedStudentRow{Vorname: "Anna", Nachname: "Müller", Klasse: "09A", GebDatum: tag(2013, 5, 14), EintrittAm: eintritt}, true, true, "gleicher Name"},
		{"Nachname neu, gleiche Klasse", parsedStudentRow{Vorname: "Anna", Nachname: "Schulz", Klasse: "05F1", GebDatum: geb}, true, false, "gleicher Vorname"},
		{"Datumskorrektur ohne Eintritt", parsedStudentRow{Vorname: "Anna", Nachname: "Müller", Klasse: "06F1", GebDatum: tag(2013, 5, 14)}, true, false, "Korrektur"},
		{"nur Geburtsdatum", parsedStudentRow{Vorname: "Ben", Nachname: "Schulz", Klasse: "09A", GebDatum: geb}, false, false, ""},
		{"nur Name, andere Klasse, anderes Datum", parsedStudentRow{Vorname: "Anna", Nachname: "Müller", Klasse: "09A", GebDatum: tag(2013, 5, 14)}, false, false, ""},
		{"Anschrift + Geburtsdatum", parsedStudentRow{Vorname: "Lea", Nachname: "Klein", Klasse: "09A", GebDatum: geb, Strasse: "Weg 1", PLZ: "61381"}, false, false, ""},
	}
	for _, f := range faelle {
		k, ok := bewertePaar(0, f.rec, &s)
		if ok != f.paar || k.sicher != f.sicher {
			t.Errorf("%s: paar=%v sicher=%v (grund %q), erwartet paar=%v sicher=%v", f.name, ok, k.sicher, k.grund, f.paar, f.sicher)
		}
		if ok && !strings.Contains(k.grund, f.grund) {
			t.Errorf("%s: Grund %q nennt %q nicht", f.name, k.grund, f.grund)
		}
	}
	// Anschrift zählt, wenn der Bestand sie hat.
	s.Strasse, s.PLZ = "Weg 1", "61381"
	if _, ok := bewertePaar(0, parsedStudentRow{Vorname: "Lea", Nachname: "Klein", Klasse: "09A", GebDatum: geb, Strasse: "Weg 1", PLZ: "61381"}, &s); !ok {
		t.Error("Geburtsdatum + Anschrift muss ein Paar ergeben")
	}
}

// Konkurrenz: Ein Abgänger, zwei mögliche Zeilen — das stärkere Paar gewinnt, jede Seite
// steht nur einmal. Nur-Name-Modus liefert nie Paare.
func TestFindeUmbenennungen_KonkurrenzUndModus(t *testing.T) {
	geb, eintritt := tag(2013, 5, 4), tag(2024, 8, 19)
	bestand := []lusdBestandsSchueler{
		bestandsZeile("alt", "Anna", "Müller", "05F1", geb, eintritt),
		bestandsZeile("weg", "Tim", "Weg", "10A", tag(2009, 1, 1), nil),
	}
	datei := lusdDatei{Modus: lusdModusName, Zeilen: []parsedStudentRow{
		{LineNum: 2, Vorname: "Anna", Nachname: "Schulz", Klasse: "06F1", GebDatum: geb},                        // vermutlich
		{LineNum: 3, Vorname: "Anna", Nachname: "Mueller", Klasse: "06F1", GebDatum: geb, EintrittAm: eintritt}, // sicher
	}}
	z := lusdZuordnung{neuZeilen: []int{0, 1}, abgaengerIDs: []string{"alt", "weg"}}
	idx := lusdIndex{abgaenger: map[string]*lusdBestandsSchueler{}}

	paare := findeUmbenennungen(datei, bestand, idx, z)
	if len(paare) != 1 || paare[0].Zeile != 3 || paare[0].SchuelerID != "alt" || !paare[0].Sicher {
		t.Fatalf("erwartet genau das sichere Paar Zeile 3 ↔ alt: %+v", paare)
	}
	if paare[0].AltNachname != "Müller" || paare[0].NeuNachname != "Mueller" || paare[0].NeuGeburtsdatum != "2013-05-04" {
		t.Errorf("Paar trägt falsche Angaben: %+v", paare[0])
	}

	datei.Modus = lusdModusNurName
	if p := findeUmbenennungen(datei, bestand, idx, z); len(p) != 0 {
		t.Errorf("Nur-Name-Modus darf keine Paare bilden: %+v", p)
	}
}

// Abgänger aus früheren Läufen sind Kandidaten, anonymisierte nicht.
func TestFindeUmbenennungen_FruehereAbgaenger(t *testing.T) {
	geb := tag(2013, 5, 4)
	alt := bestandsZeile("alt", "Anna", "Müller", "05F1", geb, nil)
	alt.IstAbgaenger = true
	anon := bestandsZeile("anon", "Abgänger", "Anonymisiert-x", "ABG", nil, nil)
	anon.IstAbgaenger, anon.Anonymisiert = true, true
	idx := lusdIndex{abgaenger: map[string]*lusdBestandsSchueler{"k1": &alt, "k2": &anon}}
	datei := lusdDatei{Modus: lusdModusName, Zeilen: []parsedStudentRow{
		{LineNum: 2, Vorname: "Anna", Nachname: "Müller-Klein", Klasse: "06F1", GebDatum: geb},
	}}
	paare := findeUmbenennungen(datei, nil, idx, lusdZuordnung{neuZeilen: []int{0}})
	if len(paare) != 1 || paare[0].SchuelerID != "alt" || !paare[0].WarAbgaenger {
		t.Fatalf("erwartet Paar mit dem früheren Abgänger (war_abgaenger): %+v", paare)
	}
}

// Die Wahl des Admins: Nur vorgeschlagene Paare zählen; ein bestätigtes Paar verlässt
// Neuzugänge und Abgänger und landet in der Zuordnung mit Datumsübernahme.
func TestUebernimmUmbenennungen(t *testing.T) {
	datei := lusdDatei{Zeilen: []parsedStudentRow{{LineNum: 2}, {LineNum: 3}}}
	paare := []UmbenennungDiff{{Zeile: 3, SchuelerID: "alt"}}
	z := lusdZuordnung{zielID: map[int]string{}, geburtsdatumSetzen: map[int]bool{}, abgaengerIDs: []string{"alt", "weg"}}
	res := &LusdPreviewResult{
		NewStudents: []StudentDiff{{ID: "zeile-2"}, {ID: "zeile-3"}},
		Graduates:   []StudentDiff{{ID: "alt"}, {ID: "weg"}},
	}
	if err := uebernimmUmbenennungen(datei, []umbenennungWahl{{Zeile: 3, SchuelerID: "alt"}}, paare, &z, res); err != nil {
		t.Fatal(err)
	}
	if z.zielID[1] != "alt" || !z.geburtsdatumSetzen[1] || len(z.abgaengerIDs) != 1 || z.abgaengerIDs[0] != "weg" {
		t.Errorf("Zuordnung falsch: %+v", z)
	}
	if len(res.NewStudents) != 1 || res.NewStudents[0].ID != "zeile-2" || len(res.Graduates) != 1 || res.Graduates[0].ID != "weg" {
		t.Errorf("Vorschau nicht bereinigt: neu=%+v abg=%+v", res.NewStudents, res.Graduates)
	}
	if !paare[0].Bestaetigt {
		t.Error("Paar muss als bestätigt markiert sein")
	}

	// Fremde Kombination → Fehler, nichts geraten: unbekannte Zeile UND bekannte Zeile
	// mit fremder ID (die zweite Form fasste der Test bis 02.09.2026 nicht).
	for _, w := range []umbenennungWahl{{Zeile: 2, SchuelerID: "weg"}, {Zeile: 3, SchuelerID: "weg"}} {
		err := uebernimmUmbenennungen(datei, []umbenennungWahl{w}, paare, &z, res)
		if _, ok := err.(*errUmbenennungUngueltig); !ok {
			t.Errorf("Wahl %+v: erwartet errUmbenennungUngueltig, bekam %v", w, err)
		}
	}
}

// Kopfzeilen-Alias des Schuleintritts im Stil des Individuellen Berichts.
func TestLusdHeader_SchuleintrittAlias(t *testing.T) {
	m, err := lusdHeaderMap([]string{"Schueler_Vorname", "Schueler_Nachname", "Klassen_Klassenbezeichnung", "Schueler_Geburtsdatum", "Schueler_Eintritt_AktuelleSchule"})
	if err != nil {
		t.Fatal(err)
	}
	if idx, ok := m[lusdColEintritt]; !ok || idx != 4 {
		t.Errorf("Schueler_Eintritt_AktuelleSchule nicht als Schuleintritt erkannt: %+v", m)
	}
	row, err := parseLUSDRow([]string{"Anna", "Müller", "05F1", "04.05.2013", "19.08.2024"}, m, 2)
	if err != nil {
		t.Fatal(err)
	}
	if row.EintrittAm == nil || row.EintrittAm.Format("2006-01-02") != "2024-08-19" {
		t.Errorf("Schuleintritt nicht geparst: %v", row.EintrittAm)
	}
}

// Zwillinge: gleiches Geburtsdatum, gleicher Schuleintritt, gleicher Nachname — nur
// der Vorname unterscheidet sie. Ein abweichender Vorname bei gleichem Geburtsdatum ist
// ein Gegen-Signal: höchstens „vermutlich", nie vorangekreuzt (Rasterdurchgang 02.09.).
func TestBewertePaar_ZwillingNieSicher(t *testing.T) {
	geb, eintritt := tag(2013, 5, 4), tag(2024, 8, 19)
	anna := bestandsZeile("anna", "Anna", "Müller", "05F1", geb, eintritt)
	lena := parsedStudentRow{Vorname: "Lena", Nachname: "Müller", Klasse: "06F1", GebDatum: geb, EintrittAm: eintritt}
	k, ok := bewertePaar(0, lena, &anna)
	if !ok {
		t.Fatal("Zwilling darf als vermutliches Paar angeboten werden")
	}
	if k.sicher {
		t.Fatalf("Zwilling darf nie sicher sein: %+v", k)
	}
	if !strings.Contains(k.grund, "Vorname abweichend") {
		t.Errorf("Grund muss das Gegen-Signal nennen: %q", k.grund)
	}
	// Vorname-Tippfehler ohne Präfix-Beziehung ebenso: nur vermutlich.
	aiman := bestandsZeile("s", "Ayman", "Sharaf", "05F1", geb, eintritt)
	k, _ = bewertePaar(0, parsedStudentRow{Vorname: "Aiman", Nachname: "Sharaf", Klasse: "06F1", GebDatum: geb, EintrittAm: eintritt}, &aiman)
	if k.sicher {
		t.Errorf("abweichender Vorname bei gleichem Datum: nicht sicher, bekam %+v", k)
	}
}

// Gleichstand: Zwei Bestandsschüler tragen für dieselbe Zeile exakt gleich starke
// Signale. Dann wird nicht gewürfelt, sondern kein Paar gebildet — der Admin führt
// von Hand zusammen. Das Ergebnis darf nicht von der Reihenfolge des Bestands abhängen.
func TestWaehlePaare_GleichstandKeinPaarUndReihenfolgeEgal(t *testing.T) {
	geb, eintritt := tag(2013, 5, 4), tag(2024, 8, 19)
	anna := bestandsZeile("anna", "Anna", "Müller", "05F1", geb, eintritt)
	lena := bestandsZeile("lena", "Lena", "Müller", "05F1", geb, eintritt)
	datei := lusdDatei{Modus: lusdModusName, Zeilen: []parsedStudentRow{
		{LineNum: 2, Vorname: "Mia", Nachname: "Müller", Klasse: "06F1", GebDatum: geb, EintrittAm: eintritt},
	}}
	idx := lusdIndex{abgaenger: map[string]*lusdBestandsSchueler{}}
	for _, reihenfolge := range [][]lusdBestandsSchueler{{anna, lena}, {lena, anna}} {
		z := lusdZuordnung{neuZeilen: []int{0}, abgaengerIDs: []string{reihenfolge[0].ID, reihenfolge[1].ID}}
		if p := findeUmbenennungen(datei, reihenfolge, idx, z); len(p) != 0 {
			t.Errorf("Gleichstand darf kein Paar ergeben, Reihenfolge %s/%s: %+v", reihenfolge[0].ID, reihenfolge[1].ID, p)
		}
	}
	// Bricht ein Signal den Gleichstand (Vorname passt zu einem), wird genau der
	// gewählt — unabhängig von der Reihenfolge.
	datei.Zeilen[0].Vorname = "Lena-Marie"
	for _, reihenfolge := range [][]lusdBestandsSchueler{{anna, lena}, {lena, anna}} {
		z := lusdZuordnung{neuZeilen: []int{0}, abgaengerIDs: []string{reihenfolge[0].ID, reihenfolge[1].ID}}
		p := findeUmbenennungen(datei, reihenfolge, idx, z)
		if len(p) != 1 || p[0].SchuelerID != "lena" {
			t.Errorf("Reihenfolge %s/%s: erwartet genau Lena, bekam %+v", reihenfolge[0].ID, reihenfolge[1].ID, p)
		}
	}
}

// Paare nur im Modus Name + Geburtsdatum: Im ID-Modus löst die LUSD-ID die Umbenennung
// selbst; ein Paar hätte dort die alte lusd_id behalten und wäre beim nächsten Lauf
// erneut Abgänger + Neuzugang gewesen (Rasterdurchgang 02.09.2026).
func TestFindeUmbenennungen_NurImNamensmodus(t *testing.T) {
	geb, eintritt := tag(2013, 5, 4), tag(2024, 8, 19)
	bestand := []lusdBestandsSchueler{bestandsZeile("alt", "Anna", "Müller", "05F1", geb, eintritt)}
	datei := lusdDatei{Modus: lusdModusID, Zeilen: []parsedStudentRow{
		{LineNum: 2, LusdID: "L-NEU", Vorname: "Anna", Nachname: "Mueller", Klasse: "06F1", GebDatum: geb, EintrittAm: eintritt},
	}}
	z := lusdZuordnung{neuZeilen: []int{0}, abgaengerIDs: []string{"alt"}}
	if p := findeUmbenennungen(datei, bestand, lusdIndex{abgaenger: map[string]*lusdBestandsSchueler{}}, z); len(p) != 0 {
		t.Errorf("ID-Modus darf keine Paare bilden: %+v", p)
	}
}

// Ein früherer Abgänger, den dieser Lauf schon per Schlüssel als Rückkehrer zuordnet,
// ist kein Paar-Kandidat mehr — sonst zeigten zwei Zeilen auf dieselbe ID und der
// Batch schriebe beide (letzter gewinnt).
func TestFindeUmbenennungen_RueckkehrerIstKeinKandidat(t *testing.T) {
	geb := tag(2013, 5, 4)
	alt := bestandsZeile("alt", "Anna", "Müller", "05F1", geb, nil)
	alt.IstAbgaenger = true
	idx := lusdIndex{abgaenger: map[string]*lusdBestandsSchueler{"k1": &alt}}
	datei := lusdDatei{Modus: lusdModusName, Zeilen: []parsedStudentRow{
		{LineNum: 2, Vorname: "Anna", Nachname: "Müller", Klasse: "06F1", GebDatum: geb},       // Rückkehrer per Schlüssel
		{LineNum: 3, Vorname: "Anna", Nachname: "Müller-Klein", Klasse: "06F1", GebDatum: geb}, // sähe wie ein Paar aus
	}}
	z := lusdZuordnung{neuZeilen: []int{1}, zielID: map[int]string{0: "alt"}}
	if p := findeUmbenennungen(datei, nil, idx, z); len(p) != 0 {
		t.Errorf("schon zugeordneter Rückkehrer darf kein Paar-Kandidat sein: %+v", p)
	}
}
