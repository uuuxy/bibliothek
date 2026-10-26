import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';

// Kommt die Liste der fehlenden Etiketten nicht, steht ihr Grund da. „Für alle Exemplare wurden
// Etiketten gedruckt" wäre eine falsche Entwarnung, und eine zweite, allgemeine Meldung neben
// der des Servers (apiGet zeigt sie selbst) verdeckte den Grund.
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiGet: vi.fn(),
	apiPost: vi.fn()
}));
vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

import { apiGet } from '../../apiFetch.js';
import { snackbars } from '../../stores/snackbars.svelte.js';
import EtikettenNachdruck from './EtikettenNachdruck.svelte';

describe('Fehlende Etiketten: die Liste kommt nicht', () => {
	beforeEach(() => {
		vi.mocked(apiGet).mockReset();
		vi.mocked(snackbars.addToast).mockClear();
	});

	it('zeigt den Grund statt „Für alle Exemplare wurden Etiketten gedruckt" und lädt neu', async () => {
		vi.mocked(apiGet).mockRejectedValue(new Error('Die Datenbank antwortet nicht'));
		render(EtikettenNachdruck);

		await vi.waitFor(() => expect(screen.getByText('Exemplare nicht geladen')).toBeTruthy());
		expect(screen.getByText('Die Datenbank antwortet nicht')).toBeTruthy();
		expect(screen.queryByText(/Für alle Exemplare wurden Etiketten gedruckt/)).toBeNull();
		expect(snackbars.addToast, 'eine zweite Meldung neben der des Servers').not.toHaveBeenCalled();

		vi.mocked(apiGet).mockImplementation(async (/** @type {string} */ url) =>
			url.includes('/anzahl') ? { anzahl: 0 } : []
		);
		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));
		await vi.waitFor(() =>
			expect(screen.getByText(/Für alle Exemplare wurden Etiketten gedruckt/)).toBeTruthy()
		);
		expect(screen.queryByText('Exemplare nicht geladen')).toBeNull();
	});
});
