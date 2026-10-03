import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../../../../lib/stores/bestaetigung.svelte.js', () => ({
	bestaetigen: vi.fn(),
	fragen: vi.fn(),
	loeschenBestaetigen: vi.fn()
}));
vi.mock('../../../../lib/stores/authStore.svelte.js', () => ({ authStore: { currentUser: null } }));
vi.mock('$lib/store.svelte.js', () => ({
	appState: { bestandsAnsicht: 'mit', bookToEdit: null, selectedBook: null },
	showToast: vi.fn()
}));

import { apiFetch } from '../../../../lib/apiFetch.js';
import { bestaetigen, fragen } from '../../../../lib/stores/bestaetigung.svelte.js';
import { appState, showToast } from '$lib/store.svelte.js';
import AdminBuchAktionen from './AdminBuchAktionen.svelte';

/** @param {number} status @param {any} koerper */
const antwort = (status, koerper) =>
	/** @type {any} */ ({ ok: status < 400, status, json: async () => koerper });
const MELDUNG = 'Diese ISBN trägt schon der Titel „Drachenreiter“.';
const DUBLETTE = antwort(409, {
	error: MELDUNG,
	vorhanden: { id: 'titel-1', title: 'Drachenreiter', ohneExemplar: true }
});

/** @param {any} formular */
function maske(formular) {
	const { component } = render(AdminBuchAktionen, {
		props: { books: [], isEditMode: true, formular }
	});
	return component;
}

beforeEach(() => {
	vi.clearAllMocks();
	// Auch eine nicht abgeholte Antwort eines früheren Tests fällt weg.
	vi.mocked(apiFetch).mockReset();
	appState.bestandsAnsicht = 'mit';
	appState.bookToEdit = null;
});

// Die ISBN eines neuen Buchs trägt schon ein Titel, den keine Suche zeigt, weil er kein
// Exemplar hat: Die Maske fragt und öffnet ihn, statt nur „existiert bereits" zu melden.
describe('AdminBuchAktionen: vergebene ISBN beim Anlegen', () => {
	it('fragt nach und öffnet den vorhandenen Titel', async () => {
		vi.mocked(apiFetch).mockResolvedValue(DUBLETTE);
		vi.mocked(bestaetigen).mockResolvedValue(true);
		await maske({
			id: null,
			isbn: '9783791504544',
			title: 'Drachenreiter',
			stock: 1
		}).saveChanges();

		const frage = vi.mocked(bestaetigen).mock.calls[0][0];
		expect(frage.aktion).toBe('Titel öffnen');
		expect(frage.text).toContain(MELDUNG);
		expect(appState.bookToEdit).toEqual({ id: 'titel-1' });
		expect(showToast).not.toHaveBeenCalled();
	});

	it('Abbrechen lässt die Maske stehen', async () => {
		vi.mocked(apiFetch).mockResolvedValue(DUBLETTE);
		vi.mocked(bestaetigen).mockResolvedValue(false);
		await maske({
			id: null,
			isbn: '9783791504544',
			title: 'Drachenreiter',
			stock: 1
		}).saveChanges();
		expect(appState.bookToEdit).toBeNull();
	});

	it('beim Ändern bleibt es bei der Meldung', async () => {
		vi.mocked(apiFetch).mockResolvedValue(DUBLETTE);
		await maske({ id: 'titel-2', isbn: '9783791504544', title: 'Anderer', stock: 1 }).saveChanges();
		expect(bestaetigen).not.toHaveBeenCalled();
		expect(showToast).toHaveBeenCalledWith(MELDUNG, 'error');
	});
});

// Pflicht ist der Titel, die ISBN nicht: Rund ein Drittel der Titel aus Littera trägt keine.
describe('AdminBuchAktionen: Titel ohne ISBN', () => {
	const rumpf = (/** @type {number} */ n) =>
		JSON.parse(String(vi.mocked(apiFetch).mock.calls[n][1]?.body));

	it('ein vorhandener Titel ohne ISBN wird gespeichert', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(200, { data: { id: 'titel-9', stock: 1 } }));
		await maske({
			id: 'titel-9',
			isbn: '',
			title: 'Bild der Wissenschaft',
			signatur: 'Z 1',
			stock: 1,
			stockGesehen: 1
		}).saveChanges();

		expect(vi.mocked(apiFetch).mock.calls[0][1]?.method).toBe('PUT');
		expect(rumpf(0)).toMatchObject({ isbn: '', signatur: 'Z 1' });
		expect(showToast).toHaveBeenCalledWith('Buch erfolgreich gespeichert!', 'success');
	});

	it('ein neuer Titel ohne ISBN wird angelegt', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(201, { data: { id: 'neu', stock: 1 } }));
		await maske({ id: null, isbn: '', title: 'Die Siedler von Catan', stock: 1 }).saveChanges();

		expect(vi.mocked(apiFetch).mock.calls[0][1]?.method).toBe('POST');
		expect(rumpf(0).isbn).toBe('');
		expect(showToast).toHaveBeenCalledWith('Buch erfolgreich gespeichert!', 'success');
	});
});

// Ohne ISBN heißt ein vorhandener Titel gleich. Hefte und Bände tragen denselben Titel und
// Autor: Die Maske fragt, statt abzulehnen, und legt nur nach der Antwort „Anderes Medium" an.
describe('AdminBuchAktionen: gleichnamiger Titel ohne ISBN', () => {
	const HINWEIS = 'Ohne ISBN steht schon ein Titel „Bild der Wissenschaft“ im Katalog.';
	const GLEICH = antwort(409, {
		error: HINWEIS,
		vorhanden: { id: 'titel-7', title: 'Bild der Wissenschaft', ohneExemplar: false },
		gleicherTitel: true
	});
	const heft = () => ({ id: null, isbn: '', title: 'Bild der Wissenschaft', stock: 1 });
	const rumpf = (/** @type {number} */ n) =>
		JSON.parse(String(vi.mocked(apiFetch).mock.calls[n][1]?.body));

	it('fragt, ob es dasselbe Medium ist, mit zwei Antworten', async () => {
		vi.mocked(apiFetch).mockResolvedValue(GLEICH);
		vi.mocked(fragen).mockResolvedValue(null);
		await maske(heft()).saveChanges();

		const frage = vi.mocked(fragen).mock.calls[0][0];
		expect(frage.titel).toBe('Ist es dasselbe Medium?');
		expect(frage.text).toContain(HINWEIS);
		expect([frage.aktion, frage.abbruch]).toEqual(['Titel öffnen', 'Anderes Medium']);
		expect(bestaetigen).not.toHaveBeenCalled();
	});

	it('„Titel öffnen" führt zum vorhandenen und legt nichts an', async () => {
		vi.mocked(apiFetch).mockResolvedValue(GLEICH);
		vi.mocked(fragen).mockResolvedValue(true);
		await maske(heft()).saveChanges();

		expect(appState.bookToEdit).toEqual({ id: 'titel-7' });
		expect(apiFetch).toHaveBeenCalledTimes(1);
		expect(showToast).not.toHaveBeenCalled();
	});

	it('„Anderes Medium" schickt denselben Titel noch einmal und meldet das Speichern', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(GLEICH);
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort(201, { data: { id: 'neu', stock: 1 } }));
		vi.mocked(fragen).mockResolvedValue(false);
		await maske(heft()).saveChanges();

		expect(apiFetch).toHaveBeenCalledTimes(2);
		expect('anderesMedium' in rumpf(0)).toBe(false);
		expect(rumpf(1)).toMatchObject({ title: 'Bild der Wissenschaft', anderesMedium: true });
		expect(appState.bookToEdit).toBeNull();
		expect(showToast).toHaveBeenCalledWith('Buch erfolgreich gespeichert!', 'success');
	});

	it('den Dialog nur zu schließen ist keine Antwort: nichts wird angelegt', async () => {
		vi.mocked(apiFetch).mockResolvedValue(GLEICH);
		vi.mocked(fragen).mockResolvedValue(null);
		await maske(heft()).saveChanges();

		expect(apiFetch).toHaveBeenCalledTimes(1);
		expect(appState.bookToEdit).toBeNull();
		expect(showToast).not.toHaveBeenCalled();
	});

	it('nach dem Schließen fragt der nächste Klick auf „Speichern" wieder', async () => {
		vi.mocked(apiFetch).mockResolvedValue(GLEICH);
		vi.mocked(fragen).mockResolvedValue(null);
		const aktionen = maske(heft());
		await aktionen.saveChanges();
		await aktionen.saveChanges();

		expect(fragen).toHaveBeenCalledTimes(2);
	});
});

describe('AdminBuchAktionen: neuer Titel ohne Exemplar', () => {
	it('die Meldung nennt die Sicht, in der er steht', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(201, { data: { id: 'neu', stock: 0 } }));
		await maske({ id: null, isbn: '9783791504544', title: 'Nur Titel', stock: 0 }).saveChanges();
		expect(vi.mocked(showToast).mock.calls[0][0]).toContain('„Ohne Exemplare“');
	});

	it('mit Exemplar bleibt es bei der gewohnten Meldung', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(201, { data: { id: 'neu', stock: 1 } }));
		await maske({ id: null, isbn: '9783791504544', title: 'Mit Buch', stock: 1 }).saveChanges();
		expect(showToast).toHaveBeenCalledWith('Buch erfolgreich gespeichert!', 'success');
	});
});

// Die Rückfrage misst an der Zahl vom Öffnen der Maske, nicht an der Titelliste: Über
// „Neues Buch" und eine vergebene ISBN geöffnet, steht der Titel bei gefilterter Liste
// nicht in ihr (books ist hier leer).
describe('AdminBuchAktionen: Bestand eines vorhandenen Titels', () => {
	const titel = { id: 'titel-3', isbn: '9783791504544', title: 'Tintenherz', stockGesehen: 3 };
	const rumpf = () => JSON.parse(String(vi.mocked(apiFetch).mock.calls[0][1]?.body));

	it('verringert: fragt mit beiden Zahlen, auch wenn der Titel nicht in der Liste steht', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(200, { data: { id: 'titel-3' } }));
		vi.mocked(bestaetigen).mockResolvedValue(true);
		await maske({ ...titel, stock: 1 }).saveChanges();

		const frage = vi.mocked(bestaetigen).mock.calls[0][0];
		expect(frage.titel).toBe('Bestand von 3 auf 1 verringern?');
		expect(frage.gefaehrlich).toBe(true);
		expect(rumpf()).toMatchObject({ stock: 1, stockGesehen: 3 });
	});

	it('Abbrechen: nichts wird gespeichert', async () => {
		vi.mocked(bestaetigen).mockResolvedValue(false);
		await maske({ ...titel, stock: 1 }).saveChanges();
		expect(apiFetch).not.toHaveBeenCalled();
	});

	it('geleertes Feld: keine Rückfrage, der Bestand geht nicht mit', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(200, { data: { id: 'titel-3' } }));
		await maske({ ...titel, stock: null }).saveChanges();
		expect(bestaetigen).not.toHaveBeenCalled();
		expect('stock' in rumpf()).toBe(false);
	});

	it('unverändert: keine Rückfrage, der Bestand geht nicht mit', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(200, { data: { id: 'titel-3' } }));
		await maske({ ...titel, stock: 3 }).saveChanges();
		expect(bestaetigen).not.toHaveBeenCalled();
		expect('stock' in rumpf()).toBe(false);
	});

	it('inzwischen geändert: das Feld zeigt den Stand des Servers, die Maske bleibt offen', async () => {
		const meldung = 'Der Bestand wurde inzwischen an anderer Stelle geändert: jetzt 6 statt 3.';
		vi.mocked(apiFetch).mockResolvedValue(antwort(409, { error: meldung, bestand: 6 }));
		const formular = { ...titel, stock: 5, signatur: 'Fun 1' };
		await maske(formular).saveChanges();
		expect(showToast).toHaveBeenCalledWith(meldung, 'error');
		expect(formular).toMatchObject({ stock: 6, stockGesehen: 6, signatur: 'Fun 1' });
	});
});

// Ein Doppelklick auf „Speichern": Die zweite Anfrage träfe den Stand, den die erste eben
// geschrieben hat, und käme als Ablehnung zurück — beim neuen Titel als vergebene ISBN, bei
// einem geänderten Bestand als „inzwischen geändert".
describe('AdminBuchAktionen: zweiter Klick, solange das Speichern läuft', () => {
	it('schickt eine Anfrage und meldet einmal', async () => {
		/** @type {((wert: any) => void)[]} */
		const wartende = [];
		vi.mocked(apiFetch).mockImplementation(() => new Promise((fertig) => wartende.push(fertig)));
		const aktionen = maske({ id: null, isbn: '9783791504544', title: 'Mit Buch', stock: 1 });

		const erster = aktionen.saveChanges();
		const zweiter = aktionen.saveChanges();
		for (const fertig of wartende) fertig(antwort(201, { data: { id: 'neu', stock: 1 } }));
		await Promise.all([erster, zweiter]);

		expect(apiFetch).toHaveBeenCalledTimes(1);
		expect(showToast).toHaveBeenCalledTimes(1);
	});

	it('nach einer Ablehnung geht das Speichern wieder', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort(500, { error: 'Speichern fehlgeschlagen' }));
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort(201, { data: { id: 'neu', stock: 1 } }));
		const aktionen = maske({ id: null, isbn: '9783791504544', title: 'Mit Buch', stock: 1 });

		await aktionen.saveChanges();
		await aktionen.saveChanges();

		expect(apiFetch).toHaveBeenCalledTimes(2);
		expect(showToast).toHaveBeenLastCalledWith('Buch erfolgreich gespeichert!', 'success');
	});
});
