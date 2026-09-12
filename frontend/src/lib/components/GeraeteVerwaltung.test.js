import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import GeraeteVerwaltung from './GeraeteVerwaltung.svelte';
import { apiFetch } from '../apiFetch.js';

vi.mock('../apiFetch.js', () => ({
	apiFetch: vi.fn(),
	apiClient: { post: vi.fn(), put: vi.fn() },
	extractApiError: vi.fn(async (/** @type {any} */ res) => JSON.parse(await res.text()).error)
}));

vi.mock('../stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));

// „Noch keine Geräte erfasst" heißt: Der Schrank ist leer, leg eins an. Bis zum
// 12.09.2026 stand derselbe Satz da, wenn der Abruf gescheitert war — mit der Einladung,
// Laptops ein zweites Mal anzulegen, die längst im Bestand stehen (Register,
// Bestands-Durchgang 10.09.2026).
describe('GeraeteVerwaltung', () => {
	beforeEach(() => vi.mocked(apiFetch).mockReset());

	it('unterscheidet „Schrank leer" von „nicht geladen"', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: false,
				status: 500,
				text: async () => JSON.stringify({ error: 'Datenbank nicht erreichbar' })
			})
		);
		const { findByText, queryByText } = render(GeraeteVerwaltung);

		expect(await findByText('Datenbank nicht erreichbar')).toBeTruthy();
		expect(queryByText('Noch keine Geräte erfasst.')).toBeNull();
	});

	it('leer bleibt leer', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ data: [] }) })
		);
		const { findByText } = render(GeraeteVerwaltung);
		expect(await findByText('Noch keine Geräte erfasst.')).toBeTruthy();
	});
});
