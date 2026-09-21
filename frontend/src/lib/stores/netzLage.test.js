import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

/**
 * netzLage misst, statt navigator.onLine zu glauben (21.09.2026): Ein Browser kann
 * „offline" melden und den Server trotzdem erreichen. Das Modul liest navigator.onLine
 * beim Laden — jeder Test lädt es deshalb frisch.
 */
describe('netzLage', () => {
	/** @param {boolean} wert */
	const browserMeldet = (wert) =>
		Object.defineProperty(window.navigator, 'onLine', { get: () => wert, configurable: true });

	const lade = async () => (await import('./netzLage.svelte.js')).netzLage;
	/** Lässt die laufende Probe (fetch + then) zu Ende kommen. */
	const probeFertig = () => vi.waitFor(() => expect(fetch).toHaveBeenCalled());

	beforeEach(() => {
		vi.resetModules();
		vi.stubGlobal('fetch', vi.fn());
	});

	afterEach(() => {
		vi.unstubAllGlobals();
		browserMeldet(true);
	});

	it('Browser meldet offline, /health antwortet: nicht offline', async () => {
		browserMeldet(false);
		vi.mocked(fetch).mockResolvedValue(/** @type {any} */ ({ ok: true }));
		const netz = await lade();
		await probeFertig();
		await Promise.resolve();
		expect(netz.offline).toBe(false);
		expect(vi.mocked(fetch).mock.calls[0][0]).toBe('/health');
	});

	it('Browser meldet offline, /health antwortet nicht: offline', async () => {
		browserMeldet(false);
		vi.mocked(fetch).mockRejectedValue(new TypeError('Failed to fetch'));
		const netz = await lade();
		await vi.waitFor(() => expect(netz.offline).toBe(true));
	});

	it('eine Fehlerantwort von /health zählt nicht als erreichbar', async () => {
		browserMeldet(false);
		vi.mocked(fetch).mockResolvedValue(/** @type {any} */ ({ ok: false, status: 502 }));
		const netz = await lade();
		await vi.waitFor(() => expect(netz.offline).toBe(true));
	});

	it('Browser meldet online: es wird nicht gemessen', async () => {
		browserMeldet(true);
		const netz = await lade();
		expect(netz.offline).toBe(false);
		expect(fetch).not.toHaveBeenCalled();
	});

	it('meldet die Rückkehr genau dann, wenn vorher offline war', async () => {
		browserMeldet(true);
		vi.mocked(fetch).mockRejectedValue(new TypeError('Failed to fetch'));
		const netz = await lade();
		const zurueck = vi.fn();
		netz.beiRueckkehr(zurueck);

		window.dispatchEvent(new Event('online'));
		expect(zurueck, 'war nie offline').not.toHaveBeenCalled();

		window.dispatchEvent(new Event('offline'));
		await vi.waitFor(() => expect(netz.offline).toBe(true));
		window.dispatchEvent(new Event('online'));
		expect(netz.offline).toBe(false);
		expect(zurueck).toHaveBeenCalledTimes(1);
	});

	it('eine abgemeldete Rückkehr-Meldung kommt nicht mehr an', async () => {
		browserMeldet(true);
		vi.mocked(fetch).mockRejectedValue(new TypeError('Failed to fetch'));
		const netz = await lade();
		const zurueck = vi.fn();
		netz.beiRueckkehr(zurueck)();

		window.dispatchEvent(new Event('offline'));
		await vi.waitFor(() => expect(netz.offline).toBe(true));
		window.dispatchEvent(new Event('online'));
		expect(zurueck).not.toHaveBeenCalled();
	});

	it('eine vom online-Ereignis überholte Probe schreibt nicht mehr', async () => {
		browserMeldet(true);
		/** @type {(grund: unknown) => void} */
		let scheitere = () => {};
		vi.mocked(fetch).mockReturnValue(
			new Promise((_, reject) => {
				scheitere = reject;
			})
		);
		const netz = await lade();

		window.dispatchEvent(new Event('offline'));
		window.dispatchEvent(new Event('online'));
		scheitere(new TypeError('Failed to fetch'));
		await new Promise((r) => setTimeout(r, 0));
		expect(netz.offline, 'die alte Probe hat das online-Ereignis überschrieben').toBe(false);
	});
});
