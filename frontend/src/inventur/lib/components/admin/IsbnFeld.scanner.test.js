import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../../../../lib/stores/bestaetigung.svelte.js', () => ({ bestaetigen: vi.fn() }));
vi.mock('$lib/store.svelte.js', () => ({ appState: { bookToEdit: null }, showToast: vi.fn() }));

import { apiFetch } from '../../../../lib/apiFetch.js';
import IsbnFeld from './IsbnFeld.svelte';

const ISBN = '9783791504650';
const KATALOG = '/api/books/vorhanden';
const DIENSTE = '/api/lookup/';

/** @param {any} koerper */
const antwort = (koerper) =>
	/** @type {any} */ ({ ok: true, status: 200, json: async () => koerper });

/** @param {string} anfang */
const aufrufe = (anfang) =>
	vi.mocked(apiFetch).mock.calls.filter(([url]) => String(url).startsWith(anfang)).length;

/** @param {any} formular @param {boolean} [wirdGescannt] */
function feld(formular, wirdGescannt = false) {
	const screen = render(IsbnFeld, { formular, wirdGescannt });
	return /** @type {HTMLInputElement} */ (screen.getByLabelText('ISBN'));
}
const fertig = () => new Promise((r) => setTimeout(r, 0));

beforeEach(() => {
	vi.clearAllMocks();
	vi.mocked(apiFetch).mockImplementation(async (url) =>
		String(url).startsWith(DIENSTE) ? antwort({ data: {} }) : antwort({ data: { vorhanden: null } })
	);
});

// Ein Handscanner tippt blind in das Feld mit dem Fokus und schließt mit der Eingabetaste.
describe('IsbnFeld: Handscanner', () => {
	it('eine neue Maske stellt den Fokus ins ISBN-Feld', () => {
		const eingabe = feld({ id: null, isbn: '', title: '' });
		expect(document.activeElement).toBe(eingabe);
	});

	it('ein vorhandener Titel nicht: Ein Scan überschriebe seine ISBN', () => {
		const eingabe = feld({ id: 'titel-1', isbn: ISBN, title: 'Tintenherz' });
		expect(document.activeElement).not.toBe(eingabe);
	});

	it('hinter dem Kamera-Fenster nicht', () => {
		const eingabe = feld({ id: null, isbn: '', title: '' }, true);
		expect(document.activeElement).not.toBe(eingabe);
	});

	it('die Eingabetaste fragt wie das Verlassen und markiert die ISBN', async () => {
		const eingabe = feld({ id: null, isbn: ISBN, title: '' });

		await fireEvent.keyDown(eingabe, { key: 'Enter' });
		await fertig();

		expect([aufrufe(KATALOG), aufrufe(DIENSTE)]).toEqual([1, 1]);
		expect([eingabe.selectionStart, eingabe.selectionEnd]).toEqual([0, ISBN.length]);
	});

	it('das Verlassen nach der Eingabetaste fragt nicht noch einmal, ein späteres wieder', async () => {
		const eingabe = feld({ id: null, isbn: ISBN, title: '' });
		await fireEvent.keyDown(eingabe, { key: 'Enter' });
		await fertig();
		expect([aufrufe(KATALOG), aufrufe(DIENSTE)], 'nach der Eingabetaste').toEqual([1, 1]);

		await fireEvent.blur(eingabe);
		await fertig();
		expect([aufrufe(KATALOG), aufrufe(DIENSTE)], 'nach dem Verlassen').toEqual([1, 1]);

		await fireEvent.blur(eingabe);
		await fertig();
		expect([aufrufe(KATALOG), aufrufe(DIENSTE)]).toEqual([2, 2]);
	});

	it('andere Tasten und ein leeres Feld fragen nicht', async () => {
		const leer = feld({ id: null, isbn: '', title: '' });
		await fireEvent.keyDown(leer, { key: 'Enter' });
		const voll = feld({ id: null, isbn: ISBN, title: '' });
		await fireEvent.keyDown(voll, { key: '7' });
		await fertig();
		expect(vi.mocked(apiFetch)).not.toHaveBeenCalled();
	});
});
