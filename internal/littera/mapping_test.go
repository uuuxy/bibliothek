package littera

import (
	"strings"
	"testing"
)

// Die Kopfzeilen und Beispielsätze stammen 1:1 aus dem Altbestand des Sekretariats
// (littera_sav.mdb, exportiert am 04.08.2026) — nicht aus einer ausgedachten Struktur.
// Genau daran ist das alte Werkzeug gescheitert: Es fragte Spalten ab, die es nie gab.

const titelCSV = `Buchungsdatum,Buchungsnummer,Haupttitel,Untertitel,ISBN,Erscheinungsjahr,Verlag,Urheber,Verfasserangabe,Haupteintrag,Annotation
"08/03/01 11:41:58",2,"Geschichte und Geschehen I","",978-3-12-415600-6,1997,11,"","","Geschichte und Geschehen I",""
"08/03/01 11:41:58",10,"Die Republik von Weimar 1918-1933","",978-3-12-490250-4,1995,11,"","Reinhard Neebe","Die Republik von Weimar 1918-1933","Kurze Einfuehrung"
"08/03/01 11:41:58",11,"Ohne Jahr","",,"[ca. 2003]",11,"Meier, Anna","","Ohne Jahr",""
"08/03/01 11:41:58",,"Ohne Schluessel","",,1999,11,"","","",""
`

func TestLeseTitel_EchteSpaltennamen(t *testing.T) {
	titel, err := LeseTitel(strings.NewReader(titelCSV))
	if err != nil {
		t.Fatalf("Titel lesen: %v", err)
	}

	// Die Zeile ohne Buchungsnummer faellt raus — ohne Schluessel laesst sich ihr
	// kein Exemplar zuordnen, sie waere ein Katalogeintrag ohne Buecher.
	if len(titel) != 3 {
		t.Fatalf("erwartet 3 Titel (Zeile ohne Schluessel verworfen), waren %d", len(titel))
	}

	if titel[0].Haupttitel != "Geschichte und Geschehen I" {
		t.Errorf("Haupttitel falsch: %q", titel[0].Haupttitel)
	}
	if titel[0].ID != "2" {
		t.Errorf("Schluessel muss die Buchungsnummer sein, war %q", titel[0].ID)
	}

	// Verfasserangabe ist die Autorenquelle ...
	if titel[1].Autor != "Reinhard Neebe" {
		t.Errorf("Autor aus Verfasserangabe erwartet, war %q", titel[1].Autor)
	}
	// ... Haupteintrag NICHT: Er wiederholt bei Sachbuechern den Titel, und der haette
	// hier als Autorenname gestanden.
	if titel[0].Autor == titel[0].Haupttitel && titel[0].Autor != "" {
		t.Errorf("Haupteintrag darf nicht als Autor durchschlagen: %q", titel[0].Autor)
	}
	// Urheber greift, wenn Verfasserangabe leer ist.
	if titel[2].Autor != "Meier, Anna" {
		t.Errorf("Rueckfall auf Urheber erwartet, war %q", titel[2].Autor)
	}
}

func TestJahrAus(t *testing.T) {
	faelle := map[string]int{
		"1997":       1997,
		"[ca. 2003]": 2003,
		"":           0,
		"o.J.":       0,
		"[1995]":     1995,
		// Kein Zusammenkleben ueber Trenner: aus "19" und "97" darf nicht 1997 werden.
		"19-97": 0,
	}
	for roh, erwartet := range faelle {
		if got := jahrAus(roh); got != erwartet {
			t.Errorf("jahrAus(%q) = %d, erwartet %d", roh, got, erwartet)
		}
	}
}

const exemplarCSV = `Buchungsnummer,Titel,Barcode,Sig1,Sig2,Zugangsdatum,Preis
4483,2,"B-00001","LMF Deu 7","Bie","08/03/01 00:00:00",12.90
4484,2,"B-00002","LMF Deu 7","Bie","08/03/01 00:00:00",12.90
4485,10,"B-00003","Ga","Bos","09/05/12 00:00:00",0
4486,11,"B-00004","0","0","",
4487,,"B-00005","Ea","Die","",
`

func TestLeseExemplare_SignaturUndFremdschluessel(t *testing.T) {
	ex, err := LeseExemplare(strings.NewReader(exemplarCSV))
	if err != nil {
		t.Fatalf("Exemplare lesen: %v", err)
	}

	// Das Exemplar ohne Titel-Fremdschluessel faellt raus.
	if len(ex) != 4 {
		t.Fatalf("erwartet 4 Exemplare, waren %d", len(ex))
	}

	if ex[0].Signatur != "LMF Deu 7 / Bie" {
		t.Errorf("Signatur falsch zusammengesetzt: %q", ex[0].Signatur)
	}
	if ex[0].TitelID != "2" {
		t.Errorf("Fremdschluessel muss aus Spalte 'Titel' kommen, war %q", ex[0].TitelID)
	}
	if ex[0].Preis != 12.90 {
		t.Errorf("Preis falsch: %v", ex[0].Preis)
	}
	// Der Platzhalter "0" bedeutet "keine Angabe" und darf nicht am Buch landen.
	if ex[3].Signatur != "" {
		t.Errorf("Platzhalter 0 muss verworfen werden, war %q", ex[3].Signatur)
	}
}

// TestSignaturTrifftDenInventurScope ist der Test, der die beiden Welten verbindet:
// Die aus Littera zusammengesetzte Signatur MUSS von der Praefix-Regel des
// Inventur-Scopes getroffen werden. Sonst zeigt der Katalog ein Regal an, das die
// Inventur nicht findet.
//
// Die Regel schneidet am Leerzeichen — deshalb " / " und nicht "/".
func TestSignaturTrifftDenInventurScope(t *testing.T) {
	voll := SignaturAus("LMF Deu 7", "Bie")
	regal := "LMF Deu 7"

	// Nachbildung von repository.SignaturPraefixBedingung in Go: gleich ODER
	// Praefix mit folgendem Leerzeichen.
	trifft := voll == regal || strings.HasPrefix(voll, regal+" ")
	if !trifft {
		t.Errorf("Regal %q findet sein eigenes Buch %q nicht", regal, voll)
	}

	// Gegenprobe: Das Nachbarregal darf nicht hineinreichen.
	nachbar := "LMF Deu"
	if voll == nachbar || strings.HasPrefix(voll, nachbar+" ") {
		// "LMF Deu" IST ein gueltiges Oberregal von "LMF Deu 7" — das ist gewollt.
		t.Logf("Oberregal %q umfasst %q (beabsichtigt)", nachbar, voll)
	}
	fremd := "LMF De"
	if voll == fremd || strings.HasPrefix(voll, fremd+" ") {
		t.Errorf("fremde Adresse %q darf %q nicht treffen", fremd, voll)
	}
}

func TestSignaturJeTitel_HaeufigsterWertUndAbweichungen(t *testing.T) {
	exemplare := []Exemplar{
		{TitelID: "1", Signatur: "LMF Deu 7 / Bie"},
		{TitelID: "1", Signatur: "LMF Deu 7 / Bie"},
		{TitelID: "1", Signatur: "LMF Deu 8 / Bie"}, // Ausreisser
		{TitelID: "2", Signatur: "Ga / Bos"},
		{TitelID: "3", Signatur: ""}, // ohne Signatur: gar kein Eintrag
	}

	signaturen, abweichend := SignaturJeTitel(exemplare)

	if signaturen["1"] != "LMF Deu 7 / Bie" {
		t.Errorf("haeufigster Wert erwartet, war %q", signaturen["1"])
	}
	if signaturen["2"] != "Ga / Bos" {
		t.Errorf("Titel 2 falsch: %q", signaturen["2"])
	}
	if _, da := signaturen["3"]; da {
		t.Error("Titel ohne Signatur darf keinen Eintrag bekommen")
	}

	// Die Abweichung MUSS gemeldet werden — sonst verschwindet eine echte
	// Regalabweichung still im Import.
	if len(abweichend) != 1 || abweichend[0] != "1" {
		t.Errorf("Titel 1 muss als abweichend gemeldet werden, war %v", abweichend)
	}
}

func TestVerlagNamen(t *testing.T) {
	const verlagCSV = `Buchungsdatum,Buchungsnummer,Verlag,Ort
,7,"Goldmann","Stuttgart"
"03/05/03 15:17:54",8,"Fischer","Frankfurt a.M."
`
	namen, err := VerlagNamen(strings.NewReader(verlagCSV))
	if err != nil {
		t.Fatalf("Verlage lesen: %v", err)
	}
	if namen["7"] != "Goldmann" || namen["8"] != "Fischer" {
		t.Errorf("Verlagsauflösung falsch: %v", namen)
	}
}

// TestCSVMitKommaImFeld sichert den Grund ab, warum hier der Standard-CSV-Leser steht
// und kein selbstgebautes Aufteilen an ",": Die Annotationen im Altbestand enthalten
// Kommata und Zeilenumbrueche.
func TestCSVMitKommaImFeld(t *testing.T) {
	const csvMitKomma = `Buchungsnummer,Haupttitel,Annotation,ISBN
5,"Titel, mit Komma","Eine Zusammenfassung, die ein Komma enthaelt",978-3-12-000000-0
`
	titel, err := LeseTitel(strings.NewReader(csvMitKomma))
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}
	if len(titel) != 1 {
		t.Fatalf("erwartet 1 Titel, waren %d", len(titel))
	}
	if titel[0].Haupttitel != "Titel, mit Komma" {
		t.Errorf("Titel mit Komma falsch zerlegt: %q", titel[0].Haupttitel)
	}
	if titel[0].ISBN != "978-3-12-000000-0" {
		t.Errorf("ISBN verrutscht: %q", titel[0].ISBN)
	}
}
