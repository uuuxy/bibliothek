import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { fokusFalle } from './fokusFalle.js';

/** Lässt den Mikrotask durchlaufen, in dem die Falle beim Öffnen den Fokus setzt. */
const nachDemOeffnen = () => Promise.resolve();

/**
 * Ein Dialog mit Knöpfen, eingehängt ins Dokument.
 * @param {string[]} namen
 */
function dialog(namen) {
	const node = document.createElement('div');
	node.setAttribute('role', 'dialog');
	node.tabIndex = -1;
	const knoepfe = namen.map((name) => {
		const knopf = document.createElement('button');
		knopf.textContent = name;
		node.appendChild(knopf);
		return knopf;
	});
	document.body.appendChild(node);
	return { node, knoepfe };
}

/**
 * Drückt Tab auf dem Element, das gerade den Fokus hat.
 * @param {{ shift?: boolean }} [wie]
 * @returns {boolean} hat die Falle die Taste verbraucht?
 */
function tab({ shift = false } = {}) {
	const ereignis = new KeyboardEvent('keydown', {
		key: 'Tab',
		shiftKey: shift,
		bubbles: true,
		cancelable: true
	});
	(document.activeElement ?? document.body).dispatchEvent(ereignis);
	return ereignis.defaultPrevented;
}

beforeEach(() => {
	// jsdom rechnet kein Layout: Ohne die Attrappe gälte jedes Element als unsichtbar,
	// und die Falle fände keinen Kandidaten.
	vi.spyOn(HTMLElement.prototype, 'getClientRects').mockReturnValue(
		/** @type {any} */ ([{ width: 1, height: 1 }])
	);
});

afterEach(() => {
	vi.restoreAllMocks();
	document.body.replaceChildren();
});

describe('fokusFalle', () => {
	it('setzt den Fokus beim Öffnen auf das erste Element im Dialog', async () => {
		const { node, knoepfe } = dialog(['Abbrechen', 'Speichern']);

		const falle = fokusFalle(node);
		await nachDemOeffnen();

		expect(document.activeElement).toBe(knoepfe[0]);
		falle.destroy();
	});

	it('Tab auf dem letzten Element führt zum ersten, Shift+Tab auf dem ersten zum letzten', async () => {
		const { node, knoepfe } = dialog(['Eins', 'Zwei', 'Drei']);
		const falle = fokusFalle(node);
		await nachDemOeffnen();

		knoepfe[2].focus();
		expect(tab()).toBe(true);
		expect(document.activeElement).toBe(knoepfe[0]);

		expect(tab({ shift: true })).toBe(true);
		expect(document.activeElement).toBe(knoepfe[2]);
		falle.destroy();
	});

	// Safari lässt beim Tab Knöpfe aus. Bliebe die Mitte dem Browser, spränge der Fokus vom letzten
	// Feld am Knopf vorbei aus dem Dialog; im Browser-Test mit WebKit gesehen.
	it('geht auch in der Mitte selbst weiter, vor und zurück', async () => {
		const { node, knoepfe } = dialog(['Eins', 'Zwei', 'Drei']);
		const falle = fokusFalle(node);
		await nachDemOeffnen();

		knoepfe[0].focus();
		expect(tab()).toBe(true);
		expect(document.activeElement).toBe(knoepfe[1]);
		expect(tab()).toBe(true);
		expect(document.activeElement).toBe(knoepfe[2]);
		expect(tab({ shift: true })).toBe(true);
		expect(document.activeElement).toBe(knoepfe[1]);
		falle.destroy();
	});

	it('eine Radiogruppe ist ein Halt: der gewählte Knopf', async () => {
		const { node, knoepfe } = dialog(['Davor', 'Danach']);
		const radios = ['a', 'b', 'c'].map((wert) => {
			const radio = document.createElement('input');
			radio.type = 'radio';
			radio.name = 'gruppe';
			radio.value = wert;
			radio.checked = wert === 'b';
			return radio;
		});
		knoepfe[1].before(...radios);
		const falle = fokusFalle(node);
		await nachDemOeffnen();

		knoepfe[0].focus();
		tab();
		expect(document.activeElement).toBe(radios[1]);
		tab();
		expect(document.activeElement).toBe(knoepfe[1]);
		falle.destroy();
	});

	it('lässt eine Taste stehen, die ein Bauteil im Dialog schon verbraucht hat', async () => {
		const { node, knoepfe } = dialog(['Eins', 'Zwei']);
		const falle = fokusFalle(node);
		await nachDemOeffnen();
		knoepfe[0].addEventListener('keydown', (e) => e.preventDefault());

		knoepfe[0].focus();
		tab();

		expect(document.activeElement).toBe(knoepfe[0]);
		falle.destroy();
	});

	it('geht über ein Element hinweg, das den Fokus nicht nimmt', async () => {
		const { node, knoepfe } = dialog(['Eins', 'Zwei', 'Drei']);
		const falle = fokusFalle(node);
		await nachDemOeffnen();
		knoepfe[1].focus = () => {};

		knoepfe[0].focus();
		tab();

		expect(document.activeElement).toBe(knoepfe[2]);
		falle.destroy();
	});

	it('hält den Fokus auf dem Dialog selbst, wenn nichts darin fokussierbar ist', async () => {
		const { node } = dialog([]);
		const falle = fokusFalle(node);
		await nachDemOeffnen();

		expect(tab()).toBe(true);
		expect(document.activeElement).toBe(node);
		falle.destroy();
	});

	// Eine Rückfrage liegt im DOM neben dem fragenden Dialog. Solange sie offen ist, gehört
	// der Fokus ihr; die Falle darunter hält still.
	it('nur die oberste Falle reagiert', async () => {
		const unten = dialog(['Unten 1', 'Unten 2']);
		const untereFalle = fokusFalle(unten.node);
		await nachDemOeffnen();
		const oben = dialog(['Oben 1', 'Oben 2']);
		const obereFalle = fokusFalle(oben.node);
		await nachDemOeffnen();
		expect(document.activeElement, 'die obere Falle nimmt den Fokus').toBe(oben.knoepfe[0]);

		unten.knoepfe[1].focus();
		expect(tab(), 'die untere Falle verbraucht die Taste nicht').toBe(false);
		expect(document.activeElement).toBe(unten.knoepfe[1]);

		oben.knoepfe[1].focus();
		expect(tab()).toBe(true);
		expect(document.activeElement).toBe(oben.knoepfe[0]);

		obereFalle.destroy();
		unten.knoepfe[1].focus();
		expect(tab(), 'nach dem Schließen der oberen ist die untere wieder dran').toBe(true);
		expect(document.activeElement).toBe(unten.knoepfe[0]);
		untereFalle.destroy();
	});

	it('öffnet eine zweite Falle, bevor die erste den Fokus gesetzt hat, bekommt ihn die zweite', async () => {
		const unten = dialog(['Unten']);
		const oben = dialog(['Oben']);
		const untereFalle = fokusFalle(unten.node);
		const obereFalle = fokusFalle(oben.node);
		await nachDemOeffnen();

		expect(document.activeElement).toBe(oben.knoepfe[0]);
		obereFalle.destroy();
		untereFalle.destroy();
	});

	it('gibt den Fokus beim Schließen an den Auslöser zurück', async () => {
		const ausloeser = document.createElement('button');
		document.body.appendChild(ausloeser);
		ausloeser.focus();
		const { node } = dialog(['Schließen']);
		const falle = fokusFalle(node);
		await nachDemOeffnen();

		falle.destroy();

		expect(document.activeElement).toBe(ausloeser);
	});
});
