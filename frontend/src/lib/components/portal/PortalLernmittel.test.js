import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import PortalLernmittel from './PortalLernmittel.svelte';
import { apiFetch } from '../../apiFetch.js';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

// Kommen die Klassensätze nicht, sagt der Reiter es und lädt mit „Erneut versuchen“ noch einmal,
// statt die ganze Seite neu laden zu lassen.
describe('PortalLernmittel: die Klassensätze kommen nicht', () => {
	it('lädt mit „Erneut versuchen“ noch einmal', async () => {
		vi.mocked(apiFetch)
			.mockResolvedValueOnce(/** @type {any} */ ({ ok: false, status: 500 }))
			.mockResolvedValue(/** @type {any} */ ({ ok: true, status: 200, json: async () => [] }));
		const screen = render(PortalLernmittel);
		expect(await screen.findByText(/Klassensätze konnten nicht geladen werden/)).toBeTruthy();

		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));

		expect(await screen.findByText('Noch keine Klassensätze zugeordnet')).toBeTruthy();
		expect(apiFetch).toHaveBeenCalledTimes(2);
	});
});
