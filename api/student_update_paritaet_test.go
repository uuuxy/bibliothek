package api

import (
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// Gate gegen die andere Hälfte der Datenverlust-Bugklasse — hier am dynamischen
// UPDATE der Schülerdatei.
//
// inventur/schema_paritaet_test.go bewacht die eine Seite: eine SPALTE, die kein Schreiber
// füllt. Diese Seite ist die gefährlichere, weil sie täglich benutzt wird: ein FELD im
// Request-Typ, für das niemand ein b.add* aufruft.
//
// Was dann passiert, ist genau das Muster, das auf diesem Projekt schon mehrfach zugeschlagen
// hat: Das Formular schickt den Wert, der Server nimmt ihn an, antwortet mit HTTP 200 — und
// speichert ihn nicht. Kein Fehler, keine Lücke im Bildschirm, der alte Wert bleibt stehen.
// Auffallen kann das nur jemandem, der die Daten hinterher von Hand ansieht.
//
// Der Test füllt den Request-Typ per Reflexion — deshalb ist ein NEUES Feld automatisch
// mitgeprüft, ohne dass jemand daran denken muss. Genau das unterscheidet ihn von einem
// Test, der zwölf Felder aufzählt: Der wäre grün geblieben, als das dreizehnte dazukam.

// Felder des Request-Typs, die absichtlich in KEINE SET-Zuweisung münden. Jedes braucht
// einen Grund, der über „steht sonst rot" hinausgeht.
var nichtInsUpdate = map[string]string{
	// lusd_id hat einen eigenen kontrollierten Pfad (pruefeUndSetzeLusdID im Handler):
	// nur nachtragbar wenn leer, eindeutig, auditiert. Ein rohes b.add* im generischen
	// Builder verknüpfte den Datensatz sonst ungeprüft mit einer fremden LUSD-Identität
	// (Betreiber-Entscheidung 18.08.2026). Das FELD wird also sehr wohl geschrieben —
	// nur nicht hier, sondern nach eigener Prüfung. Test: TestLusdIDKontrolliertNachtragbar.
	"lusd_id": "kontrollierter Pfad pruefeUndSetzeLusdID (nur nachtragbar wenn leer, eindeutig, auditiert)",
}

// beispielwert liefert einen Wert, den JEDES Feld dieses Typs verträgt.
//
// „2010-05-04" ist bewusst für alle Strings gewählt: Geburtsdatum wird geparst
// (parseGeburtsdatum, erwartet YYYY-MM-DD) und würde bei „x" die Anfrage mit 400 abbrechen —
// der Test hätte dann NICHTS geprüft und wäre trotzdem an der Zusage unten vorbeigelaufen.
func beispielwert(t reflect.Type) reflect.Value {
	switch t.Kind() {
	case reflect.String:
		return reflect.ValueOf("2010-05-04")
	case reflect.Int:
		return reflect.ValueOf(2030)
	case reflect.Bool:
		return reflect.ValueOf(true)
	default:
		return reflect.Value{}
	}
}

// alleFelderGesetzt baut einen patchStudentRequest, in dem jedes Feld belegt ist.
func alleFelderGesetzt(t *testing.T) (*patchStudentRequest, []string) {
	t.Helper()

	req := &patchStudentRequest{}
	v := reflect.ValueOf(req).Elem()
	typ := v.Type()

	var jsonNamen []string
	for i := 0; i < typ.NumField(); i++ {
		feld := typ.Field(i)
		name := strings.Split(feld.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		jsonNamen = append(jsonNamen, name)

		if feld.Type.Kind() != reflect.Pointer {
			t.Fatalf("Feld %s ist kein Zeiger — der PATCH unterscheidet „nicht geschickt“ "+
				"von „auf leer gesetzt“ über nil, das muss so bleiben", feld.Name)
		}
		wert := beispielwert(feld.Type.Elem())
		if !wert.IsValid() {
			t.Fatalf("Feld %s hat den Typ %s, für den beispielwert() keinen Wert kennt — "+
				"bitte dort ergänzen, sonst prüft dieser Test das Feld nicht", feld.Name, feld.Type.Elem())
		}
		zeiger := reflect.New(feld.Type.Elem())
		zeiger.Elem().Set(wert)
		v.Field(i).Set(zeiger)
	}
	return req, jsonNamen
}

func TestPatchSchuelerVerwirftKeinFeld(t *testing.T) {
	req, jsonNamen := alleFelderGesetzt(t)

	w := httptest.NewRecorder()
	b, ok := baueSchuelerUpdate(w, req)
	if !ok {
		t.Fatalf("baueSchuelerUpdate hat abgelehnt (HTTP %d): %s", w.Code, w.Body.String())
	}
	query, _ := b.build("UPDATE schueler SET aktualisiert_am = CURRENT_TIMESTAMP", "irgendeine-id")

	// Nur die SET-Liste: Das WHERE nennt id, und ohne diese Eingrenzung gälte ein Feld
	// namens „id" als geschrieben.
	setTeil := query
	if i := strings.Index(query, " WHERE "); i > 0 {
		setTeil = query[:i]
	}
	geschrieben := func(spalte string) bool {
		return strings.Contains(setTeil, ", "+spalte+" = $")
	}

	// Drei Gegenproben gegen den stillen Nulllauf. Ohne sie wäre der Test auch dann grün,
	// wenn die Reflexion nichts fände oder die Erkennung alles bejahte.
	if len(jsonNamen) < 8 {
		t.Fatalf("nur %d Felder im Request-Typ gefunden — die Reflexion ist kaputt: %v", len(jsonNamen), jsonNamen)
	}
	if !geschrieben("vorname") {
		t.Fatalf("„vorname“ gilt als nicht geschrieben — die Erkennung der SET-Liste ist kaputt.\nQuery: %s", query)
	}
	if geschrieben("gibtesnicht") {
		t.Fatal("eine erfundene Spalte gilt als geschrieben — die Erkennung ist zu großzügig")
	}

	var verworfen []string
	for _, name := range jsonNamen {
		if geschrieben(name) {
			continue
		}
		if _, bekannt := nichtInsUpdate[name]; bekannt {
			continue
		}
		verworfen = append(verworfen, name)
	}

	if len(verworfen) > 0 {
		t.Errorf(
			"patchStudentRequest nimmt %d Feld(er) entgegen, die in KEINE SET-Zuweisung münden: %s\n"+
				"Das Formular schickt sie, der Server antwortet 200 — und speichert sie nicht.\n"+
				"Entweder in baueSchuelerUpdate ein b.add* ergänzen oder in nichtInsUpdate\n"+
				"eintragen, MIT dem Grund und dem Weg, der stattdessen zuständig ist.\n"+
				"Query war: %s",
			len(verworfen), strings.Join(verworfen, ", "), query)
	}

	// Die Ausnahmeliste darf nicht verwildern.
	vorhanden := map[string]bool{}
	for _, n := range jsonNamen {
		vorhanden[n] = true
	}
	for name := range nichtInsUpdate {
		if !vorhanden[name] {
			t.Errorf("Ausnahme für %q, aber das Feld gibt es in patchStudentRequest nicht mehr", name)
		}
	}
}
