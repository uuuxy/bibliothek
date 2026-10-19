import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { flushSync } from 'svelte';

vi.mock('./bestaetigung.svelte.js', () => ({ verwerfenBestaetigen: vi.fn() }));

import { verwerfenBestaetigen } from './bestaetigung.svelte.js';
import { uiStore } from './uiStore.svelte.js';
import { schuetzeUngespeichertes } from './verlassenSchutz.svelte.js';

// Der Verlassen-Schutz einer Maske (docs/OFFEN.md 5.79): Ein Wechsel der Seite hält an und
// fragt, das Fenster fragt der Browser, und nach dem Abbau der Maske ist der Wächter weg.

/** Die nächste Frage bleibt offen, bis der Test sie beantwortet. */
function halteFrage() {
	/** @type {(ja: boolean) => void} */
	let antworte = () => {};
	vi.mocked(verwerfenBestaetigen).mockImplementationOnce(
		() => new Promise((fertig) => (antworte = fertig))
	);
	return (/** @type {boolean} */ ja) => antworte(ja);
}

/** Baut den Schutz auf wie ein Bauteil. @param {() => boolean} ungespeichert */
function baueAuf(ungespeichert) {
	const abbau = $effect.root(() => schuetzeUngespeichertes(ungespeichert));
	flushSync();
	return abbau;
}

/** @type {() => void} */
let abbau = () => {};
beforeEach(() => {
	vi.clearAllMocks();
	uiStore.verlassenSperre = null;
	uiStore.erzwingeWechsel();
	uiStore.activeTab = 'media_catalog';
});
afterEach(() => abbau());

describe('Verlassen-Schutz einer Maske', () => {
	it('hält den Wechsel an, fragt und führt ihn nach „Verwerfen und weiter" aus', async () => {
		const antworte = halteFrage();
		abbau = baueAuf(() => true);

		uiStore.activeTab = 'kiosk';
		flushSync();
		expect(uiStore.activeTab, 'angehalten').toBe('media_catalog');
		expect(verwerfenBestaetigen).toHaveBeenCalledWith('Verwerfen und weiter');

		antworte(true);
		await vi.waitFor(() => expect(uiStore.activeTab).toBe('kiosk'));
		expect(uiStore.blockierterWechsel).toBeNull();
	});

	it('„Bleiben" lässt die Seite stehen', async () => {
		const antworte = halteFrage();
		abbau = baueAuf(() => true);

		uiStore.activeTab = 'kiosk';
		flushSync();
		antworte(false);
		await vi.waitFor(() => expect(uiStore.blockierterWechsel).toBeNull());
		expect(uiStore.activeTab).toBe('media_catalog');
	});

	it('beantwortet eine ältere Frage nicht für einen neueren Wechsel', async () => {
		const ersteAntwort = halteFrage();
		const zweiteAntwort = halteFrage();
		abbau = baueAuf(() => true);

		uiStore.activeTab = 'kiosk';
		flushSync();
		// Die Zurück-Taste bei offener Frage hält einen zweiten Wechsel an.
		uiStore.activeTab = 'settings';
		flushSync();
		expect(verwerfenBestaetigen).toHaveBeenCalledTimes(2);

		// Eine zweite Rückfrage beantwortet die erste mit „nein" (bestaetigung.svelte.js); hier
		// käme ein „ja" an, das für den älteren Wechsel nicht mehr gilt.
		ersteAntwort(true);
		await Promise.resolve();
		await Promise.resolve();
		expect(uiStore.activeTab, 'die ältere Frage wechselt nicht').toBe('media_catalog');
		expect(uiStore.blockierterWechsel).toBe('settings');

		zweiteAntwort(true);
		await vi.waitFor(() => expect(uiStore.activeTab).toBe('settings'));
	});

	it('ohne Ungespeichertes geht der Wechsel durch, ohne zu fragen', () => {
		abbau = baueAuf(() => false);
		uiStore.activeTab = 'kiosk';
		flushSync();
		expect(uiStore.activeTab).toBe('kiosk');
		expect(verwerfenBestaetigen).not.toHaveBeenCalled();
	});

	it('das Fenster fragt nur, solange etwas ungespeichert ist', () => {
		let ungespeichert = true;
		abbau = baueAuf(() => ungespeichert);
		const schliessen = () => {
			const ereignis = new Event('beforeunload', { cancelable: true });
			window.dispatchEvent(ereignis);
			return ereignis.defaultPrevented;
		};
		expect(schliessen(), 'mit Ungespeichertem').toBe(true);
		ungespeichert = false;
		expect(schliessen(), 'ohne').toBe(false);
	});

	it('nach dem Abbau ist der Wächter weg, auch für das Fenster', () => {
		abbau = baueAuf(() => true);
		expect(uiStore.verlassenSperre).not.toBeNull();
		abbau();
		abbau = () => {};
		expect(uiStore.verlassenSperre).toBeNull();
		const ereignis = new Event('beforeunload', { cancelable: true });
		window.dispatchEvent(ereignis);
		expect(ereignis.defaultPrevented).toBe(false);
	});
});
