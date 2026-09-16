import { describe, it, expect, vi, beforeEach } from 'vitest';

// Der Zähler gehört jeder Theken-Rolle, die Liste nur `view_students` — beides prüft der
// Server. Hier steht, was der Browser daraus macht: dass er die WIRKUNG liest (die Zeile
// verschwindet erst, wenn der Server zugestimmt hat), dass ein 403 in einem Satz steht,
// den ein Mensch versteht, und dass die Zahl auf das Live-Signal hin neu geholt wird.
//
// Bis zum 16.09.2026 gab es diese Tür im Browser gar nicht: Der Server schrieb Meldungen,
// die niemand zu sehen bekam (Bugklasse „Nie verdrahtet", OFFEN.md Abschnitt 2).

/** @type {{ handler: null | (() => void) }} */
const sse = { handler: null };
vi.mock('../liveEvents.js', () => ({
	abonniere: (_name, cb) => {
		sse.handler = cb;
		return () => (sse.handler = null);
	}
}));
const fetchMock = vi.fn();
const postMock = vi.fn();
vi.mock('../apiFetch.js', () => ({
	apiFetch: (...args) => fetchMock(...args),
	apiClient: { post: (...args) => postMock(...args) }
}));

const { nachbuchMeldungen: m } = await import('./nachbuchMeldungen.svelte.js');

/** @param {any} daten @param {number} status */
const antwort = (daten, status = 200) => ({ ok: status < 400, status, json: async () => daten });

const zeile = (id, ergebnis = 'umgebucht') => ({
	id,
	barcode: 'B-1',
	titel: 'Momo',
	ergebnis,
	gescannt_am: '2026-09-16T10:00:00Z'
});

describe('Meldungen aus dem Nachbuchen', () => {
	beforeEach(() => {
		m._zuruecksetzen();
		vi.clearAllMocks();
	});

	it('holt die Zahl fürs Band', async () => {
		fetchMock.mockResolvedValue(antwort({ offen: 3 }));
		await m.zaehle();
		expect(m.offen).toBe(3);
	});

	it('lässt die Zahl stehen, wenn der Zähler nicht erreichbar ist', async () => {
		fetchMock.mockResolvedValue(antwort({ offen: 2 }));
		await m.zaehle();
		fetchMock.mockRejectedValue(new TypeError('Failed to fetch'));
		await m.zaehle();
		expect(m.offen, 'eine alte Zahl ist besser als eine falsche 0').toBe(2);
	});

	it('sagt bei fehlendem Recht, woran es liegt', async () => {
		fetchMock.mockResolvedValue(antwort({}, 403));
		await m.lade();
		expect(m.fehler).toMatch(/Recht/);
		expect(m.liste).toHaveLength(0);
	});

	it('nimmt die Zahl aus der geladenen Liste — Band und Liste zeigen dasselbe', async () => {
		fetchMock.mockResolvedValue(antwort([zeile('a'), zeile('b')]));
		await m.lade();
		expect(m.liste).toHaveLength(2);
		expect(m.offen).toBe(2);
	});

	// Erledigt ist, was der Server entschieden hat: Erst danach wird neu geladen.
	it('quittiert und holt die Liste danach neu', async () => {
		postMock.mockResolvedValue(antwort({ status: 'ok' }));
		fetchMock.mockResolvedValue(antwort([zeile('b')]));

		await m.quittiere('a');

		expect(postMock).toHaveBeenCalledWith('/api/action/nachbuch-meldungen/a/quittieren', {});
		expect(m.liste.map((z) => z.id)).toEqual(['b']);
		expect(m.fehler).toBe('');
	});

	// Ein anderer Arbeitsplatz war schneller: Das ist kein Fehler dieses Bedieners.
	it('behandelt eine schon quittierte Meldung (404) nicht als Fehler', async () => {
		postMock.mockResolvedValue(antwort({}, 404));
		fetchMock.mockResolvedValue(antwort([]));

		await m.quittiere('a');

		expect(m.fehler).toBe('');
		expect(m.liste).toHaveLength(0);
	});

	it('meldet einen echten Fehlschlag beim Quittieren', async () => {
		postMock.mockResolvedValue(antwort({}, 500));
		await m.quittiere('a');
		expect(m.fehler).toMatch(/500/);
	});

	// Der zweite Arbeitsplatz zeigte sonst die alte Zahl bis zur nächsten Anmeldung.
	it('holt die Zahl neu, wenn der Server eine Änderung meldet', async () => {
		fetchMock.mockResolvedValue(antwort({ offen: 0 }));
		m.init();
		expect(sse.handler, 'ohne Abo bleibt das Band stehen').not.toBeNull();

		fetchMock.mockResolvedValue(antwort({ offen: 5 }));
		sse.handler?.();
		await vi.waitFor(() => expect(m.offen).toBe(5));
	});
});
