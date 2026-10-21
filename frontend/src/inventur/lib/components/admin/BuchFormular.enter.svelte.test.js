import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', async (importOriginal) => ({
	...(await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('../../../../lib/stores/bestaetigung.svelte.js', () => ({ bestaetigen: vi.fn() }));
vi.mock('$lib/store.svelte.js', () => ({ appState: { bookToEdit: null }, melde: vi.fn() }));

import { apiFetch } from '../../../../lib/apiFetch.js';
import BuchFormular from './BuchFormular.svelte';
import { leeresBuchFormular } from './buchFormularRegeln.js';

const ISBN = '9783060130764';

/** @param {number} status @param {any} koerper */
const antwort = (status, koerper) =>
	/** @type {any} */ ({ ok: status < 400, status, json: async () => koerper });

/** @param {any} formular */
function maske(formular) {
	const onSave = vi.fn();
	const screen = render(BuchFormular, {
		formular,
		onClose: () => {},
		onSave,
		onCoverUpload: () => {},
		onCoverNeuHolen: () => {},
		onAssignClass: () => {}
	});
	return { screen, onSave };
}
const kurz = () => new Promise((r) => setTimeout(r, 20));

beforeEach(() => {
	vi.clearAllMocks();
	vi.mocked(apiFetch).mockImplementation(async (url) => {
		const u = String(url);
		if (u.startsWith('/api/books/vorhanden')) return antwort(200, { data: { vorhanden: null } });
		if (u.startsWith('/api/lookup/')) return antwort(404, {});
		return antwort(200, []);
	});
});

// Die Felder hängen ihr onkeydown über Svelte an die Wurzel der Seite. Die Aktion der Maske
// muss deren preventDefault sehen, sonst speicherte das Enter der ISBN-Abfrage mit.
describe('BuchFormular: Enter speichert', () => {
	it('aus dem Titel wie der Klick auf „Speichern"', async () => {
		const formular = $state({ ...leeresBuchFormular(), title: 'Krabat', signatur: 'Pre' });
		const { screen, onSave } = maske(formular);
		await fireEvent.keyDown(screen.getByLabelText('Titel *'), { key: 'Enter' });
		await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
	});

	it('im ISBN-Feld fragt Enter ab und speichert nicht, auch im leeren Feld', async () => {
		const formular = $state({
			...leeresBuchFormular(),
			title: 'Krabat',
			signatur: 'Pre',
			isbn: ISBN
		});
		const { screen, onSave } = maske(formular);
		const isbn = screen.getByLabelText('ISBN');
		await fireEvent.keyDown(isbn, { key: 'Enter' });
		await kurz();
		expect(vi.mocked(apiFetch).mock.calls.some(([u]) => String(u).includes(ISBN))).toBe(true);

		formular.isbn = '';
		await fireEvent.keyDown(isbn, { key: 'Enter' });
		await kurz();
		expect(onSave).not.toHaveBeenCalled();
	});

	it('im Feld der Schlagworte übernimmt Enter das Wort und speichert nicht', async () => {
		const formular = $state({ ...leeresBuchFormular(), title: 'Krabat', signatur: 'Pre' });
		const { screen, onSave } = maske(formular);
		const schlagworte = screen.getByLabelText('Schlagworte');
		await fireEvent.input(schlagworte, { target: { value: 'Sorben' } });
		await fireEvent.keyDown(schlagworte, { key: 'Enter' });
		await kurz();
		expect(formular.schlagworte).toEqual(['Sorben']);
		expect(onSave).not.toHaveBeenCalled();
	});

	it('das Enter am Ende eines Scans in den Titel speichert nicht', async () => {
		const formular = $state({ ...leeresBuchFormular(), title: 'Krabat', signatur: 'Pre' });
		const { screen, onSave } = maske(formular);
		const titel = screen.getByLabelText('Titel *');
		/** @param {string} key @param {number} zeit */
		const druecke = (key, zeit) => {
			const e = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true });
			Object.defineProperty(e, 'timeStamp', { value: zeit });
			titel.dispatchEvent(e);
		};
		[...'B-00123'].forEach((key, i) => druecke(key, 1000 + i * 5));
		druecke('Enter', 1040);
		await kurz();
		expect(onSave).not.toHaveBeenCalled();
	});
});
