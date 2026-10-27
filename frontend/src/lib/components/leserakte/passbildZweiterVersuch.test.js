import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiClient: { post: vi.fn() }
}));

import { apiClient } from '../../apiFetch.js';
import { serverAntwort } from '../../serverAntwort.js';
import WebcamAufnahme from './WebcamAufnahme.svelte';

// Scheiterte das Speichern des Passbilds, holte „Erneut versuchen" eine zweite Kamera, ohne die
// erste anzuhalten. Die erste lief danach weiter, auch nach dem Schließen des Dialogs, in
// Chromium, Firefox und WebKit (OFFEN.md 5.106).
describe('Passbild: der zweite Versuch nach einem gescheiterten Speichern', () => {
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
		vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(
			/** @type {any} */ ({ drawImage: vi.fn() })
		);
		vi.spyOn(HTMLCanvasElement.prototype, 'toDataURL').mockReturnValue('data:image/jpeg;base64,AA');
	});

	afterEach(() => {
		vi.useRealTimers();
		vi.restoreAllMocks();
	});

	it('hält die erste Kamera an, bevor es die zweite holt', async () => {
		vi.mocked(apiClient.post)
			.mockRejectedValueOnce(new Error('Keine Verbindung zum Server.'))
			.mockResolvedValueOnce(serverAntwort(200, { url: '/api/schueler/s1/photo' }));
		const onCapture = vi.fn();
		const { unmount } = render(WebcamAufnahme, {
			props: { leserKennung: 's1', onCapture, onClose: vi.fn() }
		});
		await vi.advanceTimersByTimeAsync(50);

		await fireEvent.click(screen.getByRole('button', { name: 'Foto aufnehmen' }));
		await vi.advanceTimersByTimeAsync(50);
		expect(screen.getByText('Aufnahme fehlgeschlagen: Keine Verbindung zum Server.')).toBeTruthy();

		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));
		await vi.advanceTimersByTimeAsync(50);
		expect(kamera).toHaveBeenCalledTimes(2);
		expect(spuren[0].stop, 'die erste Kamera läuft neben der zweiten weiter').toHaveBeenCalled();

		await fireEvent.click(screen.getByRole('button', { name: 'Foto aufnehmen' }));
		await vi.advanceTimersByTimeAsync(50);
		expect(onCapture).toHaveBeenCalledTimes(1);

		unmount();
		expect(spuren[1].stop).toHaveBeenCalled();
	});
});
