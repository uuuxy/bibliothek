import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('../apiFetch.js', () => ({
	apiGet: vi.fn(async () => []),
	apiPost: vi.fn(async () => ({})),
	apiPut: vi.fn(async () => ({})),
	apiDelete: vi.fn(async () => ({}))
}));
vi.mock('./toastStore.svelte.js', () => ({
	toastStore: { addToast: vi.fn() }
}));

/**
 * Frische Store-Instanz je Test. Nötig, weil orderStore ein Singleton mit privatem
 * Ladezeitpunkt ist — genau der Zustand, um den es hier geht, liesse sich von aussen
 * sonst nicht zurücksetzen.
 */
async function frischerStore() {
	vi.resetModules();
	const { apiGet } = await import('../apiFetch.js');
	vi.mocked(apiGet).mockClear();
	const { orderStore } = await import('./orderStore.svelte.js');
	return { store: orderStore, apiGet: vi.mocked(apiGet) };
}

describe('orderStore.init – Ladeverhalten', () => {
	beforeEach(() => {
		vi.useFakeTimers();
	});
	afterEach(() => {
		vi.useRealTimers();
	});

	it('lädt beim ersten Aufruf alle drei Datenquellen', async () => {
		const { store, apiGet } = await frischerStore();

		await store.init();

		expect(apiGet).toHaveBeenCalledTimes(3);
		expect(apiGet).toHaveBeenCalledWith('/api/bestellungen');
	});

	// Der Kern: BestellWorkspace wird bei JEDEM Tab-Wechsel neu gemountet
	// (Router nutzt {#if}/{:else if}). Vorher lud jeder dieser Wechsel den kompletten
	// Bestellbedarf neu — die grösste Liste der Anwendung. Genau das war der Hänger.
	it('lädt beim erneuten Mount nicht noch einmal, solange die Daten frisch sind', async () => {
		const { store, apiGet } = await frischerStore();

		await store.init();
		apiGet.mockClear();

		await store.init(); // Rückkehr auf den Tab

		expect(apiGet).not.toHaveBeenCalled();
	});

	it('frischt nach Ablauf der Frist wieder auf', async () => {
		const { store, apiGet } = await frischerStore();

		await store.init();
		apiGet.mockClear();

		vi.advanceTimersByTime(61_000);
		await store.init();
		await vi.waitFor(() => expect(apiGet).toHaveBeenCalledTimes(3));
	});

	// Ein Fehlschlag ist keine Frische: Wer offline den Tab öffnete, soll beim nächsten
	// Mount sofort einen neuen Versuch bekommen — nicht 60 Sekunden leere Listen.
	it('cacht einen fehlgeschlagenen Ladeversuch nicht', async () => {
		const { store, apiGet } = await frischerStore();
		apiGet.mockRejectedValue(new Error('offline'));

		await store.init(); // erster Versuch scheitert
		apiGet.mockClear();
		apiGet.mockResolvedValue([]);

		await store.init(); // direkt danach — muss erneut laden

		expect(apiGet).toHaveBeenCalledTimes(3);
	});

	// Veraltete Daten dürfen die Ansicht nicht blockieren: init() kehrt sofort zurück,
	// der Abgleich läuft daneben. Andernfalls hätte der Cache den Hänger nur verschoben.
	it('blockiert beim Auffrischen nicht auf dem Netz', async () => {
		const { store, apiGet } = await frischerStore();
		await store.init();

		let aufgeloest = false;
		apiGet.mockImplementation(
			() =>
				new Promise((resolve) => {
					setTimeout(() => {
						aufgeloest = true;
						resolve([]);
					}, 5_000);
				})
		);

		vi.advanceTimersByTime(61_000);
		await store.init();

		// init() ist zurück, obwohl der Abruf noch läuft.
		expect(aufgeloest).toBe(false);
	});
});
