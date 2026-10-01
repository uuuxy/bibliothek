import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render } from '@testing-library/svelte';

vi.mock('$lib/components/scanner/barcode_detector.js', () => ({
	createBarcodeDetector: vi.fn(async () => ({ detector: { detect: async () => [] } }))
}));
vi.mock('./apiFetch.js', () => ({ apiFetch: vi.fn(), apiClient: { post: vi.fn() } }));

import KameraScanner from '../inventur/lib/components/scanner/KameraScanner.svelte';
import WebcamCapture from './WebcamCapture.svelte';

// Die Anwendung bleibt hinter dem Sperrbildschirm stehen und wird träge (inert), also auch
// ein Bauteil mit offener Kamera. Der Strom darf dort nicht weiterlaufen: Ein vorgehaltener
// Barcode würde gelesen, und das Licht der Kamera brennt neben einem gesperrten Bildschirm.
describe('Kamera hinter der Sperre', () => {
	/** @type {{ stop: import('vitest').Mock }[]} */
	let spuren;
	/** @type {import('vitest').Mock} */
	let kamera;

	beforeEach(() => {
		vi.useFakeTimers();
		spuren = [];
		kamera = vi.fn(async () => {
			const spur = { stop: vi.fn() };
			spuren.push(spur);
			return { getTracks: () => [spur] };
		});
		Object.defineProperty(navigator, 'mediaDevices', {
			configurable: true,
			value: { getUserMedia: kamera }
		});
		HTMLMediaElement.prototype.play = vi.fn(async () => {});
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	/** Schaltet den Bereich träge oder wieder frei, wie der Anwendungsrahmen es tut.
	 * @param {HTMLElement} bereich @param {boolean} traege */
	async function stelle(bereich, traege) {
		bereich.toggleAttribute('inert', traege);
		await vi.advanceTimersByTimeAsync(50);
	}

	it.each([
		[
			'der Barcode-Scanner',
			() => render(KameraScanner, { props: { onDecode: vi.fn(), onStatusChange: vi.fn() } })
		],
		[
			'die Aufnahme des Ausweisfotos',
			() =>
				render(WebcamCapture, { props: { studentId: 's1', onCapture: vi.fn(), onClose: vi.fn() } })
		]
	])('%s hält den Strom an und läuft nach dem Aufschließen wieder', async (_name, baue) => {
		const { unmount, container } = baue();
		await vi.advanceTimersByTimeAsync(400);
		expect(kamera, 'die Kamera startet mit dem Bauteil').toHaveBeenCalledTimes(1);
		expect(spuren[0].stop).not.toHaveBeenCalled();

		await stelle(container, true);
		expect(spuren[0].stop, 'im trägen Bereich läuft die Kamera weiter').toHaveBeenCalledTimes(1);
		expect(kamera).toHaveBeenCalledTimes(1);

		await stelle(container, false);
		expect(kamera, 'nach dem Aufschließen bleibt die Kamera aus').toHaveBeenCalledTimes(2);
		expect(spuren[1].stop).not.toHaveBeenCalled();

		unmount();
		expect(spuren[1].stop).toHaveBeenCalled();
	});
});
