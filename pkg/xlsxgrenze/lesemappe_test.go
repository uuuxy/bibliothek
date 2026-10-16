package xlsxgrenze

import (
	"bytes"
	"errors"
	"testing"

	"github.com/xuri/excelize/v2"
)

// LeseMappe ist die Schranke zwischen einer hochgeladenen Datei und excelize: Ein Absturz der
// Bibliothek wird ein Fehler, ein OLE-Container kommt nicht bei ihr an, und ein Fehler der
// Lese-Funktion geht unverändert durch. Die Abstürze selbst lassen sich nicht an jeder
// gemeldeten Lücke nachstellen; nachgestellt ist die Schranke: Ohne den recover in LeseMappe
// reißt die erste Probe die Goroutine des Tests.

// kleineMappe ist eine gewöhnliche Arbeitsmappe mit einer Zelle.
func kleineMappe(t *testing.T) []byte {
	t.Helper()
	f := excelize.NewFile()
	if err := f.SetCellValue("Sheet1", "A1", "harmlos"); err != nil {
		t.Fatal(err)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestLeseMappe_AbsturzDerBibliothekWirdEinFehler(t *testing.T) {
	_, err := LeseMappe(bytes.NewReader(kleineMappe(t)), func(f *excelize.File) ([][]string, error) {
		// Steht für einen Absturz in excelize: lies läuft hinter derselben Schranke wie
		// OpenReader und jedes GetRows.
		panic("index out of range [-1]")
	})
	if !errors.Is(err, errMappeBeschaedigt) {
		t.Fatalf("LeseMappe(…) = %v, erwartet errMappeBeschaedigt", err)
	}
	if !IstUnlesbar(err) {
		t.Fatalf("IstUnlesbar muss den abgefangenen Absturz als unlesbare Datei einordnen")
	}
}

func TestLeseMappe_OleContainerWirdVorExcelizeAbgewiesen(t *testing.T) {
	// Die ersten acht Bytes eines verschlüsselten Office-Containers, danach Füllwerk. Mehr
	// braucht es nicht: excelize entscheidet am Kopf für den Weg der Entschlüsselung, und
	// dorthin darf die Datei nicht.
	ole := append(append([]byte{}, oleMagie...), bytes.Repeat([]byte{0}, 64)...)
	_, err := LeseMappe(bytes.NewReader(ole), func(f *excelize.File) ([][]string, error) {
		t.Fatal("die Lese-Funktion darf für einen OLE-Container nicht laufen")
		return nil, nil
	})
	if !errors.Is(err, errMappeVerschluesselt) {
		t.Fatalf("LeseMappe(…) = %v, erwartet errMappeVerschluesselt", err)
	}
}

func TestLeseMappe_KaputtesZipIstBeschaedigtNichtFehler500(t *testing.T) {
	_, err := LeseMappe(bytes.NewReader([]byte("kein zip")), func(f *excelize.File) ([][]string, error) {
		t.Fatal("die Lese-Funktion darf für ein kaputtes Zip nicht laufen")
		return nil, nil
	})
	if !errors.Is(err, errMappeBeschaedigt) {
		t.Fatalf("LeseMappe(…) = %v, erwartet errMappeBeschaedigt", err)
	}
}

func TestLeseMappe_LiestUndReichtEigeneFehlerUnveraendertDurch(t *testing.T) {
	zeilen, err := LeseMappe(bytes.NewReader(kleineMappe(t)), func(f *excelize.File) ([][]string, error) {
		return f.GetRows(f.GetSheetList()[0])
	})
	if err != nil || len(zeilen) != 1 || zeilen[0][0] != "harmlos" {
		t.Fatalf("Zeilen %v, Fehler %v — erwartet eine Zeile „harmlos“", zeilen, err)
	}

	eigener := errors.New("keine daten gefunden")
	_, err = LeseMappe(bytes.NewReader(kleineMappe(t)), func(f *excelize.File) ([][]string, error) {
		return nil, eigener
	})
	if !errors.Is(err, eigener) || IstUnlesbar(err) {
		t.Fatalf("der Fehler der Lese-Funktion muss unverändert durchgehen, war %v", err)
	}
}
