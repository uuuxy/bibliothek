import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import BookAkte from './BookAkte.svelte';
import { apiFetch } from './apiFetch.js';
import { appState } from '../inventur/lib/store.svelte.js';
import { authStore } from './stores/authStore.svelte.js';

vi.mock('./apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
	vi.mocked(apiFetch).mockResolvedValue(
		/** @type {any} */ ({ ok: true, status: 200, json: async () => [] })
	);
	// Der Kopf kommt aus der Auswahl der Titel-Verwaltung; die Listen antworten leer.
	appState.selectedBook = /** @type {any} */ ({ id: 'A', title: 'Momo', authors: 'Ende' });
	authStore.currentUser = /** @type {any} */ ({ rolle: 'admin', permissions: ['*'] });
});

// Die Reiter der Akte kommen aus ui/Reiter wie in Bestellwesen und Bestandsbüchern: eine
// benannte Leiste mit der Rolle tablist, ein Reiter je Bereich.
describe('Buchakte: Reiter', () => {
	it('stehen in einer benannten Reiterleiste und wechseln mit einem Klick', async () => {
		const screen = render(BookAkte, { bookId: 'A', onBack: vi.fn() });

		const leiste = await screen.findByRole('tablist', { name: 'Bereiche der Buchakte' });
		const namen = [...leiste.querySelectorAll('[role="tab"]')].map((r) => r.textContent?.trim());
		expect(namen).toEqual(['Ausleiher (0)', 'Exemplare (0)', 'Vormerkungen (0)', 'Historie']);

		const historie = screen.getByRole('tab', { name: 'Historie' });
		await fireEvent.click(historie);
		expect(historie.getAttribute('aria-selected')).toBe('true');
		expect(screen.getByText('Noch keine Ausleihen in der Datenbank vorhanden.')).toBeTruthy();
	});
});
