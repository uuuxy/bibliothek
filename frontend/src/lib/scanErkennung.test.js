import { describe, it, expect, vi } from 'vitest';
import {
	erzeugeScanErkennung,
	scanSchutz,
	SCAN_HOECHSTABSTAND_MS,
	SCAN_MINDESTZEICHEN
} from './scanErkennung.js';

/**
 * Tippt eine Folge in die Erkennung und liefert, was sie zum Enter sagt.
 * @param {ReturnType<typeof erzeugeScanErkennung>} erkennung
 * @param {{ key: string, nach: number, repeat?: boolean, ctrlKey?: boolean }[]} folge  Abstand zur vorigen Taste in ms
 * @param {number} [start]
 */
function tippe(erkennung, folge, start = 1000) {
	let zeit = start;
	let ergebnis = false;
	for (const t of folge) {
		zeit += t.nach;
		ergebnis = erkennung.taste(
			/** @type {any} */ ({
				key: t.key,
				timeStamp: zeit,
				repeat: !!t.repeat,
				ctrlKey: !!t.ctrlKey,
				metaKey: false,
				altKey: false
			})
		);
	}
	return ergebnis;
}

/** @param {string} text @param {number} abstand */
const zeichen = (text, abstand) => [...text].map((key) => ({ key, nach: abstand }));
const ENTER = (/** @type {number} */ nach) => ({ key: 'Enter', nach });

// Am Sperrbildschirm steht der Fokus im Passwortfeld. Ein Scan ging als Passwort zum Server,
// jeder zählte als Fehlversuch, nach fünf war das Konto an diesem Rechner 15 Minuten gesperrt.
describe('scanErkennung: Scanner oder Mensch', () => {
	it('die Grenzen stehen fest', () => {
		expect([SCAN_MINDESTZEICHEN, SCAN_HOECHSTABSTAND_MS]).toEqual([6, 50]);
	});

	it.each([
		['Etikett eines Buchs', 'B-00123'],
		['EAN eines Littera-Etiketts', '5896800039556'],
		['Ausweis mit Großbuchstaben', 'A-104711']
	])('%s, vom Scanner getippt: Scan', (_name, nummer) => {
		expect(tippe(erzeugeScanErkennung(), [...zeichen(nummer, 5), ENTER(5)])).toBe(true);
	});

	it('ein Scanner mit 40 ms je Zeichen: Scan', () => {
		expect(tippe(erzeugeScanErkennung(), [...zeichen('5896800039556', 40), ENTER(40)])).toBe(true);
	});

	it('Umschalttasten zwischen den Zeichen unterbrechen den Scan nicht', () => {
		const folge = [
			{ key: 'Shift', nach: 2 },
			{ key: 'B', nach: 2 },
			{ key: 'Shift', nach: 2 },
			...zeichen('-00123', 4),
			ENTER(4)
		];
		expect(tippe(erzeugeScanErkennung(), folge)).toBe(true);
	});

	it('ein Mensch, der schnell tippt: kein Scan', () => {
		expect(tippe(erzeugeScanErkennung(), [...zeichen('geheimesPasswort', 90), ENTER(120)])).toBe(
			false
		);
	});

	it('schnell getippt, aber das Enter kommt später: kein Scan', () => {
		expect(tippe(erzeugeScanErkennung(), [...zeichen('B-00123', 5), ENTER(400)])).toBe(false);
	});

	it('weniger Zeichen als die kürzeste Nummer: kein Scan', () => {
		expect(tippe(erzeugeScanErkennung(), [...zeichen('12345', 5), ENTER(5)])).toBe(false);
	});

	it('eingefügtes oder vom Browser ausgefülltes Passwort (kein Tastendruck) und Enter: kein Scan', () => {
		expect(tippe(erzeugeScanErkennung(), [ENTER(0)])).toBe(false);
	});

	it('eine gehaltene Taste: kein Scan', () => {
		const gehalten = [...'aaaaaaaa'].map((key) => ({ key, nach: 30, repeat: true }));
		expect(tippe(erzeugeScanErkennung(), [...gehalten, ENTER(30)])).toBe(false);
	});

	it('ein Kürzel in der Folge bricht sie ab', () => {
		const folge = [
			...zeichen('B-00', 5),
			{ key: 'v', nach: 5, ctrlKey: true },
			...zeichen('123', 5)
		];
		expect(tippe(erzeugeScanErkennung(), [...folge, ENTER(5)])).toBe(false);
	});

	it('erst langsam getippt, dann ein Scan hinterher: Scan', () => {
		const folge = [...zeichen('ab', 300), { key: 'B', nach: 2000 }, ...zeichen('-00123', 5)];
		expect(tippe(erzeugeScanErkennung(), [...folge, ENTER(5)])).toBe(true);
	});

	it('nach einem Scan beginnt die Zählung neu', () => {
		const erkennung = erzeugeScanErkennung();
		expect(tippe(erkennung, [...zeichen('B-00123', 5), ENTER(5)])).toBe(true);
		expect(tippe(erkennung, [ENTER(5)], 2000)).toBe(false);
	});
});

describe('scanSchutz: das Enter eines Scans gilt nicht', () => {
	/** @param {HTMLElement} ziel @param {string} key @param {number} zeit */
	function druecke(ziel, key, zeit) {
		const e = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true });
		Object.defineProperty(e, 'timeStamp', { value: zeit });
		ziel.dispatchEvent(e);
		return e;
	}

	it('hält das Enter an, bevor Feld und Formular es bekommen, und ruft beiScan', () => {
		const form = document.createElement('form');
		const feld = document.createElement('input');
		form.append(feld);
		document.body.append(form);
		const beiScan = vi.fn();
		const amFeld = vi.fn();
		feld.addEventListener('keydown', amFeld);
		const aktion = scanSchutz(form, beiScan);

		// Halb getipptes Passwort, dann der Scan: Der Browser setzt jedes Zeichen nach seinem
		// Tastendruck ein.
		feld.value = 'gehe';
		const eingaben = vi.fn();
		feld.addEventListener('input', eingaben);
		[...'B-00123'].forEach((key, i) => {
			druecke(feld, key, 1000 + i * 5);
			feld.value += key;
		});
		const enter = druecke(feld, 'Enter', 1040);
		expect(beiScan).toHaveBeenCalledTimes(1);
		expect(enter.defaultPrevented).toBe(true);
		expect(amFeld).toHaveBeenCalledTimes(7);
		expect(feld.value).toBe('gehe');
		expect(eingaben).toHaveBeenCalledTimes(1);

		// Großbuchstabe mitten im Scan: Die Umschalttaste beginnt keine neue Folge, sonst
		// bliebe das erste Zeichen des Scans im Feld stehen.
		druecke(feld, 'a', 2000);
		feld.value += 'a';
		druecke(feld, 'Shift', 2003);
		[...'B-00123'].forEach((key, i) => {
			druecke(feld, key, 2006 + i * 5);
			feld.value += key;
		});
		druecke(feld, 'Enter', 2045);
		expect(beiScan).toHaveBeenCalledTimes(2);
		expect(feld.value).toBe('gehe');
		amFeld.mockClear();

		const getippt = druecke(feld, 'Enter', 5000);
		expect(getippt.defaultPrevented).toBe(false);
		expect(amFeld).toHaveBeenCalledTimes(1);

		aktion.destroy();
		form.remove();
	});
});
