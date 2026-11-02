import { describe, it, expect, vi, beforeEach } from 'vitest';

// Jeder Abruf hängt am Recht seiner Route und endet mit der Anmeldung. Das Abo der Nachbuch-Zahl
// fragte sonst nach dem Abmelden für das nächste Konto weiter, auch ohne Theken-Recht.
const beendeNachbuch = vi.fn();
const nachbuch = { init: vi.fn(() => beendeNachbuch) };
vi.mock('./nachbuchMeldungen.svelte.js', () => ({ nachbuchMeldungen: nachbuch }));
vi.mock('./offlineSync.svelte.js', () => ({ offlineSync: { init: vi.fn() } }));
vi.mock('./uiStore.svelte.js', () => ({
	uiStore: {
		fetchPendingReservierungen: vi.fn(),
		fetchOffeneAnliegen: vi.fn(),
		fetchOffeneEtiketten: vi.fn()
	}
}));

const { starteHintergrundAbrufe } = await import('./hintergrundAbrufe.svelte.js');

describe('Hintergrund-Abrufe', () => {
	beforeEach(() => vi.clearAllMocks());

	it('das Aufräumen beendet das Abo der Nachbuch-Zahl', () => {
		const aufraeumen = starteHintergrundAbrufe({
			rolle: 'helfer',
			permissions: ['perform_actions']
		});
		expect(nachbuch.init).toHaveBeenCalledOnce();

		aufraeumen();
		expect(beendeNachbuch).toHaveBeenCalledOnce();
	});

	it('ohne Theken-Recht gibt es kein Abo der Nachbuch-Zahl', () => {
		starteHintergrundAbrufe({ rolle: 'lehrkraft', permissions: ['view_books'] })();
		expect(nachbuch.init).not.toHaveBeenCalled();
	});
});
