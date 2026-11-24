import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import FolieLmfHeute from './FolieLmfHeute.svelte';

/** @param {string} art @param {{ stunde: number, klassen: string[], vermerk: string }[]} termine */
const tag = (art, termine) => ({ art, datum: '2027-08-10', termine });

// Die Folie am Tag des LMF-Plans: was heute ist, und je Stunde die Klassen, so wie die Bibliothek
// den Plan geschrieben hat.
describe('Folie „Heute" auf dem Flur-Monitor', () => {
	it('nennt Art und Tag und zeigt je Zeile Stunde, Klassen und Vermerk', () => {
		const screen = render(FolieLmfHeute, {
			heute: tag('ausgabe', [
				{ stunde: 1, klassen: ['05F'], vermerk: '' },
				{ stunde: 2, klassen: ['06F1', '06F2'], vermerk: 'nur Rückgabe' },
				{ stunde: 3, klassen: [], vermerk: 'Nachzügler' }
			])
		});
		expect(screen.getByRole('heading').textContent?.replace(/\s+/g, ' ').trim()).toBe(
			'Heute Bücherausgabe'
		);
		expect(screen.getByText('Dienstag, 10. August 2027')).toBeTruthy();
		const zeilen = screen
			.getAllByRole('listitem')
			.map((z) => z.textContent?.replace(/\s+/g, ' ').trim());
		expect(zeilen).toEqual(['1 05F', '2 06F1, 06F2 nur Rückgabe', '3 Nachzügler']);
		expect(
			screen.getByText('Hinweis'),
			'ein Vermerk an einer Klasse bekommt eine Spalte'
		).toBeTruthy();
	});

	it('heißt am Tag vor den Sommerferien „Büchertausch" und zeigt ohne Vermerk an einer Klasse keine Spalte dafür', () => {
		const screen = render(FolieLmfHeute, {
			heute: tag('rueckgabe', [
				{ stunde: 1, klassen: ['07R3'], vermerk: '' },
				{ stunde: 2, klassen: [], vermerk: 'Aufräumen' }
			])
		});
		expect(screen.getByRole('heading').textContent?.replace(/\s+/g, ' ').trim()).toBe(
			'Heute Büchertausch'
		);
		expect(screen.queryByText('Hinweis')).toBeNull();
	});

	it('setzt mehr als sechs Zeilen kleiner, damit alle auf die Folie passen', () => {
		const zeilen = (/** @type {number} */ n) =>
			Array.from({ length: n }, (_, i) => ({ stunde: i + 1, klassen: [`05F${i}`], vermerk: '' }));
		const sechs = render(FolieLmfHeute, { heute: tag('ausgabe', zeilen(6)) });
		expect(sechs.getByText('05F0').className).toContain('text-display-small-emphasized');
		sechs.unmount();
		const acht = render(FolieLmfHeute, { heute: tag('ausgabe', zeilen(8)) });
		expect(acht.getByText('05F0').className).toContain('text-headline-small-emphasized');
	});
});
