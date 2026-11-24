import { describe, it, expect, vi, beforeAll } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import MahnwesenTabelle from './MahnwesenTabelle.svelte';
import { apiFetch } from '../../apiFetch.js';
import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';

// Nur apiFetch ersetzen, den Rest des Moduls behalten (siehe MahnwesenAktionen.test.js).
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	...(await importOriginal()),
	apiFetch: vi.fn()
}));

// 51 Kinder, eins mehr, als die Tabelle zuerst zeichnet. Kind 50 ist am kürzesten überfällig
// und steht deshalb zuletzt.
const schueler = Array.from({ length: 51 }, (_, i) => ({
	schueler_id: `s${i}`,
	name: `Kind ${String(i).padStart(2, '0')}`,
	klasse: '07H2',
	medien: [{ ausleihe_id: `a${i}`, titel: 'Band', tage_ueberfaellig: 100 - i, mahnstufe: 0 }]
}));

beforeAll(async () => {
	/** @type {any} */ (apiFetch).mockResolvedValue({
		ok: true,
		json: async () => ({ klassen: [{ klasse: '07H2', schueler }] })
	});
	await mahnwesenStore.lade();
	if (mahnwesenStore.gefilterteSchueler.length !== 51) throw new Error('Testdaten nicht geladen');
});

describe('MahnwesenTabelle mit vielen Kindern', () => {
	it('zeichnet zuerst 50 Zeilen und „Mehr laden" die übrigen; die Auswahl umfasst alle', async () => {
		const liste = render(MahnwesenTabelle);

		// Kopfzeile und 50 Kinder
		expect(liste.getAllByRole('row')).toHaveLength(51);
		expect(liste.queryByRole('row', { name: /Kind 50/ })).toBeNull();

		await fireEvent.click(
			liste.getByRole('checkbox', { name: 'Alle überfälligen Schüler auswählen' })
		);
		expect(mahnwesenStore.gewaehlteKennungen.size).toBe(51);

		await fireEvent.click(liste.getByRole('button', { name: 'Mehr laden, noch 1 Zeile übrig' }));
		expect(liste.getByRole('row', { name: /Kind 50/ })).toBeTruthy();
		expect(liste.queryByRole('button', { name: /Mehr laden/ })).toBeNull();
	});
});
