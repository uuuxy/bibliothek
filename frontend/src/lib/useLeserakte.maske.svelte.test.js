import { describe, it, expect, vi, beforeEach } from 'vitest';
import { flushSync } from 'svelte';

vi.mock('./apiFetch.js', async (original) => ({ ...(await original()), apiFetch: vi.fn() }));

import { apiFetch } from './apiFetch.js';
import { useLeserakte } from './useLeserakte.svelte.js';

// Solange „Stammdaten bearbeiten" offen ist, lädt die Akte desselben Lesers nicht neu: An der
// Theke lud jeder Scan und jede Buchung eines anderen Platzes die Akte neu, und die Maske kam
// leer wieder. „Zurück" holt das Neuladen nach.

/** @param {any} body @returns {any} */
const ok = (body) => ({ ok: true, json: async () => body });

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
	vi.mocked(apiFetch).mockImplementation(async (url) => {
		const u = String(url);
		if (u.includes('vormerkungen')) return ok([]);
		if (u.includes('schadensfaelle') || u.includes('bescheide')) return ok({ data: [] });
		return ok({ id: u.includes('/B') ? 'B' : 'A', vorname: 'Anna' });
	});
});

/** Eine Akte von A mit offener Maske, die Abrufe des Ladens gezählt und vergessen. */
async function offeneMaske() {
	const st = useLeserakte();
	await st.ladeAkte('A');
	st.bearbeitenOffen = true;
	vi.mocked(apiFetch).mockClear();
	return st;
}

describe('Akte mit offener Maske „Stammdaten bearbeiten"', () => {
	it('lädt nicht neu; „Zurück" holt es nach', async () => {
		const st = await offeneMaske();

		await st.ladeAkte('A');
		expect(apiFetch, 'kein Abruf unter der offenen Maske').not.toHaveBeenCalled();
		expect(st.bearbeitenOffen).toBe(true);

		st.schliesseBearbeiten();
		expect(st.bearbeitenOffen).toBe(false);
		await vi.waitFor(() => expect(apiFetch).toHaveBeenCalledTimes(4));
	});

	it('„Zurück" ohne verpasstes Neuladen ruft nichts ab', async () => {
		const st = await offeneMaske();
		st.schliesseBearbeiten();
		expect(apiFetch).not.toHaveBeenCalled();
	});

	it('ein anderer Leser lädt sofort', async () => {
		const st = await offeneMaske();
		await st.ladeAkte('B');
		expect(st.akte?.id).toBe('B');
	});

	// Die Leserakte ruft ladeAkte aus einem $effect (beim Wechsel des Lesers, zusammen mit
	// schliesseAlleBlaetter). Hinge er an der offenen Maske, schlösse er sie bei jedem Öffnen.
	it('der $effect, der die Akte lädt, hängt nicht an der offenen Maske', async () => {
		const st = useLeserakte();
		let laeufe = 0;
		const abbau = $effect.root(() => {
			$effect(() => {
				laeufe += 1;
				st.schliesseAlleBlaetter();
				st.ladeAkte('A');
			});
		});
		flushSync();
		await vi.waitFor(() => expect(st.akte?.id).toBe('A'));

		st.bearbeitenOffen = true;
		flushSync();
		expect(st.bearbeitenOffen, 'die Maske bleibt offen').toBe(true);
		expect(laeufe).toBe(1);
		abbau();
	});
});
