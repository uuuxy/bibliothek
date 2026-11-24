import { it, expect, vi, beforeAll } from 'vitest';
import { apiFetch } from '../apiFetch.js';
import { mahnwesenStore } from './mahnwesen.svelte.js';

// Nur apiFetch ersetzen, den Rest des Moduls behalten (siehe MahnwesenAktionen.test.js).
vi.mock('../apiFetch.js', async (importOriginal) => ({
	...(await importOriginal()),
	apiFetch: vi.fn()
}));

/** @param {string} id @param {string} name */
const kind = (id, name) => ({
	schueler_id: id,
	name,
	klasse: '07H2',
	medien: [{ ausleihe_id: `a-${id}`, titel: 'Band', tage_ueberfaellig: 5, mahnstufe: 0 }]
});

beforeAll(async () => {
	/** @type {any} */ (apiFetch).mockResolvedValue({
		ok: true,
		json: async () => ({
			klassen: [
				{
					klasse: '07H2',
					schueler: [
						kind('s1', 'Mehmet Öztürk'),
						kind('s2', 'Ahmad Al-Kahil'),
						kind('s3', 'Anna Müller')
					]
				}
			]
		})
	});
	await mahnwesenStore.lade();
});

// Dieselbe Regel wie an der Theke (utils/suchnorm.js): ohne Akzente, ohne Bindestrich, Vor- und
// Nachname in beliebiger Reihenfolge.
it.each([
	['ozturk', ['Mehmet Öztürk']],
	['alkahil', ['Ahmad Al-Kahil']],
	['al kahil', ['Ahmad Al-Kahil']],
	['mueller anna', ['Anna Müller']],
	['07h2', ['Ahmad Al-Kahil', 'Anna Müller', 'Mehmet Öztürk']]
])('die Mahnliste findet zu „%s“ %j', (suche, erwartet) => {
	mahnwesenStore.suche = suche;
	expect(mahnwesenStore.gefilterteSchueler.map((/** @type {any} */ s) => s.name)).toEqual(erwartet);
});
