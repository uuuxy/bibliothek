import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', async (importOriginal) => ({
	...(await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('../../../../lib/stores/bestaetigung.svelte.js', () => ({ bestaetigen: vi.fn() }));
vi.mock('$lib/store.svelte.js', () => ({ appState: { bookToEdit: null }, showToast: vi.fn() }));

import { apiFetch } from '../../../../lib/apiFetch.js';
import { bestaetigen } from '../../../../lib/stores/bestaetigung.svelte.js';
import BuchFormular from './BuchFormular.svelte';
import { leeresBuchFormular } from './buch_form_optionen.js';

const ISBN = '9783060130764';

/** @param {number} status @param {any} koerper */
const antwort = (status, koerper) =>
	/** @type {any} */ ({ ok: status < 400, status, json: async () => koerper });

/**
 * Der Server der Maske: was der eigene Katalog zur ISBN sagt und was die Katalogdienste
 * liefern. Alles Übrige (Systematik, Schlagworte) antwortet mit einer leeren Liste.
 * @param {any} vorhanden @param {() => Promise<any>} [dienste]
 */
function server(vorhanden, dienste = async () => antwort(404, {})) {
	vi.mocked(apiFetch).mockImplementation(async (url) => {
		const u = String(url);
		if (u.startsWith('/api/books/vorhanden')) return antwort(200, { data: vorhanden });
		if (u.startsWith('/api/lookup/')) return dienste();
		return antwort(200, []);
	});
}

/** Verlässt das ISBN-Feld und klickt „Speichern", wie ein Klick auf den Knopf es tut.
 * @param {any} formular @param {() => void} onSave */
async function verlasseIsbnUndSpeichere(formular, onSave) {
	const screen = render(BuchFormular, {
		formular,
		onClose: () => {},
		onSave,
		onCoverUpload: () => {},
		onCoverNeuHolen: () => {},
		onAssignClass: () => {}
	});
	await fireEvent.blur(screen.getByLabelText('ISBN'));
	await fireEvent.click(screen.getByRole('button', { name: 'Speichern' }));
}
const kurz = () => new Promise((r) => setTimeout(r, 20));

beforeEach(() => vi.clearAllMocks());

// Ein Klick auf „Speichern" verlässt das ISBN-Feld, und dessen Abfrage beginnt. Der Server
// trägt beim Speichern nichts nach: Ginge der Titel sofort hinaus, fehlten ihm die Angaben,
// die einen Augenblick später in der Maske stehen.
describe('BuchFormular: Speichern während der ISBN-Abfrage', () => {
	it('wartet auf die Angaben der Katalogdienste', async () => {
		/** @type {() => void} */
		let antworte = () => {};
		const dienste = new Promise((gibFrei) => (antworte = () => gibFrei(undefined)));
		server({ vorhanden: null }, async () => {
			await dienste;
			return antwort(200, { data: { title: 'Green Line 3' } });
		});
		const formular = $state({ ...leeresBuchFormular(), isbn: ISBN, signatur: 'Eng 3' });
		let titelBeimSpeichern = null;
		const onSave = vi.fn(() => (titelBeimSpeichern = formular.title));

		await verlasseIsbnUndSpeichere(formular, onSave);
		await kurz();
		expect(onSave, 'die Abfrage läuft noch').not.toHaveBeenCalled();

		antworte();
		await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
		expect(titelBeimSpeichern).toBe('Green Line 3');
	});

	it('speichert nicht, wenn die Abfrage nach einem vorhandenen Titel gefragt hat', async () => {
		server({
			vorhanden: { id: 'titel-1', title: 'Green Line 3', ohneExemplar: false },
			meldung: 'Diese ISBN trägt schon der Titel „Green Line 3“.'
		});
		vi.mocked(bestaetigen).mockResolvedValue(false);
		const formular = $state({ ...leeresBuchFormular(), isbn: ISBN, signatur: 'Eng 3' });
		const onSave = vi.fn();

		await verlasseIsbnUndSpeichere(formular, onSave);
		await waitFor(() => expect(bestaetigen).toHaveBeenCalledTimes(1));
		await kurz();

		expect(onSave).not.toHaveBeenCalled();
	});

	it('ohne laufende Abfrage speichert der Klick', async () => {
		server({ vorhanden: null });
		const formular = $state({
			...leeresBuchFormular(),
			title: 'Green Line 3',
			isbn: '',
			signatur: 'Eng 3'
		});
		const onSave = vi.fn();

		await verlasseIsbnUndSpeichere(formular, onSave);

		await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
	});

	// Der Handscanner: die Zeichen ins Feld, dann die Eingabetaste. Das zweite Buch wird
	// gescannt, während die Dienste zum ersten noch antworten.
	it('nach einem zweiten Scan während der Abfrage geht das zweite Buch hinaus', async () => {
		const ZWEITE = '9783551551672';
		/** @type {() => void} */
		let antworte = () => {};
		const erste = new Promise((gibFrei) => (antworte = () => gibFrei(undefined)));
		vi.mocked(apiFetch).mockImplementation(async (url) => {
			const u = String(url);
			if (u.startsWith('/api/books/vorhanden')) return antwort(200, { data: { vorhanden: null } });
			if (u === `/api/lookup/${ZWEITE}`) return antwort(200, { data: { title: 'Zweites Buch' } });
			if (u.startsWith('/api/lookup/')) {
				await erste;
				return antwort(200, { data: { title: 'Erstes Buch', author: 'Autorin des ersten' } });
			}
			return antwort(200, []);
		});
		const formular = $state({ ...leeresBuchFormular(), signatur: 'Eng 3' });
		/** @type {any} */
		let hinaus = null;
		const onSave = vi.fn(
			() => (hinaus = { isbn: formular.isbn, title: formular.title, author: formular.author })
		);
		const screen = render(BuchFormular, {
			formular,
			onClose: () => {},
			onSave,
			onCoverUpload: () => {},
			onCoverNeuHolen: () => {},
			onAssignClass: () => {}
		});
		const feld = screen.getByLabelText('ISBN');
		/** @param {string} code */
		const scanne = async (code) => {
			await fireEvent.input(feld, { target: { value: code } });
			await fireEvent.keyDown(feld, { key: 'Enter' });
		};
		const abfragenZurErsten = () =>
			vi.mocked(apiFetch).mock.calls.filter(([url]) => String(url) === `/api/lookup/${ISBN}`)
				.length;

		await scanne(ISBN);
		await waitFor(() => expect(abfragenZurErsten()).toBe(1));
		await scanne(ZWEITE);
		antworte();
		await fireEvent.blur(feld);
		await fireEvent.click(screen.getByRole('button', { name: 'Speichern' }));

		await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
		expect(hinaus).toEqual({ isbn: ZWEITE, title: 'Zweites Buch', author: '' });
	});
});

// Pflicht ist der Titel, die ISBN nicht. Fehlt der Titel, speichert der Klick nicht und führt
// ins Feld; der Fehler steht dort erst nach dem Klick und geht mit dem ersten Zeichen.
describe('BuchFormular: Speichern ohne Titel und ohne ISBN', () => {
	const FEHLER = 'Bitte den Titel eintragen. Gespeichert wird erst mit ihm.';
	beforeEach(() => {
		Element.prototype.scrollIntoView = vi.fn();
	});

	/** @param {any} formular @param {() => void} onSave */
	const maske = (formular, onSave) =>
		render(BuchFormular, {
			formular,
			onClose: () => {},
			onSave,
			onCoverUpload: () => {},
			onCoverNeuHolen: () => {},
			onAssignClass: () => {}
		});

	it('ohne Titel: der Klick speichert nicht, führt ins Feld und nennt dort den Grund', async () => {
		server({ vorhanden: null });
		const formular = $state({ ...leeresBuchFormular(), isbn: '', signatur: 'Spi 1' });
		const onSave = vi.fn();
		const screen = maske(formular, onSave);
		const titel = screen.getByLabelText('Titel *');

		expect(screen.queryByText(FEHLER), 'vor dem Klick steht kein Fehler am Feld').toBeNull();
		expect(titel.getAttribute('aria-invalid')).toBeNull();

		await fireEvent.click(screen.getByRole('button', { name: 'Speichern' }));
		await waitFor(() => expect(screen.getByText(FEHLER)).toBeTruthy());
		expect(onSave).not.toHaveBeenCalled();
		expect(document.activeElement?.id).toBe('buch-titel');
		expect(titel.getAttribute('aria-invalid')).toBe('true');

		await fireEvent.input(titel, { target: { value: 'Die Siedler von Catan' } });
		await waitFor(() => expect(screen.queryByText(FEHLER)).toBeNull());
	});

	it('ein Titel ohne ISBN geht hinaus', async () => {
		server({ vorhanden: null });
		const formular = $state({
			...leeresBuchFormular(),
			title: 'Die Siedler von Catan',
			isbn: '',
			signatur: 'Spi 1'
		});
		const onSave = vi.fn();
		const screen = maske(formular, onSave);

		await fireEvent.click(screen.getByRole('button', { name: 'Speichern' }));

		await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
		expect(
			vi.mocked(apiFetch).mock.calls.some(([url]) => String(url).startsWith('/api/lookup/'))
		).toBe(false);
	});

	it('ein vorhandener Titel ohne ISBN geht hinaus', async () => {
		server({ vorhanden: null });
		const formular = $state({
			...leeresBuchFormular(),
			id: 'titel-9',
			title: 'Bild der Wissenschaft',
			isbn: '',
			schlagworte: []
		});
		const onSave = vi.fn();
		const screen = maske(formular, onSave);

		await fireEvent.click(screen.getByRole('button', { name: 'Speichern' }));

		await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
	});
});

// „Speichern" bleibt bedienbar, auch wenn die Pflicht-Signatur fehlt: Der Klick speichert
// nicht und setzt den Fokus ins Feld, das den Grund nennt.
describe('BuchFormular: Speichern ohne Pflicht-Signatur', () => {
	beforeEach(() => {
		// jsdom rechnet kein Layout und kennt scrollIntoView nicht.
		Element.prototype.scrollIntoView = vi.fn();
	});

	it('ein Bibliotheksbuch: der Klick speichert nicht und führt ins Feld Signatur', async () => {
		server({ vorhanden: null });
		const formular = $state({ ...leeresBuchFormular(), title: 'Tintenherz', isbn: '' });
		const onSave = vi.fn();

		await verlasseIsbnUndSpeichere(formular, onSave);
		await kurz();

		expect(onSave).not.toHaveBeenCalled();
		expect(document.activeElement?.id).toBe('buch-signatur');
		expect(Element.prototype.scrollIntoView).toHaveBeenCalledTimes(1);
	});

	it('ein Lernmittel speichert ohne Signatur', async () => {
		server({ vorhanden: null });
		const formular = $state({
			...leeresBuchFormular(),
			title: 'Green Line 3',
			isbn: '',
			istLernmittel: true
		});
		const onSave = vi.fn();

		await verlasseIsbnUndSpeichere(formular, onSave);

		await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
	});
});
