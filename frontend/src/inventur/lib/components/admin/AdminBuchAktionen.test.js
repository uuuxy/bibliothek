import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../../../../lib/stores/bestaetigung.svelte.js', () => ({
	bestaetigen: vi.fn(),
	loeschenBestaetigen: vi.fn()
}));
vi.mock('../../../../lib/stores/authStore.svelte.js', () => ({ authStore: { currentUser: null } }));
vi.mock('$lib/store.svelte.js', () => ({
	appState: { bestandsAnsicht: 'mit', bookToEdit: null, selectedBook: null },
	showToast: vi.fn()
}));

import { apiFetch } from '../../../../lib/apiFetch.js';
import { bestaetigen } from '../../../../lib/stores/bestaetigung.svelte.js';
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
