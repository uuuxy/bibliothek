package inventur

import (
	"reflect"
	"testing"

	"bibliothek/internal/tuertest"
	"bibliothek/pdf"

	"github.com/google/go-cmp/cmp"
)

// Das Blatt der Schulbücher entsteht aus derselben Antwort wie der Reiter im Portal. Jedes Feld
// kommt aus seiner Quelle, die Werte sind je Feld verschieden; ein Feld, das pdf.Schulbuch später
// dazubekommt und das niemand füllt, fällt an tuertest.LeereFelder auf.
func TestSchulbuecherFuerPDF_FuelltJedesFeldAusDerAbfrage(t *testing.T) {
	titel := []LernmittelTitel{{
		ID: "titel-1", Titel: "Mathematik 7", Autor: "Autorin", Fach: "Mathematik",
		CoverURL: "/uploads/cover.png", ISBN: "9783060000001", Gesamt: 30, Verliehen: 12,
		Verfuegbar: 17, JahrgangVon: 7, JahrgangBis: 8, Zweig: "Gymnasium", Gezaehlt: "02.02.2026",
		AuflagenBestand: []AuflageImBestand{
			{ID: "neu", Auflage: "4. Aufl.", Erscheinungsjahr: 2023, Gesamt: 18},
			{ID: "alt", Auflage: "3. Aufl.", Erscheinungsjahr: 2019, Gesamt: 12},
		},
	}}
	will := []pdf.Schulbuch{{
		Titel: "Mathematik 7", Autor: "Autorin", Fach: "Mathematik", CoverURL: "/uploads/cover.png",
		ISBN: "9783060000001", JahrgangVon: 7, JahrgangBis: 8, Zweig: "Gymnasium",
		Gezaehlt: "02.02.2026", Gesamt: 30, Verliehen: 12, Verfuegbar: 17,
		Auflagen: []pdf.SchulbuchAuflage{
			{Auflage: "4. Aufl.", Erscheinungsjahr: 2023, Gesamt: 18},
			{Auflage: "3. Aufl.", Erscheinungsjahr: 2019, Gesamt: 12},
		},
	}}
	ist := schulbuecherFuerPDF(titel)
	if !cmp.Equal(ist, will) {
		t.Errorf("Zeilen =\n%+v\nerwartet\n%+v", ist, will)
	}
	if leer := tuertest.LeereFelder(reflect.ValueOf(ist[0]), "pdf.Schulbuch"); len(leer) > 0 {
		t.Errorf("schulbuecherFuerPDF füllt diese Felder nicht: %v", leer)
	}
}
