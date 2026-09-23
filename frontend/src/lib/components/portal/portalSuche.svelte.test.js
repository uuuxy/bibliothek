import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { flushSync } from 'svelte';
import { erzeugePortalSuche } from './portalSuche.svelte.js';
import { apiFetch } from '../../apiFetch.js';

vi.mock('../../apiFetch.js', () => ({ apiFetch: vi.fn() }));

// Die Suche in „Mein Portal" (docs/OFFEN.md 4.20): Ein Filter sucht auch ohne Text, Text
// und Filter gehen zusammen an den öffentlichen Katalog, die Gesamtzahl kommt aus dem Kopf
// X-Treffer-Gesamt — und eine späte Antwort der vorigen Suche darf die Treffer der neuen
// nicht überschreiben. Wer schnell zwischen zwei Filtern wechselt, sähe sonst die Titel
// des ersten unter dem zweiten.

/** @param {any[]} liste @param {number} [gesamt] */
const antwort = (liste, gesamt = liste.length) =>
	/** @type {any} */ ({
		ok: true,
		json: async () => liste,
		headers: new Headers({ 'X-Treffer-Gesamt': String(gesamt) })
	});

/** Erzeugt die Suche in einer Effekt-Wurzel, wie in schuelerSuche.svelte.test.js. */
function aufbau() {
	/** @type {any} */
	let suche;
	const stopp = $effect.root(() => {
		suche = erzeugePortalSuche();
	});
	return { suche, stopp };
}

describe('Portal-Suche', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		vi.mocked(apiFetch).mockReset();
	});
	afterEach(() => vi.useRealTimers());

	it('sucht mit einem Filter auch ohne Text und nimmt die Gesamtzahl aus dem Kopf', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort([{ id: 't1', titel: 'Mondflug' }], 52));
		const { suche, stopp } = aufbau();
		try {
			suche.schlagwort = 'w1';
			flushSync();
			await vi.advanceTimersByTimeAsync(300);
			expect(vi.mocked(apiFetch).mock.calls.at(-1)?.[0]).toBe(
				'/api/public/opac/suche?schlagwort_id=w1'
			);
			expect(suche.treffer.map((/** @type {any} */ t) => t.titel)).toEqual(['Mondflug']);
			expect(suche.gesamt).toBe(52);
			expect(suche.leer).toBe(false);
		} finally {
			stopp();
		}
	});

	it('schickt Text und Filter zusammen; ein Zeichen allein sucht noch nicht', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort([]));
		const { suche, stopp } = aufbau();
		try {
			suche.text = 'M';
			flushSync();
			await vi.advanceTimersByTimeAsync(300);
			expect(apiFetch).not.toHaveBeenCalled();

			suche.text = 'Mond & Sterne';
			suche.schlagwort = 'w1';
			flushSync();
			await vi.advanceTimersByTimeAsync(300);
			expect(vi.mocked(apiFetch).mock.calls.at(-1)?.[0]).toBe(
				'/api/public/opac/suche?q=Mond%20%26%20Sterne&schlagwort_id=w1'
			);
		} finally {
			stopp();
		}
	});

	it('verwirft eine späte Antwort der vorigen Suche', async () => {
		/** @type {(r: any) => void} */
		let ersteAntwort = () => {};
		vi.mocked(apiFetch)
			.mockImplementationOnce(() => new Promise((fertig) => (ersteAntwort = fertig)))
			.mockResolvedValueOnce(antwort([{ id: 't2', titel: 'Krimi-Titel' }]));
		const { suche, stopp } = aufbau();
		try {
			suche.schlagwort = 'fantasy';
			flushSync();
			await vi.advanceTimersByTimeAsync(300);
			suche.schlagwort = 'krimi';
			flushSync();
			await vi.advanceTimersByTimeAsync(300);
			expect(suche.treffer.map((/** @type {any} */ t) => t.titel)).toEqual(['Krimi-Titel']);

			ersteAntwort(antwort([{ id: 't1', titel: 'Fantasy-Titel' }]));
			await vi.advanceTimersByTimeAsync(0);
			expect(suche.treffer.map((/** @type {any} */ t) => t.titel)).toEqual(['Krimi-Titel']);
		} finally {
			stopp();
		}
	});

	it('zeigt nach einer abgelehnten Suche keine Treffer von vorher, sondern den Fehler', async () => {
		vi.mocked(apiFetch)
			.mockResolvedValueOnce(antwort([{ id: 't1', titel: 'Mondflug' }]))
			.mockResolvedValueOnce(/** @type {any} */ ({ ok: false, status: 500 }));
		const { suche, stopp } = aufbau();
		try {
			suche.schlagwort = 'w1';
			flushSync();
			await vi.advanceTimersByTimeAsync(300);
			expect(suche.treffer).toHaveLength(1);

			suche.text = 'Sterne';
			flushSync();
			await vi.advanceTimersByTimeAsync(300);
			expect(suche.treffer, 'die Treffer der vorigen Suche stehen noch da').toHaveLength(0);
			expect(suche.fehler).toBe(true);
		} finally {
			stopp();
		}
	});

	it('ohne Text und Filter fragt sie nichts an und zeigt den Überblick', async () => {
		const { suche, stopp } = aufbau();
		try {
			flushSync();
			await vi.advanceTimersByTimeAsync(300);
			expect(apiFetch).not.toHaveBeenCalled();
			expect(suche.aktiv).toBe(false);
			expect(suche.leer).toBe(true);
		} finally {
			stopp();
		}
	});
});
