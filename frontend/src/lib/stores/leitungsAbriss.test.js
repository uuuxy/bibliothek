import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Der Server beendet die Live-Leitung, sobald die Anmeldung nicht mehr gilt (sse/sse.go). Danach
// zeigt die Anwendung, was am Server gilt: den Sperrbildschirm oder die Anmeldung, und nicht
// nach 25 Sekunden „Keine Verbindung“.

const server = vi.hoisted(() => ({ stand: /** @type {any} */ ({ zustand: 'offen', konto: {} }) }));
vi.mock('./sperreAmServer.js', () => ({ leseSperrzustand: vi.fn(async () => server.stand) }));
const sperre = vi.hoisted(() => ({ vomServerGesperrt: vi.fn() }));
vi.mock('./leerlaufSperre.svelte.js', () => ({ leerlaufSperre: sperre }));
const anmeldung = vi.hoisted(() => ({ sitzungAbgelaufen: vi.fn() }));
vi.mock('./authStore.svelte.js', () => ({ authStore: anmeldung }));

import { verbinde, trenne } from '../liveEvents.js';
import { leseSperrzustand } from './sperreAmServer.js';
import './leitungsAbriss.js';

/** Ein EventSource, dessen Abriss der Test auslöst. */
class FakeEventSource {
	/** @type {FakeEventSource | null} */
	static letzte = null;

	constructor() {
		/** @type {((e: Event) => void) | null} */
		this.onerror = null;
		FakeEventSource.letzte = this;
	}

	addEventListener() {}

	close() {}
}

/**
 * Die Leitung reißt ab, der Server antwortet auf die Frage nach der Anmeldung mit `stand`.
 * Zurück kommt der Test, wenn die Antwort verarbeitet ist.
 * @param {any} stand
 */
async function reisstAb(stand) {
	server.stand = stand;
	verbinde();
	FakeEventSource.letzte?.onerror?.(new Event('error'));
	expect(leseSperrzustand).toHaveBeenCalledTimes(1);
	await vi.mocked(leseSperrzustand).mock.results[0].value;
}

describe('Abriss der Live-Leitung', () => {
	beforeEach(() => {
		vi.stubGlobal('EventSource', FakeEventSource);
		vi.useFakeTimers();
		vi.clearAllMocks();
	});

	afterEach(() => {
		trenne();
		vi.useRealTimers();
		vi.unstubAllGlobals();
	});

	it('gesperrt am Server: der Sperrbildschirm kommt', async () => {
		await reisstAb({ zustand: 'gesperrt', grund: 'hoechstdauer' });

		expect(sperre.vomServerGesperrt).toHaveBeenCalledTimes(1);
		expect(anmeldung.sitzungAbgelaufen).not.toHaveBeenCalled();
	});

	it('abgemeldet am Server: die Anmeldung kommt', async () => {
		await reisstAb({ zustand: 'beendet' });

		expect(anmeldung.sitzungAbgelaufen).toHaveBeenCalledTimes(1);
		expect(sperre.vomServerGesperrt).not.toHaveBeenCalled();
	});

	// Offen heißt: Die Leitung riss aus einem anderen Grund ab, etwa beim Neustart des Servers. Ohne
	// Antwort weiß niemand, was gilt. In beiden Fällen baut liveEvents.js neu auf, und nach 25
	// Sekunden ohne Lebenszeichen steht das Band.
	it.each([
		['offen', { zustand: 'offen', konto: {} }],
		['ohne Antwort', { zustand: 'unbekannt' }]
	])('%s: weder Sperre noch Abmeldung', async (_, stand) => {
		await reisstAb(stand);

		expect(sperre.vomServerGesperrt).not.toHaveBeenCalled();
		expect(anmeldung.sitzungAbgelaufen).not.toHaveBeenCalled();
	});
});
