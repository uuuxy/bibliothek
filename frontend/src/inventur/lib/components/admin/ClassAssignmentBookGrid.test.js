import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import ClassAssignmentBookGrid from './ClassAssignmentBookGrid.svelte';

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
	const { container } = render(ClassAssignmentBookGrid, { books });
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
