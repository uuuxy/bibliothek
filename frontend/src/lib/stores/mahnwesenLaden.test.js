import { it, expect, vi } from 'vitest';
import { apiFetch } from '../apiFetch.js';
import { mahnwesenStore } from './mahnwesen.svelte.js';

vi.mock('../apiFetch.js', async (importOriginal) => ({
	...(await importOriginal()),
	apiFetch: vi.fn()
}));

/** @param {string[]} ids */
const liste = (ids) => ({
	klassen: [
		{
			klasse: '07H2',
			schueler: ids.map((id) => ({
				schueler_id: id,
				name: `Kind ${id}`,
				klasse: '07H2',
				medien: [{ ausleihe_id: `a-${id}`, titel: 'Band', tage_ueberfaellig: 5, mahnstufe: 0 }]
			}))
		}
	]
});

/** Eine Antwort, die erst kommt, wenn der Test sie freigibt. */
function spaeteAntwort() {
	/** @type {(v: any) => void} */
	let gib = () => {};
	const versprechen = new Promise((r) => (gib = r));
	return {
		versprechen,
		gib: (/** @type {string[]} */ ids) => gib({ ok: true, json: async () => liste(ids) })
	};
}

const ids = () =>
	mahnwesenStore.gefilterteSchueler.map((/** @type {any} */ s) => s.schueler_id).sort();

// Zwei Abrufe überschneiden sich (Rückkehr auf die Seite, dann ein Nachladen nach einer Rückgabe):
// Kommt die Antwort des älteren zuletzt, darf sie den neueren Stand nicht überschreiben.
it('eine überholte Antwort überschreibt den neueren Stand nicht', async () => {
	const alt = spaeteAntwort();
	const neu = spaeteAntwort();
	/** @type {any} */ (apiFetch)
		.mockReturnValueOnce(alt.versprechen)
		.mockReturnValueOnce(neu.versprechen);
	const erster = mahnwesenStore.lade();
	const zweiter = mahnwesenStore.lade();
	neu.gib(['s2']);
	await zweiter;
	alt.gib(['s1', 's2']);
	await erster;
	expect(ids()).toEqual(['s2']);
	expect(mahnwesenStore.laedt).toBe(false);
});

// Das stille Nachladen nach einer Rückgabe an einem anderen Platz lässt die Auswahl stehen, soweit
// die Kinder noch in der Liste sind; ein Laden von Hand fängt mit leerer Auswahl an.
it('das stille Nachladen behält die Auswahl der Kinder, die noch in der Liste stehen', async () => {
	/** @type {any} */ (apiFetch).mockResolvedValueOnce({
		ok: true,
		json: async () => liste(['s1', 's2', 's3'])
	});
	await mahnwesenStore.lade();
	mahnwesenStore.schalteAuswahl('s1');
	mahnwesenStore.schalteAuswahl('s2');

	/** @type {any} */ (apiFetch).mockResolvedValueOnce({
		ok: true,
		json: async () => liste(['s2', 's3'])
	});
	expect(await mahnwesenStore.lade({ still: true })).toBe(true);
	expect([...mahnwesenStore.gewaehlteKennungen]).toEqual(['s2']);

	/** @type {any} */ (apiFetch).mockResolvedValueOnce({
		ok: true,
		json: async () => liste(['s2', 's3'])
	});
	await mahnwesenStore.lade();
	expect([...mahnwesenStore.gewaehlteKennungen]).toEqual([]);
});

// Scheitert das stille Nachladen, bleibt die Liste stehen und es kommt keine Fehlermeldung über
// die Seite; der Aufrufer (abonniereNeuladen) erfährt es und versucht es später neu.
it('ein gescheitertes stilles Nachladen lässt die Liste stehen und meldet false', async () => {
	/** @type {any} */ (apiFetch).mockResolvedValueOnce({
		ok: true,
		json: async () => liste(['s1'])
	});
	await mahnwesenStore.lade();
	/** @type {any} */ (apiFetch).mockRejectedValueOnce(new Error('Netz weg'));
	expect(await mahnwesenStore.lade({ still: true })).toBe(false);
	expect(ids()).toEqual(['s1']);
	expect(mahnwesenStore.fehler).toBe(null);
});
