import { describe, it, expect, vi, afterEach } from 'vitest';
import { baueListenDruckHtml, druckeDokument } from './listenDruck.js';

const NUTZLAST = `"><img src="https://fremder-host/?daten=1">`;

afterEach(() => {
	vi.restoreAllMocks();
});

describe('baueListenDruckHtml', () => {
	it('maskiert Überschrift, Meta, Spaltenköpfe, Zellen und Klassennamen', () => {
		const html = baueListenDruckHtml({
			ueberschrift: NUTZLAST,
			meta: NUTZLAST,
			spalten: [NUTZLAST, { text: NUTZLAST, klasse: NUTZLAST }],
			zeilen: [[NUTZLAST, { text: NUTZLAST, klasse: NUTZLAST }]]
		});
		expect(html).not.toContain('<img');
		expect(html).not.toContain('fremder-host/?daten=1"');
		// Fenstertitel, Überschrift, Meta, zwei Köpfe, zwei Zellen, der Klassenname am Kopf
		// und zweimal an der Zelle darunter (ihr eigener und der ihrer Spalte).
		expect(html.match(/&lt;img/g) ?? []).toHaveLength(10);
	});

	it('setzt eine Klasse nur an die Zelle, die eine nennt', () => {
		const html = baueListenDruckHtml({
			ueberschrift: 'Liste',
			meta: '',
			spalten: ['A', 'B'],
			zeilen: [['eins', { text: 'zwei', klasse: 'overdue' }]]
		});
		expect(html).toContain('<tr><td>eins</td><td class="overdue">zwei</td></tr>');
	});

	// Eine schmale Spalte bricht nicht um; das gilt für jede ihrer Zellen, nicht nur den Kopf.
	it('gibt die Klasse eines Spaltenkopfs an die Zellen seiner Spalte weiter', () => {
		const html = baueListenDruckHtml({
			ueberschrift: 'Liste',
			meta: '',
			spalten: [{ text: 'A', klasse: 'schmal' }, 'B'],
			zeilen: [[{ text: 'eins', klasse: 'mono' }, 'zwei']]
		});
		expect(html).toContain('<tr><th class="schmal">A</th><th>B</th></tr>');
		expect(html).toContain('<tr><td class="mono schmal">eins</td><td>zwei</td></tr>');
	});

	it('setzt die Liste nur auf Wunsch eng', () => {
		const liste = { ueberschrift: 'Liste', meta: '', spalten: ['A'], zeilen: [['eins']] };
		expect(baueListenDruckHtml(liste)).toContain('<body>');
		expect(baueListenDruckHtml({ ...liste, dicht: true })).toContain('<body class="dicht">');
	});
});

describe('druckeDokument', () => {
	it('meldet false, wenn der Browser das Fenster nicht öffnet', () => {
		vi.spyOn(window, 'open').mockReturnValue(null);
		expect(druckeDokument('<p>x</p>')).toBe(false);
	});

	it('schreibt das Dokument ins neue Fenster und druckt von hier aus', () => {
		const fenster = {
			document: { open: vi.fn(), write: vi.fn(), close: vi.fn() },
			focus: vi.fn(),
			print: vi.fn()
		};
		vi.spyOn(window, 'open').mockReturnValue(/** @type {any} */ (fenster));

		expect(druckeDokument('<p>x</p>')).toBe(true);
		expect(fenster.document.write).toHaveBeenCalledWith('<p>x</p>');
		expect(fenster.document.close).toHaveBeenCalledTimes(1);
		expect(fenster.print).toHaveBeenCalledTimes(1);
	});
});
