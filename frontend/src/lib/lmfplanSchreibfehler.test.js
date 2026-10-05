import { describe, it, expect, vi, beforeEach } from 'vitest';
import { erzeugePlaner } from './lmfplanPlaner.svelte.js';
import { apiFetch } from './apiFetch.js';
import { showToast } from '../inventur/lib/store.svelte.js';

vi.mock('./apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../inventur/lib/store.svelte.js', () => ({ showToast: vi.fn() }));

// Scheitert ein Schreibweg am Netz oder am Zeitlimit, wirft der Dienst. Der Planer meldet es
// mit dem Grund; ohne die Meldung würde nur der Knopf wieder aktiv.
describe('LMF-Planer: ein Schreibweg scheitert', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(showToast).mockReset();
	});

	it('nennt beim Speichern den Grund und gibt den Knopf wieder frei', async () => {
		vi.mocked(apiFetch).mockRejectedValue(new Error('Zeitlimit überschritten'));
		const planer = erzeugePlaner();

		await planer.speichern();

		expect(showToast).toHaveBeenCalledWith(
			'Speichern fehlgeschlagen: Zeitlimit überschritten',
			'error'
		);
		expect(planer.zustand.speichert).toBe(false);
	});

	it('gibt einen geworfenen Wert, der kein Error ist, als Text aus', async () => {
		vi.mocked(apiFetch).mockRejectedValue('abgebrochen');
		const planer = erzeugePlaner();

		await planer.speichern();

		expect(showToast).toHaveBeenCalledWith('Speichern fehlgeschlagen: abgebrochen', 'error');
	});
});
