package api

import (
	"strings"
	"testing"
	"time"

	"bibliothek/internal/auskunft"
	"bibliothek/internal/pdftest"
	"bibliothek/pdf"
)

// Das Blatt der Auskunft nennt den erstellten Bescheid in Worten. internal/auskunft kann api
// nicht einbinden und führt den Namen der Aktion deshalb als Text; weicht er von dem ab, was
// diese Tür ins Protokoll schreibt, stünde auf dem Blatt der Name der Aktion.
func TestBescheid_AuskunftNenntDasErstellenInWorten(t *testing.T) {
	a := auskunft.DsgvoAuskunftResponse{
		Art: "Auskunft nach Art. 15 DSGVO",
		Verwaltung: []auskunft.DsgvoVerwaltungsEintrag{
			{Aktion: auditBescheidErstellt, Zeitpunkt: time.Date(2026, 10, 8, 10, 15, 0, 0, time.UTC)},
		},
	}
	roh, err := auskunft.GenerateDsgvoAuskunftPDF(a, pdf.SchuleInfo{Name: "Testschule"})
	if err != nil {
		t.Fatal(err)
	}
	blatt := strings.Join(pdftest.Texte(t, roh), "\n")
	if !strings.Contains(blatt, "Verwaltungsprotokolle zu diesem Datensatz (1)") {
		t.Fatal("der Eintrag steht nicht im Abschnitt der Verwaltungsprotokolle")
	}
	if !strings.Contains(blatt, "Schadensersatz-Bescheid erstellt") {
		t.Error("das Blatt nennt den erstellten Bescheid nicht in Worten")
	}
	if strings.Contains(blatt, auditBescheidErstellt) {
		t.Errorf("auf dem Blatt steht der Name der Aktion (%s)", auditBescheidErstellt)
	}
}
