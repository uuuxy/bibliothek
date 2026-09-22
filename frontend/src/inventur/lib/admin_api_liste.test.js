import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../../lib/apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('./store.svelte.js', () => ({
	appState: { searchQuery: '', bestandsAnsicht: 'mit', adminAuthenticated: true }
}));
import { apiFetch } from '../../lib/apiFetch.js';
import { appState } from './store.svelte.js';
import { holeBuecherListe } from './admin_api.js';

// Die Titelverwaltung hat zwei Sichten derselben Liste (docs/OFFEN.md 9.4, 22.09.2026):
// den Katalog und die Aufräumsicht mit den Titeln ohne Exemplar. Die Sicht reist als
// Parameter `bestand=ohne` zur einen Tür GET /api/books — der Katalog ohne Parameter.
describe('admin_api: holeBuecherListe', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, status: 200, json: async () => ({ data: [] }) })
		);
		appState.searchQuery = '';
		appState.bestandsAnsicht = 'mit';
	});

	it('der Katalog fragt die Tür ohne Sicht-Parameter', async () => {
		await holeBuecherListe();
		expect(vi.mocked(apiFetch).mock.calls[0][0]).toBe('/api/books');
	});

	it('die Aufräumsicht reicht bestand=ohne mit, neben der Suche', async () => {
		appState.bestandsAnsicht = 'ohne';
		appState.searchQuery = 'Mathe 9';
		await holeBuecherListe();
		expect(vi.mocked(apiFetch).mock.calls[0][0]).toBe('/api/books?q=Mathe+9&bestand=ohne');
	});
});
