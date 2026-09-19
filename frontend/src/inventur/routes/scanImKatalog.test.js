import { describe, it, expect, vi, beforeAll } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import Katalogseite from './+page.svelte';

/**
 * Der ganze Weg eines Kamera-Scans im Medienkatalog, an einem Stück: KameraScanner →
 * CameraScanner → Suchpille → StartseitenFilter → Seite → Kachelraster.
 *
 * Grund für diesen Test (Rückmeldung 18.09.2026): „wenn ich bei medienkatalog die kamera
 * verwende passiert nichts — ein Fenster öffnet sich eine Millisekunde und dann nichts."
 * Genau das steht hier: gescannt wird die EAN vom Buchrücken, im Bestand steht dieselbe
 * ISBN mit Bindestrichen. Das Raster hat keinen Leer-Zustand — „nichts gefunden" sieht
 * deshalb aus wie „nichts passiert".
 *
 * Ersetzt wird NUR der Erkenner (er braucht eine Kamera) und das Laden der Bücher. Der
 * Weg dazwischen läuft echt; er ist die Frage.
 */
const GESCANNT = '9783060130764';

vi.mock('$lib/components/scanner/barcode_detector.js', async (importOriginal) => ({
	...(await importOriginal()),
	createBarcodeDetector: async () => ({
		name: 'test',
		detector: { detect: async () => [{ rawValue: '9783060130764' }] }
	})
}));

vi.mock('$lib/startseiten_api.js', async (importOriginal) => ({
	...(await importOriginal()),
	buecherLaden: async () => [
		{ id: 'b1', title: 'Green Line 3', isbn: '978-3-06-013076-4', author: 'Cornelsen' },
		{ id: 'b2', title: 'Ganz anderes Buch', isbn: '9783127335712', author: 'Klett' }
	]
}));

beforeAll(() => {
	// Lücken von jsdom, die nur dieser Weg braucht: eine Kamera und ein Video, das spielt.
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

describe('Kamera-Scan im Medienkatalog', () => {
	it('legt den gescannten Code ins Suchfeld und zeigt den Titel mit dieser ISBN', async () => {
		render(Katalogseite);
		const feld = /** @type {HTMLInputElement} */ (
			await screen.findByLabelText(/Suchen nach Titel/)
		);

		screen.getByRole('button', { name: /Kamera-Barcode-Scanner/ }).click();
		// Erste Hälfte: Der Code kommt überhaupt an — über drei Bindungen hinweg.
		await waitFor(() => expect(feld.value).toBe(GESCANNT), { timeout: 4000 });

		// Zweite Hälfte, und die eigentliche Beschwerde: Es muss auch etwas zu sehen sein.
		// Im Bestand steht „978-3-06-013076-4", der Scanner liefert Ziffern ohne Trenner.
		expect(screen.getByText('Green Line 3')).toBeTruthy();
		expect(screen.queryByText('Ganz anderes Buch')).toBeNull();
	});

	it('sagt es, wenn zum gesuchten Code nichts im Bestand steht', async () => {
		render(Katalogseite);
		const feld = /** @type {HTMLInputElement} */ (
			await screen.findByLabelText(/Suchen nach Titel/)
		);

		// Ohne diese Meldung sähe „nicht im Bestand" aus wie „die Kamera hat nichts gelesen":
		// Das Kachelraster wäre schlicht leer.
		await fireEvent.input(feld, { target: { value: '9781234567897' } });

		expect(await screen.findByText('Keine Bücher gefunden')).toBeTruthy();
		expect(screen.getByText(/9781234567897/)).toBeTruthy();
		expect(screen.queryByText('Green Line 3')).toBeNull();
	});
});
