import { describe, it, expect, afterEach } from 'vitest';
import { mehrfachauswahl } from './mehrfachauswahl.js';

// M3, Desktop: „Select a range: Shift + click“, „Select all: Control + A, or Cmd + A (Mac)“.
// Die Liste hier hält ihren Zustand wie eine echte: Jedes Kästchen meldet seinen Klick an
// `gewaehlt`, das Kästchen für alle wählt alles oder nichts. jsdom misst nicht; die Listen
// bekommen deshalb einen Kasten, damit sie als sichtbar gelten.

/** @type {(() => void)[]} */
let aufraeumen = [];
afterEach(() => {
	for (const f of aufraeumen.splice(0)) f();
	document.body.innerHTML = '';
});

/** @param {{ anzahl?: number, alle?: boolean }} [o] */
function liste({ anzahl = 5, alle = true } = {}) {
	const el = document.createElement('div');
	el.getClientRects = () => /** @type {DOMRectList} */ (/** @type {unknown} */ ([{}]));
	/** @type {Set<number>} */
	const gewaehlt = new Set();
	const zeilen = Array.from({ length: anzahl }, (_, i) => {
		const k = document.createElement('input');
		k.type = 'checkbox';
		k.addEventListener('change', () => (k.checked ? gewaehlt.add(i) : gewaehlt.delete(i)));
		el.append(k);
		return k;
	});
	if (alle) {
		const kopf = document.createElement('input');
		kopf.type = 'checkbox';
		kopf.dataset.auswahlAlle = '';
		kopf.addEventListener('click', () => {
			const ziel = gewaehlt.size !== anzahl;
			zeilen.forEach((k, i) => {
				k.checked = ziel;
				if (ziel) gewaehlt.add(i);
				else gewaehlt.delete(i);
			});
		});
		el.prepend(kopf);
	}
	document.body.append(el);
	const aktion = mehrfachauswahl(el);
	aufraeumen.push(() => aktion.destroy());
	return { el, zeilen, gewaehlt };
}

/** @param {HTMLElement} k @param {boolean} [umschalt] */
const klick = (k, umschalt = false) =>
	k.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, shiftKey: umschalt }));

/** @param {EventTarget} ziel @param {KeyboardEventInit} [taste] */
function strgA(ziel = document.body, taste = { ctrlKey: true }) {
	const e = new KeyboardEvent('keydown', { key: 'a', bubbles: true, cancelable: true, ...taste });
	ziel.dispatchEvent(e);
	return e.defaultPrevented;
}

const sortiert = (/** @type {Set<number>} */ s) => [...s].sort();

describe('mehrfachauswahl: Umschalt+Klick wählt einen Bereich', () => {
	it('wählt vom zuletzt geklickten bis zum Kästchen mit Umschalt', () => {
		const { zeilen, gewaehlt } = liste();
		klick(zeilen[1]);
		klick(zeilen[3], true);
		expect(sortiert(gewaehlt)).toEqual([1, 2, 3]);
	});

	it('gibt dem Bereich den neuen Zustand des Kästchens, auch rückwärts und beim Abwählen', () => {
		const { zeilen, gewaehlt } = liste();
		klick(zeilen[4]);
		klick(zeilen[0], true);
		expect(sortiert(gewaehlt)).toEqual([0, 1, 2, 3, 4]);
		klick(zeilen[2], true);
		expect(sortiert(gewaehlt)).toEqual([3, 4]);
	});

	it('wählt ohne Umschalt und ohne Anker nur das eine Kästchen', () => {
		const { zeilen, gewaehlt } = liste();
		klick(zeilen[3], true);
		expect(sortiert(gewaehlt)).toEqual([3]);
		klick(zeilen[0]);
		expect(sortiert(gewaehlt)).toEqual([0, 3]);
	});

	it('nimmt Umschalt+Klick auf den Text einer Zeile wie auf ihr Kästchen', () => {
		const { el, zeilen, gewaehlt } = liste();
		for (const k of zeilen) {
			const zeile = document.createElement('label');
			const text = document.createElement('span');
			text.textContent = 'Zeile';
			k.replaceWith(zeile);
			zeile.append(k, text);
		}
		klick(zeilen[0]);
		const text = /** @type {HTMLElement} */ (el.querySelectorAll('span')[3]);
		klick(text, true);
		expect(sortiert(gewaehlt)).toEqual([0, 1, 2, 3]);
	});

	it('lässt gesperrte Kästchen im Bereich aus', () => {
		const { zeilen, gewaehlt } = liste();
		zeilen[2].disabled = true;
		klick(zeilen[0]);
		klick(zeilen[4], true);
		expect(sortiert(gewaehlt)).toEqual([0, 1, 3, 4]);
	});
});

describe('mehrfachauswahl: Strg/Cmd+A wählt alle', () => {
	it('klickt das Kästchen für alle, mit Strg wie mit Cmd', () => {
		const { gewaehlt } = liste();
		expect(strgA()).toBe(true);
		expect(gewaehlt.size).toBe(5);
		const zweite = liste();
		zweite.zeilen[0].focus();
		expect(strgA(zweite.zeilen[0], { metaKey: true })).toBe(true);
		expect(zweite.gewaehlt.size).toBe(5);
	});

	it('wählt bei teilweiser Auswahl alles und hebt eine volle Auswahl nicht auf', () => {
		const { zeilen, gewaehlt } = liste();
		klick(zeilen[1]);
		strgA();
		expect(gewaehlt.size).toBe(5);
		strgA();
		expect(gewaehlt.size).toBe(5);
	});

	it('klickt ohne Kästchen für alle jedes leere Kästchen', () => {
		const { zeilen, gewaehlt } = liste({ alle: false });
		klick(zeilen[2]);
		strgA();
		expect(sortiert(gewaehlt)).toEqual([0, 1, 2, 3, 4]);
	});

	it('lässt Strg+A im Textfeld beim Text und reagiert nicht auf Strg+Umschalt+A', () => {
		const { el, gewaehlt } = liste();
		const feld = document.createElement('input');
		el.append(feld);
		expect(strgA(feld)).toBe(false);
		expect(strgA(document.body, { ctrlKey: true, shiftKey: true })).toBe(false);
		expect(gewaehlt.size).toBe(0);
	});

	it('wählt bei zwei Listen nur die mit dem Fokus, ohne Fokus keine', () => {
		const a = liste();
		const b = liste();
		expect(strgA()).toBe(false);
		b.zeilen[1].focus();
		strgA(b.zeilen[1]);
		expect([a.gewaehlt.size, b.gewaehlt.size]).toEqual([0, 5]);
	});

	it('meint bei offenem Dialog nur die Liste in ihm', () => {
		const seite = liste();
		const dialog = document.createElement('div');
		dialog.setAttribute('aria-modal', 'true');
		document.body.append(dialog);
		expect(strgA()).toBe(false);
		const imDialog = liste();
		dialog.append(imDialog.el);
		strgA();
		expect([seite.gewaehlt.size, imDialog.gewaehlt.size]).toEqual([0, 5]);
	});
});
