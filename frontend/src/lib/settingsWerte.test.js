import { describe, it, expect } from 'vitest';
import { zahlOderLeer, sammleZahlen } from './settingsWerte.js';

/**
 * Die getippte 0 ist ein Wert („aus"), ein LEERES Feld ist keiner. Der Unterschied ist
 * der Kern zweier Regressionen: Erst machte `Number(null) || 0` aus dem leeren Feld
 * still eine 0 und schaltete damit die Lesehistorie-Befristung ab (Prüfung 22.08.2026,
 * A4); danach hieß leer „nicht mitschicken", was beim Speichern je Kategorie
 * (23.08.2026) niemand mehr erraten soll — leer wird jetzt gemeldet.
 */
describe('zahlOderLeer', () => {
	it('getippte 0 ist ein Wert (aus), kein leeres Feld', () => {
		expect(zahlOderLeer(0)).toBe(0);
		expect(zahlOderLeer('0')).toBe(0);
	});
	it('leeres Feld (null/""/undefined) ist keine Angabe', () => {
		expect(zahlOderLeer(null)).toBeNull();
		expect(zahlOderLeer('')).toBeNull();
		expect(zahlOderLeer(undefined)).toBeNull();
	});
	it('Unlesbares und Negatives wird nicht zur 0', () => {
		expect(zahlOderLeer('abc')).toBeNull();
		expect(zahlOderLeer(-5)).toBeNull();
		expect(zahlOderLeer(NaN)).toBeNull();
	});
	it('normale Zahlen kommen ganzzahlig durch', () => {
		expect(zahlOderLeer(90)).toBe(90);
		expect(zahlOderLeer('730')).toBe(730);
		expect(zahlOderLeer(15.7)).toBe(15);
	});
});

describe('sammleZahlen', () => {
	it('baut den Patch aus den ausgefüllten Feldern', () => {
		const { werte, fehlend } = sammleZahlen([
			{ schluessel: 'frist_buch_tage', label: 'Tage / Buch', wert: 28 },
			{ schluessel: 'lesehistorie_tage', label: 'Lesehistorie', wert: '0' }
		]);
		expect(werte).toEqual({ frist_buch_tage: 28, lesehistorie_tage: 0 });
		expect(fehlend).toEqual([]);
	});

	it('meldet einen Wert unterhalb der Feldgrenze — das Backend ersetzt ihn sonst still', () => {
		// Eine getippte 0 in „Tage / Buch" wird serverseitig zu 21. Ohne diese Prüfung
		// bekäme man eine Erfolgsmeldung und fände danach 21 im Feld.
		const { werte, fehlend } = sammleZahlen([
			{ schluessel: 'frist_buch_tage', label: 'Tage / Buch', wert: 0, min: 1 },
			// Dasselbe Feld mit min 0: Dort BEDEUTET die 0 „aus" und ist ein Wert.
			{ schluessel: 'sperre_minuten', label: 'Sperrbildschirm nach', wert: 0, min: 0 }
		]);
		expect(werte).toEqual({ sperre_minuten: 0 });
		expect(fehlend).toEqual(['Tage / Buch (mindestens 1)']);
	});

	it('meldet leere Felder mit ihrer Beschriftung und lässt sie aus dem Patch', () => {
		const { werte, fehlend } = sammleZahlen([
			{ schluessel: 'frist_buch_tage', label: 'Tage / Buch', wert: '' },
			{ schluessel: 'frist_medien_tage', label: 'Tage / Medien', wert: 7 },
			{ schluessel: 'max_ausleihen_schueler', label: 'Max. Ausleihen', wert: null }
		]);
		// Der Patch trägt NUR die gültigen Felder — der Aufrufer schickt ihn bei
		// fehlenden Angaben ohnehin nicht ab, aber ein halber Patch darf nie entstehen.
		expect(werte).toEqual({ frist_medien_tage: 7 });
		expect(fehlend).toEqual(['Tage / Buch', 'Max. Ausleihen']);
	});
});
