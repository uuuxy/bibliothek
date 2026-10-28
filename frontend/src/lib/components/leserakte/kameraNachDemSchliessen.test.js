import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render } from '@testing-library/svelte';

vi.mock('$lib/components/scanner/barcodeErkennung.js', () => ({
	erzeugeBarcodeErkenner: vi.fn(async () => ({ detector: { detect: async () => [] } }))
}));
vi.mock('../../apiFetch.js', () => ({ apiFetch: vi.fn(), apiClient: { post: vi.fn() } }));

import KameraScanner from '../../../inventur/lib/components/scanner/KameraScanner.svelte';
import WebcamAufnahme from './WebcamAufnahme.svelte';

// Der Browser braucht für die Kamera einen Moment, beim ersten Mal auch die Erlaubnis. Wer den
// Dialog in dieser Zeit schließt, darf keine laufende Kamera zurücklassen: Ihr Licht brennt, und
// niemand kann sie mehr anhalten. Die Kamera hier antwortet erst, wenn der Test sie freigibt.
describe('Kamera nach dem Schließen', () => {
	/** @type {{ stop: import('vitest').Mock }[]} */
	let spuren;
	/** @type {(() => void)[]} */
	let freigeben;
	/** @type {import('vitest').Mock} */
	let kamera;

	beforeEach(() => {
		vi.useFakeTimers();
		spuren = [];
		freigeben = [];
		kamera = vi.fn(
			() =>
				new Promise((fertig) => {
					const spur = { stop: vi.fn() };
					spuren.push(spur);
					freigeben.push(() => fertig({ getTracks: () => [spur] }));
				})
		);
		Object.defineProperty(navigator, 'mediaDevices', {
			configurable: true,
			value: { getUserMedia: kamera }
		});
		HTMLMediaElement.prototype.play = vi.fn(async () => {});
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it.each([
		[
			'der Barcode-Scanner',
			() => render(KameraScanner, { props: { onDecode: vi.fn(), onStatusChange: vi.fn() } })
		],
		[
			'die Aufnahme des Ausweisfotos',
			() =>
				render(WebcamAufnahme, {
					props: { leserKennung: 's1', onCapture: vi.fn(), onClose: vi.fn() }
				})
		]
	])('%s hält eine Kamera an, die erst nach dem Schließen kommt', async (_name, baue) => {
		const { unmount } = baue();
		await vi.advanceTimersByTimeAsync(400);
		expect(kamera, 'die Kamera wird angefragt').toHaveBeenCalledTimes(1);

		unmount();
		freigeben[0]();
		await vi.advanceTimersByTimeAsync(50);
		expect(spuren[0].stop, 'die Kamera läuft nach dem Schließen weiter').toHaveBeenCalled();
	});

	it('der Barcode-Scanner fragt keine Kamera an, wenn er vor dem Start geschlossen wird', async () => {
		const { unmount } = render(KameraScanner, {
			props: { onDecode: vi.fn(), onStatusChange: vi.fn() }
		});
		await vi.advanceTimersByTimeAsync(100);
		unmount();
		await vi.advanceTimersByTimeAsync(400);
		expect(kamera, 'die Kamera startet nach dem Schließen').not.toHaveBeenCalled();
	});

	// Sperre während des Starts, Aufschließen, bevor die erste Kamera kommt: Die erste hält an,
	// die zweite läuft. Eine Sperre gegen doppelte Starts darf den Start nach dem Aufschließen
	// nicht verschlucken.
	it('der Barcode-Scanner läuft nach Sperre und Aufschließen mitten im Start', async () => {
		const { container, unmount } = render(KameraScanner, {
			props: { onDecode: vi.fn(), onStatusChange: vi.fn() }
		});
		await vi.advanceTimersByTimeAsync(400);
		expect(kamera).toHaveBeenCalledTimes(1);

		container.toggleAttribute('inert', true);
		await vi.advanceTimersByTimeAsync(50);
		container.toggleAttribute('inert', false);
		await vi.advanceTimersByTimeAsync(50);
		expect(kamera, 'nach dem Aufschließen startet die Kamera neu').toHaveBeenCalledTimes(2);

		freigeben[0]();
		freigeben[1]();
		await vi.advanceTimersByTimeAsync(50);
		expect(spuren[0].stop, 'die Kamera von vor der Sperre läuft weiter').toHaveBeenCalled();
		expect(spuren[1].stop, 'die Kamera nach dem Aufschließen ist aus').not.toHaveBeenCalled();

		unmount();
		expect(spuren[1].stop).toHaveBeenCalled();
	});
});
