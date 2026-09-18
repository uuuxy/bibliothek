import { describe, it, expect, vi, beforeAll } from 'vitest';
import { render, screen, waitFor } from '@testing-library/svelte';
import Suchfeld from './Suchfeld.svelte';
import Suchpille from './Suchpille.svelte';

// Kamera-Scanner in den gemeinsamen Suchbauteilen (18.09.2026): ein Schalter `kamera`,
// Standard aus. Der Test hält beides fest — dass der Knopf mit dem Schalter da ist UND
// dass er ohne ihn fehlt: Die Suchpille steht auf gut einem Dutzend Seiten, und ein
// Kamera-Knopf, der überall auftaucht, wäre genau die Nebenwirkung, die der Schalter
// verhindern soll. Die Kamera selbst wird hier nicht gestartet (jsdom hat keine).
const KNOPF = /Kamera-Barcode-Scanner/;
const PROPS = { wert: '', platzhalter: 'Suchen …', etikett: 'Suche' };

describe('Kamera-Schalter der Suchbauteile', () => {
	it('Suchfeld: Knopf nur mit kamera', () => {
		const mit = render(Suchfeld, { ...PROPS, kamera: true });
		expect(mit.queryByRole('button', { name: KNOPF })).toBeTruthy();
		mit.unmount();
		const ohne = render(Suchfeld, { ...PROPS });
		expect(ohne.queryByRole('button', { name: KNOPF })).toBeNull();
	});

	it('Suchpille: Knopf nur mit kamera', () => {
		const mit = render(Suchpille, { ...PROPS, id: 'p1', kamera: true });
		expect(mit.queryByRole('button', { name: KNOPF })).toBeTruthy();
		mit.unmount();
		const ohne = render(Suchpille, { ...PROPS, id: 'p2' });
		expect(ohne.queryByRole('button', { name: KNOPF })).toBeNull();
	});
});

// Was ein erkannter Code an die Suche übergibt.
//
// Bis zum 18.09.2026 übergab er nichts: Das Bauteil schickte dem Feld ein künstliches
// input-Ereignis, damit die Suche losläuft „als hätte jemand getippt" — und an genau
// diesem Ereignis hängt Svelte die Rückschreibung von bind:value. Die las den noch
// leeren DOM-Wert zurück und löschte den gescannten Code, bevor ihn jemand sehen konnte.
// Am Bildschirm sah das so aus: „ein Fenster öffnet sich eine Millisekunde und dann
// nichts." Deshalb prüft dieser Test nicht, DASS ein Ereignis kommt, sondern WAS es trägt.
const GESCANNT = '9783060130764';

vi.mock('$lib/components/scanner/barcode_detector.js', async (importOriginal) => ({
	...(await importOriginal()),
	createBarcodeDetector: async () => ({
		name: 'test',
		detector: { detect: async () => [{ rawValue: '9783060130764' }] }
	})
}));

beforeAll(() => {
	Object.defineProperty(navigator, 'mediaDevices', {
		configurable: true,
		value: { getUserMedia: async () => ({ getTracks: () => [] }) }
	});
	HTMLMediaElement.prototype.play = vi.fn().mockResolvedValue(undefined);
	Object.defineProperty(HTMLMediaElement.prototype, 'readyState', {
		configurable: true,
		get: () => 2
	});
});

describe('Übergabe eines erkannten Codes', () => {
	it('Suchfeld: das input-Ereignis trägt den gescannten Code', async () => {
		/** @type {string[]} */
		const gesehen = [];
		render(Suchfeld, {
			...PROPS,
			kamera: true,
			oninput: (/** @type {any} */ e) => gesehen.push(e.target.value)
		});
		screen.getByRole('button', { name: KNOPF }).click();
		await waitFor(() => expect(gesehen.at(-1)).toBe(GESCANNT), { timeout: 4000 });
	});

	it('Suchfeld: mit onscan bekommt der Aufrufer den Code statt eines Ereignisses', async () => {
		/** @type {string[]} */
		const gescannt = [];
		render(Suchfeld, {
			...PROPS,
			kamera: true,
			onscan: (/** @type {string} */ code) => gescannt.push(code)
		});
		screen.getByRole('button', { name: KNOPF }).click();
		await waitFor(() => expect(gescannt).toEqual([GESCANNT]), { timeout: 4000 });
	});
});
