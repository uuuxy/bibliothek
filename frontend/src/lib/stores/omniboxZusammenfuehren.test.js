import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Nach dem Zusammenführen zweier Datensätze hält die Theke ihren aktiven Schüler auf das
// Ziel um (OFFEN.md 3.2, 15.09.2026). Bis dahin blieb die gelöschte Kennung stehen: Die
// Akte zeigte schon das Ziel, aber jede weitere Buchung — und jeder Eintrag in der
// Offline-Warteschlange (enqueueOfflineAction mit activeStudent.id) — lief auf einen
// Datensatz, den es nicht mehr gab.
const apiFetch = vi.hoisted(() => vi.fn());
vi.mock('../apiFetch.js', () => ({ apiFetch, apiClient: { post: vi.fn() } }));
vi.mock('../audio.js', () => ({
	playSoundSuccess: vi.fn(),
	playSoundError: vi.fn(),
	playSuccessBeep: vi.fn(),
	playErrorBeep: vi.fn()
}));
vi.mock('../../inventur/lib/store.svelte.js', () => ({ showToast: vi.fn() }));

import { omniboxStore } from './omnibox.svelte.js';

// Nach jedem Fall die Zeitgeber der Theke stoppen. `scanfeldWiederScharfstellen` plant
// einen Fokussprung über 50 ms, der `document` anfasst — endet die Datei vorher, baut
// Vitest jsdom ab, und der Rückruf reisst den GANZEN Lauf rot („Unhandled Errors:
// document is not defined"), obwohl jeder Test grün ist. Genau so stand die CI am
// 16.09.2026. Belegt in stores/omniboxZeitgeber.test.js.
afterEach(() => omniboxStore.stoppeZeitgeber());

describe('Theke nach dem Zusammenführen', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		omniboxStore.activeStudent = { id: 'quelle', vorname: 'Anna', is_manually_blocked: true };
	});

	it('hängt den aktiven Schüler auf das Ziel um, mit dessen Daten', async () => {
		apiFetch.mockResolvedValue({
			ok: true,
			json: async () => ({ id: 'ziel', vorname: 'Anna', is_manually_blocked: false })
		});
		await omniboxStore.uebernimmZusammengefuehrt('ziel');
		expect(apiFetch).toHaveBeenCalledWith('/api/schueler/ziel');
		expect(omniboxStore.activeStudent).toEqual({
			id: 'ziel',
			vorname: 'Anna',
			is_manually_blocked: false
		});
	});

	it('trägt die Ziel-Kennung auch, wenn das Nachladen scheitert', async () => {
		apiFetch.mockResolvedValue({ ok: false, status: 503, json: async () => ({}) });
		await omniboxStore.uebernimmZusammengefuehrt('ziel');
		expect(omniboxStore.activeStudent?.id, 'die gelöschte Kennung bliebe sonst stehen').toBe(
			'ziel'
		);
	});

	it('lässt ein überholtes Nachladen nicht gewinnen', async () => {
		/** @type {() => void} */
		let loesen = () => {};
		apiFetch.mockImplementation(async () => {
			await new Promise((r) => (loesen = /** @type {any} */ (r)));
			return { ok: true, json: async () => ({ id: 'ziel', vorname: 'Anna' }) };
		});
		const laeuft = omniboxStore.uebernimmZusammengefuehrt('ziel');
		// Inzwischen wurde ein anderer Ausweis gescannt.
		omniboxStore.activeStudent = { id: 'anderer', vorname: 'Ben' };
		loesen();
		await laeuft;
		expect(omniboxStore.activeStudent?.id).toBe('anderer');
	});
});
