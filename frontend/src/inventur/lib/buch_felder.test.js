import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../../lib/apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('./store.svelte.js', () => ({ appState: {}, showToast: vi.fn() }));
import { apiFetch } from '../../lib/apiFetch.js';
import { geaenderteFelder, merkeStand, uebernimmGespeichert } from './buch_felder.js';
import { speichereBuch, titelFuerMaske } from './buch_speichern.js';

/** @param {number} status @param {any} koerper */
const antwort = (status, koerper) =>
	/** @type {any} */ ({ ok: status < 400, status, json: async () => koerper });

/** Ein Titel, wie der Einzelabruf ihn liefert. */
const titel = () => ({
	id: 'abc',
	isbn: '9783060130764',
	title: 'Natura 2',
	author: 'Beyer',
	untertitel: '',
	verlag: 'Klett',
	erscheinungsjahr: 2019,
	gradeLevel: 7,
	istLernmittel: true,
	lastCounted: '2026-01-15T00:00:00Z',
	listenpreis: 12.5,
	schlagworte: ['Biologie', 'Zelle'],
	erweiterteEigenschaften: { littera_id: 4711 },
	stock: 30,
	standorte: [{ standort: 'Regal 11', anzahl: 30 }]
});

/** Öffnet die Maske wie die Seite: Einzelabruf, dann der Stand vom Öffnen. */
async function oeffne() {
	vi.mocked(apiFetch).mockResolvedValueOnce(antwort(200, titel()));
	return titelFuerMaske({ id: 'abc' });
}

beforeEach(() => vi.mocked(apiFetch).mockReset());

// Die Maske schickte bei jedem Speichern alle Felder mit dem Stand vom Öffnen zurück: Was ein
// anderer Platz inzwischen an einem Feld gespeichert hatte, war danach wieder das alte.
describe('buch_felder: geaenderteFelder', () => {
	it('unberührte Maske: kein Feld', async () => {
		expect(geaenderteFelder(await oeffne())).toEqual({});
	});

	it('ein geändertes Feld: nur dieses', async () => {
		const formular = await oeffne();
		formular.verlag = 'Cornelsen';
		expect(geaenderteFelder(formular)).toEqual({ verlag: 'Cornelsen' });
	});

	it('zurück auf den alten Wert getippt ist keine Änderung', async () => {
		const formular = await oeffne();
		formular.verlag = 'Cornelsen';
		formular.verlag = 'Klett';
		expect(geaenderteFelder(formular)).toEqual({});
	});

	it('ein geleertes Textfeld geht leer hinaus', async () => {
		const formular = await oeffne();
		formular.verlag = '';
		expect(geaenderteFelder(formular)).toEqual({ verlag: '' });
	});

	it.each([[null], [undefined]])(
		'ein geleertes Zahlenfeld (%s) geht als null hinaus',
		async (leer) => {
			const formular = await oeffne();
			formular.listenpreis = leer;
			const felder = geaenderteFelder(formular);
			expect(felder).toEqual({ listenpreis: null });
			// undefined ließe JSON weg, und der Server hielte das Feld für nicht genannt.
			expect(JSON.parse(JSON.stringify(felder))).toHaveProperty('listenpreis', null);
		}
	);

	it.each([[null], [undefined]])(
		"leer bleibt leer: '' gegen %s ist keine Änderung",
		async (leer) => {
			const formular = await oeffne();
			formular.untertitel = leer;
			expect(geaenderteFelder(formular)).toEqual({});
		}
	);

	it('die Klasse kommt aus der Auswahl als Text und zählt als Zahl', async () => {
		const formular = await oeffne();
		formular.gradeLevel = '7';
		expect(geaenderteFelder(formular)).toEqual({});
		formular.gradeLevel = '8';
		expect(geaenderteFelder(formular)).toEqual({ gradeLevel: 8 });
	});

	it('das Zähldatum vergleicht den Tag, den die Maske zeigt', async () => {
		const formular = await oeffne();
		expect(formular.lastCounted).toBe('2026-01-15');
		expect(geaenderteFelder(formular)).toEqual({});
		formular.lastCounted = '';
		expect(geaenderteFelder(formular)).toEqual({ lastCounted: null });
	});

	it('Schlagworte: dieselben in einer neuen Liste sind keine Änderung, andere schon', async () => {
		const formular = await oeffne();
		formular.schlagworte = ['Biologie', 'Zelle'];
		expect(geaenderteFelder(formular)).toEqual({});
		formular.schlagworte.push('Genetik');
		expect(geaenderteFelder(formular)).toEqual({ schlagworte: ['Biologie', 'Zelle', 'Genetik'] });
	});

	it('Kennung und Bestand sind keine Felder des Titels', async () => {
		const formular = await oeffne();
		formular.stock = 35;
		formular.stockGesehen = 31;
		expect(geaenderteFelder(formular)).toEqual({});
	});

	it('ohne den Stand vom Öffnen: Fehler statt aller Felder', () => {
		expect(() => geaenderteFelder(titel())).toThrow('Der Stand vom Öffnen des Titels fehlt.');
	});
});

describe('buch_felder: der Stand vom Öffnen', () => {
	it('ist eine eigene Kopie', () => {
		const formular = /** @type {any} */ (titel());
		merkeStand(formular);
		formular.schlagworte.push('Genetik');
		formular.erweiterteEigenschaften.littera_id = 1;
		expect(formular.geladen.schlagworte).toEqual(['Biologie', 'Zelle']);
		expect(formular.geladen.erweiterteEigenschaften).toEqual({ littera_id: 4711 });
		expect('geladen' in formular.geladen).toBe(false);
	});

	it('ein neuer Titel bekommt keinen', async () => {
		expect('geladen' in (await titelFuerMaske({ title: 'Neu', stock: 1 }))).toBe(false);
	});

	it('ein schon gespeichertes Cover geht beim Speichern nicht noch einmal mit', async () => {
		const formular = await oeffne();
		uebernimmGespeichert(formular, 'coverUrl', '/covers/neu.webp');
		expect(formular.coverUrl).toBe('/covers/neu.webp');
		expect(geaenderteFelder(formular)).toEqual({});
	});
});

describe('buch_speichern: der Rumpf eines vorhandenen Titels', () => {
	/** @param {any} formular */
	async function gesendet(formular) {
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort(200, { data: { id: 'abc' } }));
		await speichereBuch(formular);
		return JSON.parse(String(vi.mocked(apiFetch).mock.calls.at(-1)?.[1]?.body));
	}

	it('nennt nur das geänderte Feld', async () => {
		const formular = await oeffne();
		formular.verlag = 'Cornelsen';
		expect(await gesendet(formular)).toEqual({ verlag: 'Cornelsen' });
	});

	it('unberührte Maske: ein leerer Rumpf', async () => {
		expect(await gesendet(await oeffne())).toEqual({});
	});

	it('der Bestand geht neben den Feldern mit seinen zwei Zahlen', async () => {
		const formular = await oeffne();
		formular.title = 'Natura 2 Neu';
		formular.stock = 32;
		expect(await gesendet(formular)).toEqual({
			title: 'Natura 2 Neu',
			stock: 32,
			stockGesehen: 30
		});
	});
});
