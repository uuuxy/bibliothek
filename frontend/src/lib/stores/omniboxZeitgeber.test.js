import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Kein Zeitgeber der Theke überlebt die Seite.
//
// Der Fall, den dieser Test festhält (CI am 16.09.2026): Der Lauf war ROT bei 128 grünen
// Dateien und 697 grünen Tests — „Unhandled Errors: ReferenceError: document is not
// defined", gemeldet aus stores/omnibox.svelte.js, zugeordnet der Datei
// omniboxSperrDialog.test.js. Es war kein Testfehler: `scanfeldWiederScharfstellen` plante
// einen setTimeout über 50 ms, der `document.getElementById('omnibox-input')` anfasst.
// Endete die Testdatei vorher, baute Vitest jsdom ab — und der Rückruf lief ohne
// `document`. Ein einziger solcher Rückruf färbt den GANZEN Lauf rot, obwohl kein Test
// etwas falsch macht, und der Fehler zeigt auf die Testdatei statt auf den Timer.
//
// Dieselbe Bugklasse traf am 11.09.2026 actions/keyboardNav.js. Die Antwort ist dieselbe:
// ein Handle je Zeitgeber, verworfen vor dem Neuplanen und beim Abbau.
//
// Gemessen wird hier an der ZUSAGE, nicht am Wortlaut: nach stoppeZeitgeber() steht kein
// Timer mehr offen, und ein Rückruf, der die Seite anfasst, kann nicht mehr feuern.
const post = vi.hoisted(() => vi.fn());
vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn(), apiClient: { post } }));
vi.mock('../audio.js', () => ({
	playSoundSuccess: vi.fn(),
	playSoundError: vi.fn(),
	playSuccessBeep: vi.fn(),
	playErrorBeep: vi.fn()
}));
vi.mock('../../inventur/lib/store.svelte.js', () => ({ showToast: vi.fn() }));

import { omniboxStore } from './omnibox.svelte.js';
import { uiStore } from './uiStore.svelte.js';

describe('Zeitgeber der Theke', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		omniboxStore.blockAlert = null;
		omniboxStore.showCamera = false;
	});
	afterEach(() => {
		omniboxStore.stoppeZeitgeber();
		vi.useRealTimers();
	});

	it('stoppt jeden laufenden Zeitgeber auf einen Schlag', () => {
		omniboxStore.triggerScreenFlash('success');
		omniboxStore.triggerShake();
		omniboxStore.triggerFlash('green');
		uiStore.beimWechselZurTheke?.();
		expect(vi.getTimerCount(), 'die Probe misst nichts, wenn gar kein Timer läuft').toBeGreaterThan(
			0
		);

		omniboxStore.stoppeZeitgeber();
		expect(vi.getTimerCount()).toBe(0);
	});

	// Der Kern: DER Rückruf, der den CI-Lauf gerissen hat. Nachgestellt wird genau das,
	// was Vitest am Ende einer Datei tut — jsdom weg, Timer noch da.
	it('lässt keinen Rückruf auf die Seite greifen, nachdem sie abgebaut ist', () => {
		uiStore.beimWechselZurTheke?.();
		omniboxStore.stoppeZeitgeber();

		const doc = globalThis.document;
		// @ts-expect-error — der Abbau der Testumgebung, von Hand nachgestellt
		delete globalThis.document;
		let geknallt = '';
		try {
			vi.advanceTimersByTime(1000);
		} catch (e) {
			geknallt = String(e);
		} finally {
			globalThis.document = doc;
		}
		expect(geknallt, 'ein Zeitgeber hat die Seite überlebt und ins Leere gegriffen').toBe('');
	});

	it('plant den Fokussprung nicht doppelt', () => {
		uiStore.beimWechselZurTheke?.();
		const nachDemErsten = vi.getTimerCount();
		uiStore.beimWechselZurTheke?.();
		expect(vi.getTimerCount(), 'der zweite Aufruf hat einen zweiten Timer stehen lassen').toBe(
			nachDemErsten
		);
	});
});
