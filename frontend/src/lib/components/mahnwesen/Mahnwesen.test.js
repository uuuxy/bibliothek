import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import Mahnwesen from './Mahnwesen.svelte';
import { apiFetch, apiGet } from '../../apiFetch.js';
import { bescheideStore } from '../../stores/bescheide.svelte.js';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	...(await importOriginal()),
	apiFetch: vi.fn(),
	apiGet: vi.fn()
}));

// Der Store des Reiters „Schadensersatz" gilt für die Sitzung. Geladen wurde er nur beim ersten
// Besuch des Mahnwesens; danach zeigte der Reiter diesen Stand, auch wenn an einem anderen Platz
// inzwischen eine Forderung gemeldet oder ein Bescheid übergeben war.
describe('Mahnwesen', () => {
	it('lädt den Reiter „Schadensersatz" bei jedem Besuch', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ klassen: [] }) })
		);
		vi.mocked(apiGet).mockResolvedValue([]);

		render(Mahnwesen).unmount();
		await vi.waitFor(() => expect(bescheideStore.geladen).toBe(true));
		vi.mocked(apiGet).mockClear();

		render(Mahnwesen);
		await vi.waitFor(() => expect(apiGet).toHaveBeenCalledWith('/api/bescheide'));
	});
});
