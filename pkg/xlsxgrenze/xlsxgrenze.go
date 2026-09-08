// Package xlsxgrenze ist die EINE Stelle, an der das Programm sagt, wie groß eine
// entpackte XLSX-Datei werden darf.
//
// Eine .xlsx ist ein Zip. Ohne Grenze inflationiert excelize eine winzige, stark
// komprimierte Datei (aufgeblähte sharedStrings) weit über die 100 MB der Leitung hinaus
// in den Speicher — ein authentifizierter Nutzer mit Importrecht konnte den Dienst so
// aus dem Speicher werfen (Sicherheits-Audit 07.09.2026, niedrig). Drei Importwege lesen
// XLSX (Listenimport, Littera, LUSD); sie holen sich die Optionen hier, damit die Grenze
// nicht dreimal verschieden ist.
package xlsxgrenze

import "github.com/xuri/excelize/v2"

// EntpacktMaxBytes ist die Grenze: 256 MB entpackt. Der größte echte Import
// (Littera-Altbestand, 61.580 Exemplare) liegt weit darunter; eine Bombe erreicht die
// Grenze in Sekunden.
const EntpacktMaxBytes = 256 << 20

// Optionen liefert die excelize-Optionen mit Entpackgrenze für jeden XLSX-Leser.
func Optionen() excelize.Options {
	return excelize.Options{UnzipSizeLimit: EntpacktMaxBytes, UnzipXMLSizeLimit: EntpacktMaxBytes}
}
