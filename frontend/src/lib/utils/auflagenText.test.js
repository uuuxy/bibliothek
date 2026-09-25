import { describe, it, expect } from 'vitest';
import {
	auflagenBeschriftung,
	auflagenAufschluesselung,
	auflagenDerKlasse,
	auflagenHinweisText
} from './auflagenText.js';
import pruefung from './auflagenText.faelle.json';

// Wie eine Auflage in Listen heißt: Titel derselben Reihe unterscheiden sich oft nur in
// Auflage und Jahr (docs/OFFEN.md 4.18). Die JavaScript-Seite der geteilten Prüffälle — der
// Ausdruck (inventur/lernmittel_pdf.go) liest dieselbe Datei in
// TestAuflagenBeschriftung_WieImBrowser.
describe('auflagenBeschriftung und auflagenAufschluesselung, geteilte Prüffälle', () => {
	it('liest die gemeinsame Datei überhaupt ein', () => {
		// Ohne diesen Boden liefe die Suite still grün, wenn die Datei umbenannt wird.
		expect(pruefung.beschriftung.length).toBeGreaterThanOrEqual(6);
		expect(pruefung.aufschluesselung.length).toBeGreaterThanOrEqual(2);
	});

	for (const f of pruefung.beschriftung) {
		it(`${f.fall}: ${JSON.stringify([f.auflage, f.jahr])}`, () => {
			expect(auflagenBeschriftung({ auflage: f.auflage, erscheinungsjahr: f.jahr })).toBe(f.soll);
		});
	}

	// Die Beschriftung steht dort mitten im Satz — groß beginnt sie nur mit einem Substantiv.
	for (const f of pruefung.aufschluesselung) {
		it(f.fall, () => {
			const auflagen = f.auflagen.map((/** @type {any} */ a) => ({
				auflage: a.auflage,
				erscheinungsjahr: a.jahr,
				gesamt_bestand: a.bestand
			}));
			expect(auflagenAufschluesselung(auflagen)).toBe(f.soll);
		});
	}

	// Nur im Browser: Ein Feld mit omitempty fehlt in der Antwort, wenn es leer ist — so das
	// Erscheinungsjahr einer Zeile des Bestellbedarfs (api/reorders.go).
	it('kommt ohne die Felder aus', () => {
		expect(auflagenBeschriftung({})).toBe('Auflage ohne Angabe');
		expect(auflagenAufschluesselung([{ gesamt_bestand: 3 }, { gesamt_bestand: 1 }])).toBe(
			'Bestand aus 2 Auflagen: Auflage ohne Angabe (3), Auflage ohne Angabe (1)'
		);
	});
});

// Die Auflagen einer Klasse auf der Klassensatz-Kachel (4.18, Stufe 5): Die Reihenfolge ist die
// des Servers, die Auflage der Kachel trägt denselben Zusatz wie in der Titelmaske.
describe('auflagenDerKlasse', () => {
	it('nennt jede Auflage mit ihren Kindern und markiert die der Kachel', () => {
		expect(
			auflagenDerKlasse(
				[
					{ id: 'neu', auflage: '4. Aufl.', erscheinungsjahr: 2023, kinder: 20 },
					{ id: 'alt', auflage: '3. Aufl.', erscheinungsjahr: 2019, kinder: 1 }
				],
				'neu'
			)
		).toEqual(['4. Aufl. · 2023: 20 Kinder — diese Auflage', '3. Aufl. · 2019: 1 Kind']);
	});

	it('markiert nichts, wenn die Auflage der Kachel in der Klasse fehlt', () => {
		// Von Hand ist die 3. Auflage zugeordnet, die Kinder haben alle die 4.
		expect(
			auflagenDerKlasse(
				[{ id: 'neu', auflage: '4. Aufl.', erscheinungsjahr: 2023, kinder: 6 }],
				'alt'
			)
		).toEqual(['4. Aufl. · 2023: 6 Kinder']);
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
