import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../../../../lib/stores/bestaetigung.svelte.js', () => ({ bestaetigen: vi.fn() }));
vi.mock('$lib/store.svelte.js', () => ({ appState: { bookToEdit: null }, showToast: vi.fn() }));

import { apiFetch } from '../../../../lib/apiFetch.js';
import { showToast } from '$lib/store.svelte.js';
import IsbnFeld from './IsbnFeld.svelte';
import { erzeugeIsbnAbfrage } from './isbnAbfrage.svelte.js';

const ISBN = '9783791504650';

/** @param {number} status @param {any} koerper */
const antwort = (status, koerper) =>
	/** @type {any} */ ({ ok: status < 400, status, json: async () => koerper });

/** Der eigene Katalog kennt die ISBN nicht; die Katalogdienste antworten mit `dienste`.
 * @param {() => any} dienste */
function server(dienste) {
	vi.mocked(apiFetch).mockImplementation(async (url) =>
		String(url).startsWith('/api/lookup/') ? dienste() : antwort(200, { data: { vorhanden: null } })
	);
}

/** @param {any} formular */
function feld(formular) {
	const abfrage = erzeugeIsbnAbfrage(
		() => formular,
		() => undefined
	);
	const screen = render(IsbnFeld, { formular, wirdGescannt: false, abfrage });
	const eingabe = /** @type {HTMLInputElement} */ (screen.getByLabelText('ISBN'));
	/** Der Text unter dem Feld, an den aria-describedby zeigt. */
	const hinweis = () => {
		const id = eingabe.getAttribute('aria-describedby');
		return id ? (document.getElementById(id)?.textContent ?? '') : '';
	};
	return { eingabe, hinweis };
}
const fertig = () => new Promise((r) => setTimeout(r, 0));

beforeEach(() => vi.clearAllMocks());

// Eine Abfrage ohne Treffer sagt am Feld, woran es lag, und der Satz bleibt stehen: „Der
// Dienst ist fort" und „die ISBN kennt niemand" führen zu verschiedenem Handeln.
describe('IsbnFeld: der Ausgang einer Abfrage ohne Treffer', () => {
	it('die Katalogdienste sind nicht erreichbar: Fehler am Feld', async () => {
		server(() => antwort(502, { error: 'Katalogdienste nicht erreichbar' }));
		const formular = { id: null, isbn: ISBN, title: '' };
		const { eingabe, hinweis } = feld(formular);

		await fireEvent.blur(eingabe);
		await fertig();

		expect(hinweis()).toContain('Die Katalogdienste sind nicht erreichbar.');
		expect(hinweis()).toContain('Angaben von Hand eintragen');
		expect(eingabe.getAttribute('aria-invalid')).toBe('true');
		expect(showToast).toHaveBeenCalledWith(hinweis(), 'error');
		expect(formular.title).toBe('');
	});

	it('die ISBN kennt kein Dienst: ein Hinweis, kein Fehler', async () => {
		server(() => antwort(404, { error: 'metadaten nicht gefunden' }));
		const { eingabe, hinweis } = feld({ id: null, isbn: ISBN, title: '' });

		await fireEvent.blur(eingabe);
		await fertig();

		expect(hinweis()).toBe('Zu dieser ISBN ist bei den Katalogdiensten nichts bekannt.');
		expect(eingabe.getAttribute('aria-invalid')).toBeNull();
		expect(showToast).toHaveBeenCalledWith(hinweis(), 'info');
	});

	it('der eigene Server antwortet nicht: Fehler am Feld', async () => {
		server(() => {
			throw new Error('Netzwerk-Timeout');
		});
		const { eingabe, hinweis } = feld({ id: null, isbn: ISBN, title: '' });

		await fireEvent.blur(eingabe);
		await fertig();

		expect(hinweis()).toContain('Die ISBN-Abfrage ist fehlgeschlagen.');
		expect(eingabe.getAttribute('aria-invalid')).toBe('true');
	});

	it('eine geänderte ISBN und ein Treffer nehmen den Satz weg', async () => {
		let dienste = () => antwort(502, {});
		server(() => dienste());
		const formular = { id: null, isbn: ISBN, title: '' };
		const { eingabe, hinweis } = feld(formular);
		await fireEvent.blur(eingabe);
		await fertig();
		expect(hinweis()).not.toBe('');

		await fireEvent.input(eingabe, { target: { value: '9783551551672' } });
		expect(hinweis(), 'beim Tippen').toBe('');

		await fireEvent.blur(eingabe);
		await fertig();
		expect(hinweis()).not.toBe('');
		dienste = () => antwort(200, { data: { title: 'Tintenherz' } });
		await fireEvent.blur(eingabe);
		await fertig();
		expect(hinweis(), 'nach dem Treffer').toBe('');
		expect(formular.title).toBe('Tintenherz');
	});
});
