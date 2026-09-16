import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Der Sperr-Dialog („Ausleihe blockiert", mit „Einmalig ignorieren") hängt am Merkmal
// X-Sperre: uebergehbar, nicht am Wortlaut der Meldung.
//
// Bis zum 13.09.2026 öffnete handleActionHttpError den Dialog nur, wenn im Fehlertext
// „Sperre", „Sperr-Automatik" oder „überfällig" stand. Die Schadens-Sperre („1 unbezahlte(r)
// Schadensfall/-fälle offen") traf keins davon — am Stack nachgestellt: 403, kein Dialog,
// kein Override-Knopf, obwohl der Server das Übergehen erlaubt. Bei der System-Sperre
// entschied der Sperrgrund: „Abgänger-Sperre" enthält das Wort, eine Helferin ohne
// view_students bekommt den Grund aber nicht zu sehen.
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

// Nach jedem Fall die Zeitgeber der Theke stoppen. `scanfeldWiederScharfstellen` plant
// einen Fokussprung über 50 ms, der `document` anfasst — endet die Datei vorher, baut
// Vitest jsdom ab, und der Rückruf reisst den GANZEN Lauf rot („Unhandled Errors:
// document is not defined"), obwohl jeder Test grün ist. Genau so stand die CI am
// 16.09.2026. Belegt in stores/omniboxZeitgeber.test.js.
afterEach(() => omniboxStore.stoppeZeitgeber());

/**
 * Eine 403-Antwort, wie /api/action sie schickt. headers.get ist bewusst von Hand gebaut:
 * Der Test soll nicht davon abhängen, ob die Testumgebung `Response` kennt.
 * @param {string} meldung
 * @param {Record<string, string>} [kopf]
 */
function antwort403(meldung, kopf = {}) {
	const klein = Object.fromEntries(Object.entries(kopf).map(([k, v]) => [k.toLowerCase(), v]));
	return {
		ok: false,
		status: 403,
		headers: { get: (/** @type {string} */ name) => klein[name.toLowerCase()] ?? null },
		text: async () => JSON.stringify({ error: meldung }),
		json: async () => ({ error: meldung })
	};
}

async function scanne() {
	omniboxStore.queryVal = 'B-4711';
	await omniboxStore.submitAction(new Event('submit'));
}

describe('Sperr-Dialog an der Theke', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		omniboxStore.blockAlert = null;
		omniboxStore.activeStudent = { id: 'schueler-7', vorname: 'Anna', nachname: 'Müller' };
	});

	it('öffnet für eine übergehbare Sperre, auch wenn der Text keins der alten Wörter enthält', async () => {
		post.mockResolvedValue(
			antwort403(
				'ausleihe für diese/n Schüler/in ist gesperrt: 1 unbezahlte(r) Schadensfall/-fälle offen',
				{
					'X-Sperre': 'uebergehbar'
				}
			)
		);
		await scanne();
		expect(
			omniboxStore.blockAlert,
			'Schadens-Sperre ohne Dialog — kein Override möglich'
		).not.toBeNull();
		expect(omniboxStore.blockAlert?.query).toBe('B-4711');
	});

	it('bleibt zu, wenn nur der Text nach Sperre klingt', async () => {
		post.mockResolvedValue(
			antwort403('ausleihe für diese/n Schüler/in ist gesperrt: Manuelle Sperre')
		);
		await scanne();
		expect(omniboxStore.blockAlert, 'Dialog am Wortlaut statt am Merkmal').toBeNull();
	});

	it('bleibt zu bei einem 403, das keine Sperre ist', async () => {
		post.mockResolvedValue(antwort403('keine Berechtigung für diese Aktion'));
		await scanne();
		expect(omniboxStore.blockAlert).toBeNull();
	});
});
