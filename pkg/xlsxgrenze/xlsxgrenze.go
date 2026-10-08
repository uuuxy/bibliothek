// Package xlsxgrenze ist die EINE Stelle, an der das Programm eine hochgeladene XLSX-Datei
// öffnet und liest — mit der Entpackgrenze und hinter einer Schranke gegen Abstürze.
//
// Eine .xlsx ist ein Zip. Ohne Grenze inflationiert excelize eine winzige, stark
// komprimierte Datei (aufgeblähte sharedStrings) weit über die 100 MB der Leitung hinaus
// in den Speicher — ein authentifizierter Nutzer mit Importrecht konnte den Dienst so
// aus dem Speicher werfen (Sicherheits-Audit 07.09.2026, niedrig).
//
// Dazu kommt, dass eine präparierte Datei die Bibliothek beim Öffnen oder Lesen zum
// Absturz bringen kann (mehrere Meldungen vom 07.10.2026, OFFEN.md 5.10); excelize fängt
// den Absturz selbst nicht ab, und ohne Schranke risse er die Goroutine der Anfrage und
// damit den Prozess mit. MitMappe setzt die Schranke: Ein Absturz der Bibliothek wird zu
// einem Fehler, die Anfrage scheitert, der Dienst läuft weiter. Drei Importwege lesen XLSX
// (Listenimport, Littera, LUSD); sie gehen alle hier durch, damit Grenze und Schranke
// nicht dreimal verschieden sind.
package xlsxgrenze

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

// EntpacktMaxBytes ist die Grenze: 256 MB entpackt. Der größte echte Import
// (Littera-Altbestand, 61.580 Exemplare) liegt weit darunter; eine Bombe erreicht die
// Grenze in Sekunden.
const EntpacktMaxBytes = 256 << 20

// ErrMappeBeschaedigt steht für eine Datei, die excelize nicht öffnen oder lesen kann —
// auch für einen abgefangenen Absturz der Bibliothek. Der Aufrufer fasst sie nach außen
// als „ungültige Datei", nicht als Serverfehler.
var ErrMappeBeschaedigt = errors.New("xlsx-mappe beschädigt")

// ErrMappeVerschluesselt steht für einen verschlüsselten (OLE-)Container. Das Programm liest
// nur unverschlüsselte XLSX; so eine Datei geht gar nicht erst an excelize, weil deren
// Entschlüsselungsweg an einer selbst gewählten Zahl ungebremst rechnet (CVE-2026-107219).
var ErrMappeVerschluesselt = errors.New("xlsx-mappe verschlüsselt")

// oleMagie sind die ersten acht Bytes eines OLE2-Containers (verschlüsselte oder sehr alte
// Office-Datei). excelize schlägt bei so einem Kopf den Entschlüsselungsweg ein, noch bevor
// feststeht, ob überhaupt ein Passwort vorliegt.
var oleMagie = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}

// Optionen liefert die excelize-Optionen mit Entpackgrenze. Beide Grenzen tragen denselben
// Wert: Nur so bleibt der verwundbare Auslagerungs-Weg der Zeichenkettentabelle unerreichbar
// (negativer_sharedstring_test.go).
func Optionen() excelize.Options {
	return excelize.Options{UnzipSizeLimit: EntpacktMaxBytes, UnzipXMLSizeLimit: EntpacktMaxBytes}
}

// IstUnlesbar sagt, ob der Fehler eine kaputte oder verschlüsselte Datei meint — und keinen
// Fehler, den die Lese-Funktion selbst formuliert hat. Der Aufrufer übersetzt das in seine
// eigene „ungültige Datei"-Meldung und lässt alles andere unverändert durch.
func IstUnlesbar(err error) bool {
	return errors.Is(err, ErrMappeBeschaedigt) || errors.Is(err, ErrMappeVerschluesselt)
}

// MitMappe öffnet die XLSX-Datei aus r mit der Entpackgrenze und reicht sie an lies. Der
// Rückgabewert von lies kommt unverändert zurück; schlägt das Öffnen fehl oder stürzt die
// Bibliothek beim Öffnen oder in lies ab, kommt stattdessen ein Fehler mit ErrMappeBeschaedigt.
// Ein OLE-Container wird vor dem Öffnen abgewiesen (ErrMappeVerschluesselt).
//
// lies soll nur mit excelize sprechen (Blätter, Zeilen) und die gelesenen Zeichenketten
// zurückgeben; eigene Weiterverarbeitung gehört hinter MitMappe, damit die Schranke allein
// die Bibliothek umschließt und keinen eigenen Fehler verschluckt.
func MitMappe[T any](r io.Reader, lies func(*excelize.File) (T, error)) (ergebnis T, err error) {
	gepuffert := bufio.NewReader(r)
	// Peek verschiebt die Leseposition nicht: excelize liest danach alle Bytes.
	if kopf, perr := gepuffert.Peek(len(oleMagie)); perr == nil && bytes.Equal(kopf, oleMagie) {
		return ergebnis, ErrMappeVerschluesselt
	}

	// Die Schranke: Ein Absturz in OpenReader oder in lies wird zum Fehler, statt die
	// Goroutine der Anfrage zu reißen. excelize bringt keinen eigenen recover mit.
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%w: absturz beim lesen: %v", ErrMappeBeschaedigt, p)
		}
	}()

	f, oerr := excelize.OpenReader(gepuffert, Optionen())
	if oerr != nil {
		return ergebnis, fmt.Errorf("%w: %v", ErrMappeBeschaedigt, oerr)
	}
	defer func() { _ = f.Close() }() //nolint:errcheck

	return lies(f)
}
