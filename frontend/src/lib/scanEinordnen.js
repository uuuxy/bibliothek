/**
 * Einordnung eines Scans OHNE NETZ — der Zwilling von internal/service/omnibox_service.go.
 *
 * Mit Netz entscheidet der Server, was ein Scan ist: Vorsilbe, sonst der Reihe nach Buch,
 * Schüler-, Lehrerausweis, Volltextsuche. Ohne Netz gibt es niemanden zu fragen, und die
 * Theke muss es selbst wissen — sonst laufen die folgenden Bücher auf die falsche Person,
 * und das faellt erst beim Nachbuchen auf (entschieden am 13.09.2026).
 *
 * Die Regeln, und warum sie so sind:
 *
 *  - `A-` / `S-` / `L-` sind Ausweise. Vergeben wird seit dem 16.09.2026 nur noch `A-`;
 *    gelesen werden alle drei, weil es Nummern von frueher gibt und Nummern nie recycelt
 *    werden. Dieselbe Liste wie im Server-Switch.
 *  - `B-` ist ein Buch, `LMF-` ebenfalls. Beide gelten AUCH OHNE geladene Barcode-Liste
 *    (Entscheidung vom 13.09.2026): Die Vorsilbe ist eindeutig, und eine fehlende Liste
 *    darf nicht dazu fuehren, dass ein klar erkennbares Buch verworfen wird.
 *  - `G-` ist ein Geraet. Geraete offline stehen ausdruecklich NICHT im Umfang
 *    (OFFEN.md 2.4) — sie haengen an einer Checkliste, die es ohne Netz nicht gibt.
 *    Deshalb eine eigene Art mit eigener Meldung statt eines stillen „unklar".
 *  - Alles andere ist eine nackte Nummer, und die entscheidet die Barcode-Liste. Erst
 *    roh nachschlagen, dann ueber die Littera-Rueckrechnung: Littera druckt die
 *    Mediennummer im Klartext und codiert im Strichcode eine EAN-13 (litteraEtikett.js).
 *    Steht sie drauf, ist es ein Buch; sonst „unklar".
 *
 * Warum „unklar" und nicht „dann eben Ausweis": Ein Schuelerausweis des Altbestands
 * liefert gemessen `B97601826457`, ein Buchetikett eine 13-stellige EAN. Wer hier raet,
 * schreibt die naechsten Buecher einem fremden Kind zu. Ein unklarer Scan sperrt deshalb
 * die Zuordnung, bis ein eindeutiger Ausweis kommt.
 */
import { dekodiereLitteraEtikett } from './litteraEtikett.js';

/** Die Vorsilben eines Ausweises — dieselbe Menge wie im Server-Switch. */
export const AUSWEIS_VORSILBEN = ['A-', 'S-', 'L-'];

/** Vorsilben, die ein Buch auch ohne Barcode-Liste eindeutig machen. */
export const BUCH_VORSILBEN = ['B-', 'LMF-'];

/**
 * @typedef {{ art: 'buch' | 'ausweis' | 'geraet' | 'suche' | 'unklar', nummer: string }} ScanEinordnung
 */

/**
 * Ordnet einen Scan ohne Netz ein.
 *
 * @param {string} roh der gescannte Text
 * @param {(nummer: string) => boolean} istBuch Nachschlag in der Barcode-Liste des Rechners
 * @returns {ScanEinordnung}
 */
export function ordneScanEin(roh, istBuch) {
	const scan = (roh ?? '').trim();
	if (scan === '') return { art: 'unklar', nummer: '' };

	if (AUSWEIS_VORSILBEN.some((v) => scan.startsWith(v))) return { art: 'ausweis', nummer: scan };
	if (BUCH_VORSILBEN.some((v) => scan.startsWith(v))) return { art: 'buch', nummer: scan };
	if (scan.startsWith('G-')) return { art: 'geraet', nummer: scan };

	// Eine EINGABE ist kein Scan: Wer einen Namen tippt, sucht eine Person, und die Suche
	// braucht den Server — auf dem Theken-Rechner liegen keine Personendaten (Entscheidung
	// vom 13.09.2026). Das verdient eine eigene Auskunft: „steht nicht in der Buchliste"
	// waere ueber einen Namen schlicht falsch und half niemandem weiter.
	//
	// Erkannt am Fehlen von Ziffern oder an einem Leerzeichen — beides kommt aus keinem
	// Strichcode. Eine Nummer wie B97601826457 bleibt damit „unklar", ein „Mueller" oder
	// „Anna Mueller" wird zur Suche.
	if (!/\d/.test(scan) || /\s/.test(scan)) return { art: 'suche', nummer: scan };

	// Ohne Vorsilbe entscheidet die Liste. Roh zuerst: Der Altbestand traegt seine
	// Littera-Mediennummer nackt als Exemplar-Barcode.
	if (istBuch(scan)) return { art: 'buch', nummer: scan };

	// Dann das Etikett: Der Strichcode traegt die EAN-13, die Liste die Nummer darin.
	// Gebucht wird unter der NUMMER, nicht unter dem Aufdruck — der Server kennt nur sie.
	const ausEtikett = dekodiereLitteraEtikett(scan);
	if (ausEtikett !== null && istBuch(ausEtikett)) return { art: 'buch', nummer: ausEtikett };

	return { art: 'unklar', nummer: scan };
}
