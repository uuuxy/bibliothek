import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, waitFor } from '@testing-library/svelte';

let erkannt = '';
const detect = vi.fn(async () => (erkannt ? [{ rawValue: erkannt }] : []));
vi.mock('$lib/components/scanner/barcodeErkennung.js', () => ({
	erzeugeBarcodeErkenner: vi.fn(async () => ({ detector: { detect } }))
}));

import KameraScanner from './KameraScanner.svelte';

// Die Kamera sieht denselben Strichcode in vielen Bildern hintereinander. Gemeldet wird er nur
// einmal, sonst würde an der Theke ein Buch ausgeliehen und im nächsten Bild zurückgegeben. Ein
// anderer Code zählt sofort, derselbe erst nach der Pause von drei Sekunden wieder.
describe('KameraScanner: ein Code wird einmal gemeldet', () => {
	let jetzt = 1_000_000;

	beforeEach(() => {
		erkannt = '';
		detect.mockClear();
		vi.spyOn(Date, 'now').mockImplementation(() => jetzt);
		Object.defineProperty(navigator, 'mediaDevices', {
			configurable: true,
			value: { getUserMedia: vi.fn(async () => ({ getTracks: () => [{ stop: vi.fn() }] })) }
		});
		HTMLMediaElement.prototype.play = vi.fn(async () => {});
		vi.spyOn(HTMLMediaElement.prototype, 'readyState', 'get').mockReturnValue(4);
		vi.stubGlobal('requestAnimationFrame', (/** @type {FrameRequestCallback} */ f) =>
			setTimeout(() => f(0), 0)
		);
	});

	afterEach(() => {
		vi.restoreAllMocks();
		vi.unstubAllGlobals();
	});

	/** Wartet, bis die Erkennung noch einige Bilder gesehen hat. */
	async function weitereBilder() {
		const stand = detect.mock.calls.length;
		await waitFor(() => expect(detect.mock.calls.length).toBeGreaterThan(stand + 2), {
			timeout: 2000
		});
	}

	it('meldet denselben Code einmal, einen anderen sofort, denselben nach der Pause wieder', async () => {
		const onDecode = vi.fn();
		render(KameraScanner, { props: { onDecode, onStatusChange: vi.fn() } });
		await waitFor(() => expect(detect).toHaveBeenCalled(), { timeout: 2000 });

		erkannt = 'B-1';
		await weitereBilder();
		expect(onDecode.mock.calls).toEqual([['B-1']]);

		erkannt = 'B-2';
		await weitereBilder();
		expect(onDecode.mock.calls).toEqual([['B-1'], ['B-2']]);

		jetzt += 3001;
		await weitereBilder();
		expect(onDecode.mock.calls).toEqual([['B-1'], ['B-2'], ['B-2']]);
	});
});
