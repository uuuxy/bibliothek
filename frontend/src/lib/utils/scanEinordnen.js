/**
 * Einordnung eines Scans ohne Netz: der Zwilling von internal/service/omnibox.go.
 *
 * Mit Netz entscheidet der Server, was ein Scan ist: Vorsilbe, sonst der Reihe nach Buch,
 * Schüler-, Lehrerausweis, Volltextsuche. Ohne Netz gibt es niemanden zu fragen, und die
 * Theke muss es selbst wissen; sonst laufen die folgenden Bücher auf die falsche Person,
 * und das fällt erst beim Nachbuchen auf.
 *
 * Die Regeln, und warum sie so sind:
 *
 *  - `A-` / `S-` / `L-` sind Ausweise. Vergeben wird nur `A-`; gelesen werden alle drei,
 *    weil es ältere Nummern gibt und Nummern nie recycelt werden. Dieselbe Liste wie im
 *    Server-Switch.
 *  - `B-` ist ein Buch, `LMF-` ebenfalls. Beide gelten auch ohne geladene Barcode-Liste:
 *    Die Vorsilbe ist eindeutig, und eine fehlende Liste darf nicht dazu führen, dass ein
 *    klar erkennbares Buch verworfen wird.
 *  - `G-` ist ein Gerät. Geräte gehören nicht zum Betrieb ohne Netz, sie hängen an einer
 *    Checkliste, die es ohne Netz nicht gibt. Deshalb eine eigene Art mit eigener Meldung
 *    statt eines stillen „unklar".
 *  - Alles andere ist eine nackte Nummer, und die entscheidet die Barcode-Liste. Erst
 *    roh nachschlagen, dann über die Littera-Rückrechnung: Littera druckt die
 *    Mediennummer im Klartext und codiert im Strichcode eine EAN-13 (litteraEtikett.js).
 *    Steht sie drauf, ist es ein Buch; sonst „unklar".
 *
 * Warum „unklar" und nicht „dann eben Ausweis": Ein Schülerausweis des Altbestands
 * liefert gemessen `B97601826457`, ein Buchetikett eine 13-stellige EAN. Wer hier rät,
 * schreibt die nächsten Bücher einem fremden Kind zu. Ein unklarer Scan sperrt deshalb
 * die Zuordnung, bis ein eindeutiger Ausweis kommt.
 */
import { dekodiereLitteraEtikett } from './litteraEtikett.js';
import { ohnePruefzeichen } from './code39Pruefzeichen.js';

/**
 * Vereinheitlicht die Vorsilbe eines Scans — `s-10001` wird zu `S-10001`.
 *
 * Warum das noetig ist: Gespeichert sind die Nummern mit GROSSER Vorsilbe, und der Server
 * schlaegt exakt nach (`WHERE barcode_id = $1`). Wer die Nummer von Hand tippt, weil eine
 * Karte nicht mehr lesbar ist, trifft sonst nichts — weder mit noch ohne Netz. Ein Scanner
 * liest den Aufdruck gross; ein Mensch tippt, wie er will.
 *
 * Warum NICHT einfach alles gross: Jemand sucht nach „s-bahn", und daraus wuerde ein
 * Ausweis „S-bahn". Deshalb greift die Vereinheitlichung nur, wenn hinter der Vorsilbe
 * wirklich eine ZIFFER steht — das trennt eine Nummer von einem Wort mit Bindestrich.
 *
 * @param {string} roh
 * @returns {string}
 */
export function normalisiereScan(roh) {
	const scan = (roh ?? '').trim();
	for (const v of ['LMF-', 'A-', 'S-', 'L-', 'B-', 'G-']) {
		if (
			scan.length > v.length &&
			scan.slice(0, v.length).toUpperCase() === v &&
			/\d/.test(scan[v.length])
		) {
			return v + scan.slice(v.length);
		}
	}
	return scan;
}

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
	const scan = normalisiereScan(roh);
	if (scan === '') return { art: 'unklar', nummer: '' };

	if (AUSWEIS_VORSILBEN.some((v) => scan.startsWith(v))) return { art: 'ausweis', nummer: scan };
	if (BUCH_VORSILBEN.some((v) => scan.startsWith(v))) return { art: 'buch', nummer: scan };
	if (scan.startsWith('G-')) return { art: 'geraet', nummer: scan };

	// Eine Eingabe ist kein Scan: Wer einen Namen tippt, sucht eine Person, und die Suche
	// braucht den Server, denn auf dem Theken-Rechner liegen keine Personendaten. Das
	// verdient eine eigene Auskunft: „steht nicht in der Buchliste" wäre über einen Namen
	// schlicht falsch und hülfe niemandem weiter.
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

	// Zuletzt ein älterer Aufdruck: Code 39 mit Prüfzeichen, das das Lesegerät als Teil der
	// Nummer zurückgibt (aus „58968" wird „58968-"). Diese Karten und Etiketten sind im
	// Umlauf und sollen weiter funktionieren; der Server macht online dasselbe
	// (internal/service/omnibox.go).
	//
	// Zuletzt und nur mit Treffer in der Liste: Bei 43 möglichen Zeichen sieht im Schnitt
	// jeder 43. gültige Code zufällig so aus, als hänge ein Prüfzeichen dran. Erst hier
	// unten, nachdem roh und Littera nichts ergeben haben, ist ein falscher Treffer nur dort
	// möglich, wo die gekürzte Nummer existiert und die volle nicht.
	const ohneZeichen = ohnePruefzeichen(scan);
	if (ohneZeichen !== null && istBuch(ohneZeichen)) return { art: 'buch', nummer: ohneZeichen };

	return { art: 'unklar', nummer: scan };
}
