import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

/** @type {Map<string, () => void>} */
const zuhoerer = new Map();
vi.mock('../liveEvents.js', () => ({
	abonniere: (/** @type {string} */ name, /** @type {() => void} */ handler) => {
		zuhoerer.set(name, handler);
		return () => zuhoerer.delete(name);
	}
}));

const { Programmstand, KOPF_OBERFLAECHENSTAND } = await import('./programmstand.svelte.js');

const ALT = '/assets/index-alt111.js';
const NEU = '/assets/index-neu222.js';

/** @param {string} stand */
function antwortMit(stand) {
	return new Response('{}', { headers: { [KOPF_OBERFLAECHENSTAND]: stand } });
}

// Nach einem Update läuft ein offenes Fenster mit dem alten Programm weiter. Es muss das merken.
describe('programmstand', () => {
	beforeEach(() => zuhoerer.clear());
	afterEach(() => vi.unstubAllGlobals());

	it('ist veraltet, sobald der Server einen anderen Stand nennt', () => {
		const stand = new Programmstand(() => {});
		stand.starte(ALT);
		stand.merkeAus(antwortMit(ALT));
		expect(stand.veraltet).toBe(false);
		stand.merkeAus(antwortMit(NEU));
		expect(stand.veraltet).toBe(true);
	});

	it('ohne eigenen Stand (Entwicklungsserver) oder ohne Kopf in der Antwort nie veraltet', () => {
		const ohneEigenen = new Programmstand(() => {});
		ohneEigenen.starte('');
		ohneEigenen.merkeAus(antwortMit(NEU));
		expect(ohneEigenen.veraltet).toBe(false);
		expect(zuhoerer.size).toBe(0);

		const ohneKopf = new Programmstand(() => {});
		ohneKopf.starte(ALT);
		ohneKopf.merkeAus(new Response('{}'));
		expect(ohneKopf.veraltet).toBe(false);
	});

	it('eine Antwort ohne Kopf löscht den bekannten Stand des Servers nicht', () => {
		const stand = new Programmstand(() => {});
		stand.starte(ALT);
		stand.merkeAus(antwortMit(NEU));
		stand.merkeAus(new Response('{}'));
		expect(stand.veraltet).toBe(true);
	});

	it('fragt den Server, sobald die Live-Leitung steht, nicht erst bei der nächsten Anfrage', async () => {
		const fetch = vi.fn(async () => antwortMit(NEU));
		vi.stubGlobal('fetch', fetch);
		const stand = new Programmstand(() => {});
		stand.starte(ALT);

		zuhoerer.get('connected')?.();
		await vi.waitFor(() => expect(stand.veraltet).toBe(true));
		expect(fetch).toHaveBeenCalledWith('/health', { cache: 'no-store' });
	});

	it('ohne Antwort des Servers bleibt es beim bekannten Stand', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => {
				throw new TypeError('Failed to fetch');
			})
		);
		const stand = new Programmstand(() => {});
		stand.starte(ALT);
		await stand.frageServer();
		expect(stand.veraltet).toBe(false);
	});
});
