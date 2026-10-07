import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('idb', () => ({
	openDB: async () => ({
		get: async () => undefined,
		put: async () => undefined,
		delete: async () => undefined
	})
}));

// Ob Netz da ist, sagt netzLage (gemessen an /health); der Test stellt die Auskunft selbst.
const netz = vi.hoisted(() => ({ offline: false }));
vi.mock('./netzLage.svelte.js', () => ({ netzLage: netz }));

import { apiFetch } from '../apiFetch.js';
import { buchBarcodes } from './buchBarcodes.svelte.js';

/** Eine Antwort des Servers, die „unverändert" sagt — der Normalfall. */
function unveraendert() {
	return /** @type {any} */ ({ status: 304, ok: false });
}

// Die Barcode-Liste altert unbemerkt (OFFEN.md 5.19, entschieden am 16.09.2026:
// stündlich nachfassen).
//
// Geholt wurde die Liste beim Anmelden — und ein Kiosk-Tab steht zwölf Stunden offen.
// Was am Vormittag neu inventarisiert wurde, war am Nachmittag ohne Netz eine „unklare"
// Nummer, und niemand konnte das wissen. Der Zeitpunkt des letzten Abgleichs wurde
// gespeichert und nirgends bewertet.
describe('Barcode-Liste: stündlich nachfassen', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		vi.mocked(apiFetch).mockResolvedValue(unveraendert());
	});

	afterEach(() => {
		buchBarcodes.stoppeZeitgeber();
		netz.offline = false;
		vi.useRealTimers();
		vi.clearAllMocks();
	});

	it('fragt nach einer Stunde von selbst nach', async () => {
		buchBarcodes.starteAbgleich();
		expect(apiFetch).not.toHaveBeenCalled(); // nicht sofort — das tut bereitstellen()

		await vi.advanceTimersByTimeAsync(60 * 60 * 1000);
		expect(apiFetch).toHaveBeenCalledTimes(1);

		await vi.advanceTimersByTimeAsync(60 * 60 * 1000);
		expect(apiFetch, 'und danach wieder').toHaveBeenCalledTimes(2);
	});

	// Ohne Netz wäre die Frage ein Fehler pro Stunde im Protokoll für nichts. Die Liste
	// vom letzten Mal gilt weiter — das ist der ganze Sinn der Ablage.
	it('fragt ohne Netz gar nicht erst', async () => {
		netz.offline = true;
		buchBarcodes.starteAbgleich();
		await vi.advanceTimersByTimeAsync(3 * 60 * 60 * 1000);
		expect(apiFetch).not.toHaveBeenCalled();
	});

	// Ein Browser kann „offline" melden, obwohl der Server antwortet. Hinge der Abgleich an
	// dieser Auskunft, bliebe die Liste an so einem Rechner den ganzen Tag alt.
	it('fragt nach, wenn nur der Browser „offline" meldet', async () => {
		// `onLine` liegt am Prototyp, nicht an der Instanz: Die eigene Eigenschaft wird
		// angelegt und danach entfernt, damit für die folgenden Tests der Prototyp gilt.
		Object.defineProperty(navigator, 'onLine', { value: false, configurable: true });
		try {
			buchBarcodes.starteAbgleich();
			await vi.advanceTimersByTimeAsync(60 * 60 * 1000);
			expect(apiFetch).toHaveBeenCalledTimes(1);
		} finally {
			Reflect.deleteProperty(navigator, 'onLine');
		}
		expect(navigator.onLine, 'der Rechner ist danach wieder online').toBe(true);
	});

	// Ein Zeitgeber, der die Anmeldung überlebt, fragt für niemanden nach — und färbt im
	// Test den ganzen Lauf rot, wenn er nach dem Abbau von jsdom noch feuert
	// (frontend-hygiene-thekenzeitgeber.test.js).
	it('lässt sich beenden, und dann ist Ruhe', async () => {
		buchBarcodes.starteAbgleich();
		buchBarcodes.stoppeZeitgeber();
		await vi.advanceTimersByTimeAsync(5 * 60 * 60 * 1000);
		expect(apiFetch).not.toHaveBeenCalled();
	});

	it('startet keinen zweiten Zeitgeber, wenn zweimal gestartet wird', async () => {
		buchBarcodes.starteAbgleich();
		buchBarcodes.starteAbgleich();
		await vi.advanceTimersByTimeAsync(60 * 60 * 1000);
		expect(apiFetch, 'zwei Zeitgeber fragten doppelt').toHaveBeenCalledTimes(1);
	});
});
