import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, waitFor } from '@testing-library/svelte';

const detect = vi.fn(async () => {
	throw new Error('Kein Bild lesbar');
});
vi.mock('$lib/components/scanner/barcodeErkennung.js', () => ({
	erzeugeBarcodeErkenner: vi.fn(async () => ({ detector: { detect } }))
}));

import KameraScanner from './KameraScanner.svelte';

// Ein einzelnes misslungenes Bild ist belanglos. Scheitert jedes, meldet der Scanner den Grund
// einmal, nach dem dritten Bild in Folge, und nicht mit jedem Bild.
describe('KameraScanner: misslungene Bilder', () => {
	beforeEach(() => {
		detect.mockClear();
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

	it('meldet den Grund einmal, nach dem dritten misslungenen Bild in Folge', async () => {
		const onStatusChange = vi.fn();
		const ansicht = render(KameraScanner, { props: { onDecode: vi.fn(), onStatusChange } });
		await waitFor(() => expect(detect.mock.calls.length).toBeGreaterThan(5), { timeout: 3000 });

		const meldungen = onStatusChange.mock.calls.filter(([text]) =>
			String(text).startsWith('Erkennung meldet')
		);
		expect(meldungen).toEqual([['Erkennung meldet: Kein Bild lesbar']]);
		ansicht.unmount();
	});
});
