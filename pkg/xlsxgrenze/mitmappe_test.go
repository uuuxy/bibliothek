package xlsxgrenze

import (
	"bytes"
	"errors"
	"testing"

	"github.com/xuri/excelize/v2"
)

// MitMappe ist die Schranke zwischen einer hochgeladenen Datei und excelize: Ein Absturz
// der Bibliothek wird ein Fehler, ein OLE-Container kommt gar nicht erst bei ihr an, und
// ein Fehler der Lese-Funktion geht unverändert durch. Die Abstürze selbst lassen sich
// nicht an jeder der gemeldeten Lücken nachstellen (OFFEN.md 5.10); nachgestellt ist die
// Schranke — ohne den recover in MitMappe reißt die erste Probe die Test-Goroutine.

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

func TestMitMappe_AbsturzDerBibliothekWirdEinFehler(t *testing.T) {
	_, err := MitMappe(bytes.NewReader(kleineMappe(t)), func(f *excelize.File) ([][]string, error) {
		// Stellvertretend für einen Absturz in excelize: lies läuft im selben
		// Schrankenbereich wie OpenReader und jede GetRows-Zeile.
		panic("index out of range [-1]")
	})
	if !errors.Is(err, ErrMappeBeschaedigt) {
		t.Fatalf("erwartet ErrMappeBeschaedigt, war %v", err)
	}
	if !IstUnlesbar(err) {
		t.Fatalf("IstUnlesbar muss den abgefangenen Absturz als unlesbare Datei einordnen")
	}
}

func TestMitMappe_OleContainerWirdVorExcelizeAbgewiesen(t *testing.T) {
	// Die ersten acht Bytes eines verschlüsselten Office-Containers, danach Füllwerk: Mehr
	// braucht es nicht, denn excelize entscheidet allein am Kopf für den
	// Entschlüsselungsweg — und genau dorthin darf die Datei nicht.
	ole := append(append([]byte{}, oleMagie...), bytes.Repeat([]byte{0}, 64)...)
	_, err := MitMappe(bytes.NewReader(ole), func(f *excelize.File) ([][]string, error) {
		t.Fatal("die Lese-Funktion darf für einen OLE-Container nicht laufen")
		return nil, nil
	})
	if !errors.Is(err, ErrMappeVerschluesselt) {
		t.Fatalf("erwartet ErrMappeVerschluesselt, war %v", err)
	}
}

func TestMitMappe_KaputtesZipIstBeschaedigtNichtFehler500(t *testing.T) {
	_, err := MitMappe(bytes.NewReader([]byte("kein zip")), func(f *excelize.File) ([][]string, error) {
		t.Fatal("die Lese-Funktion darf für ein kaputtes Zip nicht laufen")
		return nil, nil
	})
	if !errors.Is(err, ErrMappeBeschaedigt) {
		t.Fatalf("erwartet ErrMappeBeschaedigt, war %v", err)
	}
}

func TestMitMappe_LiestUndReichtEigeneFehlerUnveraendertDurch(t *testing.T) {
	zeilen, err := MitMappe(bytes.NewReader(kleineMappe(t)), func(f *excelize.File) ([][]string, error) {
		return f.GetRows(f.GetSheetList()[0])
	})
	if err != nil || len(zeilen) != 1 || zeilen[0][0] != "harmlos" {
		t.Fatalf("Zeilen %v, Fehler %v — erwartet eine Zeile „harmlos“", zeilen, err)
	}

	eigener := errors.New("keine daten gefunden")
	_, err = MitMappe(bytes.NewReader(kleineMappe(t)), func(f *excelize.File) ([][]string, error) {
		return nil, eigener
	})
	if !errors.Is(err, eigener) || IstUnlesbar(err) {
		t.Fatalf("der Fehler der Lese-Funktion muss unverändert durchgehen, war %v", err)
	}
}
