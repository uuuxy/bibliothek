import { describe, it, expect, vi, beforeAll } from 'vitest';
import { render, within } from '@testing-library/svelte';
import MahnwesenTable from './MahnwesenTable.svelte';
import { apiFetch } from '../../apiFetch.js';
import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';

// Nur apiFetch ersetzen, den Rest des Moduls behalten (siehe MahnwesenAktionen.test.js).
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	...(await importOriginal()),
	apiFetch: vi.fn()
}));

/**
 * Die Spalte „Status" nennt, was hinausging: wie oft und wann zuletzt ein Mahnbrief
 * gedruckt wurde. Vorher stand dort „1. Erinnerung" oder „Mahnung", gerechnet aus den Tagen
 * über der Frist, gleich ob je ein Brief gedruckt wurde.
 */
beforeAll(async () => {
	/** @type {any} */ (apiFetch).mockResolvedValue({
		ok: true,
		json: async () => ({
			klassen: [
				{
					klasse: '07H2',
					schueler: [
						{
							schueler_id: 's1',
							name: 'Ida Zweimal',
							klasse: '07H2',
							medien: [
								{
									ausleihe_id: 'a1',
									titel: 'Band A',
									tage_ueberfaellig: 20,
									mahnstufe: 2,
									letztes_mahndatum: '2026-09-26'
								},
								{ ausleihe_id: 'a2', titel: 'Band B', tage_ueberfaellig: 3, mahnstufe: 0 }
							]
						},
						{
							schueler_id: 's2',
							name: 'Timo Nochnie',
							klasse: '07H2',
							medien: [{ ausleihe_id: 'a3', titel: 'Band C', tage_ueberfaellig: 30, mahnstufe: 0 }]
						}
					]
				}
			]
		})
	});
	await mahnwesenStore.fetchData();
	if (mahnwesenStore.filteredSchueler.length !== 2) throw new Error('Testdaten nicht geladen');
});

describe('MahnwesenTable', () => {
	it('nennt je Kind, wie oft und wann zuletzt gemahnt wurde', () => {
		const liste = render(MahnwesenTable);

		const ida = within(liste.getByRole('row', { name: /Ida Zweimal/ }));
		expect(ida.getByText('2× gemahnt, zuletzt 26.09.2026')).toBeTruthy();
		expect(ida.getByText('20 Tage überfällig')).toBeTruthy();

		// 30 Tage über der Frist und kein Brief: Die Zeile sagt das, statt „Mahnung" zu nennen.
		const timo = within(liste.getByRole('row', { name: /Timo Nochnie/ }));
		expect(timo.getByText('noch nicht gemahnt')).toBeTruthy();
		expect(timo.getByText('30 Tage überfällig')).toBeTruthy();

		expect(liste.queryByText('Mahnung')).toBeNull();
		expect(liste.queryByText('1. Erinnerung')).toBeNull();
	});
});
