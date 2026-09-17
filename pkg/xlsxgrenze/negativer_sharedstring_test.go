package xlsxgrenze_test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"testing"

	"bibliothek/pkg/closeutil"
	"bibliothek/pkg/xlsxgrenze"

	"github.com/xuri/excelize/v2"
)

// GO-2026-6452 (CVE-2026-59162, veröffentlicht 16.09.2026): „Panic via negative
// shared-string index in github.com/xuri/excelize". Eine Zelle vom Typ `s` mit einem
// NEGATIVEN Index in die Zeichenkettentabelle greift am Bereichsschutz vorbei.
//
// Es gibt dafür keine heile Fassung — der Eintrag in der Datenbank führt alle Versionen
// ab 0 und nennt keine mit Fix. Deshalb steht hier nachgemessen, statt geglaubt, warum
// dieses Programm den Panic nicht auslösen kann. Am 17.09.2026 an v2.11.0 geprüft:
//
//  1. Der gewöhnliche Weg (Zeichenkettentabelle im Speicher) HAT den Bereichsschutz —
//     `getValueFrom` in cell.go prüft `xlsxSI < 0 || xlsxSI >= len(d.SI)`. Der Fix des
//     Advisories ist dort also bereits eingebaut, nur trägt der Datenbank-Eintrag das
//     nicht nach.
//  2. Ungeschützt ist allein der AUSLAGERUNGS-Weg: Ist die Zeichenkettentabelle größer
//     als UnzipXMLSizeLimit, schreibt excelize sie in eine Temp-Datei, und
//     `getFromStringItem` prüft dort nur die OBERE Grenze (`len(...) <= index`). Ein
//     Index von -1 läuft daran vorbei und trifft `f.sharedStringItem[-1]`.
//
// Und genau dieser Weg ist mit unseren Optionen unerreichbar: Optionen() setzt BEIDE
// Grenzen auf denselben Wert. Eine Datei, deren Zeichenkettentabelle groß genug zum
// Auslagern wäre, überschreitet damit zwangsläufig auch die Gesamtgrenze — und die prüft
// excelize VORHER (lib.go: `unzipSize += fileSize` vor der Auslagerung).
//
// Diese Tests halten beides fest. Wer die beiden Grenzen auseinanderzieht — etwa um
// Speicher zu sparen —, öffnet den verwundbaren Weg und wird hier rot.

// bosartigeMappe baut die kleinste .xlsx mit einer Zelle `t="s"` und dem gegebenen Index.
func bosartigeMappe(t *testing.T, index string) []byte {
	t.Helper()
	puffer := &bytes.Buffer{}
	z := zip.NewWriter(puffer)
	teile := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
<Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`,
		"xl/workbook.xml": `<?xml version="1.0" encoding="UTF-8"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets><sheet name="Tabelle1" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>
</Relationships>`,
		"xl/sharedStrings.xml": `<?xml version="1.0" encoding="UTF-8"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="1" uniqueCount="1"><si><t>harmlos</t></si></sst>`,
		"xl/worksheets/sheet1.xml": fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<sheetData><row r="1"><c r="A1" t="s"><v>%s</v></c></row></sheetData></worksheet>`, index),
	}
	for name, inhalt := range teile {
		w, err := z.Create(name)
		if err != nil {
			t.Fatalf("zip %s: %v", name, err)
		}
		if _, err := w.Write([]byte(inhalt)); err != nil {
			t.Fatalf("schreiben %s: %v", name, err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatalf("zip schließen: %v", err)
	}
	return puffer.Bytes()
}

// liesMit liest die Mappe mit den gegebenen Optionen und meldet einen Panic als Wert,
// statt den Testlauf mitzureißen.
func liesMit(daten []byte, opt excelize.Options) (gepanickt any) {
	defer func() { gepanickt = recover() }()
	f, err := excelize.OpenReader(bytes.NewReader(daten), opt)
	if err != nil {
		return nil // abgewiesen — genau das ist hier ein gutes Ergebnis
	}
	defer closeutil.LogClose(f, "xlsxgrenze-probe")
	// Ein Fehler ist hier ebenfalls ein gutes Ergebnis: Gemessen wird der PANIC, nicht
	// die Frage, ob die präparierte Mappe lesbar ist.
	if _, err := f.GetRows("Tabelle1", excelize.Options{RawCellValue: true}); err != nil {
		return nil
	}
	return nil
}

// Der Nachweis, dass der Panic ECHT ist. Ohne ihn wäre der Test darunter ein grüner
// Test über eine Gefahr, die es vielleicht gar nicht gibt — und niemand wüsste, ob die
// Optionen ihn abwehren oder ob nie etwas zu abzuwehren war.
func TestNegativerSharedStringIndex_IstEineEchteGefahr(t *testing.T) {
	daten := bosartigeMappe(t, "-1")
	// Die Grenzen auseinandergezogen: kleine XML-Grenze, große Gesamtgrenze. Dann lagert
	// excelize die Zeichenkettentabelle aus, und der ungeschützte Weg greift.
	gepanickt := liesMit(daten, excelize.Options{UnzipSizeLimit: 1 << 20, UnzipXMLSizeLimit: 64})
	if gepanickt == nil {
		t.Skip("kein Panic mehr — vermutlich ist GO-2026-6452 in dieser excelize-Fassung behoben. " +
			"Dann gehört die Ausnahme in security/vuln-ausnahmen.json gelöscht.")
	}
	t.Logf("erwarteter Panic im Auslagerungs-Weg: %v", gepanickt)
}

// Und der Test, auf den es ankommt: MIT unseren Optionen passiert nichts.
func TestNegativerSharedStringIndex_UnsereOptionenWehrenAb(t *testing.T) {
	for _, index := range []string{"-1", "-2147483648", "0", "99999"} {
		t.Run("index="+index, func(t *testing.T) {
			if gepanickt := liesMit(bosartigeMappe(t, index), xlsxgrenze.Optionen()); gepanickt != nil {
				t.Errorf("Panic mit den Optionen dieses Programms: %v — ein Import mit einer "+
					"präparierten Datei reißt damit die Anfrage ab (GO-2026-6452)", gepanickt)
			}
		})
	}
}

// Die tragende Invariante, im Kleinen gemessen: Sind beide Grenzen GLEICH, wird eine
// Datei, die groß genug zum Auslagern wäre, vorher abgewiesen.
func TestGleicheGrenzenMachenDenAuslagerungsWegUnerreichbar(t *testing.T) {
	daten := bosartigeMappe(t, "-1")
	for _, grenze := range []int64{64, 128, 256, 512, 1024} {
		if gepanickt := liesMit(daten, excelize.Options{UnzipSizeLimit: grenze, UnzipXMLSizeLimit: grenze}); gepanickt != nil {
			t.Errorf("Grenze %d: Panic trotz gleicher Grenzen — die Annahme hinter Optionen() "+
				"stimmt nicht mehr: %v", grenze, gepanickt)
		}
	}
}

// Optionen() MUSS beide Grenzen gleich setzen. Wer die XML-Grenze senkt, um Speicher zu
// sparen, macht den verwundbaren Weg erreichbar — und zwar lautlos.
func TestOptionenSetzenBeideGrenzenGleich(t *testing.T) {
	o := xlsxgrenze.Optionen()
	if o.UnzipSizeLimit != o.UnzipXMLSizeLimit {
		t.Fatalf("UnzipSizeLimit=%d, UnzipXMLSizeLimit=%d — verschieden. Damit lagert excelize "+
			"die Zeichenkettentabelle in eine Temp-Datei aus, und dort fehlt der Schutz gegen "+
			"negative Indizes (GO-2026-6452).", o.UnzipSizeLimit, o.UnzipXMLSizeLimit)
	}
	if o.UnzipSizeLimit != xlsxgrenze.EntpacktMaxBytes {
		t.Errorf("UnzipSizeLimit=%d, erwartet %d", o.UnzipSizeLimit, xlsxgrenze.EntpacktMaxBytes)
	}
}
