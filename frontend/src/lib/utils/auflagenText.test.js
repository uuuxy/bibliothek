import { describe, it, expect } from 'vitest';
import {
	auflagenBeschriftung,
	auflagenAufschluesselung,
	auflagenHinweisText
} from './auflagenText.js';

// Wie eine Auflage in Listen heißt: Titel derselben Reihe unterscheiden sich oft nur in
// Auflage und Jahr (docs/OFFEN.md 4.18).
describe('auflagenBeschriftung', () => {
	it('nennt Auflage und Jahr', () => {
		expect(auflagenBeschriftung({ auflage: '4. Aufl.', erscheinungsjahr: 2023 })).toBe(
			'4. Aufl. · 2023'
		);
	});

	it('nennt das Jahr nicht zweimal, wenn es schon in der Auflage steht', () => {
		expect(auflagenBeschriftung({ auflage: '4. Aufl. 2023', erscheinungsjahr: 2023 })).toBe(
			'4. Aufl. 2023'
		);
	});

	it('kommt mit nur einem der beiden aus — und mit keinem', () => {
		expect(auflagenBeschriftung({ auflage: ' 2. Auflage ', erscheinungsjahr: 0 })).toBe(
			'2. Auflage'
		);
		expect(auflagenBeschriftung({ erscheinungsjahr: 2019 })).toBe('Ausgabe 2019');
		expect(auflagenBeschriftung({})).toBe('Auflage ohne Angabe');
	});
});

// Die Aufschlüsselung einer Zeile im Bestellbedarf (4.18, Stufe 3): Die Zahlen der Zeile sind
// die Summe; hier steht, woraus sie besteht.
describe('auflagenAufschluesselung', () => {
	it('nennt jede Auflage mit ihrem Bestand, in der Reihenfolge des Servers', () => {
		expect(
			auflagenAufschluesselung([
				{ auflage: '4. Aufl.', erscheinungsjahr: 2023, gesamt_bestand: 30 },
				{ auflage: '3. Aufl.', erscheinungsjahr: 2019, gesamt_bestand: 0 }
			])
		).toBe('Bestand aus 2 Auflagen: 4. Aufl. · 2023 (30), 3. Aufl. · 2019 (0)');
	});

	// Die Beschriftung steht hier mitten im Satz — groß beginnt sie nur mit einem Substantiv.
	it('bleibt ein Satz, wenn eine Auflage ohne Angabe ist', () => {
		expect(
			auflagenAufschluesselung([
				{ auflage: '4. Aufl.', erscheinungsjahr: 2023, gesamt_bestand: 30 },
				{ gesamt_bestand: 3 }
			])
		).toBe('Bestand aus 2 Auflagen: 4. Aufl. · 2023 (30), Auflage ohne Angabe (3)');
	});
});

// Die Hinweiszeile der Theke bei gemischten Auflagen (4.18, Stufe 5).
describe('auflagenHinweisText', () => {
	it('nennt Klasse, die anderen Auflagen mit ihren Kindern und dieses Exemplar', () => {
		expect(
			auflagenHinweisText({
				klasse: '07B',
				auflage: '4. Aufl.',
				erscheinungsjahr: 2023,
				andere: [
					{ auflage: '3. Aufl.', erscheinungsjahr: 2019, kinder: 12 },
					{ auflage: '2. Aufl.', erscheinungsjahr: 2015, kinder: 1 }
				]
			})
		).toBe(
			'Andere Auflage in der 07B: 12 Kinder haben 3. Aufl. · 2019, 1 Kind hat 2. Aufl. · 2015 — dieses Exemplar ist 4. Aufl. · 2023.'
		);
	});

	it('bleibt ein Satz, wenn Auflage oder Jahr fehlen', () => {
		expect(
			auflagenHinweisText({ klasse: '07B', erscheinungsjahr: 2023, andere: [{ kinder: 3 }] })
		).toBe(
			'Andere Auflage in der 07B: 3 Kinder haben Auflage ohne Angabe — dieses Exemplar ist Ausgabe 2023.'
		);
	});
});
