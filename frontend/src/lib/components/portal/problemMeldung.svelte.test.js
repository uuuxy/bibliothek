import { describe, it, expect, vi, beforeEach } from 'vitest';
import { erzeugeProblemMeldung, OHNE_BUCH } from './problemMeldung.svelte.js';
import { apiFetch } from '../../apiFetch.js';
import { toastStore } from '../../stores/toastStore.svelte.js';

vi.mock('../../apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../../stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));

/** Der Rumpf der letzten Anfrage an die Tür für Anliegen. */
function gesendet() {
	const ruf = vi.mocked(apiFetch).mock.calls.findLast(([url]) => url === '/api/anliegen');
	return ruf ? JSON.parse(/** @type {any} */ (ruf[1]).body) : null;
}

describe('Problem melden', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(toastStore.addToast).mockReset();
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ id: 'neu' }) })
		);
	});

	// Am Treffer ist das Buch gewählt: Sein Titel steht in der Meldung, nicht eine Eingabe.
	it('schickt am Treffer den Titel des Buchs als Meldung', async () => {
		const nachSenden = vi.fn();
		const meldung = erzeugeProblemMeldung(nachSenden);
		meldung.oeffne('titel-1');
		const f = meldung.form('titel-1');
		f.klasse = ' 8G3 ';
		f.text = ' falsche Auflage ';

		expect(await meldung.senden('titel-1', 'Markl Biologie 2')).toBe(true);

		expect(gesendet()).toEqual({
			art: 'meldung',
			titel_text: 'Markl Biologie 2',
			klasse: '8G3',
			kommentar: 'falsche Auflage'
		});
		expect(nachSenden).toHaveBeenCalled();
		// Das Formular ist danach zu und leer.
		expect(meldung.form('titel-1')).toMatchObject({ open: false, klasse: '', text: '' });
	});

	it('nimmt ohne Buch den Text aus „Worum geht es?"', async () => {
		const meldung = erzeugeProblemMeldung(vi.fn());
		meldung.oeffne(OHNE_BUCH);
		const f = meldung.form(OHNE_BUCH);
		f.worum = 'die Bücher der 8G3';
		f.text = 'drei fehlen';

		expect(await meldung.senden(OHNE_BUCH)).toBe(true);
		expect(gesendet().titel_text).toBe('die Bücher der 8G3');
	});

	// Eine Meldung ohne Beschreibung nennt nur ein Buch; ohne Gegenstand nennt sie nichts.
	it.each([
		['ohne Beschreibung', 'titel-1', 'Markl Biologie 2', { text: '   ' }],
		['ohne Gegenstand', OHNE_BUCH, undefined, { worum: ' ', text: 'drei fehlen' }]
	])('schickt %s nichts', async (_was, key, titel, eingabe) => {
		const meldung = erzeugeProblemMeldung(vi.fn());
		meldung.oeffne(key);
		Object.assign(meldung.form(key), eingabe);

		expect(await meldung.senden(key, titel)).toBe(false);
		expect(gesendet()).toBeNull();
		expect(meldung.form(key).open).toBe(true);
	});

	// Lehnt die Bibliothek die Meldung ab, bleibt die Eingabe stehen und das Formular offen.
	it('behält die Eingabe, wenn die Meldung nicht angenommen wird', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: false, json: async () => ({ error: 'gerade nicht möglich' }) })
		);
		const nachSenden = vi.fn();
		const meldung = erzeugeProblemMeldung(nachSenden);
		meldung.oeffne('titel-1');
		meldung.form('titel-1').text = 'Seiten fehlen';

		expect(await meldung.senden('titel-1', 'Markl Biologie 2')).toBe(false);

		expect(meldung.form('titel-1')).toMatchObject({
			open: true,
			text: 'Seiten fehlen',
			sending: false
		});
		expect(vi.mocked(toastStore.addToast)).toHaveBeenCalledWith('gerade nicht möglich', 'error');
		expect(nachSenden).not.toHaveBeenCalled();
	});

	it('schließt ein Formular, ohne die Eingabe zu verwerfen', () => {
		const meldung = erzeugeProblemMeldung(vi.fn());
		meldung.oeffne('titel-1');
		meldung.form('titel-1').text = 'angefangen';
		meldung.schliesse('titel-1');

		expect(meldung.form('titel-1')).toMatchObject({ open: false, text: 'angefangen' });
	});
});
