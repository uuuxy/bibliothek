import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../../lib/apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../../lib/stores/bestaetigung.svelte.js', () => ({ bestaetigen: vi.fn() }));
vi.mock('./store.svelte.js', () => ({ appState: { bookToEdit: null }, showToast: vi.fn() }));
import { apiFetch } from '../../lib/apiFetch.js';
import { bestaetigen } from '../../lib/stores/bestaetigung.svelte.js';
import { appState, showToast } from './store.svelte.js';
import {
	speichereBuch,
	stehtInSicht,
	DubletteFehler,
	vorhandenerTitel,
	frageWennVergeben
} from './buch_speichern.js';

/** @param {number} status @param {any} koerper */
const antwort = (status, koerper) =>
	/** @type {any} */ ({ ok: status < 400, status, json: async () => koerper });

// Die Auskunft vor dem Speichern: Trägt die ISBN schon ein Titel, erfährt die Maske es bei
// der Eingabe und fragt sofort — sonst erst beim Speichern, wenn alles eingetragen ist.
describe('buch_speichern: die ISBN ist schon vergeben', () => {
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

// Eine vergebene ISBN lehnt der Server mit 409 ab und nennt den Titel, der sie trägt. Ohne
// Exemplar steht dieser Titel in keiner Suche — die Maske braucht seine Kennung, um zu ihm
// zu führen.
describe('buch_speichern: speichereBuch', () => {
	beforeEach(() => vi.mocked(apiFetch).mockReset());

	it('legt ohne id an und ändert mit id', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(201, { data: { id: 'neu', stock: 1 } }));
		expect(await speichereBuch({ id: null, isbn: '978', stock: 1 })).toEqual({
			id: 'neu',
			stock: 1
		});
		await speichereBuch({ id: 'abc', isbn: '978', stock: 2 });
		const aufrufe = vi.mocked(apiFetch).mock.calls;
		expect([aufrufe[0][0], aufrufe[0][1]?.method]).toEqual(['/api/books', 'POST']);
		expect([aufrufe[1][0], aufrufe[1][1]?.method]).toEqual(['/api/books/abc', 'PUT']);
	});

	it('409 mit vorhandenem Titel wird ein DubletteFehler mit Kennung und Meldung', async () => {
		const vorhanden = { id: 'titel-1', title: 'Drachenreiter', ohneExemplar: true };
		vi.mocked(apiFetch).mockResolvedValue(
			antwort(409, { error: 'Diese ISBN trägt schon der Titel „Drachenreiter“.', vorhanden })
		);
		const fehler = await speichereBuch({ id: null, isbn: '978' }).catch((e) => e);
		expect(fehler).toBeInstanceOf(DubletteFehler);
		expect(fehler.vorhanden).toEqual(vorhanden);
		expect(fehler.message).toContain('Drachenreiter');
	});

	it('409 ohne vorhandenen Titel bleibt eine einfache Meldung', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(409, { error: 'existiert bereits' }));
		const fehler = await speichereBuch({ id: null, isbn: '978' }).catch((e) => e);
		expect(fehler).not.toBeInstanceOf(DubletteFehler);
		expect(fehler.message).toBe('existiert bereits');
	});

	it('ein Bestand ohne Zahl wird nicht mitgeschickt', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(200, { data: { id: 'abc' } }));
		await speichereBuch({ id: 'abc', isbn: '978', stock: undefined });
		const koerper = JSON.parse(String(vi.mocked(apiFetch).mock.calls[0][1]?.body));
		expect('stock' in koerper).toBe(false);
	});
});

describe('buch_speichern: stehtInSicht', () => {
	it.each([
		[1, 'mit', true],
		[0, 'mit', false],
		[0, 'ohne', true],
		[3, 'ohne', false]
	])('Bestand %s in der Sicht „%s": %s', (bestand, sicht, erwartet) => {
		expect(stehtInSicht(bestand, /** @type {'mit'|'ohne'} */ (sicht))).toBe(erwartet);
	});
});
