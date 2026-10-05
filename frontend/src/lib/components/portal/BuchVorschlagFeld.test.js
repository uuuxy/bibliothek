import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import BuchVorschlagFeld from './BuchVorschlagFeld.svelte';
import { apiFetch } from '../../apiFetch.js';

vi.mock('../../apiFetch.js', () => ({ apiFetch: vi.fn() }));

// Das Feld schlägt Titel aus dem Katalog vor und bleibt dabei ein Textfeld: Ein Vorschlag
// füllt es, ein eigener Text bleibt stehen, und die Tabulatortaste führt ins nächste Feld
// statt durch die Vorschläge.

const titel = [
	{ id: 't1', titel: 'Markl Biologie 1', autor: 'Markl', isbn: '978-3-12-150010-7' },
	{ id: 't2', titel: 'Markl Biologie 2', autor: 'Markl', isbn: '978-3-12-150020-6' }
];

/** @param {any[]} liste */
const antwort = (liste) =>
	/** @type {any} */ ({ ok: true, json: async () => liste, headers: new Headers() });

/** Tippt einen Text und wartet die Verzögerung der Suche ab. @param {HTMLElement} feld @param {string} text */
async function tippe(feld, text) {
	await fireEvent.input(feld, { target: { value: text } });
	await vi.advanceTimersByTimeAsync(300);
}

describe('BuchVorschlagFeld', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockResolvedValue(antwort(titel));
	});
	afterEach(() => vi.useRealTimers());

	it('schlägt beim Tippen Titel vor und sucht im Katalog des Portals', async () => {
		const s = render(BuchVorschlagFeld, { label: 'Welches Buch?' });
		const feld = s.getByRole('combobox', { name: 'Welches Buch?' });
		expect(feld.getAttribute('aria-expanded')).toBe('false');

		await tippe(feld, 'Markl');

		expect(vi.mocked(apiFetch).mock.calls.at(-1)?.[0]).toBe('/api/public/opac/suche?q=Markl');
		// Vor dem Titel steht das Cover; ohne Bild ist das der Anfangsbuchstabe.
		const zeilen = s.getAllByRole('option').map((o) => o.textContent?.replace(/\s+/g, ' ').trim());
		expect(zeilen).toHaveLength(2);
		expect(zeilen[0]).toMatch(/Markl Biologie 1 Markl · 978-3-12-150010-7$/);
		expect(zeilen[1]).toMatch(/Markl Biologie 2 Markl · 978-3-12-150020-6$/);
		expect(feld.getAttribute('aria-expanded')).toBe('true');
	});

	it('übernimmt den mit den Pfeiltasten gewählten Vorschlag und schließt die Liste', async () => {
		const s = render(BuchVorschlagFeld, { label: 'Welches Buch?' });
		const feld = /** @type {HTMLInputElement} */ (s.getByRole('combobox'));
		await tippe(feld, 'Markl');

		await fireEvent.keyDown(feld, { key: 'ArrowDown' });
		await fireEvent.keyDown(feld, { key: 'ArrowDown' });
		const aktiv = feld.getAttribute('aria-activedescendant');
		expect(document.getElementById(/** @type {string} */ (aktiv))?.textContent).toContain(
			'Markl Biologie 2'
		);
		await fireEvent.keyDown(feld, { key: 'Enter' });

		expect(feld.value).toBe('Markl Biologie 2');
		expect(s.queryByRole('listbox')).toBeNull();
		// Der gewählte Titel schlägt sich nicht selbst wieder vor.
		await vi.advanceTimersByTimeAsync(300);
		expect(s.queryByRole('listbox')).toBeNull();
	});

	// Die Pfeiltaste trifft noch die alte Liste, während die nächste Suche unterwegs ist. Hinge
	// die Markierung an der Stelle, stünde sie nach dem Eintreffen auf einem anderen Titel.
	it('hält die Markierung am Titel, wenn neue Vorschläge eintreffen', async () => {
		const s = render(BuchVorschlagFeld, { label: 'Welches Buch?' });
		const feld = /** @type {HTMLInputElement} */ (s.getByRole('combobox'));
		await tippe(feld, 'Markl');

		vi.mocked(apiFetch).mockResolvedValue(
			antwort([{ id: 't0', titel: 'Markl Biologie 0', autor: 'Markl' }, ...titel])
		);
		await fireEvent.input(feld, { target: { value: 'Markl B' } });
		await fireEvent.keyDown(feld, { key: 'ArrowDown' });
		await vi.advanceTimersByTimeAsync(300);
		expect(s.getAllByRole('option')).toHaveLength(3);
		await fireEvent.keyDown(feld, { key: 'Enter' });

		expect(feld.value).toBe('Markl Biologie 1');
	});

	it('lässt eigenen Text stehen: Tab schließt die Vorschläge, ohne zu wählen', async () => {
		const s = render(BuchVorschlagFeld, { label: 'Welches Buch?' });
		const feld = /** @type {HTMLInputElement} */ (s.getByRole('combobox'));
		await tippe(feld, 'Markl');
		expect(s.queryByRole('listbox')).toBeTruthy();

		await fireEvent.keyDown(feld, { key: 'Tab' });

		expect(s.queryByRole('listbox')).toBeNull();
		expect(feld.value).toBe('Markl');
	});

	it('zeigt ohne Treffer keine Liste', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort([]));
		const s = render(BuchVorschlagFeld, { label: 'Welches Buch?' });
		const feld = s.getByRole('combobox');

		await tippe(feld, '8G3 hat die falschen Bücher');

		expect(s.queryByRole('listbox')).toBeNull();
		expect(feld.getAttribute('aria-expanded')).toBe('false');
	});
});
