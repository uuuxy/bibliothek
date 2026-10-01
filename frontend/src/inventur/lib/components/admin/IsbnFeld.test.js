import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../../../../lib/stores/bestaetigung.svelte.js', () => ({ bestaetigen: vi.fn() }));
vi.mock('$lib/store.svelte.js', () => ({ appState: { bookToEdit: null }, showToast: vi.fn() }));

import { apiFetch } from '../../../../lib/apiFetch.js';
import { bestaetigen } from '../../../../lib/stores/bestaetigung.svelte.js';
import { appState, showToast } from '$lib/store.svelte.js';
import IsbnFeld from './IsbnFeld.svelte';

const ISBN = '9783791504650';
const MELDUNG = 'Diese ISBN trägt schon der Titel „Tintenherz“.';
const KATALOG = '/api/books/vorhanden';
const DIENSTE = '/api/lookup/';

/** @param {number} status @param {any} koerper */
const antwort = (status, koerper) =>
	/** @type {any} */ ({ ok: status < 400, status, json: async () => koerper });

/**
 * Der eigene Katalog kennt die ISBN ('vergeben'), kennt sie nicht ('frei') oder antwortet
 * nicht ('gestört'); die Katalogdienste liefern immer einen Titel.
 * @param {'vergeben'|'frei'|'gestört'} katalog
 */
function server(katalog) {
	vi.mocked(apiFetch).mockImplementation(async (url) => {
		const u = String(url);
		if (u.startsWith(KATALOG)) {
			if (katalog === 'gestört') return antwort(500, { error: 'Interner Serverfehler' });
			return antwort(200, {
				data:
					katalog === 'vergeben'
						? {
								vorhanden: { id: 'titel-1', title: 'Tintenherz', ohneExemplar: false },
								meldung: MELDUNG
							}
						: { vorhanden: null }
			});
		}
		if (u.startsWith(DIENSTE)) {
			return antwort(200, { data: { title: 'Tintenherz', author: 'Funke, Cornelia' } });
		}
		return antwort(200, {});
	});
}
/** @param {string} anfang */
const aufrufe = (anfang) =>
	vi.mocked(apiFetch).mock.calls.filter(([url]) => String(url).startsWith(anfang)).length;

/** @param {any} formular */
function feld(formular) {
	const screen = render(IsbnFeld, { formular, wirdGescannt: false });
	return {
		eingabe: screen.getByLabelText('ISBN'),
		knopf: screen.getByRole('button', { name: 'Daten aus dem Internet aktualisieren' })
	};
}
/** Wartet, bis der angestoßene Ablauf samt seiner Abfragen durch ist. */
const fertig = () => new Promise((r) => setTimeout(r, 0));

beforeEach(() => {
	vi.clearAllMocks();
	appState.bookToEdit = null;
});

// Littera prüft bei der Eingabe der ISBN auf Dubletten. Die Maske „Neues Buch" fragt deshalb
// den eigenen Katalog, sobald die ISBN im Feld steht — und nicht erst beim Speichern, wenn
// Signatur, Fach und alles Weitere eingetragen sind.
describe('IsbnFeld: neue Maske, die ISBN trägt schon ein Titel', () => {
	it('fragt beim Verlassen des Feldes und öffnet den vorhandenen Titel', async () => {
		server('vergeben');
		vi.mocked(bestaetigen).mockResolvedValue(true);
		const { eingabe } = feld({ id: null, isbn: ISBN, title: '' });

		await fireEvent.blur(eingabe);
		await fertig();

		const frage = vi.mocked(bestaetigen).mock.calls[0][0];
		expect(frage.titel).toBe('Vorhandenen Titel öffnen?');
		expect(frage.text).toContain(MELDUNG);
		expect(appState.bookToEdit).toEqual({ id: 'titel-1' });
		expect(aufrufe(DIENSTE), 'für ein vorhandenes Buch wird nichts geladen').toBe(0);
	});

	it('fragt auch, wenn der Titel schon eingetragen ist', async () => {
		server('vergeben');
		vi.mocked(bestaetigen).mockResolvedValue(false);
		const { eingabe } = feld({ id: null, isbn: ISBN, title: 'Von Hand getippt' });

		await fireEvent.blur(eingabe);
		await fertig();

		expect(bestaetigen).toHaveBeenCalledTimes(1);
		expect(appState.bookToEdit, 'Abbrechen lässt die Maske stehen').toBeNull();
		expect(aufrufe(DIENSTE)).toBe(0);
	});

	it('der Knopf fragt ebenfalls zuerst den Katalog', async () => {
		server('vergeben');
		vi.mocked(bestaetigen).mockResolvedValue(false);
		const { knopf } = feld({ id: null, isbn: ISBN, title: '' });

		await fireEvent.click(knopf);
		await fertig();

		expect(bestaetigen).toHaveBeenCalledTimes(1);
		expect(aufrufe(DIENSTE)).toBe(0);
	});
});

describe('IsbnFeld: neue Maske, die ISBN ist frei', () => {
	it('ohne Titel lädt das Verlassen des Feldes die Angaben', async () => {
		server('frei');
		const formular = { id: null, isbn: ISBN, title: '' };
		const { eingabe } = feld(formular);

		await fireEvent.blur(eingabe);
		await fertig();

		expect(bestaetigen).not.toHaveBeenCalled();
		expect([aufrufe(KATALOG), aufrufe(DIENSTE)]).toEqual([1, 1]);
		expect(formular.title).toBe('Tintenherz');
	});

	it('mit Titel wird nur der Katalog gefragt', async () => {
		server('frei');
		const formular = { id: null, isbn: ISBN, title: 'Von Hand getippt' };
		const { eingabe } = feld(formular);

		await fireEvent.blur(eingabe);
		await fertig();

		expect([aufrufe(KATALOG), aufrufe(DIENSTE)]).toEqual([1, 0]);
		expect(formular.title).toBe('Von Hand getippt');
	});

	it('ein Klick auf den Knopf verlässt auch das Feld: eine Frage, ein Nachschlagen', async () => {
		server('frei');
		const formular = { id: null, isbn: ISBN, title: 'Von Hand getippt' };
		const { eingabe, knopf } = feld(formular);

		// Beide Auslöser im selben Augenblick, wie der Browser sie bei einem Klick liefert.
		fireEvent.blur(eingabe);
		fireEvent.click(knopf);
		await fertig();

		expect([aufrufe(KATALOG), aufrufe(DIENSTE)]).toEqual([1, 1]);
		expect(formular.title, 'der Knopf lädt auch über einen eingetragenen Titel').toBe('Tintenherz');
	});

	it('lässt sich der Katalog nicht fragen, sagt die Maske es und lädt trotzdem', async () => {
		server('gestört');
		const { eingabe } = feld({ id: null, isbn: ISBN, title: '' });

		await fireEvent.blur(eingabe);
		await fertig();

		expect(showToast).toHaveBeenCalledWith(
			'Ob es diese ISBN schon gibt, ließ sich nicht prüfen.',
			'warning'
		);
		expect(bestaetigen).not.toHaveBeenCalled();
		expect(aufrufe(DIENSTE)).toBe(1);
	});
});

// Ein Titel, den es schon gibt, trägt seine ISBN selbst: Die Auskunft nennte ihn als vergeben.
describe('IsbnFeld: vorhandener Titel', () => {
	it('fragt den Katalog nicht und lädt wie bisher', async () => {
		server('vergeben');
		const { eingabe, knopf } = feld({ id: 'titel-1', isbn: ISBN, title: 'Tintenherz' });

		await fireEvent.blur(eingabe);
		await fertig();
		expect([aufrufe(KATALOG), aufrufe(DIENSTE)]).toEqual([0, 0]);

		await fireEvent.click(knopf);
		await fertig();
		expect([aufrufe(KATALOG), aufrufe(DIENSTE)]).toEqual([0, 1]);
		expect(bestaetigen).not.toHaveBeenCalled();
	});

	it('ohne ISBN sagt der Knopf, was fehlt', async () => {
		server('frei');
		const { knopf } = feld({ id: null, isbn: '', title: '' });
		await fireEvent.click(knopf);
		await fertig();
		expect(showToast).toHaveBeenCalledWith('Bitte zuerst eine ISBN eingeben.', 'error');
		expect(vi.mocked(apiFetch)).not.toHaveBeenCalled();
	});
});
