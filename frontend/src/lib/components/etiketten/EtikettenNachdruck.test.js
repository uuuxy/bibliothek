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

// Ein Handscanner mit amerikanischer Tastatur tippt an einem deutschen Rechner den Bindestrich
// als ß (FACHKONZEPT §1); wer ein Etikett zum Nachdruck scannt, sucht nach der Nummer.
describe('Fehlende Etiketten: Suche nach einem gescannten Etikett', () => {
	it('liest ein ß hinter der Vorsilbe als Bindestrich', async () => {
		vi.mocked(apiGet).mockReset();
		vi.mocked(apiGet).mockImplementation(async (/** @type {string} */ url) =>
			url.includes('/anzahl') ? { anzahl: 0 } : []
		);
		render(EtikettenNachdruck);
		await vi.waitFor(() => expect(vi.mocked(apiGet)).toHaveBeenCalled());

		await fireEvent.input(screen.getByRole('searchbox', { name: 'Exemplare filtern' }), {
			target: { value: 'Bß00123' }
		});
		await vi.waitFor(() => {
			const urls = vi.mocked(apiGet).mock.calls.map((c) => String(c[0]));
			expect(urls).toContain('/api/exemplare/etiketten-offen?q=B-00123');
		});
	});
});
