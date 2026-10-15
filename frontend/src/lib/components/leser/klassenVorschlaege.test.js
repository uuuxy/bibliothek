import { describe, it, expect, vi, beforeEach } from 'vitest';
import { erzeugeKlassenVorschlaege, klassenPlatzhalter } from './klassenVorschlaege.svelte.js';
import { apiFetch } from '../../apiFetch.js';

vi.mock('../../apiFetch.js', () => ({
	apiFetch: vi.fn(),
	extractApiError: vi.fn(async () => 'Serverfehler')
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
	/** @type {any} */ ({ ok, status: ok ? 200 : 500, json: async () => daten });

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
		expect(vorschlaege.ladefehler).toBe(false);
	});

	it('meldet einen gescheiterten Abruf, statt „Keine Klassen" zu behaupten', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(false));

		const vorschlaege = erzeugeKlassenVorschlaege();
		await vorschlaege.lade();

		expect(vorschlaege.liste).toEqual([]);
		expect(vorschlaege.ladefehler).toBe(true);
		expect(klassenPlatzhalter(vorschlaege.liste.length, vorschlaege.ladefehler)).toBe(
			'Klassen nicht geladen'
		);
	});

	it('meldet auch einen Netzfehler', async () => {
		vi.mocked(apiFetch).mockRejectedValue(new TypeError('Failed to fetch'));

		const vorschlaege = erzeugeKlassenVorschlaege();
		await vorschlaege.lade();

		expect(vorschlaege.ladefehler).toBe(true);
	});

	it('behält eine geladene Liste, wenn ein späterer Abruf scheitert, und nimmt den Fehler zurück, sobald einer gelingt', async () => {
		const vorschlaege = erzeugeKlassenVorschlaege();
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort(true, ['05F1']));
		await vorschlaege.lade();
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort(false));
		await vorschlaege.lade();

		expect(vorschlaege.liste).toEqual(['05F1']);
		expect(vorschlaege.ladefehler).toBe(true);

		vi.mocked(apiFetch).mockResolvedValueOnce(antwort(true, ['05F1', '05F2']));
		await vorschlaege.lade();
		expect(vorschlaege.liste).toEqual(['05F1', '05F2']);
		expect(vorschlaege.ladefehler).toBe(false);
	});
});

describe('klassenPlatzhalter', () => {
	it('lädt zum Wählen ein, sobald es etwas zu wählen gibt', () => {
		expect(klassenPlatzhalter(3, false)).toBe('Klasse wählen');
		expect(klassenPlatzhalter(3, true)).toBe('Klasse wählen');
	});

	it('unterscheidet die leere Liste vom gescheiterten Abruf', () => {
		expect(klassenPlatzhalter(0, false)).toBe('Keine Klassen');
		expect(klassenPlatzhalter(0, false, 'Keine weiteren Klassen')).toBe('Keine weiteren Klassen');
		expect(klassenPlatzhalter(0, true, 'Keine weiteren Klassen')).toBe('Klassen nicht geladen');
	});
});
