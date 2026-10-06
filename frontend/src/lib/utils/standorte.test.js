import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn() }));

import { apiFetch } from '../apiFetch.js';
import { ladeStandorte, standorteAusExemplaren, standortZeile } from './standorte.js';

// Der Standort steht am Exemplar (docs/OFFEN.md 5.53). Kopf der Buchakte und Titel-Verwaltung
// nennen die Standorte eines Titels in derselben Zeile; der Kopf zählt sie aus den Karten, die
// Titel-Verwaltung bekommt sie gezählt vom Server. Beide zählen nur den Bestand.

describe('standorteAusExemplaren', () => {
	it('zählt je Standort die Exemplare im Bestand', () => {
		const exemplare = [
			{ standort: 'Bibliothek, Regal 3B', im_bestand: true },
			{ standort: 'Lehrerschrank', im_bestand: true },
			{ standort: 'Bibliothek, Regal 3B', im_bestand: true },
			{ standort: '', im_bestand: true },
			{ im_bestand: true }
		];
		expect(standorteAusExemplaren(exemplare)).toEqual([
			{ standort: 'Bibliothek, Regal 3B', anzahl: 2 },
			{ standort: 'Lehrerschrank', anzahl: 1 }
		]);
	});

	it('zählt ausgesonderte und bestellte Exemplare nicht mit', () => {
		const exemplare = [
			{ standort: 'Keller', im_bestand: false },
			{ standort: 'Lehrerschrank', im_bestand: true }
		];
		expect(standorteAusExemplaren(exemplare)).toEqual([{ standort: 'Lehrerschrank', anzahl: 1 }]);
	});

	it('liefert ohne Exemplare eine leere Liste', () => {
		expect(standorteAusExemplaren([])).toEqual([]);
		expect(standorteAusExemplaren(/** @type {any} */ (null))).toEqual([]);
	});
});

describe('standortZeile', () => {
	it('nennt den häufigsten Standort zuerst und trennt mit dem Punkt', () => {
		expect(
			standortZeile([
				{ standort: 'Lehrerschrank', anzahl: 1 },
				{ standort: 'Bibliothek, Regal 3B', anzahl: 2 }
			])
		).toBe('Bibliothek, Regal 3B (2) · Lehrerschrank (1)');
	});

	it('ordnet gleich häufige nach dem Namen, Umlaute wie im Deutschen', () => {
		expect(
			standortZeile([
				{ standort: 'Zimmer 12', anzahl: 1 },
				{ standort: 'Ärztezimmer', anzahl: 1 },
				{ standort: 'Bibliothek', anzahl: 1 }
			])
		).toBe('Ärztezimmer (1) · Bibliothek (1) · Zimmer 12 (1)');
	});

	it('ändert die übergebene Liste nicht', () => {
		const liste = [
			{ standort: 'B', anzahl: 1 },
			{ standort: 'A', anzahl: 5 }
		];
		standortZeile(liste);
		expect(liste[0].standort).toBe('B');
	});

	it('ist ohne Standorte leer', () => {
		expect(standortZeile([])).toBe('');
		expect(standortZeile(undefined)).toBe('');
		expect(standortZeile(null)).toBe('');
	});
});

describe('ladeStandorte', () => {
	// Mit Klammern: Gäbe der Haken den Mock zurück, riefe Vitest ihn nach dem Test als Aufräumer auf.
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
	});

	it('liefert die Liste des Servers', async () => {
		const liste = [{ standort: 'Lehrerschrank', anzahl: 3 }];
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => liste })
		);
		expect(await ladeStandorte()).toEqual(liste);
		expect(apiFetch).toHaveBeenCalledWith('/api/exemplare/standorte');
	});

	it('bleibt leer, wenn der Server nicht antwortet oder ablehnt', async () => {
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: false }));
		expect(await ladeStandorte()).toEqual([]);
		vi.mocked(apiFetch).mockImplementation(() => Promise.reject(new Error('offline')));
		expect(await ladeStandorte()).toEqual([]);
	});
});
