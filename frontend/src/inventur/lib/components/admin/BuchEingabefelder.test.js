import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', () => ({ apiFetch: vi.fn(), apiPut: vi.fn() }));
vi.mock('$lib/store.svelte.js', () => ({ showToast: vi.fn() }));

import { apiFetch } from '../../../../lib/apiFetch.js';
import BuchEingabefelder from './BuchEingabefelder.svelte';
import { erzeugeDnbSchlagwortVorschlag } from '../../../../lib/utils/dnbSchlagwortVorschlag.svelte.js';
import { leeresBuchFormular } from './buch_form_optionen.js';

const DNB = '/api/schlagworte/dnb-vorschlag?isbn=9783751200530';

// Der Schlagwort-Vorschlag der DNB im Buchformular (entschieden am 30.09.2026): Holt die ISBN
// Titel und Autor, kommen die Schlagworte der DNB von selbst dazu — ein neuer Titel entsteht
// so nicht mehr ohne Vorschlag. Bei einem Titel, den es schon gibt, fragt erst der Knopf, und
// angeboten wird nur, was er noch nicht trägt.
beforeEach(() => {
	vi.clearAllMocks();
	vi.mocked(apiFetch).mockImplementation(async (url) => {
		const u = String(url);
		if (u.startsWith('/api/lookup/')) {
			return /** @type {any} */ ({
				ok: true,
				status: 200,
				json: async () => ({ data: { title: 'Dunkelnacht', author: 'Boie, Kirsten' } })
			});
		}
		if (u === DNB) {
			return /** @type {any} */ ({
				ok: true,
				status: 200,
				json: async () => ({
					dnb_satz: true,
					schlagwort_vorschlaege: ['Freundschaft', 'Krieg'],
					schlagwort_vorschlaege_neu: ['Schulstress']
				})
			});
		}
		return /** @type {any} */ ({ ok: true, status: 200, json: async () => [] });
	});
});

/** @param {any} felder — über dem leeren Formular der Seite (leeresBuchFormular) */
function maske(felder) {
	return render(BuchEingabefelder, {
		formular: { ...leeresBuchFormular(), ...felder },
		dnbVorschlag: erzeugeDnbSchlagwortVorschlag()
	});
}
const dnbGefragt = () => vi.mocked(apiFetch).mock.calls.some(([url]) => url === DNB);

describe('BuchEingabefelder: Schlagworte aus der DNB', () => {
	it('bringt sie nach der ISBN-Abfrage eines neuen Titels von selbst', async () => {
		const screen = maske({ isbn: '978-3-7512-0053-0', title: '', schlagworte: [] });
		expect(dnbGefragt(), 'erst die Abfrage, nicht schon das Öffnen').toBe(false);

		await fireEvent.click(
			screen.getByRole('button', { name: 'Daten aus dem Internet aktualisieren' })
		);

		const liste = await vi.waitFor(() =>
			screen.getByRole('group', { name: 'Vorschläge aus der DNB' })
		);
		expect(liste.textContent).toContain('Freundschaft');
		expect(
			screen.getByRole('group', { name: 'Neue Schlagworte aus der DNB' }).textContent
		).toContain('Schulstress');
		expect(screen.queryByRole('button', { name: 'Vorschläge aus der DNB' })).toBeNull();
	});

	it('fragt bei einem vorhandenen Titel erst auf Knopfdruck und nur nach dem, was fehlt', async () => {
		const screen = maske({
			id: 't-1',
			isbn: '978-3-7512-0053-0',
			title: 'Dunkelnacht',
			schlagworte: ['Krieg']
		});
		expect(screen.queryByRole('group', { name: 'Vorschläge aus der DNB' })).toBeNull();
		expect(dnbGefragt()).toBe(false);

		await fireEvent.click(screen.getByRole('button', { name: 'Vorschläge aus der DNB' }));

		const liste = await vi.waitFor(() =>
			screen.getByRole('group', { name: 'Vorschläge aus der DNB' })
		);
		expect(liste.textContent).toContain('Freundschaft');
		expect(liste.textContent, 'trägt der Titel schon').not.toContain('Krieg');
	});
});
