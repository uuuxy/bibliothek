import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/svelte';

vi.mock('$lib/components/scanner/barcodeErkennung.js', () => ({
	erzeugeBarcodeErkenner: vi.fn(async () => ({ detector: { detect: async () => [] } }))
}));
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	...(await importOriginal()),
	apiFetch: vi.fn(),
	apiClient: { post: vi.fn() }
}));

import KameraScanner from '../../../inventur/lib/components/scanner/KameraScanner.svelte';
import WebcamAufnahme from './WebcamAufnahme.svelte';

// Verweigert der Browser die Kamera, nennt er den Grund am Namen der Ausnahme und in einem
// englischen Text, je Browser anders. Beide Bauteile mit Kamera sagen es auf Deutsch.
const VERWEIGERT =
	'Der Browser hat den Zugriff auf die Kamera verweigert. Freigeben lässt er sich in den Website-Einstellungen des Browsers.';

describe('Die Kamera wird verweigert', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		Object.defineProperty(navigator, 'mediaDevices', {
			configurable: true,
			value: {
				getUserMedia: vi.fn(async () => {
					throw new DOMException('Permission denied', 'NotAllowedError');
				})
			}
		});
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('der Barcode-Scanner nennt die Verweigerung auf Deutsch', async () => {
		const onStatusChange = vi.fn();
		render(KameraScanner, { props: { onDecode: vi.fn(), onStatusChange } });
		await vi.advanceTimersByTimeAsync(400);

		expect(onStatusChange).toHaveBeenLastCalledWith(VERWEIGERT);
		expect(onStatusChange.mock.calls.flat().join(' ')).not.toContain('Permission denied');
	});

	it('die Aufnahme des Ausweisfotos nennt die Verweigerung auf Deutsch und bietet einen neuen Versuch', async () => {
		render(WebcamAufnahme, { props: { leserKennung: 's1', onCapture: vi.fn(), onClose: vi.fn() } });
		await vi.advanceTimersByTimeAsync(50);

		expect(screen.getByText(VERWEIGERT)).toBeTruthy();
		expect(screen.queryByText(/Permission denied/)).toBeNull();
		expect(screen.getByRole('button', { name: 'Erneut versuchen' })).toBeTruthy();
	});
});
