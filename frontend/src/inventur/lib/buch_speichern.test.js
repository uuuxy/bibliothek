import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../../lib/apiFetch.js', () => ({ apiFetch: vi.fn() }));
import { apiFetch } from '../../lib/apiFetch.js';
import { speichereBuch, stehtInSicht, DubletteFehler } from './buch_speichern.js';

/** @param {number} status @param {any} koerper */
const antwort = (status, koerper) =>
	/** @type {any} */ ({ ok: status < 400, status, json: async () => koerper });

/** Die Maske eines vorhandenen Titels trägt den Stand vom Öffnen (titelFuerMaske).
 * @param {any} formular */
const geoeffnet = (formular) => ({ ...formular, geladen: { ...formular } });

// Eine vergebene ISBN lehnt der Server mit 409 ab und nennt den Titel, der sie trägt. Ohne
// Exemplar steht dieser Titel in keiner Suche — die Maske braucht seine Kennung, um zu ihm
// zu führen.
describe('buch_speichern: speichereBuch', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
	});

	it('legt ohne id an und ändert mit id', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(201, { data: { id: 'neu', stock: 1 } }));
		expect(await speichereBuch({ id: null, isbn: '978', stock: 1 })).toEqual({
			id: 'neu',
			stock: 1
		});
		await speichereBuch(geoeffnet({ id: 'abc', isbn: '978', stock: 2 }));
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

	it('eine vergebene ISBN ist kein gleichnamiger Titel', async () => {
		const vorhanden = { id: 'titel-1', title: 'Drachenreiter', ohneExemplar: false };
		vi.mocked(apiFetch).mockResolvedValue(antwort(409, { error: 'vergeben', vorhanden }));
		const fehler = await speichereBuch({ id: null, isbn: '978' }).catch((e) => e);
		expect(fehler.gleicherTitel).toBe(false);
	});

	// Ohne ISBN nennt der Server den Titel, der gleich heißt, und kennzeichnet die Antwort: Die
	// Maske fragt dann, statt abzulehnen.
	it('409 mit gleicherTitel wird ein DubletteFehler, der die Frage erlaubt', async () => {
		const vorhanden = { id: 'titel-2', title: 'Bild der Wissenschaft', ohneExemplar: false };
		vi.mocked(apiFetch).mockResolvedValue(
			antwort(409, { error: 'Ohne ISBN steht schon …', vorhanden, gleicherTitel: true })
		);
		const fehler = await speichereBuch({ id: null, isbn: '' }).catch((e) => e);
		expect(fehler).toBeInstanceOf(DubletteFehler);
		expect(fehler.gleicherTitel).toBe(true);
		expect(fehler.vorhanden).toEqual(vorhanden);
	});

	it('„anderes Medium" geht nur nach der Antwort darauf mit', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(201, { data: { id: 'neu' } }));
		const rumpf = (/** @type {number} */ n) =>
			JSON.parse(String(vi.mocked(apiFetch).mock.calls[n][1]?.body));

		await speichereBuch({ id: null, isbn: '', title: 'Heft 3' });
		expect('anderesMedium' in rumpf(0)).toBe(false);
		expect(rumpf(0).isbn).toBe('');

		await speichereBuch({ id: null, isbn: '', title: 'Heft 3' }, { anderesMedium: true });
		expect(rumpf(1).anderesMedium).toBe(true);
	});

	it('409 ohne vorhandenen Titel bleibt eine einfache Meldung', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(409, { error: 'existiert bereits' }));
		const fehler = await speichereBuch({ id: null, isbn: '978' }).catch((e) => e);
		expect(fehler).not.toBeInstanceOf(DubletteFehler);
		expect(fehler.message).toBe('existiert bereits');
	});

	it('ein Bestand ohne Zahl wird nicht mitgeschickt', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(200, { data: { id: 'abc' } }));
		await speichereBuch(geoeffnet({ id: 'abc', isbn: '978', stock: undefined }));
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
