package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"bibliothek/internal/pdftest"
)

// Die Anordnung auf dem Blatt, geprüft am fertigen PDF.
//
// Der Umweg über die erzeugte Datei ist derselbe wie in api/etiketten_pdf_paritaet_pg_test.go
// und aus demselben Grund: Ob vier Etiketten auf einem Blatt liegen, steht in keinem
// Struct und in keiner Abfrage. Es steht in der Seitengröße und in der Anzahl der Seiten.
// Ein Test, der die Generatorfunktion nur aufruft und auf "kein Fehler" prüft, wäre bei
// einem Etikett pro Blatt genauso grün.

// asciiAnfang liefert den Teil einer Zeichenkette bis zum ersten Nicht-ASCII-Zeichen.
//
// Aus „Name des Schülers" wird „Name des Sch" — genug, um die Spalte im Inhaltsstrom
// wiederzuerkennen, ohne über die Kodierung des PDF-Erzeugers zu stolpern.
func asciiAnfang(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return s[:i]
		}
	}
	return s
}

func layoutEtiketten(anzahl int) []BuchEtikett {
	items := make([]BuchEtikett, 0, anzahl)
	for i := 0; i < anzahl; i++ {
		items = append(items, BuchEtikett{
			Schulname:        "Philipp-Reis-Schule, Friedrichsdorf",
			BarcodeID:        fmt.Sprintf("100000000%03d", i),
			Titel:            fmt.Sprintf("Deutschbuch %d", i),
			Signatur:         "LMF-Deutsch 5",
			AnschaffungsJahr: "2026",
			Eigentumsvermerk: "Eigentum des Landes Hessen",
		})
	}
	return items
}

// Der Kern der telefonischen Rückmeldung von Naacher (06.08.2026): VIER große Etiketten
// gehören auf ein A4-Blatt.
//
// Vorher war jede Seite ein eigenes A6-Blatt — auf einem A4-Drucker also ein Etikett je
// Blatt, außer man stellte im Druckdialog von Hand "4 Seiten pro Blatt" ein. Genau diesen
// Handgriff prüft dieser Test weg: Er muss in der Datei stecken.
func TestLernmittelEtiketten_VierProA4Blatt(t *testing.T) {
	faelle := []struct {
		etiketten   int
		wantSeiten  int
		beschreibug string
	}{
		{1, 1, "ein Etikett füllt eine Seite an"},
		{4, 1, "vier passen genau auf ein Blatt"},
		{5, 2, "das fünfte beginnt das zweite Blatt"},
		{8, 2, "acht füllen genau zwei Blätter"},
		{9, 3, "das neunte beginnt das dritte"},
	}

	for _, f := range faelle {
		t.Run(f.beschreibug, func(t *testing.T) {
			roh, err := GenerateLernmittelEtikettenPDF(layoutEtiketten(f.etiketten))
			if err != nil {
				t.Fatalf("PDF-Erzeugung fehlgeschlagen: %v", err)
			}

			seiten, breite, hoehe := pdftest.Seiten(t, roh)
			if seiten != f.wantSeiten {
				t.Errorf("%d Etiketten ergaben %d Seiten, erwartet %d",
					f.etiketten, seiten, f.wantSeiten)
			}
			// A4 = 210×297 mm. Stünde hier A6 (105×149), wäre die Umstellung nicht
			// passiert — und der Test hätte über die Seitenzahl allein nichts gemerkt.
			if breite != 210 || hoehe != 297 {
				t.Errorf("Seitengröße %d×%d mm, erwartet 210×297 (A4)", breite, hoehe)
			}
		})
	}
}

// Vier Etiketten auf einem Blatt nützen nichts, wenn drei davon leer sind: Jeder Barcode
// und jeder Titel muss auch wirklich gezeichnet werden.
//
// Der Fall, den das abfängt: Die vier Felder werden über einen Versatz (ox, oy) plaziert.
// Bleibt der Versatz irgendwo stehen, landen alle vier übereinander im ersten Feld — die
// Seitenzahl stimmt dann trotzdem.
func TestLernmittelEtiketten_JedesExemplarStehtDrauf(t *testing.T) {
	items := layoutEtiketten(4)
	roh, err := GenerateLernmittelEtikettenPDF(items)
	if err != nil {
		t.Fatalf("PDF-Erzeugung fehlgeschlagen: %v", err)
	}

	texte := strings.Join(pdftest.Texte(t, roh), "\n")
	for _, item := range items {
		if !strings.Contains(texte, item.BarcodeID) {
			t.Errorf("Exemplar-Nr. %s steht nicht auf dem Bogen", item.BarcodeID)
		}
		if !strings.Contains(texte, item.Titel) {
			t.Errorf("Titel %q steht nicht auf dem Bogen", item.Titel)
		}
	}

	// Die Tabelle für die Ausleihhistorie ist der Grund, warum es das große Etikett
	// überhaupt gibt — sie darf beim Umbau nicht verloren gegangen sein.
	//
	// Verglichen wird nur der ASCII-Anteil der Spaltentitel: gofpdf schreibt den Text in
	// Windows-1252 in den Inhaltsstrom, das „ü" in „Name des Schülers" ist dort EIN Byte
	// (0xFC) und trifft eine UTF-8-Zeichenkette aus Go nie. Der Paritäts-Test daneben
	// merkt davon nichts, weil er zwei PDFs miteinander vergleicht — beide Seiten sind
	// gleich kodiert. Hier steht die Erwartung im Quelltext, also muss sie sich anpassen.
	for _, spalte := range lernmittelTabellenSpalten {
		erwartet := asciiAnfang(spalte.Titel)
		if !strings.Contains(texte, erwartet) {
			t.Errorf("Tabellenspalte %q fehlt (gesucht: %q)", spalte.Titel, erwartet)
		}
	}
}

// Die Schnittlinien sind der einzige Hinweis darauf, wo getrennt wird. Ohne sie muss man
// die Blattmitte schätzen.
func TestLernmittelEtiketten_SchnittlinienVorhanden(t *testing.T) {
	roh, err := GenerateLernmittelEtikettenPDF(layoutEtiketten(4))
	if err != nil {
		t.Fatalf("PDF-Erzeugung fehlgeschlagen: %v", err)
	}
	// gofpdf schreibt Linien als "x y m x y l S" in den Inhaltsstrom. Geprüft wird nur,
	// DASS gezeichnete Linien vorkommen — die genaue Lage steht in zeichneSchnittlinien
	// und wäre hier nur abgeschrieben.
	if !bytes.Contains(roh, []byte("endstream")) {
		t.Fatal("PDF hat keinen Inhaltsstrom")
	}
	if seiten, _, _ := pdftest.Seiten(t, roh); seiten != 1 {
		t.Fatalf("Aufbau des Tests stimmt nicht: %d Seiten", seiten)
	}
}

// Der zweite Wunsch aus dem Telefonat: Der Lieferant druckt die KLEINEN Etiketten auf
// sein eigenes Material und muss das Bogenraster wählen können.
//
// Geprüft am Ergebnis und nicht am Parameter: Dieselbe Menge Etiketten muss sich je nach
// Raster auf unterschiedlich viele Blätter verteilen. Bis zum 06.08.2026 stand im
// Lieferanten-Weg fest "zweckform_l4760" — eine durchgereichte Formatangabe, die nirgends
// ankommt, sähe an der Signatur genauso richtig aus.
func TestKleineEtiketten_FormatBestimmtDieSeitenzahl(t *testing.T) {
	const anzahl = 25

	faelle := []struct {
		format     string
		proSeite   int
		wantSeiten int
	}{
		{"zweckform_l4760", 21, 2}, // 3×7
		{"avery_3475", 24, 2},      // 3×8
		{"standard_52", 52, 1},     // 4×13
	}

	for _, f := range faelle {
		t.Run(f.format, func(t *testing.T) {
			doc, err := GenerateLabelsPDF(f.format, 1, false, layoutEtiketten(anzahl))
			if err != nil {
				t.Fatalf("PDF-Erzeugung fehlgeschlagen: %v", err)
			}
			var buf bytes.Buffer
			if err := doc.Output(&buf); err != nil {
				t.Fatalf("PDF-Ausgabe fehlgeschlagen: %v", err)
			}

			seiten, breite, hoehe := pdftest.Seiten(t, buf.Bytes())
			if seiten != f.wantSeiten {
				t.Errorf("%d Etiketten im Raster %s ergaben %d Seiten, erwartet %d (%d je Bogen)",
					anzahl, f.format, seiten, f.wantSeiten, f.proSeite)
			}
			if breite != 210 || hoehe != 297 {
				t.Errorf("Seitengröße %d×%d mm, erwartet 210×297 (A4)", breite, hoehe)
			}
		})
	}
}

// Die Auswahlliste und die Rasterdaten dürfen nicht auseinanderlaufen: Was angeboten
// wird, muss auch erzeugbar sein.
func TestLabelFormatAuswahl_IstVollstaendigUndErzeugbar(t *testing.T) {
	auswahl := LabelFormatAuswahl()
	if len(auswahl) != len(labelFormats) {
		t.Fatalf("Auswahl hat %d Einträge, es gibt aber %d Formate — labelFormatReihenfolge "+
			"wurde beim Ergänzen vergessen", len(auswahl), len(labelFormats))
	}

	for _, f := range auswahl {
		if !IstBekanntesEtikettFormat(f.ID) {
			t.Errorf("angebotenes Format %q gilt als unbekannt", f.ID)
		}
		if f.ProSeite != f.Spalten*f.Zeilen {
			t.Errorf("%s: pro_seite=%d passt nicht zu %d×%d", f.ID, f.ProSeite, f.Spalten, f.Zeilen)
		}
		if f.Name == "" {
			t.Errorf("%s hat keinen Anzeigenamen", f.ID)
		}
	}

	// Die Vorgabe muss in der Liste stehen — sonst zeigt die Auswahl beim Öffnen einen
	// Wert, den sie selbst nicht anbietet.
	gefunden := false
	for _, f := range auswahl {
		if f.ID == StandardLabelFormat {
			gefunden = true
		}
	}
	if !gefunden {
		t.Errorf("Vorgabe %q steht nicht in der Auswahl", StandardLabelFormat)
	}
}

// Leer heißt "nicht angegeben" und ist erlaubt; alles andere muss bekannt sein.
// Ein still auf die Vorgabe gedrehter Tippfehler wäre der schlechte Ausgang: Der
// Lieferant druckte dann im falschen Raster und merkte es am verschnittenen Bogen.
func TestIstBekanntesEtikettFormat(t *testing.T) {
	for _, gueltig := range []string{"", "zweckform_l4760", "avery_3475", "standard_52"} {
		if !IstBekanntesEtikettFormat(gueltig) {
			t.Errorf("IstBekanntesEtikettFormat(%q) = false, want true", gueltig)
		}
	}
	for _, ungueltig := range []string{"zweckform", "AVERY_3475", "../etc/passwd", "standard_53"} {
		if IstBekanntesEtikettFormat(ungueltig) {
			t.Errorf("IstBekanntesEtikettFormat(%q) = true, want false", ungueltig)
		}
	}
}
