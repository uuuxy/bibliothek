// Package xlsxgrenze öffnet und liest eine hochgeladene XLSX-Datei mit einer Grenze für das
// Entpacken und hinter einer Schranke gegen Abstürze der Bibliothek.
//
// Eine .xlsx ist ein Zip. Ohne Grenze packt excelize eine kleine, stark gepackte Datei weit
// über die 100 MB der Leitung hinaus in den Speicher aus, und ein angemeldeter Nutzer mit
// Importrecht könnte den Dienst so beenden.
//
// Eine präparierte Datei kann die Bibliothek außerdem beim Öffnen oder Lesen zum Absturz
// bringen. excelize fängt ihn nicht ab, und ohne Schranke risse er die Goroutine der Anfrage
// und damit den Prozess mit. LeseMappe macht aus dem Absturz einen Fehler: Die Anfrage
// scheitert, der Dienst läuft weiter. Die drei Wege, die XLSX lesen (Listenimport, Littera,
// LUSD), gehen alle hier durch, damit Grenze und Schranke überall dieselben sind.
package xlsxgrenze

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

// EntpacktMaxBytes ist die Grenze für das Entpacken: 256 MB. Der größte Import der Schule
// (Altbestand aus Littera, 61.580 Exemplare) liegt weit darunter; eine präparierte Datei
// erreicht die Grenze in Sekunden.
const EntpacktMaxBytes = 256 << 20

// errMappeBeschaedigt steht für eine Datei, die excelize nicht öffnen oder lesen kann, auch
// für einen abgefangenen Absturz der Bibliothek. Der Rufer meldet sie als ungültige Datei und
// nicht als Fehler des Servers (IstUnlesbar).
var errMappeBeschaedigt = errors.New("xlsx-mappe beschädigt")

// errMappeVerschluesselt steht für einen verschlüsselten Container (OLE). Das Programm liest
// nur unverschlüsselte XLSX; eine solche Datei geht nicht an excelize, weil dessen Weg der
// Entschlüsselung mit einer Zahl aus der Datei ungebremst rechnet (CVE-2026-107219).
var errMappeVerschluesselt = errors.New("xlsx-mappe verschlüsselt")

// oleMagie sind die ersten acht Bytes eines OLE2-Containers (verschlüsselte oder sehr alte
// Office-Datei). excelize schlägt bei einem solchen Kopf den Weg der Entschlüsselung ein, noch
// bevor feststeht, ob ein Passwort vorliegt.
var oleMagie = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}

// Optionen liefert die Optionen für excelize mit der Grenze für das Entpacken. Beide Grenzen
// tragen denselben Wert: Nur so bleibt der verwundbare Weg unerreichbar, auf dem excelize die
// Tabelle der Zeichenketten auslagert (negativer_sharedstring_test.go).
func Optionen() excelize.Options {
	return excelize.Options{UnzipSizeLimit: EntpacktMaxBytes, UnzipXMLSizeLimit: EntpacktMaxBytes}
}

// IstUnlesbar sagt, ob der Fehler eine kaputte oder verschlüsselte Datei meint und keinen
// Fehler, den die Lese-Funktion selbst gemeldet hat. Der Rufer macht daraus seine Meldung
// „ungültige Datei" und reicht alles andere unverändert durch.
func IstUnlesbar(err error) bool {
	return errors.Is(err, errMappeBeschaedigt) || errors.Is(err, errMappeVerschluesselt)
}

// LeseMappe öffnet die XLSX-Datei aus r mit der Grenze für das Entpacken und reicht sie an
// lies. Was lies liefert, kommt unverändert zurück. Schlägt das Öffnen fehl oder stürzt die
// Bibliothek beim Öffnen oder in lies ab, kommt ein Fehler, den IstUnlesbar erkennt; ein
// OLE-Container wird vor dem Öffnen abgewiesen.
//
// lies soll nur mit excelize sprechen (Blätter, Zeilen) und die gelesenen Zeichenketten
// liefern. Die Verarbeitung gehört hinter LeseMappe, damit die Schranke nur die Bibliothek
// umschließt und keinen eigenen Fehler verschluckt.
func LeseMappe[T any](r io.Reader, lies func(*excelize.File) (T, error)) (ergebnis T, err error) {
	gepuffert := bufio.NewReader(r)
	// Peek verschiebt die Leseposition nicht: excelize liest danach alle Bytes.
	if kopf, perr := gepuffert.Peek(len(oleMagie)); perr == nil && bytes.Equal(kopf, oleMagie) {
		return ergebnis, errMappeVerschluesselt
	}

	// Ein Absturz in OpenReader oder in lies wird hier zum Fehler; excelize fängt ihn nicht
	// selbst ab.
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%w: absturz beim lesen: %v", errMappeBeschaedigt, p)
		}
	}()

	mappe, oerr := excelize.OpenReader(gepuffert, Optionen())
	if oerr != nil {
		return ergebnis, fmt.Errorf("%w: %w", errMappeBeschaedigt, oerr)
	}
	defer func() { _ = mappe.Close() }() //nolint:errcheck

	return lies(mappe)
}
