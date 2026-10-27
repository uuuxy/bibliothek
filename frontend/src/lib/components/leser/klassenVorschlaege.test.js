import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
	erzeugeKlassenVorschlaege,
	klassenPlatzhalter,
	klassenLadefehler
} from './klassenVorschlaege.svelte.js';
import { apiFetch } from '../../apiFetch.js';
import { serverAntwort } from '../../serverAntwort.js';

// leseApiFehler bleibt echt (ARCHITEKTUR.md 8.14).
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

// Die Auswahl im Anlegen-Dialog las bis zum 15.09.2026 die Tabelle lesergruppen, die kein
// Schreibweg füllt: Angeboten wurde nur „Manuell eingeben…". Die Klassen der Schule liefert
// GET /api/klassen — dieselbe Quelle wie Druck-Center und LMF-Verlängerung.
//
// Seit dem 30.09.2026 wird die Klasse nur noch gewählt. Ein gescheiterter Abruf lässt sich
// also nicht mehr durch Tippen umgehen, und er darf nicht aussehen wie eine Schule ohne
// Klassen.

/** @param {boolean} ok @param {any} [daten] */
const antwort = (ok, daten = {}) =>
	ok ? serverAntwort(200, daten) : serverAntwort(503, { error: 'Datenbank nicht erreichbar' });

beforeEach(() => {
	vi.clearAllMocks();
});

describe('erzeugeKlassenVorschlaege', () => {
	it('bietet die Klassen der Schule an', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(true, ['05F1', '07H', '10R']));

		const vorschlaege = erzeugeKlassenVorschlaege();
		await vorschlaege.lade();

		expect(apiFetch).toHaveBeenCalledWith('/api/klassen');
		expect(vorschlaege.liste).toEqual(['05F1', '07H', '10R']);
		expect(vorschlaege.ladefehler).toBe('');
	});

	// Mit dem Grund des Servers: Bis zur Leserakte im Durchgang aus OFFEN.md 5.106 stand an jeder
	// Klassenwahl derselbe Satz, gleich was der Server antwortete.
	it('meldet einen gescheiterten Abruf mit Grund, statt „Keine Klassen" zu behaupten', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(false));

		const vorschlaege = erzeugeKlassenVorschlaege();
		await vorschlaege.lade();

		expect(vorschlaege.liste).toEqual([]);
		expect(vorschlaege.ladefehler).toBe('Datenbank nicht erreichbar');
		expect(klassenPlatzhalter(vorschlaege.liste.length, vorschlaege.ladefehler)).toBe(
			'Klassen nicht geladen'
		);
	});

	it('meldet auch einen Netzfehler', async () => {
		vi.mocked(apiFetch).mockRejectedValue(new Error('Keine Verbindung zum Server.'));

		const vorschlaege = erzeugeKlassenVorschlaege();
		await vorschlaege.lade();

		expect(vorschlaege.ladefehler).toBe('Keine Verbindung zum Server.');
	});

	it('behält eine geladene Liste, wenn ein späterer Abruf scheitert, und nimmt den Fehler zurück, sobald einer gelingt', async () => {
		const vorschlaege = erzeugeKlassenVorschlaege();
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort(true, ['05F1']));
		await vorschlaege.lade();
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort(false));
		await vorschlaege.lade();

		expect(vorschlaege.liste).toEqual(['05F1']);
		expect(vorschlaege.ladefehler).toBe('Datenbank nicht erreichbar');

		vi.mocked(apiFetch).mockResolvedValueOnce(antwort(true, ['05F1', '05F2']));
		await vorschlaege.lade();
		expect(vorschlaege.liste).toEqual(['05F1', '05F2']);
		expect(vorschlaege.ladefehler).toBe('');
	});
});

describe('klassenPlatzhalter', () => {
	it('lädt zum Wählen ein, sobald es etwas zu wählen gibt', () => {
		expect(klassenPlatzhalter(3, '')).toBe('Klasse wählen');
		expect(klassenPlatzhalter(3, 'Datenbank nicht erreichbar')).toBe('Klasse wählen');
	});

	it('unterscheidet die leere Liste vom gescheiterten Abruf', () => {
		expect(klassenPlatzhalter(0, '')).toBe('Keine Klassen');
		expect(klassenPlatzhalter(0, '', 'Keine weiteren Klassen')).toBe('Keine weiteren Klassen');
		expect(klassenPlatzhalter(0, 'Fehler 503', 'Keine weiteren Klassen')).toBe(
			'Klassen nicht geladen'
		);
	});
});

describe('klassenLadefehler', () => {
	it('nennt den Grund und den Weg, mit einem Punkt am Ende des Grundes', () => {
		expect(klassenLadefehler('Datenbank nicht erreichbar')).toBe(
			'Die Klassen ließen sich nicht laden: Datenbank nicht erreichbar. Bitte neu öffnen.'
		);
		expect(klassenLadefehler('Keine Verbindung zum Server.')).toBe(
			'Die Klassen ließen sich nicht laden: Keine Verbindung zum Server. Bitte neu öffnen.'
		);
	});
});
