import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../../../lib/apiFetch.js', () => ({
	apiFetch: vi.fn(),
	apiPut: vi.fn(),
	apiClient: { post: vi.fn() },
	extractApiError: vi.fn(async (/** @type {any} */ res) => `Fehler ${res.status}`)
}));
vi.mock('$lib/store.svelte.js', () => ({ showToast: vi.fn() }));

import { apiFetch } from '../../../../lib/apiFetch.js';
import BuchEingabefelderKategorisierung from './BuchEingabefelderKategorisierung.svelte';
import { leeresBuchFormular } from './buch_form_optionen.js';

beforeEach(() => {
	vi.clearAllMocks();
	vi.mocked(apiFetch).mockResolvedValue(
		/** @type {any} */ ({ ok: true, status: 200, json: async () => [] })
	);
});

/** Die Gruppe „An der Schule" mit einem Formular, in das der Test tippt.
 * @param {Record<string, any>} [zusatz] */
function gruppe(zusatz = {}) {
	/** @type {any} */
	const formular = $state({ ...leeresBuchFormular(), ...zusatz });
	const { container } = render(BuchEingabefelderKategorisierung, { formular });
	const feld = (/** @type {string} */ id) =>
		/** @type {HTMLInputElement} */ (container.querySelector(`#${id}`));
	return {
		formular,
		container,
		von: () => feld('buch-jahrgang-von'),
		bis: () => feld('buch-jahrgang-bis'),
		/** @param {string} id @param {string} wert */
		tippe: async (id, wert) => {
			await fireEvent.input(feld(id), { target: { value: wert } });
			await tick();
		}
	};
}

// Der Jahrgang am Titel ist eine Angabe: „von" und „bis". Das Feld „Klasse" gibt es nicht
// mehr, und „bis" geht mit „von" mit, weil die meisten Schulbücher für ein Jahr gelten.
describe('Buchmaske: der Jahrgang', () => {
	it('führt kein Feld „Klasse"', () => {
		const { container } = gruppe();
		expect(container.querySelector('#buch-klasse')).toBeNull();
		const beschriftungen = [...container.querySelectorAll('label')].map((l) =>
			(l.textContent ?? '').trim()
		);
		expect(beschriftungen).not.toContain('Klasse');
		expect(beschriftungen).toEqual(
			expect.arrayContaining(['Geeignet für Jahrgang von', 'bis Jahrgang'])
		);
	});

	it('wer in „von" eine Zahl tippt, hat dieselbe in „bis"', async () => {
		const m = gruppe();
		await m.tippe('buch-jahrgang-von', '7');
		expect([m.formular.jahrgangVon, m.formular.jahrgangBis]).toEqual([7, 7]);
		expect(m.bis().value).toBe('7');
	});

	it('eine zweistellige Zahl kommt Ziffer für Ziffer an', async () => {
		const m = gruppe();
		await m.tippe('buch-jahrgang-von', '1');
		await m.tippe('buch-jahrgang-von', '10');
		expect([m.formular.jahrgangVon, m.formular.jahrgangBis]).toEqual([10, 10]);
	});

	it('ein eigener Wert in „bis" bleibt stehen', async () => {
		const m = gruppe({ jahrgangVon: 7, jahrgangBis: 7 });
		await m.tippe('buch-jahrgang-bis', '10');
		await m.tippe('buch-jahrgang-von', '8');
		expect([m.formular.jahrgangVon, m.formular.jahrgangBis]).toEqual([8, 10]);
		expect(m.bis().value).toBe('10');
	});

	it('ein Jahr berichtigen: „bis" geht mit', async () => {
		const m = gruppe({ jahrgangVon: 7, jahrgangBis: 7 });
		await m.tippe('buch-jahrgang-von', '6');
		expect([m.formular.jahrgangVon, m.formular.jahrgangBis]).toEqual([6, 6]);
	});

	it('ein geleertes „von" leert ein mitgegangenes „bis"', async () => {
		const m = gruppe({ jahrgangVon: 7, jahrgangBis: 7 });
		await m.tippe('buch-jahrgang-von', '');
		expect(m.formular.jahrgangVon ?? null).toBeNull();
		expect(m.formular.jahrgangBis ?? null).toBeNull();
		expect(m.bis().value).toBe('');
	});
});
