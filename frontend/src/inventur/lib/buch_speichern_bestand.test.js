import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../../lib/apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../../lib/stores/bestaetigung.svelte.js', () => ({ bestaetigen: vi.fn() }));
vi.mock('./store.svelte.js', () => ({ appState: { bookToEdit: null }, showToast: vi.fn() }));
import { apiFetch } from '../../lib/apiFetch.js';
import {
	speichereBuch,
	bestandsAngabe,
	verringernRueckfrage,
	ladeBestandNach,
	BestandVeraltetFehler
} from './buch_speichern.js';

/** @param {number} status @param {any} koerper */
const antwort = (status, koerper) =>
	/** @type {any} */ ({ ok: status < 400, status, json: async () => koerper });

/** Die Maske eines vorhandenen Titels trägt den Stand vom Öffnen (titelFuerMaske).
 * @param {any} formular */
const geoeffnet = (formular) =>
	formular.id ? { ...formular, geladen: { ...formular } } : formular;

/** @param {any} formular */
async function gesendet(formular) {
	vi.mocked(apiFetch).mockResolvedValue(antwort(200, { data: { id: formular.id ?? 'neu' } }));
	await speichereBuch(geoeffnet(formular));
	return JSON.parse(String(vi.mocked(apiFetch).mock.calls[0][1]?.body));
}

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
});

// Eine offen gelassene Maske schickte bei jedem Speichern den Bestand vom Öffnen mit, und
// der Server setzte ihn durch: Was ein anderer Platz inzwischen angelegt hatte, wurde
// ausgesondert. Der Bestand geht nur noch hinaus, wenn jemand das Feld geändert hat.
describe('buch_speichern: der Bestand eines vorhandenen Titels', () => {
	it('unverändert: weder stock noch stockGesehen im Rumpf', async () => {
		const koerper = await gesendet({ id: 'abc', isbn: '978', stock: 3, stockGesehen: 3 });
		expect('stock' in koerper).toBe(false);
		expect('stockGesehen' in koerper).toBe(false);
	});

	it('geändert: die neue Zahl und die vom Öffnen', async () => {
		const koerper = await gesendet({ id: 'abc', isbn: '978', stock: 5, stockGesehen: 3 });
		expect(koerper.stock).toBe(5);
		expect(koerper.stockGesehen).toBe(3);
	});

	it('aus dem Feld kommt die Zahl auch als Text', async () => {
		const koerper = await gesendet({ id: 'abc', isbn: '978', stock: '5', stockGesehen: 3 });
		expect(koerper.stock).toBe(5);
	});

	it.each([[null], [undefined], ['']])('geleertes Feld (%s): nichts zum Bestand', async (leer) => {
		const koerper = await gesendet({ id: 'abc', isbn: '978', stock: leer, stockGesehen: 3 });
		expect('stock' in koerper).toBe(false);
		expect('stockGesehen' in koerper).toBe(false);
	});

	it('ohne Zahl vom Öffnen geht nur die neue hinaus — der Server lehnt eine Änderung ab', async () => {
		const koerper = await gesendet({ id: 'abc', isbn: '978', stock: 5 });
		expect(koerper.stock).toBe(5);
		expect('stockGesehen' in koerper).toBe(false);
	});

	it.each([[2.5], [-1], ['abc']])(
		'keine ganze Zahl ab 0 (%s): Fehler, keine Anfrage',
		async (feld) => {
			const fehler = await speichereBuch(
				geoeffnet({ id: 'abc', isbn: '978', stock: feld, stockGesehen: 3 })
			).catch((e) => e);
			expect(fehler.message).toBe('Der Bestand muss eine ganze Zahl ab 0 sein.');
			expect(apiFetch).not.toHaveBeenCalled();
		}
	);

	it('409 mit dem Stand des Servers wird ein BestandVeraltetFehler', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(409, { error: 'jetzt 6 statt 1', bestand: 6 }));
		const fehler = await speichereBuch(
			geoeffnet({ id: 'abc', isbn: '978', stock: 2, stockGesehen: 1 })
		).catch((e) => e);
		expect(fehler).toBeInstanceOf(BestandVeraltetFehler);
		expect(fehler.bestand).toBe(6);
		expect(fehler.message).toBe('jetzt 6 statt 1');
	});
});

describe('buch_speichern: der Bestand eines neuen Titels', () => {
	it('geht als Zahl hinaus, ohne stockGesehen', () => {
		expect(bestandsAngabe({ id: null, stock: 1 })).toEqual({ stock: 1 });
		expect(bestandsAngabe({ id: null, stock: 0 })).toEqual({ stock: 0 });
	});

	it('leer: nichts — der Titel entsteht ohne Exemplar', () => {
		expect(bestandsAngabe({ id: null, stock: null })).toEqual({});
	});
});

// Die Rückfrage verglich gegen die geladene Titelliste. Stand der Titel nicht in ihr (über
// „Neues Buch" und eine vergebene ISBN geöffnet, Liste gefiltert), kam sie nicht.
describe('buch_speichern: verringernRueckfrage', () => {
	it('nennt beide Zahlen und wie viele Exemplare es trifft', () => {
		const frage = verringernRueckfrage({ id: 'abc', stock: 1, stockGesehen: 3 });
		expect(frage?.titel).toBe('Bestand von 3 auf 1 verringern?');
		expect(frage?.text).toContain('2 Exemplare werden ausgesondert');
		expect(frage?.aktion).toBe('Verringern');
		expect(frage?.gefaehrlich).toBe(true);
	});

	it('ein einzelnes Exemplar steht in der Einzahl', () => {
		expect(verringernRueckfrage({ id: 'abc', stock: 2, stockGesehen: 3 })?.text).toContain(
			'Ein Exemplar wird ausgesondert'
		);
	});

	it.each([
		['unverändert', { id: 'abc', stock: 3, stockGesehen: 3 }],
		['erhöht', { id: 'abc', stock: 5, stockGesehen: 3 }],
		['geleert', { id: 'abc', stock: null, stockGesehen: 3 }],
		['neuer Titel', { id: null, stock: 0 }],
		['ohne Zahl vom Öffnen', { id: 'abc', stock: 1 }]
	])('%s: keine Rückfrage', (_name, formular) => {
		expect(verringernRueckfrage(formular)).toBeNull();
	});
});

// Nach dem Löschen eines Exemplars aus der Maske zählte sie selbst eins herunter, auch wenn
// das Exemplar ausgesondert oder bestellt war und im Bestand nicht mitzählte.
describe('buch_speichern: ladeBestandNach', () => {
	it('unberührtes Feld: beide Zahlen folgen dem Server', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(200, { id: 'abc', stock: 2 }));
		const formular = { id: 'abc', stock: 3, stockGesehen: 3 };
		await ladeBestandNach(formular);
		expect(formular).toMatchObject({ stock: 2, stockGesehen: 2 });
		expect(vi.mocked(apiFetch).mock.calls[0][0]).toBe('/api/books/abc');
	});

	it('getippte Zahl bleibt, nur die gesehene folgt', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(200, { id: 'abc', stock: 2 }));
		const formular = { id: 'abc', stock: 7, stockGesehen: 3 };
		await ladeBestandNach(formular);
		expect(formular).toMatchObject({ stock: 7, stockGesehen: 2 });
	});

	it('Nachladen scheitert: die Zahlen bleiben', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(500, {}));
		const formular = { id: 'abc', stock: 3, stockGesehen: 3 };
		await ladeBestandNach(formular);
		expect(formular).toMatchObject({ stock: 3, stockGesehen: 3 });
	});
});
