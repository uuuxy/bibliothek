import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../../lib/apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('./store.svelte.js', () => ({
	appState: { searchQuery: '', bestandsAnsicht: 'mit', adminAuthenticated: true }
}));
import { apiFetch } from '../../lib/apiFetch.js';
import { appState } from './store.svelte.js';
import { holeBuecherListe } from './admin_api.js';
import { serverAntwort } from '../../lib/serverAntwort.js';

// Die Titelverwaltung hat zwei Sichten derselben Liste (docs/OFFEN.md 9.4, 22.09.2026):
// den Katalog und die Aufräumsicht mit den Titeln ohne Exemplar. Die Sicht reist als
// Parameter `bestand=ohne` zur einen Tür GET /api/books — der Katalog ohne Parameter.
describe('admin_api: holeBuecherListe', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, status: 200, json: async () => [] })
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

	// Die Seite zeigt den Grund an; ein Merkwort auf Englisch stünde dort sonst als Meldung.
	it('nennt bei 401 den Satz des Servers und nimmt die Anmeldung zurück', async () => {
		appState.adminAuthenticated = true;
		vi.mocked(apiFetch).mockResolvedValueOnce(
			serverAntwort(401, { error: 'Nicht angemeldet. Bitte neu anmelden.' })
		);

		await expect(holeBuecherListe()).rejects.toThrow('Nicht angemeldet. Bitte neu anmelden.');
		expect(appState.adminAuthenticated).toBe(false);
	});
});
