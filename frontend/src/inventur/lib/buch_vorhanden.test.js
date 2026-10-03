import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../../lib/apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../../lib/stores/bestaetigung.svelte.js', () => ({ bestaetigen: vi.fn() }));
vi.mock('./store.svelte.js', () => ({ appState: { bookToEdit: null }, showToast: vi.fn() }));
import { apiFetch } from '../../lib/apiFetch.js';
import { bestaetigen } from '../../lib/stores/bestaetigung.svelte.js';
import { appState, showToast } from './store.svelte.js';
import { vorhandenerTitel, frageWennVergeben } from './buch_vorhanden.js';

/** @param {number} status @param {any} koerper */
const antwort = (status, koerper) =>
	/** @type {any} */ ({ ok: status < 400, status, json: async () => koerper });

// Die Auskunft vor dem Speichern: Trägt die ISBN schon ein Titel, erfährt die Maske es bei
// der Eingabe und fragt sofort — sonst erst beim Speichern, wenn alles eingetragen ist.
describe('buch_vorhanden: die ISBN ist schon vergeben', () => {
	const MELDUNG = 'Diese ISBN trägt schon der Titel „Drachenreiter“.';
	const VORHANDEN = { id: 'titel-1', title: 'Drachenreiter', ohneExemplar: false };

	beforeEach(() => {
		vi.clearAllMocks();
		appState.bookToEdit = null;
	});

	it('fragt den Katalog nach der ISBN, auch mit Bindestrichen und Leerzeichen', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			antwort(200, { data: { vorhanden: VORHANDEN, meldung: MELDUNG } })
		);
		expect(await vorhandenerTitel('978-3 7915')).toEqual({
			meldung: MELDUNG,
			vorhanden: VORHANDEN
		});
		expect(vi.mocked(apiFetch).mock.calls[0][0]).toBe('/api/books/vorhanden?isbn=978-3%207915');
	});

	it('kein Titel mit dieser ISBN: null', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(200, { data: { vorhanden: null } }));
		expect(await vorhandenerTitel('9783791504544')).toBeNull();
	});

	it('vergeben: fragt und öffnet den vorhandenen Titel', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			antwort(200, { data: { vorhanden: VORHANDEN, meldung: MELDUNG } })
		);
		vi.mocked(bestaetigen).mockResolvedValue(true);
		expect(await frageWennVergeben('9783791504544')).toBe(true);
		const frage = vi.mocked(bestaetigen).mock.calls[0][0];
		expect(frage.titel).toBe('Vorhandenen Titel öffnen?');
		expect(frage.text).toContain(MELDUNG);
		expect(frage.aktion).toBe('Titel öffnen');
		expect(appState.bookToEdit).toEqual({ id: 'titel-1' });
	});

	it('vergeben und abgebrochen: die Maske bleibt, die ISBN gilt weiter als vergeben', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			antwort(200, { data: { vorhanden: VORHANDEN, meldung: MELDUNG } })
		);
		vi.mocked(bestaetigen).mockResolvedValue(false);
		expect(await frageWennVergeben('9783791504544')).toBe(true);
		expect(appState.bookToEdit).toBeNull();
	});

	it('frei: keine Frage', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(200, { data: { vorhanden: null } }));
		expect(await frageWennVergeben('9783791504544')).toBe(false);
		expect(bestaetigen).not.toHaveBeenCalled();
		expect(showToast).not.toHaveBeenCalled();
	});

	it.each([
		['der Server antwortet mit einem Fehler', () => Promise.resolve(antwort(500, { error: 'x' }))],
		['der Server ist nicht erreichbar', () => Promise.reject(new TypeError('Failed to fetch'))]
	])('%s: Meldung, keine Frage, die Maske geht weiter', async (_name, antwortet) => {
		vi.mocked(apiFetch).mockImplementation(antwortet);
		expect(await frageWennVergeben('9783791504544')).toBe(false);
		expect(bestaetigen).not.toHaveBeenCalled();
		expect(showToast).toHaveBeenCalledWith(
			'Ob es diese ISBN schon gibt, ließ sich nicht prüfen.',
			'warning'
		);
	});
});
