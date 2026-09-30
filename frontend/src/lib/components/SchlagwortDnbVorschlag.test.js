import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn(), apiPut: vi.fn() }));

import { apiFetch } from '../apiFetch.js';
import SchlagwortDnbVorschlag from './SchlagwortDnbVorschlag.svelte';
import { erzeugeDnbSchlagwortVorschlag } from '../utils/dnbSchlagwortVorschlag.svelte.js';

const ISBN = '978-3-7512-0053-0';
const URL = '/api/schlagworte/dnb-vorschlag?isbn=9783751200530';

/** @param {any} antwort @param {number} [status] */
function dnbAntwortet(antwort, status = 200) {
	vi.mocked(apiFetch).mockImplementation(
		async () => /** @type {any} */ ({ ok: status < 400, status, json: async () => antwort })
	);
}

/** @param {{ isbn?: string, werte?: string[] }} [props] */
function knopf(props = {}) {
	const vorschlag = erzeugeDnbSchlagwortVorschlag();
	const screen = render(SchlagwortDnbVorschlag, { vorschlag, isbn: ISBN, werte: [], ...props });
	return { vorschlag, screen };
}

// Der Knopf „Vorschläge aus der DNB" unter einem Schlagwort-Feld (entschieden am 30.09.2026) und
// die Zeile, die die Antwort der DNB ansagt. Jede Antwort hat ihren Satz: Sähen „die DNB kennt
// die ISBN nicht", „nichts Neues" und „nicht erreichbar" gleich aus, tippte an der Theke jemand
// Wörter von Hand, die die DNB eine Minute später geliefert hätte.
describe('SchlagwortDnbVorschlag', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fragt die DNB mit der ISBN ohne Striche und verschwindet, wenn der Vorschlag da ist', async () => {
		dnbAntwortet({
			dnb_satz: true,
			schlagwort_vorschlaege: ['Krieg'],
			schlagwort_vorschlaege_neu: []
		});
		const { vorschlag, screen } = knopf();

		await fireEvent.click(screen.getByRole('button', { name: 'Vorschläge aus der DNB' }));

		expect(apiFetch).toHaveBeenCalledWith(URL);
		await vi.waitFor(() => expect(vorschlag.liste(ISBN)).toEqual(['Krieg']));
		expect(screen.queryByRole('button', { name: 'Vorschläge aus der DNB' })).toBeNull();
		expect(screen.queryByRole('status')).toBeNull();
	});

	it('sagt, wenn die DNB die ISBN nicht kennt', async () => {
		dnbAntwortet({ dnb_satz: false, schlagwort_vorschlaege: [], schlagwort_vorschlaege_neu: [] });
		const { screen } = knopf();

		await fireEvent.click(screen.getByRole('button', { name: 'Vorschläge aus der DNB' }));

		await vi.waitFor(() =>
			expect(screen.getByRole('status').textContent).toMatch(/Die DNB kennt diese ISBN nicht\./)
		);
	});

	it('sagt, wenn der Titel alles schon trägt, was die DNB nennt', async () => {
		dnbAntwortet({
			dnb_satz: true,
			schlagwort_vorschlaege: ['Krieg'],
			schlagwort_vorschlaege_neu: []
		});
		const { screen } = knopf({ werte: ['krieg'] });

		await fireEvent.click(screen.getByRole('button', { name: 'Vorschläge aus der DNB' }));

		await vi.waitFor(() =>
			expect(screen.getByRole('status').textContent).toMatch(
				/Keine weiteren Schlagworte aus der DNB\./
			)
		);
	});

	it('meldet einen Ausfall als Ausfall und lässt den Knopf zum erneuten Versuch stehen', async () => {
		dnbAntwortet({ error: 'DNB nicht erreichbar' }, 502);
		const { screen } = knopf();

		await fireEvent.click(screen.getByRole('button', { name: 'Vorschläge aus der DNB' }));

		await vi.waitFor(() =>
			expect(screen.getByRole('alert').textContent).toMatch(/Die DNB ist nicht erreichbar/)
		);
		expect(screen.getByRole('button', { name: 'Vorschläge aus der DNB' })).toBeTruthy();
	});

	it('zeigt ohne ISBN keinen Knopf', () => {
		const { screen } = knopf({ isbn: '' });
		expect(screen.queryByRole('button')).toBeNull();
	});
});

// Der Vorschlag gilt für die ISBN, zu der er geholt wurde, und nur die jüngste Antwort schreibt:
// Im Buchformular kann die ISBN geändert werden, während die DNB noch antwortet.
describe('erzeugeDnbSchlagwortVorschlag', () => {
	beforeEach(() => vi.clearAllMocks());

	it('bietet einer anderen ISBN nichts an', async () => {
		dnbAntwortet({
			dnb_satz: true,
			schlagwort_vorschlaege: ['Krieg'],
			schlagwort_vorschlaege_neu: ['Judo']
		});
		const vorschlag = erzeugeDnbSchlagwortVorschlag();

		await vorschlag.lade(ISBN);

		expect(vorschlag.liste('9783751200530')).toEqual(['Krieg']);
		expect(vorschlag.neu(ISBN)).toEqual(['Judo']);
		expect(vorschlag.liste('9783423083003')).toEqual([]);
		expect(vorschlag.status('9783423083003')).toBe('');
	});

	it('lässt eine überholte Antwort fallen', async () => {
		/** @type {(v: any) => void} */
		let ersteAntwort = () => {};
		vi.mocked(apiFetch)
			.mockImplementationOnce(() => new Promise((fertig) => (ersteAntwort = fertig)))
			.mockImplementationOnce(
				async () =>
					/** @type {any} */ ({
						ok: true,
						json: async () => ({
							dnb_satz: true,
							schlagwort_vorschlaege: ['Neu'],
							schlagwort_vorschlaege_neu: []
						})
					})
			);
		const vorschlag = erzeugeDnbSchlagwortVorschlag();

		const erste = vorschlag.lade('9783423083003');
		await vorschlag.lade(ISBN);
		ersteAntwort({
			ok: true,
			json: async () => ({
				dnb_satz: true,
				schlagwort_vorschlaege: ['Alt'],
				schlagwort_vorschlaege_neu: []
			})
		});
		await erste;

		expect(vorschlag.liste(ISBN)).toEqual(['Neu']);
		expect(vorschlag.status('9783423083003')).toBe('');
	});
});
