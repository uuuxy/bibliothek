import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import AnliegenListe from './AnliegenListe.svelte';
import { apiFetch } from '../../apiFetch.js';

vi.mock('../../apiFetch.js', () => ({
	apiFetch: vi.fn(),
	extractApiError: vi.fn(async (/** @type {any} */ res) => JSON.parse(await res.text()).error)
}));
vi.mock('../../stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));

// „Keine offenen Anliegen" ist eine Aussage über die Arbeit der Bibliothek: nichts zu tun.
// Bis zum 12.09.2026 stand derselbe Satz da, wenn der Abruf gescheitert war
// (`anliegen = res.ok ? await res.json() : []`) — die Wünsche und Meldungen des
// Kollegiums blieben dann liegen, ohne dass jemand davon wusste (Register,
// Bestands-Durchgang 10.09.2026).
describe('AnliegenListe', () => {
	beforeEach(() => vi.mocked(apiFetch).mockReset());

	it('unterscheidet „nichts zu tun" von „nicht geladen"', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: false,
				status: 503,
				text: async () => JSON.stringify({ error: 'Datenbank nicht erreichbar' })
			})
		);
		const { findByText, queryByText } = render(AnliegenListe);

		expect(await findByText('Datenbank nicht erreichbar')).toBeTruthy();
		expect(queryByText('Keine offenen Anliegen.')).toBeNull();
	});

	it('leer bleibt leer', async () => {
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: true, json: async () => [] }));
		const { findByText } = render(AnliegenListe);
		expect(await findByText('Keine offenen Anliegen.')).toBeTruthy();
	});
});
