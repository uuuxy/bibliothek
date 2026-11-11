import { describe, it, expect } from 'vitest';
import { fireEvent, render } from '@testing-library/svelte';
import KlassensatzTitelRaster from './KlassensatzTitelRaster.svelte';

/** @param {Record<string, any>} zusatz */
const buch = (zusatz) => ({
	id: 't-1',
	title: 'Natura',
	author: 'Beyer',
	subject: 'Biologie',
	coverUrl: '',
	...zusatz
});

/** Der Text der Karten, je Buch. @param {any[]} books */
function karten(books) {
	const { container } = render(KlassensatzTitelRaster, { titelListe: books });
	return [...container.querySelectorAll('button')].map((k) => (k.textContent ?? '').trim());
}

// Die Karten der Klassenzuweisung tragen den Jahrgang als Marke, aus „von … bis" wie Maske
// und Titelliste.
describe('Klassenzuweisung: die Marke für den Jahrgang', () => {
	it('nennt ein Jahr und eine Spanne', () => {
		const [einJahr, spanne] = karten([
			buch({ id: 'a', jahrgangVon: 7, jahrgangBis: 7 }),
			buch({ id: 'b', jahrgangVon: 7, jahrgangBis: 10 })
		]);
		expect(einJahr).toContain('Jg. 7');
		expect(einJahr).not.toContain('Jg. 7–');
		expect(spanne).toContain('Jg. 7–10');
	});

	it('trägt ohne „von … bis" keine Marke', () => {
		const [karte] = karten([buch({ jahrgangVon: 0, jahrgangBis: 0 })]);
		expect(karte).toContain('Natura');
		expect(karte).not.toMatch(/Jg\.|Kl\./);
	});
});

// Der Platzhalter verspricht die ISBN; sie trifft auch in der anderen Länge und mit Bindestrichen.
describe('Klassenzuweisung: Suche nach der ISBN', () => {
	const buecher = [
		buch({ id: 'a', title: 'Lambacher Schweizer 7', isbn: '9783127338713' }),
		buch({ id: 'b', title: 'Green Line 3', isbn: '9783128342306' })
	];

	it.each([
		['dreizehnstellig', '9783128342306'],
		['mit Bindestrichen', '978-3-12-834230-6'],
		['zehnstellig', '3128342309'],
		['Teil der Nummer', '834230']
	])('%s', async (_wie, eingabe) => {
		const ansicht = render(KlassensatzTitelRaster, { titelListe: buecher });
		await fireEvent.input(ansicht.getByLabelText('Titel durchsuchen'), {
			target: { value: eingabe }
		});
		const titel = [...ansicht.container.querySelectorAll('button')]
			.map((k) => (k.textContent ?? '').trim())
			.filter((t) => t.includes('Green') || t.includes('Lambacher'));
		expect(titel).toHaveLength(1);
		expect(titel[0]).toContain('Green Line 3');
	});
});
